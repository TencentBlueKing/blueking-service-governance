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

// Package delete provides the 'envvar delete' sub-command group.
package delete

import "github.com/spf13/cobra"

// NewCmd creates the delete sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an environment variable",
		Long: `Delete an environment variable from the server.

Supports deleting scoped (workspace/envType) vars, env-scoped vars, and app-defined vars.`,
		DisableFlagsInUseLine: true,
	}

	// 删除 scoped 环境变量（workspace/envType）
	cmd.AddCommand(NewPublicCmd())
	// 删除应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 删除环境某个变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
