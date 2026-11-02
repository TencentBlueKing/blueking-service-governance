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

package mods_test

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/admincommand"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/configmerge"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
)

func toJSON(view any) string {
	raw, err := json.Marshal(view)
	Expect(err).NotTo(HaveOccurred())
	return string(raw)
}

func mustResolve(framework, language string) mods.Set {
	set, err := mods.Resolve(appruntime.Stack{
		Type: appruntime.AppTypeBKMSApp, Framework: framework, Language: language,
	})
	Expect(err).NotTo(HaveOccurred())
	return set
}

var _ = Describe("Of", func() {
	It("reads the language of a legacy trpc app from TrpcSpec", func() {
		set := mods.Of(&bkmsapp.Application{
			Type:     bkmsapp.AppTypeTRPC,
			TrpcSpec: &bkmsapp.TrpcSpec{Language: appruntime.LanguageGo},
		})
		Expect(set.Stack.Framework).To(Equal(appruntime.FrameworkTRPC))
		Expect(set.Stack.Language).To(Equal(appruntime.LanguageGo))
		Expect(set.PlatformBuild.Supported).To(BeTrue())
	})

	It("keeps devmode available for a legacy taf app without a language", func() {
		set := mods.Of(&bkmsapp.Application{Type: bkmsapp.AppTypeTAF})
		Expect(set.Stack.Language).To(BeEmpty())
		Expect(set.DevMode.Supported).To(BeTrue())
		Expect(set.DevMode.Props.ScriptSet).To(Equal(devmode.ScriptSetTAF))
		Expect(set.PlatformBuild.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedStack))
	})

	It("marks every capability unsupported when the stack contradicts itself", func() {
		set := mods.Of(&bkmsapp.Application{
			Type:     bkmsapp.AppTypeTRPC,
			Language: appruntime.LanguageCpp,
			TrpcSpec: &bkmsapp.TrpcSpec{Language: appruntime.LanguageGo},
		})
		views := set.Public().Mods
		Expect(views).To(HaveLen(9))
		for id, mod := range views {
			Expect(mod.Supported).To(BeFalse(), string(id))
			Expect(mod.ReasonCode).To(Equal(runtimemod.ReasonInvalidStack), string(id))
		}

		_, err := set.DevMode.Require()
		Expect(err).To(MatchError(runtimemod.ErrModUnsupported))
		Expect(err.Error()).To(ContainSubstring("devMode: invalid_stack"))
	})
})

