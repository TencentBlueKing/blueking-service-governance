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

// Package service 封装应用配置管理的业务逻辑层。
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	bscpworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// Manager 应用配置管理业务管理器。
type Manager struct {
	feedAddr string

	client bscpapi.ConfigClient

	configStore model.Store
}

// NewManager 创建 Manager
func NewManager(
	user auth.User,
	configStore model.Store,
) (*Manager, error) {
	configClient, err := bscpapi.NewConfigClient(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bscp config client")
	}

	return &Manager{
		client:      configClient,
		configStore: configStore,
		feedAddr:    svccfg.G.BSCP.FeedAddr,
	}, nil
}

// === Metadata 管理 ===

// InitMetadata 初始化配置管理（幂等）。
func (m *Manager) InitMetadata(
	ctx context.Context,
	params *InitMetadataParams,
) (*model.Metadata, error) {
	// 先检查是否已存在
	existing, err := m.configStore.GetMetadata(ctx, params.AppID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, model.ErrMetadataNotFound) {
		return nil, errors.Wrap(err, "get metadata")
	}

	// 获取或创建 Credential
	cred, err := m.GetOrCreateCredential(ctx, params.BscpBizID, cast.ToInt64(params.BscpProjectID))
	if err != nil {
		metrics.BscpcfgStepFailed("credential")
		return nil, errors.Wrap(err, "get or create bscpcfg credential")
	}

	// 获取或创建后置脚本
	hookID, err := m.GetOrCreatePostHook(ctx, params.BscpBizID, cast.ToInt64(params.BscpProjectID), params.AppID)
	if err != nil {
		metrics.BscpcfgStepFailed("post_hook")
		return nil, errors.Wrap(err, "get or create post hook")
	}

	meta := &model.Metadata{
		AppID:          params.AppID,
		BscpBizID:      params.BscpBizID,
		ProjectID:      params.BscpProjectID,
		ProjectKey:     params.BscpProjectKey,
		CredentialID:   fmt.Sprintf("%d", cred.ID),
		CredentialName: cred.Name,
		Token:          cred.EncCredential,
		FeedAddr:       m.feedAddr,
		WorkloadName:   params.WorkloadName,
		WorkloadKind:   params.WorkloadKind,
		PostHookID:     fmt.Sprintf("%d", hookID),
		Operator:       params.Operator,
	}

	if err = m.configStore.CreateMetadata(ctx, meta); err != nil {
		return nil, errors.Wrap(err, "create metadata")
	}

	return meta, nil
}

// PatchMetadata 批量更新 Metadata 字段。
func (m *Manager) PatchMetadata(ctx context.Context, appID string, updateData *model.MetadataUpdate) error {
	return m.configStore.UpdateMetadata(ctx, appID, updateData)
}

// === EnvBinding 管理 ===

