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

package storereg_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	alertstrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
)

var _ = Describe("Store registry", func() {
	AfterEach(func() {
		storereg.Reset()
	})

	It("should register resource lifecycle hooks during initialization", func() {
		storereg.Init(context.Background())

		Expect(
			bkmsenv.IsDeleteHookRegistered(envvarhooks.CleanupScopedEnvVarsByEnvHookName),
		).To(BeTrue(), "envvars cleanup hook must be registered by store registry")
		Expect(
			bkmsworkspace.IsPreDeleteHookRegistered(appdefaultshooks.CleanupRulesByWorkspaceHookName),
		).To(BeTrue(), "workspace AppSpec rule cleanup hook must be registered by store registry")
	})

	It("should register alert strategy env update hook during initialization", func() {
		storereg.Init(context.Background())

		Expect(
			bkmsenv.IsUpdateHookRegistered(alertstrategyhooks.ReconcileEnvTypeChangeHookName),
		).To(BeTrue(), "alert strategy env type change hook must be registered by store registry")
	})
})
