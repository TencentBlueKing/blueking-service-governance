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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
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
	params.MountDir = NormalizeMountDir(params.MountDir)

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

	if err = s.ensureSingleFrameworkDef(ctx, params.AppID, params.AppType, kind); err != nil {
		return nil, err
	}

	def, err := s.createDef(ctx, params, kind, policy)
	if err != nil {
		return nil, err
	}

	acf, err := s.createFileAndVersion(ctx, params, def)
	if err != nil {
		_ = s.DefStore.DeleteIfExists(ctx, store.ByID(def.ID))
		return nil, err
	}
	return &AppConfigFileWithDef{AppConfigFile: *acf, Def: def}, nil
}

func (s *AppCfgFileDefService) createDef(
	ctx context.Context, params CreateCfgFileParams, kind ConfigKind, policy ConfigKindPolicy,
) (*AppConfigFileDef, error) {
	enableEnvVarRender := policy.DefaultEnableEnvVarRender()
	if params.EnableEnvVarRender != nil {
		enableEnvVarRender = *params.EnableEnvVarRender
	}

	envConfigMode := EnvConfigMode{
		// 初始创建默认为统一配置
		IsUnifiedConfig: true,
	}
	if params.MountedEnvNames != nil {
		envConfigMode.MountedEnvNames = *params.MountedEnvNames
	}
	policy.InitializeEnvConfigMode(&envConfigMode, params)

	def := AppConfigFileDef{
		AppID:              params.AppID,
		Name:               params.Name,
		ConfigKind:         kind,
		MountDir:           params.MountDir,
		EnvConfigMode:      envConfigMode,
		EnableEnvVarRender: enableEnvVarRender,
		Creator:            params.Creator,
	}
	if err := s.DefStore.Create(ctx, &def); err != nil {
		return nil, errors.Wrap(err, "creating def record")
	}
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
	if update.MountDir != nil {
		update.MountDir = lo.ToPtr(NormalizeMountDir(*update.MountDir))
	}

	if err := s.validateDefUpdateByPolicy(def, update); err != nil {
		return err
	}

	name := lo.FromPtrOr(update.Name, def.Name)
	mountDir := lo.FromPtrOr(update.MountDir, def.MountDir)
	if name != def.Name || mountDir != def.MountDir {
		if err := s.ensureMountIdentityAvailable(ctx, def, name, mountDir); err != nil {
			return err
		}
	}

	applyStaticDefFields(def, update)

	if update.EnvContentStrategy != nil {
		if err := s.applyFrameworkEnvContentStrategyChange(
			ctx, def, *update.EnvContentStrategy, update.Operator,
		); err != nil {
			return err
		}
	}
	if err := s.applyUnifiedConfigChange(ctx, def, update.IsUnifiedConfig); err != nil {
		return err
	}
	if update.MountedEnvNames != nil {
		if err := s.applyMountedEnvNamesChange(ctx, def, *update.MountedEnvNames); err != nil {
			return err
		}
	}
	if err := UpdateDef(ctx, s.DefStore, def); err != nil {
		return errors.Wrap(err, "updating def record")
	}
	return nil
}

// validateDefUpdateByPolicy 校验本次更新是否触碰了当前 kind 不允许修改的 def 字段。
// 未触碰任何受策略约束的字段时直接放行，不必按 kind 查表。
func (s *AppCfgFileDefService) validateDefUpdateByPolicy(def *AppConfigFileDef, update FileDefUpdate) error {
	if update.Name == nil && update.MountDir == nil && update.EnableEnvVarRender == nil &&
		update.MountedEnvNames == nil && update.EnvContentStrategy == nil {
		return nil
	}
	policy, err := s.policyFor(def.ConfigKind)
	if err != nil {
		return errors.Wrap(err, "loading config kind policy for def update")
	}
	for _, c := range []struct {
		changed bool
		allowed bool
		reason  string
	}{
		{
			update.Name != nil && *update.Name != def.Name, policy.AllowNameUpdate(update.AppType),
			"this config kind does not allow modifying name",
		},
		{
			update.MountDir != nil, policy.AllowMountDirUpdate(update.AppType),
			"this config kind does not support modifying mountDir via def",
		},
		{
			update.MountedEnvNames != nil, policy.AllowMountedEnvNamesUpdate(),
			"this config kind does not support mountedEnvNames",
		},
		{
			update.EnableEnvVarRender != nil, policy.AllowEnableEnvVarRenderUpdate(),
			"this config kind does not allow modifying enableEnvVarRender",
		},
		{
			update.EnvContentStrategy != nil, policy.AllowEnvContentStrategyUpdate(),
			"this config kind does not support modifying envContentStrategy",
		},
	} {
		if c.changed && !c.allowed {
			return errors.Wrap(ErrInvalidConfigSpec, c.reason)
		}
	}
	return nil
}

