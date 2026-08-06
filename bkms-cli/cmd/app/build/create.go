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
)

// NewCreateCmd returns a Command instance for 'app build create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, branch, imageTag string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application build",
		Long: `Create a new build for an application.

This command triggers a new build process for the specified application using
the provided branch and image tag.`,
		Example: `  # Create a build for an application
  bkms-cli app build create --app demo --branch main --image-tag v1.0.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := client.BuildOptions{
				Branch:   branch,
				ImageTag: imageTag,
			}

			if err := client.New().CreateAppBuild(cmd.Context(), appID, opts); err != nil {
				return errors.Wrap(err, "create app build")
			}

			fmt.Println("✓ Build created successfully")
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&branch, "branch", "", "code branch to build")
	cmd.Flags().StringVar(&imageTag, "image-tag", "", "image tag to build")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("branch")
	_ = cmd.MarkFlagRequired("image-tag")

	return cmd
}