var _ = Describe("Resolve", func() {
	DescribeTable("rejects an invalid stack",
		func(stack appruntime.Stack, substring string) {
			_, err := mods.Resolve(stack)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(substring))
		},
		Entry("unknown framework", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: "quic", Language: appruntime.LanguageGo,
		}, "unknown framework"),
		Entry("taf with go", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTAF, Language: appruntime.LanguageGo,
		}, "does not support language"),
		Entry("unknown language", appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: "java",
		}, "unknown language"),
	)

	It("reports every capability as unsupported for helm", func() {
		set, err := mods.Resolve(appruntime.Stack{Type: appruntime.AppTypeHelm})
		Expect(err).NotTo(HaveOccurred())
		for _, mod := range set.Public().Mods {
			Expect(mod.Supported).To(BeFalse())
			Expect(mod.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedAppType))
		}
	})

	It("supports devmode but not platformBuild for trpc+cpp", func() {
		set := mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageCpp)
		Expect(set.DevMode.Supported).To(BeTrue())
		Expect(set.PlatformBuild.Supported).To(BeFalse())
		Expect(set.PlatformBuild.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedStack))
	})

	It("reports a missing language separately from an unsupported stack", func() {
		trpcNoLanguage := mustResolve(appruntime.FrameworkTRPC, "")
		Expect(trpcNoLanguage.PlatformBuild.ReasonCode).To(Equal(runtimemod.ReasonIncompleteLanguage))
		Expect(trpcNoLanguage.AdminCommand.ReasonCode).To(Equal(runtimemod.ReasonIncompleteLanguage))
		Expect(trpcNoLanguage.DevMode.Supported).To(BeTrue())

		taf := mustResolve(appruntime.FrameworkTAF, appruntime.LanguageCpp)
		Expect(taf.PlatformBuild.ReasonCode).To(Equal(runtimemod.ReasonUnsupportedStack))
	})

	It("selects the admin config layout per language", func() {
		Expect(mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageGo).AdminCommand.Props.ConfigLayout).
			To(Equal(admincommand.TrpcLayoutNested))
		Expect(mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageCpp).AdminCommand.Props.ConfigLayout).
			To(Equal(admincommand.TrpcLayoutFlat))
		Expect(mustResolve(appruntime.FrameworkTAF, appruntime.LanguageCpp).AdminCommand.Props.Protocol).
			To(Equal(admincommand.ProtocolTAF))
	})

	It("registers admin layouts for out-of-catalog languages while keeping them unreachable", func() {
		Expect(admincommand.Spec.Resolve(appruntime.FrameworkTRPC, appruntime.LanguagePython).Props.ConfigLayout).
			To(Equal(admincommand.TrpcLayoutNested))
		Expect(admincommand.Spec.Resolve(appruntime.FrameworkTRPC, appruntime.LanguageNodeJS).Props.ConfigLayout).
			To(Equal(admincommand.TrpcLayoutFlat))
		Expect(admincommand.Spec.Resolve(appruntime.FrameworkTRPC, appruntime.LanguageJava).Props.ConfigLayout).
			To(Equal(admincommand.TrpcLayoutNestedPrefixed))

		// java 不在语言目录中；nodejs / python 只对 blank 开放，tRPC 组合不合法
		_, err := mods.Resolve(appruntime.Stack{
			Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageJava,
		})
		Expect(errors.Is(err, appruntime.ErrUnknownLanguage)).To(BeTrue())
		for _, language := range []string{appruntime.LanguageNodeJS, appruntime.LanguagePython} {
			_, err = mods.Resolve(appruntime.Stack{
				Type: appruntime.AppTypeBKMSApp, Framework: appruntime.FrameworkTRPC, Language: language,
			})
			Expect(errors.Is(err, appruntime.ErrInvalidCombination)).To(BeTrue(), language)
		}
	})

	DescribeTable("treats blank as a framework without framework capabilities",
		func(language string) {
			for id, mod := range mustResolve(appruntime.FrameworkBlank, language).Public().Mods {
				if id == runtimemod.ModConfigMerge {
					// 配置文件合并是基座功能，只是策略随框架变化
					continue
				}
				Expect(mod.Supported).To(BeFalse(), string(id))
			}
		},
		Entry("go", appruntime.LanguageGo),
		Entry("cpp", appruntime.LanguageCpp),
		Entry("nodejs", appruntime.LanguageNodeJS),
		Entry("python", appruntime.LanguagePython),
	)

	DescribeTable("selects the config merge strategy per framework",
		func(framework, language string, expected configmerge.Strategy) {
			props, err := mustResolve(framework, language).ConfigMerge.Require()
			Expect(err).NotTo(HaveOccurred())
			Expect(props.Strategy).To(Equal(expected))
		},
		Entry("trpc", appruntime.FrameworkTRPC, appruntime.LanguageGo, configmerge.StrategyNullPreserving),
		Entry("taf", appruntime.FrameworkTAF, "", configmerge.StrategyNullPreserving),
		Entry("blank", appruntime.FrameworkBlank, appruntime.LanguageGo, configmerge.StrategyStrategicMerge),
	)

	It("does not let a caller mutate later results through returned props", func() {
		first := mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageGo)
		first.DevMode.Props.WorkPath = "/mutated"

		second := mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageGo)
		Expect(second.DevMode.Props.WorkPath).To(Equal(devmode.TrpcWorkPath))
	})
})

