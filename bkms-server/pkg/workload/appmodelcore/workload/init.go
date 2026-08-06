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

// Package workload wires up default workload plugins and section registrations at process start.
package workload

import (
	"sync"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/standard"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
)

var initOnce sync.Once

// InitPlugin initializes and registers workload plugins with its dependencies.
// This function must be called after database initialization and before the server starts.
func InitPlugin(
	appConfigFileStore appcfg.AppConfigFileStore,
	polarisConfigStore polaris.PolarisConfigStore,
) {
	initOnce.Do(func() {
		plugin.MustRegisterWorkloadPlugin(standard.Plugin{})
		plugin.MustRegisterWorkloadPlugin(trpc.NewPlugin(
			appConfigFileStore,
			polarisConfigStore,
		))
		plugin.MustRegisterWorkloadPlugin(taf.NewPlugin(appConfigFileStore))
	})
}
