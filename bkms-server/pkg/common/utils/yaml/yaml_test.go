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

package yaml_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/yaml"
)

var _ = Describe("UnmarshalMultipleDocuments", func() {
	Context("when processing valid YAML data", func() {
		It("should successfully parse a single document", func() {
			yamlData := `
name: test
version: 1.0
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).NotTo(HaveOccurred())
			Expect(docs).To(HaveLen(1))
			Expect(docs[0]).To(HaveKeyWithValue("name", "test"))
			Expect(docs[0]).To(HaveKeyWithValue("version", 1.0))
		})

		It("should successfully parse multiple documents", func() {
			yamlData := `
name: first
version: 1.0
---
name: second
version: 2.0
---
name: third
version: 3.0
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).NotTo(HaveOccurred())
			Expect(docs).To(HaveLen(3))

			Expect(docs[0]).To(HaveKeyWithValue("name", "first"))
			Expect(docs[0]).To(HaveKeyWithValue("version", 1.0))

			Expect(docs[1]).To(HaveKeyWithValue("name", "second"))
			Expect(docs[1]).To(HaveKeyWithValue("version", 2.0))

			Expect(docs[2]).To(HaveKeyWithValue("name", "third"))
			Expect(docs[2]).To(HaveKeyWithValue("version", 3.0))
		})

		It("should handle documents with complex structures", func() {
			yamlData := `
metadata:
  name: test-app
  labels:
    app: test
    version: v1
spec:
  replicas: 3
  ports:
    - 8080
    - 9090
---
kind: Service
metadata:
  name: test-service
spec:
  selector:
    app: test
  ports:
    - port: 80
      targetPort: 8080
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).NotTo(HaveOccurred())
			Expect(docs).To(HaveLen(2))

			// 验证第一个文档
			firstDoc := docs[0]
			Expect(firstDoc).To(HaveKey("metadata"))
			Expect(firstDoc).To(HaveKey("spec"))

			metadata := firstDoc["metadata"].(map[string]any)
			Expect(metadata).To(HaveKeyWithValue("name", "test-app"))

			// 验证第二个文档
			secondDoc := docs[1]
			Expect(secondDoc).To(HaveKeyWithValue("kind", "Service"))
		})
	})

	Context("when handling edge cases", func() {
		It("should handle empty string", func() {
			docs, err := yaml.UnmarshalMultipleDocuments("")

			Expect(err).NotTo(HaveOccurred())
			Expect(docs).To(BeEmpty())
		})

		It("should handle string with only separators", func() {
			yamlData := "---\n---\n---"
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).NotTo(HaveOccurred())
			Expect(len(docs)).To(Equal(3))
		})

		It("should handle YAML with empty documents", func() {
			yamlData := `
name: first
---

---
name: second
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).NotTo(HaveOccurred())
			Expect(docs).To(HaveLen(3))
			Expect(docs[0]).To(HaveKeyWithValue("name", "first"))
			Expect(docs[2]).To(HaveKeyWithValue("name", "second"))
		})
	})

	Context("when processing invalid YAML data", func() {
		It("should return syntax error", func() {
			yamlData := `
name: test
  invalid: indentation
version: 1.0
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).To(HaveOccurred())
			Expect(docs).To(BeNil())
		})

		It("should return invalid character error", func() {
			yamlData := `
name: test
version: [invalid yaml structure
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).To(HaveOccurred())
			Expect(docs).To(BeNil())
		})

		It("should handle multi-document YAML with invalid documents", func() {
			yamlData := `
name: valid
---
invalid: [yaml
---
name: another-valid
`
			docs, err := yaml.UnmarshalMultipleDocuments(yamlData)

			Expect(err).To(HaveOccurred())
			Expect(docs).To(BeNil())
		})
	})
})
