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

package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
)

var _ = Describe("Render", func() {
	envVars := map[string]string{
		"BKMS_APP_NAME":      "my-app",
		"BKMS_ENV_NAMESPACE": "default",
		"MY_REGION":          "cn",
	}

	It("should return as-is when there is no ${{ placeholder", func() {
		got, err := render.New(render.SetEnvContext(envVars)).Render("hello world")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("hello world"))
	})

	It("should not render bare variable without namespace", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			"App: ${{BKMS_APP_NAME}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("App: "))
	})

	It("should render env-prefixed variable", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			"App: ${{env.BKMS_APP_NAME}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("App: my-app"))
	})

	It("should render multiple variables", func() {
		got, err := render.New(render.SetEnvContext(envVars)).Render(
			"${{env.BKMS_APP_NAME}} in ${{env.BKMS_ENV_NAMESPACE}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("my-app in default"))
	})

	It("should render with spaces inside braces", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			"${{ env.BKMS_APP_NAME }}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("my-app"))
	})

	It("should render env prefix with spaces inside braces", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			"${{ env.BKMS_APP_NAME }}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("my-app"))
	})

	It("should render build namespace only", func() {
		buildGot, err := render.New(render.SetBkmsContext(map[string]string{"CHART_VERSION": "1.0.0"})).
			Render("${{bkms.CHART_VERSION}}")
		Expect(err).NotTo(HaveOccurred())
		rootGot, err := render.New(render.SetBkmsContext(map[string]string{"CHART_VERSION": "1.0.0"})).
			Render("${{CHART_VERSION}}")
		Expect(err).NotTo(HaveOccurred())
		Expect(buildGot).To(Equal("1.0.0"))
		Expect(rootGot).To(Equal(""))
	})

	It("should keep env and build namespaces separate", func() {
		got, err := render.New(
			render.SetEnvContext(map[string]string{"SHARED": "from-env"}),
			render.SetBkmsContext(map[string]string{"SHARED": "from-build", "BUILD_ONLY": "yes"}),
		).Render(
			"env=${{env.SHARED}} build=${{bkms.SHARED}} only=${{bkms.BUILD_ONLY}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("env=from-env build=from-build only=yes"))
	})

	It("should keep same key in different namespaces without root conflict", func() {
		got, err := render.New(
			render.SetEnvContext(map[string]string{"SHARED": "from-env"}),
			render.SetBkmsContext(map[string]string{"SHARED": "from-build"}),
		).Render("env=${{env.SHARED}} build=${{bkms.SHARED}} root=${{SHARED}}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("env=from-env build=from-build root="))
	})

	It("should preserve legacy {{ }} format untouched", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			"{{.BKMS.ENV.FOO}} and ${{env.BKMS_APP_NAME}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("{{.BKMS.ENV.FOO}} and my-app"))
	})

	It("should preserve legacy raw function syntax untouched", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			`{{ raw "{{.bcs.pod_name}}" }} and ${{env.BKMS_APP_NAME}}`,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(`{{ raw "{{.bcs.pod_name}}" }} and my-app`))
	})

	It("should preserve legacy raw function syntax with escaped quotes untouched", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			`{{ raw \"{{.bcs.pod_name}}\" }} and ${{env.BKMS_APP_NAME}}`,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(`{{ raw \"{{.bcs.pod_name}}\" }} and my-app`))
	})

	It("should render undefined variable as empty", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"OTHER": "value"})).Render(
			"${{env.UNDEFINED_VAR}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(""))
	})

	It("should render non-string any value", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{
			"PORT":    "8080",
			"ENABLED": "true",
		})).Render(
			"port=${{env.PORT}} enabled=${{env.ENABLED}}",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("port=8080 enabled=true"))
	})

	It("should preserve if control structure syntax as plain text", func() {
		input := `{% if BKMS_APP_NAME %}${{env.BKMS_APP_NAME}}{% endif %}`
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			input,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(`{% if BKMS_APP_NAME %}my-app{% endif %}`))
	})

	It("should preserve for loop syntax as plain text", func() {
		input := `{% for item in items %}${{env.BKMS_APP_NAME}}{% endfor %}`
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			input,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(`{% for item in items %}my-app{% endfor %}`))
	})

	It("should preserve comment syntax as plain text", func() {
		input := `{# this is a comment #} ${{env.BKMS_APP_NAME}}`
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).Render(
			input,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(`{# this is a comment #} my-app`))
	})

	It("should render with nil data as empty substitutions", func() {
		got, err := render.New(render.SetEnvContext(map[string]string{"BKMS_APP_NAME": "my-app"})).
			Render("${{MISSING}}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(""))
	})
})

