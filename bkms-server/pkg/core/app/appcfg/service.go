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
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// AppConfigFileService encapsulates file and version persistence logic.
type AppConfigFileService struct {
	fileStore    AppConfigFileStore
	versionStore AppConfigFileVersionStore
}

var (
	// ErrInvalidConfigSpec 表示配置文件字段组合不满足当前领域约束。
	ErrInvalidConfigSpec = errors.New("invalid app config file spec")
	// ErrPlainConfigMountPathConflict 表示同一应用下已有其他 plain 文件占用了相同 mountPath。
	ErrPlainConfigMountPathConflict = errors.New("plain config mountPath already exists")
	// ErrEnvConfigRequiresDefaultFile 表示只有默认逻辑文件才能切换按环境配置模式。
	ErrEnvConfigRequiresDefaultFile = errors.New("env config can only be updated on default config file")
	// ErrPlainEnvInstanceDeleteNotAllowed 表示 plain 环境实例必须通过默认逻辑文件的 env-config-policy
	// 接口统一管理删除。
	ErrPlainEnvInstanceDeleteNotAllowed = errors.New(
		"plain env instance must be deleted via env-config-policy on the default file",
	)
	// ErrFallbackRequiresIndependentConfig 表示 FallbackConfigEnv 仅在按环境独立配置模式下有效。
	ErrFallbackRequiresIndependentConfig = errors.New(
		"fallbackConfigEnv is only valid in independent env config mode",
	)
)

// NewAppConfigFileService creates a service instance.
func NewAppConfigFileService(
	fileStore AppConfigFileStore,
	versionStore AppConfigFileVersionStore,
) *AppConfigFileService {
	return &AppConfigFileService{
		fileStore:    fileStore,
		versionStore: versionStore,
	}
}

// Create persists a new config file and creates the initial version.
func (s *AppConfigFileService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFile, error) {
	params = normalizeCreateCfgFileParams(params)
	params.MountedEnvNames = normalizeEnvNames(params.MountedEnvNames)
	if err := validatePlainCreateParams(params); err != nil {
		return nil, err
	}

	acf := AppConfigFile{
		AppConfigFileContentSpec: AppConfigFileContentSpec{
			AppID:      params.AppID,
			EnvName:    params.EnvName,
			Name:       params.Name,
			Type:       params.Type,
			ConfigKind: params.ConfigKind,
			Creator:    params.Creator,
			VersionedContent: VersionedContent{
				ContentSourceType: params.ContentSourceType,
				Format:            params.Format,
				BSCPConfig:        params.BSCPConfig,
				Content:           params.Content,
				OverlayContent:    params.OverlayContent,
			},
		},
		EnvConfigPolicy: EnvConfigPolicy{
			MountPath:              params.MountPath,
			DefaultAppConfigFileID: params.DefaultAppConfigFileID,
			IsUnifiedConfig:        params.IsUnifiedConfig,
			MountedEnvNames:        params.MountedEnvNames,
		},
		Updater:        params.Creator,
		CurrentVersion: 1,
	}
	if params.BaseAppConfigFileID != nil {
		acf.BaseAppConfigFileID = params.BaseAppConfigFileID
	}
	acf.initializeContentFields(params.Type, params.ContentSourceType)
	if params.Content != nil {
		acf.Content = params.Content
	}
	if params.OverlayContent != nil {
		acf.OverlayContent = params.OverlayContent
	}
	if acf.Format == "" {
		acf.Format = FileFormatYAML
	}
	if acf.ConfigKind == "" {
		acf.ConfigKind = ConfigKindFramework
	}
	if err := s.validatePlainConfigFile(ctx, &acf); err != nil {
		return nil, err
	}
	if err := s.validateOverlayBase(ctx, &acf); err != nil {
		return nil, err
	}

	fileID, err := s.createFileWithInitialVersion(ctx, acf, params.Description)
	if err != nil {
		return nil, err
	}
	acf.ID = fileID
	return &acf, nil
}

func (s *AppConfigFileService) createFileWithInitialVersion(
	ctx context.Context,
	acf AppConfigFile,
	description string,
) (bson.ObjectID, error) {
	fileID, err := s.fileStore.Add(ctx, acf)
	if err != nil {
		return bson.NilObjectID, err
	}
	if s.versionStore == nil {
		return bson.NilObjectID, errors.New("versionStore is required but not configured")
	}

	acf.ID = fileID
	version, err := s.buildVersionRecord(
		ctx,
		acf,
		AppConfigFileVersionOperationTypeCreate,
		description,
		nil,
		acf.Creator,
	)
	if err != nil {
		return bson.NilObjectID, err
	}
	if _, err = s.versionStore.Add(ctx, version); err != nil {
		return bson.NilObjectID, err
	}
	return fileID, nil
}

