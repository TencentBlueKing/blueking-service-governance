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

package ingress

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("Ingress Parse", func() {
	Context("when manifest is nil", func() {
		It("does not panic and returns Unknown", func() {
			Expect(Parse(nil).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when spec.rules is missing", func() {
		It("returns Unknown", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"loadBalancer": map[string]any{
						"ingress": []any{
							map[string]any{"ip": "1.2.3.4"},
						},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when spec.rules is empty slice", func() {
		It("returns Unknown", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"rules": []any{},
				},
				"status": map[string]any{
					"loadBalancer": map[string]any{
						"ingress": []any{
							map[string]any{"ip": "1.2.3.4"},
						},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when status.loadBalancer.ingress has assigned ip", func() {
		It("returns Healthy", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "app.example.com"},
					},
				},
				"status": map[string]any{
					"loadBalancer": map[string]any{
						"ingress": []any{
							map[string]any{"ip": "1.2.3.4"},
						},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Healthy))
		})
	})

	Context("when status.loadBalancer.ingress has assigned hostname", func() {
		It("returns Healthy", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "app.example.com"},
					},
				},
				"status": map[string]any{
					"loadBalancer": map[string]any{
						"ingress": []any{
							map[string]any{"hostname": "lb-xxx.example.com"},
						},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Healthy))
		})
	})

	Context("when status.loadBalancer.ingress is empty slice", func() {
		It("returns Progressing", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "app.example.com"},
					},
				},
				"status": map[string]any{
					"loadBalancer": map[string]any{
						"ingress": []any{},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Progressing))
		})
	})

	Context("when status is entirely missing", func() {
		It("returns Progressing as controller has not reconciled yet", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "app.example.com"},
					},
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Progressing))
		})
	})
})
