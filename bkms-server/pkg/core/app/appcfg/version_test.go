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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AppConfigFileVersion", func() {
	var diApp *fxtest.App
	var appStore *bkmsapp.ApplicationStore
	var fileStore *appcfg.AppConfigFileStore
	var versions *appcfg.AppConfigFileVersionStore
	var base *appcfg.BaseAppCfgFileService
	var ctx context.Context
	var appID string
	var fileID bson.ObjectID

	newVersion := func(version int64, createdAt time.Time, creator, description string) *appcfg.AppConfigFileVersion {
		return lo.ToPtr(dbfactory.AppConfigFileVersion(&dbfactory.AppConfigFileVersionOpts{
			AppConfigFileID: fileID,
			AppID:           appID,
			Version:         version,
			Description:     description,
			Creator:         creator,
			CreatedAt:       createdAt,
		}))
	}
	addVersion := func(v *appcfg.AppConfigFileVersion) bson.ObjectID {
		Expect(versions.Create(ctx, v)).To(Succeed())
		return v.ID
	}
	versionNumbers := func(items []appcfg.AppConfigFileVersion) []int64 {
		return lo.Map(items, func(v appcfg.AppConfigFileVersion, _ int) int64 { return v.Version })
	}

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupMetaColl(appcfg.AppConfigFileVersionMeta)).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_files")).To(Succeed())

		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			appcfg.FxModule,
			fx.Populate(),
		)
		diApp.RequireStart()
		db := testutil.NewDB(GinkgoT())
		versions = store.Of(db, appcfg.AppConfigFileVersionMeta)
		base = appcfg.NewBaseAppCfgFileService(db)
		fileStore = base.FileStore

		app := dbfactory.Application(ctx, appStore)
		appID = app.ID
		fileID = bson.NewObjectID()
		err := fileStore.Create(ctx, &appcfg.AppConfigFile{
			ID:             fileID,
			AppID:          appID,
			EnvName:        appcfg.EnvNameDefault,
			Type:           appcfg.AppConfigFileTypeNormal,
			CurrentVersion: 3,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("Meta", func() {
		It("should persist all fields of a version record", func() {
			baseID := bson.NewObjectID()
			content := "foo: v2"
			version := &appcfg.AppConfigFileVersion{
				AppConfigFileID: fileID,
				AppID:           appID,
				EnvName:         "prod",
				Name:            "values",
				Type:            appcfg.AppConfigFileTypeOverlay,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType:   appcfg.ContentSourceTypeLocal,
					Format:              appcfg.FileFormatYAML,
					OverlayContent:      &content,
					BaseAppConfigFileID: &baseID,
				},
				Creator:             "tester",
				Version:             2,
				Description:         "second version",
				BaseVersion:         lo.ToPtr(int64(3)),
				OperationType:       appcfg.AppConfigFileVersionOperationTypeRollback,
				RollbackFromVersion: lo.ToPtr(int64(1)),
			}
			id := addVersion(version)

			got, err := versions.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AppConfigFileID).To(Equal(fileID))
			Expect(got.BaseAppConfigFileID).To(Equal(&baseID))
			Expect(got.BaseVersion).To(Equal(lo.ToPtr(int64(3))))
			Expect(got.RollbackFromVersion).To(Equal(lo.ToPtr(int64(1))))
			Expect(got.OverlayContent).To(Equal(&content))
		})

		It("should reject a duplicate version number in the same file", func() {
			addVersion(newVersion(1, time.Now(), "alice", "v1"))

			err := versions.Create(ctx, newVersion(1, time.Now(), "bob", "duplicate"))
			Expect(err).To(MatchError(appcfg.ErrAppCfgFileVersionAlreadyExists))
		})

		It("should allow the same version number for different files", func() {
			addVersion(newVersion(1, time.Now(), "alice", "file-1"))

			v := newVersion(1, time.Now(), "alice", "file-2")
			v.AppConfigFileID = bson.NewObjectID()
			Expect(versions.Create(ctx, v)).To(Succeed())
		})
	})

	Context("ListVersions", func() {
		It("should apply the query filters from the serializer and paginate", func() {
			for i, desc := range []string{"initial import", "hotfix rollback", "release"} {
				v := newVersion(int64(i+1), time.Now().Add(time.Duration(i-3)*time.Hour), "alice", desc)
				v.EnvName = "prod"
				if i == 1 {
					v.Creator = "bob"
				}
				addVersion(v)
			}
			filter, err := serializer.ListAppConfigFileVersionsQueryInput{
				AppConfigFileID: lo.ToPtr(fileID.Hex()),
				EnvName:         lo.ToPtr("prod"),
				Creator:         lo.ToPtr("alice"),
				Description:     lo.ToPtr("LEA"),
			}.ToVersionListOpt()
			Expect(err).NotTo(HaveOccurred())

			page, err := base.ListVersions(ctx, appID, 1, 1, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(1)))
			Expect(versionNumbers(page.Items)).To(Equal([]int64{3}))
		})

		It("should match the description keyword literally", func() {
			addVersion(newVersion(1, time.Now(), "alice", "release v1.0"))
			addVersion(newVersion(2, time.Now(), "alice", "release v100"))

			filter, err := serializer.ListAppConfigFileVersionsQueryInput{
				Description: lo.ToPtr("v1."),
			}.ToVersionListOpt()
			Expect(err).NotTo(HaveOccurred())
			page, err := base.ListVersions(ctx, appID, 1, 10, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(versionNumbers(page.Items)).To(Equal([]int64{1}))
		})

		It("should reject an invalid appConfigFileID", func() {
			_, err := serializer.ListAppConfigFileVersionsQueryInput{
				AppConfigFileID: lo.ToPtr("bad"),
			}.ToVersionListOpt()
			Expect(err).To(HaveOccurred())
		})

		It("should order versions newest first and break ties by _id", func() {
			sharedTime := time.Now()
			for i := int64(1); i <= 3; i++ {
				addVersion(newVersion(i, sharedTime, "alice", ""))
			}

			page, err := base.ListVersions(ctx, appID, 1, 2, appcfg.VersionsByFile(fileID))
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(3)))
			Expect(page.HasNext).To(BeTrue())
			Expect(versionNumbers(page.Items)).To(Equal([]int64{3, 2}))

			page, err = base.ListVersions(ctx, appID, 2, 2, appcfg.VersionsByFile(fileID))
			Expect(err).NotTo(HaveOccurred())
			Expect(versionNumbers(page.Items)).To(Equal([]int64{1}))
		})

		It("should only list versions of the given app", func() {
			addVersion(newVersion(1, time.Now(), "alice", ""))
			other := newVersion(1, time.Now(), "alice", "")
			other.AppID = "other-app"
			other.AppConfigFileID = bson.NewObjectID()
			addVersion(other)

			page, err := base.ListVersions(ctx, appID, 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.Items[0].AppID).To(Equal(appID))
		})
	})

	Context("GetVersionByAppAndID", func() {
		It("should not return a version of another app", func() {
			id := addVersion(newVersion(2, time.Now(), "tester", ""))

			got, err := base.GetVersionByAppAndID(ctx, appID, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(id))

			_, err = base.GetVersionByAppAndID(ctx, "another-app", id)
			Expect(err).To(MatchError(appcfg.ErrAppCfgFileVersionNotFound))
		})
	})

	Context("CompareVersions", func() {
		It("should return both versions of the same app", func() {
			id1 := addVersion(newVersion(1, time.Now(), "alice", ""))
			id2 := addVersion(newVersion(2, time.Now(), "bob", ""))

			previous, current, err := base.CompareVersions(ctx, appID, id1, id2)
			Expect(err).NotTo(HaveOccurred())
			Expect(previous.Version).To(Equal(int64(1)))
			Expect(current.Version).To(Equal(int64(2)))
		})

		It("should treat a version of another app as not found", func() {
			id1 := addVersion(newVersion(1, time.Now(), "alice", ""))
			other := newVersion(1, time.Now(), "carol", "")
			other.AppID = "other-app"
			other.AppConfigFileID = bson.NewObjectID()
			otherID := addVersion(other)

			_, _, err := base.CompareVersions(ctx, appID, id1, otherID)
			Expect(err).To(MatchError(appcfg.ErrAppCfgFileVersionNotFound))
		})
	})

	Context("DeleteVersion", func() {
		It("should soft delete a historical version and hide it from the list", func() {
			id := addVersion(newVersion(1, time.Now(), "alice", "to be deleted"))
			addVersion(newVersion(3, time.Now(), "alice", "current"))

			deleted, _, err := base.DeleteVersion(ctx, appID, id, "deleter")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.ID).To(Equal(id))

			got, err := base.GetVersionByAppAndID(ctx, appID, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsDeleted).To(BeTrue())
			Expect(got.Deleter).To(Equal("deleter"))
			Expect(got.DeletedAt).NotTo(BeNil())

			page, err := base.ListVersions(ctx, appID, 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(versionNumbers(page.Items)).To(Equal([]int64{3}))
		})

		It("should reject deleting the current version", func() {
			id := addVersion(newVersion(3, time.Now(), "alice", "current"))

			_, _, err := base.DeleteVersion(ctx, appID, id, "deleter")
			Expect(err).To(MatchError(appcfg.ErrUsingVersionCannotBeDeleted))
		})
	})

	Context("VersionsByFile", func() {
		It("should hard delete all versions of a file including soft-deleted ones", func() {
			soft := newVersion(1, time.Now(), "alice", "already deleted")
			soft.IsDeleted = true
			addVersion(soft)
			addVersion(newVersion(2, time.Now(), "bob", "active"))
			other := newVersion(1, time.Now(), "alice", "file-B v1")
			other.AppConfigFileID = bson.NewObjectID()
			addVersion(other)

			count, err := versions.DeleteMany(ctx, appcfg.VersionsByFile(fileID))
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(2)))

			Expect(versions.Count(ctx, appcfg.VersionsByFile(fileID))).To(BeZero())
			Expect(versions.Count(ctx, appcfg.VersionsByFile(other.AppConfigFileID))).To(Equal(int64(1)))
		})
	})
})
