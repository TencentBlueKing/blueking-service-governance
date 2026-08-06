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

// Package appcfgfile provides app config file commands.
package appcfgfile

import "github.com/spf13/cobra"

// NewCmd returns a Command instance for 'app app-cfg-file' command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app-cfg-file",
		Short: "Manage application config files",
		Long: `Manage application config files.

Use this command to view or edit config file content used by BKMS applications.`,
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewEditCmd())
	cmd.AddCommand(NewViewCmd())
	// Versions
	cmd.AddCommand(NewListVersionsCmd())
	cmd.AddCommand(NewViewVersionCmd())
	cmd.AddCommand(NewRollbackVersionCmd())
	cmd.AddCommand(NewDeleteVersionCmd())

	return cmd
}
