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

// defaultEnvLabel is used only for CLI display and error messages.
// 这个值主要是为了更好的去表达“默认配置”，避免直接对外展示 env: "" 这种。
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

	return &ViewResult{
		File:           file,
		Details:        details,
		Content:        details.Content,
		OverlayContent: details.OverlayContent,
		EnvName:        formatEnvName(envName),
	}, nil
}

// findCfgFileBy 根据给定的条件去**明确挑选**一个 app cfg 配置文件对象，以方便进行后续操作。该函数设计较为灵活，支持
// 多种过滤方式，并且互相组合，其中：
//
// - envName：匹配特定环境比如 test，默认情况下仅匹配默认环境（""）
// - cfgFileName：匹配特定的配置文件名，环境下存在多个文件时必须提供，否则会因为无法确定是哪一个而报错
//
// 错误返回：找不到匹配，匹配的文件数量超过一个。
func findCfgFileBy(files []client.AppConfigFile, envName, cfgFileName string) (client.AppConfigFile, error) {
	// Helm apps keep EnvName empty and may own multiple config files. The optional
	// cfgFileName selector lets callers disambiguate those app-level files by name.
	matches := lo.Filter(files, func(file client.AppConfigFile, _ int) bool {
		if file.EnvName != envName {
			return false
		}
		return cfgFileName == "" || file.Name == cfgFileName
	})
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
