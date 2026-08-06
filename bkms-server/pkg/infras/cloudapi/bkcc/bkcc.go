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

// Package bkcc provides api client to bkcc（蓝鲸配置平台）
package bkcc

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// ApiClient bk-cmdb API 客户端（用户态）
type ApiClient struct {
	// user 用户态
	user auth.User

	// BkApiClient bk-cmdb API 客户端
	define.BkApiClient
}

// New 创建 bkcc Client，根据配置返回真实客户端或 stub 客户端
func New(user auth.User) (Client, error) {
	// 测试时使用 stub 客户端
	if config.G.Development.UseStubBkCMDB {
		log.InfoNoContext("use stub bkcc client according to config")
		return NewStub(user), nil
	}

	authInfo, err := generateAuthInfo(user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate auth info")
	}

	client, err := bkapi.NewBkApiClient(apiName, buildClientConfig(authInfo))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create bk-cmdb api client")
	}

	return &ApiClient{BkApiClient: client, user: user}, nil
}

// generateAuthInfo 生成鉴权信息
func generateAuthInfo(user auth.User) (string, error) {
	authorization, err := json.Marshal(map[string]string{
		"bk_app_code":       config.G.BkApp.Code,
		"bk_app_secret":     config.G.BkApp.Secret,
		"bk_username":       user.ID,
		user.Cred.CredKey(): user.Cred.CredValue(),
	})
	if err != nil {
		return "", err
	}

	return string(authorization), nil
}

// buildClientConfig 构建客户端配置
func buildClientConfig(authInfo string) bkapi.ClientConfig {
	return bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:        config.G.BkApiStages.BkCC,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("x-bkapi-authorization", authInfo),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	}
}

// handleOperation 发起请求并检查结果
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric(apiName, apiOperation.FullName(), started, &err)

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
			"call bkcc api %s failed, status code: %d, err: %s", opName, resp.StatusCode, errMsg,
		)
	}

	// 根据业务返回码判断是否失败
	code := cast.ToInt(result["code"])
	if code != 0 {
		msg := mapx.GetStr(result, "message")
		return nil, errors.Errorf(
			"call bkcc api %s failed, code: %d, message: %s", opName, code, msg,
		)
	}

	return result, nil
}
