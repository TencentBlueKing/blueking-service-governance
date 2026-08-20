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

package client

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
)

var _ = Describe("federation three-way merge patch", func() {
	It("should null omitted fields that exist in last-applied without touching server-managed fields", func() {
		original := []byte(`{
			"metadata": {"name": "web", "labels": {"app": "web", "component": "sidecar"}},
			"spec": {
				"template": {"spec": {"initContainers": [{"name": "init", "image": "busybox"}]}}
			}
		}`)
		modified := []byte(`{
			"metadata": {"name": "web", "labels": {"app": "web"}},
			"spec": {
				"template": {"spec": {"containers": [{"name": "app", "image": "nginx"}]}}
			}
		}`)
		current := []byte(`{
			"metadata": {
				"name": "web",
				"uid": "live-uid",
				"labels": {"app": "web", "component": "sidecar", "injected": "keep"}
			},
			"spec": {
				"clusterIP": "10.0.0.8",
				"template": {
					"spec": {
						"initContainers": [{"name": "init", "image": "busybox"}],
						"containers": [{"name": "app", "image": "nginx"}]
					}
				}
			},
			"status": {"replicas": 1}
		}`)

		patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current)
		Expect(err).NotTo(HaveOccurred())

		var patchMap map[string]any
		Expect(json.Unmarshal(patch, &patchMap)).To(Succeed())

		labels := patchMap["metadata"].(map[string]any)["labels"].(map[string]any)
		Expect(labels).To(HaveKey("component"))
		Expect(labels["component"]).To(BeNil())
		Expect(labels).NotTo(HaveKey("injected"))

		spec := patchMap["spec"].(map[string]any)
		Expect(spec).NotTo(HaveKey("clusterIP"))
		Expect(patchMap).NotTo(HaveKey("status"))

		initContainers := spec["template"].(map[string]any)["spec"].(map[string]any)["initContainers"]
		Expect(initContainers).To(BeNil())
	})

	It("should not delete live leftovers when last-applied is missing", func() {
		modified := []byte(`{"metadata": {"name": "web"}, "spec": {"ports": [{"port": 80}]}}`)
		current := []byte(`{
			"metadata": {"name": "web"},
			"spec": {"clusterIP": "10.0.0.8", "ports": [{"port": 80}], "initContainers": [{"name": "old"}]}
		}`)

		patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(nil, modified, current)
		Expect(err).NotTo(HaveOccurred())

		var patchMap map[string]any
		Expect(json.Unmarshal(patch, &patchMap)).To(Succeed())
		spec, _ := patchMap["spec"].(map[string]any)
		if spec != nil {
			Expect(spec).NotTo(HaveKey("clusterIP"))
			Expect(spec).NotTo(HaveKey("initContainers"))
		}
	})

	It("should copy manifest and record last-applied without nesting the annotation", func() {
		input := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": "svc",
				"annotations": map[string]any{
					"custom": "keep",
				},
			},
			"spec": map[string]any{"ports": []any{map[string]any{"port": 80}}},
		}

		desired, err := prepareFederationDesired(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(
			input["metadata"].(map[string]any)["annotations"].(map[string]any),
		).NotTo(HaveKey(lastAppliedAnnotation))

		ann := desired["metadata"].(map[string]any)["annotations"].(map[string]any)
		Expect(ann["custom"]).To(Equal("keep"))
		raw, ok := ann[lastAppliedAnnotation].(string)
		Expect(ok).To(BeTrue())
		Expect(raw).NotTo(ContainSubstring(lastAppliedAnnotation))

		var stored map[string]any
		Expect(json.Unmarshal([]byte(raw), &stored)).To(Succeed())
		storedAnn := stored["metadata"].(map[string]any)["annotations"].(map[string]any)
		Expect(storedAnn).To(HaveKeyWithValue("custom", "keep"))
		Expect(storedAnn).NotTo(HaveKey(lastAppliedAnnotation))
	})
})
