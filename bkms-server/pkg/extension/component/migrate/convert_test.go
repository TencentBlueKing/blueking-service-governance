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

package migrate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/migrate"
)

var _ = Describe("Legacy component output conversion", func() {
	It("converts sorted dot paths and individual specs into root YAML fragments", func() {
		output := `type: ComponentOutput
name: Demo
patcher:
  spec.template.spec:
    containers:
      - name: "{{ .containerName }}"
  metadata.labels:
    team: "{{ .team }}"
spec:
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: "{{ .name }}"
`
		patchers, specs, err := migrate.ConvertLegacyOutput(output)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(2))
		Expect(specs).To(HaveLen(1))

		var first, second map[string]any
		Expect(yaml.Unmarshal([]byte(patchers[0]), &first)).To(Succeed())
		Expect(yaml.Unmarshal([]byte(patchers[1]), &second)).To(Succeed())
		Expect(first).To(HaveKey("metadata"))
		Expect(second).To(HaveKey("spec"))
		Expect(patchers[0]).To(ContainSubstring("{{.team}}"))
		Expect(patchers[1]).To(ContainSubstring("{{.containerName}}"))
		Expect(specs[0]).To(ContainSubstring("{{.name}}"))
	})

	It("ignores unknown top-level fields", func() {
		patchers, specs, err := migrate.ConvertLegacyOutput(`patcher:
  spec.replicas: 2
specs:
  - apiVersion: v1
    kind: Secret
unknownField: ignored
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(1))
		Expect(specs).To(BeEmpty())
	})

	It("preserves double-quoted template scalars", func() {
		patchers, _, err := migrate.ConvertLegacyOutput(`patcher:
  metadata.annotations:
    message: "{{ .message }}"
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(1))

		rendered, err := render.RenderGoTemplate(patchers[0], map[string]any{"message": "O'Reilly"})
		Expect(err).NotTo(HaveOccurred())
		var patcher map[string]any
		Expect(yaml.Unmarshal([]byte(rendered), &patcher)).To(Succeed())
		Expect(patcher).To(HaveKeyWithValue("metadata", HaveKeyWithValue(
			"annotations",
			HaveKeyWithValue("message", "O'Reilly"),
		)))
	})

	It("converts overlapping legacy paths in deterministic order", func() {
		patchers, _, err := migrate.ConvertLegacyOutput(`patcher:
  spec:
    replicas: 1
  spec.replicas: 2
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(2))

		var first, second map[string]any
		Expect(yaml.Unmarshal([]byte(patchers[0]), &first)).To(Succeed())
		Expect(yaml.Unmarshal([]byte(patchers[1]), &second)).To(Succeed())
		Expect(first).To(HaveKeyWithValue("spec", HaveKeyWithValue("replicas", 1)))
		Expect(second).To(HaveKeyWithValue("spec", HaveKeyWithValue("replicas", 2)))
	})

	It("marks control templates for manual migration", func() {
		_, _, err := migrate.ConvertLegacyOutput(`patcher: {}
spec:
  {{- range .items }}
  - apiVersion: v1
    kind: ConfigMap
  {{- end }}
`)
		Expect(err).To(MatchError(ContainSubstring("manual migration")))
	})

	It("marks template declarations for manual migration", func() {
		_, _, err := migrate.ConvertLegacyOutput(`patcher:
  metadata.labels:
    team: '{{ $team := .team }}{{ $team }}'
`)
		Expect(err).To(MatchError(ContainSubstring("manual migration")))
	})

	It("rewrites raw actions as string literal actions", func() {
		patchers, _, err := migrate.ConvertLegacyOutput(`patcher:
  spec.template.spec:
    containers:
      - name: main
        env:
          - name: POD_NAME
            value: '{{ raw "{{.bcs.pod_name}}" }}'
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(1))
		Expect(patchers[0]).To(ContainSubstring(`{{"{{.bcs.pod_name}}"}}`))
		Expect(patchers[0]).NotTo(ContainSubstring("raw"))
		Expect(component.ValidateFragmentTemplate(patchers[0])).To(Succeed())
	})

	It("rewrites escaped raw actions as string literal actions", func() {
		patchers, _, err := migrate.ConvertLegacyOutput(`patcher:
  spec.template.spec:
    containers:
      - name: main
        env:
          - name: POD_NAME
            value: "{{ raw \"{{.bcs.pod_name}}\" }}"
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(patchers).To(HaveLen(1))
		Expect(patchers[0]).To(ContainSubstring(`{{"{{.bcs.pod_name}}"}}`))
		Expect(patchers[0]).NotTo(ContainSubstring("raw"))
		Expect(component.ValidateFragmentTemplate(patchers[0])).To(Succeed())
	})

	It("rejects non-literal raw actions", func() {
		_, _, err := migrate.ConvertLegacyOutput(`patcher:
  metadata.labels:
    value: '{{ raw .value }}'
`)
		Expect(err).To(MatchError(ContainSubstring("exactly one string literal")))
	})

	DescribeTable("treats explicit null sections as empty",
		func(output string, patcherCount, specCount int) {
			patchers, specs, err := migrate.ConvertLegacyOutput(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(patchers).To(HaveLen(patcherCount))
			Expect(specs).To(HaveLen(specCount))
		},
		Entry("null spec", `patcher:
  spec.replicas: 2
spec:
`, 1, 0),
		Entry("null patcher", `patcher:
spec:
  - apiVersion: v1
    kind: ConfigMap
`, 0, 1),
	)
})
