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

package appcfg

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// AppCfgFileDefService 场景层服务，感知 ConfigKind。
// 内嵌 BaseAppCfgFileService 以直接暴露底层 CRUD 方法；
// 仅在需要 policy 校验/过滤时覆盖或新增方法。
type AppCfgFileDefService struct {
	*BaseAppCfgFileService
	policies map[ConfigKind]ConfigKindPolicy
}

// NewAppCfgFileDefService 创建场景层服务。
func NewAppCfgFileDefService(
	base *BaseAppCfgFileService,
	policies map[ConfigKind]ConfigKindPolicy,
) *AppCfgFileDefService {
	if policies == nil {
		policies = DefaultPolicies
	}
	return &AppCfgFileDefService{BaseAppCfgFileService: base, policies: policies}
}

func (s *AppCfgFileDefService) policyFor(kind ConfigKind) (ConfigKindPolicy, error) {
	p, ok := s.policies[kind]
	if !ok {
		return nil, errors.Errorf("unsupported config kind: %s", kind)
	}
	return p, nil
}

// --- Kind 感知的方法（覆盖或新增） ---

// Create 创建配置文件（含 def + 默认文件记录 + 初始版本），执行 kind 级别的校验。
func (s *AppCfgFileDefService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFileWithDef, error) {
	kind := params.ConfigKind
	if kind == "" {
		kind = ConfigKindFramework
	}

	policy, err := s.policyFor(kind)
	if err != nil {
		return nil, err
	}

	var contentToValidate string
	if params.Content != nil {
		contentToValidate = *params.Content
	}
	if err = policy.ValidateContent(contentToValidate, params.Format); err != nil {
		return nil, errors.Wrap(err, "kind-specific content validation")
	}

	def, err := s.createDef(ctx, params, kind)
	if err != nil {
		return nil, err
	}

	acf, err := s.createFileAndVersion(ctx, params, def)
	if err != nil {
		_, _ = s.DefStore.DeleteByID(ctx, def.ID)
		return nil, err
	}
	return &AppConfigFileWithDef{AppConfigFile: *acf, Def: def}, nil
}

func (s *AppCfgFileDefService) createDef(
	ctx context.Context, params CreateCfgFileParams, kind ConfigKind,
) (*AppConfigFileDef, error) {
	def := AppConfigFileDef{
		AppID:      params.AppID,
		Name:       params.Name,
		ConfigKind: kind,
		MountDir:   params.MountDir,
		EnvConfigMode: EnvConfigMode{
			IsUnifiedConfig: true,
		},
		Creator: params.Creator,
	}
	defID, err := s.DefStore.Add(ctx, def)
	if err != nil {
		return nil, errors.Wrap(err, "creating def record")
	}
	def.ID = defID
	return &def, nil
}

func (s *AppCfgFileDefService) createFileAndVersion(
	ctx context.Context, params CreateCfgFileParams, def *AppConfigFileDef,
) (*AppConfigFile, error) {
	acf := AppConfigFile{
		DefID:   def.ID,
		AppID:   params.AppID,
		EnvName: params.EnvName,
		Type:    params.Type,
		VersionedContent: VersionedContent{
			ContentSourceType:   params.ContentSourceType,
			Format:              params.Format,
			BSCPConfig:          params.BSCPConfig,
			Content:             params.Content,
			OverlayContent:      params.OverlayContent,
			BaseAppConfigFileID: params.BaseAppConfigFileID,
		},
		Creator:        params.Creator,
		Updater:        params.Creator,
		CurrentVersion: 1,
	}
	if acf.Format == "" {
		acf.Format = FileFormatYAML
	}
	acf.initializeContentFields(params.Type, params.ContentSourceType)
	if params.Content != nil {
		acf.Content = params.Content
	}
	if params.OverlayContent != nil {
		acf.OverlayContent = params.OverlayContent
	}

	return s.CreateFileWithVersion(ctx, acf, params.Name, params.Description, params.Creator)
}

// UpdateFileDef 更新逻辑文件的 def 信息（name、isUnifiedConfig 等），不产生版本记录。
// 切换环境配置模式时会执行额外操作（如切回统一配置需清理环境实例）。
func (s *AppCfgFileDefService) UpdateFileDef(
	ctx context.Context,
	def *AppConfigFileDef,
	update FileDefUpdate,
) error {
	if def == nil {
		return errors.New("def is required")
	}

	if update.MountDir != nil {
		if policy, err := s.policyFor(def.ConfigKind); err == nil && !policy.AllowMountDirUpdate() {
			return errors.New("this config kind does not support modifying mountDir via def")
		}
	}

	applyStaticDefFields(def, update)

	if update.HasEnvConfigChanges() && update.IsUnifiedConfig != nil {
		if err := s.applyEnvConfigChange(ctx, def, *update.IsUnifiedConfig); err != nil {
			return err
		}
	}

	if _, err := s.DefStore.Update(ctx, *def); err != nil {
		return errors.Wrap(err, "updating def record")
	}
	return nil
}

// applyEnvConfigChange 处理环境配置模式切换的副作用。
func (s *AppCfgFileDefService) applyEnvConfigChange(
	ctx context.Context,
	def *AppConfigFileDef,
	isUnifiedConfig bool,
) error {
	if def.EnvConfigMode.IsUnifiedConfig == isUnifiedConfig {
		return nil
	}
	def.EnvConfigMode.IsUnifiedConfig = isUnifiedConfig

	// 从独立配置切回统一配置：删除所有环境实例及其版本记录
	if isUnifiedConfig {
		if err := s.deleteEnvInstances(ctx, def.ID); err != nil {
			return err
		}
	}
	return nil
}

// deleteEnvInstances 删除指定 def 下所有非默认的环境实例及其版本记录。
func (s *AppCfgFileDefService) deleteEnvInstances(ctx context.Context, defID bson.ObjectID) error {
	files, err := s.FileStore.ListByDefID(ctx, defID)
	if err != nil {
		return errors.Wrap(err, "listing env instances for cleanup")
	}
	for _, f := range files {
		if f.EnvName == EnvNameDefault {
			continue
		}
		if _, err = s.FileStore.DeleteByID(ctx, f.AppID, f.ID); err != nil {
			return errors.Wrapf(err, "delete env instance %s", f.ID.Hex())
		}
		if _, err = s.VersionStore.DeleteByFileID(ctx, f.ID); err != nil {
			return errors.Wrapf(err, "delete versions for env instance %s", f.ID.Hex())
		}
	}
	return nil
}
