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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AppConfigFileService", func() {
	var diApp *fxtest.App
	var appStore *bkmsapp.ApplicationStore
	var fileStore *appcfg.AppConfigFileStore
	var versionStore *appcfg.AppConfigFileVersionStore
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
		err = testutil.CleanupCollection("app_config_file_defs")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			appcfg.FxModule,
			fx.Populate(),
		)
		diApp.RequireStart()
		db := testutil.NewDB(GinkgoT())
		versionStore = store.Of(db, appcfg.AppConfigFileVersionMeta)

		app := dbfactory.Application(ctx, appStore)
		appID = app.ID

		service = appcfg.NewAppConfigFileService(db)
		fileStore = testutil.NewStore(GinkgoT(), appcfg.AppConfigFileMeta)
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
				ConfigKind:        appcfg.ConfigKindFramework,
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
			items, err := versionStore.List(ctx, appcfg.VersionsByApp(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Description).To(Equal("initial values"))
			Expect(items[0].Version).To(Equal(int64(1)))
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
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			newContent := "foo: baz"
			acf.Content = &newContent

			err = service.UpdateFile(ctx, acf, "", "new-user", appcfg.UpdateCfgFileOptions{
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
			items, err := versionStore.List(ctx, appcfg.VersionsByApp(appID))
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
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			// 更新到 v2
			content2 := "foo: v2"
			acf.Content = &content2
			err = service.UpdateFile(ctx, acf, "", "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "second version",
			})
			Expect(err).NotTo(HaveOccurred())

			// 更新到 v3
			content3 := "foo: v3"
			acf.Content = &content3
			err = service.UpdateFile(ctx, acf, "", "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "third version",
			})
			Expect(err).NotTo(HaveOccurred())

			// 获取 v2 的版本 ID
			items, err := versionStore.List(ctx, appcfg.VersionsByApp(appID))
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
			items, err = versionStore.List(ctx, appcfg.VersionsByApp(appID))
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

		It("should reject rollback when target version type differs from current file type", func() {
			defaultContent := "keep: true\nbase: default\n"
			created, err := service.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "values.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &defaultContent,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := service.DefStore.GetByID(ctx, created.DefID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = service.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			upserted, err := service.UpsertEnvContent(ctx, def, appcfg.UpsertEnvContentParams{
				EnvName:     "prod",
				Content:     "base: prod\n",
				Operator:    "editor",
				Description: "prod overlay",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(upserted.File.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))

			versions, err := versionStore.List(ctx, appcfg.VersionsByFile(upserted.File.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(HaveLen(1))
			targetVersionID := versions[0].ID

			strategy := appcfg.EnvContentStrategyOverwrite
			err = service.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				EnvContentStrategy: &strategy,
				Operator:           "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			current, err := fileStore.GetByID(ctx, upserted.File.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(current.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(current.CurrentVersion).To(Equal(int64(1)))

			_, _, err = service.Rollback(ctx, appID, targetVersionID, "operator", "", nil)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("current file type"))
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
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			// update to create a second version
			newContent := "foo: baz"
			acf.Content = &newContent
			err = service.UpdateFile(ctx, acf, "", "updater", appcfg.UpdateCfgFileOptions{
				OperationType: appcfg.AppConfigFileVersionOperationTypeUpdate,
				Description:   "second version",
			})
			Expect(err).NotTo(HaveOccurred())

			// delete the file
			deletedAcf, err := service.DeleteFile(ctx, appID, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(deletedAcf).NotTo(BeNil())

			// file should be gone
			_, err = fileStore.GetByID(ctx, acf.ID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrAppConfigFileNotFound)).To(BeTrue())

			// version records should be fully removed
			Expect(versionStore.Count(ctx, appcfg.VersionsByApp(appID))).To(BeZero())

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
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(newAcf.CurrentVersion).To(Equal(int64(1)))

			// only the new file's version should be visible
			page, err := service.ListVersions(ctx, appID, 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.Items[0].AppConfigFileID).To(Equal(newAcf.ID))
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
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).NotTo(HaveOccurred())

			// 获取当前活跃版本
			items, err := versionStore.List(ctx, appcfg.VersionsByApp(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			currentVersionID := items[0].ID

			_, _, err = service.DeleteVersion(ctx, appID, currentVersionID, "tester")

			Expect(err).To(MatchError(appcfg.ErrUsingVersionCannotBeDeleted))
		})
	})
})
