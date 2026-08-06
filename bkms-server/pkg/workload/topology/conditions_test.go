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

package topology

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ExtractConditions", func() {
	It("should extract conditions from a resource with multiple conditions", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"type":               "Available",
						"status":             "True",
						"reason":             "MinimumReplicasAvailable",
						"message":            "Deployment has minimum availability.",
						"lastTransitionTime": "2024-01-15T10:30:00Z",
					},
					map[string]any{
						"type":               "Progressing",
						"status":             "True",
						"reason":             "NewReplicaSetAvailable",
						"message":            "ReplicaSet has successfully progressed.",
						"lastTransitionTime": "2024-01-15T10:25:00Z",
					},
				},
			},
		}}

		conditions := ExtractConditions(obj)
		Expect(conditions).To(HaveLen(2))
		Expect(conditions[0].Type).To(Equal("Available"))
		Expect(conditions[0].Status).To(Equal("True"))
		Expect(conditions[0].Reason).To(Equal("MinimumReplicasAvailable"))
		Expect(conditions[0].Message).To(Equal("Deployment has minimum availability."))
		Expect(conditions[0].LastTransitionTime).To(Equal("2024-01-15T10:30:00Z"))
		Expect(conditions[1].Type).To(Equal("Progressing"))
	})

	DescribeTable("should return nil when conditions are absent or empty",
		func(raw map[string]any) {
			Expect(ExtractConditions(&unstructured.Unstructured{Object: raw})).To(BeNil())
		},
		Entry("status field is missing entirely", map[string]any{"spec": map[string]any{}}),
		Entry("status.conditions field is missing", map[string]any{"status": map[string]any{}}),
		Entry("conditions is an empty slice", map[string]any{
			"status": map[string]any{
				"conditions": []any{},
			},
		}),
	)

	It("should skip non-map entries in conditions slice", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"status": map[string]any{
				"conditions": []any{
					"not-a-map",
					map[string]any{
						"type":   "Ready",
						"status": "True",
					},
					int64(42),
				},
			},
		}}
		conditions := ExtractConditions(obj)
		Expect(conditions).To(HaveLen(1))
		Expect(conditions[0].Type).To(Equal("Ready"))
		Expect(conditions[0].Status).To(Equal("True"))
	})

	It("should handle conditions with missing optional fields gracefully", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"type":   "Ready",
						"status": "False",
					},
				},
			},
		}}
		conditions := ExtractConditions(obj)
		Expect(conditions).To(HaveLen(1))
		Expect(conditions[0].Type).To(Equal("Ready"))
		Expect(conditions[0].Status).To(Equal("False"))
		Expect(conditions[0].Reason).To(BeEmpty())
		Expect(conditions[0].Message).To(BeEmpty())
		Expect(conditions[0].LastTransitionTime).To(BeEmpty())
	})
})
