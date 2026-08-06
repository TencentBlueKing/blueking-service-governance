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

// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装
package bscp

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var (
	// stubBizs 本地开发时返回的固定业务列表
	stubBizs = []Biz{
		{ID: "100001", Name: "stub-biz-a"},
		{ID: "100002", Name: "stub-biz-b"},
	}

	// stubServices 本地开发时返回的固定服务列表
	stubServices = []Service{
		{
			ID:         "1001",
			Name:       "stub-service-file",
			Alias:      "Stub 文件服务",
			Desc:       "本地开发用 stub 文件型服务",
			ConfigType: ConfigTypeFile,
			DataType:   DataTypeAny,
		},
		{
			ID:         "1002",
			Name:       "stub-service-kv",
			Alias:      "Stub 键值服务",
			Desc:       "本地开发用 stub 键值型服务",
			ConfigType: ConfigTypeKV,
			DataType:   DataTypeString,
		},
	}

	// stubVersions 本地开发时返回的固定版本列表
	stubVersions = Versions{
		{
			ID:              "2001",
			Name:            "v1.0.0",
			Desc:            "stub 初始版本",
			IsFullyReleased: true,
			Creator:         "stub-user",
			CreatedAt:       "2024-01-01 00:00:00",
		},
	}

	// stubCredentials 本地开发时返回的固定凭证列表
	stubCredentials = []Credential{
		{
			ID:               1,
			Name:             "bkms-credential",
			Memo:             "stub credential",
			Enable:           true,
			CredentialType:   "bearToken",
			EncCredential:    "stub-token-xxxxx",
			CredentialScopes: []string{"stub-service-file"},
		},
	}
)

// StubApiClient 测试用的 BSCP API 客户端实现，返回模拟数据
type StubApiClient struct {
	user auth.User

	// createdHooks 记录已创建的脚本（name -> id），用于模拟 ListHooks 的按名查询与幂等行为
	createdHooks map[string]int64
	// nextHookID 下一个分配的脚本 ID
	nextHookID int64
}

// NewStub 创建 StubApiClient
func NewStub(user auth.User) *StubApiClient {
	return &StubApiClient{
		user:         user,
		createdHooks: make(map[string]int64),
		nextHookID:   100,
	}
}

// ListUserBizs 模拟获取用户有权限的业务列表，返回 stubBizs
func (s *StubApiClient) ListUserBizs(ctx context.Context) ([]Biz, error) {
	log.Infof(ctx, "Stub: ListUserBizs request: user=%s", s.user.ID)
	return stubBizs, nil
}

// GetBiz 模拟获取指定业务信息
func (s *StubApiClient) GetBiz(ctx context.Context, bizID string) (*Biz, error) {
	log.Infof(ctx, "Stub: GetBiz request: bizID=%s", bizID)
	for i := range stubBizs {
		if stubBizs[i].ID == bizID {
			biz := stubBizs[i]
			return &biz, nil
		}
	}
	return &Biz{ID: bizID, Name: fmt.Sprintf("stub-biz-%s", bizID)}, nil
}

// CreateService 模拟创建 BSCP 服务
func (s *StubApiClient) CreateService(ctx context.Context, req *CreateServiceReq) (*Service, error) {
	log.Infof(ctx, "Stub: CreateService request: bizID=%s, name=%s", req.BizID, req.Name)
	return &Service{
		ID:         "1099",
		Name:       req.Name,
		Alias:      req.Alias,
		Desc:       req.Memo,
		ConfigType: req.ConfigType,
		DataType:   req.DataType,
	}, nil
}

// ListBizServices 模拟获取业务下的服务列表，返回 stubServices
func (s *StubApiClient) ListBizServices(ctx context.Context, bizID string) ([]Service, error) {
	log.Infof(ctx, "Stub: ListBizServices request: bizID=%s", bizID)
	return stubServices, nil
}

// GetBizService 模拟获取指定服务
func (s *StubApiClient) GetBizService(ctx context.Context, bizID, svcID string) (*Service, error) {
	log.Infof(ctx, "Stub: GetBizService request: bizID=%s, svcID=%s", bizID, svcID)
	for i := range stubServices {
		if stubServices[i].ID == svcID {
			svc := stubServices[i]
			return &svc, nil
		}
	}
	return &Service{
		ID:         svcID,
		Name:       fmt.Sprintf("stub-service-%s", svcID),
		Alias:      "Stub 服务",
		ConfigType: ConfigTypeFile,
		DataType:   DataTypeAny,
	}, nil
}

