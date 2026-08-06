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

package workspace

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'workspace list' sub command
func NewListCmd() *cobra.Command {
	var keyword, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long: `List all BKMS workspaces you have permission to view.

This command displays workspaces with their ID, display name, and other metadata.
You can filter results using the --keyword flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaces, err := client.New().ListWorkspaces(cmd.Context(), keyword)
			if err != nil {
				return errors.Wrap(err, "list workspaces")
			}
			formatted, err := output.FormatData(cmd.Context(), workspaces, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&keyword, "keyword", "", "filter by keyword")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	return cmd
}
