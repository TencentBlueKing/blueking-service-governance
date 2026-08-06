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

package tagdeletion

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmrelease "helm.sh/helm/v3/pkg/release"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	infrahelm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("Tag deletion helpers", func() {
	DescribeTable(
		"collectLatestAppModelLaneUsages",
		func(status appmodeldeploy.Status, expected []ImageUsage) {
			// 仅保留一个 workload 和一个非 workload 资源，
			// 便于验证函数是否会正确过滤并返回占用名称。
			record := &appmodeldeploy.Record{
				ImageTag: "v1.0.0",
				Status:   status,
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.Deploy, Name: "workload-a"},
					{Kind: k8skind.SVC, Name: "svc-a"},
				},
			}

			Expect(collectLatestAppModelLaneUsages(record, "v1.0.0")).To(Equal(expected))
		},
		Entry(
			"returns workload names for latest deployed record",
			appmodeldeploy.StatusDeployed,
			[]ImageUsage{{WorkloadName: "workload-a", Status: string(appmodeldeploy.StatusDeployed)}},
		),
		Entry(
			"returns workload names for ongoing deploy",
			appmodeldeploy.StatusDeploying,
			[]ImageUsage{{WorkloadName: "workload-a", Status: string(appmodeldeploy.StatusDeploying)}},
		),
		Entry(
			"returns workload names for polling broken",
			appmodeldeploy.StatusPollingBroken,
			[]ImageUsage{{WorkloadName: "workload-a", Status: string(appmodeldeploy.StatusPollingBroken)}},
		),
		Entry(
			"returns workload names for polling timeout",
			appmodeldeploy.StatusPollingTimeout,
			[]ImageUsage{{WorkloadName: "workload-a", Status: string(appmodeldeploy.StatusPollingTimeout)}},
		),
		Entry("ignores failed record", appmodeldeploy.StatusFailed, nil),
	)

	DescribeTable(
		"collectLatestHelmLaneUsages",
		func(status helmrelease.Status, expected []ImageUsage) {
			// Helm 场景下用户可见的 workload 名称来自 ReleaseName。
			record := &helmdeploy.Record{
				ImageTag:    "v1.0.0",
				ReleaseName: "demo-release",
				Status:      status,
			}

			Expect(collectLatestHelmLaneUsages(record, "v1.0.0")).To(Equal(expected))
		},
		Entry(
			"returns release name for latest deployed record",
			helmrelease.StatusDeployed,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(helmrelease.StatusDeployed)}},
		),
		Entry(
			"returns release name for pending install",
			helmrelease.StatusPendingInstall,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(helmrelease.StatusPendingInstall)}},
		),
		Entry(
			"returns release name for pending upgrade",
			helmrelease.StatusPendingUpgrade,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(helmrelease.StatusPendingUpgrade)}},
		),
		Entry(
			"returns release name for pending rollback",
			helmrelease.StatusPendingRollback,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(helmrelease.StatusPendingRollback)}},
		),
		Entry(
			"returns release name for polling broken",
			infrahelm.StatusPollingBroken,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(infrahelm.StatusPollingBroken)}},
		),
		Entry(
			"returns release name for polling timeout",
			infrahelm.StatusPollingTimeout,
			[]ImageUsage{{WorkloadName: "demo-release", Status: string(infrahelm.StatusPollingTimeout)}},
		),
		Entry("ignores failed record", helmrelease.StatusFailed, nil),
	)

	DescribeTable(
		"shouldFallbackAppModel",
		func(record *appmodeldeploy.Record, expected bool) {
			// 该函数只判断是否需要回退到最近一次成功部署记录，
			// 因此这里直接按状态枚举断言布尔结果即可。
			Expect(shouldFallbackAppModel(record)).To(Equal(expected))
		},
		Entry(
			"does not fallback deployed record",
			&appmodeldeploy.Record{Status: appmodeldeploy.StatusDeployed},
			false,
		),
		Entry(
			"does not fallback ongoing record",
			&appmodeldeploy.Record{Status: appmodeldeploy.StatusDeploying},
			false,
		),
		Entry(
			"does not fallback polling broken record",
			&appmodeldeploy.Record{Status: appmodeldeploy.StatusPollingBroken},
			false,
		),
		Entry("falls back failed record", &appmodeldeploy.Record{Status: appmodeldeploy.StatusFailed}, true),
		Entry(
			"does not fallback uninstalled record",
			&appmodeldeploy.Record{Status: appmodeldeploy.StatusUninstalled},
			false,
		),
	)

	DescribeTable(
		"shouldFallbackHelm",
		func(record *helmdeploy.Record, expected bool) {
			// Helm 的回退规则与 appmodel 类似，但 ongoing 状态集合不同。
			Expect(shouldFallbackHelm(record)).To(Equal(expected))
		},
		Entry("does not fallback deployed record", &helmdeploy.Record{Status: helmrelease.StatusDeployed}, false),
		Entry(
			"does not fallback pending install record",
			&helmdeploy.Record{Status: helmrelease.StatusPendingInstall},
			false,
		),
		Entry(
			"does not fallback ongoing record",
			&helmdeploy.Record{Status: helmrelease.StatusPendingUpgrade},
			false,
		),
		Entry(
			"does not fallback pending rollback record",
			&helmdeploy.Record{Status: helmrelease.StatusPendingRollback},
			false,
		),
		Entry(
			"does not fallback polling timeout record",
			&helmdeploy.Record{Status: infrahelm.StatusPollingTimeout},
			false,
		),
		Entry("falls back failed record", &helmdeploy.Record{Status: helmrelease.StatusFailed}, true),
		Entry(
			"does not fallback uninstalled record",
			&helmdeploy.Record{Status: helmrelease.StatusUninstalled},
			false,
		),
	)

	It("returns unique workload names only for workload resources", func() {
		// 同时放入重复 workload 和非 workload 资源，
		// 验证函数会做过滤与去重。
		names := collectAppModelWorkloadNames(appmodeldeploy.ResourceKeys{
			{Kind: k8skind.Deploy, Name: "deploy-a"},
			{Kind: k8skind.SVC, Name: "svc-a"},
			{Kind: k8skind.Secret, Name: "secret-a"},
			{Kind: k8skind.Deploy, Name: "deploy-a"},
		})

		Expect(names).To(Equal([]string{"deploy-a"}))
	})
})
