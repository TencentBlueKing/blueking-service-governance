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
	"slices"

	"github.com/pkg/errors"
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
	// 兼容旧版调用：未指定 ConfigKind 时默认为 framework。
	if kind == "" {
		kind = ConfigKindFramework
	}

	policy, err := s.policyFor(kind)
	if err != nil {
		return nil, err
	}

	// kind 级别的创建参数校验
	if err = policy.ValidateCreateParams(params); err != nil {
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
			// 初始创建默认为统一配置
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
		if policy, err := s.policyFor(def.ConfigKind); err == nil && !policy.AllowMountDirUpdate() {
			return errors.New("this config kind does not support modifying mountDir via def")
		}
	}

	applyStaticDefFields(def, update)

	if update.HasEnvConfigChanges() {
		params := UpdateEnvConfigParams{
			IsUnifiedConfig: update.IsUnifiedConfig,
			MountedEnvNames: nil,
		}
		if update.MountedEnvNames != nil {
			params.MountedEnvNames = update.MountedEnvNames
		}
		if err := s.UpdateEnvConfig(ctx, def, params); err != nil {
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

// --- 环境实例管理 ---

// GetEnvFileDetail 获取指定环境下的文件详情，通过 ConfigKindPolicy 驱动差异行为：
//
//   - envName 为空 / 统一配置 → DisplayFile = DefaultFile
//   - policy.IsEffectiveForEnv == false → 报错（文件在该环境不生效）
//   - 有环境实例 → DisplayFile = EnvFile
//   - 无实例 + overwrite 策略 → DisplayFile = DefaultFile（默认即生效内容）
//   - 无实例 + overlay 策略 → DisplayFile = nil（无定制内容）
func (s *AppCfgFileDefService) GetEnvFileDetail(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
) (*EnvFileDetailResult, error) {
	policy, defaultFile, envFile, err := s.loadPolicyAndEnvFiles(ctx, def, envName)
	if err != nil {
		return nil, err
	}

	result := &EnvFileDetailResult{Def: def, DefaultFile: defaultFile, DisplayFile: defaultFile}

	// 未指定环境或统一配置：直接返回默认文件
	if envName == EnvNameDefault || !def.EnvConfigMode.IsIndependent() {
		s.fillEditorInfo(ctx, result)
		return result, nil
	}

	// 策略校验：该文件是否在指定环境生效
	if !policy.IsEffectiveForEnv(def, envName) {
		return nil, errors.Errorf("file %s is not effective for env %s", def.Name, envName)
	}

	strategy := policy.GetEnvInstanceStrategy()
	if envFile != nil {
		result.HasEnvInstance = true
		result.DisplayFile = envFile
	} else if strategy == EnvInstanceStrategyOverwrite {
		// overwrite 策略（如 plain）：无实例时默认文件即为生效内容
		result.DisplayFile = defaultFile
	} else if strategy == EnvInstanceStrategyOverlay {
		// overlay 策略（如 framework）：无实例表示无定制，内容为空
		result.DisplayFile = nil
	} else {
		return nil, errors.Errorf("unsupported env instance strategy: %s", strategy)
	}

	s.fillEditorInfo(ctx, result)
	return result, nil
}

// loadPolicyAndEnvFiles 返回单个 def 在目标环境下做后续决策所需的 policy、默认文件和环境实例。
// envFile 仅在“非默认环境 + 独立配置 + 策略上对该环境生效”时才会查询；其余场景固定为 nil。
func (s *AppCfgFileDefService) loadPolicyAndEnvFiles(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
) (
	policy ConfigKindPolicy,
	defaultFile *AppConfigFile,
	envFile *AppConfigFile,
	err error,
) {
	policy, err = s.policyFor(def.ConfigKind)
	if err != nil {
		return nil, nil, nil, err
	}

	defaultFile, err = s.FileStore.GetByDefIDAndEnv(ctx, def.ID, EnvNameDefault)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "loading default file")
	}

	if envName == EnvNameDefault || !def.EnvConfigMode.IsIndependent() || !policy.IsEffectiveForEnv(def, envName) {
		return policy, defaultFile, nil, nil
	}

	envFile, err = s.FindEnvInstance(ctx, def.ID, envName)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "loading env file")
	}
	return policy, defaultFile, envFile, nil
}

