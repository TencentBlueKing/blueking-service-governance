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

// Package update provides the 'envvar update' sub-command group.
package update

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewEnvCmd returns a Command instance for 'envvar update env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, key, updatedKey, value, description string
	var sensitive, noSensitive bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Update an environment-scoped environment variable",
		Long: `Update an existing environment-scoped environment variable.

The --env flag specifies the target environment name.
The --key flag specifies the variable key to update.
The --updated-key flag allows renaming the variable key (optional, defaults to --key).
Use --sensitive to mark as sensitive, or --no-sensitive to unmark.
Only specified fields will be updated.`,
		Example: `  # Update value
  bkms-cli envvar update env --env <env-name> --key MY_VAR --value new-value

  # Rename key
  bkms-cli envvar update env --env <env-name> --key MY_VAR --updated-key NEW_KEY

  # Update description
  bkms-cli envvar update env --env <env-name> --key MY_VAR --description "New description"`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sensitive && noSensitive {
				return errors.New("--sensitive and --no-sensitive cannot be used together")
			}

			key = strings.TrimSpace(key)
			updatedKey = strings.TrimSpace(updatedKey)

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			cli := client.New()

			// 解析 env name → envID
			envID, err := envhandler.ResolveEnvIDByName(cmd.Context(), cli, workspaceID, envName)
			if err != nil {
				return err
			}

			// 解析 key → varID
			varID, err := envhandler.ResolveEnvScopedEnvVarID(cmd.Context(), cli, envID, key)
			if err != nil {
				return errors.Wrapf(err, "resolve env var '%s' in env '%s'", key, envName)
			}

			// --updated-key 不传时默认使用 --key 的值（不改名）
			effectiveKey := updatedKey
			if effectiveKey == "" {
				effectiveKey = key
			}

			opts := client.UpdateScopedEnvVarOptions{
				Key: effectiveKey,
			}
			if cmd.Flags().Changed("value") {
				opts.Value = &value
			}
			if cmd.Flags().Changed("description") {
				opts.Description = &description
			}
			if sensitive {
				t := true
				opts.IsSensitive = &t
			} else if noSensitive {
				f := false
				opts.IsSensitive = &f
			}

			result, err := cli.UpdateScopedEnvVar(cmd.Context(), workspaceID, varID, opts)
			if err != nil {
				return errors.Wrap(err, "update env scoped env var")
			}

			console.Info("Updated env var: key=%s, id=%s\n", result.Key, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&key, "key", "", "current environment variable key (required)")
	cmd.Flags().StringVar(&updatedKey, "updated-key", "", "new environment variable key (optional, defaults to --key)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")
	cmd.Flags().BoolVar(&noSensitive, "no-sensitive", false, "unmark sensitive variable")

	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
