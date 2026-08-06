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
	"time"

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

var _ = Describe("AppConfigFileVersionStoreMongo", func() {
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var fileStore appcfg.AppConfigFileStore
	var store appcfg.AppConfigFileVersionStore
	var ctx context.Context
	var appID string
	var fileID bson.ObjectID

	newVersion := func(version int64, createdAt time.Time, creator, description string) appcfg.AppConfigFileVersion {
		return dbfactory.AppConfigFileVersion(&dbfactory.AppConfigFileVersionOpts{
			AppConfigFileID: fileID,
			AppID:           appID,
			Version:         version,
			Description:     description,
			Creator:         creator,
			CreatedAt:       createdAt,
		})
	}

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
			fx.Populate(&appStore, &fileStore, &store),
		)
		diApp.RequireStart()

		app := dbfactory.Application(ctx, appStore)
		fileID = bson.NewObjectID()
		_, err = fileStore.Add(ctx, appcfg.AppConfigFile{
			ID: fileID,
			AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
				AppID:             app.ID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "values-main",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		appID = app.ID
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("Add and Get", func() {
		It("should add and load a version record successfully", func() {
			baseID := bson.NewObjectID()
			baseVersion := int64(3)
			rollbackFromVersion := int64(1)
			content := "foo: v2"
			version := appcfg.AppConfigFileVersion{
				AppConfigFileID: fileID,
				AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
					AppID:               appID,
					EnvName:             "prod",
					Name:                "values",
					Type:                appcfg.AppConfigFileTypeOverlay,
					ContentSourceType:   appcfg.ContentSourceTypeLocal,
					Format:              appcfg.FileFormatYAML,
					OverlayContent:      &content,
					BaseAppConfigFileID: &baseID,
					Creator:             "tester",
					CreatedAt:           time.Now(),
				},
				Version:             2,
				Description:         "second version",
				BaseVersion:         &baseVersion,
				OperationType:       appcfg.AppConfigFileVersionOperationTypeRollback,
				RollbackFromVersion: &rollbackFromVersion,
			}

			id, err := store.Add(ctx, version)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].AppConfigFileID).To(Equal(fileID))
			Expect(got[0].BaseAppConfigFileID).NotTo(BeNil())
			Expect(*got[0].BaseAppConfigFileID).To(Equal(baseID))
			Expect(got[0].BaseVersion).NotTo(BeNil())
			Expect(*got[0].BaseVersion).To(Equal(baseVersion))
			Expect(got[0].RollbackFromVersion).NotTo(BeNil())
			Expect(*got[0].RollbackFromVersion).To(Equal(rollbackFromVersion))
			Expect(*got[0].OverlayContent).To(Equal(content))
		})

		It("should reject duplicate version number in the same file", func() {
			_, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "v1"))
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Add(ctx, newVersion(1, time.Now().Add(time.Second), "bob", "duplicate"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("app config file version already exists"))
		})

		It("should allow the same version number for different files", func() {
			_, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "file-1"))
			Expect(err).NotTo(HaveOccurred())

			otherFileID := bson.NewObjectID()
			v := newVersion(1, time.Now().Add(time.Second), "alice", "file-2")
			v.AppConfigFileID = otherFileID

			_, err = store.Add(ctx, v)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get a version by app and id", func() {
			version := newVersion(2, time.Now(), "tester", "by-app-id")
			id, err := store.Add(ctx, version)
			Expect(err).NotTo(HaveOccurred())

			items, err := store.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].ID).To(Equal(id))
			Expect(items[0].AppID).To(Equal(appID))

			items, err = store.BatchGetByAppAndIDs(ctx, "another-app", []bson.ObjectID{id})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(BeEmpty())
		})

		It("should batch get versions by app and ids", func() {
			id1, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "first"))
			Expect(err).NotTo(HaveOccurred())
			id2, err := store.Add(ctx, newVersion(2, time.Now().Add(time.Second), "bob", "second"))
			Expect(err).NotTo(HaveOccurred())

			otherAppVersion := newVersion(3, time.Now().Add(2*time.Second), "carol", "other-app")
			otherAppVersion.AppID = "other-app"
			otherID, err := store.Add(ctx, otherAppVersion)
			Expect(err).NotTo(HaveOccurred())

			items, err := store.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id1, id2, otherID})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(2))

			gotIDs := map[bson.ObjectID]struct{}{}
			for _, item := range items {
				gotIDs[item.ID] = struct{}{}
				Expect(item.AppID).To(Equal(appID))
			}
			Expect(gotIDs).To(HaveKey(id1))
			Expect(gotIDs).To(HaveKey(id2))
			Expect(gotIDs).NotTo(HaveKey(otherID))
		})
	})

	Context("List", func() {
		It("should support filters and pagination", func() {
			v1 := newVersion(1, time.Now().Add(-3*time.Hour), "alice", "initial import")
			v1.Name = "values"
			v1.EnvName = "prod"
			v2 := newVersion(2, time.Now().Add(-2*time.Hour), "bob", "hotfix rollback")
			v2.Name = "values"
			v2.EnvName = "prod"
			v3 := newVersion(3, time.Now().Add(-1*time.Hour), "alice", "release")
			v3.Name = "values"
			v3.EnvName = "prod"

			_, err := store.Add(ctx, v1)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, v2)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, v3)
			Expect(err).NotTo(HaveOccurred())

			envName := "prod"
			creator := "alice"
			description := "lea"
			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &fileID,
				EnvName:         &envName,
				Creator:         &creator,
				Description:     &description,
				Page:            1,
				PageSize:        1,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(items).To(HaveLen(1))
			Expect(items[0].Version).To(Equal(int64(3)))
			Expect(items[0].Description).To(Equal("release"))
		})

		It("should sort by version desc when filtering by appConfigFileID", func() {
			sharedTime := time.Now()
			v1 := newVersion(1, sharedTime, "alice", "v1")
			v2 := newVersion(2, sharedTime, "bob", "v2")
			v3 := newVersion(3, sharedTime, "carol", "v3")

			_, err := store.Add(ctx, v1)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, v2)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, v3)
			Expect(err).NotTo(HaveOccurred())

			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &fileID,
				Page:            1,
				PageSize:        10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(items).To(HaveLen(3))
			Expect(items[0].Version).To(Equal(int64(3)))
			Expect(items[1].Version).To(Equal(int64(2)))
			Expect(items[2].Version).To(Equal(int64(1)))
		})

		It("should hide deleted versions by default and include them when requested", func() {
			id, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "to be deleted"))
			Expect(err).NotTo(HaveOccurred())

			count, err := store.SoftDeleteByID(ctx, id, "deleter")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())

			items, total, err = store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:          appID,
				Page:           1,
				PageSize:       10,
				IncludeDeleted: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(items).To(HaveLen(1))
			Expect(items[0].IsDeleted).To(BeTrue())
		})
	})

	Context("GetByFileAndVersion and SoftDeleteByID", func() {
		It("should not return a soft-deleted version from GetByFileAndVersion", func() {
			id, err := store.Add(ctx, newVersion(5, time.Now(), "alice", "candidate"))
			Expect(err).NotTo(HaveOccurred())

			count, err := store.SoftDeleteByID(ctx, id, "deleter")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			got, err := store.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].IsDeleted).To(BeTrue())
			Expect(got[0].Deleter).To(Equal("deleter"))
			Expect(got[0].DeletedAt).NotTo(BeNil())

			_, err = store.GetByFileAndVersion(ctx, fileID, 5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Context("DeleteByFileID", func() {
		It("should hard-delete all versions belonging to a file", func() {
			_, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "v1"))
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, newVersion(2, time.Now().Add(time.Second), "bob", "v2"))
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Add(ctx, newVersion(3, time.Now().Add(2*time.Second), "carol", "v3"))
			Expect(err).NotTo(HaveOccurred())

			count, err := store.DeleteByFileID(ctx, fileID)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(3)))

			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &fileID,
				Page:            1,
				PageSize:        10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())

			items, total, err = store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &fileID,
				Page:            1,
				PageSize:        10,
				IncludeDeleted:  true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())
		})

		It("should not affect versions belonging to other files", func() {
			_, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "file-A v1"))
			Expect(err).NotTo(HaveOccurred())

			otherFileID := bson.NewObjectID()
			otherV := newVersion(1, time.Now().Add(time.Second), "alice", "file-B v1")
			otherV.AppConfigFileID = otherFileID
			_, err = store.Add(ctx, otherV)
			Expect(err).NotTo(HaveOccurred())

			count, err := store.DeleteByFileID(ctx, fileID)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &otherFileID,
				Page:            1,
				PageSize:        10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(items[0].IsDeleted).To(BeFalse())
		})

		It("should also remove already soft-deleted versions", func() {
			id, err := store.Add(ctx, newVersion(1, time.Now(), "alice", "already deleted"))
			Expect(err).NotTo(HaveOccurred())

			_, err = store.SoftDeleteByID(ctx, id, "first-deleter")
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Add(ctx, newVersion(2, time.Now().Add(time.Second), "bob", "active"))
			Expect(err).NotTo(HaveOccurred())

			count, err := store.DeleteByFileID(ctx, fileID)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(2)))

			items, total, err := store.List(ctx, appcfg.AppConfigFileVersionListOptions{
				AppID:           appID,
				AppConfigFileID: &fileID,
				Page:            1,
				PageSize:        10,
				IncludeDeleted:  true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(items).To(BeEmpty())
		})
	})
})
