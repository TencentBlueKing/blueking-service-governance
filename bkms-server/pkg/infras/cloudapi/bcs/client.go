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

package bcs

import (
	"context"
	"encoding/json"
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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// Client BCS API 客户端接口
type Client interface {
	// ListAuthorizedProjects 获取有权限的项目列表
	ListAuthorizedProjects(ctx context.Context) ([]Project, error)
	// GetProject 根据项目 id, 获取项目详情
	GetProject(ctx context.Context, id string) (*Project, error)
	// ListClustersByProject 获取项目下的集群列表
	ListClustersByProject(ctx context.Context, projectID string) ([]Cluster, error)
	// ListNamespacesByCluster 获取集群下的命名空间列表
	ListNamespacesByCluster(ctx context.Context, projectID, clusterID string) ([]Namespace, error)
	// CreateWebConsole 创建 web console 会话，返回 web console 访问链接
	CreateWebConsole(
		ctx context.Context,
		projectID, clusterID, namespace, podName, containerName, command string,
	) (string, error)
	// ListUserTokens 获取用户的 BCS Token 列表
	ListUserTokens(ctx context.Context) ([]UserToken, error)
}

// ApiClient api client
type ApiClient struct {
	define.BkApiClient
	user auth.User
}

// ClientOption 用于配置 BCS API 客户端的可选参数
type ClientOption struct {
	// UserMode 是否使用用户态认证
	UserMode bool
}

// New 创建 BCS API 客户端实例
//
// 默认使用应用认证模式：通过 bk_app_code 和 bk_app_secret 进行应用身份验证，
// BCS 后端通过解析 X-Bcs-Username 头获取用户信息。
//
// 当 opts.UserMode 为 true 时，使用用户态认证模式：在 x-bkapi-authorization 中
// 同时包含应用凭据和用户凭据（access_token 或 bk_ticket），适用于需要验证用户身份的接口。
func New(user auth.User, opts ...ClientOption) (Client, error) {
	if config.G.Development.UseStubBCS {
		log.InfoNoContext("use stub bcs client according to config")
		return NewStub(user), nil
	}

	var opt ClientOption
	if len(opts) > 0 {
		opt = opts[0]
	}

	authorization := buildAuthorization(user, opt.UserMode)
	clientOptions := buildClientOptions(user, opt.UserMode, authorization)

	client, err := bkapi.NewBkApiClient("bcs-api-gateway", bkapi.ClientConfig{
		BkApiUrlTmpl:  config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:         config.G.BkApiStages.BCS,
		ClientOptions: clientOptions,
	})
	if err != nil {
		return nil, err
	}
	return &ApiClient{client, user}, nil
}

// buildAuthorization 构造 x-bkapi-authorization 头的 JSON 值
func buildAuthorization(user auth.User, userMode bool) string {
	authMap := map[string]string{
		"bk_app_code":   config.G.BkApp.Code,
		"bk_app_secret": config.G.BkApp.Secret,
	}
	if userMode {
		authMap[user.Cred.CredKey()] = user.Cred.CredValue()
	}
	data, _ := json.Marshal(authMap)
	return string(data)
}

// buildClientOptions 构造 BkApiClient 选项列表
func buildClientOptions(user auth.User, userMode bool, authorization string) []define.BkApiClientOption {
	options := []define.BkApiClientOption{
		bkapi.OptSetRequestHeader("x-bkapi-authorization", authorization),
		bkapi.OptJsonResultProvider(),
		bkapi.OptTimeout(60 * time.Second),
	}
	if !userMode {
		options = append(options,
			bkapi.OptSetRequestHeader("X-Bcs-Username", user.ID),
			bkapi.OptJsonBodyProvider(),
		)
	}
	return options
}

// ListAuthorizedProjects 获取有权限的项目列表
func (c *ApiClient) ListAuthorizedProjects(ctx context.Context) ([]Project, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{Name: "list_auth_projects", Method: "GET", Path: "/bcsproject/v1/authorized_projects"},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	projects := make([]Project, 0)
	for _, item := range mapx.GetList(result, "data.results") {
		if v, ok := item.(map[string]any); ok {
			projects = append(
				projects,
				Project{
					ID:          mapx.GetStr(v, "projectID"),
					Code:        mapx.GetStr(v, "projectCode"),
					Name:        mapx.GetStr(v, "name"),
					Kind:        mapx.GetStr(v, "kind"),
					Description: mapx.GetStr(v, "description"),
					BizID:       mapx.GetStr(v, "businessID"),
					IsOffline:   mapx.GetBool(v, "isOffline"),
				},
			)
		}
	}
	return projects, nil
}

// GetProject 根据项目 id, 获取项目详情
func (c *ApiClient) GetProject(ctx context.Context, id string) (*Project, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "get_project",
			Method: "GET",
			Path:   fmt.Sprintf("/bcsproject/v1/projects/%s", id),
		},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	data := mapx.GetMap(result, "data")
	return &Project{
		ID:          mapx.GetStr(data, "projectID"),
		Code:        mapx.GetStr(data, "projectCode"),
		Name:        mapx.GetStr(data, "name"),
		Kind:        mapx.GetStr(data, "kind"),
		Description: mapx.GetStr(data, "description"),
		BizID:       mapx.GetStr(data, "businessID"),
		IsOffline:   mapx.GetBool(data, "isOffline"),
	}, nil
}

