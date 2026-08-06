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

package appspec

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewViewCmd creates the view subcommand (query all sections at once).
func NewViewCmd() *cobra.Command {
	var appID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:    "view",
		Short:  "View all AppSpec sections",
		PreRun: cmdutil.CommonPreRun,
		Long: `View all AppSpec sections for an application.

When --env is not specified, shows the default configuration.
When --env is specified, shows the effective configuration for that environment.
Start command is always shown as global config regardless of --env.`,
		Example: `  # View all sections default config
  bkms-cli app appspec view --app my-app

  # View effective config for prod environment
  bkms-cli app appspec view --app my-app --env prod

  # Output in JSON format
  bkms-cli app appspec view --app my-app -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appspec.ViewAllHandler(cmd.Context(), appID, envName, outputFormat)
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (optional, omit for default config)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
