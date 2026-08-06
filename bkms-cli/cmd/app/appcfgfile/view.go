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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewViewCmd returns a Command instance for 'app app-cfg-file view' sub command.
func NewViewCmd() *cobra.Command {
	var appID, envName, cfgFileName, outputFormat string

	cmd := &cobra.Command{
		Use:    "view",
		Short:  "View application config file content",
		PreRun: cmdutil.CommonPreRun,
		Long: `View the latest application config file content selected by app and environment.

When --env is omitted, this command views the default application-level config.
When --env is provided, this command views that environment's overlay config.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # View default config file content
  bkms-cli app app-cfg-file view --app demo

  # View environment-specific overlay config file content
  bkms-cli app app-cfg-file view --app demo --env prod

  # View one Helm config file by name when multiple files exist at app level
  bkms-cli app app-cfg-file view --app demo --name values

  # Output in JSON format, including the selected config content
  bkms-cli app app-cfg-file view --app demo --env prod -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.View(cmd.Context(), client.New(), appID, envName, cfgFileName)
			if err != nil {
				return errors.Wrap(err, "view app config file")
			}

			viewOutput, err := result.Output()
			if err != nil {
				return errors.Wrap(err, "format app config file")
			}
			formatted, err := output.FormatData(cmd.Context(), viewOutput, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
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
