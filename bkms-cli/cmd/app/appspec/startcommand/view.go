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

package startcommand

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewViewCmd returns a Command instance for 'appspec start-command view' sub command.
func NewViewCmd() *cobra.Command {
	var appID, outputFormat string

	cmd := &cobra.Command{
		Use:    "view",
		Short:  "View application start command configuration",
		PreRun: cmdutil.CommonPreRun,
		Long: `View the application start command and arguments configuration.

Displays the current container entrypoint command and arguments for the application.`,
		Example: `  # View current start command
  bkms-cli app appspec start-command view --app my-app

  # Output in JSON format
  bkms-cli app appspec start-command view --app my-app -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := appspec.ViewStartCommandHandler(cmd.Context(), appID)
			if err != nil {
				return errors.Wrap(err, "view start command")
			}

			if outputFormat != "" {
				formatted, fmtErr := output.FormatData(cmd.Context(), result, outputFormat)
				if fmtErr != nil {
					return errors.Wrap(fmtErr, "format output")
				}
				fmt.Println(formatted)
				return nil
			}

			// Default: table format
			fmt.Println(result.FormatTable())
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
