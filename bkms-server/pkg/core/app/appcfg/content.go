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

// ErrNoConfigFileFound is returned when no config file is found for the given criteria.
var ErrNoConfigFileFound = errors.New("no config file found")

// ConfigFileWithContent 配置文件的挂载信息与编译后最终内容的轻量值对象。
type ConfigFileWithContent struct {
	Name     string
	MountDir string
	Content  string
}

type effectiveFileGroup struct {
	defaultFile *AppConfigFile
	envFile     *AppConfigFile
	def         AppConfigFileDef
}

// AppConfigContentProvider 提供环境级配置内容解析能力，workload 层通过该接口获取生效的配置文件内容，无需直接依赖 service 实现。
type AppConfigContentProvider interface {
	// GetFrameworkContent 获取 framework 类型配置（单条）。
	GetFrameworkContent(ctx context.Context, appID, envName string) (*ConfigFileWithContent, error)
	// GetPlainContents 获取 plain 类型配置（可能多条）。
	GetPlainContents(ctx context.Context, appID, envName string) ([]ConfigFileWithContent, error)
}

// NewContentProvider 构造一个 AppConfigContentProvider 实例
func NewContentProvider(
	fileStore AppConfigFileStore,
	defStore AppConfigFileDefStore,
	versionStore AppConfigFileVersionStore,
) AppConfigContentProvider {
	return NewAppConfigFileService(fileStore, defStore, versionStore)
}

// GetFrameworkContent 获取应用在指定环境下生效的框架配置文件内容。
// TODO: 待挂载路径迁移至 def 后，plugin 应改为使用此处返回的 MountDir。
func (s *AppCfgFileDefService) GetFrameworkContent(
	ctx context.Context,
	appID, envName string,
) (*ConfigFileWithContent, error) {
	items, err := s.listEffectiveContents(ctx, appID, envName, ConfigKindFramework)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNoConfigFileFound
	}
	// framework 类型约束为单条 def，多条时视为数据异常并报错。
	if len(items) > 1 {
		return nil, errors.Errorf("multiple framework config files found for app %s env %s", appID, envName)
	}
	return &items[0], nil
}

// GetPlainContents 获取应用在指定环境下生效的所有 plain 配置文件内容。
func (s *AppCfgFileDefService) GetPlainContents(
	ctx context.Context,
	appID, envName string,
) ([]ConfigFileWithContent, error) {
	return s.listEffectiveContents(ctx, appID, envName, ConfigKindPlain)
}