// CreateEnvBinding 创建环境绑定。
func (m *Manager) CreateEnvBinding(
	ctx context.Context,
	params *CreateEnvBindingParams,
) (*model.Snapshot, error) {
	// 1. 检查 Metadata 是否存在
	meta, err := m.configStore.GetMetadata(ctx, params.AppID)
	if err != nil {
		if errors.Is(err, model.ErrMetadataNotFound) {
			return nil, errors.New("bscpcfg metadata not initialized, please run enable-bscpcfg CMD first")
		}
		return nil, errors.Wrap(err, "get metadata")
	}

	bizID := params.BscpBizID
	projectIDStr := params.Workspace.BkSystems.BkBSCPProjectID
	if projectIDStr == "" {
		return nil, errors.New("workspace missing BkBSCPProjectID, please run bind-bscp-project CMD first")
	}
	projectID := cast.ToInt64(projectIDStr)

	// 2. 按 envName 匹配或创建 BSCP 环境（类型由 bkms 环境类型映射而来）
	bscpEnv, err := m.getOrCreateBscpEnv(ctx, bizID, projectID, params.EnvName, params.EnvType)
	if err != nil {
		metrics.BscpcfgStepFailed("bscp_env")
		return nil, errors.Wrap(err, "get or create bscp environment")
	}

	envID := cast.ToString(bscpEnv.ID)

	// 3. 获取或创建 BSCP App
	bscpApp, err := m.client.GetOrCreateApp(ctx, &bscpapi.CreateAppReq{
		BizID:      bizID,
		ProjectID:  projectID,
		EnvID:      bscpEnv.ID,
		Name:       params.AppID,
		Alias:      params.AppID,
		ConfigType: bscpapi.ConfigTypeFile,
		DataType:   bscpapi.DataTypeAny,
	})
	if err != nil {
		metrics.BscpcfgStepFailed("bscp_app")
		return nil, errors.Wrap(err, "get or create bscp app")
	}

	// 3.1 刷新 workspace 权限范围（将 BSCP App 加入 IAM 权限组合）
	if err = addBSCPPermissions(ctx, params.Workspace, bscpApp); err != nil {
		metrics.BscpcfgStepFailed("refresh_permissions")
		return nil, errors.Wrap(err, "refresh bscpcfg permissions")
	}

	// 4. 绑定后置脚本
	if postHookID := cast.ToInt64(meta.PostHookID); meta.PostHookID != "" && postHookID != 0 {
		bscpAppID := cast.ToInt64(bscpApp.ID)
		if err = m.client.UpdateConfigHook(ctx, &bscpapi.UpdateConfigHookReq{
			BizID:      bizID,
			ProjectID:  projectID,
			EnvID:      bscpEnv.ID,
			AppID:      bscpAppID,
			PostHookID: postHookID,
		}); err != nil {
			metrics.BscpcfgStepFailed("bind_hook")
			return nil, errors.Wrap(err, "bind post hook to bscp app")
		}
		log.Infof(ctx, "bound post hook %d to bscp app %d (biz %s, project %d, env %s)",
			postHookID, bscpAppID, bizID, projectID, envID)
	}

	// 5. 刷新 Credential Scope
	credID := cast.ToInt64(meta.CredentialID)
	if err = m.RefreshCredentialScopes(ctx, bizID, projectID, credID, *bscpApp, *bscpEnv); err != nil {
		metrics.BscpcfgStepFailed("refresh_scopes")
		return nil, errors.Wrap(err, "refresh credential scopes")
	}

	// 6. 写入 EnvBinding
	binding := &model.EnvBinding{
		AppID:       params.AppID,
		EnvName:     params.EnvName,
		BscpEnvID:   envID,
		BscpEnvName: bscpEnv.Spec.Name,
		BscpAppID:   bscpApp.ID,
		Operator:    params.Operator,
	}

	if err = m.configStore.CreateEnvBinding(ctx, binding); err != nil {
		return nil, errors.Wrap(err, "create env binding in store")
	}

	return &model.Snapshot{
		Metadata:   meta,
		EnvBinding: binding,
	}, nil
}

// GetSnapshot 获取指定 app+env 的聚合快照。
func (m *Manager) GetSnapshot(
	ctx context.Context,
	appID, envName string,
) (*model.Snapshot, error) {
	meta, err := m.configStore.GetMetadata(ctx, appID)
	if err != nil {
		return nil, err
	}

	binding, err := m.configStore.GetEnvBinding(ctx, appID, envName)
	if err != nil {
		return nil, err
	}

	return &model.Snapshot{
		Metadata:   meta,
		EnvBinding: binding,
	}, nil
}

// ListSnapshots 获取应用下所有环境的聚合快照列表。
func (m *Manager) ListSnapshots(
	ctx context.Context,
	appID string,
) ([]*model.Snapshot, error) {
	meta, err := m.configStore.GetMetadata(ctx, appID)
	if err != nil {
		if errors.Is(err, model.ErrMetadataNotFound) {
			return nil, model.ErrEnvBindingNotFound
		}
		return nil, errors.Wrap(err, "get metadata")
	}

	bindings, err := m.configStore.ListEnvBindingsByApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list env bindings")
	}

	results := make([]*model.Snapshot, 0, len(bindings))
	for _, binding := range bindings {
		results = append(results, &model.Snapshot{
			Metadata:   meta,
			EnvBinding: binding,
		})
	}

	return results, nil
}

