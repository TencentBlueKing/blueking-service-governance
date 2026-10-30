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

package cfgfile

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/cfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewMountPreviewCmd returns a Command instance for 'app config-file mount-preview' sub command.
func NewMountPreviewCmd() *cobra.Command {
	var appID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "mount-preview",
		Short: "Preview config files mounted in an environment",
		Long: `List the config files that will be mounted into containers in an environment.

CONTENT_SOURCE shows where the content comes from: normal (default content), overlay
(environment patch merged onto the default) or overwrite (the environment's own content).`,
		Example: `  bkms-cli app config-file mount-preview --app demo --env prod`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := handler.MountPreview(cmd.Context(), client.New(), appID, envName)
			if err != nil {
				return errors.Wrap(err, "preview app config file mounts")
			}
			return printData(cmd, items, outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