// ListServiceVersions 模拟获取服务下版本列表，返回 stubVersions
func (s *StubApiClient) ListServiceVersions(ctx context.Context, bizID, svcID string) (Versions, error) {
	log.Infof(ctx, "Stub: ListServiceVersions request: bizID=%s, svcID=%s", bizID, svcID)
	return stubVersions, nil
}

// ListServiceConfigs 模拟获取服务下的配置项列表
func (s *StubApiClient) ListServiceConfigs(ctx context.Context, bizID, svcID, versionID string) ([]Config, error) {
	log.Infof(ctx, "Stub: ListServiceConfigs request: bizID=%s, svcID=%s, versionID=%s", bizID, svcID, versionID)
	return []Config{
		NewKeyValue("app_name", "stub-app", "应用名称"),
		NewKeyValue("log_level", "info", "日志级别"),
	}, nil
}

// GetServiceConfig 模拟获取指定的配置项
func (s *StubApiClient) GetServiceConfig(ctx context.Context, bizID, svcID, versionID, id string) (Config, error) {
	log.Infof(
		ctx, "Stub: GetServiceConfig request: bizID=%s, svcID=%s, versionID=%s, id=%s",
		bizID, svcID, versionID, id,
	)
	return NewKeyValue("stub-config", "stub-value", "stub 配置项"), nil
}

// GetConfigContent 模拟获取配置项的内容
func (s *StubApiClient) GetConfigContent(ctx context.Context, bizID, svcID, versionID, id string) (string, error) {
	log.Infof(
		ctx, "Stub: GetConfigContent request: bizID=%s, svcID=%s, versionID=%s, id=%s",
		bizID, svcID, versionID, id,
	)
	return "stub-config-content", nil
}

// GetOrCreateService 模拟获取或创建 BSCP 服务
func (s *StubApiClient) GetOrCreateService(ctx context.Context, req *CreateServiceReq) (*Service, error) {
	log.Infof(ctx, "Stub: GetOrCreateService request: bizID=%s, name=%s", req.BizID, req.Name)
	// 先查找是否已存在
	for i := range stubServices {
		if stubServices[i].Name == req.Name {
			svc := stubServices[i]
			return &svc, nil
		}
	}
	return &Service{
		ID:         "1099",
		Name:       req.Name,
		Alias:      req.Alias,
		Desc:       req.Memo,
		ConfigType: req.ConfigType,
		DataType:   req.DataType,
	}, nil
}

// CreateCredential 模拟创建客户端密钥
func (s *StubApiClient) CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error) {
	log.Infof(ctx, "Stub: CreateCredential request: bizID=%s, name=%s", req.BizID, req.Name)
	return 1, nil
}

// ListCredentials 模拟获取业务下的客户端密钥列表
func (s *StubApiClient) ListCredentials(ctx context.Context, bizID string) ([]Credential, error) {
	log.Infof(ctx, "Stub: ListCredentials request: bizID=%s", bizID)
	return stubCredentials, nil
}

// UpdateCredential 模拟更新客户端密钥
func (s *StubApiClient) UpdateCredential(ctx context.Context, req *UpdateCredentialReq) error {
	log.Infof(ctx, "Stub: UpdateCredential request: bizID=%s, id=%d", req.BizID, req.ID)
	return nil
}

