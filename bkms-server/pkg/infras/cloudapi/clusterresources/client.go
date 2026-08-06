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

// Package clusterresources provides api client to bcs-cluster-resources（集群资源）
package clusterresources

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// ApiClient ClusterResources API 客户端（用户态）
// 注：与 infras/cloudapi/bcs 不同，此客户端为用户态访问
type ApiClient struct {
	define.BkApiClient
	user auth.User
}

// New 创建 ApiClient
func New(user auth.User) (*ApiClient, error) {
	authorization, _ := json.Marshal(map[string]string{
		"bk_app_code":       config.G.BkApp.Code,
		"bk_app_secret":     config.G.BkApp.Secret,
		"bk_username":       user.ID,
		user.Cred.CredKey(): user.Cred.CredValue(),
	})
	client, err := bkapi.NewBkApiClient("bcs-api-gateway", bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		// 注：ClusterResources 与 BCS 用同一个 api stage 即可
		Stage: config.G.BkApiStages.BCS,
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

// ListEvents 获取事件列表
func (c *ApiClient) ListEvents(
	ctx context.Context,
	projectCode, clusterID string,
	params ListEventParams,
) (*PaginatedEvents, error) {
	// 构造请求体
	body := map[string]any{
		"clusterId": clusterID,
		"offset":    (params.Page - 1) * params.PageSize,
		"length":    params.PageSize,
		"env":       "k8s",
	}
	if params.Level != "" {
		body["level"] = params.Level
	}
	if params.StartedAt != 0 {
		body["timeBegin"] = params.StartedAt
	}
	if params.EndedAt != 0 {
		body["timeEnd"] = params.EndedAt
	}
	if params.ComponentName != "" {
		body["component"] = params.ComponentName
	}
	if params.Namespace != "" {
		body["extraInfo.namespace"] = params.Namespace
	}
	// kind 支持多个值用 , 拼接
	if len(params.ResourceKinds) > 0 {
		body["kind"] = strings.Join(params.ResourceKinds, ",")
	}
	// extraInfo.name 支持多个值用 , 拼接
	if len(params.ResourceNames) > 0 {
		body["extraInfo.name"] = strings.Join(params.ResourceNames, ",")
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "cluster_resources_list_events",
			Method: "POST",
			Path:   "/clusterresources/api/v1/projects/{projectCode}/clusters/{clusterID}/events",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectCode": projectCode,
			"clusterID":   clusterID,
		}),
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	total := cast.ToInt64(result["total"])

	var events []EventEntry
	data, _ := mapx.Get(result, "data", nil).([]any)
	for _, d := range data {
		if e, ok := d.(map[string]any); ok {
			// 解析事件时间
			var createdAt time.Time
			if t := mapx.GetStr(e, "eventTime"); t != "" {
				createdAt, _ = time.Parse(time.RFC3339, t)
			}

			events = append(events, EventEntry{
				ClusterID:     mapx.GetStr(e, "clusterId"),
				Namespace:     mapx.GetStr(e, "extraInfo.namespace"),
				Level:         mapx.GetStr(e, "level"),
				Content:       mapx.GetStr(e, "describe"),
				Type:          mapx.GetStr(e, "type"),
				ComponentName: mapx.GetStr(e, "component"),
				ResourceKind:  mapx.GetStr(e, "extraInfo.kind"),
				ResourcesName: mapx.GetStr(e, "extraInfo.name"),
				CreatedAt:     createdAt,
			})

		}
	}

	return &PaginatedEvents{Count: total, Data: events}, nil
}

// handleOperation 发起请求并检查结果，返回响应体 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric("clusterresources", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "clusterresources", opName)
	resp, err := apiOperation.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// HTTP 状态码检查
	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf(
			"call bcs cluster resources api %s failed, status code: %d, err: %s", opName, resp.StatusCode, errMsg,
		)
	}

	// 根据返回码判断是否失败
	if code := cast.ToInt(result["code"]); code != 0 {
		return nil, errors.Errorf(
			"call bcs cluster resources api %s failed, code: %d, message: %s",
			opName, code, mapx.GetStr(result, "message"),
		)
	}
	return result, nil
}
