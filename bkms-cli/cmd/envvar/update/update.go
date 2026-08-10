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

// Package update provides the 'envvar update' sub-command group.
package update

import "github.com/spf13/cobra"

// NewCmd creates the update sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an environment variable",
		Long: `Update an existing environment variable on the server.

Supports updating public (workspace/envType/env) scoped vars and app-defined vars.`,
		DisableFlagsInUseLine: true,
	}

	// 更新公共环境变量
	cmd.AddCommand(NewPublicCmd())
	// 更新应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 更新环境变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
