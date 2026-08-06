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

// Package deploy provides deploy command group
package deploy

import "github.com/spf13/cobra"

// NewCmd returns a Command instance for 'app deploy' command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Manage application deploys",
		Long: `Manage application deploys and deploy records.

Use this command to create new deploys or view deploy records for your applications.`,
		DisableFlagsInUseLine: true,
	}

	// 创建部署
	cmd.AddCommand(NewCreateCmd())
	// 查询部署记录
	cmd.AddCommand(NewListCmd())
	// 更新实例
	cmd.AddCommand(NewUpdateCmd())

	return cmd
}
