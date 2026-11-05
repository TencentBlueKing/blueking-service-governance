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

// Package cfgfile provides app config file commands based on config file definitions.
package cfgfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/cfgfile"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewCmd returns a Command instance for 'app config-file' command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config-file",
		Short: "Manage application config files",
		Long: `Manage config files mounted into application containers.

There are two kinds of config files:
  - framework: the framework config file (Helm values, tRPC/TAF config), created with the application.
  - plain:     user-defined text files mounted to a custom directory (trpc/taf/bkmsApp apps only).

A config file starts in unified mode, where all environments share the default content.
Editing it with --env switches it to per-env mode, and that environment gets its own content.`,
		DisableFlagsInUseLine: true,
	}

	// 文件定义
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewDeleteCmd())
	// 内容
	cmd.AddCommand(NewViewCmd())
	cmd.AddCommand(NewEditCmd())
	cmd.AddCommand(NewResetCmd())
	cmd.AddCommand(NewListEnvInstancesCmd())
	cmd.AddCommand(NewMountPreviewCmd())
	// 版本
	cmd.AddCommand(NewListVersionsCmd())
	cmd.AddCommand(NewViewVersionCmd())
	cmd.AddCommand(NewDiffVersionCmd())
	cmd.AddCommand(NewRollbackVersionCmd())
	cmd.AddCommand(NewDeleteVersionCmd())

	return cmd
}

// addFileRefFlags 注册定位配置文件的参数：--name（可配合 --mount-dir）或 --id。
func addFileRefFlags(cmd *cobra.Command, ref *handler.FileRef) {
	cmd.Flags().StringVar(&ref.Name, "name", "", "config file name")
	cmd.Flags().StringVar(&ref.MountDir, "mount-dir", "", "mount directory, to select among files with the same name")
	cmd.Flags().StringVar(&ref.ID, "id", "", "config file ID")
	cmd.MarkFlagsOneRequired("name", "id")
	cmd.MarkFlagsMutuallyExclusive("name", "id")
	cmd.MarkFlagsMutuallyExclusive("mount-dir", "id")
}

// addEnvFlag 注册 --env 参数，不指定时操作默认文件。
func addEnvFlag(cmd *cobra.Command, envName *string) {
	cmd.Flags().StringVar(envName, "env", "", "environment name; omit to use the default file")
}

// addContentFlags 注册内容输入参数，--file 与 --file-content 互斥。
func addContentFlags(cmd *cobra.Command, filePath, fileContent *string) {
	cmd.Flags().StringVarP(filePath, "file", "f", "", "path of the file to read content from")
	cmd.Flags().StringVar(fileContent, "file-content", "", "content literal")
	cmd.MarkFlagsMutuallyExclusive("file", "file-content")
}

// readContent 读取 --file 或 --file-content 指定的内容，provided 表示用户是否提供了内容。
func readContent(cmd *cobra.Command, filePath, fileContent string) (content string, provided bool, err error) {
	if cmd.Flags().Changed("file-content") {
		return fileContent, true, nil
	}
	if !cmd.Flags().Changed("file") {
		return "", false, nil
	}
	if filePath == "" {
		return "", false, clierr.Usagef("--file cannot be empty")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false, errors.Wrap(err, "read content file")
	}
	return string(data), true, nil
}

// requireYes 在执行会删除数据的操作前要求 --yes 确认，未确认时只展示影响并退出。
func requireYes(yes bool, warnings ...string) error {
	for _, w := range warnings {
		console.Warn("%s", w)
	}
	if yes {
		return nil
	}
	return clierr.Usagef("this operation deletes data and cannot be undone, re-run with --yes to confirm")
}

// printData 按 --output 格式输出数据。
func printData(cmd *cobra.Command, data any, format string) error {
	formatted, err := output.FormatData(cmd.Context(), data, format)
	if err != nil {
		return errors.Wrap(err, "format output")
	}
	fmt.Println(formatted)
	return nil
}

func formatEnvLabel(envName string) string {
	if envName == "" {
		return "<default>"
	}
	return envName
}

func joinEnvNames(envNames []string) string {
	return strings.Join(envNames, ", ")
}
