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

package runtimemod_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

type testProps struct{ Value string }

func (p testProps) Public() any { return p }

var spec = runtimemod.Spec[testProps]{
	ID: runtimemod.ModDevMode,
	Resolve: runtimemod.ByFramework(map[string]testProps{
		appruntime.FrameworkTRPC: {Value: "trpc"},
	}),
}

var _ = Describe("ByFramework", func() {
	It("supports a declared framework regardless of language", func() {
		Expect(spec.ResolveStack(bkmsStack(appruntime.FrameworkTRPC, "")).Supported).To(BeTrue())
		Expect(spec.ResolveStack(bkmsStack(appruntime.FrameworkTRPC, appruntime.LanguageGo)).Props.Value).
			To(Equal("trpc"))
	})

	It("reports an undeclared framework as unsupported", func() {
		mod := spec.ResolveStack(bkmsStack(appruntime.FrameworkBlank, appruntime.LanguageGo))
		Expect(mod.Supported).To(BeFalse())
		Expect(mod.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedStack))
	})
})

var _ = Describe("ByLanguage", func() {
	byLanguage := runtimemod.Spec[testProps]{
		ID: runtimemod.ModPlatformBuild,
		Resolve: runtimemod.ByLanguage(map[string]testProps{
			appruntime.LanguageGo: {Value: "go"},
		}),
	}

	It("reports a missing language as incomplete instead of unsupported", func() {
		mod := byLanguage.ResolveStack(bkmsStack(appruntime.FrameworkTRPC, ""))
		Expect(mod.ReasonCode).To(Equal(runtimemod.ReasonIncompleteLanguage))
	})

	It("supports a declared language and rejects others", func() {
		Expect(byLanguage.ResolveStack(bkmsStack(appruntime.FrameworkTRPC, appruntime.LanguageGo)).Supported).
			To(BeTrue())
		Expect(byLanguage.ResolveStack(bkmsStack(appruntime.FrameworkTRPC, appruntime.LanguageCpp)).ReasonCode).
			To(Equal(runtimemod.ReasonUnsupportedStack))
	})
})

var _ = Describe("Spec", func() {
	It("marks every capability unsupported for non-bkmsApp applications", func() {
		mod := spec.ResolveStack(appruntime.Stack{Type: appruntime.AppTypeHelm})
		Expect(mod.Supported).To(BeFalse())
		Expect(mod.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedAppType))
	})

	It("returns ErrModUnsupported with the capability ID when unsupported", func() {
		mod := spec.ResolveStack(appruntime.Stack{Type: appruntime.AppTypeHelm})
		_, err := mod.Require()
		Expect(err).To(MatchError(runtimemod.ErrModUnsupported))
		Expect(err.Error()).To(ContainSubstring("devMode"))
	})
})

func bkmsStack(framework, language string) appruntime.Stack {
	return appruntime.Stack{Type: appruntime.AppTypeBKMSApp, Framework: framework, Language: language}
}
