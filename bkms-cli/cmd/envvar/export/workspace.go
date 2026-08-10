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

// Package export provides the 'envvar export' sub-command group.
package export

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewPublicCmd returns a Command instance for 'envvar export scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, filePath string

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "Export scoped (workspace/envType) environment variables",
		Long: `Export scoped environment variables from the server.

The exported content is in dotenv format. By default it is printed to stdout.
Use -f to write it to a file.`,
		Example: `  # Export scoped env vars to stdout
  bkms-cli envvar export scoped --workspace <workspaceID>

  # Export scoped env vars to a file
  bkms-cli envvar export scoped --workspace <workspaceID> -f vars.env

  # Use default workspace
  bkms-cli envvar export scoped -f vars.env`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			content, err := client.New().ExportPublicEnvVars(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "export scoped env vars")
			}

			return writeExportContent(content, filePath)
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "output file path (default: stdout)")

	return cmd
}

// writeExportContent 将导出内容写入文件或输出到 stdout。
func writeExportContent(content, filePath string) error {
	if filePath == "" {
		fmt.Print(content)
		return nil
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return errors.Wrap(err, "write export file")
	}
	console.Info("Exported to %s\n", filePath)
	return nil
}
