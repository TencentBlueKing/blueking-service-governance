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
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	bscpworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// Manager 应用配置管理业务管理器。
type Manager struct {
	client bscpapi.ConfigClient

	metadata    *model.MetadataStore
	envBindings *model.EnvBindingStore
}

// NewManager 创建 Manager
func NewManager(
	user auth.User,
	db *mongo.Database,
) (*Manager, error) {
	configClient, err := bscpapi.NewConfigClient(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bscp config client")
	}

	return &Manager{
		client:      configClient,
		metadata:    store.Of(db, model.MetadataMeta),
		envBindings: store.Of(db, model.EnvBindingMeta),
	}, nil
}

// === Metadata 管理 ===

// InitMetadata 初始化配置管理（幂等）。
func (m *Manager) InitMetadata(
	ctx context.Context,
	params *InitMetadataParams,
) (*model.Metadata, error) {
	// 先检查是否已存在
	existing, err := m.metadata.Get(ctx, model.ByApp(params.AppID))
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, model.ErrMetadataNotFound) {
		return nil, errors.Wrap(err, "get metadata")
	}

	// 检查 workspace 是否已绑定 BSCP 项目
	if params.BscpBizID == "" || params.BscpProjectID == "" {
		return nil, errors.New("workspace is not bound to a BSCP project")
	}

	// 获取或创建后置脚本
	hookID, err := m.GetOrCreatePostHook(ctx, params.BscpBizID, cast.ToInt64(params.BscpProjectID), params.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "get or create post hook")
	}

	meta := &model.Metadata{
		AppID:        params.AppID,
		Enable:       true,
		WorkloadName: params.WorkloadName,
		WorkloadKind: params.WorkloadKind,
		PostHookID:   fmt.Sprintf("%d", hookID),
		Operator:     params.Operator,
	}

	if err = m.metadata.Create(ctx, meta); err != nil {
		return nil, errors.Wrap(err, "create metadata")
	}

	return meta, nil
}

// PatchMetadata 局部更新 Metadata，fields 的 key 为待更新的字段名，未出现的字段保持不变。
// fields 为空时返回 store.ErrNoFieldsToUpdate，Metadata 不存在时返回 ErrMetadataNotFound
func (m *Manager) PatchMetadata(ctx context.Context, appID string, fields map[string]any) error {
	if err := m.metadata.Update(ctx, fields, model.ByApp(appID)); err != nil {
		return errors.Wrapf(err, "update metadata of app %s", appID)
	}
	return nil
}

// === EnvBinding 管理 ===