var _ = Describe("ExtractVars", func() {
	It("should return nil when text has no variable marker", func() {
		got, err := render.ExtractVars("image: my-static-image:latest")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("should return nil for empty text", func() {
		got, err := render.ExtractVars("")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("should extract a single namespaced variable", func() {
		got, err := render.ExtractVars("image: ${{ bkms.IMAGE }}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{
			"bkms": {"IMAGE": {}},
		}))
	})

	It("should extract variables across multiple contexts", func() {
		got, err := render.ExtractVars("${{ bkms.IMAGE }} run by ${{ env.APP_NAME }}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{
			"bkms": {"IMAGE": {}},
			"env":  {"APP_NAME": {}},
		}))
	})

	It("should group multiple variables under the same context", func() {
		got, err := render.ExtractVars(`registry: ${{ bkms.IMAGE_REGISTRY }}
repository: ${{ bkms.IMAGE_REPOSITORY }}
tag: ${{ bkms.IMAGE_TAG }}`)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{
			"bkms": {
				"IMAGE_REGISTRY":   {},
				"IMAGE_REPOSITORY": {},
				"IMAGE_TAG":        {},
			},
		}))
	})

	It("should deduplicate repeated variable references", func() {
		got, err := render.ExtractVars("${{ bkms.IMAGE }} and again ${{ bkms.IMAGE }}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{
			"bkms": {"IMAGE": {}},
		}))
	})

	It("should extract variables embedded in nested YAML", func() {
		got, err := render.ExtractVars(`global:
  workload:
    - name: web
      image: ${{ bkms.IMAGE }}`)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{
			"bkms": {"IMAGE": {}},
		}))
	})

	It("should ignore bare variables without a namespace", func() {
		got, err := render.ExtractVars("${{ IMAGE }}")
		Expect(err).NotTo(HaveOccurred())
		// Bare name has no context, so it is not collected; result is empty (but non-nil).
		Expect(got).To(Equal(render.VarsSet{}))
	})

	It("should ignore expressions nested deeper than one level", func() {
		got, err := render.ExtractVars("${{ bkms.image.tag }}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{}))
	})

	It("should ignore filter expressions", func() {
		got, err := render.ExtractVars("${{ bkms.IMAGE | upper }}")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(render.VarsSet{}))
	})

	It("should return an error for malformed template syntax", func() {
		_, err := render.ExtractVars("${{ bkms. }}")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("RenderGoTemplate", func() {
	It("should return as-is when there is no placeholder", func() {
		got, err := render.RenderGoTemplate("hello world", map[string]any{
			"BKMS": map[string]any{"ENV": map[string]any{"FOO": "bar"}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("hello world"))
	})

	It("should render env var placeholder", func() {
		got, err := render.RenderGoTemplate("Welcome {{ .BKMS.ENV.FOO_TITLE }}", map[string]any{
			"BKMS": map[string]any{"ENV": map[string]any{"FOO_TITLE": "Production"}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("Welcome Production"))
	})

	It("should render builtin property", func() {
		got, err := render.RenderGoTemplate("App: {{ .bkmsAppName }}", map[string]any{
			"bkmsAppName": "my-service",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("App: my-service"))
	})

	It("should preserve template syntax with raw function", func() {
		got, err := render.RenderGoTemplate(`{{ raw "{{.bcs.pod_name}}" }}`, map[string]any{
			"BKMS": map[string]any{"ENV": map[string]any{}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("{{.bcs.pod_name}}"))
	})

	It("should normalize escaped quotes", func() {
		got, err := render.RenderGoTemplate(`{{ raw \"{{.bcs.pod_name}}\" }}`, map[string]any{
			"BKMS": map[string]any{"ENV": map[string]any{}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("{{.bcs.pod_name}}"))
	})

	It("should work with nil tmplData for no-placeholder string", func() {
		got, err := render.RenderGoTemplate("no template here", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("no template here"))
	})
})
