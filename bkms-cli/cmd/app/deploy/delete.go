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

package deploy

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
	deployhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeployDeleteCmd returns a Command instance for 'app deploy delete' sub command
func NewDeployDeleteCmd() *cobra.Command {
	var appID, envName, envType, deployID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete (undeploy) an application from an environment",
		Long: `Remove the deployment of an application from one or more environments.

For helm applications, you must specify --deploy-id (from 'deploy list') and a single
environment. For trpc and taf applications, the entire environment deployment is removed.

The --env flag supports multiple environment names separated by commas (e.g. --env test,staging).
The --env-type flag removes deployments from every environment of the given type(s), separated by
commas (e.g. --env-type test,development). Valid types: development | test | staging | production.
--env and --env-type are mutually exclusive; exactly one of them must be provided.`,
		Example: `  # Delete trpc/taf deployment
  bkms-cli app deploy delete --app myapp --env test

  # Delete helm deployment (deploy-id required)
  bkms-cli app deploy delete --app myapp --env test --deploy-id deploy1

  # Delete without confirmation prompt
  bkms-cli app deploy delete --app myapp --env test --yes

  # Delete deployments from multiple environments
  bkms-cli app deploy delete --app myapp --env test,staging

  # Delete deployments from all environments of given types
  bkms-cli app deploy delete --app myapp --env-type test,development`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := client.New()

			app, err := cli.GetApp(cmd.Context(), appID)
			if err != nil {
				return errors.Wrap(err, "get app")
			}

			if app.Type == constant.AppTypeHelm && deployID == "" {
				return clierr.Usagef("--deploy-id is required for helm applications (see 'app deploy list')")
			}

			// 解析并校验卸载目标环境
			envNames, err := deployhandler.ResolveAndValidateEnvNames(cmd.Context(), cli, appID, envName, envType)
			if err != nil {
				return err
			}

			// helm 卸载需针对单个环境（deploy-id 绑定单个部署记录）
			if app.Type == constant.AppTypeHelm && len(envNames) > 1 {
				return clierr.Usagef("helm applications support deleting a single environment at a time")
			}

			printDeployDeleteConfirmInfo(app, envNames, deployID)

			confirmed, confirmErr := cmdutil.PromptConfirm("Confirm delete deployment? (yes/no): ", yes)
			if confirmErr != nil {
				return errors.Wrap(confirmErr, "read confirmation")
			}
			if !confirmed {
				return clierr.ErrCancelled
			}

			return deployhandler.DeleteDeployBatch(cmd.Context(), cli, app.Type, appID, envNames, deployID)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().
		StringVar(&envType, "env-type", "", "environment type (comma-separated): development | test | staging | production")
	cmd.Flags().StringVar(&deployID, "deploy-id", "", "deploy record ID (required for helm apps)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	_ = cmd.MarkFlagRequired("app")
	// --env 与 --env-type 二选一，在发起请求前校验
	cmd.MarkFlagsOneRequired("env", "env-type")
	cmd.MarkFlagsMutuallyExclusive("env", "env-type")

	return cmd
}

func printDeployDeleteConfirmInfo(app *client.AppFull, envNames []string, deployID string) {
	console.Info("  App:  %s (%s)", app.Name, app.ID)
	console.Info("  Env:  %s", strings.Join(envNames, ", "))
	console.Info("  Type: %s", app.Type)
	if deployID != "" {
		console.Info("  DeployID: %s", deployID)
	}
	console.Info("")
}
