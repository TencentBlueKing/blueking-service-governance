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

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	bscpworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// Manager 应用配置管理业务管理器。
type Manager struct {
	feedAddr string

	client bscpapi.Client

	configStore model.Store
}

// NewManager 创建 Manager
func NewManager(
	user auth.User,
	configStore model.Store,
) (*Manager, error) {
	client, err := bscpapi.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bscp client")
	}

	return &Manager{
		client:      client,
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
	cred, err := m.GetOrCreateCredential(ctx, params.BscpBizID)
	if err != nil {
		return nil, errors.Wrap(err, "get or create bscpcfg credential")
	}

	// 获取或创建后置脚本
	hookID, err := m.GetOrCreatePostHook(ctx, params.BscpBizID, params.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "get or create post hook")
	}

	meta := &model.Metadata{
		AppID:          params.AppID,
		BscpBizID:      params.BscpBizID,
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

// CreateEnvBinding 创建环境绑定（前置条件：已 InitMetadata）。
func (m *Manager) CreateEnvBinding(
	ctx context.Context,
	params *CreateEnvBindingParams,
) (*model.Snapshot, error) {
	// 1. 检查 Metadata 是否存在（要求先 InitMetadata）
	meta, err := m.configStore.GetMetadata(ctx, params.AppID)
	if err != nil {
		if errors.Is(err, model.ErrMetadataNotFound) {
			return nil, errors.New("bscpcfg metadata not initialized, please call InitMetadata first")
		}
		return nil, errors.Wrap(err, "get metadata")
	}

	// 2. 获取或创建 file 服务并刷新 IAM 权限
	fileSvc, err := m.getOrCreateFileService(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "get or create file service")
	}

	// 3. 绑定后置脚本到 file 服务
	if postHookID := cast.ToInt64(meta.PostHookID); meta.PostHookID != "" && postHookID != 0 {
		fileSvcID := cast.ToInt64(fileSvc.ID)
		if err = m.client.UpdateConfigHook(ctx, &bscpapi.UpdateConfigHookReq{
			BizID:      params.BscpBizID,
			AppID:      fileSvcID,
			PostHookID: postHookID,
		}); err != nil {
			return nil, errors.Wrap(err, "bind post hook to file service")
		}
		log.Infof(ctx, "bound post hook %d to file service %d (biz %s)", postHookID, fileSvcID, params.BscpBizID)
	}

	// 4. 刷新 Credential 权限
	credID := cast.ToInt64(meta.CredentialID)
	if err = m.RefreshCredentialScopes(ctx, params.BscpBizID, credID); err != nil {
		return nil, errors.Wrap(err, "refresh credential scopes")
	}

	// 5. 创建 EnvBinding
	binding := &model.EnvBinding{
		AppID:   params.AppID,
		EnvName: params.EnvName,
		Services: []model.ServiceRef{
			{ID: fileSvc.ID, Name: fileSvc.Name},
		},
		DefaultServiceID: fileSvc.ID,
		Operator:         params.Operator,
	}

	if err = m.configStore.CreateEnvBinding(ctx, binding); err != nil {
		return nil, errors.Wrap(err, "create env binding in store")
	}

	return &model.Snapshot{
		Metadata:   meta,
		EnvBinding: binding,
	}, nil
}

