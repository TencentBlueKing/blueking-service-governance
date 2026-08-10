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
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewPublicCmd returns a Command instance for 'envvar import scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "Import scoped (workspace/envType) environment variables",
		Long: `Import scoped environment variables from a local .env file.

The file will be uploaded to the server and parsed as workspace-level
and/or envType-level scoped environment variables.

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import scoped env vars from a file
  bkms-cli envvar import scoped --workspace <workspaceID> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import scoped --workspace <workspaceID> -f vars.env --preview

  # Preview with JSON output
  bkms-cli envvar import scoped -f vars.env --preview -o json

  # Example .env file content for scoped (workspace/envType) import:
  # ─────────────────────────────────────────
  # # desc: workspace key shared across all envs
  # # scopeType: workspace
  # WORKSPACE_KEY=workspace-value
  #
  # # desc: override for development env type
  # # scopeType: envType
  # # scopeValue: development
  # SHARED_KEY=dev-override
  #
  # # scopeType: workspace
  # ANOTHER_GLOBAL=global-value
  # ─────────────────────────────────────────
  # Note: scoped import REQUIRES scopeType metadata for each variable.
  # Supported scopeType values: workspace, envType.
  # scopeValue is required when scopeType is envType.`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			if preview {
				previewResult, err := client.New().PreviewPublicEnvVars(cmd.Context(), workspaceID, filePath)
				if err != nil {
					return errors.Wrap(err, "preview scoped env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportPublicEnvVars(cmd.Context(), workspaceID, filePath)
			if err != nil {
				return errors.Wrap(err, "import scoped env vars")
			}

			console.Info("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("file")

	return cmd
}