// UpdateFile persists a file change and appends a version record when requested.
func (s *AppConfigFileService) UpdateFile(
	ctx context.Context,
	acf *AppConfigFile,
	operator string,
	opts UpdateCfgFileOptions,
) error {
	if err := s.validatePlainConfigFile(ctx, acf); err != nil {
		return err
	}
	if err := s.validateOverlayBase(ctx, acf); err != nil {
		return err
	}

	curVersion := acf.CurrentVersion

	if opts.ExpectedCurrentVersion != nil && *opts.ExpectedCurrentVersion != curVersion {
		return ErrAppConfigFileVersionConflict
	}

	acf.CurrentVersion = curVersion + 1
	acf.Updater = operator

	if s.versionStore == nil {
		return errors.New("versionStore is required but not configured")
	}

	version, err := s.buildVersionRecord(
		ctx,
		*acf,
		opts.OperationType,
		opts.Description,
		opts.RollbackFromVersion,
		operator,
	)
	if err != nil {
		return err
	}

	if _, err = s.fileStore.UpdateIfVersionMatches(ctx, *acf, curVersion); err != nil {
		return err
	}

	_, err = s.versionStore.Add(ctx, version)
	if err != nil {
		return err
	}
	return nil
}

// UpdateEnvConfig 用于在统一配置模式与按环境独立配置模式之间切换默认逻辑文件。
func (s *AppConfigFileService) UpdateEnvConfig(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	if defaultFile == nil || defaultFile.EnvName != EnvNameDefault {
		return ErrEnvConfigRequiresDefaultFile
	}
	params.MountedEnvNames = normalizeEnvNames(params.MountedEnvNames)
	params.FallbackConfigEnv = strings.TrimSpace(params.FallbackConfigEnv)

	if defaultFile.GetConfigKind() == ConfigKindPlain {
		return s.updatePlainEnvConfig(ctx, defaultFile, params)
	}
	if params.MountedEnvNames != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config does not support mountedEnvNames")
	}
	return s.updateFrameworkEnvConfig(ctx, defaultFile, params)
}

// CleanupPlainEnvInstancesByEnv 在环境被删除后清理对应的 plain 环境实例，并同步根文件元数据。
func (s *AppConfigFileService) CleanupPlainEnvInstancesByEnv(
	ctx context.Context,
	appID string,
	envName string,
) error {
	allFiles, err := s.fileStore.List(ctx, appID)
	if err != nil {
		return err
	}
	// 第一步：删除该环境对应的 plain 独立实例。
	for _, item := range allFiles {
		if item.GetConfigKind() != ConfigKindPlain || item.EnvName != envName || item.DefaultAppConfigFileID == nil {
			continue
		}
		if _, err = s.deleteFileRecordAndVersions(ctx, appID, item.ID); err != nil {
			return err
		}
	}

	// 第二步：扫描全部 plain root，从指定环境挂载范围中移除已删除的环境名。
	// 引用状态下没有独立实例，也必须同步回收 MountedEnvNames。
	for _, item := range allFiles {
		if item.GetConfigKind() != ConfigKindPlain || item.EnvName != EnvNameDefault {
			continue
		}
		// nil 表示"全部环境"模式，删除某个环境不影响语义，无需更新。
		if item.MountedEnvNames == nil {
			continue
		}
		if !lo.Contains(item.MountedEnvNames, envName) {
			continue
		}
		rootFile, getErr := s.fileStore.GetByID(ctx, item.ID)
		if getErr != nil {
			return getErr
		}
		updated := lo.Without(rootFile.MountedEnvNames, envName)
		if slices.Equal(updated, rootFile.MountedEnvNames) {
			continue
		}
		// lo.Without 返回空切片（而非 nil），保持"指定环境"语义，不会意外切换为"全部环境"模式。
		rootFile.MountedEnvNames = updated
		if err = s.UpdateFile(ctx, rootFile, CfgSystemUser, UpdateCfgFileOptions{
			OperationType: AppConfigFileVersionOperationTypeUpdate,
			Description:   "环境删除时回收配置实例",
		}); err != nil {
			return err
		}
	}
	return nil
}