// fillEditorInfo 为详情结果填充编辑器相关信息（editableContentField / baseContentInfo）。
// DisplayFile 为 nil 时（overlay 无实例），标记为不可编辑。
func (s *AppCfgFileDefService) fillEditorInfo(ctx context.Context, result *EnvFileDetailResult) {
	if result.DisplayFile == nil {
		result.EditableContentField = string(EditableContentFieldNone)
		return
	}
	editor, err := NewAppConfigFileEditor(s.FileStore, s.DefStore, result.DisplayFile)
	if err != nil {
		result.EditableContentField = string(EditableContentFieldNone)
		return
	}
	result.EditableContentField = string(editor.GetEditableContentField())

	provider, err := NewBaseContentProvider(s.FileStore, s.DefStore, result.DisplayFile)
	if err != nil {
		return
	}
	info, pErr := provider.GetInfo(ctx)
	if pErr != nil {
		// ErrBaseContentEmpty 表示无 base（normal local 文件），不是异常
		return
	}
	result.BaseContentInfo = info
}

// UpsertEnvContent 处理一次内容更新的完整用例：准备目标文件、校验编译后内容，并创建或更新文件版本。
func (s *AppCfgFileDefService) UpsertEnvContent(
	ctx context.Context,
	def *AppConfigFileDef,
	params UpsertEnvContentParams,
) (*UpsertEnvContentResult, error) {
	targetFile, compiledContent, isNewFile, err := s.PrepareEnvContentUpdate(
		ctx, def, params.EnvName, params.Content, params.Operator,
	)
	if err != nil {
		return nil, errors.Wrap(err, "preparing env content update")
	}

	if params.ValidateCompiledContent != nil {
		if err = params.ValidateCompiledContent(targetFile, compiledContent); err != nil {
			return nil, errors.Wrap(err, "validating compiled content")
		}
	}

	if isNewFile {
		targetFile, err = s.CreateFileWithVersion(
			ctx, *targetFile, def.Name, params.Description, params.Operator,
		)
		if err != nil {
			return nil, errors.Wrap(err, "creating env instance")
		}
	} else {
		if err = s.UpdateFile(ctx, targetFile, def.Name, params.Operator, UpdateCfgFileOptions{
			OperationType:          AppConfigFileVersionOperationTypeUpdate,
			Description:            params.Description,
			ExpectedCurrentVersion: params.ExpectedCurrentVersion,
		}); err != nil {
			return &UpsertEnvContentResult{File: targetFile, CompiledContent: compiledContent},
				errors.Wrap(err, "updating target file")
		}
	}

	return &UpsertEnvContentResult{File: targetFile, CompiledContent: compiledContent}, nil
}

// PrepareEnvContentUpdate 解析并应用一次内容更新，但不持久化，返回已写入内存的目标文件、编译后的内容，以及是否需要新建环境实例。
func (s *AppCfgFileDefService) PrepareEnvContentUpdate(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
	content string,
	operator string,
) (*AppConfigFile, string, bool, error) {
	if def == nil {
		return nil, "", false, errors.New("def is required")
	}

	policy, err := s.policyFor(def.ConfigKind)
	if err != nil {
		return nil, "", false, err
	}

	defaultFileWithDef, err := s.GetDefaultFileWithDef(ctx, def.ID)
	if err != nil {
		return nil, "", false, err
	}

	if err = policy.ValidateContent(content, defaultFileWithDef.GetConfigFormat()); err != nil {
		return nil, "", false, errors.Wrap(ErrInvalidConfigSpec, err.Error())
	}

	targetFile, isNewFile, err := s.prepareUpsertFileForContentUpdate(
		ctx, def, envName, content, operator, policy, *defaultFileWithDef,
	)
	if err != nil {
		return nil, "", false, err
	}

	compiledContent, err := s.applyContentUpdate(ctx, targetFile, content)
	if err != nil {
		return nil, "", false, err
	}
	return targetFile, compiledContent, isNewFile, nil
}

