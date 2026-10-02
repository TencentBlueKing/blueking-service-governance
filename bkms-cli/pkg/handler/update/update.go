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

// Package update handles CLI update checks and installation.
package update

import (
	"context"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/updater"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// Run checks or installs an update, respecting package-managed installations.
func Run(ctx context.Context, checkOnly, force bool) error {
	var (
		info updater.Info
		err  error
	)
	if checkOnly {
		info, err = updater.Check(ctx)
	} else {
		info, err = updater.Update(ctx, force)
	}
	if err != nil {
		return err
	}

	switch {
	case !info.Available:
		console.Info("bkms-cli %s is up to date", info.CurrentVersion)
	case checkOnly || info.UpgradeCommand != "" && !force:
		console.Info("bkms-cli %s is available (current: %s)", info.LatestVersion, info.CurrentVersion)
		if info.UpgradeCommand != "" {
			console.Info("Upgrade with: %s", info.UpgradeCommand)
			if !checkOnly {
				console.Info("Or use --force to replace the binary from the configured update source.")
			}
		}
	default:
		console.Info("bkms-cli updated from %s to %s", info.CurrentVersion, info.LatestVersion)
	}
	return nil
}
