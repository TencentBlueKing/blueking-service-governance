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

// Package delete provides the 'envvar delete' sub-command group.
package delete

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewPublicCmd returns a Command instance for 'envvar delete scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, key, scope string

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "Delete a scoped (workspace/envType) environment variable",
		Long: `Delete an existing scoped environment variable by key and scope.

The --scope flag specifies the scope using the format "type[:value]":
  - (default)          : workspace-level variable
  - envType:<value>    : envType-level variable (e.g. envType:development)

If --scope is not specified, defaults to workspace level.

The --key flag specifies the variable key to delete.`,
		Example: `  # Delete a workspace-scoped env var (default scope)
  bkms-cli envvar delete scoped --key MY_VAR

  # Delete an envType-scoped env var
  bkms-cli envvar delete scoped --scope envType:development --key MY_VAR`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if err = cli.DeleteScopedEnvVar(cmd.Context(), workspaceID, varID); err != nil {
				return errors.Wrap(err, "delete scoped env var")
			}

			console.Info("Deleted env var: key=%s, scope=%s\n", key, scope)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&scope, "scope", "", "scope in format 'workspace' or 'envType:<value>' (default: workspace)")

	_ = cmd.MarkFlagRequired("key")

	return cmd
}
