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

package overview

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

var _ = Describe("autoscaling helpers", func() {
	Describe("queryAutoscalingStatusForTarget", func() {
		var (
			ctx    context.Context
			sem    *semaphore.Weighted
			client *gpa.ClusterClient
			target autoscalingTarget
		)

		BeforeEach(func() {
			ctx = testutil.NewDefaultTenantContext()
			sem = semaphore.NewWeighted(maxConcurrentK8sRequests)
			client = &gpa.ClusterClient{}
			target = autoscalingTarget{envName: "prod", namespace: "ns-1", crName: "gpa-app"}
		})

		It("maps CR status fields on success", func() {
			mockey.PatchConvey("gpa get succeeds", GinkgoT(), func() {
				mockey.Mock((*gpa.ClusterClient).GetStatus).Return(&gpa.GPAStatus{
					CurrentReplicas: 3,
					DesiredReplicas: 5,
					LastScaleTime:   "2026-08-11T10:00:00Z",
					Phase:           "Active",
					StatusMessage:   "scaling in progress",
				}, nil).Build()

				status := queryAutoscalingStatusForTarget(ctx, sem, client, "cls-1", target)
				Expect(status).NotTo(BeNil())
				Expect(status.CurrentReplicas).To(Equal(int32(3)))
				Expect(status.DesiredReplicas).To(Equal(int32(5)))
				Expect(status.LastScaleTime).To(Equal("2026-08-11T10:00:00Z"))
				Expect(status.Phase).To(Equal("Active"))
				Expect(status.StatusMessage).To(Equal("scaling in progress"))
			})
		})

		It("returns nil when the CR does not exist in cluster", func() {
			mockey.PatchConvey("gpa CR not found", GinkgoT(), func() {
				mockey.Mock((*gpa.ClusterClient).GetStatus).Return(nil, gpa.ErrCRNotFound).Build()

				Expect(queryAutoscalingStatusForTarget(ctx, sem, client, "cls-1", target)).To(BeNil())
			})
		})

		It("returns nil when the cluster query fails", func() {
			mockey.PatchConvey("gpa get fails", GinkgoT(), func() {
				mockey.Mock((*gpa.ClusterClient).GetStatus).Return(nil, errors.New("cluster unreachable")).Build()

				Expect(queryAutoscalingStatusForTarget(ctx, sem, client, "cls-1", target)).To(BeNil())
			})
		})
	})

	Describe("groupAutoscalingTargetsByCluster", func() {
		newEnv := func(name, clusterID, namespace string) envmodel.Environment {
			return envmodel.Environment{
				Name:    name,
				Cluster: envmodel.BizCluster{ClusterID: clusterID, Namespace: namespace},
			}
		}
		newRow := func(envName string, autoscaling *AutoscalingInfo) EnvRow {
			return EnvRow{EnvName: envName, Autoscaling: autoscaling}
		}

		It("groups enabled targets by cluster with namespace and CR name", func() {
			envs := []envmodel.Environment{
				newEnv("dev", "cls-1", "ns-dev"),
				newEnv("test", "cls-1", "ns-test"),
				newEnv("prod", "cls-2", "ns-prod"),
			}
			rows := []EnvRow{
				newRow("dev", &AutoscalingInfo{Enabled: true, CRName: "gpa-dev"}),
				newRow("test", &AutoscalingInfo{Enabled: true, CRName: "gpa-test"}),
				newRow("prod", &AutoscalingInfo{Enabled: true, CRName: "gpa-prod"}),
			}

			byCluster := groupAutoscalingTargetsByCluster(envs, rows)
			Expect(byCluster).To(HaveLen(2))
			Expect(byCluster["cls-1"]).To(ConsistOf(
				autoscalingTarget{envName: "dev", namespace: "ns-dev", crName: "gpa-dev"},
				autoscalingTarget{envName: "test", namespace: "ns-test", crName: "gpa-test"},
			))
			Expect(byCluster["cls-2"]).To(ConsistOf(
				autoscalingTarget{envName: "prod", namespace: "ns-prod", crName: "gpa-prod"},
			))
		})

		It("skips rows without an enabled gpa config", func() {
			envs := []envmodel.Environment{
				newEnv("no-config", "cls-1", "ns-1"),
				newEnv("disabled", "cls-1", "ns-2"),
				newEnv("no-cr-name", "cls-1", "ns-3"),
			}
			rows := []EnvRow{
				newRow("no-config", nil),
				newRow("disabled", &AutoscalingInfo{Enabled: false, CRName: "gpa-disabled"}),
				newRow("no-cr-name", &AutoscalingInfo{Enabled: true}),
			}

			Expect(groupAutoscalingTargetsByCluster(envs, rows)).To(BeEmpty())
		})

		It("skips rows whose env is missing or has no cluster", func() {
			envs := []envmodel.Environment{newEnv("no-cluster", "", "ns-1")}
			rows := []EnvRow{
				newRow("no-cluster", &AutoscalingInfo{Enabled: true, CRName: "gpa-a"}),
				newRow("not-tracked", &AutoscalingInfo{Enabled: true, CRName: "gpa-b"}),
			}

			Expect(groupAutoscalingTargetsByCluster(envs, rows)).To(BeEmpty())
		})
	})
})
