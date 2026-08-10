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
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewPublicCmd returns a Command instance for 'envvar list scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, outputFormat string

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "List scoped (workspace/envType/env) environment variables",
		Long: `List all scoped environment variables in a workspace.

Displays key, value, scopeType, scopeValue, description, and isSensitive fields.
Sensitive values are masked with '******'.`,
		Example: `  # List scoped env vars in default workspace
  bkms-cli envvar list scoped

  # List scoped env vars in a specific workspace
  bkms-cli envvar list scoped --workspace <workspaceID>

  # Output as JSON
  bkms-cli envvar list scoped --workspace <workspaceID> -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envVars, err := client.New().ListPublicEnvVars(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "list scoped env vars")
			}

			formatted, err := output.FormatData(cmd.Context(), envVars, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	return cmd
}
