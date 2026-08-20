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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8smanifest "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/manifest"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("BuildNodeManifest", func() {
	It("should return raw YAML for non-Secret resources", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "test-deploy",
				"namespace": "default",
			},
			"spec": map[string]any{
				"replicas": int64(3),
			},
		}}

		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Format).To(Equal("yaml"))
		Expect(manifest.Truncated).To(BeFalse())
		Expect(manifest.Content).To(ContainSubstring("name: test-deploy"))
		Expect(manifest.Content).To(ContainSubstring("replicas: 3"))
	})

	It("should return sanitized view for masked Secret resources", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "my-secret",
				"namespace": "default",
			},
			"type": "Opaque",
			// nosec G101
			"data": map[string]any{
				"password": "c2VjcmV0",
				"token":    "dG9rZW4=",
			},
			"stringData": map[string]any{
				"plain": "secret-text",
			},
		}}

		k8smanifest.NewMasker(nil, envvartypes.SensitiveValueMask).Mask(obj)
		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Content).To(ContainSubstring(envvartypes.SensitiveValueMask))
		Expect(manifest.Content).NotTo(ContainSubstring("c2VjcmV0"))
		Expect(manifest.Content).NotTo(ContainSubstring("dG9rZW4="))
		Expect(manifest.Content).NotTo(ContainSubstring("secret-text"))
	})

	It("should remove managedFields from the manifest", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-cm",
				"namespace": "default",
				"managedFields": []any{
					map[string]any{
						"manager":   "kubectl",
						"operation": "Update",
					},
				},
			},
			"data": map[string]any{
				"key1": "value1",
			},
		}}

		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Content).NotTo(ContainSubstring("managedFields"))
		Expect(manifest.Content).To(ContainSubstring("key1: value1"))
	})

	It("should remove last-applied-configuration annotation", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-cm",
				"namespace": "default",
				"annotations": map[string]any{
					"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1","kind":"ConfigMap"}`,
					"bkms.tencent.com/last-applied-configuration":      `{"apiVersion":"v1","kind":"ConfigMap"}`,
					"custom-annotation":                                "keep-me",
				},
			},
		}}

		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Content).NotTo(ContainSubstring("last-applied-configuration"))
		Expect(manifest.Content).To(ContainSubstring("custom-annotation"))
		Expect(manifest.Content).To(ContainSubstring("keep-me"))
	})

	It("should remove annotations field entirely when last-applied-configuration is the only annotation", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-cm",
				"namespace": "default",
				"annotations": map[string]any{
					"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1"}`,
				},
			},
		}}

		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Content).NotTo(ContainSubstring("annotations"))
	})

	It("should truncate oversized manifests exceeding 1MB limit", func() {
		// 构建一个超过 1MB 的 ConfigMap
		largeValue := strings.Repeat("a", maxManifestSize+1)
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "large-cm",
				"namespace": "default",
			},
			"data": map[string]any{
				"large-key": largeValue,
			},
		}}

		manifest, err := BuildNodeManifest(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Truncated).To(BeTrue())
		Expect(manifest.Content).To(ContainSubstring("too large"))
	})
})