// FindEnvInstance 查找指定 def 下某环境的实例记录，不存在返回 nil。
func (s *AppCfgFileDefService) FindEnvInstance(
	ctx context.Context,
	defID bson.ObjectID,
	envName string,
) (*AppConfigFile, error) {
	acf, err := s.FileStore.GetByDefIDAndEnv(ctx, defID, envName)
	if err != nil {
		if errors.Is(err, ErrAppConfigFileNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return acf, nil
}

// CreateEnvInstance 基于默认文件创建环境级配置实例。
// framework → overlay 类型（仅存 overlayContent，部署时与默认文件 merge）。
// plain → normal 类型（完整复制默认内容，overwrite 语义）。
func (s *AppCfgFileDefService) CreateEnvInstance(
	ctx context.Context,
	defaultFile AppConfigFileWithDef,
	params CreateEnvInstanceParams,
) (*AppConfigFile, error) {
	acf, err := s.buildEnvInstance(defaultFile, params)
	if err != nil {
		return nil, err
	}
	return s.CreateFileWithVersion(ctx, *acf, defaultFile.GetName(), params.Description, params.Operator)
}

func (s *AppCfgFileDefService) prepareUpsertFileForContentUpdate(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
	content string,
	operator string,
	policy ConfigKindPolicy,
	defaultFile AppConfigFileWithDef,
) (*AppConfigFile, bool, error) {
	if envName == EnvNameDefault || !def.EnvConfigMode.IsIndependent() {
		return &defaultFile.AppConfigFile, false, nil
	}

	if !policy.IsEffectiveForEnv(def, envName) {
		return nil, false, errors.Wrapf(ErrInvalidConfigSpec, "file %s is not effective for env %s", def.Name, envName)
	}

	existing, err := s.FindEnvInstance(ctx, def.ID, envName)
	if err != nil {
		return nil, false, errors.Wrap(err, "finding env instance")
	}
	if existing != nil {
		return existing, false, nil
	}

	params := CreateEnvInstanceParams{
		EnvName:  envName,
		Operator: operator,
	}
	if policy.GetEnvInstanceStrategy() == EnvInstanceStrategyOverwrite {
		params.Content = &content
	} else {
		params.OverlayContent = &content
	}

	acf, err := s.buildEnvInstance(defaultFile, params)
	if err != nil {
		return nil, false, err
	}
	return acf, true, nil
}

func (s *AppCfgFileDefService) applyContentUpdate(
	ctx context.Context,
	targetFile *AppConfigFile,
	content string,
) (string, error) {
	editor, err := NewAppConfigFileEditor(s.FileStore, s.DefStore, targetFile)
	if err != nil {
		return "", errors.Wrap(err, "creating app config file editor")
	}

	switch editor.GetEditableContentField() {
	case EditableContentFieldContent:
		if err = editor.SetContent(content); err != nil {
			return "", errors.Wrap(err, "setting content")
		}
	case EditableContentFieldOverlayContent:
		if err = editor.SetOverlayContent(content); err != nil {
			return "", errors.Wrap(err, "setting overlay content")
		}
	default:
		return "", errors.Wrap(ErrInvalidConfigSpec, "target file has no editable content field")
	}

	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return "", errors.Wrap(err, "compiling content")
	}
	return compiledContent, nil
}

func (s *AppCfgFileDefService) buildEnvInstance(
	defaultFile AppConfigFileWithDef,
	params CreateEnvInstanceParams,
) (*AppConfigFile, error) {
	kind := defaultFile.GetConfigKind()
	switch kind {
	case ConfigKindPlain:
		return s.buildPlainEnvInstance(defaultFile, params)
	case ConfigKindFramework:
		return s.buildFrameworkEnvOverlay(defaultFile, params)
	default:
		return nil, errors.Errorf("unsupported config kind for env instance: %s", kind)
	}
}

// buildPlainEnvInstance 以 overwrite 方式复制默认内容构造 plain 环境实例。
func (s *AppCfgFileDefService) buildPlainEnvInstance(
	defaultFile AppConfigFileWithDef,
	params CreateEnvInstanceParams,
) (*AppConfigFile, error) {
	if !(PlainPolicy{}).IsEffectiveForEnv(defaultFile.Def, params.EnvName) {
		return nil, errors.Wrap(ErrInvalidConfigSpec, "plain env instance envName is not in mountedEnvNames")
	}
	// 确定初始内容：优先使用参数传入的内容，否则复制默认文件内容。
	content := defaultFile.Content
	if params.Content != nil {
		content = params.Content
	}
	acf := AppConfigFile{
		DefID:   defaultFile.DefID,
		AppID:   defaultFile.AppID,
		EnvName: params.EnvName,
		Type:    AppConfigFileTypeNormal,
		VersionedContent: VersionedContent{
			ContentSourceType: ContentSourceTypeLocal,
			Format:            defaultFile.GetConfigFormat(),
			Content:           content,
		},
		Creator:        params.Operator,
		Updater:        params.Operator,
		CurrentVersion: 1,
	}
	return &acf, nil
}

// buildFrameworkEnvOverlay 构造 framework overlay 类型的环境实例。
func (s *AppCfgFileDefService) buildFrameworkEnvOverlay(
	defaultFile AppConfigFileWithDef,
	params CreateEnvInstanceParams,
) (*AppConfigFile, error) {
	if params.OverlayContent == nil {
		return nil, errors.Wrap(ErrInvalidConfigSpec, "framework env overlay requires overlayContent")
	}
	baseID := defaultFile.ID
	acf := AppConfigFile{
		DefID:   defaultFile.DefID,
		AppID:   defaultFile.AppID,
		EnvName: params.EnvName,
		Type:    AppConfigFileTypeOverlay,
		VersionedContent: VersionedContent{
			// 环境 overlay 来自用户输入的补丁内容，而不是继承默认文件的数据源。
			ContentSourceType:   ContentSourceTypeLocal,
			Format:              defaultFile.GetConfigFormat(),
			OverlayContent:      params.OverlayContent,
			BaseAppConfigFileID: &baseID,
		},
		Creator:        params.Operator,
		Updater:        params.Operator,
		CurrentVersion: 1,
	}
	return &acf, nil
}

// UpdateEnvConfig 更新环境配置策略（统一/独立配置模式 + 挂载环境列表）。
func (s *AppCfgFileDefService) UpdateEnvConfig(
	ctx context.Context,
	def *AppConfigFileDef,
	params UpdateEnvConfigParams,
) error {
	// 处理统一/独立配置切换
	if params.IsUnifiedConfig != nil {
		if err := s.applyEnvConfigChange(ctx, def, *params.IsUnifiedConfig); err != nil {
			return err
		}
	}
	// 处理挂载环境列表变更（仅 plain 有效）
	if params.MountedEnvNames != nil {
		if err := s.applyMountedEnvNamesChange(ctx, def, *params.MountedEnvNames); err != nil {
			return err
		}
	}
	return nil
}

// ResetEnvInstanceToDefault 恢复指定环境为默认配置，删除该环境的独立实例及其版本历史。
func (s *AppCfgFileDefService) ResetEnvInstanceToDefault(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
) error {
	if !def.EnvConfigMode.IsIndependent() {
		return ErrResetToDefaultRequiresIndependentConfig
	}
	return s.deleteEnvInstanceByName(ctx, def.ID, def.AppID, envName)
}

// CleanupPlainEnvInstancesByEnv 环境被删除时，清理该环境下所有 plain def 的实例，
// 并从各 def 的 mountedEnvNames 中移除该环境名。
//
// TODO(non-atomic): 当前逐条删除实例并更新 def，若中途失败会出现部分 def 已更新、
// 部分未更新的中间状态。后续应引入 MongoDB 事务或补偿机制保证批量操作的原子性。
func (s *AppCfgFileDefService) CleanupPlainEnvInstancesByEnv(
	ctx context.Context,
	appID, envName string,
) error {
	plainDefs, err := s.DefStore.ListByApp(ctx, appID, DefFilterConfigKind(ConfigKindPlain))
	if err != nil {
		return errors.Wrap(err, "listing plain defs for env cleanup")
	}
	for _, def := range plainDefs {
		// 删除该环境的实例
		if err = s.deleteEnvInstanceByName(ctx, def.ID, appID, envName); err != nil {
			return err
		}
		// 从 mountedEnvNames 中移除
		if def.EnvConfigMode.ContainsEnv(envName) {
			newNames := make([]string, 0, len(def.EnvConfigMode.MountedEnvNames))
			for _, n := range def.EnvConfigMode.MountedEnvNames {
				if n != envName {
					newNames = append(newNames, n)
				}
			}
			def.EnvConfigMode.MountedEnvNames = newNames
			if _, err = s.DefStore.Update(ctx, def); err != nil {
				return errors.Wrapf(err, "update def %s after removing env %s", def.ID.Hex(), envName)
			}
		}
	}
	return nil
}

// --- 挂载预览 ---

// MountPreviewItem 挂载预览中的单条文件信息。
type MountPreviewItem struct {
	Def           AppConfigFileDef
	ContentSource AppConfigFileType // normal / overlay / overwrite
	HasEnvFile    bool              // 该环境是否有独立的文件实例
}

// GetMountPreview 获取指定环境下最终会挂载到容器内的所有配置文件及其内容来源。
func (s *AppCfgFileDefService) GetMountPreview(
	ctx context.Context,
	appID, envName string,
) ([]MountPreviewItem, error) {
	defs, err := s.DefStore.ListByApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "listing defs for mount preview")
	}

	result := make([]MountPreviewItem, 0, len(defs))
	for _, def := range defs {
		policy, pErr := s.policyFor(def.ConfigKind)
		if pErr != nil {
			continue
		}
		// 策略判断该 def 在目标环境是否生效
		if !policy.IsEffectiveForEnv(&def, envName) {
			continue
		}

		item := MountPreviewItem{Def: def, ContentSource: AppConfigFileTypeNormal, HasEnvFile: false}

		// 查找该环境是否有独立实例
		envFile, fErr := s.FileStore.GetByDefIDAndEnv(ctx, def.ID, envName)
		if fErr != nil && !errors.Is(fErr, ErrAppConfigFileNotFound) {
			return nil, errors.Wrapf(fErr, "querying env instance for def %s", def.ID.Hex())
		}
		if envFile != nil {
			item.HasEnvFile = true
			item.ContentSource = envFile.Type
			// overwrite 策略下 normal 实例语义为 overwrite
			if policy.GetEnvInstanceStrategy() == EnvInstanceStrategyOverwrite &&
				envFile.Type == AppConfigFileTypeNormal {
				item.ContentSource = "overwrite"
			}
		}

		result = append(result, item)
	}
	return result, nil
}

