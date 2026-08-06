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

// Package deploy provides deploy list command
package deploy

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'app deploy list' sub command
func NewListCmd() *cobra.Command {
	var appID, envName, trafficLaneName, keyword, outputFormat, workspaceID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List application deploy records",
		Long: `List recent deploy records for an application.

This command retrieves the most recent deploy records (up to 10) for the
specified application. You can filter results using keywords.

The --env flag supports multiple environment names separated by commas (e.g. --env prod,staging).
When multiple environments are specified, records will be retrieved for each environment
and grouped by environment name. Environment names are validated against the workspace.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional. Otherwise, you must specify it explicitly.`,
		Example: `  # List all deploy records for an application
  bkms-cli app deploy list --app demo --env prod

  # Filter deploy records by traffic lane name
  bkms-cli app deploy list --app demo --env prod --trafficLane canary

  # Filter deploy records by keyword
  bkms-cli app deploy list --app demo --env prod --keyword main

  # Specify workspace explicitly
  bkms-cli app deploy list --workspace ws-demo --app demo --env prod

  # Output in JSON format
  bkms-cli app deploy list --app demo --env prod -o json

  # List deploy records for multiple environments
  bkms-cli app deploy list --app demo --env prod,staging,test`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			records, err := deploy.ListDeploy(cmd.Context(), workspaceID, appID, envName, trafficLaneName, keyword)
			if records != nil {
				formatted, formatErr := output.FormatData(cmd.Context(), records, outputFormat)
				if formatErr != nil {
					return errors.Wrap(formatErr, "format output")
				}
				fmt.Println(formatted)
			}
			if err != nil {
				return errors.Wrap(err, "list deploy records")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVar(&trafficLaneName, "trafficLane", "", "traffic lane name")
	cmd.Flags().StringVar(&keyword, "keyword", "", "filter by keyword")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
