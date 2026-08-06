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

package resources

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewViewCmd returns a Command instance for 'appspec resources view' sub command.
func NewViewCmd() *cobra.Command {
	var appID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:    "view",
		Short:  "View resources configuration",
		PreRun: cmdutil.CommonPreRun,
		Long: `View the resource specifications for the application.

When --env is omitted, this command views the default application-level resource config.
When --env is provided, this command views the effective resource config for that environment.`,
		Example: `  # View default resources config
  bkms-cli app appspec resources view --app my-app

  # View effective resources config for an environment
  bkms-cli app appspec resources view --app my-app --env prod

  # Output in JSON format
  bkms-cli app appspec resources view --app my-app -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appspec.ViewHandler(cmd.Context(), appID, envName, client.AppSpecSectionResources, outputFormat)
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (optional, omit for default config)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