// updateFrameworkEnvConfig 更新 framework 文件的按环境配置元数据。
// framework 文件不支持 mountedEnvNames（始终对所有环境生效），因此只切换 isUnifiedConfig 标志。
func (s *AppConfigFileService) updateFrameworkEnvConfig(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	// 状态未变化时直接跳过，避免产生无意义的版本记录。
	if defaultFile.IsUnifiedConfig == params.IsUnifiedConfig {
		return nil
	}

	// 开启独立配置：仅更新元数据标志，不创建 env instance（由前端按需创建）。
	if !params.IsUnifiedConfig {
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = nil
		return s.UpdateFile(ctx, defaultFile, params.Operator, UpdateCfgFileOptions{
			OperationType:          AppConfigFileVersionOperationTypeUpdate,
			Description:            params.Description,
			ExpectedCurrentVersion: params.ExpectedCurrentVersion,
		})
	}

	// 切回统一配置：先删除环境实例，再把 root 标成统一，避免中途失败留下「已统一但仍有 instance」。
	envFiles, err := s.listEnvInstanceFiles(ctx, *defaultFile)
	if err != nil {
		return err
	}
	for _, envFile := range envFiles {
		if _, err = s.deleteFileRecordAndVersions(ctx, defaultFile.AppID, envFile.ID); err != nil {
			return err
		}
	}
	defaultFile.IsUnifiedConfig = true
	defaultFile.MountedEnvNames = nil
	return s.UpdateFile(ctx, defaultFile, params.Operator, UpdateCfgFileOptions{
		OperationType:          AppConfigFileVersionOperationTypeUpdate,
		Description:            params.Description,
		ExpectedCurrentVersion: params.ExpectedCurrentVersion,
	})
}

// updatePlainEnvConfig 更新 plain 文件的按环境配置元数据，并同步环境实例。
func (s *AppConfigFileService) updatePlainEnvConfig(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	// 优先处理回退为共用配置。
	if params.FallbackConfigEnv != "" {
		return s.fallbackPlainEnvInstance(ctx, defaultFile, params)
	}
	if !params.IsUnifiedConfig {
		return s.enablePlainEnvConfig(ctx, defaultFile, params)
	}
	return s.disablePlainEnvConfig(ctx, defaultFile, params)
}

// enablePlainEnvConfig 开启或调整按环境独立配置。
// 引用模型下不预创建 env instance，只更新默认记录元数据。
// 若调整挂载范围导致某些已有 env instance 不在新范围内，删除这些实例。
func (s *AppConfigFileService) enablePlainEnvConfig(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	// 已经是独立配置且挂载范围未变化时跳过，避免产生无意义的版本记录。
	if !defaultFile.IsUnifiedConfig && slices.Equal(defaultFile.MountedEnvNames, params.MountedEnvNames) {
		return nil
	}
	if params.MountedEnvNames != nil {
		envFiles, err := s.listEnvInstanceFiles(ctx, *defaultFile)
		if err != nil {
			return err
		}
		targetEnvSet := make(map[string]struct{}, len(params.MountedEnvNames))
		for _, envName := range params.MountedEnvNames {
			targetEnvSet[envName] = struct{}{}
		}
		for _, envFile := range envFiles {
			if _, ok := targetEnvSet[envFile.EnvName]; ok {
				continue
			}
			if _, err = s.deleteFileRecordAndVersions(ctx, defaultFile.AppID, envFile.ID); err != nil {
				return err
			}
		}
	}

	defaultFile.IsUnifiedConfig = false
	defaultFile.MountedEnvNames = cloneStringSlice(params.MountedEnvNames)
	return s.UpdateFile(ctx, defaultFile, params.Operator, UpdateCfgFileOptions{
		OperationType:          AppConfigFileVersionOperationTypeUpdate,
		Description:            params.Description,
		ExpectedCurrentVersion: params.ExpectedCurrentVersion,
	})
}

