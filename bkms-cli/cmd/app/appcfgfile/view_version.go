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

// NewViewVersionCmd returns a Command instance for 'app app-cfg-file view-version' sub command.
func NewViewVersionCmd() *cobra.Command {
	var appID, envName, cfgFileName, versionID, outputFormat string
	var version int64

	cmd := &cobra.Command{
		Use:    "view-version",
		Short:  "View one history version of an application config file",
		PreRun: cmdutil.CommonPreRun,
		Long: `View one history version of the application config file selected by app and environment.

Use exactly one of --version or --version-id to identify the target history version.
When --env is omitted, this command reads a version of the default application-level config.
When --env is provided, this command reads a version of that environment's overlay config.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # View version 7 of the default config file
  bkms-cli app app-cfg-file view-version --app demo --version 7

  # View one version by version record ID
  bkms-cli app app-cfg-file view-version --app demo --env prod --version-id <record-id>

  # View one Helm config file version by name
  bkms-cli app app-cfg-file view-version --app demo --name values --version 3

  # Output in JSON format
  bkms-cli app app-cfg-file view-version --app demo --env prod --version 7 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseVersionRefOptions(cmd, version, versionID)
			if err != nil {
				return err
			}

			result, err := handler.ViewVersion(cmd.Context(), client.New(), appID, envName, cfgFileName, opts)
			if err != nil {
				return errors.Wrap(err, "view app config file version")
			}

			viewOutput, err := result.Output()
			if err != nil {
				return errors.Wrap(err, "format app config file version")
			}
			formatted, err := output.FormatData(cmd.Context(), viewOutput, outputFormat)
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
	registerVersionRefFlags(cmd, &version, &versionID)
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
