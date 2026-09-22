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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// AppCfgFileDefService 场景层服务，感知 ConfigKind, 内嵌 BaseAppCfgFileService 以直接暴露底层 CRUD 方法；
// 仅在需要 policy 校验/过滤时覆盖或新增方法。
type AppCfgFileDefService struct {
	*BaseAppCfgFileService
	policies map[ConfigKind]ConfigKindPolicy
}

// NewAppCfgFileDefService 创建场景层服务。
func NewAppCfgFileDefService(base *BaseAppCfgFileService) *AppCfgFileDefService {
	return &AppCfgFileDefService{BaseAppCfgFileService: base, policies: DefaultPolicies}
}

func (s *AppCfgFileDefService) policyFor(kind ConfigKind) (ConfigKindPolicy, error) {
	p, ok := s.policies[kind]
	if !ok {
		return nil, errors.Wrapf(ErrInvalidConfigSpec, "unsupported config kind: %s", kind)
	}
	return p, nil
}

// --- Create / Update ---

// Create 创建配置文件（含 def + 默认文件记录 + 初始版本），执行 kind 级别的校验。
func (s *AppCfgFileDefService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFileWithDef, error) {
	kind := params.ConfigKind
	if kind == "" {
		return nil, errors.Wrap(ErrInvalidConfigSpec, "config kind is required")
	}

	policy, err := s.policyFor(kind)
	if err != nil {
		return nil, err
	}

	// kind 级别的创建参数校验
	if err = policy.ValidateCreateParams(params); err != nil {
		return nil, err
	}

	contentToValidate := lo.FromPtr(params.Content)
	if contentToValidate == "" {
		contentToValidate = lo.FromPtr(params.OverlayContent)
	}
	if err = policy.ValidateContent(contentToValidate, params.Format); err != nil {
		return nil, errors.Wrap(err, "kind-specific content validation")
	}

	// 框架页按环境保存仍走旧 Create 接口。tRPC/TAF 的环境 overlay 必须挂在
	// 已有 framework def 下，否则部署按 def 配对会丢掉环境内容。
	if isFrameworkEnvOverlayCreate(params) {
		return s.attachFrameworkEnvOverlay(ctx, params)
	}

	if err = s.ensureSingleFrameworkDef(ctx, params.AppID, params.AppType, kind); err != nil {
		return nil, err
	}

	def, err := s.createDef(ctx, params, kind, policy)
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
	ctx context.Context, params CreateCfgFileParams, kind ConfigKind, policy ConfigKindPolicy,
) (*AppConfigFileDef, error) {
	def := AppConfigFileDef{
		AppID:      params.AppID,
		Name:       params.Name,
		ConfigKind: kind,
		MountDir:   params.MountDir,
		EnvConfigMode: EnvConfigMode{
			// 初始创建默认为统一配置
			IsUnifiedConfig: true,
		},
		EnableEnvVarRender: policy.DefaultEnableEnvVarRender(),
		Creator:            params.Creator,
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

// UpdateAppCfgFileDef 更新逻辑文件的 def 信息（name、isUnifiedConfig 等），不产生版本记录。
// 切换环境配置模式，挂载环境时会执行额外操作（如切回统一配置需清理环境实例）。
func (s *AppCfgFileDefService) UpdateAppCfgFileDef(
	ctx context.Context,
	def *AppConfigFileDef,
	update FileDefUpdate,
) error {
	if def == nil {
		return errors.New("def is required")
	}

	// 按策略区分
	if update.MountDir != nil || update.EnableEnvVarRender != nil {
		policy, err := s.policyFor(def.ConfigKind)
		if err != nil {
			return errors.Wrap(err, "loading config kind policy for def update")
		}
		if update.MountDir != nil && !policy.AllowMountDirUpdate() {
			return errors.Wrap(ErrInvalidConfigSpec, "this config kind does not support modifying mountDir via def")
		}
		if update.EnableEnvVarRender != nil && !policy.AllowEnableEnvVarRenderUpdate() {
			return errors.Wrap(ErrInvalidConfigSpec,
				"this config kind does not allow modifying enableEnvVarRender")
		}
	}

	applyStaticDefFields(def, update)

	if update.IsUnifiedConfig != nil && def.EnvConfigMode.IsUnifiedConfig != *update.IsUnifiedConfig {
		def.EnvConfigMode.IsUnifiedConfig = *update.IsUnifiedConfig
		// 从独立配置切回统一配置：删除所有环境实例及其版本记录
		if *update.IsUnifiedConfig {
			if err := s.deleteEnvInstances(ctx, def.ID); err != nil {
				return err
			}
		}
	}
	if update.MountedEnvNames != nil {
		if err := s.applyMountedEnvNamesChange(ctx, def, *update.MountedEnvNames); err != nil {
			return err
		}
	}
	if _, err := s.DefStore.Update(ctx, *def); err != nil {
		return errors.Wrap(err, "updating def record")
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

func (s *AppCfgFileDefService) ensureSingleFrameworkDef(
	ctx context.Context,
	appID, appType string,
	kind ConfigKind,
) error {
	if kind != ConfigKindFramework {
		return nil
	}
	// 与 app.AppTypeTRPC / AppTypeTAF 对齐，避免 appcfg 反向依赖 app 包。
	if appType != "trpc" && appType != "taf" {
		return nil
	}
	defs, err := s.DefStore.ListByApp(ctx, appID, DefFilterConfigKind(ConfigKindFramework))
	if err != nil {
		return errors.Wrap(err, "listing framework defs")
	}
	if len(defs) > 0 {
		return errors.Wrap(ErrInvalidConfigSpec, "trpc/taf application allows only one framework config file")
	}
	return nil
}

func isFrameworkEnvOverlayCreate(params CreateCfgFileParams) bool {
	return params.ConfigKind == ConfigKindFramework &&
		params.EnvName != EnvNameDefault &&
		params.Type == AppConfigFileTypeOverlay &&
		params.BaseAppConfigFileID != nil
}

// attachFrameworkEnvOverlay 把环境 overlay 挂到 base 文件所属的 framework def 上，
// 而不是再新建一条 def。旧 POST /app-config-files 按环境保存时走这条路径。
func (s *AppCfgFileDefService) attachFrameworkEnvOverlay(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFileWithDef, error) {
	baseFile, err := s.FileStore.GetByID(ctx, *params.BaseAppConfigFileID)
	if err != nil {
		return nil, errors.Wrap(err, "loading base app config file")
	}
	if baseFile.AppID != params.AppID {
		return nil, errors.Wrap(ErrInvalidConfigSpec, "base app config file does not belong to the app")
	}

	defaultFile, err := s.GetDefaultFileWithDef(ctx, baseFile.DefID)
	if err != nil {
		return nil, errors.Wrap(err, "loading default file for framework overlay")
	}
	def := defaultFile.Def

	// 若是第一次开启按环境配置，需要修改配置项
	if !def.EnvConfigMode.IsIndependent() {
		if err = s.UpdateAppCfgFileDef(ctx, def, FileDefUpdate{
			IsUnifiedConfig: lo.ToPtr(false),
			Operator:        params.Creator,
		}); err != nil {
			return nil, errors.Wrap(err, "switching framework def to independent config")
		}
	}

	existing, err := s.FindEnvInstance(ctx, def.ID, params.EnvName)
	if err != nil {
		return nil, errors.Wrap(err, "finding existing framework env overlay")
	}
	if existing != nil {
		return &AppConfigFileWithDef{AppConfigFile: *existing, Def: def}, nil
	}

	overlay := lo.FromPtr(params.OverlayContent)
	created, err := s.CreateEnvInstance(ctx, *defaultFile, CreateEnvInstanceParams{
		EnvName:        params.EnvName,
		OverlayContent: &overlay,
		Operator:       params.Creator,
		Description:    params.Description,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating framework env overlay on existing def")
	}
	return &AppConfigFileWithDef{AppConfigFile: *created, Def: def}, nil
}
