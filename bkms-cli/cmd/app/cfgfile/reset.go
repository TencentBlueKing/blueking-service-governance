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
)

// NewResetCmd returns a Command instance for 'app config-file reset' sub command.
func NewResetCmd() *cobra.Command {
	var appID, envName string
	var ref handler.FileRef
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset an environment to the default content",
		Long: `Delete the content of one environment, so that it uses the default content again.
The environment's version history is deleted too. This operation requires --yes.`,
		Example: `  bkms-cli app config-file reset --app demo --name app.conf --env prod --yes`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli := client.New()
			plan, err := handler.PlanReset(cmd.Context(), cli, appID, ref, envName)
			if err != nil {
				return errors.Wrap(err, "plan app config file env reset")
			}
			if !plan.HasEnvInstance {
				console.Info("Env %s of config file %s already uses the default content", envName, plan.Def.Name)
				return nil
			}
			if err = requireYes(yes, fmt.Sprintf(
				"Content of env %s in config file %s will be deleted, including version history.",
				envName, plan.Def.Name,
			)); err != nil {
				return err
			}

			if err = handler.ApplyReset(cmd.Context(), cli, appID, plan); err != nil {
				return errors.Wrap(err, "reset app config file env")
			}
			console.Info("✓ Env %s of config file %s now uses the default content", envName, plan.Def.Name)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm deleting the environment content")
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