// listEffectiveContents 获取指定应用在目标环境下实际生效的配置文件内容。
func (s *AppCfgFileDefService) listEffectiveContents(
	ctx context.Context,
	appID, envName string,
	configKind ConfigKind,
) ([]ConfigFileWithContent, error) {
	// 1. 按 configKind 查 def 列表
	defs, err := s.DefStore.ListByApp(ctx, appID, DefFilterConfigKind(configKind))
	if err != nil {
		return nil, errors.Wrapf(err, "list %s defs for app %s", configKind, appID)
	}
	if len(defs) == 0 {
		return nil, nil
	}

	defIDs := make([]bson.ObjectID, 0, len(defs))
	defByID := make(map[bson.ObjectID]AppConfigFileDef, len(defs))
	for _, d := range defs {
		defIDs = append(defIDs, d.ID)
		defByID[d.ID] = d
	}

	// 2. 查默认文件 + 目标环境文件
	envNames := []AcfListOption{AcfFilterDefIDs(defIDs)}
	configFiles, err := s.FileStore.List(ctx, appID, envNames...)
	if err != nil {
		return nil, errors.Wrapf(err, "list %s config files for app %s", configKind, appID)
	}

	// 3. 按 def 分组：默认文件 + 环境实例
	groups := make(map[string]*effectiveFileGroup)
	for i := range configFiles {
		f := &configFiles[i]
		defHex := f.DefID.Hex()
		g, ok := groups[defHex]
		if !ok {
			def := defByID[f.DefID]
			g = &effectiveFileGroup{def: def}
			groups[defHex] = g
		}
		if f.EnvName == EnvNameDefault {
			g.defaultFile = f
		} else if f.EnvName == envName {
			g.envFile = f
		}
	}

	// 4. 解析每个 def 的生效内容
	result := make([]ConfigFileWithContent, 0, len(defs))
	for _, def := range defs {
		policy, pErr := s.policyFor(def.ConfigKind)
		if pErr != nil {
			continue
		}
		if !policy.IsEffectiveForEnv(&def, envName) {
			continue
		}

		g := groups[def.ID.Hex()]
		if g == nil || g.defaultFile == nil {
			continue
		}

		item, itemErr := s.buildEffectiveContentItem(ctx, def, envName, g)
		if itemErr != nil {
			return nil, itemErr
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *AppCfgFileDefService) buildEffectiveContentItem(
	ctx context.Context,
	def AppConfigFileDef,
	envName string,
	group *effectiveFileGroup,
) (ConfigFileWithContent, error) {
	targetFile := group.defaultFile
	if group.envFile != nil {
		targetFile = group.envFile
	}

	editor, err := NewAppConfigFileEditor(s.FileStore, s.DefStore, targetFile)
	if err != nil {
		return ConfigFileWithContent{}, errors.Wrapf(err, "creating editor for def %s", def.ID.Hex())
	}
	content, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return ConfigFileWithContent{}, errors.Wrapf(err, "compiling content for def %s env %s", def.ID.Hex(), envName)
	}

	return ConfigFileWithContent{
		Name:     def.Name,
		MountDir: def.MountDir,
		Content:  content,
	}, nil
}

// GetEnvContent retrieves the selected config file and its compiled content with priority:
// 1. Environment-specific config (envName = current environment name)
// 2. Application-level default config (envName = "")
//
// Deprecated: 兼容 workload 层旧调用方（trpc/taf plugin），后续将由 AppConfigContentProvider 替代，本次暂时不处理
// todo 切换
func GetEnvContent(
	ctx context.Context,
	store AppConfigFileStore,
	defStore AppConfigFileDefStore,
	appID, envName string,
) (*AppConfigFile, string, string, error) {
	// 1. Try environment-specific config
	acf, content, err := getConfigFileAndCompiledContent(ctx, store, defStore, appID, envName)
	if err != nil && !errors.Is(err, ErrNoConfigFileFound) {
		return nil, "", "", errors.Wrapf(err, "getting env-specific config for app %s env %s", appID, envName)
	}

	// 2. Fall back to app-level default config
	if errors.Is(err, ErrNoConfigFileFound) {
		acf, content, err = getConfigFileAndCompiledContent(ctx, store, defStore, appID, EnvNameDefault)
		if err != nil {
			return nil, "", "", errors.Wrapf(err, "getting app-level config for app %s", appID)
		}
	}

	// 3. Resolve file name from def
	def, err := defStore.GetByID(ctx, acf.DefID)
	if err != nil {
		return nil, "", "", errors.Wrap(err, "loading def for name resolution")
	}
	return acf, def.Name, content, nil
}

// getConfigFileAndCompiledContent retrieves the config file for an app environment and compiles its content.
//
// Deprecated: 兼容 GetEnvContent，后续将由 AppConfigContentProvider 替代。
func getConfigFileAndCompiledContent(
	ctx context.Context,
	store AppConfigFileStore,
	defStore AppConfigFileDefStore,
	appID, envName string,
) (*AppConfigFile, string, error) {
	configFiles, err := store.List(ctx, appID, AcfFilterEnvName(envName))
	if err != nil {
		return nil, "", errors.Wrapf(err, "list config files for app %s env %s", appID, envName)
	}
	if len(configFiles) == 0 {
		return nil, "", ErrNoConfigFileFound
	}
	if len(configFiles) > 1 {
		return nil, "", errors.Errorf("multiple config files found for app %s env %s", appID, envName)
	}

	acf := &configFiles[0]
	editor, err := NewAppConfigFileEditor(store, defStore, acf)
	if err != nil {
		return nil, "", errors.Wrap(err, "creating app config file editor")
	}
	content, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "compiling app config file content")
	}
	return acf, content, nil
}
