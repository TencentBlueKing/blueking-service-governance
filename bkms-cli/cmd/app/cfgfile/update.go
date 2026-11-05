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

// updateFlags 保存 'app config-file update' 绑定的原始参数值。
// 是否修改某项由 flag 是否被显式指定决定，因此需要保留 flag 原值而不直接绑定到 UpdateOptions。
type updateFlags struct {
	newName            string
	newMountDir        string
	mountEnvNames      []string
	envContentStrategy string
	perEnv             bool
	unified            bool
	mountAllEnvs       bool
	enableEnvVarRender bool
	yes                bool
}

// NewUpdateCmd returns a Command instance for 'app config-file update' sub command.
func NewUpdateCmd() *cobra.Command {
	var appID string
	var ref handler.FileRef
	var flags updateFlags

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update config file settings",
		Long: `Update the settings of a config file. Only the given flags are changed.

Which settings can be changed depends on the file kind, and the server makes the final check:
  - plain:     name, mount directory, mounted environments, env mode, env var rendering.
  - framework: env mode, env content strategy; name (except trpc) and mount directory (trpc/taf only).

Switching to --unified deletes all environment-specific content, and narrowing --mount-envs deletes
the content of the removed environments, including their version history. Both require --yes.

--mount-all-envs mounts the file on the environments that currently exist; environments created
later are not mounted automatically.`,
		Example: `  # Rename and move a plain file
  bkms-cli app config-file update --app demo --name app.conf --new-name app.ini --new-mount-dir /etc/demo

  # Mount on dev and prod only
  bkms-cli app config-file update --app demo --name app.conf --mount-envs dev,prod --yes

  # Allow environment-specific content
  bkms-cli app config-file update --app demo --name app.conf --per-env

  # Go back to shared content, deleting all environment-specific content
  bkms-cli app config-file update --app demo --name app.conf --unified --yes

  # Change how a framework file applies environment content
  bkms-cli app config-file update --app demo --name trpc_go.yaml --env-content-strategy overwrite`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateUpdateArgs(cmd, args, flags.envContentStrategy)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, appID, ref, flags.toUpdateOptions(cmd), flags.yes)
		},
	}

	addUpdateFlags(cmd, &appID, &ref, &flags)

	return cmd
}

// toUpdateOptions 把显式指定的 flag 转换为更新参数，未指定的项保持不修改。
func (f *updateFlags) toUpdateOptions(cmd *cobra.Command) handler.UpdateOptions {
	flags := cmd.Flags()
	opts := handler.UpdateOptions{MountAllEnvs: f.mountAllEnvs}
	if flags.Changed("new-name") {
		opts.NewName = &f.newName
	}
	if flags.Changed("new-mount-dir") {
		opts.NewMountDir = &f.newMountDir
	}
	if flags.Changed("mount-envs") {
		opts.MountEnvNames = f.mountEnvNames
	}
	if f.perEnv || f.unified {
		opts.IsUnifiedConfig = &f.unified
	}
	if flags.Changed("enable-env-var-render") {
		opts.EnableEnvVarRender = &f.enableEnvVarRender
	}
	if flags.Changed("env-content-strategy") {
		opts.EnvContentStrategy = &f.envContentStrategy
	}
	return opts
}

// addUpdateFlags 注册更新参数以及与参数组合相关的约束。
func addUpdateFlags(cmd *cobra.Command, appID *string, ref *handler.FileRef, f *updateFlags) {
	cmdutil.AddAppFlags(cmd, appID)
	addFileRefFlags(cmd, ref)
	cmd.Flags().StringVar(&f.newName, "new-name", "", "new file name")
	cmd.Flags().StringVar(&f.newMountDir, "new-mount-dir", "", "new absolute mount directory")
	cmd.Flags().StringSliceVar(&f.mountEnvNames, "mount-envs", nil, "mount on these environments only")
	cmd.Flags().BoolVar(&f.mountAllEnvs, "mount-all-envs", false, "mount on all current environments")
	cmd.Flags().BoolVar(&f.perEnv, "per-env", false, "allow environment-specific content")
	cmd.Flags().BoolVar(&f.unified, "unified", false, "share the default content across all environments")
	cmd.Flags().BoolVar(&f.enableEnvVarRender, "enable-env-var-render", false, "enable environment variable rendering")
	cmd.Flags().StringVar(
		&f.envContentStrategy, "env-content-strategy", "", "framework files only: overlay (patch) or overwrite (full)",
	)
	cmd.Flags().BoolVarP(&f.yes, "yes", "y", false, "confirm deleting environment-specific content")
	cmd.MarkFlagsMutuallyExclusive("mount-envs", "mount-all-envs")
	cmd.MarkFlagsMutuallyExclusive("per-env", "unified")
	cmd.MarkFlagsOneRequired(
		"new-name", "new-mount-dir", "mount-envs", "mount-all-envs", "per-env", "unified",
		"enable-env-var-render", "env-content-strategy",
	)
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.RegisterFlagCompletionFunc("env-content-strategy", cobra.FixedCompletions(
		[]string{"overlay", "overwrite"}, cobra.ShellCompDirectiveNoFileComp,
	))
}

// validateUpdateArgs 校验参数取值，并在解析应用 ID 后执行前置检查。
func validateUpdateArgs(cmd *cobra.Command, args []string, envContentStrategy string) error {
	if cmd.Flags().Changed("env-content-strategy") &&
		envContentStrategy != "overlay" && envContentStrategy != "overwrite" {
		return clierr.Usagef("--env-content-strategy must be overlay or overwrite")
	}
	return cmdutil.ResolveAppPreRunE(cmd, args)
}

// runUpdate 生成并执行更新计划，会删除环境内容时要求 --yes 确认。
func runUpdate(cmd *cobra.Command, appID string, ref handler.FileRef, opts handler.UpdateOptions, yes bool) error {
	cli := client.New()
	plan, err := handler.PlanUpdate(cmd.Context(), cli, appID, ref, opts)
	if err != nil {
		return errors.Wrap(err, "plan app config file update")
	}
	if !plan.HasChanges() {
		console.Info("Config file %s is already up to date", plan.Def.Name)
		return nil
	}
	if len(plan.RemovedEnvInstances) > 0 {
		if err = requireYes(yes, fmt.Sprintf(
			"Content of envs [%s] in config file %s will be deleted, including version history.",
			joinEnvNames(plan.RemovedEnvInstances), plan.Def.Name,
		)); err != nil {
			return err
		}
	}

	def, err := handler.ApplyUpdate(cmd.Context(), cli, appID, plan)
	if err != nil {
		return errors.Wrap(err, "update app config file")
	}
	console.Info("✓ Config file %s updated (id: %s)", def.Name, def.ID)
	return nil
}
