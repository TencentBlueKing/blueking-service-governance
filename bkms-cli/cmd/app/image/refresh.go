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

package image

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	imagehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/image"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewRefreshCmd returns a Command instance for 'app image refresh' sub command
func NewRefreshCmd() *cobra.Command {
	var appID string

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh application images from the image registry",
		Long: `Sync the latest image tags of an application from its image registry.

Images built by the platform are synced automatically after a successful build.
Run this command when images are pushed to the registry by other means
(for example an external pipeline), so they show up in 'app image list'
and can be promoted.`,
		Example: `  # Refresh images for an application
  bkms-cli app image refresh --app demo`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := imagehandler.RefreshImages(cmd.Context(), client.New(), appID)
			if err != nil {
				return err
			}

			if result.Status == client.RefreshAppImagesStatusRefreshing {
				console.Warn("Images of app %s are already being refreshed, please check again later.", appID)
				return nil
			}
			console.Info(
				"✓ Refreshed images of app %s: %s added, %s removed",
				appID, countOrZero(result.AddedTagCnt), countOrZero(result.RemovedTagCnt),
			)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// countOrZero 服务端数量字段为字符串，缺省时按 0 展示
func countOrZero(cnt string) string {
	if cnt == "" {
		return "0"
	}
	return cnt
}
