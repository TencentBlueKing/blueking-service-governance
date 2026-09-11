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

package app

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewGetCmd returns a Command instance for 'app get' sub command
func NewGetCmd() *cobra.Command {
	var appID, outputFormat string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get application details",
		Long: `Get the full definition of an application, including build config and app model spec.

The output in YAML format is compatible with 'app create -f', enabling a read-modify-write workflow.`,
		Example: `  # Get app details in default table format
  bkms-cli app get --app myapp

  # Get app details in YAML (can be saved and used with 'app create -f')
  bkms-cli app get --app myapp -o yaml

  # Get app details in JSON
  bkms-cli app get --app myapp -o json`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := client.New().GetApp(cmd.Context(), appID)
			if err != nil {
				return errors.Wrap(err, "get app")
			}

			if outputFormat != "" {
				formatted, fmtErr := output.FormatData(cmd.Context(), app, outputFormat)
				if fmtErr != nil {
					return errors.Wrap(fmtErr, "format output")
				}
				console.Info("%s", formatted)
				return nil
			}

			printAppDetails(app)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	output.AddFormatFlag(cmd, &outputFormat)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}

func printAppDetails(app *client.AppFull) {
	console.Info("  %-20s %s", "ID:", app.ID)
	console.Info("  %-20s %s", "Name:", app.Name)
	console.Info("  %-20s %s", "DisplayName:", app.DisplayName)
	console.Info("  %-20s %s", "Type:", app.Type)
	if app.BuildConfig != nil {
		console.Info("  %-20s %s", "BuildSource:", formatBuildSource(app.BuildConfig))
	}
	if app.AppModelSpec != nil {
		cmd := strings.Join(append(app.AppModelSpec.Command, app.AppModelSpec.Args...), " ")
		console.Info("  %-20s %s", "Command:", cmd)
	}
}

func formatBuildSource(bc *client.BuildConfig) string {
	if bc == nil {
		return "-"
	}
	switch bc.SourceType {
	case "codeRepository":
		if bc.RepoBuildConfig != nil {
			return "codeRepository (" + bc.RepoBuildConfig.Type + ": " + bc.RepoBuildConfig.RepoURL + ")"
		}
	case "imageRegistry":
		if bc.ImageBuildConfig != nil {
			return "imageRegistry (" + bc.ImageBuildConfig.Name + ")"
		}
	case "pipeline":
		if bc.PipelineBuildConfig != nil {
			return "pipeline (" + bc.PipelineBuildConfig.PipelineID + ")"
		}
	}
	return bc.SourceType
}
