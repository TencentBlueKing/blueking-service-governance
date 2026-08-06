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

// Package app provide app command
package app

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	apphandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/app"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewCreateCmd returns a Command instance for 'app create' sub command
func NewCreateCmd() *cobra.Command {
	var specFile, workspaceID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application",
		Long: `Create a new application in a workspace from a YAML spec file.

Supported application types: trpc, taf, helm, agones.

The YAML spec file structure is consistent with the backend API request body.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional. Otherwise, you must specify it explicitly.

YAML spec file fields:

  [Common - top level]
    id:           Application ID (optional, auto-generated if not specified)
    name:         Application name (required)
    type:         Application type: trpc | taf | helm | agones (required)
    buildConfig:  Build configuration (required)
    appModelSpec: App model spec for trpc/taf (required when type=trpc/taf)
    helmSpec:     Helm spec for helm/agones (required when type=helm/agones)

  [buildConfig - sourceType=imageRegistry]
    buildConfig.sourceType:                  imageRegistry
    buildConfig.imageBuildConfig.name:       Full image name (required)
    buildConfig.imageBuildConfig.username:   Registry username (optional)
    buildConfig.imageBuildConfig.password:   Registry password (optional)

  [buildConfig - sourceType=codeRepository]
    buildConfig.sourceType:                       codeRepository
    buildConfig.repoBuildConfig.type:             TGit | GitHub (required)
    buildConfig.repoBuildConfig.repoAlias:        Repository alias (required)
    buildConfig.repoBuildConfig.repoURL:          Repository URL (required)
    buildConfig.repoBuildConfig.defaultBranch:    Default branch (required)
    buildConfig.repoBuildConfig.sourceDir:        Source directory (optional)
    buildConfig.repoBuildConfig.dockerfile:       Dockerfile path (optional)
    buildConfig.repoBuildConfig.dockerBuildArgs:  Docker build args (optional)

  [buildConfig - sourceType=pipeline]
    buildConfig.sourceType:                       pipeline
    buildConfig.pipelineBuildConfig.pipelineID:   Pipeline ID (required)
    buildConfig.pipelineBuildConfig.params:       Pipeline params (optional)

  [appModelSpec.trpcSpec] (type=trpc)
    appModelSpec.trpcSpec.language:      go | cpp (required)
    appModelSpec.trpcSpec.fileName:      Config file name (required)
    appModelSpec.trpcSpec.filePath:      Config file path (required)
    appModelSpec.trpcSpec.fileContent:   Config file content (optional)
    appModelSpec.command:                Command (optional)
    appModelSpec.args:                   Args (optional)
    appModelSpec.envVars:                Environment variables (optional, list of key/value/description)

  [appModelSpec.tafSpec] (type=taf)
    appModelSpec.tafSpec.fileName:       Config file name (required)
    appModelSpec.tafSpec.filePath:       Config file path (required)
    appModelSpec.tafSpec.fileContent:    Config file content (optional)
    appModelSpec.command:                Command (optional)
    appModelSpec.args:                   Args (optional)
    appModelSpec.envVars:                Environment variables (optional, list of key/value/description)

  [helmSpec] (type=helm or type=agones)
    helmSpec.helmSource.repoType:     HelmRepo | BCSRepo | GitRepo (required)
    helmSpec.helmSource.valueFiles:   Value files list (optional, default: ["values.yaml"])

  [helmSpec.helmSource - repoType=HelmRepo]
    helmSpec.helmSource.helmRepoConfig.repoURL:     Helm repo URL (required)
    helmSpec.helmSource.helmRepoConfig.chartName:   Chart name (required)
    helmSpec.helmSource.helmRepoConfig.username:    Repo username (optional)
    helmSpec.helmSource.helmRepoConfig.password:    Repo password (optional)

  [helmSpec.helmSource - repoType=BCSRepo]
    helmSpec.helmSource.bcsRepoConfig.projectCode:  BCS project code (required)
    helmSpec.helmSource.bcsRepoConfig.repoName:     BCS repo name (required)
    helmSpec.helmSource.bcsRepoConfig.chartName:    Chart name (required)

  [helmSpec.helmSource - repoType=GitRepo]
    helmSpec.helmSource.gitRepoConfig.type:         TGit | GitHub (required)
    helmSpec.helmSource.gitRepoConfig.repoAlias:    Git repo alias (required)
    helmSpec.helmSource.gitRepoConfig.repoURL:      Git repo URL (required)
    helmSpec.helmSource.gitRepoConfig.revision:     Git revision/branch (required)
    helmSpec.helmSource.gitRepoConfig.sourceDir:    Helm chart directory (required)`,
		Example: `  # Create an application from a YAML spec file:
  bkms-cli app create -f app.yaml

  # Specify workspace explicitly:
  bkms-cli app create -f app.yaml --workspace ws-demo`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			app, err := apphandler.CreateApp(cmd.Context(), workspaceID, specFile)
			if err != nil {
				return errors.Wrap(err, "create app")
			}

			fmt.Printf("✓ App created successfully\n")
			fmt.Printf("  ID:   %s\n", app.ID)
			fmt.Printf("  Name: %s\n", app.Name)
			fmt.Printf("  Type: %s\n", app.Type)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "app spec file path (YAML)")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}
