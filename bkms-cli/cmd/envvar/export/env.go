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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewEnvCmd returns a Command instance for 'envvar export env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, filePath string

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Export environment-scoped environment variables",
		Long: `Export environment-scoped environment variables from the server.

The --env flag accepts an environment name (e.g. prod, stag, teamdev).
The exported content is in dotenv format. By default it is printed to stdout.
Use -f to write it to a file.`,
		Example: `  # Export env-scoped env vars to stdout
  bkms-cli envvar export env --env <env-name>

  # Export env-scoped env vars to a file
  bkms-cli envvar export env --env <env-name> -f vars.env`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envID, err := handler.ResolveEnvIDByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return err
			}

			content, err := client.New().ExportEnvScopedEnvVars(cmd.Context(), envID)
			if err != nil {
				return errors.Wrap(err, "export env scoped env vars")
			}

			return writeExportContent(content, filePath)
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "output file path (default: stdout)")

	_ = cmd.MarkFlagRequired("env")

	return cmd
}
