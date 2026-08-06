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

// DeleteVersionResult contains the selected config file and deleted history version ID.
type DeleteVersionResult struct {
	// File is the selected app config file metadata from the list API.
	File client.AppConfigFile
	// VersionID is the deleted history version record ID.
	VersionID string
	// EnvName is the user-facing environment label used in output.
	EnvName string
}

// DeleteVersion deletes one history version of the config file selected by app and environment.
func DeleteVersion(
	ctx context.Context,
	cli client.Client,
	appID, envName, cfgFileName string,
	opts VersionRefOptions,
) (*DeleteVersionResult, error) {
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

	if err := cli.DeleteAppConfigFileVersion(ctx, appID, versionID); err != nil {
		return nil, errors.Wrap(err, "delete app config file version")
	}

	return &DeleteVersionResult{
		File:      file,
		VersionID: versionID,
		EnvName:   formatEnvName(envName),
	}, nil
}
