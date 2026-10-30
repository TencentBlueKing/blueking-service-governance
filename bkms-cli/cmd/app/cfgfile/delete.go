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

// NewDeleteCmd returns a Command instance for 'app config-file delete' sub command.
func NewDeleteCmd() *cobra.Command {
	var appID string
	var ref handler.FileRef
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a config file",
		Long: `Delete a config file, together with the content of all its environments and all version history.
This operation cannot be undone and requires --yes.`,
		Example: `  bkms-cli app config-file delete --app demo --name app.conf --yes`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli := client.New()
			plan, err := handler.PlanDelete(cmd.Context(), cli, appID, ref)
			if err != nil {
				return errors.Wrap(err, "plan app config file deletion")
			}

			warnings := []string{fmt.Sprintf(
				"Config file %s (%s, id: %s) and all its version history will be deleted.",
				plan.Def.Name, plan.Def.ConfigKind, plan.Def.ID,
			)}
			if len(plan.EnvInstances) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"Content of envs [%s] will be deleted too.", joinEnvNames(plan.EnvInstances),
				))
			}
			if err = requireYes(yes, warnings...); err != nil {
				return err
			}

			if err = handler.ApplyDelete(cmd.Context(), cli, appID, plan); err != nil {
				return errors.Wrap(err, "delete app config file")
			}
			console.Info("✓ Config file %s deleted", plan.Def.Name)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm deletion")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}
