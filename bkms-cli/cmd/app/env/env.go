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

// Package env 提供应用环境查询及专属特性环境管理命令。
package env

import "github.com/spf13/cobra"

// NewCmd 创建应用环境命令组。
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage application environments",
		Long: `List standard and feature environments available to an application.
Create and delete manage only application-owned feature environments.
Use 'bkms-cli env' to manage workspace standard environments.`,
	}
	// 查询应用可用的标准环境和专属特性环境
	cmd.AddCommand(NewListCmd())
	// 从标准环境创建应用专属特性环境
	cmd.AddCommand(NewCreateCmd())
	// 删除应用专属特性环境
	cmd.AddCommand(NewDeleteCmd())
	return cmd
}
