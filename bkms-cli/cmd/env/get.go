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

package env

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewGetCmd returns a Command instance for 'env get' sub command
func NewGetCmd() *cobra.Command {
	var workspaceID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get environment details",
		Long:  `Get the full details of an environment by its name.`,
		Example: `  # Get env details in default table format
  bkms-cli env get --workspace ws1 --env staging

  # Get env details as YAML
  bkms-cli env get --workspace ws1 --env staging -o yaml`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			cli := client.New()
			envID, err := envhandler.ResolveEnvIDByName(cmd.Context(), cli, workspaceID, envName)
			if err != nil {
				return errors.Wrap(err, "get env")
			}
			env, err := cli.GetEnv(cmd.Context(), envID)
			if err != nil {
				return errors.Wrap(err, "get env")
			}

			if outputFormat != "" {
				formatted, fmtErr := output.FormatData(cmd.Context(), env, outputFormat)
				if fmtErr != nil {
					return errors.Wrap(fmtErr, "format output")
				}
				console.Info("%s", formatted)
				return nil
			}

			printEnvDetails(env)
			return nil
		},
	}

	cmdutil.AddWorkspaceFlag(cmd, &workspaceID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	output.AddFormatFlag(cmd, &outputFormat)

	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func printEnvDetails(env *client.Env) {
	console.Info("  %-20s %s", "ID:", env.ID)
	console.Info("  %-20s %s", "Name:", env.Name)
	console.Info("  %-20s %s", "DisplayName:", env.DisplayName)
	console.Info("  %-20s %s", "Type:", env.Type)
	if env.Description != "" {
		console.Info("  %-20s %s", "Description:", env.Description)
	}
	if env.Cluster != nil {
		console.Info("  %-20s %s", "ClusterID:", env.Cluster.ClusterID)
		console.Info("  %-20s %s", "ClusterType:", env.Cluster.ClusterType)
		console.Info("  %-20s %s", "Namespace:", env.Cluster.Namespace)
	}
	if env.UpdatedAt != "" {
		console.Info("  %-20s %s", "UpdatedAt:", env.UpdatedAt)
	}
}
