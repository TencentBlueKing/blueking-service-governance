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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListVersionsCmd returns a Command instance for 'app app-cfg-file list-versions' sub command.
func NewListVersionsCmd() *cobra.Command {
	var appID, envName, cfgFileName, outputFormat string

	cmd := &cobra.Command{
		Use:    "list-versions",
		Short:  "List all history versions of an application config file",
		PreRun: cmdutil.CommonPreRun,
		Long: `List all history versions of the application config file selected by app and environment.

When --env is omitted, this command lists versions of the default application-level config.
When --env is provided, this command lists versions of that environment's overlay config.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # List all versions of the default config file
  bkms-cli app app-cfg-file list-versions --app demo

  # List all versions of an environment-specific overlay config file
  bkms-cli app app-cfg-file list-versions --app demo --env prod

  # List all versions of one Helm config file by name
  bkms-cli app app-cfg-file list-versions --app demo --name values

  # Output in JSON format
  bkms-cli app app-cfg-file list-versions --app demo --env prod -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.ListVersions(cmd.Context(), client.New(), appID, envName, cfgFileName)
			if err != nil {
				return errors.Wrap(err, "list app config file versions")
			}

			listOutput, err := result.Output()
			if err != nil {
				return errors.Wrap(err, "format app config file versions")
			}
			formatted, err := output.FormatData(cmd.Context(), listOutput, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			console.Info("%s", formatted)
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
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
