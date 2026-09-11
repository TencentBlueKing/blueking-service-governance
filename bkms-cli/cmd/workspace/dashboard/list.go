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

package dashboard

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'workspace dashboard list' sub command
func NewListCmd() *cobra.Command {
	var workspaceID, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dashboard directory tree",
		Long: `List the dashboard directory tree in a workspace.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional.`,
		Example: `  # List dashboard directory tree (default workspace)
  bkms-cli workspace dashboard list

  # List dashboard directory tree in a specific workspace
  bkms-cli workspace dashboard list --workspace ws-demo

  # Output in JSON format
  bkms-cli workspace dashboard list -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			tree, err := client.New().ListDashboardDirectoryTree(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "list dashboard directory tree")
			}

			formatted, err := output.FormatData(cmd.Context(), tree, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			console.Info("%s", formatted)
			return nil
		},
	}

	cmdutil.AddWorkspaceFlag(cmd, &workspaceID)
	output.AddFormatFlag(cmd, &outputFormat)

	return cmd
}
