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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppCfgFileDefService — Env Content", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	AfterEach(func() {
		f.DiApp.RequireStop()
	})

	Context("UpdateEnvContent", func() {
		It("should create framework env overlay for independent config", func() {
			content := "base: default\nkeep: true\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "framework.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
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

			overlayContent := "base: prod\n"
			updated, compiled, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, "prod", overlayContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			updated, err = f.Svc.CreateFileWithVersion(f.Ctx, *updated, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvName).To(Equal("prod"))
			Expect(updated.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))
			Expect(updated.OverlayContent).NotTo(BeNil())
			Expect(*updated.OverlayContent).To(Equal(overlayContent))
			Expect(updated.BaseAppConfigFileID).NotTo(BeNil())
			Expect(*updated.BaseAppConfigFileID).To(Equal(result.ID))
			Expect(updated.CurrentVersion).To(Equal(int64(1)))
			Expect(compiled).To(ContainSubstring("base: prod"))
			Expect(compiled).To(ContainSubstring("keep: true"))
		})

		It("should update existing framework env overlay content", func() {
			content := "base: default\nkeep: true\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "framework.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
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

			initialOverlay := "base: prod\n"
			created, compiled, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, "prod", initialOverlay, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(compiled).To(ContainSubstring("base: prod"))

			newOverlay := "base: prod-v2\nnewKey: enabled\n"
			updated, compiled, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, "prod", newOverlay, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeFalse())
			err = f.Svc.UpdateFile(f.Ctx, updated, def.Name, "editor", appcfg.UpdateCfgFileOptions{
				OperationType:          appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:            "update prod overlay",
				ExpectedCurrentVersion: lo.ToPtr(created.CurrentVersion),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.ID).To(Equal(created.ID))
			Expect(updated.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))
			Expect(updated.OverlayContent).NotTo(BeNil())
			Expect(*updated.OverlayContent).To(Equal(newOverlay))
			Expect(updated.CurrentVersion).To(Equal(int64(2)))
			Expect(compiled).To(ContainSubstring("base: prod-v2"))
			Expect(compiled).To(ContainSubstring("keep: true"))
			Expect(compiled).To(ContainSubstring("newKey: enabled"))
		})

		It("should create plain env file with overwrite strategy", func() {
			result := f.createPlainFile("plain.yaml", "/data", "key: default\n")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			envContent := "key: prod\n"
			updated, compiled, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, "prod", envContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			updated, err = f.Svc.CreateFileWithVersion(f.Ctx, *updated, def.Name, "create plain prod file", "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvName).To(Equal("prod"))
			Expect(updated.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(updated.Content).NotTo(BeNil())
			Expect(*updated.Content).To(Equal(envContent))
			Expect(updated.CurrentVersion).To(Equal(int64(1)))
			Expect(compiled).To(Equal(envContent))
		})

		It("should write to default file for unified config", func() {
			content := "base: default\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "unified.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			// 保持统一配置（默认就是）

			newContent := "base: updated\n"
			updated, _, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, "prod", newContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeFalse())
			// 统一配置时写到默认文件
			Expect(updated.ID).To(Equal(result.ID))
			Expect(updated.EnvName).To(Equal(appcfg.EnvNameDefault))
		})

		It("should write to default file for default env name", func() {
			content := "base: default\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "direct-default.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newContent := "base: updated\n"
			updated, _, isNewFile, err := f.Svc.PrepareEnvContentUpdate(
				f.Ctx, def, appcfg.EnvNameDefault, newContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeFalse())
			Expect(updated.ID).To(Equal(result.ID))
		})

		It("should reject writing to unmounted env even in unified config mode", func() {
			result := f.createPlainFile("write-limited.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			_, _, _, err = f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "staging", "new-content", "editor")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("not effective"))
		})

		It("should succeed with UpsertEnvContent for new file creation", func() {
			content := "base: default\nkeep: true\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "upsert-new.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
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

			upsertResult, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "prod",
				Content:     "base: prod\n",
				Operator:    "editor",
				Description: "create prod",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(upsertResult.File).NotTo(BeNil())
			Expect(upsertResult.File.EnvName).To(Equal("prod"))
			Expect(upsertResult.CompiledContent).To(ContainSubstring("base: prod"))
			Expect(upsertResult.CompiledContent).To(ContainSubstring("keep: true"))
		})

		It("should succeed with UpsertEnvContent for existing file update", func() {
			content := "base: default\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "upsert-update.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
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

			// 第一次 upsert 创建
			firstResult, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "prod",
				Content:     "base: prod-v1\n",
				Operator:    "editor",
				Description: "create prod",
			})
			Expect(err).NotTo(HaveOccurred())

			// 第二次 upsert 更新
			secondResult, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:                "prod",
				Content:                "base: prod-v2\n",
				Operator:               "editor",
				Description:            "update prod",
				ExpectedCurrentVersion: lo.ToPtr(firstResult.File.CurrentVersion),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(secondResult.File.ID).To(Equal(firstResult.File.ID))
			Expect(secondResult.CompiledContent).To(ContainSubstring("base: prod-v2"))
		})

		It("should reject when ValidateCompiledContent callback returns error", func() {
			content := "base: default\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "validate-cb.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     appcfg.EnvNameDefault,
				Content:     "base: updated\n",
				Operator:    "editor",
				Description: "should fail validation",
				ValidateCompiledContent: func(_ *appcfg.AppConfigFile, _ string) error {
					return fmt.Errorf("custom validation failed")
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("custom validation failed"))
		})

		It("should return target file in result when upsert hits version conflict", func() {
			content := "base: default\nkeep: true\n"
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "framework.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "init",
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

			initialOverlay := "base: prod\n"
			created, _, isNewFile, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", initialOverlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			conflictResult, err := f.Svc.UpsertEnvContent(f.Ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:                "prod",
				Content:                "base: prod-v2\n",
				Operator:               "editor",
				Description:            "conflict update",
				ExpectedCurrentVersion: lo.ToPtr(created.CurrentVersion - 1),
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrAppConfigFileVersionConflict)).To(BeTrue())
			Expect(conflictResult).NotTo(BeNil())
			Expect(conflictResult.File).NotTo(BeNil())
			Expect(conflictResult.File.ID).To(Equal(created.ID))
		})
	})
})