// applyUnifiedConfigChange 切换统一配置开关。
// 从独立配置切回统一配置时，顺带删除所有环境实例及其版本记录。
func (s *AppCfgFileDefService) applyUnifiedConfigChange(
	ctx context.Context,
	def *AppConfigFileDef,
	isUnifiedConfig *bool,
) error {
	if isUnifiedConfig == nil || def.EnvConfigMode.IsUnifiedConfig == *isUnifiedConfig {
		return nil
	}
	def.EnvConfigMode.IsUnifiedConfig = *isUnifiedConfig
	if !*isUnifiedConfig {
		return nil
	}
	return s.deleteEnvInstances(ctx, def.ID)
}

// applyFrameworkEnvContentStrategyChange 更新框架定义的环境内容策略。
// 没有环境实例时只改字段。已有环境实例时，把每条实例改写成目标策略对应的文件形态，
// 但不新增版本记录，currentVersion 保持不变。
//
// 调用方已通过 policy 保证当前 kind 支持修改 envContentStrategy。
// TODO(non-atomic): 先改写环境实例，再由上层 UpdateAppCfgFileDef 持久化 def。
func (s *AppCfgFileDefService) applyFrameworkEnvContentStrategyChange(
	ctx context.Context,
	def *AppConfigFileDef,
	strategy EnvContentStrategy,
	operator string,
) error {
	if err := validateFrameworkEnvContentStrategy(strategy); err != nil {
		return err
	}
	if def.EnvConfigMode.EnvContentStrategy == strategy {
		return nil
	}
	if def.EnvConfigMode.IsIndependent() {
		if err := s.convertEnvInstancesForStrategy(ctx, def.ID, strategy, operator); err != nil {
			return err
		}
	}
	def.EnvConfigMode.EnvContentStrategy = strategy
	return nil
}

// convertEnvInstancesForStrategy 把已有环境实例转换成目标策略的存储形态。
func (s *AppCfgFileDefService) convertEnvInstancesForStrategy(
	ctx context.Context,
	defID bson.ObjectID,
	strategy EnvContentStrategy,
	operator string,
) error {
	defaultFile, err := s.FileStore.Get(ctx, FileByDefEnv(defID, EnvNameDefault))
	if err != nil {
		return errors.Wrap(err, "loading default file for strategy switch")
	}
	files, err := s.FileStore.List(ctx, FilesByDef(defID))
	if err != nil {
		return errors.Wrap(err, "listing env instances for strategy switch")
	}
	for i := range files {
		file := &files[i]
		if file.EnvName == EnvNameDefault {
			continue
		}
		rawContent := file.Content
		if file.Type == AppConfigFileTypeOverlay {
			rawContent = file.OverlayContent
		}

		var (
			targetType           AppConfigFileType
			targetContent        *string
			targetOverlayContent *string
		)
		targetBaseID := file.BaseAppConfigFileID

		switch strategy {
		case EnvContentStrategyOverwrite:
			targetType = AppConfigFileTypeNormal
			targetContent = rawContent
			targetOverlayContent = nil
		case EnvContentStrategyOverlay:
			targetType = AppConfigFileTypeOverlay
			targetContent = nil
			targetOverlayContent = rawContent
			if targetBaseID == nil {
				baseID := defaultFile.ID
				targetBaseID = &baseID
			}
		default:
			return errors.Wrapf(ErrInvalidConfigSpec, "unsupported envContentStrategy %q", strategy)
		}

		if err := UpdateEnvInstanceForStrategySwitch(
			ctx,
			s.FileStore,
			file.ID,
			targetType,
			targetContent,
			targetOverlayContent,
			targetBaseID,
			operator,
		); err != nil {
			return errors.Wrapf(err, "converting env instance %s", file.ID.Hex())
		}
	}
	return nil
}