// disablePlainEnvConfig 切回统一配置：先删除环境实例，再把 root 标成统一。
// 已经是统一模式时，仍允许单独更新挂载范围。
func (s *AppConfigFileService) disablePlainEnvConfig(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	if defaultFile.IsUnifiedConfig && slices.Equal(defaultFile.MountedEnvNames, params.MountedEnvNames) {
		return nil
	}
	if !defaultFile.IsUnifiedConfig {
		envFiles, listErr := s.listEnvInstanceFiles(ctx, *defaultFile)
		if listErr != nil {
			return listErr
		}
		for _, envFile := range envFiles {
			if _, err := s.deleteFileRecordAndVersions(ctx, defaultFile.AppID, envFile.ID); err != nil {
				return err
			}
		}
	}
	defaultFile.IsUnifiedConfig = true
	defaultFile.MountedEnvNames = cloneStringSlice(params.MountedEnvNames)
	return s.UpdateFile(ctx, defaultFile, params.Operator, UpdateCfgFileOptions{
		OperationType:          AppConfigFileVersionOperationTypeUpdate,
		Description:            params.Description,
		ExpectedCurrentVersion: params.ExpectedCurrentVersion,
	})
}

// fallbackPlainEnvInstance 将指定环境回退为引用状态，删除其独立记录和版本历史。
func (s *AppConfigFileService) fallbackPlainEnvInstance(
	ctx context.Context,
	defaultFile *AppConfigFile,
	params UpdateEnvConfigParams,
) error {
	if defaultFile.IsUnifiedConfig {
		return ErrFallbackRequiresIndependentConfig
	}
	envFiles, err := s.listEnvInstanceFiles(ctx, *defaultFile)
	if err != nil {
		return err
	}
	for _, envFile := range envFiles {
		if envFile.EnvName != params.FallbackConfigEnv {
			continue
		}
		if _, err = s.deleteFileRecordAndVersions(ctx, defaultFile.AppID, envFile.ID); err != nil {
			return err
		}
		break
	}
	return nil
}

// normalizeCreateCfgFileParams applies default values expected by new writes.
func normalizeCreateCfgFileParams(params CreateCfgFileParams) CreateCfgFileParams {
	if params.ConfigKind == "" {
		params.ConfigKind = ConfigKindFramework
	}
	params.MountPath = strings.TrimSpace(params.MountPath)
	return params
}

// validatePlainCreateParams 在持久化之前，校验普通文件（plain-file）类型配置特有的创建约束。
func validatePlainCreateParams(params CreateCfgFileParams) error {
	if params.ConfigKind != ConfigKindPlain && strings.TrimSpace(params.MountPath) != "" {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config does not support mountPath")
	}
	if params.ConfigKind != ConfigKindPlain && params.MountedEnvNames != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config does not support mountedEnvNames")
	}
	if params.ConfigKind != ConfigKindPlain {
		return nil
	}
	if params.Type != AppConfigFileTypeNormal {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config only supports normal type")
	}
	if params.ContentSourceType != ContentSourceTypeLocal {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config only supports local content source")
	}
	if params.BaseAppConfigFileID != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config does not support base app config file")
	}
	if params.OverlayContent != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config does not support overlay content")
	}
	if err := validatePlainMountPath(params.MountPath); err != nil {
		return err
	}
	if params.EnvName != EnvNameDefault && params.DefaultAppConfigFileID == nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain env instance requires defaultAppConfigFileID")
	}
	return nil
}

// validatePlainMountPath 要求 mountPath 是容器内的绝对文件路径：以 / 开头、不是根目录、不以 / 结尾。
func validatePlainMountPath(mountPath string) error {
	if mountPath == "" {
		return errors.Wrap(ErrInvalidConfigSpec, "mountPath is required for plain config")
	}
	if !strings.HasPrefix(mountPath, "/") || mountPath == "/" || strings.HasSuffix(mountPath, "/") {
		return errors.Wrap(ErrInvalidConfigSpec, "mountPath must be an absolute file path")
	}
	return nil
}

