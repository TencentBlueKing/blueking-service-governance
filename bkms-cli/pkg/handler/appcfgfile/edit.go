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

const (
	appConfigFileTypeNormal  = "normal"
	appConfigFileTypeOverlay = "overlay"

	editableContentFieldNone           = "none"
	editableContentFieldContent        = "content"
	editableContentFieldOverlayContent = "overlayContent"
)

// EditOptions contains the new content and metadata for updating an app config file.
type EditOptions struct {
	// Content is the new raw content provided by the user.
	Content string
	// Description is the version description recorded by the server.
	Description string
}

// EditResult contains the selected config file and update response.
type EditResult struct {
	// File is the selected app config file metadata from the list API.
	File client.AppConfigFile
	// Details is the raw details response read before updating the file.
	Details *client.AppConfigFileDetails
	// UpdateResult is the raw update response returned by the server.
	UpdateResult *client.AppConfigFileContentUpdateResult
	// EnvName is the user-facing environment label used in output.
	EnvName string
}

// Edit updates the app config file content selected by app and environment.
func Edit(
	ctx context.Context,
	cli client.Client,
	appID, envName, cfgFileName string,
	opts EditOptions,
) (*EditResult, error) {
	files, err := cli.ListAppConfigFiles(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}

	file, err := findCfgFileBy(files, envName, cfgFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "find app config file for app %s", appID)
	}

	details, err := cli.GetAppConfigFileDetails(ctx, appID, file.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get app config file details")
	}
	if details == nil {
		return nil, errors.Errorf("empty details for app config file %s", file.ID)
	}

	updateOpts := client.AppConfigFileContentOptions{
		Content:        opts.Content,
		Description:    opts.Description,
		CurrentVersion: &details.CurrentVersion,
	}
	updateResult, err := updateByEditableContentField(
		ctx,
		cli,
		appID,
		file.ID,
		details.EditableContentField,
		updateOpts,
	)
	if err != nil {
		return nil, err
	}
	if updateResult == nil {
		return nil, errors.Errorf("empty update result for app config file %s", file.ID)
	}

	return &EditResult{
		File:         file,
		Details:      details,
		UpdateResult: updateResult,
		EnvName:      formatEnvName(envName),
	}, nil
}

func updateByEditableContentField(
	ctx context.Context,
	cli client.Client,
	appID, fileID, editableContentField string,
	opts client.AppConfigFileContentOptions,
) (*client.AppConfigFileContentUpdateResult, error) {
	switch editableContentField {
	case editableContentFieldContent:
		updateResult, err := cli.UpdateAppConfigFileContent(ctx, appID, fileID, opts)
		if err != nil {
			return nil, errors.Wrap(err, "update app config file content")
		}
		return updateResult, nil
	case editableContentFieldOverlayContent:
		updateResult, err := cli.UpdateAppConfigFileOverlayContent(ctx, appID, fileID, opts)
		if err != nil {
			return nil, errors.Wrap(err, "update app config file overlay content")
		}
		return updateResult, nil
	case editableContentFieldNone:
		return nil, errors.Errorf("app config file %s is not editable", fileID)
	default:
		return nil, errors.Errorf(
			"unsupported editable content field %s for app config file %s",
			editableContentField,
			fileID,
		)
	}
}
