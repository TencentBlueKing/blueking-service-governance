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

// Package deploy provides deploy create command
package deploy

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewCreateCmd returns a Command instance for 'app deploy create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, envName, deploySpecFile, workspaceID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application deploy",
		Long: `Create a new deploy for an application.

Supported application types: helm, trpc, taf.

The --env flag supports multiple environment names separated by commas (e.g. --env prod,staging).
When multiple environments are specified, the deploy will be executed for each environment sequentially.
Environment names are validated against the workspace before deployment.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional. Otherwise, you must specify it explicitly.

Deploy file fields by type:

  [helm]
    imageTag:      Image tag (required)
    chartVersion:  Chart version
    valuesFile:    Values file ID
    trafficLane:   Traffic lane name

  [trpc]
    imageTag:      Image tag (required)
    replicas:      Number of replicas, >= 1 (required)

  [taf]
    imageTag:      Image tag (required)
    replicas:      Number of replicas, >= 1 (required)`,
		Example: `  # 1. Helm deploy file (helm-deploy.yaml):
  imageTag: v1.0.0
  chartVersion: 1.2.3
  valuesFile: values-prod
  trafficLane: grayscale-lane

  # Execute deploy
  bkms-cli app deploy create --app my-app --env prod -f helm-deploy.yaml

  # Specify workspace explicitly
  bkms-cli app deploy create --workspace ws-demo --app my-app --env prod -f helm-deploy.yaml

  # 2. Trpc deploy file (trpc-deploy.yaml):
  replicas: 3
  imageTag: v1.0.0

  # Execute deploy
  bkms-cli app deploy create --app my-app --env prod -f trpc-deploy.yaml

  # 3. TAF deploy file (taf-deploy.yaml):
  imageTag: v1.0.0
  replicas: 2

  # Execute deploy
  bkms-cli app deploy create --app my-app --env prod -f taf-deploy.yaml

  # 4. Deploy to multiple environments at once
  bkms-cli app deploy create --app my-app --env prod,staging,test -f trpc-deploy.yaml`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			if err := deploy.CreateDeploy(cmd.Context(), workspaceID, appID, envName, deploySpecFile); err != nil {
				return errors.Wrap(err, "create app deploy")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVarP(&deploySpecFile, "deploy-spec-file", "f", "", "deploy spec file path")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("deploy-spec-file")

	return cmd
}