// validatePlainConfigFile 校验 plain 文件约束以及 mountPath 唯一性。
func (s *AppConfigFileService) validatePlainConfigFile(ctx context.Context, acf *AppConfigFile) error {
	if acf == nil {
		return nil
	}
	acf.MountPath = strings.TrimSpace(acf.MountPath)
	if acf.GetConfigKind() != ConfigKindPlain {
		if acf.MountPath != "" {
			return errors.Wrap(ErrInvalidConfigSpec, "framework config does not support mountPath")
		}
		if acf.MountedEnvNames != nil {
			return errors.Wrap(ErrInvalidConfigSpec, "framework config does not support mountedEnvNames")
		}
		return nil
	}
	if acf.Type != AppConfigFileTypeNormal {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config only supports normal type")
	}
	if acf.ContentSourceType != ContentSourceTypeLocal {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config only supports local content source")
	}
	if acf.BaseAppConfigFileID != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config does not support base app config file")
	}
	if acf.OverlayContent != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config does not support overlay content")
	}

	if err := validatePlainMountPath(acf.MountPath); err != nil {
		return err
	}
	if acf.DefaultAppConfigFileID != nil {
		root, err := s.fileStore.GetByID(ctx, *acf.DefaultAppConfigFileID)
		if err != nil {
			return err
		}
		if root != nil && root.MountPath != acf.MountPath {
			return errors.Wrap(ErrInvalidConfigSpec, "plain env instance cannot change mountPath")
		}
	}

	existingFiles, err := s.fileStore.ListByAppAndMountPath(ctx, acf.AppID, acf.MountPath)
	if err != nil {
		return err
	}
	for _, existing := range existingFiles {
		if existing.ID == acf.ID {
			continue
		}
		if belongToSamePlainLogicalFile(*acf, existing) {
			continue
		}
		return ErrPlainConfigMountPathConflict
	}
	return nil
}

// validateOverlayBase 要求 overlay 的 base 必须是同一应用下的 framework 文件。
func (s *AppConfigFileService) validateOverlayBase(ctx context.Context, acf *AppConfigFile) error {
	if acf == nil || acf.Type != AppConfigFileTypeOverlay || acf.BaseAppConfigFileID == nil {
		return nil
	}
	base, err := s.fileStore.GetByID(ctx, *acf.BaseAppConfigFileID)
	if err != nil {
		return err
	}
	if base == nil {
		return errors.Wrap(ErrInvalidConfigSpec, "overlay base app config file not found")
	}
	if base.GetConfigKind() != ConfigKindFramework {
		return errors.Wrap(ErrInvalidConfigSpec, "overlay base must be a framework config file")
	}
	return nil
}

// belongToSamePlainLogicalFile reports whether two plain config records belong to the same logical file group.
func belongToSamePlainLogicalFile(current, existing AppConfigFile) bool {
	currentRootID, currentOK := current.GetLogicalRootID(current.ID)
	existingRootID, existingOK := existing.GetLogicalRootID(existing.ID)
	if currentOK && existingOK && currentRootID == existingRootID {
		return true
	}
	if currentOK && currentRootID == existing.ID {
		return true
	}
	if existingOK && existingRootID == current.ID {
		return true
	}
	return false
}

// listEnvInstanceFiles 列出某个逻辑配置文件下的所有环境级实例记录。
func (s *AppConfigFileService) listEnvInstanceFiles(
	ctx context.Context,
	defaultFile AppConfigFile,
) ([]AppConfigFile, error) {
	allFiles, err := s.fileStore.List(ctx, defaultFile.AppID)
	if err != nil {
		return nil, err
	}
	result := make([]AppConfigFile, 0)
	for _, item := range allFiles {
		if item.EnvName == EnvNameDefault {
			continue
		}
		switch defaultFile.GetConfigKind() {
		case ConfigKindPlain:
			rootID, ok := item.GetLogicalRootID(item.ID)
			if ok && rootID == defaultFile.ID {
				result = append(result, item)
			}
		default:
			// Helm overlay 的 envName 为空，上面已经跳过，不会误删。
			if item.GetConfigKind() != ConfigKindFramework {
				continue
			}
			if item.Type == AppConfigFileTypeNormal {
				result = append(result, item)
				continue
			}
			if item.Type == AppConfigFileTypeOverlay &&
				item.BaseAppConfigFileID != nil &&
				*item.BaseAppConfigFileID == defaultFile.ID {
				result = append(result, item)
			}
		}
	}
	return result, nil
}

// FindPlainEnvInstance 查找默认逻辑文件下指定环境的独立实例。
// 找不到时返回 (nil, nil)。
func (s *AppConfigFileService) FindPlainEnvInstance(
	ctx context.Context,
	defaultFile AppConfigFile,
	envName string,
) (*AppConfigFile, error) {
	envFiles, err := s.listEnvInstanceFiles(ctx, defaultFile)
	if err != nil {
		return nil, err
	}
	for _, item := range envFiles {
		if item.EnvName != envName {
			continue
		}
		found := item
		return &found, nil
	}
	return nil, nil
}

