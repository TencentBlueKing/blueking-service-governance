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

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// ConfigStubClient 配置管理 BSCP API 的 stub 实现
type ConfigStubClient struct {
	user         auth.User
	createdHooks map[string]int64
	nextHookID   int64
}

// NewConfigStub 创建配置管理 Stub
func NewConfigStub(user auth.User) *ConfigStubClient {
	return &ConfigStubClient{user: user, createdHooks: make(map[string]int64), nextHookID: 100}
}

// ListProjects stub
func (s *ConfigStubClient) ListProjects(ctx context.Context, bizID string) ([]Project, error) {
	log.Infof(ctx, "ConfigStub: ListProjects bizID=%s", bizID)
	return []Project{{ID: 1, Spec: ProjectSpec{Name: "Default Project", Key: "BK-BSCP-00001", IsDefault: true}}}, nil
}

// GetProjectByKey stub
func (s *ConfigStubClient) GetProjectByKey(ctx context.Context, bizID, key string) (*Project, error) {
	log.Infof(ctx, "ConfigStub: GetProjectByKey bizID=%s, key=%s", bizID, key)
	return &Project{ID: 1, Spec: ProjectSpec{Name: "Default Project", Key: key, IsDefault: true}}, nil
}

// ListEnvironments stub
func (s *ConfigStubClient) ListEnvironments(
	ctx context.Context,
	bizID string,
	projectID int64,
) (*EnvironmentsResp, error) {
	log.Infof(ctx, "ConfigStub: ListEnvironments bizID=%s, projectID=%d", bizID, projectID)
	return &EnvironmentsResp{
		ProdEnvironments: []Environment{{ID: 1, Spec: EnvironmentSpec{Name: "Default", Type: "prod"}}},
		DevEnvironments:  []Environment{{ID: 2, Spec: EnvironmentSpec{Name: "dev", Type: "dev"}}},
	}, nil
}

// CreateEnvironment stub
func (s *ConfigStubClient) CreateEnvironment(ctx context.Context, req *CreateEnvironmentReq) (*Environment, error) {
	log.Infof(ctx, "ConfigStub: CreateEnvironment name=%s", req.Name)
	return &Environment{ID: 999, Spec: EnvironmentSpec{Name: req.Name, Type: req.Type}}, nil
}

// CreateApp stub
func (s *ConfigStubClient) CreateApp(ctx context.Context, req *CreateAppReq) (*App, error) {
	log.Infof(ctx, "ConfigStub: CreateApp name=%s", req.Name)
	return &App{
		ID:         "1099",
		Name:       req.Name,
		Alias:      req.Alias,
		ConfigType: req.ConfigType,
		DataType:   req.DataType,
	}, nil
}

// ListEnvApps stub
func (s *ConfigStubClient) ListEnvApps(ctx context.Context, bizID string, projectID, envID int64) ([]App, error) {
	log.Infof(ctx, "ConfigStub: ListEnvApps bizID=%s, projectID=%d, envID=%d", bizID, projectID, envID)
	return nil, nil
}

// GetOrCreateApp stub
func (s *ConfigStubClient) GetOrCreateApp(ctx context.Context, req *CreateAppReq) (*App, error) {
	log.Infof(ctx, "ConfigStub: GetOrCreateApp name=%s", req.Name)
	return &App{
		ID:         "1099",
		Name:       req.Name,
		Alias:      req.Alias,
		ConfigType: req.ConfigType,
		DataType:   req.DataType,
	}, nil
}

// CreateCredential stub
func (s *ConfigStubClient) CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error) {
	log.Infof(ctx, "ConfigStub: CreateCredential name=%s", req.Name)
	return 1, nil
}

// ListCredentials stub
func (s *ConfigStubClient) ListCredentials(ctx context.Context, bizID string, projectID int64) ([]Credential, error) {
	log.Infof(ctx, "ConfigStub: ListCredentials bizID=%s, projectID=%d", bizID, projectID)
	return []Credential{{ID: 1, Name: "bkms-credential", Enable: true, EncCredential: "stub-token-xxxxx"}}, nil
}

// UpdateCredentialScope stub
func (s *ConfigStubClient) UpdateCredentialScope(ctx context.Context, _ *UpdateCredentialScopeReq) error {
	log.Infof(ctx, "ConfigStub: UpdateCredentialScope")
	return nil
}

// ListCredentialScopes stub
func (s *ConfigStubClient) ListCredentialScopes(
	ctx context.Context,
	bizID string, projectID, credentialID int64,
) ([]CredentialScope, error) {
	log.Infof(ctx, "ConfigStub: ListCredentialScopes credentialID=%d", credentialID)
	return []CredentialScope{{ID: 1, App: "stub-app", Scope: "/**"}}, nil
}

// CreateHook stub
func (s *ConfigStubClient) CreateHook(ctx context.Context, req *CreateHookReq) (int64, error) {
	log.Infof(ctx, "ConfigStub: CreateHook name=%s", req.Name)
	id := s.nextHookID
	s.nextHookID++
	s.createdHooks[req.Name] = id
	return id, nil
}

// ListHooks stub
func (s *ConfigStubClient) ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error) {
	log.Infof(ctx, "ConfigStub: ListHooks name=%s", req.Name)
	if req.Name != "" {
		if id, ok := s.createdHooks[req.Name]; ok {
			return &ListHooksResp{Count: 1, Details: []HookListItem{{Hook: Hook{ID: id, Name: req.Name}}}}, nil
		}
		return &ListHooksResp{Count: 0}, nil
	}
	return &ListHooksResp{Count: 0}, nil
}

// UpdateConfigHook stub
func (s *ConfigStubClient) UpdateConfigHook(ctx context.Context, _ *UpdateConfigHookReq) error {
	log.Infof(ctx, "ConfigStub: UpdateConfigHook")
	return nil
}