// CreateEnvBinding 创建环境绑定。
func (m *Manager) CreateEnvBinding(
	ctx context.Context,
	params *CreateEnvBindingParams,
) (*model.Snapshot, error) {
	// 1. 检查 Metadata 是否存在且启用
	meta, err := m.metadata.Get(ctx, model.ByApp(params.AppID))
	if err != nil {
		if errors.Is(err, model.ErrMetadataNotFound) {
			return nil, errors.New("bscpcfg metadata is not initialized")
		}
		return nil, errors.Wrap(err, "get metadata")
	}
	if !meta.Enable {
		return nil, errors.New("bscpcfg is disabled")
	}

	bizID, projectID, err := bscpProject(params.Workspace)
	if err != nil {
		return nil, err
	}

	// 2. 按 envName 匹配或创建 BSCP 环境（类型由 bkms 环境类型映射而来）
	bscpEnv, err := m.getOrCreateBscpEnv(ctx, bizID, projectID, params.EnvName, params.EnvType)
	if err != nil {
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
		metrics.BscpcfgStepFailed("get_or_create_app")
		return nil, errors.Wrap(err, "get or create bscp app")
	}

	// 3.1 刷新 workspace 权限范围（将 BSCP App 加入 IAM 权限组合）
	if err = addBSCPPermissions(ctx, params.Workspace, bscpApp); err != nil {
		return nil, errors.Wrap(err, "refresh bscpcfg permissions")
	}

	// 4. 绑定后置脚本
	if err = m.bindPostHook(ctx, meta, bizID, projectID, bscpEnv.ID, bscpApp.ID, envID); err != nil {
		return nil, err
	}

	// 5. 刷新 Credential Scope
	credID := cast.ToInt64(params.Workspace.BkSystems.BscpCredentialID)
	if err = m.RefreshCredentialScopes(ctx, bizID, projectID, credID, *bscpApp, *bscpEnv); err != nil {
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

	if err = m.envBindings.Create(ctx, binding); err != nil {
		return nil, errors.Wrap(err, "create env binding in store")
	}

	return &model.Snapshot{
		Metadata:   meta,
		EnvBinding: binding,
	}, nil
}

func bscpProject(ws *workspace.Workspace) (string, int64, error) {
	bizID := ws.BkSystems.BkCCBizID
	projectID := ws.BkSystems.BkBSCPProjectID
	if projectID == "" {
		return "", 0, errors.New("workspace is not bound to a BSCP project")
	}
	if ws.BkSystems.BscpCredentialID == "" {
		return "", 0, errors.New("workspace bscp credential is not ready")
	}
	return bizID, cast.ToInt64(projectID), nil
}

func (m *Manager) bindPostHook(
	ctx context.Context,
	meta *model.Metadata,
	bizID string,
	projectID, bscpEnvID int64,
	bscpAppID, envID string,
) error {
	postHookID := cast.ToInt64(meta.PostHookID)
	if meta.PostHookID == "" || postHookID == 0 {
		return nil
	}
	appID := cast.ToInt64(bscpAppID)
	if err := m.client.UpdateConfigHook(ctx, &bscpapi.UpdateConfigHookReq{
		BizID:      bizID,
		ProjectID:  projectID,
		EnvID:      bscpEnvID,
		AppID:      appID,
		PostHookID: postHookID,
	}); err != nil {
		metrics.BscpcfgStepFailed("bind_post_hook")
		return errors.Wrap(err, "bind post hook to bscp app")
	}
	log.Infof(ctx, "bound post hook %d to bscp app %d (biz %s, project %d, env %s)",
		postHookID, appID, bizID, projectID, envID)
	return nil
}

// GetSnapshot 获取指定 app+env 的聚合快照。
func (m *Manager) GetSnapshot(
	ctx context.Context,
	appID, envName string,
) (*model.Snapshot, error) {
	meta, err := m.metadata.Get(ctx, model.ByApp(appID))
	if err != nil {
		return nil, errors.Wrap(err, "get metadata")
	}

	binding, err := m.envBindings.Get(ctx, model.ByAppAndEnv(appID, envName))
	if err != nil {
		return nil, errors.Wrap(err, "get env binding")
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
	meta, err := m.metadata.Get(ctx, model.ByApp(appID))
	if err != nil {
		if errors.Is(err, model.ErrMetadataNotFound) {
			return nil, model.ErrEnvBindingNotFound
		}
		return nil, errors.Wrap(err, "get metadata")
	}

	bindings, err := m.envBindings.List(ctx, model.ByApp(appID))
	if err != nil {
		return nil, errors.Wrap(err, "list env bindings")
	}

	results := make([]*model.Snapshot, 0, len(bindings))
	for i := range bindings {
		results = append(results, &model.Snapshot{
			Metadata:   meta,
			EnvBinding: &bindings[i],
		})
	}

	return results, nil
}

// DeleteEnvBinding 删除指定环境绑定。
func (m *Manager) DeleteEnvBinding(ctx context.Context, appID, envName string) error {
	if err := m.envBindings.Delete(ctx, model.ByAppAndEnv(appID, envName)); err != nil {
		return errors.Wrapf(err, "delete env binding of app %s env %s", appID, envName)
	}
	return nil
}

// DeleteByApp 删除应用下所有配置（级联删除）。
func (m *Manager) DeleteByApp(ctx context.Context, appID string) error {
	// 先删除所有 EnvBinding
	if _, err := m.envBindings.DeleteMany(ctx, model.ByApp(appID)); err != nil {
		return errors.Wrap(err, "delete env bindings by app")
	}
	// 再删除 Metadata
	if err := m.metadata.DeleteIfExists(ctx, model.ByApp(appID)); err != nil {
		return errors.Wrap(err, "delete metadata")
	}
	return nil
}

// RefreshCredentialScopes 为指定 app+env 添加 credential scope（幂等）。
func (m *Manager) RefreshCredentialScopes(
	ctx context.Context,
	bizID string,
	projectID, credentialID int64,
	app bscpapi.App,
	env bscpapi.Environment,
) (err error) {
	defer func() {
		if err != nil {
			metrics.BscpcfgStepFailed("refresh_credential_scopes")
		}
	}()

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
) (hookID int64, err error) {
	defer func() {
		if err != nil {
			metrics.BscpcfgStepFailed("get_or_create_post_hook")
		}
	}()

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
	hookID, err = m.client.CreateHook(ctx, &bscpapi.CreateHookReq{
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
) (env *bscpapi.Environment, err error) {
	defer func() {
		if err != nil {
			metrics.BscpcfgStepFailed("get_or_create_env")
		}
	}()

	bscpEnvType := ToBscpEnvType(envType)
	if bscpEnvType == "" {
		return nil, errors.Errorf("unsupported bkms env type %q", envType)
	}

	envResp, err := m.client.ListEnvironments(ctx, bizID, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list environments for biz %s, project %d", bizID, projectID)
	}

	// 按类型、名称匹配
	for _, e := range envResp.AllEnvironments() {
		if strings.EqualFold(e.Spec.Type, bscpEnvType) && strings.EqualFold(e.Spec.Name, envName) {
			return &e, nil
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
	env, err = m.client.CreateEnvironment(ctx, &bscpapi.CreateEnvironmentReq{
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
