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
})
