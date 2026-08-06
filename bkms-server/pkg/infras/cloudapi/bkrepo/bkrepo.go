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

// Package bkrepo provides api client to bkrepo（蓝盾制品库）
package bkrepo

import (
	"context"
	"encoding/base64"
	"fmt"
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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// ApiClient 蓝盾制品库 API 客户端
type ApiClient struct {
	define.BkApiClient
	operator string
}

// New 创建 Client，根据配置返回真实客户端或 stub 客户端
// 需注意：使用平台账号进行认证
// ref: https://github.com/TencentBlueKing/bk-repo/blob/master/docs/apidoc-user/common/common.md
func New(operator string) (Client, error) {
	// 测试时使用 stub 客户端
	if config.G.Development.UseStubBkRepo {
		log.InfoNoContext("use stub bkrepo client according to config")
		return NewStub(operator), nil
	}

	authorization := fmt.Sprintf(
		"Platform %s", base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", config.G.BKRepo.Username, config.G.BKRepo.Password)),
		),
	)

	client, err := bkapi.NewBkApiClient("bkrepo", bkapi.ClientConfig{
		Endpoint: config.G.BKRepo.BaseUrl,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("Authorization", authorization),
			bkapi.OptSetRequestHeader("X-BKREPO-UID", operator),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	})
	if err != nil {
		return nil, err
	}
	return &ApiClient{client, operator}, nil
}

// ------------------------------------------ 蓝盾制品库用户管理 API ------------------------------------------

// CreateUserToProject 创建用户（公共账号）并绑定为项目管理员
// ref: https://github.com/TencentBlueKing/bk-repo/blob/master/docs/apidoc-user/auth/user.md
func (c *ApiClient) CreateUserToProject(
	ctx context.Context, projectID, username, password string, associatedUsers []string,
) error {
	body := map[string]any{
		"userId":    username,
		"name":      username,
		"pwd":       password,
		"asstUsers": associatedUsers,
		"projectId": projectID,
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_user_to_project",
			Method: "POST",
			Path:   "/auth/api/user/create/project",
		},
		bkapi.OptSetRequestBody(body),
	)

	return c.handleOperationWithoutResult(ctx, apiOperation)
}

// ------------------------------------------ 蓝盾制品库项目管理 API ------------------------------------------

// CreateProject 创建制品库项目
// ref: https://github.com/TencentBlueKing/bk-repo/blob/master/docs/apidoc-user/repo/project.md
func (c *ApiClient) CreateProject(ctx context.Context, projectID string) error {
	body := map[string]any{
		"name":        projectID,
		"displayName": projectID,
		"description": "蓝鲸服务治理",
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_project",
			Method: "POST",
			Path:   "/repository/api/project/create",
		},
		bkapi.OptSetRequestBody(body),
	)

	return c.handleOperationWithoutResult(ctx, apiOperation)
}

// ------------------------------------------ 蓝盾制品库仓库管理 API ------------------------------------------

// CreateRepository 创建制品库仓库
// ref: https://github.com/TencentBlueKing/bk-repo/blob/master/docs/apidoc-user/repo/repository.md
func (c *ApiClient) CreateRepository(
	ctx context.Context, projectID, repoName, repoType, description string, isPublic bool,
) error {
	body := map[string]any{
		"projectId":   projectID,
		"name":        repoName,
		"type":        repoType,
		"public":      isPublic,
		"description": description,
		"category":    "LOCAL",
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_repo",
			Method: "POST",
			Path:   "/repository/api/repo/create",
		},
		bkapi.OptSetRequestBody(body),
	)

	return c.handleOperationWithoutResult(ctx, apiOperation)
}

// handleOperation 发起请求并检查结果，返回响应体 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric("bkrepo", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bkrepo", opName)
	resp, err := apiOperation.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 根据返回码判断是否失败
	code := cast.ToInt(mapx.Get(result, "code", 0))
	// 针对特定的制品库错误码，做特殊处理
	switch code {
	case projectExistCode:
		// 如果制品库项目已经存在，则不认为是错误
		return nil, nil
	case repositoryExistCode:
		// 如果制品库仓库已经存在，则不认为是错误
		return nil, nil
	}

	// HTTP 状态码检查
	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf(
			"call bkrepo api %s failed, status code: %d, err: %s",
			opName, resp.StatusCode, errMsg,
		)
	}

	// 其他错误 -> 制品库 API 错误
	if code != 0 {
		return nil, errors.Errorf(
			"call bkrepo api %s failed, code: %d, message: %s",
			opName, code, mapx.GetStr(result, "message"),
		)
	}
	return result, nil
}

// handleOperationWithoutResult 发起请求，忽略返回数据但仍处理错误
// 本方法是 handleOperation 的简化版本，用于不需要返回数据的场景（如：Create 操作）
func (c *ApiClient) handleOperationWithoutResult(ctx context.Context, apiOperation define.Operation) error {
	_, err := c.handleOperation(ctx, apiOperation)
	return err
}