// CreatePlainEnvInstance 基于默认逻辑文件创建一条 plain 环境级配置实例记录。
// 当用户首次修改某个处于引用状态的环境内容时，由 handler 调用此方法，
// 以请求内容为初始值创建独立实例，使该环境脱离对默认配置的引用。
func (s *AppConfigFileService) CreatePlainEnvInstance(
	ctx context.Context,
	defaultFile AppConfigFile,
	envName string,
	content *string,
	operator string,
	description string,
) (*AppConfigFile, error) {
	if !defaultFile.IsMountedToEnv(envName) {
		return nil, errors.Wrap(ErrInvalidConfigSpec, "plain env instance envName is not in mountedEnvNames")
	}
	return s.Create(ctx, CreateCfgFileParams{
		AppID:                  defaultFile.AppID,
		EnvName:                envName,
		Name:                   buildEnvScopedConfigName(defaultFile.Name, envName),
		Type:                   defaultFile.Type,
		ContentSourceType:      defaultFile.ContentSourceType,
		Format:                 defaultFile.GetConfigFormat(),
		ConfigKind:             defaultFile.GetConfigKind(),
		MountPath:              defaultFile.MountPath,
		DefaultAppConfigFileID: &defaultFile.ID,
		IsUnifiedConfig:        true,
		Content:                content,
		Creator:                operator,
		Description:            description,
	})
}

// buildEnvScopedConfigName 为环境实例生成唯一名称（数据库有 appID+name 唯一索引）。
// 例如默认记录 "feature-flags" 的 prod 实例名为 "feature-flags--prod"。
func buildEnvScopedConfigName(baseName, envName string) string {
	return fmt.Sprintf("%s--%s", baseName, envName)
}

func cloneStringSlice(items []string) []string {
	if items == nil {
		return nil
	}
	cloned := append([]string(nil), items...)
	return cloned
}

