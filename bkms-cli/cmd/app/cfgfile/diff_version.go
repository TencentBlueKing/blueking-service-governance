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

package cfgfile

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/cfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewDiffVersionCmd returns a Command instance for 'app config-file diff-version' sub command.
func NewDiffVersionCmd() *cobra.Command {
	var appID, envName, outputFormat string
	var ref handler.FileRef
	var from, to int64

	cmd := &cobra.Command{
		Use:   "diff-version",
		Short: "Compare two config file versions",
		Long: `Compare two versions of a config file and print a unified diff.
Without --to, the version given by --from is compared with the current version.
With --output, both versions are printed as structured data instead.

` + versionEnvNote,
		Example: `  # Compare version 3 with the current version
  bkms-cli app config-file diff-version --app demo --name app.conf --from 3

  # Compare two versions of the prod content
  bkms-cli app config-file diff-version --app demo --name app.conf --env prod --from 2 --to 4`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var toVersion *int64
			if cmd.Flags().Changed("to") {
				toVersion = &to
			}
			result, err := handler.DiffVersions(cmd.Context(), client.New(), appID, ref, envName, from, toVersion)
			if err != nil {
				return errors.Wrap(err, "diff app config file versions")
			}
			tipsSharesDefault(&result.VersionsResult, envName)

			if outputFormat != "" {
				return printData(cmd, result.Output(), outputFormat)
			}
			diff, err := result.UnifiedDiff()
			if err != nil {
				return err
			}
			if diff == "" {
				console.Info("No differences")
				return nil
			}
			fmt.Print(diff)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	cmd.Flags().Int64Var(&from, "from", 0, "old version number")
	cmd.Flags().Int64Var(&to, "to", 0, "new version number (default: current version)")
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}
