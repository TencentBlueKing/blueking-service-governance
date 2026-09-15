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

// Package publish 开发模式文件发布命令
package publish

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	publishhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/publish"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// newHistoryCmd 创建 publish history 子命令
func newHistoryCmd() *cobra.Command {
	var appID, envName, keyword, outputFormat string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List dev mode publish history records",
		Long: `List dev mode publish history records for the specified app and environment.

Each record corresponds to one instance (pod). Records are returned in
reverse-chronological order.`,
		Example: `  # List publish history for an app environment
  bkms-cli app publish history --app myapp --env stage

  # Filter by keyword (binary name / operator / instance)
  bkms-cli app publish history --app myapp --env stage --keyword my-binary

  # Output in JSON format
  bkms-cli app publish history --app myapp --env stage -o json`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := publishhandler.ListHistory(cmd.Context(), client.New(), appID, envName, keyword)
			if err != nil {
				return errors.Wrap(err, "list dev mode publish history")
			}

			formatted, err := output.FormatData(cmd.Context(), rows, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			console.Info("%s", formatted)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "Environment name (required)")
	cmd.Flags().StringVar(&keyword, "keyword", "", "Filter by keyword (binary name / operator / instance)")
	output.AddFormatFlag(cmd, &outputFormat)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
