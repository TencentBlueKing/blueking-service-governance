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

// Package instance provides instance command group
package instance

import "github.com/spf13/cobra"

// NewCmd returns a Command instance for 'app instance' command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage application instances",
		Long: `Manage application running instances (Pods).

Use this command to view and manage running instances for your applications.`,
		DisableFlagsInUseLine: true,
	}

	// 查询应用实例列表
	cmd.AddCommand(NewListCmd())
	// 查询管理命令列表
	cmd.AddCommand(NewListAdminCmdsCmd())
	// 执行管理命令（自动根据 appType 路由 Trpc/Taf）
	cmd.AddCommand(NewExecAdminCmdCmd())
	// 将本地 TCP 端口转发到单个应用实例 Pod
	cmd.AddCommand(NewPortForwardCmd())

	return cmd
}
