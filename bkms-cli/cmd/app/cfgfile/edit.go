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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewEditCmd returns a Command instance for 'app config-file edit' sub command.
func NewEditCmd() *cobra.Command {
	var appID, envName, filePath, fileContent string
	var ref handler.FileRef
	var opts handler.EditOptions
	var viewCompiledContent bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit config file content",
		Long: `Replace the content of a config file. Each edit creates a new version.

Without --env, this command writes the default content, which is used by every environment
that has no content of its own. With --env, it writes only that environment's content.
If the file uses unified config, it is switched to per-env config first; other environments
keep using the default content.

For framework files with overlay strategy, environment content is a patch merged onto the
default file; use --view-compiled-content to check the merged result.`,
		Example: `  # Edit the default content
  bkms-cli app config-file edit --app demo --name app.conf -f ./app.conf

  # Edit only the prod content
  bkms-cli app config-file edit --app demo --name app.conf --env prod -f ./app-prod.conf

  # Edit from a literal value with a version description
  bkms-cli app config-file edit --app demo --name app.conf --file-content $'key=value\n' --description "init"`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			content, provided, err := readContent(cmd, filePath, fileContent)
			if err != nil {
				return err
			}
			if !provided {
				return clierr.Usagef("one of --file or --file-content is required")
			}
			opts.Content = content

			result, err := handler.Edit(cmd.Context(), client.New(), appID, ref, envName, opts)
			if result != nil && result.SwitchedToPerEnv {
				console.Tips("Config file %s switched to per-env config", result.Def.Name)
			}
			if err != nil {
				return errors.Wrap(err, "edit app config file")
			}
			if envName != "" && result.CreatedEnvInstance {
				console.Tips("Created content for env %s", envName)
			}

			if viewCompiledContent {
				fmt.Print(result.UpdateResult.Content)
				return nil
			}
			console.Info(
				"✓ Config file %s (env: %s) updated to version %d",
				result.Def.Name, formatEnvLabel(envName), result.UpdateResult.CurrentVersion,
			)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	addContentFlags(cmd, &filePath, &fileContent)
	cmd.MarkFlagsOneRequired("file", "file-content")
	cmd.Flags().StringVar(&opts.Description, "description", "", "version description")
	cmd.Flags().
		BoolVar(&viewCompiledContent, "view-compiled-content", false, "print the compiled content after update")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}
