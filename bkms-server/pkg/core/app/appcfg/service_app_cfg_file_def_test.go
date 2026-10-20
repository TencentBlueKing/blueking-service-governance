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
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppCfgFileDefService — Create / Update / Delete", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	Context("Create", func() {
		It("should reject a second helm values file in another directory", func() {
			content := "key: value"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "values.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           "helm",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "values.yaml",
				MountDir:          "/etc/b",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           "helm",
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
		})

		It("should create def, file and initial version", func() {
			result := f.createFrameworkFile("values.yaml")

			Expect(result.Def).NotTo(BeNil())
			Expect(result.Def.Name).To(Equal("values.yaml"))
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindFramework))
			Expect(result.Def.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
			Expect(result.DefID).To(Equal(result.Def.ID))
			Expect(result.CurrentVersion).To(Equal(int64(1)))

			gotDef, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.Name).To(Equal("values.yaml"))

			gotFile, err := f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.DefID).To(Equal(result.Def.ID))
		})

		It("should attach framework env overlay to the base file def", func() {
			base := f.createFrameworkFile("default")
			overlay := "server:\n  app: test\n"
			created, err := f.Svc.AttachFrameworkEnvOverlay(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:               f.AppID,
				EnvName:             "test",
				Name:                "test",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				BaseAppConfigFileID: &base.ID,
				OverlayContent:      &overlay,
				Creator:             "tester",
				Description:         "env overlay",
				ConfigKind:          appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.DefID).To(Equal(base.DefID))
			Expect(created.EnvName).To(Equal("test"))
			Expect(created.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))
			Expect(created.BaseAppConfigFileID).NotTo(BeNil())
			Expect(*created.BaseAppConfigFileID).To(Equal(base.ID))
			Expect(lo.FromPtr(created.OverlayContent)).To(Equal(overlay))

			defs, err := appcfg.ListDefsByApp(
				f.Ctx,
				f.DefStore,
				f.AppID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(1))
			Expect(defs[0].EnvConfigMode.IsUnifiedConfig).To(BeFalse())
		})

		It("should create overwrite env instance through compatibility entry for app model defs", func() {
			content := "default: true\n"
			base, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           "trpc",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(base.Def.EnvConfigMode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverwrite))

			created, err := f.Svc.AttachFrameworkEnvOverlay(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:               f.AppID,
				EnvName:             "test",
				Name:                "test",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				BaseAppConfigFileID: &base.ID,
				Creator:             "tester",
				Description:         "env create",
				ConfigKind:          appcfg.ConfigKindFramework,
				AppType:             "trpc",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(lo.FromPtr(created.Content)).To(BeEmpty())
			Expect(created.OverlayContent).To(BeNil())
		})

		It("should reuse existing framework env overlay instead of creating another def", func() {
			base := f.createFrameworkFile("default")
			overlay := "a: 1\n"
			first, err := f.Svc.AttachFrameworkEnvOverlay(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:               f.AppID,
				EnvName:             "test",
				Name:                "test",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				BaseAppConfigFileID: &base.ID,
				OverlayContent:      &overlay,
				Creator:             "tester",
				ConfigKind:          appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			second, err := f.Svc.AttachFrameworkEnvOverlay(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:               f.AppID,
				EnvName:             "test",
				Name:                "test",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				BaseAppConfigFileID: &base.ID,
				Creator:             "tester",
				ConfigKind:          appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(second.ID).To(Equal(first.ID))
			Expect(second.DefID).To(Equal(base.DefID))

			defs, err := appcfg.ListDefsByApp(
				f.Ctx,
				f.DefStore,
				f.AppID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(1))
		})

		DescribeTable("framework def uniqueness by app type", func(
			appType string,
			firstName string,
			expectSecondCreateErr bool,
		) {
			content := "key: value"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              firstName,
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           appType,
			})
			Expect(err).NotTo(HaveOccurred())

			content2 := "key: other"
			second, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "another.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content2,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           appType,
			})
			if expectSecondCreateErr {
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("only one framework config file"))
				return
			}

			Expect(err).NotTo(HaveOccurred())
			Expect(second.Def.Name).To(Equal("another.yaml"))

			defs, err := appcfg.ListDefsByApp(
				f.Ctx,
				f.DefStore,
				f.AppID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(2))
		},
			Entry("trpc apps allow only one framework def", "trpc", "default", true),
			Entry("bkmsApp apps allow only one framework def", "bkmsApp", "app-model.yaml", true),
			Entry("helm apps allow multiple framework defs", "helm", "values.yaml", false),
		)

		It("should reject empty config kind", func() {
			content := "key: value"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "legacy-values.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "legacy init",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("config kind is required"))
		})

		It("should reject invalid YAML content", func() {
			content := "invalid: [yaml"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "bad.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("content validation"))
		})

		It("should accept non-YAML content for plain kind", func() {
			content := "this is { not yaml ["
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "plain.conf",
				MountDir:          "/etc/app",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindPlain,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindPlain))
		})

		It("should set EnableEnvVarRender=true for framework kind", func() {
			result := f.createFrameworkFile("fw.yaml")
			Expect(result.Def.EnableEnvVarRender).To(BeTrue())

			gotDef, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.EnableEnvVarRender).To(BeTrue())
		})

		It("should set EnableEnvVarRender=false for plain kind", func() {
			result := f.createPlainFile("plain.conf", "/etc/app", "content")
			Expect(result.Def.EnableEnvVarRender).To(BeFalse())

			gotDef, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.EnableEnvVarRender).To(BeFalse())
		})

		It("should allow plain kind create with mountedEnvNames and EnableEnvVarRender", func() {
			content := "plain content"
			mounted := []string{"prod", "staging"}
			enableRender := true
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:              f.AppID,
				EnvName:            appcfg.EnvNameDefault,
				Name:               "plain-render.conf",
				MountDir:           "/etc/app",
				Type:               appcfg.AppConfigFileTypeNormal,
				ContentSourceType:  appcfg.ContentSourceTypeLocal,
				Format:             appcfg.FileFormatYAML,
				Content:            &content,
				Creator:            "tester",
				ConfigKind:         appcfg.ConfigKindPlain,
				MountedEnvNames:    &mounted,
				EnableEnvVarRender: &enableRender,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Def.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
			Expect(result.Def.EnvConfigMode.MountedEnvNames).To(Equal(mounted))
			Expect(result.Def.EnableEnvVarRender).To(BeTrue())
		})

		It("should reject framework kind create with EnableEnvVarRender=false", func() {
			content := "key: value"
			disableRender := false
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:              f.AppID,
				EnvName:            appcfg.EnvNameDefault,
				Name:               "fw.yaml",
				Type:               appcfg.AppConfigFileTypeNormal,
				ContentSourceType:  appcfg.ContentSourceTypeLocal,
				Format:             appcfg.FileFormatYAML,
				Content:            &content,
				Creator:            "tester",
				ConfigKind:         appcfg.ConfigKindFramework,
				EnableEnvVarRender: &disableRender,
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("enableEnvVarRender"))
		})

		DescribeTable(
			"initial framework env content strategy by app type",
			func(
				appType string,
				name string,
				format appcfg.FileFormat,
				want appcfg.EnvContentStrategy,
			) {
				content := "key: value"
				result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
					AppID:             f.AppID,
					EnvName:           appcfg.EnvNameDefault,
					Name:              name,
					Type:              appcfg.AppConfigFileTypeNormal,
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            format,
					Content:           &content,
					Creator:           "tester",
					ConfigKind:        appcfg.ConfigKindFramework,
					AppType:           appType,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Def.EnvConfigMode.EnvContentStrategy).To(Equal(want))
			},
			Entry(
				"trpc uses overwrite",
				"trpc",
				"trpc_go.yaml",
				appcfg.FileFormatYAML,
				appcfg.EnvContentStrategyOverwrite,
			),
			Entry("taf uses overwrite", "taf", "taf.conf", appcfg.FileFormatTAF, appcfg.EnvContentStrategyOverwrite),
			Entry("helm uses overlay", "helm", "default", appcfg.FileFormatYAML, appcfg.EnvContentStrategyOverlay),
			Entry(
				"empty app type defaults to overlay",
				"",
				"values.yaml",
				appcfg.FileFormatYAML,
				appcfg.EnvContentStrategyOverlay,
			),
		)

		It("should reject framework kind create with mountedEnvNames", func() {
			content := "key: value"
			mountedEnvNames := []string{"prod"}
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "fw-mounted.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				MountedEnvNames:   &mountedEnvNames,
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("mountedEnvNames"))
		})
	})

	Context("UpdateAppCfgFileDef", func() {
		It("should update name", func() {
			result := f.createFrameworkFile("old-name.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := "new-name.yaml"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-name.yaml"))
		})

		It("should reject mountedEnvNames update for framework kind", func() {
			result := f.createFrameworkFile("fw-mounted.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("mountedEnvNames"))

			unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.EnvConfigMode.MountedEnvNames).To(BeNil())
		})

		It("should clean a trailing slash from a plain mountDir", func() {
			result := f.createPlainFile("slash.conf", "/etc/app/", "content")
			Expect(result.Def.MountDir).To(Equal("/etc/app"))
		})

		It("should allow the same file name under different mount dirs", func() {
			first := f.createPlainFile("app.conf", "/etc/a", "a")
			second := f.createPlainFile("app.conf", "/etc/b", "b")
			Expect(first.Def.Name).To(Equal("app.conf"))
			Expect(second.Def.Name).To(Equal("app.conf"))
			Expect(first.Def.ID).NotTo(Equal(second.Def.ID))
		})

		It("should reject a rename onto an existing mount identity", func() {
			first := f.createPlainFile("a.yaml", "/etc/app", "a")
			second := f.createPlainFile("b.yaml", "/etc/app", "b")
			def, err := f.DefStore.GetByID(f.Ctx, second.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := first.Def.Name
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				AppType:  "helm",
				Operator: "editor",
			})
			Expect(errors.Is(err, appcfg.ErrAppConfigFileDefAlreadyExists)).To(BeTrue())

			unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.Name).To(Equal("b.yaml"))
		})

		It("should allow mountDir update for a trpc framework file", func() {
			result := f.createFrameworkFile("trpc_go.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/usr/local/trpc/bin/"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				AppType:  "trpc",
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(Equal("/usr/local/trpc/bin"))
		})

		It("should reject renaming a trpc framework file", func() {
			result := f.createFrameworkFile("trpc_go.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := "other.yaml"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				AppType:  "trpc",
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("does not allow modifying name"))

			unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.Name).To(Equal("trpc_go.yaml"))
		})

		It("should reject mountDir update for framework kind", func() {
			result := f.createFrameworkFile("fw.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("mountDir"))
		})

		It("should reject mountDir update for unsupported config kind", func() {
			def := &appcfg.AppConfigFileDef{
				AppID:      f.AppID,
				Name:       "legacy.yaml",
				ConfigKind: "",
				MountDir:   "/old/path",
				Creator:    "tester",
				EnvConfigMode: appcfg.EnvConfigMode{
					IsUnifiedConfig: true,
				},
			}
			err := f.DefStore.Create(f.Ctx, def)
			defID := def.ID
			Expect(err).NotTo(HaveOccurred())

			def, err = f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("unsupported config kind"))

			unchanged, err := f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.MountDir).To(Equal("/old/path"))
		})

		It("should allow multiple plain files to share the same mountDir", func() {
			first := f.createPlainFile("a.conf", "/etc/app", "a: 1")
			second := f.createPlainFile("b.conf", "/etc/app", "b: 2")

			Expect(first.Def.MountDir).To(Equal("/etc/app"))
			Expect(second.Def.MountDir).To(Equal("/etc/app"))
			Expect(first.Def.ID).NotTo(Equal(second.Def.ID))
		})

		It("should allow updating a plain mountDir to share another file directory", func() {
			f.createPlainFile("keep.conf", "/etc/app", "keep")
			movable := f.createPlainFile("move.conf", "/old/path", "move")
			def, err := f.DefStore.GetByID(f.Ctx, movable.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			shared := "/etc/app"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &shared,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(Equal("/etc/app"))
		})

		It("should allow mountDir update for plain kind", func() {
			result := f.createPlainFile("editable.conf", "/old/path", "content")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(Equal("/new/path"))
		})

		It("should allow plain kind to update EnableEnvVarRender to true", func() {
			result := f.createPlainFile("toggle.conf", "/etc/app", "content")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(def.EnableEnvVarRender).To(BeFalse())

			enable := true
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnableEnvVarRender: &enable,
				Operator:           "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnableEnvVarRender).To(BeTrue())
		})

		It("should reject EnableEnvVarRender update for framework kind", func() {
			result := f.createFrameworkFile("fw.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			disable := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnableEnvVarRender: &disable,
				Operator:           "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("enableEnvVarRender"))
		})

		It("should reject update when def is nil", func() {
			err := f.Svc.UpdateAppCfgFileDef(f.Ctx, nil, appcfg.FileDefUpdate{
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("def is required"))
		})

		It("should reject env instance strategy updates on plain defs", func() {
			result := f.createPlainFile("plain.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			strategy := appcfg.EnvContentStrategyOverwrite
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnvContentStrategy: &strategy,
				Operator:           "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
		})

		It("should change env instance strategy without rewriting files when no env instance exists", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			strategy := appcfg.EnvContentStrategyOverwrite
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnvContentStrategy: &strategy,
				Operator:           "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			gotDef, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.EnvConfigMode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverwrite))

			gotFile, err := f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.CurrentVersion).To(Equal(int64(1)))
			Expect(gotFile.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
		})

		It("should keep raw env content and currentVersion when switching framework strategy to overwrite", func() {
			content := "keep: true\nbase: default\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "framework.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			upserted, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "prod",
				Content:     "base: prod\n",
				Operator:    "editor",
				Description: "prod overlay",
			})
			Expect(err).NotTo(HaveOccurred())

			strategy := appcfg.EnvContentStrategyOverwrite
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnvContentStrategy: &strategy,
				Operator:           "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := f.FileStore.GetByID(f.Ctx, upserted.File.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(got.OverlayContent).To(BeNil())
			Expect(got.BaseAppConfigFileID).NotTo(BeNil())
			Expect(*got.BaseAppConfigFileID).To(Equal(result.ID))
			Expect(lo.FromPtr(got.Content)).To(Equal("base: prod\n"))
			Expect(got.CurrentVersion).To(Equal(int64(1)))

			versions, err := f.Svc.VersionStore.List(f.Ctx, appcfg.VersionsByFile(got.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(HaveLen(1))

			gotDef, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.EnvConfigMode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverwrite))
		})

		It("should keep current content as overlay when switching framework strategy to overlay", func() {
			content := "keep: true\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
				AppType:           "trpc",
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			full := "keep: true\nbase: prod\n"
			upserted, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "prod",
				Content:     full,
				Operator:    "editor",
				Description: "prod full content",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(upserted.File.Type).To(Equal(appcfg.AppConfigFileTypeNormal))

			strategy := appcfg.EnvContentStrategyOverlay
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				EnvContentStrategy: &strategy,
				Operator:           "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := f.FileStore.GetByID(f.Ctx, upserted.File.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))
			Expect(got.Content).To(BeNil())
			Expect(lo.FromPtr(got.OverlayContent)).To(Equal(full))
			Expect(got.BaseAppConfigFileID).NotTo(BeNil())
			Expect(*got.BaseAppConfigFileID).To(Equal(result.ID))
			Expect(got.CurrentVersion).To(Equal(int64(1)))

			versions, err := f.Svc.VersionStore.List(f.Ctx, appcfg.VersionsByFile(got.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(HaveLen(1))
		})

		It("should clean up removed env instances when shrinking mountedEnvNames", func() {
			result := f.createPlainFile("shrink.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 挂载 prod + staging，切到独立配置
			isUnified := false
			mounted := []string{"prod", "staging"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 为 staging 创建环境实例
			upserted, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "staging",
				Content:     "key=staging",
				Operator:    "editor",
				Description: "staging file",
			})
			Expect(err).NotTo(HaveOccurred())
			stagingFile := upserted.File

			// 缩减 mountedEnvNames 到只有 prod
			def, err = f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			newMounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &newMounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// staging 实例应已被清理
			_, err = f.FileStore.GetByID(f.Ctx, stagingFile.ID)
			Expect(err).To(HaveOccurred())

			// 默认文件仍存在
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should clean up removed env instances when shrinking mountedEnvNames from nil", func() {
			result := f.createPlainFile("shrink-nil.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			upserted, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "staging",
				Content:     "key=staging",
				Operator:    "editor",
				Description: "staging file",
			})
			Expect(err).NotTo(HaveOccurred())
			stagingFile := upserted.File

			def, err = f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			newMounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &newMounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.FileStore.GetByID(f.Ctx, stagingFile.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should clean up env instances when switching to unified config", func() {
			result := f.createFrameworkFile("env-test.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 先切换到独立配置
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建一个环境实例
			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   def.ID,
				AppID:   f.AppID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			err = f.FileStore.Create(f.Ctx, &envFile)
			envFileID := envFile.ID
			Expect(err).NotTo(HaveOccurred())

			// 切回统一配置
			def, err = f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified = true
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 环境实例应已被删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认实例仍然存在
			defaultFile, err := f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.EnvName).To(Equal(appcfg.EnvNameDefault))
		})
	})

	Context("DeleteFile (cascade)", func() {
		It("should delete default file along with its def and sibling env instances", func() {
			result := f.createFrameworkFile("cascade.yaml")

			// 添加一个环境实例
			envContent := "env: staging"
			envFile := appcfg.AppConfigFile{
				DefID:   result.Def.ID,
				AppID:   f.AppID,
				EnvName: "staging",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			err := f.FileStore.Create(f.Ctx, &envFile)
			envFileID := envFile.ID
			Expect(err).NotTo(HaveOccurred())

			// 删除默认文件
			_, err = f.Svc.DeleteFile(f.Ctx, f.AppID, result.ID)
			Expect(err).NotTo(HaveOccurred())

			// 默认文件已删除
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).To(HaveOccurred())

			// 环境实例已被级联删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// def 已被删除
			_, err = f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should only delete the env instance when deleting a non-default file", func() {
			result := f.createFrameworkFile("partial.yaml")
			defID := result.Def.ID

			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   f.AppID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			err := f.FileStore.Create(f.Ctx, &envFile)
			envFileID := envFile.ID
			Expect(err).NotTo(HaveOccurred())

			// 仅删除环境实例
			_, err = f.Svc.DeleteFile(f.Ctx, f.AppID, envFileID)
			Expect(err).NotTo(HaveOccurred())

			// 环境实例已删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认文件和 def 仍然存在
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			_, err = f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("SyncFrameworkFileLocation", func() {
		It("should reject an app that does not have exactly one framework def", func() {
			f.createFrameworkFile("one.yaml")
			f.createFrameworkFile("two.yaml")

			err := f.Svc.SyncFrameworkFileLocation(f.Ctx, f.AppID, "taf", "renamed.yaml", "/etc/app")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("2 framework config files"))

			defs, listErr := appcfg.ListDefsByApp(
				f.Ctx, f.DefStore, f.AppID, appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
			)
			Expect(listErr).NotTo(HaveOccurred())
			Expect(defs).To(ConsistOf(
				HaveField("Name", "one.yaml"),
				HaveField("Name", "two.yaml"),
			))
		})

		It("should keep the current name and mount dir when both inputs are empty", func() {
			result := f.createFrameworkFile("keep.yaml")
			mountDir := "/etc/keep"
			err := f.Svc.UpdateAppCfgFileDef(f.Ctx, result.Def, appcfg.FileDefUpdate{
				MountDir: &mountDir,
				AppType:  "taf",
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Svc.SyncFrameworkFileLocation(f.Ctx, f.AppID, "taf", "", "")
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("keep.yaml"))
			Expect(updated.MountDir).To(Equal("/etc/keep"))
		})

		It("should reject a framework rename onto an existing mount identity", func() {
			result := f.createFrameworkFile("trpc_go.yaml")
			mountDir := "/etc/app"
			err := f.Svc.UpdateAppCfgFileDef(f.Ctx, result.Def, appcfg.FileDefUpdate{
				MountDir: &mountDir,
				AppType:  "taf",
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())
			f.createPlainFile("taken.yaml", "/etc/app", "taken")

			err = f.Svc.SyncFrameworkFileLocation(f.Ctx, f.AppID, "taf", "taken.yaml", "")
			Expect(errors.Is(err, appcfg.ErrAppConfigFileDefAlreadyExists)).To(BeTrue())

			unchanged, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.Name).To(Equal("trpc_go.yaml"))
		})

		It("should reject renaming a trpc framework file", func() {
			result := f.createFrameworkFile("trpc_go.yaml")

			err := f.Svc.SyncFrameworkFileLocation(f.Ctx, f.AppID, "trpc", "other.yaml", "")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("does not allow modifying name"))

			unchanged, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.Name).To(Equal("trpc_go.yaml"))
			Expect(unchanged.MountDir).To(BeEmpty())
		})

		It("should update only the mount dir", func() {
			result := f.createFrameworkFile("trpc_go.yaml")

			err := f.Svc.SyncFrameworkFileLocation(f.Ctx, f.AppID, "trpc", "", "/usr/local/trpc/bin/")
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("trpc_go.yaml"))
			Expect(updated.MountDir).To(Equal("/usr/local/trpc/bin"))
		})
	})
})