// CheckCredentialName 模拟检测客户端密钥名称是否已存在
func (s *StubApiClient) CheckCredentialName(ctx context.Context, bizID, name string) (bool, error) {
	log.Infof(ctx, "Stub: CheckCredentialName request: bizID=%s, name=%s", bizID, name)
	for _, cred := range stubCredentials {
		if cred.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// UpdateCredentialScope 模拟更新客户端密钥关联服务规则
func (s *StubApiClient) UpdateCredentialScope(ctx context.Context, req *UpdateCredentialScopeReq) error {
	log.Infof(ctx, "Stub: UpdateCredentialScope request: bizID=%s, credentialID=%s", req.BizID, req.CredentialID)
	return nil
}

// ListCredentialScopes 模拟获取客户端密钥关联服务列表
func (s *StubApiClient) ListCredentialScopes(
	ctx context.Context,
	bizID, credentialID string,
) ([]CredentialScope, error) {
	log.Infof(ctx, "Stub: ListCredentialScopes request: bizID=%s, credentialID=%s", bizID, credentialID)
	return []CredentialScope{
		{ID: 1, App: "stub-service-file", Scope: "/**"},
	}, nil
}

// CreateHook 模拟创建脚本（记录已创建脚本以支持幂等查询）
func (s *StubApiClient) CreateHook(ctx context.Context, req *CreateHookReq) (int64, error) {
	log.Infof(ctx, "Stub: CreateHook request: bizID=%s, name=%s, type=%s", req.BizID, req.Name, req.Type)
	if s.createdHooks == nil {
		s.createdHooks = make(map[string]int64)
	}
	if s.nextHookID == 0 {
		s.nextHookID = 100
	}
	id := s.nextHookID
	s.nextHookID++
	s.createdHooks[req.Name] = id
	return id, nil
}

// DeleteHook 模拟删除脚本
func (s *StubApiClient) DeleteHook(ctx context.Context, req *DeleteHookReq) error {
	log.Infof(ctx, "Stub: DeleteHook request: bizID=%s, hookID=%d, force=%v", req.BizID, req.HookID, req.Force)
	return nil
}

// GetHook 模拟获取脚本详情
func (s *StubApiClient) GetHook(ctx context.Context, bizID string, hookID int64) (*Hook, error) {
	log.Infof(ctx, "Stub: GetHook request: bizID=%s, hookID=%d", bizID, hookID)
	return &Hook{
		ID:       hookID,
		Name:     "stub-hook",
		Type:     "shell",
		Tags:     []string{"test"},
		Memo:     "stub hook script",
		Creator:  "stub-user",
		CreateAt: "2024-01-01 00:00:00",
		UpdateAt: "2024-01-01 00:00:00",
	}, nil
}

// GetReleaseHook 模拟获取版本绑定的前后置脚本
func (s *StubApiClient) GetReleaseHook(
	ctx context.Context, bizID string, appID, releaseID int64,
) (*ReleaseHook, error) {
	log.Infof(ctx, "Stub: GetReleaseHook request: bizID=%s, appID=%d, releaseID=%d", bizID, appID, releaseID)
	return &ReleaseHook{
		PreHook: &ReleaseHookDetail{
			HookID:           1,
			HookName:         "stub-pre-hook",
			HookRevisionID:   1,
			HookRevisionName: "v1",
			Type:             "shell",
			Content:          "#!/bin/bash\necho 'pre hook'",
		},
		PostHook: &ReleaseHookDetail{
			HookID:           2,
			HookName:         "stub-post-hook",
			HookRevisionID:   1,
			HookRevisionName: "v1",
			Type:             "shell",
			Content:          "#!/bin/bash\necho 'post hook'",
		},
	}, nil
}

// ListHooks 模拟获取脚本列表（按名称返回已创建脚本，模拟幂等查询）
func (s *StubApiClient) ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error) {
	log.Infof(ctx, "Stub: ListHooks request: bizID=%s, name=%s", req.BizID, req.Name)

	// 按名称查询：返回已创建的同名脚本；未创建则返回空列表
	if req.Name != "" {
		if id, ok := s.createdHooks[req.Name]; ok {
			return &ListHooksResp{
				Count: 1,
				Details: []HookListItem{
					{Hook: Hook{ID: id, Name: req.Name, Type: "shell"}},
				},
			}, nil
		}
		return &ListHooksResp{Count: 0}, nil
	}

	// 无名称查询：返回默认脚本列表
	return &ListHooksResp{
		Count: 1,
		Details: []HookListItem{
			{
				Hook: Hook{
					ID:       1,
					Name:     "stub-hook",
					Type:     "shell",
					Tags:     []string{"test"},
					Memo:     "stub hook script",
					Creator:  "stub-user",
					CreateAt: "2024-01-01 00:00:00",
					UpdateAt: "2024-01-01 00:00:00",
				},
				BoundNum:            1,
				ConfirmDelete:       false,
				PublishedRevisionID: 1,
			},
		},
	}, nil
}

// UpdateConfigHook 模拟更新服务绑定的前后置脚本
func (s *StubApiClient) UpdateConfigHook(ctx context.Context, req *UpdateConfigHookReq) error {
	log.Infof(ctx, "Stub: UpdateConfigHook request: bizID=%s, appID=%d, preHookID=%d, postHookID=%d",
		req.BizID, req.AppID, req.PreHookID, req.PostHookID)
	return nil
}

// UpdateHook 模拟更新脚本信息
func (s *StubApiClient) UpdateHook(ctx context.Context, req *UpdateHookReq) error {
	log.Infof(ctx, "Stub: UpdateHook request: bizID=%s, hookID=%d", req.BizID, req.HookID)
	return nil
}
