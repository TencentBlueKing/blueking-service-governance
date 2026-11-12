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

package frameworktmpl

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

var _ = Describe("Builtin templates", func() {
	It("provides exactly one builtin template for every group with a framework config file", func() {
		var groups []Group
		for _, def := range appruntime.Catalog() {
			for _, language := range def.SupportedLanguages {
				if _, _, err := ResolveGroup(def.Name, language); err == nil {
					groups = append(groups, Group{Framework: def.Name, Language: language})
				}
			}
		}
		Expect(groups).NotTo(BeEmpty())

		builtinGroups := make([]Group, 0, len(builtins))
		for _, b := range builtins {
			builtinGroups = append(builtinGroups, b.Group)
		}
		Expect(builtinGroups).To(ConsistOf(groups))
	})

	It("ships builtin contents that pass format validation", func() {
		for _, b := range builtins {
			_, format, err := ResolveGroup(b.Framework, b.Language)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Content).NotTo(BeEmpty(), b.ID)
			Expect(len(b.Content)).To(BeNumerically("<=", MaxContentSize), b.ID)
			Expect(appcfg.FrameworkPolicy{}.ValidateContent(b.Content, format)).To(Succeed(), b.ID)
			Expect(fileNamePattern.MatchString(b.FileName)).To(BeTrue(), b.ID)
		}
	})

	It("looks up builtin templates by ID and group as copies", func() {
		group := Group{Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageGo}
		view, ok := builtinOf(group)
		Expect(ok).To(BeTrue())
		Expect(view.ID).To(Equal("builtin-trpc-go"))
		Expect(view.Builtin).To(BeTrue())
		Expect(view.Name).To(Equal(BuiltinName))

		view.Content = "mutated"
		again, ok := lookupBuiltin("builtin-trpc-go")
		Expect(ok).To(BeTrue())
		Expect(again.Content).NotTo(Equal("mutated"))

		_, ok = lookupBuiltin("builtin-trpc-python")
		Expect(ok).To(BeFalse())
	})
})
