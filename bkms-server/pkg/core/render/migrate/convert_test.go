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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render/migrate"
)

var _ = Describe("Convert", func() {
	DescribeTable("rewrites",
		func(input, want string) {
			got, err := migrate.Convert(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("BKMS env field", `{{ .BKMS.ENV.FOO }}`, `${{env.FOO}}`),
		Entry("builtin camelCase", `{{ .bkmsAppName }}`, `${{env.BKMS_APP_NAME}}`),
		Entry("root gonja → env", `app=${{BKMS_APP_NAME}}`, `app=${{env.BKMS_APP_NAME}}`),
		Entry("namespaced env unchanged", `${{env.X}}`, `${{env.X}}`),
		Entry("mixed legacy + root + env",
			`x={{ .bkmsAppName }} y=${{VAR}} z={{ .BKMS.ENV.E }}`,
			`x=${{env.BKMS_APP_NAME}} y=${{env.VAR}} z=${{env.E}}`),
		Entry("raw literal expanded",
			`{{ raw "{{.bcs.pod_name}}" }}`,
			`{{.bcs.pod_name}}`),
		Entry("plain text untouched", `hello world`, `hello world`),
		Entry("empty string", ``, ``),
	)

	DescribeTable("rejected inputs",
		func(input string) {
			_, err := migrate.Convert(input)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, migrate.ErrNeedsManual)).To(BeTrue(), "err=%v", err)
		},
		Entry("range control flow", `{{- range $k, $v := .labels }}x{{- end }}`),
		Entry("if branch", `{{ if .x }}y{{ end }}`),
		Entry("with branch", `{{ with .x }}y{{ end }}`),
		Entry("define template", `{{ define "x" }}y{{ end }}`),
		Entry("pipeline filter", `{{ .x | upper }}`),
		Entry("function call", `{{ printf "%s" .x }}`),
		Entry("multi-segment field", `{{ .a.b }}`),
		Entry("unknown env field", `{{ .replicas }}`),
		Entry("raw with gonja marker", `{{ raw "${{X}}" }}`),
		Entry("gonja filter in source", `${{ x | upper }}`),
		Entry("gonja test in source", `${{ x is none }}`),
		Entry("gonja conditional output", `${{ x if y else z }}`),
		Entry("gonja string literal output", `${{ "hello" }}`),
	)
})
