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

// NewListCmd returns a Command instance for 'app config-file list' sub command.
func NewListCmd() *cobra.Command {
	var appID, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List application config files",
		Long: `List all config files of an application, including framework and plain files.

MOUNT_ENVS "*" means the file is mounted on all environments.
ENV_MODE "unified" means all environments share the default content; "per-env" means
environments may have their own content.`,
		Example: `  # List config files
  bkms-cli app config-file list --app demo

  # Output in JSON format
  bkms-cli app config-file list --app demo -o json`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := handler.List(cmd.Context(), client.New(), appID)
			if err != nil {
				return errors.Wrap(err, "list app config files")
			}
			return printData(cmd, items, outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")

	return cmd
}