// BindServices 更新环境绑定的下发服务列表。
func (m *Manager) BindServices(
	ctx context.Context,
	appID, envName, bizID string,
	newServices []model.ServiceRef,
) error {
	// 1. 获取现有 EnvBinding
	existingBinding, err := m.configStore.GetEnvBinding(ctx, appID, envName)
	if err != nil {
		return errors.Wrap(err, "get existing env binding")
	}

	// 2. 校验 newServices 中是否包含 defaultServiceID
	if err = validateDefaultService(existingBinding, newServices); err != nil {
		return err
	}

	// 3. 刷新 Credential 权限
	cred, err := m.GetOrCreateCredential(ctx, bizID)
	if err != nil {
		return errors.Wrap(err, "get or create credential")
	}
	if err = m.RefreshCredentialScopes(ctx, bizID, cred.ID); err != nil {
		return errors.Wrap(err, "refresh credential scopes")
	}

	// 4. 写入 EnvBinding store
	updateData := &model.EnvBindingUpdate{Services: &newServices}
	if err = m.configStore.UpdateEnvBinding(ctx, appID, envName, updateData); err != nil {
		return errors.Wrap(err, "update env binding in store")
	}

	return nil
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
	ctx context.Context, bizID, name string,
) (*bscpapi.Credential, error) {
	credentials, err := m.client.ListCredentials(ctx, bizID)
	if err != nil {
		return nil, errors.Wrapf(err, "list credentials for biz %s", bizID)
	}

	cred, found := lo.Find(credentials, func(c bscpapi.Credential) bool {
		return c.Name == name
	})
	if !found {
		return nil, errors.Wrapf(ErrCredentialNotFound, "credential %q not found in biz %s", name, bizID)
	}

	return &cred, nil
}

// GetOrCreateCredential 获取或创建 Credential（幂等）。
func (m *Manager) GetOrCreateCredential(
	ctx context.Context, bizID string,
) (*bscpapi.Credential, error) {
	// 先尝试查询
	cred, err := m.GetCredentialByName(ctx, bizID, credentialName)
	if err == nil {
		return cred, nil
	}

	// 非"未找到"错误，直接返回
	if !errors.Is(err, ErrCredentialNotFound) {
		return nil, err
	}

	// 不存在，创建新的 Credential
	log.Infof(ctx, "credential %q not found in biz %s, creating...", credentialName, bizID)
	_, createErr := m.client.CreateCredential(ctx, &bscpapi.CreateCredentialReq{
		BizID: bizID,
		Name:  credentialName,
		Memo:  "auto-created by bkms platform",
	})
	if createErr != nil {
		return nil, errors.Wrapf(createErr, "create credential %q in biz %s", credentialName, bizID)
	}

	// 创建成功后重新查询获取完整信息（含 EncCredential/Token）
	cred, err = m.GetCredentialByName(ctx, bizID, credentialName)
	if err != nil {
		return nil, errors.Wrapf(err, "get credential %q after creation in biz %s", credentialName, bizID)
	}

	return cred, nil
}

// RefreshCredentialScopes 刷新 Credential 的关联服务权限（增量 diff）。
func (m *Manager) RefreshCredentialScopes(
	ctx context.Context, bizID string, credentialID int64,
) error {
	// 获取业务下所有 BSCP 服务
	services, err := m.client.ListBizServices(ctx, bizID)
	if err != nil {
		return errors.Wrapf(err, "list biz services for biz %s", bizID)
	}

	// 服务列表为空，不执行任何操作
	if len(services) == 0 {
		log.Infof(ctx, "no services found in biz %s, skip refresh credential scopes", bizID)
		return nil
	}

	// 构建目标 Scope 列表：每个 app name + "/**"
	targetScopes := lo.Map(services, func(app bscpapi.Service, _ int) bscpapi.CredentialScopeItem {
		return bscpapi.CredentialScopeItem{
			App:   app.Name,
			Scope: defaultScope,
		}
	})

	// 获取当前 Credential 已有的 Scope 列表
	credentialIDStr := cast.ToString(credentialID)
	currentScopes, err := m.client.ListCredentialScopes(ctx, bizID, credentialIDStr)
	if err != nil {
		return errors.Wrapf(err, "list credential scopes for biz %s, credential %d", bizID, credentialID)
	}

	// 计算增量 diff
	updateReq := DiffScopes(currentScopes, targetScopes)
	if updateReq == nil {
		log.Infof(ctx, "credential scopes already up-to-date for biz %s, credential %d", bizID, credentialID)
		return nil
	}

	// 填充必要字段并执行更新
	updateReq.BizID = bizID
	updateReq.CredentialID = credentialIDStr

	if err = m.client.UpdateCredentialScope(ctx, updateReq); err != nil {
		return errors.Wrapf(err, "update credential scope for biz %s, credential %d", bizID, credentialID)
	}

	log.Infof(
		ctx, "credential scopes refreshed for biz %s, credential %d (add: %d, alter: %d, del: %d)",
		bizID, credentialID, len(updateReq.AddScope), len(updateReq.AlterScope), len(updateReq.DelID),
	)

	return nil
}

