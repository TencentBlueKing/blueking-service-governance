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

package updatestrategy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("app model conversion", func() {
	DescribeTable("ApplyToAppModel",
		func(
			spec *Spec,
			appModel *appmodel.AppModel,
			expected *appmodel.AppModel,
		) {
			Expect(ApplyToAppModel(spec, appModel)).To(Equal(expected))
		},
		Entry("sets managed update strategy fields while preserving type",
			&Spec{
				MaxUnavailable: lo.ToPtr("20%"),
				MaxSurge:       lo.ToPtr("3"),
			},
			&appmodel.AppModel{
				AppID: "app-1",
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type:           "RollingUpdate",
					MaxUnavailable: lo.ToPtr("10%"),
					MaxSurge:       lo.ToPtr("1"),
				},
			},
			&appmodel.AppModel{
				AppID: "app-1",
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type:           "RollingUpdate",
					MaxUnavailable: lo.ToPtr("20%"),
					MaxSurge:       lo.ToPtr("3"),
				},
			},
		),
		Entry("strictly resets managed update strategy fields when the section is nil",
			nil,
			&appmodel.AppModel{
				AppID: "app-2",
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type:           "RollingUpdate",
					MaxUnavailable: lo.ToPtr("10%"),
					MaxSurge:       lo.ToPtr("1"),
				},
			},
			&appmodel.AppModel{
				AppID: "app-2",
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type: "RollingUpdate",
				},
			},
		),
	)
})