// DeleteEnvBinding 删除指定环境绑定。
func (m *Manager) DeleteEnvBinding(ctx context.Context, appID, envName string) error {
	return m.configStore.DeleteEnvBinding(ctx, appID, envName)
}

// DeleteByApp 删除应用下所有配置（级联删除）。
func (m *Manager) DeleteByApp(ctx context.Context, appID string) error {
	// 先删除所有 EnvBinding
	if err := m.configStore.DeleteEnvBindingsByApp(ctx, appID); err != nil {
		return errors.Wrap(err, "delete env bindings by app")
	}
	// 再删除 Metadata
	if err := m.configStore.DeleteMetadata(ctx, appID); err != nil &&
		!errors.Is(err, model.ErrMetadataNotFound) {
		return errors.Wrap(err, "delete metadata")
	}
	return nil
}

// === cred 管理 ===

// GetCredentialByName 通过名称查询指定业务下的 Credential
func (m *Manager) GetCredentialByName(
	ctx context.Context, bizID string, projectID int64, name string,
) (*bscpapi.Credential, error) {
	credentials, err := m.client.ListCredentials(ctx, bizID, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list credentials for biz %s, project %d", bizID, projectID)
	}

	cred, found := lo.Find(credentials, func(c bscpapi.Credential) bool {
		return c.Name == name
	})
	if !found {
		return nil, errors.Wrapf(
			ErrCredentialNotFound, "credential %q not found in biz %s, project %d", name, bizID, projectID,
		)
	}

	return &cred, nil
}

// GetOrCreateCredential 获取或创建 Credential（幂等）。
func (m *Manager) GetOrCreateCredential(
	ctx context.Context, bizID string, projectID int64,
) (*bscpapi.Credential, error) {
	cred, err := m.GetCredentialByName(ctx, bizID, projectID, credentialName)
	if err == nil {
		return cred, nil
	}

	if !errors.Is(err, ErrCredentialNotFound) {
		return nil, err
	}

	log.Infof(ctx, "credential %q not found in biz %s, project %d, creating...", credentialName, bizID, projectID)
	_, createErr := m.client.CreateCredential(ctx, &bscpapi.CreateCredentialReq{
		BizID:     bizID,
		ProjectID: projectID,
		Name:      credentialName,
		Memo:      "auto-created by bkms platform",
	})
	if createErr != nil {
		return nil, errors.Wrapf(
			createErr,
			"create credential %q in biz %s, project %d",
			credentialName,
			bizID,
			projectID,
		)
	}

	cred, err = m.GetCredentialByName(ctx, bizID, projectID, credentialName)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"get credential %q after creation in biz %s, project %d",
			credentialName,
			bizID,
			projectID,
		)
	}

	return cred, nil
}

// RefreshCredentialScopes 为指定 app+env 添加 credential scope（幂等）。
func (m *Manager) RefreshCredentialScopes(
	ctx context.Context,
	bizID string,
	projectID, credentialID int64,
	app bscpapi.App,
	env bscpapi.Environment,
) error {
	targetScope := bscpapi.CredentialScopeItem{
		App:     app.Name,
		Scope:   defaultScope,
		EnvID:   env.ID,
		EnvName: env.Spec.Name,
		EnvType: env.Spec.Type,
	}

	currentScopes, err := m.client.ListCredentialScopes(ctx, bizID, projectID, credentialID)
	if err != nil {
		return errors.Wrapf(
			err, "list credential scopes for biz %s, project %d, credential %d",
			bizID, projectID, credentialID,
		)
	}

	// 已存在则跳过（幂等）
	for _, s := range currentScopes {
		if s.App == app.Name && s.EnvID == env.ID {
			log.Infof(ctx, "credential scope already exists for app %s, env %d", app.Name, env.ID)
			return nil
		}
	}

	// 新增当前 app+env 的 scope
	if err = m.client.UpdateCredentialScope(ctx, &bscpapi.UpdateCredentialScopeReq{
		BizID:        bizID,
		ProjectID:    projectID,
		CredentialID: credentialID,
		AddScope:     []bscpapi.CredentialScopeItem{targetScope},
	}); err != nil {
		return errors.Wrapf(
			err, "update credential scope for biz %s, project %d, credential %d",
			bizID, projectID, credentialID,
		)
	}

	log.Infof(ctx, "credential scope added for app %s, env %d", app.Name, env.ID)
	return nil
}

