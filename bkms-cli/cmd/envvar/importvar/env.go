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

// Package importvar provides the 'envvar import' sub-command group.
package importvar

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewEnvCmd returns a Command instance for 'envvar import env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Import environment-scoped environment variables",
		Long: `Import environment-scoped environment variables from a local .env file.

The --env flag accepts an environment name (e.g. prod, stag, teamdev).
The file will be uploaded to the server and parsed as env-level scoped
environment variables for the specified environment.

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import env-scoped env vars
  bkms-cli envvar import env --env <env-name> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import env --env <env-name> -f vars.env --preview

  # Example .env file content for env import:
  # ─────────────────────────────────────────
  # # desc: overwrite existing env key
  # ENV_ONLY_KEY=updated-env-value
  #
  # # desc: add another env key
  # ANOTHER_ENV_KEY=another-value
  #
  # MY_CONFIG=some-config-value
  # ─────────────────────────────────────────
  # Note: env import does NOT allow scope metadata (scopeType/scopeValue).
  # The target scope is determined by the --env flag.`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envID, err := handler.ResolveEnvIDByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return err
			}

			if preview {
				previewResult, previewErr := client.New().PreviewEnvScopedEnvVars(cmd.Context(), envID, filePath)
				if previewErr != nil {
					return errors.Wrap(previewErr, "preview env scoped env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportEnvScopedEnvVars(cmd.Context(), envID, filePath)
			if err != nil {
				return errors.Wrap(err, "import env scoped env vars")
			}

			console.Info("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
