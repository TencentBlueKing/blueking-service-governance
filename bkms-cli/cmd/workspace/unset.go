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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewUnsetCmd returns a Command instance for 'workspace unset' sub command
// which can unset the default workspaceID in config file
func NewUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset",
		Short: "Unset default workspace",
		Long: `Unset the default workspace ID from your configuration.

After running this command, you will need to explicitly specify the workspace ID
for commands that require it using the --workspace flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 取消设置默认工作空间
			config.G.Defaults.WorkspaceID = ""
			if err := config.G.Dump(); err != nil {
				return errors.Wrap(err, "dump config")
			}
			console.Info("unset default workspace successfully")
			return nil
		},
	}
}
