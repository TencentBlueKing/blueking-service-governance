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

package frameworkcfgtmpl

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("Template validation", func() {
	DescribeTable("ResolveGroup",
		func(framework, language string, expected Group, format appcfg.FileFormat) {
			group, gotFormat, err := ResolveGroup(framework, language)
			Expect(err).NotTo(HaveOccurred())
			Expect(group).To(Equal(expected))
			Expect(gotFormat).To(Equal(format))
		},
		Entry("trpc go", "trpc", "go", Group{"trpc", "go"}, appcfg.FileFormatYAML),
		Entry("trpc cpp", "trpc", "cpp", Group{"trpc", "cpp"}, appcfg.FileFormatYAML),
		Entry("taf cpp", "taf", "cpp", Group{"taf", "cpp"}, appcfg.FileFormatTAF),
	)

	DescribeTable("ResolveGroup rejects groups without a framework config file",
		func(framework, language string) {
			_, _, err := ResolveGroup(framework, language)
			Expect(errors.Is(err, ErrInvalidTemplate)).To(BeTrue())
		},
		Entry("blank framework", "blank", "go"),
		Entry("unsupported language", "trpc", "python"),
		Entry("trpc without language", "trpc", ""),
		Entry("taf without language", "taf", ""),
		Entry("unknown framework", "unknown", "go"),
	)

	It("trims name", func() {
		def, err := Definition{Name: "  my tpl ", FileName: "trpc_go.yaml", Content: "a: 1\n"}.
			normalize(appcfg.FileFormatYAML)
		Expect(err).NotTo(HaveOccurred())
		Expect(def.Name).To(Equal("my tpl"))
	})

	DescribeTable(
		"rejects invalid definitions",
		func(def Definition, format appcfg.FileFormat, substr string) {
			_, err := def.normalize(format)
			Expect(errors.Is(err, ErrInvalidTemplate)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(substr))
		},
		Entry("empty name", Definition{Name: " ", FileName: "a.yaml", Content: "a: 1"},
			appcfg.FileFormatYAML, "name is required"),
		Entry(
			"too long name",
			Definition{Name: strings.Repeat("名", MaxNameLength+1), FileName: "a.yaml", Content: "a: 1"},
			appcfg.FileFormatYAML,
			"name exceeds",
		),
		Entry("reserved name", Definition{Name: BuiltinName, FileName: "a.yaml", Content: "a: 1"},
			appcfg.FileFormatYAML, "reserved"),
		Entry("blank content", Definition{Name: "t", FileName: "a.yaml", Content: "\n "},
			appcfg.FileFormatYAML, "content is required"),
		Entry("invalid yaml", Definition{Name: "t", FileName: "a.yaml", Content: "a: [1"},
			appcfg.FileFormatYAML, "not valid YAML"),
	)
})
