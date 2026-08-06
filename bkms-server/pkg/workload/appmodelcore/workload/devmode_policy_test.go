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

package workload

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("shouldEnableDevModeInEnv", func() {
	DescribeTable(
		"determines whether dev mode should be enabled for a given env type",
		func(envType string, expected bool) {
			spec := &appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(true)}}
			Expect(shouldEnableDevModeInEnv(envType, spec)).To(Equal(expected))
		},
		Entry("development env enables dev mode", string(env.TypeDevelopment), true),
		Entry("test env enables dev mode", string(env.TypeTest), true),
		Entry("staging env enables dev mode", string(env.TypeStaging), true),
		Entry("production env does not enable dev mode", string(env.TypeProduction), false),
	)
})
