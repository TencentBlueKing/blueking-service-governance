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

// NewAppCmd returns a Command instance for 'envvar import app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Import app-defined environment variables",
		Long: `Import app-defined environment variables from a local .env file.

The file will be uploaded to the server and parsed as application-level
environment variables (workload.envVars).

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import app env vars
  bkms-cli envvar import app --app <appID> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import app --app <appID> -f vars.env --preview

  # Example .env file content for app import:
  # ─────────────────────────────────────────
  # # desc: Application log level
  # LOG_LEVEL=info
  #
  # # desc: Feature toggle for new UI
  # ENABLE_NEW_UI=true
  #
  # APP_TIMEOUT=30s
  # ─────────────────────────────────────────
  # Note: app import does NOT allow scope metadata (scopeType/scopeValue).`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if preview {
				previewResult, err := client.New().PreviewAppEnvVars(cmd.Context(), appID, filePath)
				if err != nil {
					return errors.Wrap(err, "preview app env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportAppEnvVars(cmd.Context(), appID, filePath)
			if err != nil {
				return errors.Wrap(err, "import app env vars")
			}

			console.Info("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
