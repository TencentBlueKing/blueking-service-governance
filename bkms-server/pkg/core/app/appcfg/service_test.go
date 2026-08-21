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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppConfigFileService", func() {
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var fileStore appcfg.AppConfigFileStore
	var versionStore appcfg.AppConfigFileVersionStore
	var service *appcfg.AppConfigFileService
	var ctx context.Context
	var appID string

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		err = testutil.CleanupCollection("app_config_file_versions")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_config_files")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &fileStore, &versionStore),
		)
		diApp.RequireStart()

		app := dbfactory.Application(ctx, appStore)
		appID = app.ID

		service = appcfg.NewAppConfigFileService(fileStore, versionStore)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("Create", func() {
		It("should persist the file and its initial version", func() {
			content := "foo: bar"

			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "initial values",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(acf.ID).NotTo(Equal(bson.NilObjectID))
			Expect(acf.CurrentVersion).To(Equal(int64(1)))

			// 验证文件已持久化到数据库
			gotFile, err := fileStore.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.AppID).To(Equal(appID))
			Expect(gotFile.CurrentVersion).To(Equal(int64(1)))

			// 验证初始版本已持久化到数据库
			items, _, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Description).To(Equal("initial values"))
			Expect(items[0].Version).To(Equal(int64(1)))
		})

		It("should persist new semantic fields for plain config files", func() {
			content := "KEY=VALUE"

			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "custom-env",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
				Description:       "initial env file",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(acf.GetConfigKind()).To(Equal(appcfg.ConfigKindPlain))
			Expect(acf.MountPath).To(Equal("/data/app/conf/custom.env"))
			Expect(acf.IsUnifiedConfig).To(BeTrue())

			gotFile, err := fileStore.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.GetConfigKind()).To(Equal(appcfg.ConfigKindPlain))
			Expect(gotFile.MountPath).To(Equal("/data/app/conf/custom.env"))
		})

		It("should reject plain config with bscp source", func() {
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "custom-env",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeBSCP,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				Creator:           "tester",
			})

			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("plain config only supports local content source")))
		})

		It("should reject plain config mountPath conflicts within one app", func() {
			content := "KEY=VALUE"
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "custom-env",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "another-file",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("json"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})

			Expect(err).To(MatchError(appcfg.ErrPlainConfigMountPathConflict))
		})
	})

	Context("UpdateFile", func() {
		It("should update version metadata and append a new version", func() {
			content := "foo: bar"

			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "initial values",
			})
			Expect(err).NotTo(HaveOccurred())

			newContent := "foo: baz"
			acf.Content = &newContent

			err = service.UpdateFile(ctx, acf, "new-user", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "update values",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(acf.CurrentVersion).To(Equal(int64(2)))
			Expect(acf.Updater).To(Equal("new-user"))

			// 验证数据库中的文件已更新
			gotFile, err := fileStore.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.CurrentVersion).To(Equal(int64(2)))
			Expect(gotFile.Updater).To(Equal("new-user"))

			// 验证新版本已持久化
			items, _, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(2))

			var latestVersion appcfg.AppConfigFileVersion
			for _, v := range items {
				if v.Version == 2 {
					latestVersion = v
				}
			}
			Expect(latestVersion.Version).To(Equal(int64(2)))
			Expect(latestVersion.Description).To(Equal("update values"))
		})

		It("should reject updating a plain config to a conflicting mountPath", func() {
			content := "KEY=VALUE"
			first, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "custom-env",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			second, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "feature-flags",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("json"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/feature-flags.json",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			second.MountPath = first.MountPath
			err = service.UpdateFile(ctx, second, "new-user", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "conflict mount path",
			})

			Expect(err).To(MatchError(appcfg.ErrPlainConfigMountPathConflict))
		})
	})

	Context("Rollback", func() {
		It("should restore the historical content and create a rollback version", func() {
			content1 := "foo: v1"
			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content1,
				Creator:           "tester",
				Description:       "initial values",
			})
			Expect(err).NotTo(HaveOccurred())

			// 更新到 v2
			content2 := "foo: v2"
			acf.Content = &content2
			err = service.UpdateFile(ctx, acf, "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "second version",
			})
			Expect(err).NotTo(HaveOccurred())

			// 更新到 v3
			content3 := "foo: v3"
			acf.Content = &content3
			err = service.UpdateFile(ctx, acf, "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "third version",
			})
			Expect(err).NotTo(HaveOccurred())

			// 获取 v2 的版本 ID
			items, _, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			var targetVersionID bson.ObjectID
			for _, v := range items {
				if v.Version == 2 {
					targetVersionID = v.ID
				}
			}

			// 回滚到 v2
			rollbackAcf, targetVersion, err := service.Rollback(ctx, appID, targetVersionID, "operator", "", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(targetVersion.Version).To(Equal(int64(2)))
			Expect(rollbackAcf.CurrentVersion).To(Equal(int64(4)))
			Expect(rollbackAcf.Updater).To(Equal("operator"))
			Expect(rollbackAcf.Content).NotTo(BeNil())
			Expect(*rollbackAcf.Content).To(Equal(content2))

			// 验证回滚版本已持久化
			items, _, err = versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(4))

			var rollbackVersion appcfg.AppConfigFileVersion
			for _, v := range items {
				if v.Version == 4 {
					rollbackVersion = v
				}
			}
			Expect(rollbackVersion.OperationType).To(Equal(appcfg.AppConfigFileVersionOperationTypeRollback))
			Expect(*rollbackVersion.RollbackFromVersion).To(Equal(int64(2)))
			Expect(rollbackVersion.Version).To(Equal(int64(4)))
		})
	})

	Context("DeleteFile", func() {
		It("should delete the file and hard-delete all its version records", func() {
			content := "foo: bar"
			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "initial values",
			})
			Expect(err).NotTo(HaveOccurred())

			// update to create a second version
			newContent := "foo: baz"
			acf.Content = &newContent
			err = service.UpdateFile(ctx, acf, "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "second version",
			})
			Expect(err).NotTo(HaveOccurred())

			// delete the file
			deletedAcf, err := service.DeleteFile(ctx, appID, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(deletedAcf.Name).To(Equal("values"))

			// file should be gone
			_, err = fileStore.GetByID(ctx, acf.ID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))

			// version records should be fully removed
			items, total, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())

			items, total, err = versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:          appID,
				Page:           1,
				PageSize:       10,
				IncludeDeleted: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())

			// recreating the same file should start fresh without version conflicts
			newAcf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester2",
				Description:       "recreated",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(newAcf.CurrentVersion).To(Equal(int64(1)))

			// only the new file's version should be visible
			items, total, err = versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(items[0].AppConfigFileID).To(Equal(newAcf.ID))
		})
	})

	Context("DeleteVersion", func() {
		It("should reject deleting the current active version", func() {
			content := "foo: bar"
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "values",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "initial values",
			})
			Expect(err).NotTo(HaveOccurred())

			// 获取当前活跃版本
			items, _, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			currentVersionID := items[0].ID

			_, _, err = service.DeleteVersion(ctx, appID, currentVersionID, "tester")

			Expect(err).To(MatchError(appcfg.ErrUsingVersionCannotBeDeleted))
		})
	})

	Context("UpdateEnvConfig", func() {
		var defaultFile *appcfg.AppConfigFile

		createPlainDefault := func() *appcfg.AppConfigFile {
			content := "feature.enabled=true"
			acf, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "custom-env",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			return acf
		}

		createPlainEnv := func(root *appcfg.AppConfigFile, envName, content string) *appcfg.AppConfigFile {
			envFile, err := service.CreatePlainEnvInstance(
				ctx, *root, envName, &content, "tester", "create env instance",
			)
			Expect(err).NotTo(HaveOccurred())
			return envFile
		}

		expectNotFound := func(id bson.ObjectID) {
			_, err := fileStore.GetByID(ctx, id)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		}

		BeforeEach(func() {
			defaultFile = createPlainDefault()
		})

		It("should enable independent env config without creating env instances", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
				Description:     "enable independent env config",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsUnifiedConfig).To(BeFalse())
			Expect(got.MountedEnvNames).To(Equal([]string{"prod", "stag"}))
			Expect(got.CurrentVersion).To(Equal(int64(2)))

			files, err := fileStore.List(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
		})

		It("should enable independent env config for all envs", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				Operator:        "tester",
				Description:     "enable independent env config for all envs",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsUnifiedConfig).To(BeFalse())
			Expect(got.MountedEnvNames).To(BeNil())
		})

		It("should switch back to unified config and delete all env instances", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			prodFile := createPlainEnv(defaultFile, "prod", "feature.enabled=false")
			stagFile := createPlainEnv(defaultFile, "stag", "feature.enabled=gray")

			err = service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				Operator:        "tester",
				Description:     "switch to unified config",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.IsUnifiedConfig).To(BeTrue())
			Expect(defaultFile.MountedEnvNames).To(BeNil())

			expectNotFound(prodFile.ID)
			expectNotFound(stagFile.ID)
			items, total, err := versionStore.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &prodFile.ID,
				Page:            1,
				PageSize:        10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())
		})

		It("should restore specified env instances to reference state", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			prodFile := createPlainEnv(defaultFile, "prod", "feature.enabled=false")
			stagFile := createPlainEnv(defaultFile, "stag", "feature.enabled=gray")

			err = service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig:   false,
				MountedEnvNames:   []string{"prod", "stag"},
				FallbackConfigEnv: "prod",
				Operator:          "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			expectNotFound(prodFile.ID)
			_, err = fileStore.GetByID(ctx, stagFile.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject fallback on unified config", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig:   false,
				FallbackConfigEnv: "prod",
				Operator:          "tester",
			})
			Expect(errors.Is(err, appcfg.ErrFallbackRequiresIndependentConfig)).To(BeTrue())
		})

		It("should keep IsUnifiedConfig unchanged on generic update", func() {
			content := "feature.enabled=true"
			defaultFile.Content = &content
			err := service.UpdateFile(ctx, defaultFile, "tester", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "keep unified config unchanged",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsUnifiedConfig).To(BeTrue())
		})

		It("should clean up stale env instances when narrowing mount scope", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			prodFile := createPlainEnv(defaultFile, "prod", "feature.enabled=false")
			stagFile := createPlainEnv(defaultFile, "stag", "feature.enabled=gray")

			err = service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod"}))

			_, err = fileStore.GetByID(ctx, prodFile.ID)
			Expect(err).NotTo(HaveOccurred())
			expectNotFound(stagFile.ID)
		})

		It("should keep mountedEnvNames when switching back to unified config", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.IsUnifiedConfig).To(BeTrue())
			Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod"}))
		})

		It("should update mountedEnvNames while already in unified config", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsUnifiedConfig).To(BeTrue())
			Expect(got.MountedEnvNames).To(Equal([]string{"prod"}))
		})

		It("should reject creating a plain env record without defaultAppConfigFileID", func() {
			content := "feature.enabled=true"
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "orphan-plain",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormat("env"),
				ConfigKind:        appcfg.ConfigKindPlain,
				MountPath:         "/data/app/conf/orphan.env",
				IsUnifiedConfig:   true,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("plain env instance requires defaultAppConfigFileID")))
		})

		It("should reject overlay whose base is a plain config file", func() {
			overlayContent := "patches:\n- foo: 1\n"
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:               appID,
				EnvName:             appcfg.EnvNameDefault,
				Name:                "plain-overlay",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				ConfigKind:          appcfg.ConfigKindFramework,
				BaseAppConfigFileID: &defaultFile.ID,
				OverlayContent:      &overlayContent,
				Creator:             "tester",
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("overlay base must be a framework config file")))
		})

		It("should create independent env instance on first modification", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			prodContent := "feature.enabled=false"
			envFile, err := service.CreatePlainEnvInstance(
				ctx, *defaultFile, "prod", &prodContent, "tester", "create independent env instance",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(envFile.EnvName).To(Equal("prod"))
			Expect(envFile.DefaultAppConfigFileID).NotTo(BeNil())
			Expect(*envFile.DefaultAppConfigFileID).To(Equal(defaultFile.ID))
			Expect(*envFile.Content).To(Equal("feature.enabled=false"))

			got, err := fileStore.GetByID(ctx, envFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(*got.Content).To(Equal("feature.enabled=false"))
		})

		It("should reject creating a plain env instance for an unmounted env", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			content := "feature.enabled=false"
			_, err = service.CreatePlainEnvInstance(ctx, *defaultFile, "stag", &content, "tester", "lazy create")
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("not in mountedEnvNames")))
		})

		It("should reject mountPath change on plain env instance", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			prodFile := createPlainEnv(defaultFile, "prod", "feature.enabled=false")
			prodFile.MountPath = "/data/app/conf/other.env"

			err = service.UpdateFile(ctx, prodFile, "tester", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("plain env instance cannot change mountPath")))
		})

		It("should find existing plain env instance for content update", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			prodFile := createPlainEnv(defaultFile, "prod", "feature.enabled=false")

			found, err := service.FindPlainEnvInstance(ctx, *defaultFile, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(found.ID).To(Equal(prodFile.ID))
			Expect(*found.Content).To(Equal("feature.enabled=false"))
		})

		It("should clean mountedEnvNames for roots without env instances when env is deleted", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.CleanupPlainEnvInstancesByEnv(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.MountedEnvNames).To(Equal([]string{"stag"}))
		})

		It("should clear mountedEnvNames to empty slice when the last scoped env is deleted", func() {
			err := service.UpdateEnvConfig(ctx, defaultFile, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.CleanupPlainEnvInstancesByEnv(ctx, appID, "prod")
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, defaultFile.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.MountedEnvNames).NotTo(BeNil())
			Expect(got.MountedEnvNames).To(Equal([]string{}))
		})

		It("should reject creating a framework file with mountedEnvNames", func() {
			content := "global:\n  namespace: Production\n"
			_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				MountedEnvNames:   []string{"prod"},
				Content:           &content,
				Creator:           "tester",
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("framework config does not support mountedEnvNames")))
		})

		DescribeTable("should reject invalid plain mountPath",
			func(mountPath string) {
				content := "feature.enabled=true"
				_, err := service.Create(ctx, appcfg.CreateCfgFileParams{
					AppID:             appID,
					EnvName:           appcfg.EnvNameDefault,
					Name:              "invalid-mount",
					Type:              appcfg.AppConfigFileTypeNormal,
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormat("env"),
					ConfigKind:        appcfg.ConfigKindPlain,
					MountPath:         mountPath,
					Content:           &content,
					Creator:           "tester",
				})
				Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
				Expect(err).To(MatchError(ContainSubstring("absolute file path")))
			},
			Entry("relative path", "data/config/custom.env"),
			Entry("root path", "/"),
			Entry("trailing slash", "/data/config/"),
		)

		It("should delete framework env files without BaseAppConfigFileID when switching to unified", func() {
			fwContent := "global:\n  namespace: Production\n"
			fwDefault, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				IsUnifiedConfig:   false,
				Content:           &fwContent,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			envContent := "global:\n  namespace: ProdEnv\n"
			envFile, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           "prod",
				Name:              "trpc_go.yaml--prod",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				Content:           &envContent,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.UpdateEnvConfig(ctx, fwDefault, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(fwDefault.IsUnifiedConfig).To(BeTrue())
			expectNotFound(envFile.ID)
		})

		It("should delete framework env overlays derived from the default file when switching to unified", func() {
			fwContent := "global:\n  namespace: Production\n"
			fwDefault, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				IsUnifiedConfig:   false,
				Content:           &fwContent,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			overlayContent := "patches:\n- path: /global/namespace\n"
			overlayFile, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:               appID,
				EnvName:             "prod",
				Name:                "trpc_go.yaml-prod-overlay",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				ConfigKind:          appcfg.ConfigKindFramework,
				BaseAppConfigFileID: &fwDefault.ID,
				OverlayContent:      &overlayContent,
				Creator:             "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.UpdateEnvConfig(ctx, fwDefault, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			expectNotFound(overlayFile.ID)
		})

		It("should not delete helm overlay with empty envName when switching framework to unified", func() {
			fwContent := "global:\n  namespace: Production\n"
			fwDefault, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				IsUnifiedConfig:   false,
				Content:           &fwContent,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			overlayContent := "image:\n  tag: v1\n"
			helmOverlay, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:               appID,
				EnvName:             appcfg.EnvNameDefault,
				Name:                "values-prod.yaml",
				Type:                appcfg.AppConfigFileTypeOverlay,
				ContentSourceType:   appcfg.ContentSourceTypeLocal,
				Format:              appcfg.FileFormatYAML,
				ConfigKind:          appcfg.ConfigKindFramework,
				BaseAppConfigFileID: &fwDefault.ID,
				OverlayContent:      &overlayContent,
				Creator:             "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.UpdateEnvConfig(ctx, fwDefault, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				Operator:        "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := fileStore.GetByID(ctx, helmOverlay.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("values-prod.yaml"))
		})

		It("should reject updating framework env config with mountedEnvNames", func() {
			fwContent := "global:\n  namespace: Production\n"
			fwDefault, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "trpc_go.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				ConfigKind:        appcfg.ConfigKindFramework,
				IsUnifiedConfig:   false,
				Content:           &fwContent,
				Creator:           "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = service.UpdateEnvConfig(ctx, fwDefault, appcfg.UpdateEnvConfigParams{
				IsUnifiedConfig: true,
				MountedEnvNames: []string{"prod"},
				Operator:        "tester",
			})
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("framework config does not support mountedEnvNames")))
		})
	})
})