// ListClustersByProject 获取项目下的集群列表
func (c *ApiClient) ListClustersByProject(ctx context.Context, projectID string) ([]Cluster, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_project_clusters",
			Method: "GET",
			Path:   fmt.Sprintf("/v4/clustermanager/v1/projects/%s/clusters", projectID),
		},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	clusters := make([]Cluster, 0)
	for _, item := range mapx.GetList(result, "data") {
		if v, ok := item.(map[string]any); ok {
			if mapx.GetStr(v, "engineType") != "k8s" {
				continue
			}

			clusters = append(
				clusters,
				Cluster{
					ID:          mapx.GetStr(v, "clusterID"),
					Name:        mapx.GetStr(v, "clusterName"),
					Type:        mapx.GetStr(v, "clusterType"),
					Environment: mapx.GetStr(v, "environment"),
					// NOTE: is_shared 并非格式错误, 而是接口本身返回不规范
					IsShared:    mapx.GetBool(v, "is_shared"),
					Description: mapx.GetStr(v, "description"),
					Status:      mapx.GetStr(v, "status"),
				},
			)
		}
	}
	return clusters, nil
}

// ListNamespacesByCluster 获取集群下的命名空间列表
func (c *ApiClient) ListNamespacesByCluster(ctx context.Context, projectID, clusterID string) ([]Namespace, error) {
	// NOTE: 目前, BCS 的联邦集群采用另外一套接口. 暂时只考虑非联邦集群的场景
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_native_namespaces",
			Method: "GET",
			Path: fmt.Sprintf(
				"/bcsproject/v1/projects/%s/clusters/%s/native/namespaces",
				projectID, clusterID,
			),
		},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	namespaces := make([]Namespace, 0)
	for _, item := range mapx.GetList(result, "data") {
		if v, ok := item.(map[string]any); ok {
			namespaces = append(
				namespaces,
				Namespace{
					Name:   mapx.GetStr(v, "name"),
					Status: mapx.GetStr(v, "status"),
				},
			)
		}
	}
	return namespaces, nil
}

// CreateWebConsole 创建 web console 会话，返回 web console 访问链接
func (c *ApiClient) CreateWebConsole(
	ctx context.Context, projectID, clusterID, namespace, podName, containerName, command string,
) (string, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_web_console_session",
			Method: "POST",
			Path:   "/v4/webconsole/api/portal/projects/{projectCode}/clusters/{clusterID}/web_console/sessions/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectCode": projectID,
			"clusterID":   clusterID,
		}),
		bkapi.OptSetRequestBody(map[string]any{
			"namespace":      namespace,
			"pod_name":       podName,
			"container_name": containerName,
			"operator":       auth.MustGetUser(ctx).ID,
			"command":        command,
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return "", err
	}

	return mapx.GetStr(result, "data.web_console_url"), nil
}

// handleOperation 发起请求并检查结果，返回响应数据 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	defer metrics.ClientRequest("bcs", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bcs", apiOperation.FullName())
	resp, err := apiOperation.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		if len(errMsg) == 0 {
			errMsg, _ = json.Marshal(result)
		}
		return nil, errors.Errorf("call bcs api %s failed, http code: %d, err: %s",
			apiOperation.FullName(), resp.StatusCode, errMsg)
	}

	if cast.ToInt(result["code"]) != 0 {
		return nil, errors.New(mapx.GetStr(result, "message"))
	}
	return result, nil
}

// ListUserTokens 获取 BCS Auth Info
func (c *ApiClient) ListUserTokens(ctx context.Context) ([]UserToken, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_user_tokens",
			Method: "GET",
			Path:   fmt.Sprintf("/v4/usermanager/v1/users/%s/tokens", c.user.ID),
		},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	tokens := make([]UserToken, 0)
	for _, item := range mapx.GetList(result, "data") {
		if v, ok := item.(map[string]any); ok {
			token := UserToken{
				Token:  mapx.GetStr(v, "token"),
				Status: cast.ToInt(v["status"]),
			}
			if ea, exists := v["expired_at"]; exists && ea != nil {
				s := cast.ToString(ea)
				token.ExpiredAt = &s
			}
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}
