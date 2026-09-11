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

// --- 环境实例查询 ---

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

// --- 环境实例创建 ---

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

// --- 环境实例变更 ---

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

// --- 环境实例删除 / 清理 ---

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
