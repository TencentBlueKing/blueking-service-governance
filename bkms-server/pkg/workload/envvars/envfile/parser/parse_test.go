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

package parser_test

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
)

var _ = Describe("ParseEnvFileRecords", func() {
	It("parses quoted values and structured scope metadata", func() {
		records, err := parserpkg.ParseEnvFileRecords(`
# desc: demo
# scopeType: envType
# scopeValue: development
GOOD_KEY="quoted value # stays"
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(records).To(Equal([]parserpkg.ParsedEnvVarRecord{{
			Line:               5,
			Key:                "GOOD_KEY",
			Value:              "quoted value # stays",
			Description:        "demo",
			DeclaredScopeType:  stringPtr("envType"),
			DeclaredScopeValue: stringPtr("development"),
		}}))
	})

	It("parses single quotes, inline comments and trailing spaces", func() {
		records, err := parserpkg.ParseEnvFileRecords(`
SINGLE_QUOTED='literal # stays'
INLINE_COMMENT=value   # comment
TRAILING_SPACES=value   
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(records).To(Equal([]parserpkg.ParsedEnvVarRecord{
			{Line: 2, Key: "SINGLE_QUOTED", Value: "literal # stays"},
			{Line: 3, Key: "INLINE_COMMENT", Value: "value"},
			{Line: 4, Key: "TRAILING_SPACES", Value: "value"},
		}))
	})

	It("parses unicode escapes including surrogate pairs", func() {
		records, err := parserpkg.ParseEnvFileRecords(`
UNICODE_ESCAPED="hello \u4F60\u597D \uD83D\uDE80"
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(records).To(Equal([]parserpkg.ParsedEnvVarRecord{{
			Line:  2,
			Key:   "UNICODE_ESCAPED",
			Value: "hello 你好 🚀",
		}}))
	})

	It("rejects unsupported metadata directives", func() {
		_, err := parserpkg.ParseEnvFileRecords(`
# owner: team-a
GOOD_KEY=good-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("unsupported metadata directive"))
		Expect(err.Error()).To(ContainSubstring("owner"))
	})

	It("reports the line number for a dangling desc directive", func() {
		_, err := parserpkg.ParseEnvFileRecords(`
GOOD_KEY=good-value
# desc: missing assignment
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("metadata directive must be followed by KEY=VALUE"))
		Expect(err.Error()).To(ContainSubstring("line 3"))
	})

	It("rejects scopeValue without scopeType", func() {
		_, err := parserpkg.ParseEnvFileRecords(`
# scopeValue: development
GOOD_KEY=good-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("scopeValue requires scopeType"))
	})

	It("rejects env scopeType metadata", func() {
		_, err := parserpkg.ParseEnvFileRecords(`
# scopeType: env
# scopeValue: prod-env
GOOD_KEY=good-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring(`scopeType "env" is not supported`))
	})

	It("rejects export syntax even if the remainder is valid", func() {
		_, err := parserpkg.ParseEnvFileRecords(`export GOOD_KEY=good-value`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring(`invalid env var key "export GOOD_KEY"`))
	})

	It("rejects keys longer than the shared CRUD limit", func() {
		_, err := parserpkg.ParseEnvFileRecords(
			strings.Repeat("A", 257) + "=too-long-key",
		)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("env var key"))
		Expect(err.Error()).To(ContainSubstring("must be at most 256 characters"))
	})

	It("rejects values longer than the shared CRUD limit", func() {
		_, err := parserpkg.ParseEnvFileRecords(
			"GOOD_KEY=" + strings.Repeat("v", 8193),
		)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("env var value"))
		Expect(err.Error()).To(ContainSubstring("must be at most 8192 characters"))
	})

	It("rejects content larger than 1 MiB by bytes", func() {
		var builder strings.Builder
		for i := 0; i < 70000; i++ {
			builder.WriteString("GOOD_KEY_")
			builder.WriteString(strings.Repeat("A", 6))
			builder.WriteString("=")
			builder.WriteString("你好")
			builder.WriteString("\n")
		}

		_, err := parserpkg.ParseEnvFileRecords(builder.String())
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("must not exceed 1048576 bytes"))
	})
})

func stringPtr(value string) *string {
	return &value
}
