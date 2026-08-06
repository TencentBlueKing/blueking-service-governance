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

// RollbackVersionOptions contains rollback metadata.
type RollbackVersionOptions struct {
	// VersionRef identifies the target history version.
	VersionRef VersionRefOptions
	// Description is the rollback version description.
	Description string
}

// RollbackVersionResult contains the selected config file and rollback result.
type RollbackVersionResult struct {
	// File is the selected app config file metadata from the list API.
	File client.AppConfigFile
	// RolledBackFile is the app config file returned by rollback API.
	RolledBackFile *client.AppConfigFile
	// CurrentVersion is the current version before rollback.
	CurrentVersion int64
	// VersionID is the rollback target history version record ID.
	VersionID string
	// EnvName is the user-facing environment label used in output.
	EnvName string
}

// RollbackVersion rolls the selected config file back to one history version.
func RollbackVersion(
	ctx context.Context,
	cli client.Client,
	appID, envName, cfgFileName string,
	opts RollbackVersionOptions,
) (*RollbackVersionResult, error) {
	if err := validate.Struct(opts.VersionRef); err != nil {
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

	versionID, err := resolveVersionID(ctx, cli, appID, file.ID, opts.VersionRef)
	if err != nil {
		return nil, err
	}

	rolledBackFile, err := cli.RollbackAppConfigFileVersion(
		ctx,
		appID,
		versionID,
		client.RollbackAppConfigFileVersionOptions{
			CurrentVersion: &file.CurrentVersion,
			Description:    opts.Description,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "rollback app config file version")
	}
	if rolledBackFile == nil {
		return nil, errors.Errorf("empty rollback result for app config file version %s", versionID)
	}

	return &RollbackVersionResult{
		File:           file,
		RolledBackFile: rolledBackFile,
		CurrentVersion: file.CurrentVersion,
		VersionID:      versionID,
		EnvName:        formatEnvName(envName),
	}, nil
}
