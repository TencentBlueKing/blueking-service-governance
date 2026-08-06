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

// Package probe provides the probe section subcommand.
package probe

import "github.com/spf13/cobra"

// NewCmd creates the probe section command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "probe",
		Short:                 "Manage health probes (liveness, readiness, startup)",
		Long:                  `Manage health probes (liveness, readiness, startup) for the application.`,
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewViewCmd())
	cmd.AddCommand(NewEditCmd())
	cmd.AddCommand(NewResetCmd())

	return cmd
}
