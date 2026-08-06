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

// Package startcommand provides the start-command subcommand.
package startcommand

import "github.com/spf13/cobra"

// NewCmd creates the start-command command group with view/edit subcommands.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "start-command",
		Short:                 "Manage application start command and args",
		Long:                  `Manage the application start command and arguments.`,
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewViewCmd())
	cmd.AddCommand(NewEditCmd())

	return cmd
}
