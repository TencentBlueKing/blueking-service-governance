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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

var _ = Describe("Builtin templates", func() {
	It("lists every group with a framework config file", func() {
		Expect(groups).To(ConsistOf(
			Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo},
			Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageCpp},
			Group{Framework: appruntime.FrameworkTAF, Language: appruntime.LanguageCpp},
		))
	})

	It("provides at most one builtin template per supported group and none for TAF", func() {
		builtinGroups := lo.Map(builtins, func(b View, _ int) Group { return b.Group })
		Expect(lo.Uniq(builtinGroups)).To(HaveLen(len(builtinGroups)))
		Expect(groups).To(ContainElements(builtinGroups))
		Expect(builtinGroups).To(ConsistOf(
			Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo},
			Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageCpp},
		))
	})

	It("ships builtin contents that pass format validation", func() {
		for _, b := range builtins {
			_, format, err := ResolveGroup(b.Framework, b.Language)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Content).NotTo(BeEmpty(), b.ID)
			Expect(appcfg.FrameworkPolicy{}.ValidateContent(b.Content, format)).To(Succeed(), b.ID)
		}
	})

	It("shares the tRPC Go content with tRPC C++ under its own file name", func() {
		goView, ok := lookupBuiltin("builtin-trpc-go")
		Expect(ok).To(BeTrue())
		cppView, ok := lookupBuiltin("builtin-trpc-cpp")
		Expect(ok).To(BeTrue())
		Expect(cppView.Content).To(Equal(goView.Content))
		Expect(goView.FileName).To(Equal("trpc_go.yaml"))
		Expect(cppView.FileName).To(Equal("trpc_cpp.yaml"))

		_, ok = lookupBuiltin("builtin-trpc-python")
		Expect(ok).To(BeFalse())
	})

	It("looks up builtin templates by ID as copies", func() {
		group := Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo}
		view, ok := lookupBuiltin(builtinID(group))
		Expect(ok).To(BeTrue())
		Expect(view.ID).To(Equal("builtin-trpc-go"))
		Expect(view.Group).To(Equal(group))
		Expect(view.Builtin).To(BeTrue())
		Expect(view.Name).To(Equal(BuiltinName))

		view.Content = "mutated"
		again, ok := lookupBuiltin("builtin-trpc-go")
		Expect(ok).To(BeTrue())
		Expect(again.Content).NotTo(Equal("mutated"))

		_, ok = lookupBuiltin("builtin-trpc-nodejs")
		Expect(ok).To(BeFalse())
		_, ok = lookupBuiltin("builtin-taf-cpp")
		Expect(ok).To(BeFalse())
	})
})