var _ = Describe("AppCfgFileDefService — UpdateHelmFile", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	It("should rename without writing a file version", func() {
		created := f.createFrameworkFile("values.yaml")
		def, err := f.DefStore.GetByID(f.Ctx, created.Def.ID)
		Expect(err).NotTo(HaveOccurred())
		file, err := f.FileStore.GetByID(f.Ctx, created.ID)
		Expect(err).NotTo(HaveOccurred())

		result, err := f.Svc.UpdateHelmFile(f.Ctx, def, file, appcfg.HelmFileUpdate{
			Name:     "values-b.yaml",
			AppType:  "helm",
			Operator: "editor",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.NameChanged).To(BeTrue())
		Expect(result.FileChanged).To(BeFalse())

		updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Name).To(Equal("values-b.yaml"))
		reloaded, err := f.FileStore.GetByID(f.Ctx, file.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(reloaded.CurrentVersion).To(Equal(int64(1)))
		versions, err := f.Svc.ListVersions(f.Ctx, f.AppID, 1, 10, appcfg.VersionsByFile(file.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(versions.Total).To(Equal(int64(1)))
		Expect(versions.Items[0].Name).To(Equal("values.yaml"))
		Expect(versions.Items[0].Version).To(Equal(int64(1)))
	})

	It("should reject a rename onto an existing mount identity without writing a version", func() {
		first := f.createFrameworkFile("a.yaml")
		second := f.createFrameworkFile("b.yaml")
		def, err := f.DefStore.GetByID(f.Ctx, second.Def.ID)
		Expect(err).NotTo(HaveOccurred())
		file, err := f.FileStore.GetByID(f.Ctx, second.ID)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.Svc.UpdateHelmFile(f.Ctx, def, file, appcfg.HelmFileUpdate{
			Name:     first.Def.Name,
			AppType:  "helm",
			Operator: "editor",
		})
		Expect(errors.Is(err, appcfg.ErrAppConfigFileDefAlreadyExists)).To(BeTrue())

		unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(unchanged.Name).To(Equal("b.yaml"))
		reloaded, err := f.FileStore.GetByID(f.Ctx, file.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(reloaded.CurrentVersion).To(Equal(int64(1)))
	})

	It("should reject renaming a trpc framework file", func() {
		created := f.createFrameworkFile("trpc_go.yaml")
		def, err := f.DefStore.GetByID(f.Ctx, created.Def.ID)
		Expect(err).NotTo(HaveOccurred())
		file, err := f.FileStore.GetByID(f.Ctx, created.ID)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.Svc.UpdateHelmFile(f.Ctx, def, file, appcfg.HelmFileUpdate{
			Name:     "other.yaml",
			AppType:  "trpc",
			Operator: "editor",
		})
		Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())

		unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(unchanged.Name).To(Equal("trpc_go.yaml"))
	})

	It("should write a version when the overlay base changes", func() {
		base := f.createFrameworkFile("base.yaml")
		other := f.createFrameworkFile("other.yaml")
		overlayContent := "foo: bar"
		overlay, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
			AppID:               f.AppID,
			EnvName:             appcfg.EnvNameDefault,
			Name:                "overlay.yaml",
			Type:                appcfg.AppConfigFileTypeOverlay,
			ContentSourceType:   appcfg.ContentSourceTypeLocal,
			Format:              appcfg.FileFormatYAML,
			BaseAppConfigFileID: &base.ID,
			OverlayContent:      &overlayContent,
			Creator:             "tester",
			ConfigKind:          appcfg.ConfigKindFramework,
			AppType:             "helm",
		})
		Expect(err).NotTo(HaveOccurred())
		def, err := f.DefStore.GetByID(f.Ctx, overlay.Def.ID)
		Expect(err).NotTo(HaveOccurred())
		file, err := f.FileStore.GetByID(f.Ctx, overlay.ID)
		Expect(err).NotTo(HaveOccurred())

		result, err := f.Svc.UpdateHelmFile(f.Ctx, def, file, appcfg.HelmFileUpdate{
			Name:                def.Name,
			BaseAppConfigFileID: &other.ID,
			AppType:             "helm",
			Operator:            "editor",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.NameChanged).To(BeFalse())
		Expect(result.FileChanged).To(BeTrue())
		Expect(file.CurrentVersion).To(Equal(int64(2)))
		Expect(file.BaseAppConfigFileID.Hex()).To(Equal(other.ID.Hex()))
	})

	It("should restore the name when the file update fails", func() {
		base := f.createFrameworkFile("base.yaml")
		other := f.createFrameworkFile("other.yaml")
		overlayContent := "foo: bar"
		overlay, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
			AppID:               f.AppID,
			EnvName:             appcfg.EnvNameDefault,
			Name:                "overlay.yaml",
			Type:                appcfg.AppConfigFileTypeOverlay,
			ContentSourceType:   appcfg.ContentSourceTypeLocal,
			Format:              appcfg.FileFormatYAML,
			BaseAppConfigFileID: &base.ID,
			OverlayContent:      &overlayContent,
			Creator:             "tester",
			ConfigKind:          appcfg.ConfigKindFramework,
			AppType:             "helm",
		})
		Expect(err).NotTo(HaveOccurred())
		def, err := f.DefStore.GetByID(f.Ctx, overlay.Def.ID)
		Expect(err).NotTo(HaveOccurred())
		file, err := f.FileStore.GetByID(f.Ctx, overlay.ID)
		Expect(err).NotTo(HaveOccurred())

		staleVersion := file.CurrentVersion + 1
		_, err = f.Svc.UpdateHelmFile(f.Ctx, def, file, appcfg.HelmFileUpdate{
			Name:                   "renamed.yaml",
			BaseAppConfigFileID:    &other.ID,
			AppType:                "helm",
			Operator:               "editor",
			ExpectedCurrentVersion: &staleVersion,
		})
		Expect(errors.Is(err, appcfg.ErrAppConfigFileVersionConflict)).To(BeTrue())

		unchanged, err := f.DefStore.GetByID(f.Ctx, def.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(unchanged.Name).To(Equal("overlay.yaml"))
		reloaded, err := f.FileStore.GetByID(f.Ctx, file.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(reloaded.CurrentVersion).To(Equal(int64(1)))
		Expect(reloaded.BaseAppConfigFileID.Hex()).To(Equal(base.ID.Hex()))
	})
})
