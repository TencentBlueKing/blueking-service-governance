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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCreateCmd returns a Command instance for 'app config-file create' sub command.
func NewCreateCmd() *cobra.Command {
	var appID, filePath, fileContent string
	var opts handler.CreateOptions
	var enableEnvVarRender bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a plain config file",
		Long: `Create a plain config file and mount it into the application containers.

The container path of the file is <mount-dir>/<name>. Plain files are only supported by
trpc/taf/bkmsApp applications. By default the file is mounted on all environments and
all environments share the same content; use 'edit --env' to give one environment its own content.`,
		Example: `  # Create an empty file mounted on all environments
  bkms-cli app config-file create --app demo --name app.conf --mount-dir /data/conf

  # Create a file from local content, mounted on dev and prod only
  bkms-cli app config-file create --app demo --name app.conf --mount-dir /data/conf \
    -f ./app.conf --mount-envs dev,prod

  # Enable environment variable rendering (${{ env.KEY }} placeholders)
  bkms-cli app config-file create --app demo --name app.conf --mount-dir /data/conf \
    -f ./app.conf --enable-env-var-render`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			content, provided, err := readContent(cmd, filePath, fileContent)
			if err != nil {
				return err
			}
			if provided {
				opts.Content = &content
			}
			if !cmd.Flags().Changed("mount-envs") {
				opts.MountEnvNames = nil
			}
			if cmd.Flags().Changed("enable-env-var-render") {
				opts.EnableEnvVarRender = &enableEnvVarRender
			}

			def, err := handler.Create(cmd.Context(), client.New(), appID, opts)
			if err != nil {
				return errors.Wrap(err, "create app config file")
			}
			console.Info("✓ Config file %s created in %s (id: %s)", def.Name, def.MountDir, def.ID)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&opts.Name, "name", "", "file name (letters, digits, '_', '-', '.')")
	cmd.Flags().StringVar(&opts.MountDir, "mount-dir", "", "absolute mount directory in the container")
	addContentFlags(cmd, &filePath, &fileContent)
	cmd.Flags().
		StringSliceVar(&opts.MountEnvNames, "mount-envs", nil, "mount on these environments only (default: all)")
	cmd.Flags().BoolVar(&enableEnvVarRender, "enable-env-var-render", false, "enable environment variable rendering")
	cmd.Flags().StringVar(&opts.Description, "description", "", "version description")
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("mount-dir")

	return cmd
}