// SyncFrameworkFileLocation 把 spec 更新里的文件名和挂载目录写到该应用唯一的 framework def。
// 空的 fileName / mountDir 表示保持 def 上的原值。trpc 改名由策略拒绝。
//
// todo 兼容 PUT /trpc-spec、PUT /taf-spec 仍携带 fileName、filePath。
// 创建时这两个值已经写在 def 上。后续 spec 只更新语言、启动命令和参数，名字和目录只通过 def 更新接口修改，查询继续从 def 填回。届时删除本函数。
func (s *AppCfgFileDefService) SyncFrameworkFileLocation(
	ctx context.Context,
	appID, appType, fileName, mountDir string,
) error {
	defs, err := ListDefsByApp(ctx, s.DefStore, appID, DefFilterConfigKind(ConfigKindFramework))
	if err != nil {
		return errors.Wrapf(err, "listing framework defs for app %s", appID)
	}
	if len(defs) != 1 {
		return errors.Errorf("app %s has %d framework config files", appID, len(defs))
	}
	def := &defs[0]

	update := FileDefUpdate{AppType: appType}
	if fileName != "" && fileName != def.Name {
		update.Name = &fileName
	}
	if cleaned := NormalizeMountDir(mountDir); cleaned != "" && cleaned != def.MountDir {
		update.MountDir = &cleaned
	}
	if update.Name == nil && update.MountDir == nil {
		return nil
	}
	if err = s.UpdateAppCfgFileDef(ctx, def, update); err != nil {
		return errors.Wrapf(err, "sync framework file location for app %s", appID)
	}
	return nil
}

// ensureMountIdentityAvailable 确认同应用下没有其他 def 占用 appID + mountDir + name。
func (s *AppCfgFileDefService) ensureMountIdentityAvailable(
	ctx context.Context,
	def *AppConfigFileDef,
	name, mountDir string,
) error {
	taken, err := s.DefStore.Exists(ctx, DefByMountIdentity(def.AppID, name, mountDir, def.ID))
	if err != nil {
		return errors.Wrapf(err, "checking config file identity for app %s", def.AppID)
	}
	if taken {
		return errors.Wrapf(
			ErrAppConfigFileDefAlreadyExists,
			"app %s already has config file %s in %q",
			def.AppID, name, mountDir,
		)
	}
	return nil
}

// deleteEnvInstances 删除指定 def 下所有非默认的环境实例及其版本记录。
func (s *AppCfgFileDefService) deleteEnvInstances(ctx context.Context, defID bson.ObjectID) error {
	files, err := s.FileStore.List(ctx, FilesByDef(defID))
	if err != nil {
		return errors.Wrap(err, "listing env instances for cleanup")
	}
	for _, f := range files {
		if f.EnvName == EnvNameDefault {
			continue
		}
		if err = DeleteFile(ctx, s.FileStore, f.AppID, f.ID); err != nil {
			return errors.Wrapf(err, "delete env instance %s", f.ID.Hex())
		}
		if _, err = s.VersionStore.DeleteMany(ctx, VersionsByFile(f.ID)); err != nil {
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
	if !appruntime.IsAppModelType(appType) {
		return nil
	}
	defs, err := ListDefsByApp(ctx, s.DefStore, appID, DefFilterConfigKind(ConfigKindFramework))
	if err != nil {
		return errors.Wrap(err, "listing framework defs")
	}
	if len(defs) > 0 {
		return errors.Wrap(ErrInvalidConfigSpec, "trpc/taf application allows only one framework config file")
	}
	return nil
}

// IsFrameworkEnvOverlayCreate 判断旧创建接口是否在「按环境写 framework overlay」。
// 该语义只属于旧 POST /app-config-files，由 handler 分流，不进入 Create。
// todo 旧创建文件实例接口下掉后可删除
func IsFrameworkEnvOverlayCreate(params CreateCfgFileParams) bool {
	return params.ConfigKind == ConfigKindFramework &&
		params.EnvName != EnvNameDefault &&
		params.Type == AppConfigFileTypeOverlay &&
		params.BaseAppConfigFileID != nil
}

// AttachFrameworkEnvOverlay 把环境 overlay 挂到 base 文件所属的 framework def 上，
// 而不是再新建一条 def。供旧 POST /app-config-files handler 调用。
// todo 旧创建文件实例接口下掉后可删除
func (s *AppCfgFileDefService) AttachFrameworkEnvOverlay(
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

	policy, err := s.policyFor(def.ConfigKind)
	if err != nil {
		return nil, errors.Wrap(err, "loading config kind policy for compatibility create")
	}
	strategy, err := policy.GetEnvContentStrategy(def)
	if err != nil {
		return nil, errors.Wrap(err, "resolving env instance strategy for compatibility create")
	}

	envParams := CreateEnvInstanceParams{
		EnvName:     params.EnvName,
		Operator:    params.Creator,
		Description: params.Description,
	}
	raw := lo.FromPtr(params.OverlayContent)
	if strategy == EnvContentStrategyOverwrite {
		envParams.Content = &raw
	} else {
		envParams.OverlayContent = &raw
	}

	created, err := s.CreateEnvInstance(ctx, *defaultFile, envParams)
	if err != nil {
		return nil, errors.Wrap(err, "creating framework env overlay on existing def")
	}
	return &AppConfigFileWithDef{AppConfigFile: *created, Def: def}, nil
}
