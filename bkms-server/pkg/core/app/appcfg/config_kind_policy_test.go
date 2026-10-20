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

package appcfg_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("FrameworkPolicy", func() {
	var policy appcfg.ConfigKindPolicy

	BeforeEach(func() {
		policy = appcfg.FrameworkPolicy{}
	})

	Describe("ValidateContent", func() {
		It("should accept empty content", func() {
			Expect(policy.ValidateContent("", appcfg.FileFormatYAML)).To(Succeed())
		})

		It("should accept valid YAML", func() {
			Expect(policy.ValidateContent("foo: bar\nlist:\n  - a\n  - b", appcfg.FileFormatYAML)).To(Succeed())
		})

		It("should reject invalid YAML", func() {
			err := policy.ValidateContent("foo: [invalid", appcfg.FileFormatYAML)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("YAML"))
		})

		It("should validate as YAML regardless of format parameter", func() {
			Expect(policy.ValidateContent("key: value", "unknown_format")).To(Succeed())
		})
	})

	Describe("ValidateCreateParams", func() {
		It("should reject disable enableEnvVarRender", func() {
			enableEnvVarRender := false
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				EnableEnvVarRender: &enableEnvVarRender,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("enableEnvVarRender"))
		})

		It("should reject mountedEnvNames", func() {
			mountedEnvNames := []string{"prod"}
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				MountedEnvNames: &mountedEnvNames,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mountedEnvNames"))
		})

		DescribeTable("should reject a mount dir unless the app type allows it",
			func(appType string) {
				err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
					AppType:  appType,
					MountDir: "/etc/a",
				})
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("mountDir"))
			},
			Entry("helm", "helm"),
			Entry("agones", "agones"),
			Entry("empty app type", ""),
		)

		DescribeTable("should allow a framework file to set a mount dir",
			func(appType string) {
				Expect(policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
					AppType:  appType,
					MountDir: "/usr/local/trpc/bin",
				})).To(Succeed())
			},
			Entry("trpc", "trpc"),
			Entry("taf", "taf"),
		)
	})

	Describe("InitializeEnvConfigMode", func() {
		It("should initialize overwrite strategy for trpc framework", func() {
			mode := &appcfg.EnvConfigMode{}
			policy.InitializeEnvConfigMode(mode, appcfg.CreateCfgFileParams{AppType: "trpc"})
			Expect(mode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverwrite))
		})

		It("should initialize overlay strategy for helm framework", func() {
			mode := &appcfg.EnvConfigMode{}
			policy.InitializeEnvConfigMode(mode, appcfg.CreateCfgFileParams{AppType: "helm"})
			Expect(mode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverlay))
		})
	})

	Describe("GetEnvContentStrategy", func() {
		It("should return persisted strategy from framework def", func() {
			strategy, err := policy.GetEnvContentStrategy(&appcfg.AppConfigFileDef{
				ConfigKind: appcfg.ConfigKindFramework,
				EnvConfigMode: appcfg.EnvConfigMode{
					EnvContentStrategy: appcfg.EnvContentStrategyOverwrite,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(Equal(appcfg.EnvContentStrategyOverwrite))
		})

		It("should reject empty persisted framework strategy", func() {
			_, err := policy.GetEnvContentStrategy(&appcfg.AppConfigFileDef{
				ConfigKind:         appcfg.ConfigKindFramework,
				EnvConfigMode:      appcfg.EnvConfigMode{},
				EnableEnvVarRender: true,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("envContentStrategy"))
		})

		It("should reject invalid persisted framework strategy", func() {
			_, err := policy.GetEnvContentStrategy(&appcfg.AppConfigFileDef{
				ConfigKind: appcfg.ConfigKindFramework,
				EnvConfigMode: appcfg.EnvConfigMode{
					EnvContentStrategy: "invalid",
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("envContentStrategy"))
		})
	})

	Describe("IsEffectiveForEnv", func() {
		It("should always be effective for all environments", func() {
			def := &appcfg.AppConfigFileDef{AppID: "app1", Name: "cfg.yaml"}
			Expect(policy.IsEffectiveForEnv(def, "prod")).To(BeTrue())
			Expect(policy.IsEffectiveForEnv(def, "dev")).To(BeTrue())
			Expect(policy.IsEffectiveForEnv(def, "")).To(BeTrue())
		})
	})

	Describe("AllowNameUpdate", func() {
		It("should not allow renaming a trpc framework file", func() {
			Expect(policy.AllowNameUpdate("trpc")).To(BeFalse())
		})

		It("should allow renaming taf and helm framework files", func() {
			Expect(policy.AllowNameUpdate("taf")).To(BeTrue())
			Expect(policy.AllowNameUpdate("helm")).To(BeTrue())
			Expect(policy.AllowNameUpdate("")).To(BeTrue())
		})
	})

	Describe("AllowMountDirUpdate", func() {
		It("should not allow mount dir update for helm", func() {
			Expect(policy.AllowMountDirUpdate("helm")).To(BeFalse())
			Expect(policy.AllowMountDirUpdate("")).To(BeFalse())
		})

		It("should allow mount dir update for trpc and taf", func() {
			Expect(policy.AllowMountDirUpdate("trpc")).To(BeTrue())
			Expect(policy.AllowMountDirUpdate("taf")).To(BeTrue())
		})
	})

	Describe("AllowMountedEnvNamesUpdate", func() {
		It("should not allow mountedEnvNames update", func() {
			Expect(policy.AllowMountedEnvNamesUpdate()).To(BeFalse())
		})
	})

	Describe("AllowEnableEnvVarRenderUpdate", func() {
		It("should not allow enableEnvVarRender update", func() {
			Expect(policy.AllowEnableEnvVarRenderUpdate()).To(BeFalse())
		})
	})

	Describe("AllowEnvContentStrategyUpdate", func() {
		It("should allow envContentStrategy update", func() {
			Expect(policy.AllowEnvContentStrategyUpdate()).To(BeTrue())
		})
	})

	Describe("DefaultEnableEnvVarRender", func() {
		It("should enable env var render by default", func() {
			Expect(policy.DefaultEnableEnvVarRender()).To(BeTrue())
		})
	})
})

var _ = Describe("DefaultPolicies", func() {
	It("should contain framework policy", func() {
		p, ok := appcfg.DefaultPolicies[appcfg.ConfigKindFramework]
		Expect(ok).To(BeTrue())
		Expect(p).To(BeAssignableToTypeOf(appcfg.FrameworkPolicy{}))
	})
})

var _ = Describe("PlainPolicy", func() {
	var policy appcfg.ConfigKindPolicy

	BeforeEach(func() {
		policy = appcfg.PlainPolicy{}
	})

	Describe("IsEffectiveForEnv", func() {
		It("should treat nil mounted env names as effective for all environments", func() {
			def := &appcfg.AppConfigFileDef{
				AppID:      "app1",
				Name:       "plain.conf",
				ConfigKind: appcfg.ConfigKindPlain,
				EnvConfigMode: appcfg.EnvConfigMode{
					IsUnifiedConfig: true,
				},
			}
			Expect(policy.IsEffectiveForEnv(def, "prod")).To(BeTrue())
			Expect(policy.IsEffectiveForEnv(def, "staging")).To(BeTrue())
		})

		It("should treat explicit empty mounted env names as not effective for any environment", func() {
			def := &appcfg.AppConfigFileDef{
				AppID:      "app1",
				Name:       "plain.conf",
				ConfigKind: appcfg.ConfigKindPlain,
				EnvConfigMode: appcfg.EnvConfigMode{
					IsUnifiedConfig: true,
					MountedEnvNames: []string{},
				},
			}
			Expect(policy.IsEffectiveForEnv(def, "prod")).To(BeFalse())
			Expect(policy.IsEffectiveForEnv(def, "staging")).To(BeFalse())
		})

		It("should only be effective for listed environments", func() {
			def := &appcfg.AppConfigFileDef{
				AppID:      "app1",
				Name:       "plain.conf",
				ConfigKind: appcfg.ConfigKindPlain,
				EnvConfigMode: appcfg.EnvConfigMode{
					IsUnifiedConfig: false,
					MountedEnvNames: []string{"prod"},
				},
			}
			Expect(policy.IsEffectiveForEnv(def, "prod")).To(BeTrue())
			Expect(policy.IsEffectiveForEnv(def, "staging")).To(BeFalse())
		})
	})

	Describe("ValidateCreateParams", func() {
		It("should accept valid plain params", func() {
			Expect(policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				MountDir:          "/data",
				ContentSourceType: appcfg.ContentSourceTypeLocal,
			})).To(Succeed())
		})

		It("should reject empty mountDir", func() {
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mountDir"))
		})

		It("should reject non-local content source", func() {
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				MountDir:          "/data",
				ContentSourceType: appcfg.ContentSourceTypeBSCP,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("local"))
		})

		It("should reject base reference", func() {
			baseID := bson.NewObjectID()
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				MountDir:            "/data",
				BaseAppConfigFileID: &baseID,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("base"))
		})

		It("should reject overlay content", func() {
			overlay := "patch"
			err := policy.ValidateCreateParams(appcfg.CreateCfgFileParams{
				MountDir:       "/data",
				OverlayContent: &overlay,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("overlay"))
		})
	})

	Describe("AllowMountedEnvNamesUpdate", func() {
		It("should allow mountedEnvNames update", func() {
			Expect(policy.AllowMountedEnvNamesUpdate()).To(BeTrue())
		})
	})

	Describe("AllowEnableEnvVarRenderUpdate", func() {
		It("should allow enableEnvVarRender update", func() {
			Expect(policy.AllowEnableEnvVarRenderUpdate()).To(BeTrue())
		})
	})

	Describe("AllowEnvContentStrategyUpdate", func() {
		It("should reject envContentStrategy update", func() {
			Expect(policy.AllowEnvContentStrategyUpdate()).To(BeFalse())
		})
	})

	Describe("InitializeEnvConfigMode", func() {
		It("should not write persisted strategy for plain config", func() {
			mode := &appcfg.EnvConfigMode{}
			policy.InitializeEnvConfigMode(mode, appcfg.CreateCfgFileParams{AppType: "trpc"})
			Expect(mode.EnvContentStrategy).To(BeEmpty())
		})
	})

	Describe("GetEnvContentStrategy", func() {
		It("should always return overwrite strategy", func() {
			strategy, err := policy.GetEnvContentStrategy(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(Equal(appcfg.EnvContentStrategyOverwrite))
		})
	})

	Describe("DefaultEnableEnvVarRender", func() {
		It("should disable env var render by default", func() {
			Expect(policy.DefaultEnableEnvVarRender()).To(BeFalse())
		})
	})
})
