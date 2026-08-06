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

// Package image provides image command
package image

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'app image list' sub command
func NewListCmd() *cobra.Command {
	var appID, keyword, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List application images",
		Long: `List all container images for an application.

This command retrieves all available container images for the specified
application. You can filter results using keywords.`,
		Example: `  # List all images for an application
  bkms-cli app image list --app demo

  # Filter images by keyword
  bkms-cli app image list --app demo --keyword v1.0

  # Output in JSON format
  bkms-cli app image list --app demo -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := client.New().ListAppImages(cmd.Context(), appID, keyword)
			if err != nil {
				return errors.Wrap(err, "list app images")
			}

			formatted, err := output.FormatData(cmd.Context(), records, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)

			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&keyword, "keyword", "", "filter by keyword")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
