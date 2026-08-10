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

// NewPublicCmd returns a Command instance for 'envvar update scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, key, updatedKey, scope, value, description string
	var sensitive, noSensitive bool

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "Update a scoped (workspace/envType) environment variable",
		Long: `Update an existing scoped environment variable.

The --scope flag specifies the scope using the format "type[:value]":
  - (default)          : workspace-level variable
  - envType:<value>    : envType-level variable (e.g. envType:development)

If --scope is not specified, defaults to workspace level.

The --key flag specifies the variable key to update.
The --updated-key flag allows renaming the variable key (optional, defaults to --key).
Use --sensitive to mark as sensitive, or --no-sensitive to unmark.
Only specified fields will be updated.`,
		Example: `  # Update value (workspace scope, default)
  bkms-cli envvar update scoped --key MY_VAR --value new-value

  # Update value (envType scope)
  bkms-cli envvar update scoped --scope envType:development --key MY_VAR --value new-value

  # Rename key (workspace scope)
  bkms-cli envvar update scoped --key MY_VAR --updated-key NEW_KEY

  # Mark as sensitive
  bkms-cli envvar update scoped --key MY_VAR --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			key = strings.TrimSpace(key)
			updatedKey = strings.TrimSpace(updatedKey)

			if sensitive && noSensitive {
				return errors.New("--sensitive and --no-sensitive cannot be used together")
			}

			scopeType, scopeValue, err := envhandler.ParseScope(scope)
			if err != nil {
				return err
			}

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			cli := client.New()

			varID, err := envhandler.ResolveScopedEnvVarID(cmd.Context(), cli, workspaceID, key, scopeType, scopeValue)
			if err != nil {
				return err
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
				return errors.Wrap(err, "update scoped env var")
			}

			console.Info("Updated env var: key=%s, id=%s\n", result.Key, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "current environment variable key (required)")
	cmd.Flags().StringVar(&updatedKey, "updated-key", "", "new environment variable key (optional, defaults to --key)")
	cmd.Flags().StringVar(&scope, "scope", "", "scope in format 'workspace' or 'envType:<value>' (default: workspace)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")
	cmd.Flags().BoolVar(&noSensitive, "no-sensitive", false, "unmark sensitive variable")

	_ = cmd.MarkFlagRequired("key")

	return cmd
}
