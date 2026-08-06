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

package postrenderer

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("PostRenderer", func() {
	Describe("NewLanePostRenderer", func() {
		Context("when labels is nil", func() {
			It("should return nil", func() {
				renderer := NewLanePostRenderer(nil)
				Expect(renderer).To(BeNil())
			})
		})

		Context("when labels is an empty map", func() {
			It("should return nil", func() {
				renderer := NewLanePostRenderer(map[string]string{})
				Expect(renderer).To(BeNil())
			})
		})

		Context("when labels is non-empty", func() {
			It("should return a non-nil PostRenderer", func() {
				labels := map[string]string{"lane": "test"}
				renderer := NewLanePostRenderer(labels)
				Expect(renderer).NotTo(BeNil())
			})
		})
	})

	Describe("Run", func() {
		Context("nil or empty-labels receiver", func() {
			It("should return input as-is", func() {
				var r *LanePostRenderer
				input := bytes.NewBufferString("apiVersion: v1\nkind: Service\n")
				output, err := r.Run(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(Equal(input))
			})
		})

		Context("empty input", func() {
			It("should output empty content", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "test"})
				input := bytes.NewBufferString("")
				output, err := r.Run(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.TrimSpace(output.String())).To(BeEmpty())
			})
		})

		Context("invalid YAML input", func() {
			It("should return an error", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "test"})
				input := bytes.NewBufferString("invalid: yaml: content: [broken")
				_, err := r.Run(input)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("decode manifest document"))
			})
		})

		Context("non-target resource (Service)", func() {
			It("should not inject lane labels", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "test"})

				input := `apiVersion: v1
kind: Service
metadata:
  name: my-svc
spec:
  ports:
    - port: 80
`
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				doc := unmarshalYAML(output.Bytes())
				spec, ok := doc["spec"].(map[string]any)
				Expect(ok).To(BeTrue())

				// Service should not have matchLabels injected
				if selector, ok := spec["selector"].(map[string]any); ok {
					if matchLabels, ok := selector["matchLabels"].(map[string]any); ok {
						Expect(matchLabels).NotTo(HaveKey("lane"))
					}
				}
			})
		})

		Context("all target Kind types", func() {
			targetKinds := []string{"Deployment", "StatefulSet", "GameDeployment", "GameStatefulSet"}

			for _, kind := range targetKinds {
				It("should inject lane labels for "+kind, func() {
					r := NewLanePostRenderer(map[string]string{"lane": "canary"})
					input := buildWorkloadYAML(kind, "test-app")
					output, err := r.Run(bytes.NewBufferString(input))
					Expect(err).NotTo(HaveOccurred())

					doc := unmarshalYAML(output.Bytes())

					matchLabels := getNestedMap(doc, "spec", "selector", "matchLabels")
					Expect(matchLabels).To(HaveKeyWithValue("lane", "canary"))

					templateLabels := getNestedMap(doc, "spec", "template", "metadata", "labels")
					Expect(templateLabels).To(HaveKeyWithValue("lane", "canary"))
				})
			}
		})

		Context("multi-document mixed resources", func() {
			It("should inject labels only for target resources and leave others unchanged", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "blue"})

				input := `apiVersion: v1
kind: Service
metadata:
  name: my-svc
spec:
  ports:
    - port: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
spec:
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: web
          image: nginx
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-sts
spec:
  selector:
    matchLabels:
      app: db
  template:
    metadata:
      labels:
        app: db
    spec:
      containers:
        - name: db
          image: postgres
`
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				docs := splitYAMLDocs(output.String())
				Expect(docs).To(HaveLen(4))

				// doc 0: Service — should not have labels injected
				assertNoLaneLabel(docs[0])
				// doc 1: Deployment — should have labels injected
				assertHasLaneLabel(docs[1], "lane", "blue")
				// doc 2: ConfigMap — should not have labels injected
				assertNoLaneLabel(docs[2])
				// doc 3: StatefulSet — should have labels injected
				assertHasLaneLabel(docs[3], "lane", "blue")
			})
		})

		Context("multi-document with empty documents", func() {
			It("should skip empty documents", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "test"})

				input := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
---
---
apiVersion: v1
kind: Service
metadata:
  name: svc
`
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				docs := splitYAMLDocs(output.String())
				Expect(docs).To(HaveLen(2))
			})
		})

		Context("multiple labels injection", func() {
			It("should inject all labels at once", func() {
				labels := map[string]string{
					"traffic.lane":    "gray",
					"traffic.version": "v2",
					"env":             "staging",
				}
				r := NewLanePostRenderer(labels)
				input := buildWorkloadYAML("Deployment", "multi-label-app")
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				doc := unmarshalYAML(output.Bytes())

				matchLabels := getNestedMap(doc, "spec", "selector", "matchLabels")
				for k, v := range labels {
					Expect(matchLabels).To(HaveKeyWithValue(k, v))
				}

				templateLabels := getNestedMap(doc, "spec", "template", "metadata", "labels")
				for k, v := range labels {
					Expect(templateLabels).To(HaveKeyWithValue(k, v))
				}
			})
		})

		Context("Deployment missing spec field", func() {
			It("should auto-create nested structure and inject labels", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "test"})

				input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: minimal
`
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				doc := unmarshalYAML(output.Bytes())

				matchLabels := getNestedMap(doc, "spec", "selector", "matchLabels")
				Expect(matchLabels).To(HaveKeyWithValue("lane", "test"))

				templateLabels := getNestedMap(doc, "spec", "template", "metadata", "labels")
				Expect(templateLabels).To(HaveKeyWithValue("lane", "test"))
			})
		})

		Context("merging with existing labels", func() {
			It("should preserve existing labels and append lane labels", func() {
				r := NewLanePostRenderer(map[string]string{"lane": "gray"})

				input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
        version: v1
    spec:
      containers:
        - name: web
          image: nginx
`
				output, err := r.Run(bytes.NewBufferString(input))
				Expect(err).NotTo(HaveOccurred())

				doc := unmarshalYAML(output.Bytes())

				matchLabels := getNestedMap(doc, "spec", "selector", "matchLabels")
				Expect(matchLabels).To(HaveKeyWithValue("lane", "gray"))
				Expect(matchLabels).To(HaveKeyWithValue("app", "web"))

				templateLabels := getNestedMap(doc, "spec", "template", "metadata", "labels")
				Expect(templateLabels).To(HaveKeyWithValue("lane", "gray"))
				Expect(templateLabels).To(HaveKeyWithValue("app", "web"))
				Expect(templateLabels).To(HaveKeyWithValue("version", "v1"))
			})
		})
	})

	Describe("ensureMap", func() {
		Context("when key does not exist", func() {
			It("should create and return a new map", func() {
				parent := map[string]any{}
				result := ensureMap(parent, "child")
				Expect(result).NotTo(BeNil())
				Expect(result).To(BeAssignableToTypeOf(map[string]any{}))
				Expect(parent["child"]).To(Equal(result))
			})
		})

		Context("when key already exists and is a map type", func() {
			It("should return the existing map directly", func() {
				existing := map[string]any{"foo": "bar"}
				parent := map[string]any{"child": existing}
				result := ensureMap(parent, "child")
				Expect(result).To(Equal(existing))
				Expect(result["foo"]).To(Equal("bar"))
			})
		})

		Context("when key already exists but is not a map type", func() {
			It("should overwrite with a new map", func() {
				parent := map[string]any{"child": "not-a-map"}
				result := ensureMap(parent, "child")
				Expect(result).NotTo(BeNil())
				Expect(result).To(BeAssignableToTypeOf(map[string]any{}))
				Expect(parent["child"]).To(Equal(result))
			})
		})
	})
})

// ==================== Helper Functions ====================

// buildWorkloadYAML builds a standard workload YAML
func buildWorkloadYAML(kind, name string) string {
	return `apiVersion: apps/v1
kind: ` + kind + `
metadata:
  name: ` + name + `
spec:
  selector:
    matchLabels:
      app: ` + name + `
  template:
    metadata:
      labels:
        app: ` + name + `
    spec:
      containers:
        - name: main
          image: nginx
`
}

// unmarshalYAML unmarshals YAML bytes into a map
func unmarshalYAML(data []byte) map[string]any {
	var doc map[string]any
	ExpectWithOffset(1, yaml.Unmarshal(data, &doc)).To(Succeed())
	return doc
}

// splitYAMLDocs splits multi-document YAML by "---" into individual documents
func splitYAMLDocs(yamlStr string) []map[string]any {
	const separator = "---"
	parts := strings.Split(yamlStr, separator)
	var docs []map[string]any
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		var doc map[string]any
		ExpectWithOffset(1, yaml.Unmarshal([]byte(trimmed), &doc)).To(Succeed())
		if doc != nil {
			docs = append(docs, doc)
		}
	}
	return docs
}

// getNestedMap retrieves a nested map[string]any by key path
func getNestedMap(doc map[string]any, keys ...string) map[string]any {
	current := doc
	for _, key := range keys {
		val, ok := current[key]
		ExpectWithOffset(1, ok).To(BeTrue(), "key %q does not exist", key)
		m, ok := val.(map[string]any)
		ExpectWithOffset(1, ok).To(BeTrue(), "value of key %q is not a map type: %T", key, val)
		current = m
	}
	return current
}

// assertHasLaneLabel asserts that the document contains the specified lane label
func assertHasLaneLabel(doc map[string]any, labelKey, labelValue string) {
	matchLabels := getNestedMap(doc, "spec", "selector", "matchLabels")
	ExpectWithOffset(1, matchLabels).To(HaveKeyWithValue(labelKey, labelValue))

	templateLabels := getNestedMap(doc, "spec", "template", "metadata", "labels")
	ExpectWithOffset(1, templateLabels).To(HaveKeyWithValue(labelKey, labelValue))
}

// assertNoLaneLabel asserts that a non-target resource does not contain lane labels
func assertNoLaneLabel(doc map[string]any) {
	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		return
	}
	selector, ok := spec["selector"].(map[string]any)
	if !ok {
		return
	}
	matchLabels, ok := selector["matchLabels"].(map[string]any)
	if !ok {
		return
	}
	ExpectWithOffset(1, matchLabels).NotTo(HaveKey("lane"))
}
