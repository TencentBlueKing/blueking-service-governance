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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// versionEnvNote 是版本类命令中 --env 的公共说明。
const versionEnvNote = `Without --env, this command works on the default file. With --env, it works on that
environment's own content; an environment without its own content has no versions of its own.`

// NewListVersionsCmd returns a Command instance for 'app config-file list-versions' sub command.
func NewListVersionsCmd() *cobra.Command {
	var appID, envName, outputFormat string
	var ref handler.FileRef

	cmd := &cobra.Command{
		Use:   "list-versions",
		Short: "List config file versions",
		Long:  "List all versions of a config file.\n\n" + versionEnvNote,
		Example: `  bkms-cli app config-file list-versions --app demo --name app.conf
  bkms-cli app config-file list-versions --app demo --name app.conf --env prod -o json`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := handler.ListVersions(cmd.Context(), client.New(), appID, ref, envName)
			if err != nil {
				return errors.Wrap(err, "list app config file versions")
			}
			tipsSharesDefault(&result.VersionsResult, envName)
			return printData(cmd, result.Output(), outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// NewViewVersionCmd returns a Command instance for 'app config-file view-version' sub command.
func NewViewVersionCmd() *cobra.Command {
	var appID, envName, versionID, outputFormat string
	var ref handler.FileRef
	var version int64

	cmd := &cobra.Command{
		Use:   "view-version",
		Short: "View one config file version",
		Long:  "View the content of one version of a config file.\n\n" + versionEnvNote,
		Example: `  bkms-cli app config-file view-version --app demo --name app.conf --version 3
  bkms-cli app config-file view-version --app demo --name app.conf --env prod --version 2 -o 'jq=.content'`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := handler.ViewVersion(
				cmd.Context(), client.New(), appID, ref, envName, toVersionRef(cmd, version, versionID),
			)
			if err != nil {
				return errors.Wrap(err, "view app config file version")
			}
			tipsSharesDefault(&result.VersionsResult, envName)
			return printData(cmd, result.Output(), outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	addVersionRefFlags(cmd, &version, &versionID)
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// NewRollbackVersionCmd returns a Command instance for 'app config-file rollback-version' sub command.
func NewRollbackVersionCmd() *cobra.Command {
	var appID, envName, versionID, description string
	var ref handler.FileRef
	var version int64

	cmd := &cobra.Command{
		Use:   "rollback-version",
		Short: "Roll back a config file to one version",
		Long: "Roll back a config file to the content of one version. The rollback creates a new version.\n\n" +
			versionEnvNote,
		Example: `  bkms-cli app config-file rollback-version --app demo --name app.conf --version 3
  bkms-cli app config-file rollback-version --app demo --name app.conf --env prod --version 2 \
    --description "rollback prod"`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := handler.RollbackVersion(
				cmd.Context(), client.New(), appID, ref, envName, toVersionRef(cmd, version, versionID), description,
			)
			if err != nil {
				return errors.Wrap(err, "rollback app config file version")
			}
			console.Info(
				"✓ Config file (env: %s) rolled back to version %d, current version is %d",
				formatEnvLabel(envName), result.Target.Version, result.Result.CurrentVersion,
			)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	addVersionRefFlags(cmd, &version, &versionID)
	cmd.Flags().StringVar(&description, "description", "", "rollback version description")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// NewDeleteVersionCmd returns a Command instance for 'app config-file delete-version' sub command.
func NewDeleteVersionCmd() *cobra.Command {
	var appID, envName, versionID string
	var ref handler.FileRef
	var version int64

	cmd := &cobra.Command{
		Use:   "delete-version",
		Short: "Delete one config file version",
		Long:  "Delete one history version of a config file. The current version cannot be deleted.\n\n" + versionEnvNote,
		Example: `  bkms-cli app config-file delete-version --app demo --name app.conf --version 3
  bkms-cli app config-file delete-version --app demo --name app.conf --env prod --version-id <record-id>`,
		Args:    cobra.NoArgs,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deleted, err := handler.DeleteVersion(
				cmd.Context(), client.New(), appID, ref, envName, toVersionRef(cmd, version, versionID),
			)
			if err != nil {
				return errors.Wrap(err, "delete app config file version")
			}
			console.Info("✓ Version %d (env: %s) deleted", deleted.Version, formatEnvLabel(envName))
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	addFileRefFlags(cmd, &ref)
	addEnvFlag(cmd, &envName)
	addVersionRefFlags(cmd, &version, &versionID)
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// addVersionRefFlags 注册定位历史版本的参数：--version 或 --version-id。
func addVersionRefFlags(cmd *cobra.Command, version *int64, versionID *string) {
	cmd.Flags().Int64Var(version, "version", 0, "version number")
	cmd.Flags().StringVar(versionID, "version-id", "", "version record ID")
	cmd.MarkFlagsOneRequired("version", "version-id")
	cmd.MarkFlagsMutuallyExclusive("version", "version-id")
}

// toVersionRef 根据用户实际指定的 flag 构造版本定位参数。
func toVersionRef(cmd *cobra.Command, version int64, versionID string) handler.VersionRef {
	if cmd.Flags().Changed("version") {
		return handler.VersionRef{Version: &version}
	}
	return handler.VersionRef{VersionID: versionID}
}

// tipsSharesDefault 在环境没有独立内容、展示的是默认文件版本时给出提示。
func tipsSharesDefault(result *handler.VersionsResult, envName string) {
	if result.SharesDefault() {
		console.Tips("Env %s has no content of its own, showing versions of the default file", envName)
	}
}
