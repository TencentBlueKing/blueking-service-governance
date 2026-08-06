/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package kubeinsight

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// ErrReportNotFound 巡检报告不存在, 通常是因为对应集群没有安装 kube-insight 组件
var ErrReportNotFound = errors.New("report not found")

// Client KubeInsight API 客户端接口
type Client interface {
	// GetLatestClusterReport 获取集群最新检查报告, 返回集群报告数据和 PDF 报告二进制
	GetLatestClusterReport(ctx context.Context, clusterID string, generatePDF bool) (*ClusterReport, []byte, error)
}

// ApiClient api client
type ApiClient struct {
	define.BkApiClient
}

// New 创建 KubeInsight API 客户端实例
//
// 该函数用于创建与 KubeInsight API 网关交互的客户端，配置了必要的认证信息：
// - 使用应用认证：通过 bk_app_code 和 bk_app_secret 进行应用身份验证
func New() (Client, error) {
	if config.G.Development.UseStubKubeInsight {
		log.InfoNoContext("use stub kubeinsight client according to config")
		return NewStub(), nil
	}
	authorization, _ := json.Marshal(map[string]string{
		"bk_app_code":   config.G.BkApp.Code,
		"bk_app_secret": config.G.BkApp.Secret,
	})
	client, err := bkapi.NewBkApiClient("bcs-kube-insight-service", bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:        config.G.BkApiStages.KubeInsight,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("x-bkapi-authorization", string(authorization)),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	})
	if err != nil {
		return nil, err
	}
	return &ApiClient{client}, nil
}

// GetLatestClusterReport 获取集群最新检查报告, 返回集群报告数据和 PDF 报告二进制
func (c *ApiClient) GetLatestClusterReport(
	ctx context.Context,
	clusterID string,
	generatePDF bool,
) (*ClusterReport, []byte, error) {
	var err error
	started := time.Now()
	defer metrics.ReportClientRequestMetric("kubeinsight", "GetLatestClusterReport", started, &err)

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "GetLatestClusterReport",
			Method: "GET",
			Path:   "/v1/kubeinsight/report/cluster/{clusterID}/latest",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"clusterID": clusterID}),
		bkapi.OptSetRequestQueryParams(map[string]string{"generate_pdf": strconv.FormatBool(generatePDF)}),
	)

	// 先解析为通用响应结构
	var rawResult GetLatestClusterReportResponse
	ctx, span := apm.StartClientSpan(ctx, "kubeinsight", "GetLatestClusterReport")
	resp, err := apiOperation.SetContext(ctx).SetResult(&rawResult).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, nil, err
	}
	if rawResult.Code != 0 {
		if rawResult.Code == http.StatusNotFound {
			return nil, nil, ErrReportNotFound
		}
		return nil, nil, errors.Errorf(
			"call kubeinsight failed, code: %d, message: %s",
			rawResult.Code,
			rawResult.Message,
		)
	}

	return rawResult.Data, rawResult.PDFData, nil
}
