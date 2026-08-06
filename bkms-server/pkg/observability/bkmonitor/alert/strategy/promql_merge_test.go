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

package strategy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("PromQL merge helpers", func() {
	newStrategy := func(code string) *AlertStrategy {
		return &AlertStrategy{
			StrategyCode: code,
			TriggerCondition: TriggerCondition{
				Count:       3,
				CheckWindow: 5,
			},
		}
	}

	Describe("buildMergedAlertPromQL", func() {
		DescribeTable("builds promql for merged targets",
			func(strategyCode string, scope mergedTargetScope, wantContains, wantNotContains []string) {
				promql := buildMergedAlertPromQL(newStrategy(strategyCode), scope)

				for _, expected := range wantContains {
					Expect(promql).To(ContainSubstring(expected))
				}
				for _, unexpected := range wantNotContains {
					Expect(promql).NotTo(ContainSubstring(unexpected))
				}
			},
			Entry("uses exact match for a single target",
				"cpu_request_usage_high",
				mergedTargetScope{
					ClusterIDs: []string{"BCS-K8S-100018"},
					Namespaces: []string{"ieg-bkms-pd-test"},
					Workloads:  []string{"trpc-test-app"},
				},
				[]string{
					`bcs_cluster_id="BCS-K8S-100018"`,
					`namespace="ieg-bkms-pd-test"`,
					`workload_name="trpc-test-app"`,
				},
				[]string{`=~`},
			),
			Entry("uses regex for merged namespaces within the same cluster",
				"cpu_request_usage_high",
				mergedTargetScope{
					ClusterIDs: []string{"BCS-K8S-100018"},
					Namespaces: []string{"ieg-bkms-pd-stage", "ieg-bkms-pd-test"},
					Workloads:  []string{"trpc-test-app"},
				},
				[]string{
					`namespace=~"ieg-bkms-pd-stage|ieg-bkms-pd-test"`,
					`bcs_cluster_id="BCS-K8S-100018"`,
				},
				nil,
			),
			Entry("uses regex for multi-cluster merged targets",
				"memory_limit_usage_high",
				mergedTargetScope{
					ClusterIDs: []string{"BCS-K8S-100018", "BCS-K8S-200019"},
					Namespaces: []string{"ns-stage", "ns-test"},
					Workloads:  []string{"my-app"},
				},
				[]string{
					`bcs_cluster_id=~"BCS-K8S-100018|BCS-K8S-200019"`,
					`namespace=~"ns-stage|ns-test"`,
				},
				nil,
			),
			Entry("uses pod restart specific selectors",
				"pod_restart_frequent",
				mergedTargetScope{
					ClusterIDs: []string{"BCS-K8S-100018"},
					Namespaces: []string{"ieg-bkms-pd-stage", "ieg-bkms-pd-test"},
					Workloads:  []string{"trpc-test-app"},
				},
				[]string{
					`namespace=~"ieg-bkms-pd-stage|ieg-bkms-pd-test"`,
					`kube_pod_container_status_restarts_total`,
					`pod_name=~"^trpc-test-app(-.*)?$"`,
				},
				nil,
			),
		)
	})

	Describe("mergeTargetScopes", func() {
		It("deduplicates cluster, namespace, and workload values", func() {
			targets := []remoteTargetContext{
				{
					Env: envmodel.Environment{
						Cluster: envmodel.BizCluster{
							ClusterID: "BCS-K8S-100018",
							Namespace: "ieg-bkms-pd-stage",
						},
					},
					Workloads: []string{"trpc-test-app"},
				},
				{
					Env: envmodel.Environment{
						Cluster: envmodel.BizCluster{
							ClusterID: "BCS-K8S-100018",
							Namespace: "ieg-bkms-pd-test",
						},
					},
					Workloads: []string{"trpc-test-app"},
				},
			}

			scope := mergeTargetScopes(targets)

			Expect(scope.ClusterIDs).To(Equal([]string{"BCS-K8S-100018"}))
			Expect(scope.Namespaces).To(HaveLen(2))
			Expect(scope.Workloads).To(Equal([]string{"trpc-test-app"}))
		})
	})

	Describe("exactOrRegex", func() {
		DescribeTable(
			"formats selectors",
			func(values []string, expected string) {
				Expect(exactOrRegex("namespace", values)).To(Equal(expected))
			},
			Entry("returns exact match for a single value", []string{"ns-test"}, `namespace="ns-test"`),
			Entry(
				"returns regex match for multiple values",
				[]string{"ns-stage", "ns-test"},
				`namespace=~"ns-stage|ns-test"`,
			),
			Entry(
				"quotes regex metacharacters for multiple values",
				[]string{"ns.stage", "ns+test"},
				`namespace=~"ns\.stage|ns\+test"`,
			),
		)
	})

	Describe("workloadMatcher", func() {
		DescribeTable(
			"formats workload selectors",
			func(workloads []string, expected string) {
				Expect(workloadMatcher(workloads)).To(Equal(expected))
			},
			Entry("returns exact match for a single workload", []string{"demo-app"}, `workload_name="demo-app"`),
			Entry(
				"quotes regex metacharacters for multiple workloads",
				[]string{"demo.app", "demo+worker"},
				`workload_name=~"^(demo\.app|demo\+worker)$"`,
			),
		)
	})

	Describe("buildAppScopedAlertPromQL", func() {
		It("keeps single-environment behavior backward compatible", func() {
			env := envmodel.Environment{
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-100018",
					Namespace: "ieg-bkms-pd-test",
				},
			}

			promql := buildAppScopedAlertPromQL(newStrategy("cpu_request_usage_high"), env, []string{"trpc-test-app"})

			Expect(promql).To(ContainSubstring(`bcs_cluster_id="BCS-K8S-100018"`))
			Expect(promql).To(ContainSubstring(`namespace="ieg-bkms-pd-test"`))
			Expect(promql).To(ContainSubstring(`* 100`))
			Expect(promql).NotTo(ContainSubstring(`=~`))
		})
	})
})
