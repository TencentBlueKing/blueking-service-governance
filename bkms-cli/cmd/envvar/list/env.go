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

// Package list provides the 'envvar list' sub-command group.
package list

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewEnvCmd returns a Command instance for 'envvar list env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "env",
		Short: "List environment-scoped environment variables",
		Long: `List all environment-scoped environment variables for a specific environment.

The --env flag accepts an environment name (e.g. prod, stag, teamdev).
Displays detailed information including conflict info.`,
		Example: `  # List env-scoped env vars
  bkms-cli envvar list env --env <env-name>

  # Output as JSON
  bkms-cli envvar list env --env <env-name> -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envID, err := handler.ResolveEnvIDByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return err
			}

			envVars, err := client.New().ListEnvScopedEnvVars(cmd.Context(), envID)
			if err != nil {
				return errors.Wrap(err, "list env scoped env vars")
			}

			// json/yaml 保持原始嵌套结构输出；table（默认）使用扁平结构展示
			var data any
			switch outputFormat {
			case "json", "yaml":
				data = envVars
			default:
				data = handler.ToTableRows(envVars)
			}

			formatted, err := output.FormatData(cmd.Context(), data, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("env")

	return cmd
}
