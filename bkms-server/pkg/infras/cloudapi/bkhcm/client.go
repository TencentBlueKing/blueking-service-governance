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

package bkhcm

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// VendorTCloudZiYan 固定云厂商标识
const VendorTCloudZiYan = "tcloud-ziyan"

// Client bk-hcm API 客户端接口
type Client interface {
	// ListRegions 查询地域列表
	ListRegions(ctx context.Context, filter *Filter, page *Page) ([]Region, error)
	// ListSubnets 查询子网列表（需要用户认证）
	ListSubnets(ctx context.Context, bkBizID int64, filter *Filter, page *Page) ([]Subnet, error)
	// ListVPCs 查询 VPC 列表（需要用户认证）
	ListVPCs(ctx context.Context, bkBizID int64, filter *Filter, page *Page) ([]VPC, error)
	// ListZones 查询可用区列表
	ListZones(ctx context.Context, region string, filter *Filter, page *Page) ([]Zone, error)
	// CreateBizApplicationForCreateLoadBalancer 创建负载均衡申请
	CreateBizApplicationForCreateLoadBalancer(ctx context.Context, req *CreateLoadBalancerReq) (string, error)
}

// ApiClient bk-hcm API 客户端实现
type ApiClient struct {
	define.BkApiClient
	user auth.User
}

// New 创建 bk-hcm API 客户端实例
//
// 该函数用于创建与 bk-hcm API 网关交互的客户端，配置了必要的认证信息：
// - 使用应用认证：通过 bk_app_code 和 bk_app_secret 进行应用身份验证
// - 部分接口（list_subnet、list_vpc）需要用户认证，通过 bk_ticket 传递
//
// 参数：
// - user: 用户信息，用于需要用户认证的接口传递 bk_ticket
func New(user auth.User) (Client, error) {
	if config.G.Development.UseStubBkHCM {
		log.InfoNoContext("use stub bkhcm client according to config")
		return NewStub(user), nil
	}

	authFields := map[string]string{
		"bk_app_code":   config.G.BkApp.Code,
		"bk_app_secret": config.G.BkApp.Secret,
	}
	// 对需要用户认证的接口，注入用户票据
	if credKey, credValue := user.Cred.CredKey(), user.Cred.CredValue(); credKey != "" && credValue != "" {
		authFields[credKey] = credValue
	}

	authorization, _ := json.Marshal(authFields)
	client, err := bkapi.NewBkApiClient("bk-hcm", bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:        config.G.BkApiStages.BkHCM,
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
	return &ApiClient{client, user}, nil
}

// handleOperation 发起请求并检查结果，返回响应数据 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	defer metrics.ReportClientRequestMetric("bkhcm", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bkhcm", apiOperation.FullName())
	resp, err := apiOperation.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("call bk-hcm api %s failed, http code: %d, err: %s",
			apiOperation.FullName(), resp.StatusCode, errMsg)
	}

	if cast.ToInt(result["code"]) != 0 {
		return nil, errors.Errorf(
			"call bk-hcm api %s failed: %s",
			apiOperation.FullName(),
			cast.ToString(result["message"]),
		)
	}
	return result, nil
}
