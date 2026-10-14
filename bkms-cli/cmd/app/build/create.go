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

// Package build provides build create command
package build

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/build"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewCreateCmd returns a Command instance for 'app build create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, branch, tag, commit, imageTag string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application build",
		Long: `Create a new build for an application.

This command triggers a new build for the specified application.
Provide exactly one of --branch, --tag, or --commit, plus --image-tag.`,
		Example: `  # Build from a branch
  bkms-cli app build create --app demo --branch main --image-tag v1.0.0

  # Build from a tag
  bkms-cli app build create --app demo --tag v1.0.0 --image-tag v1.0.0

  # Build from a commit
  bkms-cli app build create --app demo --commit a1b2c3d --image-tag v1.0.0`,
		// 先校验代码来源，再解析应用。非法 CommitID 或不满足三选一时不发任何请求
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := build.ResolveCodeRevision(branch, tag, commit); err != nil {
				return err
			}
			return cmdutil.ResolveAppPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			revisionType, revision, err := build.ResolveCodeRevision(branch, tag, commit)
			if err != nil {
				return err
			}

			opts := client.BuildOptions{
				RevisionType: revisionType,
				Revision:     revision,
				ImageTag:     imageTag,
			}

			if err := client.New().CreateAppBuild(cmd.Context(), appID, opts); err != nil {
				return errors.Wrap(err, "create app build")
			}

			fmt.Println("✓ Build created successfully")
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&branch, "branch", "", "code branch to build (mutually exclusive with --tag and --commit)")
	cmd.Flags().StringVar(&tag, "tag", "", "code tag to build (mutually exclusive with --branch and --commit)")
	cmd.Flags().StringVar(
		&commit,
		"commit",
		"",
		"commit id to build, 7-40 hex characters (mutually exclusive with --branch and --tag)",
	)
	cmd.Flags().StringVar(&imageTag, "image-tag", "", "image tag to build")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("image-tag")

	return cmd
}
