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

package gamedeployment

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("GameDeployment Parse", func() {
	Context("when manifest is nil", func() {
		It("returns Unknown", func() {
			result, err := Parse(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when updateStrategy is paused", func() {
		It("returns Suspended", func() {
			replicas := int32(3)
			manifest := map[string]any{
				"generation": int64(1),
				"spec": map[string]any{
					"replicas": replicas,
					"updateStrategy": map[string]any{
						"paused": true,
					},
				},
				"status": map[string]any{
					"observedGeneration": int64(1),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Suspended))
		})
	})

	Context("when replicas is 0 and status.replicas is 0", func() {
		It("returns Available", func() {
			manifest := map[string]any{
				"generation": int64(1),
				"spec": map[string]any{
					"replicas": int32(0),
					"updateStrategy": map[string]any{
						"paused": false,
					},
				},
				"status": map[string]any{
					"replicas":           int32(0),
					"observedGeneration": int64(1),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Available))
		})
	})

	Context("when observedGeneration is zero", func() {
		It("returns Progressing", func() {
			replicas := int32(3)
			manifest := map[string]any{
				"generation": int64(1),
				"spec": map[string]any{
					"replicas": replicas,
					"updateStrategy": map[string]any{
						"paused": false,
					},
				},
				"status": map[string]any{
					"replicas":           int32(3),
					"observedGeneration": int64(0),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
			Expect(result.Message).To(ContainSubstring("observed generation is empty"))
		})
	})

	Context("when observedGeneration < generation", func() {
		It("returns Progressing", func() {
			replicas := int32(3)
			manifest := map[string]any{
				"generation": int64(2),
				"spec": map[string]any{
					"replicas": replicas,
					"updateStrategy": map[string]any{
						"paused": false,
					},
				},
				"status": map[string]any{
					"replicas":           int32(3),
					"observedGeneration": int64(1),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
			Expect(result.Message).To(ContainSubstring("observed generation less than desired"))
		})
	})

	Context("when replicas != status.replicas", func() {
		It("returns Progressing", func() {
			replicas := int32(3)
			manifest := map[string]any{
				"generation": int64(1),
				"spec": map[string]any{
					"replicas": replicas,
					"updateStrategy": map[string]any{
						"paused": false,
					},
				},
				"status": map[string]any{
					"replicas":           int32(2),
					"observedGeneration": int64(1),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
		})
	})

	Context("when all replicas match and generation is caught up", func() {
		It("returns Healthy", func() {
			replicas := int32(3)
			manifest := map[string]any{
				"generation": int64(1),
				"spec": map[string]any{
					"replicas": replicas,
					"updateStrategy": map[string]any{
						"paused": false,
					},
				},
				"status": map[string]any{
					"replicas":             int32(3),
					"readyReplicas":        int32(3),
					"updatedReplicas":      int32(3),
					"updatedReadyReplicas": int32(3),
					"observedGeneration":   int64(1),
				},
			}
			result, err := Parse(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Healthy))
		})
	})
})