var _ = Describe("Public", func() {
	It("renders trpc+go with every capability supported", func() {
		Expect(toJSON(mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageGo).Public())).
			To(MatchJSON(`{
				"schemaVersion": 1,
				"type": "bkmsApp",
				"framework": "trpc",
				"language": "go",
				"mods": {
					"frameworkFile":         {"supported": true, "props": {"format": "yaml"}},
					"devMode":               {"supported": true, "props": {
						"workPath": "/data/bkms/dev-mode/trpc",
						"mountPath": "/data/bkms/dev-mode/trpc/configmap-scripts"
					}},
					"platformBuild":         {"supported": true},
					"adminCommand":          {"supported": true},
					"apmServiceName":        {"supported": true},
					"polarisServiceCheck":   {"supported": true},
					"polarisFrameworkPatch": {"supported": true},
					"bscpWorkloadMeta":      {"supported": true},
					"configMerge":           {"supported": true}
				}
			}`))
	})

	It("renders trpc+cpp with platformBuild unsupported", func() {
		Expect(toJSON(mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageCpp).Public())).
			To(MatchJSON(`{
				"schemaVersion": 1,
				"type": "bkmsApp",
				"framework": "trpc",
				"language": "cpp",
				"mods": {
					"frameworkFile":         {"supported": true, "props": {"format": "yaml"}},
					"devMode":               {"supported": true, "props": {
						"workPath": "/data/bkms/dev-mode/trpc",
						"mountPath": "/data/bkms/dev-mode/trpc/configmap-scripts"
					}},
					"platformBuild":         {"supported": false, "reasonCode": "unsupported_stack"},
					"adminCommand":          {"supported": true},
					"apmServiceName":        {"supported": true},
					"polarisServiceCheck":   {"supported": true},
					"polarisFrameworkPatch": {"supported": true},
					"bscpWorkloadMeta":      {"supported": true},
					"configMerge":           {"supported": true}
				}
			}`))
	})

	It("renders a taf row that never recorded a language without the language key", func() {
		Expect(toJSON(mustResolve(appruntime.FrameworkTAF, "").Public())).
			To(MatchJSON(`{
				"schemaVersion": 1,
				"type": "bkmsApp",
				"framework": "taf",
				"mods": {
					"frameworkFile":         {"supported": true, "props": {"format": "taf"}},
					"devMode":               {"supported": true, "props": {
						"workPath": "/data/bkms/dev-mode/taf",
						"mountPath": "/data/bkms/dev-mode/taf/configmap-scripts"
					}},
					"platformBuild":         {"supported": false, "reasonCode": "unsupported_stack"},
					"adminCommand":          {"supported": true},
					"apmServiceName":        {"supported": true},
					"polarisServiceCheck":   {"supported": false, "reasonCode": "unsupported_stack"},
					"polarisFrameworkPatch": {"supported": false, "reasonCode": "unsupported_stack"},
					"bscpWorkloadMeta":      {"supported": false, "reasonCode": "unsupported_stack"},
					"configMerge":           {"supported": true}
				}
			}`))
	})

	It("renders blank with only configMerge supported", func() {
		Expect(toJSON(mustResolve(appruntime.FrameworkBlank, appruntime.LanguageGo).Public())).
			To(MatchJSON(`{
				"schemaVersion": 1,
				"type": "bkmsApp",
				"framework": "blank",
				"language": "go",
				"mods": {
					"frameworkFile":         {"supported": false, "reasonCode": "unsupported_stack"},
					"devMode":               {"supported": false, "reasonCode": "unsupported_stack"},
					"platformBuild":         {"supported": false, "reasonCode": "unsupported_stack"},
					"adminCommand":          {"supported": false, "reasonCode": "unsupported_stack"},
					"apmServiceName":        {"supported": false, "reasonCode": "unsupported_stack"},
					"polarisServiceCheck":   {"supported": false, "reasonCode": "unsupported_stack"},
					"polarisFrameworkPatch": {"supported": false, "reasonCode": "unsupported_stack"},
					"bscpWorkloadMeta":      {"supported": false, "reasonCode": "unsupported_stack"},
					"configMerge":           {"supported": true}
				}
			}`))
	})

	It("renders helm with no stack fields and an app type reason", func() {
		set, err := mods.Resolve(appruntime.Stack{Type: appruntime.AppTypeHelm})
		Expect(err).NotTo(HaveOccurred())
		Expect(toJSON(set.Public())).To(MatchJSON(`{
			"schemaVersion": 1,
			"type": "helm",
			"mods": {
				"frameworkFile":         {"supported": false, "reasonCode": "unsupported_app_type"},
				"devMode":               {"supported": false, "reasonCode": "unsupported_app_type"},
				"platformBuild":         {"supported": false, "reasonCode": "unsupported_app_type"},
				"adminCommand":          {"supported": false, "reasonCode": "unsupported_app_type"},
				"apmServiceName":        {"supported": false, "reasonCode": "unsupported_app_type"},
				"polarisServiceCheck":   {"supported": false, "reasonCode": "unsupported_app_type"},
				"polarisFrameworkPatch": {"supported": false, "reasonCode": "unsupported_app_type"},
				"bscpWorkloadMeta":      {"supported": false, "reasonCode": "unsupported_app_type"},
				"configMerge":           {"supported": false, "reasonCode": "unsupported_app_type"}
			}
		}`))
	})
})

var _ = Describe("Matrix", func() {
	It("covers every legal catalog combination in stable order", func() {
		matrix, err := mods.Matrix()
		Expect(err).NotTo(HaveOccurred())

		combinations := make([]string, 0, len(matrix))
		for _, view := range matrix {
			combinations = append(combinations, view.Framework+"+"+view.Language)
		}
		Expect(combinations).To(Equal([]string{
			"blank+cpp", "blank+go", "blank+nodejs", "blank+python", "taf+cpp", "trpc+cpp", "trpc+go",
		}))
	})

	It("agrees with resolving each combination individually", func() {
		matrix, err := mods.Matrix()
		Expect(err).NotTo(HaveOccurred())
		for _, view := range matrix {
			Expect(view).To(Equal(mustResolve(view.Framework, view.Language).Public()))
		}
	})
})
