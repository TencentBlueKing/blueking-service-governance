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
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppCfgFileDefService", func() {
	var diApp *fxtest.App
	var app *bkmsapp.Application
	var appStore bkmsapp.ApplicationStore
	var fileStore appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore
	var svc *appcfg.AppCfgFileDefService
	var ctx context.Context
	var appID string

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupCollection("app_config_file_versions")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_files")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_file_defs")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &fileStore, &defStore, &versionStore),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		appID = app.ID

		base := appcfg.NewBaseAppCfgFileService(defStore, fileStore, versionStore)
		svc = appcfg.NewAppCfgFileDefService(base)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	createFile := func(name string) *appcfg.AppConfigFileWithDef {
		content := "key: value"
		result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
			AppID:             appID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              name,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &content,
			Creator:           "tester",
			Description:       "init",
			ConfigKind:        appcfg.ConfigKindFramework,
		})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	createPlainFile := func(name, mountDir, content string) *appcfg.AppConfigFileWithDef {
		result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
			AppID:             appID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              name,
			MountDir:          mountDir,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &content,
			Creator:           "tester",
			Description:       "init",
			ConfigKind:        appcfg.ConfigKindPlain,
		})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	Context("Create", func() {
		It("should create def, file and initial version", func() {
			result := createFile("values.yaml")

			Expect(result.Def).NotTo(BeNil())
			Expect(result.Def.Name).To(Equal("values.yaml"))
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindFramework))
			Expect(result.Def.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
			Expect(result.DefID).To(Equal(result.Def.ID))
			Expect(result.CurrentVersion).To(Equal(int64(1)))

			gotDef, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.Name).To(Equal("values.yaml"))

			gotFile, err := fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.DefID).To(Equal(result.Def.ID))
		})

		It("should default empty config kind to framework", func() {
			content := "key: value"
			result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "legacy-values.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "legacy init",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindFramework))
		})

		It("should reject invalid YAML content", func() {
			content := "invalid: [yaml"
			_, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
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
			Expect(err.Error()).To(ContainSubstring("content validation"))
		})

		It("should accept non-YAML content for plain kind", func() {
			content := "this is { not yaml ["
			result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
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
	})

	Context("UpdateAppCfgFileDef", func() {
		It("should update name", func() {
			result := createFile("old-name.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := "new-name.yaml"
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := defStore.GetByID(ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-name.yaml"))
		})

		It("should reject mountDir update for framework kind", func() {
			result := createFile("fw.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mountDir"))
		})

		It("should allow mountDir update for plain kind", func() {
			result := createPlainFile("editable.conf", "/old/path", "content")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := defStore.GetByID(ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(Equal("/new/path"))
		})

		It("should clean up env instances when switching to unified config", func() {
			result := createFile("env-test.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 先切换到独立配置
			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建一个环境实例
			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   def.ID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 切回统一配置
			def, err = defStore.GetByID(ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified = true
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 环境实例应已被删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认实例仍然存在
			defaultFile, err := fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.EnvName).To(Equal(appcfg.EnvNameDefault))
		})
	})

	Context("DeleteFile (cascade)", func() {
		It("should delete default file along with its def and sibling env instances", func() {
			result := createFile("cascade.yaml")

			// 添加一个环境实例
			envContent := "env: staging"
			envFile := appcfg.AppConfigFile{
				DefID:   result.Def.ID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 删除默认文件
			_, err = svc.DeleteFile(ctx, appID, result.ID)
			Expect(err).NotTo(HaveOccurred())

			// 默认文件已删除
			_, err = fileStore.GetByID(ctx, result.ID)
			Expect(err).To(HaveOccurred())

			// 环境实例已被级联删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// def 已被删除
			_, err = defStore.GetByID(ctx, result.Def.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should only delete the env instance when deleting a non-default file", func() {
			result := createFile("partial.yaml")
			defID := result.Def.ID

			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 仅删除环境实例
			_, err = svc.DeleteFile(ctx, appID, envFileID)
			Expect(err).NotTo(HaveOccurred())

			// 环境实例已删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认文件和 def 仍然存在
			_, err = fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			_, err = defStore.GetByID(ctx, defID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("UpdateEnvContent", func() {
		It("should create framework env overlay for independent config", func() {
			content := "base: default\nkeep: true\n"
			result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
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

			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlayContent := "base: prod\n"
			updated, compiled, isNewFile, err := svc.PrepareEnvContentUpdate(
				ctx, def, "prod", overlayContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			updated, err = svc.CreateFileWithVersion(ctx, *updated, def.Name, "create prod overlay", "editor")
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
			result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
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

			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			initialOverlay := "base: prod\n"
			created, compiled, isNewFile, err := svc.PrepareEnvContentUpdate(
				ctx, def, "prod", initialOverlay, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(compiled).To(ContainSubstring("base: prod"))

			newOverlay := "base: prod-v2\nnewKey: enabled\n"
			updated, compiled, isNewFile, err := svc.PrepareEnvContentUpdate(
				ctx, def, "prod", newOverlay, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeFalse())
			err = svc.UpdateFile(ctx, updated, def.Name, "editor", appcfg.UpdateCfgFileOptions{
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
			result := createPlainFile("plain.yaml", "/data", "key: default\n")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			envContent := "key: prod\n"
			updated, compiled, isNewFile, err := svc.PrepareEnvContentUpdate(
				ctx, def, "prod", envContent, "editor",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			updated, err = svc.CreateFileWithVersion(ctx, *updated, def.Name, "create plain prod file", "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvName).To(Equal("prod"))
			Expect(updated.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(updated.Content).NotTo(BeNil())
			Expect(*updated.Content).To(Equal(envContent))
			Expect(updated.CurrentVersion).To(Equal(int64(1)))
			Expect(compiled).To(Equal(envContent))
		})

		It("should return target file in result when upsert hits version conflict", func() {
			content := "base: default\nkeep: true\n"
			result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
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

			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			initialOverlay := "base: prod\n"
			created, _, isNewFile, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", initialOverlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			conflictResult, err := svc.UpsertEnvContent(ctx, def, appcfg.UpsertEnvContentParams{
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

	Context("GetEnvFileDetail", func() {
		It("should return default file for unified config", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			detail, err := svc.GetEnvFileDetail(ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(detail.DefaultFile.ID))
			Expect(detail.DisplayFile.ID).To(Equal(result.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})

		It("should return nil display file for framework overlay strategy without env instance", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := svc.GetEnvFileDetail(ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).To(BeNil())
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("none"))
		})

		It("should return default file for plain overwrite strategy without env instance", func() {
			result := createPlainFile("app.conf", "/data", "k=v")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := svc.GetEnvFileDetail(ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(detail.DefaultFile.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})

		It("should return env file when env instance exists", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, isNewFile, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			detail, err := svc.GetEnvFileDetail(ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(created.ID))
			Expect(detail.HasEnvInstance).To(BeTrue())
			Expect(detail.EditableContentField).To(Equal("overlayContent"))
			Expect(detail.BaseContentInfo).NotTo(BeNil())
			Expect(detail.BaseContentInfo.HolderID).To(Equal(result.ID))
			Expect(detail.BaseContentInfo.HolderName).To(Equal("values.yaml"))
			Expect(detail.BaseContentInfo.HolderContentSourceType).To(Equal("local"))
			Expect(detail.BaseContentInfo.IsFromAnotherFile).To(BeTrue())
			Expect(detail.BaseContentInfo.Content).To(ContainSubstring("key: value"))
		})

		It("should return error when plain file is not effective for env", func() {
			result := createPlainFile("app.conf", "/data", "k=v")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = svc.GetEnvFileDetail(ctx, def, "staging")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not effective"))
		})

		It("should return default file when querying default env name", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := svc.GetEnvFileDetail(ctx, def, appcfg.EnvNameDefault)
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(result.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})
	})

	Context("GetMountPreview", func() {
		It("should include framework defs for any env", func() {
			createFile("values.yaml")

			items, err := svc.GetMountPreview(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Def.Name).To(Equal("values.yaml"))
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(items[0].HasEnvFile).To(BeFalse())
		})

		It("should include plain defs only for mounted envs", func() {
			result := createPlainFile("app.conf", "/data", "k=v")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			mounted := []string{"prod"}
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			items, err := svc.GetMountPreview(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Def.Name).To(Equal("app.conf"))

			items, err = svc.GetMountPreview(ctx, appID, "staging")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(BeEmpty())
		})

		It("should mark HasEnvFile when env instance exists", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, _, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := svc.GetMountPreview(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
		})

		It("should show overlay content source for framework env instance", func() {
			result := createFile("values.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, _, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := svc.GetMountPreview(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileTypeOverlay))
		})

		It("should show overwrite content source for plain env instance", func() {
			result := createPlainFile("app.conf", "/data", "k=v")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			created, _, _, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", "key=prod", "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = svc.CreateFileWithVersion(ctx, *created, def.Name, "plain prod overwrite", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := svc.GetMountPreview(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileType("overwrite")))
		})
	})
})
