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

package resources

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("app model conversion", func() {
	DescribeTable("FromAppModel",
		func(appModel *appmodel.AppModel, expected *Spec) {
			Expect(FromAppModel(appModel)).To(Equal(expected))
		},
		Entry("returns nil for an empty app model",
			&appmodel.AppModel{
				Workload: appmodel.Workload{},
			},
			nil,
		),
		Entry("maps all populated fields",
			&appmodel.AppModel{
				Replicas: lo.ToPtr(int32(3)),
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu":    "100m-200m",
						"memory": "256Mi-512Mi",
					},
				},
			},
			&Spec{
				Replicas:       lo.ToPtr(int32(3)),
				CPURequests:    lo.ToPtr("100m"),
				CPULimits:      lo.ToPtr("200m"),
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("512Mi"),
			},
		),
		Entry("uses the same value for cpu requests and limits when only one value is provided",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu": "250m",
					},
				},
			},
			&Spec{
				CPURequests: lo.ToPtr("250m"),
				CPULimits:   lo.ToPtr("250m"),
			},
		),
	)

	DescribeTable("ApplyToAppModel",
		func(
			spec *Spec,
			appModel *appmodel.AppModel,
			expected *appmodel.AppModel,
		) {
			Expect(ApplyToAppModel(spec, appModel)).To(Equal(expected))
		},
		Entry("overrides selected fields and preserves the others when cpu and memory both have values",
			&Spec{
				Replicas:       lo.ToPtr(int32(6)),
				CPURequests:    lo.ToPtr("300m"),
				CPULimits:      lo.ToPtr("600m"),
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("512Mi"),
			},
			&appmodel.AppModel{
				AppID:    "app-1",
				Labels:   map[string]string{"team": "core"},
				Replicas: lo.ToPtr(int32(2)),
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu":    "100m-200m",
						"memory": "128Mi-256Mi",
					},
				},
			},
			&appmodel.AppModel{
				AppID:    "app-1",
				Labels:   map[string]string{"team": "core"},
				Replicas: lo.ToPtr(int32(6)),
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu":    "300m-600m",
						"memory": "256Mi-512Mi",
					},
				},
			},
		),
		Entry("uses the same value for memory requests and limits while resetting other managed fields",
			&Spec{
				MemoryRequests: lo.ToPtr("1Gi"),
			},
			&appmodel.AppModel{
				AppID:    "app-2",
				Replicas: lo.ToPtr(int32(4)),
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu":    "250m-500m",
						"memory": "256Mi-512Mi",
					},
				},
			},
			&appmodel.AppModel{
				AppID: "app-2",
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"memory": "1Gi-1Gi",
					},
				},
			},
		),
		Entry("strictly resets managed fields when the section is nil",
			nil,
			&appmodel.AppModel{
				AppID:    "app-3",
				Replicas: lo.ToPtr(int32(4)),
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"cpu":    "250m-500m",
						"memory": "256Mi-512Mi",
						"gpu":    "1",
					},
				},
			},
			&appmodel.AppModel{
				AppID: "app-3",
				Workload: appmodel.Workload{
					Resources: map[string]string{
						"gpu": "1",
					},
				},
			},
		),
	)
})