// --- 内部辅助函数 ---

// applyMountedEnvNamesChange 处理 plain 文件 mountedEnvNames 变更的副作用：
// 被移除的环境需要清理对应的环境实例。
//
// TODO(non-atomic): 当前先删除环境实例，再由上层 UpdateAppCfgFileDef 持久化 def。
// 后续应引入 MongoDB 事务或补偿机制保证操作原子性。
func (s *AppCfgFileDefService) applyMountedEnvNamesChange(
	ctx context.Context,
	def *AppConfigFileDef,
	newEnvNames []string,
) error {
	oldEnvNames := def.EnvConfigMode.MountedEnvNames
	def.EnvConfigMode.MountedEnvNames = newEnvNames

	// 找出被移除的环境，清理对应实例
	for _, oldEnv := range oldEnvNames {
		if !slices.Contains(newEnvNames, oldEnv) {
			if err := s.deleteEnvInstanceByName(ctx, def.ID, def.AppID, oldEnv); err != nil {
				return errors.Wrapf(err, "cleanup env instance for removed env %s", oldEnv)
			}
		}
	}
	return nil
}

// deleteEnvInstanceByName 删除指定 def + envName 的环境实例及其版本。
func (s *AppCfgFileDefService) deleteEnvInstanceByName(
	ctx context.Context,
	defID bson.ObjectID,
	appID string,
	envName string,
) error {
	acf, err := s.FileStore.GetByDefIDAndEnv(ctx, defID, envName)
	if err != nil {
		if errors.Is(err, ErrAppConfigFileNotFound) {
			return nil
		}
		return err
	}
	if _, err = s.FileStore.DeleteByID(ctx, appID, acf.ID); err != nil {
		return errors.Wrap(err, "delete env instance")
	}
	if _, err = s.VersionStore.DeleteByFileID(ctx, acf.ID); err != nil {
		return errors.Wrap(err, "delete versions for env instance")
	}
	return nil
}
