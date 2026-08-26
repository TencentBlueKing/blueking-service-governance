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
)

// ErrNoConfigFileFound is returned when no config file is found for the given criteria.
var ErrNoConfigFileFound = errors.New("no config file found")

// ConfigFileWithContent 将一个配置文件记录与其编译后的最终内容匹配
type ConfigFileWithContent struct {
	File    AppConfigFile
	Content string
}

// GetFrameworkEnvContent retrieves the framework config file selected for an app
// environment with priority:
// 1. Environment-specific framework config (envName = current environment name)
// 2. Application-level default framework config (envName = "")
//
// Returns ErrNoConfigFileFound if no framework config file exists.
func GetFrameworkEnvContent(
	ctx context.Context,
	store AppConfigFileStore,
	appID, envName string,
) (*AppConfigFile, string, error) {
	defaultAcf, defaultContent, defaultErr := getFrameworkConfigFileAndCompiledContent(
		ctx, store, appID, EnvNameDefault,
	)
	// 1. Try environment-specific config
	acf, content, err := getFrameworkConfigFileAndCompiledContent(ctx, store, appID, envName)
	if err == nil {
		return acf, content, nil
	}
	if !errors.Is(err, ErrNoConfigFileFound) {
		return nil, "", errors.Wrapf(err, "getting env-specific config for app %s env %s", appID, envName)
	}

	// 2. Fall back to app-level default config
	if defaultErr == nil {
		return defaultAcf, defaultContent, nil
	}
	if !errors.Is(defaultErr, ErrNoConfigFileFound) {
		return nil, "", errors.Wrapf(defaultErr, "getting app-level config for app %s", appID)
	}
	return nil, "", errors.Wrapf(ErrNoConfigFileFound, "getting app-level config for app %s", appID)
}

// getFrameworkConfigFileAndCompiledContent retrieves the framework config file
// for an app environment and compiles its content.
func getFrameworkConfigFileAndCompiledContent(
	ctx context.Context,
	store AppConfigFileStore,
	appID, envName string,
) (*AppConfigFile, string, error) {
	configFiles, err := store.List(
		ctx,
		appID,
		AcfFilterEnvName(envName),
		AcfFilterConfigKind(ConfigKindFramework),
	)
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
	editor, err := NewAppConfigFileEditor(store, acf)
	if err != nil {
		return nil, "", errors.Wrap(err, "creating app config file editor")
	}
	content, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "compiling app config file content")
	}
	return acf, content, nil
}

// ListEnvPlainContents 列出指定应用在目标环境下实际生效的 plain 配置文件内容。
// 整体流程分 3 步：
// 1. 先从应用全部配置文件里筛出 plain root 与各环境实例；
// 2. 再根据当前 envName 判断每个 plain root 最终应选用哪条记录生效；
// 3. 最后编译出这些生效记录的最终内容并返回。
func ListEnvPlainContents(
	ctx context.Context,
	store AppConfigFileStore,
	appID, envName string,
) ([]ConfigFileWithContent, error) {
	configFiles, err := store.List(ctx, appID, AcfFilterConfigKind(ConfigKindPlain))
	if err != nil {
		return nil, errors.Wrapf(err, "list config files for app %s", appID)
	}

	// 1. 先把 plain 默认记录和环境实例分开整理，方便后面按 root + envName 选中真正生效的记录。
	plainRoots := make([]AppConfigFile, 0)
	envFilesByRoot := make(map[string]map[string]AppConfigFile)
	for _, item := range configFiles {
		// 默认环境名为空的记录代表 plain 逻辑根文件。
		if item.EnvName == EnvNameDefault {
			plainRoots = append(plainRoots, item)
			continue
		}
		// 跳过default为空的单环境plain文件，避免脏数据
		if item.DefaultAppConfigFileID == nil {
			continue
		}
		rootID := item.DefaultAppConfigFileID.Hex()
		if envFilesByRoot[rootID] == nil {
			envFilesByRoot[rootID] = make(map[string]AppConfigFile)
		}
		envFilesByRoot[rootID][item.EnvName] = item
	}

	// 2. 逐个 root 判断当前环境下应该使用 root 自身，还是某条环境实例记录。
	result := make([]ConfigFileWithContent, 0)
	for _, root := range plainRoots {
		if !root.IsMountedToEnv(envName) {
			continue
		}
		target := root
		if root.HasIndependentEnvConfig() {
			// 引用模型：有独立 env instance 则使用，否则回退到默认记录内容（引用状态）。
			envFiles := envFilesByRoot[root.ID.Hex()]
			if envFile, ok := envFiles[envName]; ok {
				target = envFile
			}
		}

		// 3. 对选中的生效记录统一走编译流程，得到最终应返回给调用方的 plain 文件内容。
		// 当前 plain 文件只支持本地内容来源，这里主要是复用编译流程与内容读取入口。
		editor, editorErr := NewAppConfigFileEditor(store, &target)
		if editorErr != nil {
			return nil, errors.Wrap(editorErr, "creating app config file editor")
		}
		content, compileErr := editor.GetCompiledContent(ctx)
		if compileErr != nil {
			return nil, errors.Wrap(compileErr, "compiling app config file content")
		}
		result = append(result, ConfigFileWithContent{
			File:    target,
			Content: content,
		})
	}
	return result, nil
}
