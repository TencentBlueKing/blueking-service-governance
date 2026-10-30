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
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/cfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewViewCmd returns a Command instance for 'app config-file view' sub command.
func NewViewCmd() *cobra.Command {
	var appID, envName, outputFormat string
	var ref handler.FileRef
	var raw bool

	cmd := &cobra.Command{
		Use:   "view",
		Short: "View config file content",
		Long: `View the settings and content of a config file.

Without --env, this command shows the default file. With --env, it shows the content that
takes effect in that environment; an environment without its own content uses the default.

--raw prints only the editable content, which is the same field that 'edit' writes to.
For framework files with overlay strategy, the editable content of an environment is a patch
on top of the default file.`,
		Example: `  # View the default content
  bkms-cli app config-file view --app demo --name app.conf

  # View the content that takes effect in prod
  bkms-cli app config-file view --app demo --name app.conf --env prod

  # Save the editable content of prod to a local file
  bkms-cli app config-file view --app demo --name app.conf --env prod --raw > app.conf`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := handler.View(cmd.Context(), client.New(), appID, ref, envName)
			if err != nil {
				return errors.Wrap(err, "view app config file")
			}

			if raw {
				fmt.Print(result.EditableContent())
				return nil
			}
			if result.SharesDefault() {
				if result.IsUnifiedConfig() {
					console.Tips("Config file uses unified config, env %s shares the default content", envName)
				} else {
					console.Tips("Env %s has no content of its own, showing the default content", envName)
				}
			}
			return printData(cmd, result.Output(), outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	cmd.Flags().BoolVar(&raw, "raw", false, "print only the editable content")
	output.AddFormatFlag(cmd, &outputFormat)
	cmd.MarkFlagsMutuallyExclusive("raw", "output")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}