// === Hook 管理 ===

// GetOrCreatePostHook 获取或创建后置脚本（幂等）。
func (m *Manager) GetOrCreatePostHook(
	ctx context.Context, bizID string, projectID int64, appID string,
) (int64, error) {
	hookName := fmt.Sprintf("bkms-post-hook-%s", appID)

	// 先尝试查询是否已存在
	hooks, err := m.client.ListHooks(ctx, &bscpapi.ListHooksReq{
		BizID:     bizID,
		ProjectID: projectID,
		Name:      hookName,
		All:       true,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "list hooks for biz %s, project %d, name %s", bizID, projectID, hookName)
	}

	// 精确匹配已有 hook
	for _, item := range hooks.Details {
		if item.Hook.Name == hookName {
			log.Infof(
				ctx,
				"post hook %q already exists (id: %d) in biz %s, project %d",
				hookName,
				item.Hook.ID,
				bizID,
				projectID,
			)
			return item.Hook.ID, nil
		}
	}

	// 创建新的后置脚本，采用完全同步/镜像同步(Mirroring Sync)策略：
	// rsync -artzc --delete 不仅会同步变化的文件，还会删除目标目录中源目录不存在的文件，确保目标目录与源目录保持完全一致。
	scriptContent := fmt.Sprintf(
		"#!/bin/bash\nrsync -artzc --delete ${bk_bscp_app_temp_dir}/files/ %s/",
		bscpworkload.BscpShareBasePath,
	)

	log.Infof(ctx, "post hook %q not found in biz %s, project %d, creating...", hookName, bizID, projectID)
	hookID, err := m.client.CreateHook(ctx, &bscpapi.CreateHookReq{
		BizID:        bizID,
		ProjectID:    projectID,
		Name:         hookName,
		Type:         "shell",
		Content:      scriptContent,
		RevisionName: "v1",
		Tags:         []string{"bkms", "post-hook"},
		Memo:         "auto-created by bkms platform, sync config files to shared volume",
	})
	if err != nil {
		return 0, errors.Wrapf(err, "create post hook %q in biz %s, project %d", hookName, bizID, projectID)
	}

	log.Infof(ctx, "post hook %q created (id: %d) in biz %s, project %d", hookName, hookID, bizID, projectID)
	return hookID, nil
}

// getOrCreateBscpEnv 按 envName 在 BSCP 项目下匹配或创建环境。
// envType 为 bkms 环境类型，内部会转换为 BSCP 环境类型（dev/test/staging/prod）。
func (m *Manager) getOrCreateBscpEnv(
	ctx context.Context,
	bizID string, projectID int64, envName, envType string,
) (*bscpapi.Environment, error) {
	bscpEnvType := ToBscpEnvType(envType)
	if bscpEnvType == "" {
		return nil, errors.Errorf("unsupported bkms env type %q", envType)
	}

	envResp, err := m.client.ListEnvironments(ctx, bizID, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list environments for biz %s, project %d", bizID, projectID)
	}

	// 按类型、名称匹配
	for _, env := range envResp.AllEnvironments() {
		if strings.EqualFold(env.Spec.Type, bscpEnvType) && strings.EqualFold(env.Spec.Name, envName) {
			return &env, nil
		}
	}

	// 未找到则创建
	log.Infof(
		ctx,
		"bscp environment %q (type %s) not found in project %d, creating...",
		envName,
		bscpEnvType,
		projectID,
	)
	env, err := m.client.CreateEnvironment(ctx, &bscpapi.CreateEnvironmentReq{
		BizID:     bizID,
		ProjectID: projectID,
		Name:      envName,
		Type:      bscpEnvType,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "create bscp environment %q in project %d", envName, projectID)
	}

	log.Infof(ctx, "bscp environment %q created (id: %d) in project %d", envName, env.ID, projectID)
	return env, nil
}
