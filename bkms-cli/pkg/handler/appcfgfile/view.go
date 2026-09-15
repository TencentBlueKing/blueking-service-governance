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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// defaultEnvLabel 用于 CLI 展示默认配置，避免直接展示空环境名。
const defaultEnvLabel = "<default>"

// ViewResult contains the selected config file metadata and details.
type ViewResult struct {
	// File is the selected app config file metadata from the list API.
	File client.AppConfigFile
	// Details is the raw details response for the selected app config file.
	Details *client.AppConfigFileDetails
	// Content is the default app-level content from details.Content.
	Content *string
	// OverlayContent is the environment overlay content from details.OverlayContent.
	OverlayContent *string
	// EnvName is the user-facing environment label used in output.
	EnvName string
	// IsFallback indicates the displayed file is the default file, not the requested env's instance.
	IsFallback bool
}

// ViewOutput is the formatted output object for app config file view.
type ViewOutput struct {
	Name           string  `json:"name" yaml:"name"`
	ID             string  `json:"id" yaml:"id"`
	EnvName        string  `json:"envName" yaml:"envName"`
	Type           string  `json:"type" yaml:"type"`
	FileFormat     string  `json:"fileFormat" yaml:"fileFormat"`
	CurrentVersion int64   `json:"currentVersion" yaml:"currentVersion"`
	Updater        string  `json:"updater" yaml:"updater"`
	UpdatedAt      string  `json:"updatedAt" yaml:"updatedAt"`
	Content        *string `json:"content,omitempty" yaml:"content,omitempty" table:"-"`
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty" table:"-"`
}

// Output returns the structured output data for CLI formatting.
func (r *ViewResult) Output() (*ViewOutput, error) {
	if r == nil {
		return nil, errors.New("empty view result")
	}

	return &ViewOutput{
		Name:           r.File.Name,
		ID:             r.File.ID,
		EnvName:        r.EnvName,
		Type:           r.File.Type,
		FileFormat:     r.File.FileFormat,
		CurrentVersion: r.Details.CurrentVersion,
		Updater:        r.Details.Updater,
		UpdatedAt:      r.Details.UpdatedAt,
		Content:        r.Content,
		OverlayContent: r.OverlayContent,
	}, nil
}

// View returns the latest config file content selected by app and environment.
//
// Args:
// - envName: the environment name for filtering cfg file(s), "" means the default.
// - cfgFileName: optional, only select the cfg file with this name is given.
func View(ctx context.Context, cli client.Client, appID, envName, cfgFileName string) (*ViewResult, error) {
	files, err := cli.ListAppConfigFiles(ctx, appID, "")
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}

	file, isFallback, err := findViewFile(files, envName, cfgFileName)
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

	return &ViewResult{
		File:           file,
		Details:        details,
		Content:        details.Content,
		OverlayContent: details.OverlayContent,
		EnvName:        formatEnvName(file.EnvName),
		IsFallback:     isFallback,
	}, nil
}

// findCfgFileBy 根据 envName 与可选的 cfgFileName 唯一挑选一个配置文件，找不到或多于一个时返回错误。
func findCfgFileBy(files []client.AppConfigFile, envName, cfgFileName string) (client.AppConfigFile, error) {
	matches := findCfgFilesBy(files, envName, cfgFileName)
	if len(matches) == 0 {
		return client.AppConfigFile{}, errors.Errorf(
			"no app config file found for env %s%s",
			formatEnvName(envName),
			formatCfgFileNameSuffix(cfgFileName),
		)
	}
	if len(matches) > 1 {
		return client.AppConfigFile{}, errors.Errorf(
			"multiple app config files found for env %s%s: %s",
			formatEnvName(envName),
			formatCfgFileNameSuffix(cfgFileName),
			formatCandidates(matches),
		)
	}
	return matches[0], nil
}

// findCfgFilesBy 根据 envName 和可选的 cfgFileName 过滤配置文件。
// Helm 应用 EnvName 恒为空且可拥有多个文件，通过 cfgFileName 在应用级文件中消歧。
func findCfgFilesBy(files []client.AppConfigFile, envName, cfgFileName string) []client.AppConfigFile {
	return lo.Filter(files, func(file client.AppConfigFile, _ int) bool {
		if file.EnvName != envName {
			return false
		}
		return cfgFileName == "" || file.Name == cfgFileName
	})
}

// findViewFile 按环境获取配置文件，如果没有则展示默认配置文件
func findViewFile(
	files []client.AppConfigFile,
	envName, cfgFileName string,
) (client.AppConfigFile, bool, error) {
	if envName == "" {
		file, err := findCfgFileBy(files, "", cfgFileName)
		return file, false, err
	}
	matches := findCfgFilesBy(files, envName, cfgFileName)
	if len(matches) == 1 {
		return matches[0], false, nil
	}
	if len(matches) > 1 {
		return client.AppConfigFile{}, false, errors.Errorf(
			"multiple app config files found for env %s%s: %s",
			formatEnvName(envName),
			formatCfgFileNameSuffix(cfgFileName),
			formatCandidates(matches),
		)
	}
	file, err := findCfgFileBy(files, "", cfgFileName)
	return file, true, err
}

func formatEnvName(envName string) string {
	if envName == "" {
		return defaultEnvLabel
	}
	return envName
}

func formatCfgFileNameSuffix(cfgFileName string) string {
	if cfgFileName == "" {
		return ""
	}
	return fmt.Sprintf(" name %s", cfgFileName)
}

func formatCandidates(files []client.AppConfigFile) string {
	candidates := lo.Map(files, func(file client.AppConfigFile, _ int) string {
		return fmt.Sprintf("%s(%s)", file.Name, file.ID)
	})
	return strings.Join(candidates, ", ")
}
