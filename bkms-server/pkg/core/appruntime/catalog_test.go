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

package appruntime_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

var _ = Describe("Catalog", func() {
	It("returns a stable sorted copy that callers cannot mutate", func() {
		first := appruntime.Catalog()
		Expect(first).To(HaveLen(3))
		Expect([]string{first[0].Name, first[1].Name, first[2].Name}).To(Equal([]string{
			appruntime.FrameworkBlank, appruntime.FrameworkTAF, appruntime.FrameworkTRPC,
		}))
		first[0].Name = "mutated"
		first[0].SupportedLanguages[0] = "java"

		second := appruntime.Catalog()
		Expect(second[0].Name).To(Equal(appruntime.FrameworkBlank))
		Expect(second[0].SupportedLanguages).NotTo(ContainElement("java"))
	})

	DescribeTable("valid stacks",
		func(stack appruntime.Stack) {
			Expect(appruntime.ValidateStack(stack)).To(Succeed())
		},
		Entry("trpc go", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo,
		}),
		Entry("trpc cpp", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageCpp,
		}),
		Entry("taf cpp", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTAF, Language: appruntime.LanguageCpp,
		}),
		Entry("blank go", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkBlank, Language: appruntime.LanguageGo,
		}),
		Entry("blank cpp", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkBlank, Language: appruntime.LanguageCpp,
		}),
		Entry("helm", appruntime.Stack{Type: appruntime.AppTypeHelm}),
		Entry("agones", appruntime.Stack{Type: appruntime.AppTypeAgones}),
	)

	DescribeTable("invalid stacks",
		func(stack appruntime.Stack, substring string) {
			err := appruntime.ValidateStack(stack)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(substring))
		},
		Entry("blank is not empty framework", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: "", Language: appruntime.LanguageGo,
		}, "framework is required"),
		Entry("taf go", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTAF, Language: appruntime.LanguageGo,
		}, "does not support language"),
		Entry("unknown framework", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: "spring", Language: appruntime.LanguageGo,
		}, "unknown framework"),
		Entry("unknown language", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: "java",
		}, "unknown language"),
		Entry("legacy type is not normalized", appruntime.Stack{
			Type: appruntime.AppTypeTRPC, Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo,
		}, "legacy app type"),
		Entry("helm with language", appruntime.Stack{
			Type: appruntime.AppTypeHelm, Language: appruntime.LanguageGo,
		}, "must not carry language or framework"),
	)

	It("reports stack violations through matchable sentinels", func() {
		Expect(appruntime.ValidateStack(appruntime.Stack{
			Type: appruntime.AppTypeHelm, Language: appruntime.LanguageGo,
		})).To(MatchError(appruntime.ErrUnexpectedStackFields))

		Expect(appruntime.ValidateStack(appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTAF, Language: appruntime.LanguageGo,
		})).To(MatchError(appruntime.ErrInvalidCombination))
	})

	It("treats blank as distinct from an empty string", func() {
		def, ok := appruntime.LookupFramework(appruntime.FrameworkBlank)
		Expect(ok).To(BeTrue())
		Expect(def.Name).To(Equal("blank"))

		// LookupFramework hands out a deep copy, so callers cannot edit the catalog.
		def.SupportedLanguages[0] = "java"
		reread, _ := appruntime.LookupFramework(appruntime.FrameworkBlank)
		Expect(reread.SupportedLanguages).NotTo(ContainElement("java"))

		_, emptyOK := appruntime.LookupFramework("")
		Expect(emptyOK).To(BeFalse())
	})
})
