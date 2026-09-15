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

package bscp

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// ConfigClient 配置管理 BSCP API 客户端接口（支持 Project/Environment 层级）
type ConfigClient interface {
	// === 项目 / 环境 ===

	// ListProjects 获取业务下的项目列表
	ListProjects(ctx context.Context, bizID string) ([]Project, error)
	// GetProjectByKey 根据 projectKey 获取项目信息
	GetProjectByKey(ctx context.Context, bizID, projectKey string) (*Project, error)
	// ListEnvironments 获取项目下的环境列表
	ListEnvironments(ctx context.Context, bizID string, projectID int64) (*EnvironmentsResp, error)
	// CreateEnvironment 在项目下创建环境
	CreateEnvironment(ctx context.Context, req *CreateEnvironmentReq) (*Environment, error)

	// === App ===

	// CreateApp 在项目环境下创建 BSCP App
	CreateApp(ctx context.Context, req *CreateAppReq) (*App, error)
	// ListEnvApps 获取项目环境下的 App 列表
	ListEnvApps(ctx context.Context, bizID string, projectID, envID int64) ([]App, error)
	// GetOrCreateApp 获取或创建 BSCP App（幂等，按名称匹配）
	GetOrCreateApp(ctx context.Context, req *CreateAppReq) (*App, error)

	// === Credential ===

	// CreateCredential 在项目下创建客户端密钥
	CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error)
	// ListCredentials 获取项目下的客户端密钥列表
	ListCredentials(ctx context.Context, bizID string, projectID int64) ([]Credential, error)
	// UpdateCredentialScope 更新项目下密钥关联服务规则
	UpdateCredentialScope(ctx context.Context, req *UpdateCredentialScopeReq) error
	// ListCredentialScopes 获取项目下密钥关联服务列表
	ListCredentialScopes(ctx context.Context, bizID string, projectID, credentialID int64) ([]CredentialScope, error)

	// === Hook ===

	// CreateHook 在项目下创建脚本
	CreateHook(ctx context.Context, req *CreateHookReq) (int64, error)
	// ListHooks 获取项目下脚本列表
	ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error)
	// UpdateConfigHook 更新 App 绑定的前后置脚本
	UpdateConfigHook(ctx context.Context, req *UpdateConfigHookReq) error
}

// NewConfigClient 创建配置管理 BSCP API 客户端
func NewConfigClient(user auth.User) (ConfigClient, error) {
	if config.G.Development.UseStubBSCP {
		log.InfoNoContext("use stub bscp config client according to config")
		return NewConfigStub(user), nil
	}

	client, err := newApiClient(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bscp config client")
	}

	return &ConfigApiClient{ApiClient: client}, nil
}

// ConfigApiClient 配置管理 BSCP API 客户端实现
type ConfigApiClient struct {
	*ApiClient
}
