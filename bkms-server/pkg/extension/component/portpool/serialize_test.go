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

package portpool

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("parsePortPoolFromUnstructured", func() {
	It("should parse a valid PortPool CR", func() {
		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": portPoolGroupVersion,
				"kind":       portPoolKind,
				"metadata": map[string]any{
					"name": "test-pool",
					"labels": map[string]any{
						LabelKeyWorkspaceID: "ws-1",
						LabelKeyEnvName:     "dev",
					},
				},
				"spec": map[string]any{
					"poolItems": []any{
						map[string]any{
							"itemName":        "item-0",
							"protocol":        "TCP",
							"startPort":       int64(30000),
							"endPort":         int64(30100),
							"segmentLength":   int64(1),
							"external":        "ext-0",
							"loadBalancerIDs": []any{"lb-1", "lb-2"},
						},
						map[string]any{
							"itemName":  "item-1",
							"protocol":  "UDP",
							"startPort": int64(31000),
							"endPort":   int64(31100),
						},
					},
				},
				"status": map[string]any{
					"status": "Ready",
					"poolItems": []any{
						map[string]any{
							"itemName": "item-0",
							"status":   "Ready",
							"message":  "item-0 is ready",
						},
						map[string]any{
							"itemName": "item-1",
							"status":   "NotReady",
							"message":  "item-1 is not ready",
						},
					},
				},
			},
		}

		config, err := parsePortPoolFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Name).To(Equal("test-pool"))
		Expect(config.WorkspaceID).To(Equal("ws-1"))
		Expect(config.EnvName).To(Equal("dev"))
		Expect(config.Status).To(Equal("Ready"))
		Expect(config.PoolItems).To(HaveLen(2))

		Expect(config.PoolItems[0].ItemName).To(Equal("item-0"))
		Expect(config.PoolItems[0].Protocol).To(Equal("TCP"))
		Expect(config.PoolItems[0].StartPort).To(Equal(int32(30000)))
		Expect(config.PoolItems[0].EndPort).To(Equal(int32(30100)))
		Expect(config.PoolItems[0].SegmentLength).To(Equal(int32(1)))
		Expect(config.PoolItems[0].External).To(Equal("ext-0"))
		Expect(config.PoolItems[0].LoadBalancerIDs).To(Equal([]string{"lb-1", "lb-2"}))
		Expect(config.PoolItems[0].Status).To(Equal(PoolItemStatus{Status: "Ready", Message: "item-0 is ready"}))

		Expect(config.PoolItems[1].ItemName).To(Equal("item-1"))
		Expect(config.PoolItems[1].Protocol).To(Equal("UDP"))
		Expect(config.PoolItems[1].StartPort).To(Equal(int32(31000)))
		Expect(config.PoolItems[1].EndPort).To(Equal(int32(31100)))
		Expect(config.PoolItems[1].LoadBalancerIDs).To(BeEmpty())
		Expect(
			config.PoolItems[1].Status,
		).To(Equal(PoolItemStatus{Status: "NotReady", Message: "item-1 is not ready"}))
	})

	It("should return empty PoolItems when poolItems is absent", func() {
		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": portPoolGroupVersion,
				"kind":       portPoolKind,
				"metadata": map[string]any{
					"name": "empty-pool",
					"labels": map[string]any{
						LabelKeyWorkspaceID: "ws-1",
						LabelKeyEnvName:     "dev",
					},
				},
				"spec": map[string]any{},
			},
		}

		config, err := parsePortPoolFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.PoolItems).To(BeEmpty())
	})

	It("should override status to Deleting when deletionTimestamp is set", func() {
		now := metav1.Now()
		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": portPoolGroupVersion,
				"kind":       portPoolKind,
				"metadata": map[string]any{
					"name": "deleting-pool",
					"labels": map[string]any{
						LabelKeyWorkspaceID: "ws-1",
						LabelKeyEnvName:     "dev",
					},
					"deletionTimestamp": now.Format(time.RFC3339),
				},
				"spec": map[string]any{
					"poolItems": []any{
						map[string]any{
							"itemName":  "item-0",
							"startPort": int64(30000),
							"endPort":   int64(30100),
						},
					},
				},
				"status": map[string]any{
					"status": "Ready",
				},
			},
		}

		config, err := parsePortPoolFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Status).To(Equal("Deleting"))
	})
})

var _ = Describe("parsePortPoolListFromUnstructured", func() {
	It("should parse a list of PortPool CRs", func() {
		list := &unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{
				{
					Object: map[string]any{
						"apiVersion": portPoolGroupVersion,
						"kind":       portPoolKind,
						"metadata": map[string]any{
							"name": "pool-1",
							"labels": map[string]any{
								LabelKeyWorkspaceID: "ws-1",
								LabelKeyEnvName:     "dev",
							},
						},
						"spec": map[string]any{
							"poolItems": []any{
								map[string]any{
									"itemName":  "item-0",
									"startPort": int64(30000),
									"endPort":   int64(30100),
								},
							},
						},
					},
				},
				{
					Object: map[string]any{
						"apiVersion": portPoolGroupVersion,
						"kind":       portPoolKind,
						"metadata": map[string]any{
							"name": "pool-2",
							"labels": map[string]any{
								LabelKeyWorkspaceID: "ws-1",
								LabelKeyEnvName:     "dev",
							},
						},
						"spec": map[string]any{},
					},
				},
			},
		}

		configs, err := parsePortPoolListFromUnstructured(list)
		Expect(err).NotTo(HaveOccurred())
		Expect(configs).To(HaveLen(2))
		Expect(configs[0].Name).To(Equal("pool-1"))
		Expect(configs[0].PoolItems).To(HaveLen(1))
		Expect(configs[1].Name).To(Equal("pool-2"))
		Expect(configs[1].PoolItems).To(BeEmpty())
	})
})
