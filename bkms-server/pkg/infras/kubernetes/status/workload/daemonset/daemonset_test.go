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

package daemonset

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("DaemonSet Parse", func() {
	Context("when manifest is nil", func() {
		It("returns Unknown", func() {
			Expect(Parse(nil).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when Degraded condition is True", func() {
		It("returns Degraded", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type":    "Degraded",
							"status":  "True",
							"reason":  "TestReason",
							"message": "test degraded",
						},
					},
				},
			}
			result := Parse(manifest)
			Expect(result.Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when observedGeneration has not caught up", func() {
		It("returns Progressing", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(2)},
				"status": map[string]any{
					"observedGeneration": int64(1),
				},
			}
			result := Parse(manifest)
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
			Expect(result.Message).To(ContainSubstring("observedGeneration"))
		})
	})

	Context("when pods are not consistent", func() {
		It("returns Progressing with reason", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(1)},
				"status": map[string]any{
					"observedGeneration":     int64(1),
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(2),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(3),
				},
			}
			result := Parse(manifest)
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
			Expect(result.Message).To(ContainSubstring("pods are not consistent"))
		})
	})

	Context("when some pods are unavailable", func() {
		It("returns Progressing", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(1)},
				"status": map[string]any{
					"observedGeneration":     int64(1),
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(3),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(3),
					"numberUnavailable":      int64(1),
				},
			}
			result := Parse(manifest)
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
			Expect(result.Message).To(ContainSubstring("unavailable"))
		})
	})

	Context("when all checks pass", func() {
		It("returns Available", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(1)},
				"status": map[string]any{
					"observedGeneration":     int64(1),
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(3),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(3),
					"numberUnavailable":      int64(0),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Available))
		})
	})
})

var _ = Describe("arePodsConsistent", func() {
	Context("when all pod counts match", func() {
		It("returns consistent", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(3),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(3),
				},
			}
			consistent, reason := arePodsConsistent(manifest)
			Expect(consistent).To(BeTrue())
			Expect(reason).To(BeEmpty())
		})
	})

	Context("when desired != current", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(2),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(3),
				},
			}
			consistent, reason := arePodsConsistent(manifest)
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("desiredNumberScheduled != currentNumberScheduled"))
		})
	})

	Context("when desired != updated", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(3),
					"updatedNumberScheduled": int64(2),
					"numberReady":            int64(3),
				},
			}
			consistent, reason := arePodsConsistent(manifest)
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("desiredNumberScheduled != updatedNumberScheduled"))
		})
	})

	Context("when desired != ready", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"desiredNumberScheduled": int64(3),
					"currentNumberScheduled": int64(3),
					"updatedNumberScheduled": int64(3),
					"numberReady":            int64(1),
				},
			}
			consistent, reason := arePodsConsistent(manifest)
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("desiredNumberScheduled != numberReady"))
		})
	})
})
