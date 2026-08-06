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

// GetEnvContent retrieves the selected config file and its compiled content with priority:
// 1. Environment-specific config (envName = current environment name)
// 2. Application-level default config (envName = "")
//
// Returns:
// - The selected logical config file
// - The compiled content string
// - ErrNoConfigFileFound if no config file exists for both env and default
// - Other errors for query or compilation failures
func GetEnvContent(
	ctx context.Context,
	store AppConfigFileStore,
	appID, envName string,
) (*AppConfigFile, string, error) {
	// 1. Try environment-specific config
	acf, content, err := getConfigFileAndCompiledContent(ctx, store, appID, envName)
	if err == nil {
		return acf, content, nil
	}
	if !errors.Is(err, ErrNoConfigFileFound) {
		return nil, "", errors.Wrapf(err, "getting env-specific config for app %s env %s", appID, envName)
	}

	// 2. Fall back to app-level default config
	acf, content, err = getConfigFileAndCompiledContent(ctx, store, appID, EnvNameDefault)
	if err != nil {
		return nil, "", errors.Wrapf(err, "getting app-level config for app %s", appID)
	}
	return acf, content, nil
}

// getConfigFileAndCompiledContent retrieves the config file for an app environment and compiles its content.
func getConfigFileAndCompiledContent(
	ctx context.Context,
	store AppConfigFileStore,
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
