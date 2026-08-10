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

// NewAppCmd returns a Command instance for 'envvar create for app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, key, value, description string
	var sensitive bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Create an app-defined environment variable",
		Long: `Create a new environment variable directly defined by an application.

The key must be unique within the application.`,
		Example: `  # Create an app env var
  bkms-cli envvar create app --app <appID> --key MY_VAR --value my-value

  # Create a sensitive app env var
  bkms-cli envvar create app --app <appID> --key MY_VAR --value my-value --sensitive

  # Create with description
  bkms-cli envvar create app --app <appID> --key MY_VAR --value my-value --description "My variable"`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			key = strings.TrimSpace(key)

			result, err := client.New().
				CreateAppDefinedEnvVar(cmd.Context(), appID, client.CreateAppDefinedEnvVarOptions{
					Key:         key,
					Value:       value,
					Description: description,
					IsSensitive: sensitive,
				})
			if err != nil {
				return errors.Wrap(err, "create app env var")
			}

			console.Info("Created app env var: key=%s\n", result.Key)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
