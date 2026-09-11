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

package env

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeleteCmd 创建 app env delete 子命令，仅删除应用专属特性环境。
func NewDeleteCmd() *cobra.Command {
	var appID, envName string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an application feature environment",
		Long: `Delete a feature environment owned by an application.
Remove its deployments first; the server rejects deletion while deployed apps remain.
The current server deletion API does not remove the Kubernetes namespace.`,
		Example: `  bkms-cli app env delete --app my-app --env feat-1
  bkms-cli app env delete --app my-app --env feat-1 --yes`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// 在解析应用前校验目标参数，避免空白输入进入环境查询及删除确认流程。
			envName = strings.TrimSpace(envName)
			if envName == "" {
				return clierr.Usage(errors.New("--env must not be empty"))
			}
			return cmdutil.ResolveAppPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli := client.New()
			env, err := handler.ResolveFeatureEnv(cmd.Context(), cli, appID, envName)
			if err != nil {
				return err
			}

			printFeatureEnvDeleteConfirmInfo(appID, env)
			confirmed, err := cmdutil.PromptConfirm("Confirm deletion? (yes/no): ", yes)
			if err != nil {
				return errors.Wrap(err, "read confirmation")
			}
			if !confirmed {
				return clierr.ErrCancelled
			}

			// 复用服务端环境删除流程，由服务端校验部署占用并执行已注册的清理 Hook。
			if err = cli.DeleteEnv(cmd.Context(), env.ID); err != nil {
				return errors.Wrap(err, "delete feature env")
			}
			console.Info("✓ Feature environment %s (%s) deleted successfully", env.Name, env.ID)
			return nil
		},
	}
	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "Feature environment name or ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	return cmd
}

// printFeatureEnvDeleteConfirmInfo 展示应用归属、目标环境和部署位置，供用户确认。
func printFeatureEnvDeleteConfirmInfo(appID string, env *client.Env) {
	console.Warn("WARNING: This operation is IRREVERSIBLE.")
	console.Warn("Remove all deployments first. The Kubernetes namespace will not be deleted.")
	console.Info("  Application: %s", appID)
	console.Info("  Environment: %s (%s)", env.Name, env.ID)
	console.Info("  DisplayName: %s", env.DisplayName)
	if env.Cluster != nil {
		console.Info("  Cluster:     %s / %s", env.Cluster.ClusterID, env.Cluster.Namespace)
	}
}
