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

// Package envvar provide envvar command group for environment variable import/export.
package envvar

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/create"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/delete"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/export"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/importvar"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/list"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/update"
)

// NewCmd creates the envvar command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envvar",
		Short: "Manage environment variables",
		Long: `Manage environment variables import/export and CRUD operations.

Use this command to import/export environment variables from/to local .env files,
or list, create, update, delete individual environment variables.`,
		DisableFlagsInUseLine: true,
	}

	// 导入环境变量
	cmd.AddCommand(importvar.NewCmd())
	// 导出环境变量
	cmd.AddCommand(export.NewCmd())
	// 查询环境变量
	cmd.AddCommand(list.NewCmd())
	// 创建环境变量
	cmd.AddCommand(create.NewCmd())
	// 更新环境变量
	cmd.AddCommand(update.NewCmd())
	// 删除环境变量
	cmd.AddCommand(delete.NewCmd())

	return cmd
}
