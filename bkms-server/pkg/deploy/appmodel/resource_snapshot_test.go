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

package appmodel_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("resource snapshot manifest YAML helpers", func() {
	Context("NewResourceSnapshot", func() {
		It("masks sensitive env values in workload manifests without mutating the input object", func() {
			obj := unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata":   map[string]any{"name": "deploy1"},
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"containers": []any{
									map[string]any{
										"name": "main",
										"env": []any{
											map[string]any{"name": "SECRET_TOKEN", "value": "super-secret"},
											map[string]any{"name": "PUBLIC_VALUE", "value": "public-value"},
										},
									},
								},
							},
						},
					},
				},
			}

			snapshot, err := appmodel.NewResourceSnapshot(
				obj,
				map[string]string{"SECRET_TOKEN": "super-secret"},
				"app-1",
				bson.NewObjectID(),
				time.Now(),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(snapshot.Manifest).To(ContainSubstring(envvartypes.SensitiveValueMask))
			Expect(snapshot.Manifest).To(ContainSubstring("public-value"))
			Expect(snapshot.Manifest).NotTo(ContainSubstring("super-secret"))
			Expect(nestedEnvValueForSnapshotTest(obj.Object, "containers", 0, 0)).To(Equal("super-secret"))
		})

		It("masks sensitive ConfigMap entries by env var name", func() {
			obj := unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata":   map[string]any{"name": "cm1"},
					"data": map[string]any{
						"SECRET_TOKEN": "plain-secret",
						"PUBLIC_VALUE": "public",
					},
					"binaryData": map[string]any{
						"SECRET_TOKEN": "cGxhaW4tc2VjcmV0",
						"PUBLIC_VALUE": "cHVibGlj",
					},
				},
			}

			snapshot, err := appmodel.NewResourceSnapshot(
				obj,
				map[string]string{"SECRET_TOKEN": "plain-secret"},
				"app-1",
				bson.NewObjectID(),
				time.Now(),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(snapshot.Manifest).To(ContainSubstring(envvartypes.SensitiveValueMask))
			Expect(snapshot.Manifest).To(ContainSubstring("PUBLIC_VALUE"))
			Expect(snapshot.Manifest).To(ContainSubstring("public"))
			Expect(snapshot.Manifest).NotTo(ContainSubstring("plain-secret"))
			Expect(snapshot.Manifest).NotTo(ContainSubstring("cGxhaW4tc2VjcmV0"))
		})

		It("masks all Secret payload values even without sensitive env var definitions", func() {
			obj := unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata":   map[string]any{"name": "secret1"},
					"data": map[string]any{
						"password": "c2VjcmV0",
					},
					"stringData": map[string]any{
						"token": "plain-token",
					},
				},
			}

			snapshot, err := appmodel.NewResourceSnapshot(
				obj,
				nil,
				"app-1",
				bson.NewObjectID(),
				time.Now(),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(snapshot.Manifest).To(ContainSubstring(envvartypes.SensitiveValueMask))
			Expect(snapshot.Manifest).NotTo(ContainSubstring("c2VjcmV0"))
			Expect(snapshot.Manifest).NotTo(ContainSubstring("plain-token"))
		})
	})

	Context("UnstructuredToYaml", func() {
		It("serializes a ConfigMap Unstructured without truncation", func() {
			obj := unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata":   map[string]any{"name": "cm1"},
				},
			}
			out, trunc, err := appmodel.UnstructuredToYaml(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(trunc).To(BeFalse())
			Expect(out).To(ContainSubstring("cm1"))
		})

		It("truncates very large manifests", func() {
			hugeValue := make([]byte, 6<<20)
			for i := range hugeValue {
				hugeValue[i] = 'a'
			}
			obj := unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata":   map[string]any{"name": "cm-large"},
					"data": map[string]any{
						"blob": string(hugeValue),
					},
				},
			}
			out, trunc, err := appmodel.UnstructuredToYaml(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(trunc).To(BeTrue())
			Expect(out).To(ContainSubstring("truncated"))
		})
	})
})

func nestedEnvValueForSnapshotTest(obj map[string]any, field string, containerIndex, envIndex int) any {
	podSpec, found, err := unstructured.NestedMap(obj, "spec", "template", "spec")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	containers, found, err := unstructured.NestedSlice(podSpec, field)
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	container := containers[containerIndex].(map[string]any)
	envList, found, err := unstructured.NestedSlice(container, "env")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	return envList[envIndex].(map[string]any)["value"]
}
