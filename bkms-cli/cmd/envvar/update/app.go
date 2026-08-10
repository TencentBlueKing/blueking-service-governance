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
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewAppCmd returns a Command instance for 'envvar update app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, key, updatedKey, value, description string
	var sensitive, noSensitive bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Update an app-defined environment variable",
		Long: `Update an existing app-defined environment variable.

The --updated-key flag allows renaming the variable key (optional, defaults to --key).
Use --sensitive to mark as sensitive, or --no-sensitive to unmark.`,
		Example: `  # Update value
  bkms-cli envvar update app --app <appID> --key MY_VAR --value new-value

  # Rename key
  bkms-cli envvar update app --app <appID> --key MY_VAR --updated-key MY_NEW_VAR

  # Mark as sensitive
  bkms-cli envvar update app --app <appID> --key MY_VAR --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sensitive && noSensitive {
				return errors.New("--sensitive and --no-sensitive cannot be used together")
			}

			key = strings.TrimSpace(key)
			updatedKey = strings.TrimSpace(updatedKey)

			// --updated-key 不传时默认使用 --key 的值（不改名）
			effectiveKey := updatedKey
			if effectiveKey == "" {
				effectiveKey = key
			}

			opts := client.UpdateAppDefinedEnvVarOptions{
				UpdatedKey: effectiveKey,
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

			result, err := client.New().UpdateAppDefinedEnvVar(cmd.Context(), appID, key, opts)
			if err != nil {
				return errors.Wrap(err, "update app env var")
			}

			console.Info("Updated app env var: key=%s\n", result.Key)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&key, "key", "", "current environment variable key (required)")
	cmd.Flags().StringVar(&updatedKey, "updated-key", "", "new environment variable key (optional, defaults to --key)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")
	cmd.Flags().BoolVar(&noSensitive, "no-sensitive", false, "unmark sensitive variable")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
