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

package updatestrategy

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewResetCmd returns a Command instance for 'appspec update-strategy reset' sub command.
func NewResetCmd() *cobra.Command {
	var appID, envName string

	cmd := &cobra.Command{
		Use:    "reset",
		Short:  "Reset update-strategy env override to default",
		PreRun: cmdutil.CommonPreRun,
		Long: `Reset the environment-specific update strategy override back to the default configuration.

This command removes the environment overlay so that the environment inherits
the default application-level update strategy. The --env flag is required.`,
		Example: `  # Reset env override to default
  bkms-cli app appspec update-strategy reset --app my-app --env prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if envName == "" {
				return errors.New("reset requires --env to be specified")
			}

			if err := appspec.ResetHandler(cmd.Context(), appID, envName, client.AppSpecSectionUpdateStrategy); err != nil {
				return errors.Wrap(err, "reset update-strategy")
			}

			fmt.Printf("Successfully reset update-strategy for app %s in env %s to default\n", appID, envName)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required for reset)")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
