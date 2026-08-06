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

package workspace

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewSetCmd returns a Command instance for 'workspace set' sub command
// which can set the default workspaceID in config file
func NewSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set WORKSPACE_ID",
		Short: "Set default workspace",
		Long: `Set the default workspace ID for subsequent commands.

This command validates the workspace exists and you have permission to access it,
then saves it as the default workspace in your configuration file.

Example:
  bkms-cli workspace set my-workspace-id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 使用第一个参数作为 workspaceID
			if len(args) != 1 {
				return errors.New("should provide workspaceID in args")
			}
			workspaceID := args[0]

			// 验证工作空间真的存在 & 有权限
			workspace, err := client.New().GetWorkspace(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "get workspace")
			}
			if workspace == nil || workspace.ID != workspaceID {
				return errors.New("workspace not found")
			}

			// 设置默认工作空间
			config.G.Defaults.WorkspaceID = workspaceID
			if err = config.G.Dump(); err != nil {
				return errors.Wrap(err, "dump config")
			}
			console.Info("set default workspace as `%s` successfully", workspaceID)
			return nil
		},
	}
}