// === Hook 管理 ===

// GetOrCreatePostHook 获取或创建后置脚本（幂等）。
func (m *Manager) GetOrCreatePostHook(
	ctx context.Context, bizID, appID string,
) (int64, error) {
	hookName := fmt.Sprintf("bkms-post-hook-%s", appID)

	// 先尝试查询是否已存在
	hooks, err := m.client.ListHooks(ctx, &bscpapi.ListHooksReq{
		BizID: bizID,
		Name:  hookName,
		All:   true,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "list hooks for biz %s, name %s", bizID, hookName)
	}

	// 精确匹配已有 hook
	for _, item := range hooks.Details {
		if item.Hook.Name == hookName {
			log.Infof(ctx, "post hook %q already exists (id: %d) in biz %s", hookName, item.Hook.ID, bizID)
			return item.Hook.ID, nil
		}
	}

	// 创建新的后置脚本，采用完全同步/镜像同步(Mirroring Sync)策略：
	// rsync -artzc --delete 不仅会同步变化的文件，还会删除目标目录中源目录不存在的文件，确保目标目录与源目录保持完全一致。
	scriptContent := fmt.Sprintf(
		"#!/bin/bash\nrsync -artzc --delete ${bk_bscp_app_temp_dir}/files/ %s/",
		bscpworkload.BscpShareBasePath,
	)

	log.Infof(ctx, "post hook %q not found in biz %s, creating...", hookName, bizID)
	hookID, err := m.client.CreateHook(ctx, &bscpapi.CreateHookReq{
		BizID:        bizID,
		Name:         hookName,
		Type:         "shell",
		Content:      scriptContent,
		RevisionName: "v1",
		Tags:         []string{"bkms", "post-hook"},
		Memo:         "auto-created by bkms platform, sync config files to shared volume",
	})
	if err != nil {
		return 0, errors.Wrapf(err, "create post hook %q in biz %s", hookName, bizID)
	}

	log.Infof(ctx, "post hook %q created (id: %d) in biz %s", hookName, hookID, bizID)
	return hookID, nil
}

// === 内部辅助 ===

// getOrCreateFileService 获取或创建 BSCP file 服务并刷新 IAM 权限。
func (m *Manager) getOrCreateFileService(
	ctx context.Context,
	params *CreateEnvBindingParams,
) (*bscpapi.Service, error) {
	// 获取或创建 file 类型服务
	fileServiceName := fmt.Sprintf("bkms-%s-%s", params.AppName, params.EnvName)
	fileSvc, err := m.client.GetOrCreateService(
		ctx,
		bscpapi.NewCreateServiceReq(
			params.Workspace.BkSystems.BkCCBizID,
			fileServiceName,
			fileServiceName,
			bscpapi.ConfigTypeFile,
			bscpapi.DataTypeAny,
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "get or create bscpcfg file service")
	}

	// 刷新 workspace 权限范围（仅 file 服务）
	if err = addBSCPPermissions(ctx, params.Workspace, fileSvc); err != nil {
		return nil, errors.Wrap(err, "refresh bscpcfg permissions")
	}

	return fileSvc, nil
}

// validateDefaultService 校验 newServices 中是否包含 defaultServiceID
func validateDefaultService(existingBinding *model.EnvBinding, newServices []model.ServiceRef) error {
	if existingBinding.DefaultServiceID == "" {
		return nil
	}

	for _, svc := range newServices {
		if svc.ID == existingBinding.DefaultServiceID {
			return nil
		}
	}

	return errors.Errorf("services must contain the default file service (id: %s)", existingBinding.DefaultServiceID)
}
