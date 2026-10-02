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

// Package update provides the bkms-cli self-update command.
package update

import (
	"github.com/spf13/cobra"

	updatehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/update"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewCmd creates the self-update command.
func NewCmd() *cobra.Command {
	var checkOnly, force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install bkms-cli updates.",
		Example: "  bkms-cli update --check\n" +
			"  bkms-cli update\n" +
			"  bkms-cli update --force",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return updatehandler.Run(cmd.Context(), checkOnly, force)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	cmd.Flags().BoolVar(&force, "force", false, "allow replacing npm or Go builds from the configured update source")
	return cmd
}
