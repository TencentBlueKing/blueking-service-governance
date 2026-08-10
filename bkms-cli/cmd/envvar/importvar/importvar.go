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

// Package importvar provides the 'envvar import' sub-command group.
package importvar

import "github.com/spf13/cobra"

// NewCmd creates the import sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import environment variables from a local .env file",
		Long: `Import environment variables from a local .env file to the server.

Supports importing scoped (workspace/envType), env-scoped, and app-defined env vars.`,
		DisableFlagsInUseLine: true,
	}

	// 导入 scoped 环境变量（workspace/envType）
	cmd.AddCommand(NewPublicCmd())
	// 导入环境变量
	cmd.AddCommand(NewEnvCmd())
	// 导入应用环境变量
	cmd.AddCommand(NewAppCmd())

	return cmd
}
