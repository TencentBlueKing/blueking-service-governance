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

// Package create provides the 'envvar create' sub-command group.
package create

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewEnvCmd returns a Command instance for 'envvar create for env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, key, value, description string
	var sensitive bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Create an environment-scoped environment variable",
		Long: `Create a new environment-scoped environment variable.

The variable will be scoped to a specific environment (scopeType=env).
The --env flag specifies the target environment name.`,
		Example: `  # Create an env-scoped env var
  bkms-cli envvar create env --env <env-name> --key MY_VAR --value my-value

  # Create a sensitive env-scoped env var
  bkms-cli envvar create env --env <env-name> --key MY_VAR --value my-value --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			key = strings.TrimSpace(key)
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			result, err := client.New().
				CreateScopedEnvVar(cmd.Context(), workspaceID, client.CreateScopedEnvVarOptions{
					ScopeType:   "env",
					ScopeValue:  envName,
					Key:         key,
					Value:       value,
					Description: description,
					IsSensitive: sensitive,
				})
			if err != nil {
				return errors.Wrap(err, "create env scoped env var")
			}

			console.Info("Created env var: key=%s, scopeType=env, scopeValue=%s, id=%s\n",
				result.Key, result.ScopeValue, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")

	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
