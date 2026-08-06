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

package appcfgfile

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ViewVersionResult contains the selected config file and the requested history version.
type ViewVersionResult struct {
	// File is the selected app config file metadata from the list API.
	File client.AppConfigFile
	// Version is the requested history version details.
	Version *client.AppConfigFileVersion
	// EnvName is the user-facing environment label used in output.
	EnvName string
}

// Output returns the structured output data for CLI formatting.
func (r *ViewVersionResult) Output() (*VersionOutput, error) {
	if r == nil {
		return nil, errors.New("empty view version result")
	}
	if r.Version == nil {
		return nil, errors.New("empty app config file version")
	}

	output, err := toVersionOutput(*r.Version)
	if err != nil {
		return nil, err
	}
	output.EnvName = r.EnvName
	return &output, nil
}

// ViewVersion returns one history version details for the config file selected by app and environment.
func ViewVersion(
	ctx context.Context,
	cli client.Client,
	appID, envName, cfgFileName string,
	opts VersionRefOptions,
) (*ViewVersionResult, error) {
	if err := validate.Struct(opts); err != nil {
		return nil, errors.New("exactly one of versionID or version must be specified")
	}

	files, err := cli.ListAppConfigFiles(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}

	file, err := findCfgFileBy(files, envName, cfgFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "find app config file for app %s", appID)
	}

	versionID, err := resolveVersionID(ctx, cli, appID, file.ID, opts)
	if err != nil {
		return nil, err
	}

	version, err := cli.GetAppConfigFileVersion(ctx, appID, versionID)
	if err != nil {
		return nil, errors.Wrap(err, "get app config file version")
	}
	if version == nil {
		return nil, errors.Errorf("empty app config file version %s", versionID)
	}
	return &ViewVersionResult{
		File:    file,
		Version: version,
		EnvName: formatEnvName(envName),
	}, nil
}