func normalizeEnvNames(items []string) []string {
	if items == nil {
		return nil
	}
	uniq := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := uniq[item]; ok {
			continue
		}
		uniq[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func (s *AppConfigFileService) deleteFileRecordAndVersions(
	ctx context.Context,
	appID string,
	fileID bson.ObjectID,
) (*AppConfigFile, error) {
	acf, err := s.fileStore.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if _, err = s.fileStore.DeleteByID(ctx, appID, fileID); err != nil {
		return nil, errors.Wrap(err, "delete app config file")
	}
	if s.versionStore != nil {
		if _, err = s.versionStore.DeleteByFileID(ctx, fileID); err != nil {
			return nil, errors.Wrap(err, "delete version records for deleted file")
		}
	}
	return acf, nil
}

// Rollback loads the target version, applies it to the live file, and persists a new rollback version.
func (s *AppConfigFileService) Rollback(
	ctx context.Context,
	appID string,
	versionID bson.ObjectID,
	operator string,
	description string,
	expectedCurrentVersion *int64,
) (*AppConfigFile, *AppConfigFileVersion, error) {
	targetVersion, err := s.GetVersionByAppAndID(ctx, appID, versionID)
	if err != nil {
		return nil, nil, err
	}
	acf, err := s.fileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	rollbackFromVersion := targetVersion.Version
	acf.VersionedContent = targetVersion.VersionedContent

	if description == "" {
		description = fmt.Sprintf("回滚到 v%d", targetVersion.Version)
	}

	if err = s.UpdateFile(
		ctx,
		acf,
		operator,
		UpdateCfgFileOptions{
			OperationType:          AppConfigFileVersionOperationTypeRollback,
			Description:            description,
			RollbackFromVersion:    &rollbackFromVersion,
			ExpectedCurrentVersion: expectedCurrentVersion,
		},
	); err != nil {
		return nil, nil, err
	}
	return acf, targetVersion, nil
}

// ListVersions returns version records by filter.
func (s *AppConfigFileService) ListVersions(
	ctx context.Context,
	opts AppConfigFileVersionListOptions,
) ([]AppConfigFileVersion, int64, error) {
	return s.versionStore.List(ctx, opts)
}

// GetVersionByAppAndID gets one version record and ensures it belongs to the specified app.
func (s *AppConfigFileService) GetVersionByAppAndID(
	ctx context.Context,
	appID string,
	id bson.ObjectID,
) (*AppConfigFileVersion, error) {
	items, err := s.versionStore.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.Wrap(
			ErrAppCfgFileVersionNotFound,
			fmt.Sprintf("app config file version %s not found", id.Hex()),
		)
	}
	return &items[0], nil
}

// CompareVersions returns two versions for diff after ownership and relation checks.
func (s *AppConfigFileService) CompareVersions(
	ctx context.Context,
	appID string,
	previousVersionID bson.ObjectID,
	currentVersionID bson.ObjectID,
) (*AppConfigFileVersion, *AppConfigFileVersion, error) {
	versions, err := s.versionStore.BatchGetByAppAndIDs(
		ctx,
		appID,
		[]bson.ObjectID{previousVersionID, currentVersionID},
	)
	if err != nil {
		return nil, nil, err
	}
	versionMap := make(map[bson.ObjectID]AppConfigFileVersion, len(versions))
	for _, version := range versions {
		versionMap[version.ID] = version
	}
	previous, ok := versionMap[previousVersionID]
	if !ok {
		return nil, nil, errors.Wrap(ErrAppCfgFileVersionNotFound, "previous app config file version not found")
	}
	current, ok := versionMap[currentVersionID]
	if !ok {
		return nil, nil, errors.Wrap(ErrAppCfgFileVersionNotFound, "current app config file version not found")
	}
	if previous.AppConfigFileID != current.AppConfigFileID {
		return nil, nil, ErrComparedVersionsBelongToDifferentFiles
	}
	return &previous, &current, nil
}

// DeleteFile deletes a config file and hard-deletes all its version records.
func (s *AppConfigFileService) DeleteFile(
	ctx context.Context,
	appID string,
	fileID bson.ObjectID,
) (*AppConfigFile, error) {
	acf, err := s.fileStore.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if acf.GetConfigKind() == ConfigKindPlain && acf.DefaultAppConfigFileID != nil && acf.EnvName != EnvNameDefault {
		return nil, ErrPlainEnvInstanceDeleteNotAllowed
	}
	if acf.GetConfigKind() == ConfigKindPlain && acf.EnvName == EnvNameDefault {
		envFiles, listErr := s.listEnvInstanceFiles(ctx, *acf)
		if listErr != nil {
			return nil, listErr
		}
		for _, envFile := range envFiles {
			if _, err = s.deleteFileRecordAndVersions(ctx, appID, envFile.ID); err != nil {
				return nil, err
			}
		}
	}
	if _, err = s.deleteFileRecordAndVersions(ctx, appID, fileID); err != nil {
		return nil, err
	}
	return acf, nil
}

// DeleteVersion soft deletes one historical version after checking it is not the live version.
func (s *AppConfigFileService) DeleteVersion(
	ctx context.Context,
	appID string,
	versionID bson.ObjectID,
	deleter string,
) (*AppConfigFileVersion, *AppConfigFile, error) {
	targetVersion, err := s.GetVersionByAppAndID(ctx, appID, versionID)
	if err != nil {
		return nil, nil, err
	}
	acf, err := s.fileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	if acf.CurrentVersion == targetVersion.Version {
		return nil, nil, ErrUsingVersionCannotBeDeleted
	}
	if _, err = s.versionStore.SoftDeleteByID(ctx, versionID, deleter); err != nil {
		return nil, nil, err
	}
	return targetVersion, acf, nil
}

func (s *AppConfigFileService) buildVersionRecord(
	ctx context.Context,
	acf AppConfigFile,
	operationType AppConfigFileVersionOperationType,
	description string,
	rollbackFromVersion *int64,
	creator string,
) (AppConfigFileVersion, error) {
	version := AppConfigFileVersion{
		AppConfigFileID:          acf.ID,
		AppConfigFileContentSpec: acf.AppConfigFileContentSpec,
		Version:                  acf.CurrentVersion,
		Description:              description,
		OperationType:            operationType,
		RollbackFromVersion:      rollbackFromVersion,
		IsDeleted:                false,
	}
	version.Creator = creator
	version.CreatedAt = time.Now()
	if acf.BaseAppConfigFileID != nil {
		baseAcf, err := s.fileStore.GetByID(ctx, *acf.BaseAppConfigFileID)
		if err != nil {
			return AppConfigFileVersion{}, err
		}
		baseVersion := baseAcf.CurrentVersion
		version.BaseVersion = &baseVersion
	}
	return version, nil
}
