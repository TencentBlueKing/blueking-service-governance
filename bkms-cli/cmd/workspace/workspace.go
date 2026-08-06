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

// Package workspace provide workspace command
package workspace

import "github.com/spf13/cobra"

// NewCmd create workspace command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
		Long: `Manage BKMS workspaces and workspace-related configurations.

Use this command to list workspaces, set or unset default workspace for your CLI operations.`,
		DisableFlagsInUseLine: true,
	}

	// 有权限的工作空间列表
	cmd.AddCommand(NewListCmd())
	// 设置默认工作空间
	cmd.AddCommand(NewSetCmd())
	// 取消设置默认工作空间
	cmd.AddCommand(NewUnsetCmd())

	return cmd
}
