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
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewEditCmd returns a Command instance for 'app app-cfg-file edit' sub command.
func NewEditCmd() *cobra.Command {
	var appID, envName, cfgFileName, filePath, fileContent, description string
	var viewCompiledContent bool

	cmd := &cobra.Command{
		Use:    "edit",
		Short:  "Edit application config file content",
		PreRun: cmdutil.CommonPreRun,
		Long: `Edit the application config file content selected by app and environment.

When --env is omitted, this command edits the default application-level config.
When --env is provided, this command edits that environment's config file.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # Edit default config file content
  bkms-cli app app-cfg-file edit --app demo -f values.yaml

  # Edit environment-specific overlay config file content
  bkms-cli app app-cfg-file edit --app demo --env prod -f values-prod.yaml

  # Edit one Helm config file by name when multiple files exist at app level
  bkms-cli app app-cfg-file edit --app demo --name values -f values.yaml

  # Edit config file content from a literal value
  bkms-cli app app-cfg-file edit --app demo --file-content $'server:\n  port: 8081\n'

  # Show compiled content after updating
  bkms-cli app app-cfg-file edit --app demo --env prod -f values-prod.yaml --view-compiled-content`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := resolveEditContent(cmd, filePath, fileContent)
			if err != nil {
				return err
			}

			result, err := handler.Edit(
				cmd.Context(),
				client.New(),
				appID,
				envName,
				cfgFileName,
				handler.EditOptions{
					Content:     content,
					Description: description,
				},
			)
			if err != nil {
				return errors.Wrap(err, "edit app config file")
			}

			if viewCompiledContent {
				fmt.Print(result.UpdateResult.CompiledContent)
				return nil
			}
			fmt.Printf("app config file %s updated successfully\n", result.File.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVar(
		&cfgFileName,
		"name",
		"",
		"config file name; useful for Helm apps with multiple app-level config files",
	)
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "config file content path")
	cmd.Flags().StringVar(&fileContent, "file-content", "", "config file content literal")
	cmd.Flags().StringVar(&description, "description", "", "version description")
	cmd.Flags().BoolVar(&viewCompiledContent, "view-compiled-content", false, "show compiled content after update")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// 根据输入的文件路径或者文件字面量来获取输入的配置文件内容。
func resolveEditContent(cmd *cobra.Command, filePath, fileContent string) (string, error) {
	fileChanged := cmd.Flags().Changed("file")
	fileContentChanged := cmd.Flags().Changed("file-content")
	if fileChanged && fileContentChanged {
		return "", errors.New("only one of --file or --file-content can be specified")
	}
	if !fileChanged && !fileContentChanged {
		return "", errors.New("one of --file or --file-content is required")
	}
	if fileContentChanged {
		// TODO: The current server API ignores empty content instead of clearing the file.
		// Add an explicit clear/delete flow once the API contract supports it.
		return fileContent, nil
	}
	if filePath == "" {
		return "", errors.New("--file cannot be empty")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrap(err, "read app config file")
	}
	// TODO: The current server API ignores empty content instead of clearing the file.
	// Add an explicit clear/delete flow once the API contract supports it.
	return string(content), nil
}
