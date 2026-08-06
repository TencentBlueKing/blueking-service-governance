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

// Package dbm provides api client to dbm（蓝鲸数据库管理平台）
package dbm

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// apiName DBM 客户端名称，用于 metrics 上报与日志
const apiName = "bkdbm"

const defaultHTTPTimeout = 30 * time.Second

// ApiClient DBM API 客户端实现
type ApiClient struct {
	define.BkApiClient
	appCode   string
	appSecret string
}

// compile-time check
var _ Client = (*ApiClient)(nil)

// New 创建 DBM API 客户端，根据配置返回真实客户端或 stub 客户端
// 使用蓝鲸应用账号（bk_app_code / bk_app_secret）进行认证；
// bk_username（DBM 网关要求的用户身份）在每次 API 调用时由调用方按操作发起人传入。
func New(appCode, appSecret string) (Client, error) {
	// 测试时使用 stub 客户端
	if config.G.Development.UseStubDBM {
		slog.Info("use stub dbm client according to config")
		return NewStub(), nil
	}

	if appCode == "" || appSecret == "" {
		return nil, errors.New("dbm appCode and appSecret are required")
	}

	client, err := bkapi.NewBkApiClient(apiName, bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:        config.G.BkApiStages.BkDBM,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(defaultHTTPTimeout),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dbm api client")
	}
	return &ApiClient{
		BkApiClient: client,
		appCode:     appCode,
		appSecret:   appSecret,
	}, nil
}

// --- internal helpers ---

// GetTicketStatus 查询工单状态
func (c *ApiClient) GetTicketStatus(ctx context.Context, ticketID int, username string) (*TicketInfo, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "get_ticket_status",
			Method: "GET",
			Path:   "/tickets/{ticket_id}/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"ticket_id": strconv.Itoa(ticketID),
		}),
	).SetQueryParams(map[string]string{"is_reviewed": "0"}).SetHeaders(c.authHeaders(username))

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	return &TicketInfo{
		ID:     cast.ToInt(mapx.Get(result, "data.id", 0)),
		Status: normalizeTicketStatus(mapx.GetStr(result, "data.status")),
	}, nil
}

// authHeaders 构造 DBM API 鉴权请求头（X-Bkapi-Authorization）
// username 为操作发起人（bk_username），DBM 网关要求用户身份。
func (c *ApiClient) authHeaders(username string) map[string]string {
	payload := map[string]string{
		"bk_app_code":   c.appCode,
		"bk_app_secret": c.appSecret,
		"bk_username":   username,
	}
	raw, _ := json.Marshal(payload)
	return map[string]string{"X-Bkapi-Authorization": string(raw)}
}

// handleOperation 发起请求并检查结果，返回响应体 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric(apiName, opName, started, &err)

	ctx, span := apm.StartClientSpan(ctx, apiName, opName)
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
			"call dbm api %s failed, status code: %d, err: %s", opName, resp.StatusCode, errMsg,
		)
	}

	// 业务返回码检查（DBM 返回 code/result/message）
	code := cast.ToInt(mapx.Get(result, "code", 0))
	if code != 0 || !cast.ToBool(mapx.Get(result, "result", false)) {
		return nil, errors.Errorf(
			"call dbm api %s failed, code: %d, message: %s", opName, code, mapx.GetStr(result, "message"),
		)
	}
	return result, nil
}

// newCreateTicketOperation 构造提交工单的 operation（创建 / 禁用 / 删除共用 /tickets/ 接口）
func (c *ApiClient) newCreateTicketOperation(payload any, username string) define.Operation {
	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_ticket",
			Method: "POST",
			Path:   "/tickets/",
		},
		bkapi.OptSetRequestBody(payload),
	).SetHeaders(c.authHeaders(username))
}

// normalizeTicketStatus 标准化工单状态值
func normalizeTicketStatus(status string) TicketStatus {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCEEDED":
		return TicketStatusSucceeded
	case "FAILED":
		return TicketStatusFailed
	case "REVOKED", "TERMINATED":
		return TicketStatusTerminated
	case "PENDING":
		return TicketStatusPending
	default:
		return TicketStatusRunning
	}
}
