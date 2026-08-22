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
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/updater"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

const (
	updateCheckTimeout = 15 * time.Second
	npmUpgradeCommand  = "npm i -g @blueking/bkms-cli@latest"
)

// NewCmd creates the self-update command.
func NewCmd() *cobra.Command {
	var checkOnly bool
	var force bool

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
			viaNPM := updater.InstalledViaNPM()
			if viaNPM && !checkOnly && !force {
				console.Info("This bkms-cli was installed via npm. Upgrade with:")
				console.Info("  %s", npmUpgradeCommand)
				console.Info("Or use --force to replace the binary from GitHub Releases.")
				return nil
			}

			var (
				info updater.Info
				err  error
			)
			if checkOnly {
				checkContext, cancel := context.WithTimeout(cmd.Context(), updateCheckTimeout)
				defer cancel()
				info, err = updater.Check(checkContext)
			} else {
				info, err = updater.Update(cmd.Context())
			}
			if err != nil {
				return err
			}

			switch {
			case !info.Available:
				console.Info("bkms-cli %s is up to date", info.CurrentVersion)
			case checkOnly && viaNPM && !force:
				console.Info(
					"bkms-cli %s is available (current: %s); upgrade with: %s",
					info.LatestVersion,
					info.CurrentVersion,
					npmUpgradeCommand,
				)
			case checkOnly:
				console.Info("bkms-cli %s is available (current: %s)", info.LatestVersion, info.CurrentVersion)
			default:
				console.Info("bkms-cli updated from %s to %s", info.CurrentVersion, info.LatestVersion)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	cmd.Flags().BoolVar(&force, "force", false, "force self-update even when installed via npm")
	return cmd
}
