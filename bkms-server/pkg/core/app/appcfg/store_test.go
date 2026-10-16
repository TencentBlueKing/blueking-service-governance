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

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AppConfigFile store", func() {
	var files *appcfg.AppConfigFileStore
	var ctx context.Context

	var appID string
	// another app ID for testing
	var appID2 string
	var testAppConfigFile appcfg.AppConfigFile

	BeforeEach(func() {
		files = testutil.NewStore(GinkgoT(), appcfg.AppConfigFileMeta)
		ctx = context.Background()
		appID = "test-app-" + stringx.Random(6)
		appID2 = appID + stringx.Random(2)

		// The most commonly used test app config file object
		content := `global:
 image: myapp:latest
 replicas: 3`
		testAppConfigFile = appcfg.AppConfigFile{
			AppID: appID,
			Type:  appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Content:           &content,
			},
		}
	})

	AfterEach(func() {
		_, err := files.DeleteMany(ctx, store.Where("appID", bson.M{"$in": []string{appID, appID2}}))
		Expect(err).NotTo(HaveOccurred())
	})

	// newFile 基于 testAppConfigFile 创建一个文件记录，modify 用于调整字段
	newFile := func(modify func(acf *appcfg.AppConfigFile)) *appcfg.AppConfigFile {
		acf := testAppConfigFile
		if modify != nil {
			modify(&acf)
		}
		Expect(files.Create(ctx, &acf)).To(Succeed())
		return &acf
	}

	Context("Create", func() {
		It("creates a file that can be listed back", func() {
			acf := newFile(nil)

			appConfigFiles, err := appcfg.ListFiles(ctx, files, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(appConfigFiles[0].ID).To(Equal(acf.ID))
			Expect(appConfigFiles[0].Content).To(Equal(testAppConfigFile.Content))
		})

		It("rejects a duplicate file with the same defID and envName", func() {
			defID := bson.NewObjectID()
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = "prod"
			})

			dup := testAppConfigFile
			dup.DefID = defID
			dup.EnvName = "prod"
			Expect(files.Create(ctx, &dup)).To(MatchError(appcfg.ErrAppConfigFileAlreadyExists))
		})
	})

	Context("ListFiles", func() {
		It("filters by type", func() {
			newFile(nil)

			appConfigFiles, err := appcfg.ListFiles(ctx, files, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))

			By("filter with another type should return nothing")
			appConfigFiles, err = appcfg.ListFiles(ctx, files, appID, appcfg.AcfFilterType("overlay"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(BeEmpty())
		})

		It("filters by envName", func() {
			newFile(func(acf *appcfg.AppConfigFile) { acf.EnvName = appcfg.EnvNameDefault })
			newFile(func(acf *appcfg.AppConfigFile) { acf.EnvName = "prod" })

			By("list all should return 2 files")
			appConfigFiles, err := appcfg.ListFiles(ctx, files, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(2))

			By("filter by app-level (empty envName) should return 1 file")
			appConfigFiles, err = appcfg.ListFiles(ctx, files, appID, appcfg.AcfFilterEnvName(appcfg.EnvNameDefault))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(appConfigFiles[0].EnvName).To(Equal(appcfg.EnvNameDefault))

			By("filter by prod envName should return 1 file")
			appConfigFiles, err = appcfg.ListFiles(ctx, files, appID, appcfg.AcfFilterEnvName("prod"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(appConfigFiles[0].EnvName).To(Equal("prod"))

			By("filter by non-existent envName should return empty")
			appConfigFiles, err = appcfg.ListFiles(ctx, files, appID, appcfg.AcfFilterEnvName("staging"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(BeEmpty())
		})

		It("filters by def IDs", func() {
			defID1, defID2 := bson.NewObjectID(), bson.NewObjectID()
			newFile(func(acf *appcfg.AppConfigFile) { acf.DefID = defID1 })
			newFile(func(acf *appcfg.AppConfigFile) { acf.DefID = defID2 })
			newFile(func(acf *appcfg.AppConfigFile) { acf.DefID = bson.NewObjectID() })

			appConfigFiles, err := appcfg.ListFiles(ctx, files, appID, appcfg.AcfFilterDefIDs{defID1, defID2})
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(2))
			Expect([]bson.ObjectID{appConfigFiles[0].DefID, appConfigFiles[1].DefID}).To(ConsistOf(defID1, defID2))
		})

		It("rejects ordering by name", func() {
			_, err := appcfg.ListFiles(ctx, files, appID, appcfg.AcfOrderBy(appcfg.ListOrderByName))
			Expect(err).To(MatchError(ContainSubstring("unsupported orderBy field")))
		})
	})

	Context("UpdateFileIfVersionMatches", func() {
		It("updates the file when the version matches", func() {
			acf := newFile(nil)

			newContent := "latest-content"
			acf.Content = &newContent
			acf.CurrentVersion = 1
			Expect(appcfg.UpdateFileIfVersionMatches(ctx, files, acf, 0)).To(Succeed())

			dbObj, err := files.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(*dbObj.Content).To(Equal(newContent))
			Expect(dbObj.CurrentVersion).To(Equal(int64(1)))
		})

		It("rejects stale updates when the expected version mismatches", func() {
			acf := newFile(nil)

			latest := *acf
			latestContent := "latest-content"
			latest.Content = &latestContent
			latest.CurrentVersion = 1
			Expect(appcfg.UpdateFileIfVersionMatches(ctx, files, &latest, 0)).To(Succeed())

			stale := *acf
			staleContent := "stale-content"
			stale.Content = &staleContent
			stale.CurrentVersion = 1
			err := appcfg.UpdateFileIfVersionMatches(ctx, files, &stale, 0)
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileVersionConflict))

			dbObj, err := files.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(*dbObj.Content).To(Equal(latestContent))
		})

		It("reports a version conflict when the file has been deleted", func() {
			acf := testAppConfigFile
			acf.ID = bson.NewObjectID()
			err := appcfg.UpdateFileIfVersionMatches(ctx, files, &acf, 0)
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileVersionConflict))
		})
	})

	Context("DeleteFile", func() {
		It("removes an existing app config file", func() {
			acf := newFile(nil)

			Expect(appcfg.DeleteFile(ctx, files, appID, acf.ID)).To(Succeed())

			_, err := files.GetByID(ctx, acf.ID)
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileNotFound))
		})

		It("leaves files owned by other apps untouched", func() {
			acf := newFile(nil)

			Expect(appcfg.DeleteFile(ctx, files, appID2, acf.ID)).To(Succeed())

			_, err := files.GetByID(ctx, acf.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses to delete a file referenced by an overlay file", func() {
			normalFile := newFile(nil)
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.Type = appcfg.AppConfigFileTypeOverlay
				acf.BaseAppConfigFileID = &normalFile.ID
			})

			err := appcfg.DeleteFile(ctx, files, appID, normalFile.ID)
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileReferenced))

			_, err = files.GetByID(ctx, normalFile.ID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("FileByDefEnv", func() {
		It("returns the file matching defID and envName", func() {
			defID := bson.NewObjectID()
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = "prod"
			})

			got, err := files.Get(ctx, appcfg.FileByDefEnv(defID, "prod"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.DefID).To(Equal(defID))
			Expect(got.EnvName).To(Equal("prod"))
		})

		It("returns ErrAppConfigFileNotFound when no record matches", func() {
			_, err := files.Get(ctx, appcfg.FileByDefEnv(bson.NewObjectID(), "non-existent"))
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileNotFound))
		})

		It("distinguishes between different envNames under the same defID", func() {
			defID := bson.NewObjectID()
			defaultContent, prodContent := "config: default", "config: prod"
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = appcfg.EnvNameDefault
				acf.Content = &defaultContent
			})
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = "prod"
				acf.Content = &prodContent
			})

			got, err := files.Get(ctx, appcfg.FileByDefEnv(defID, "prod"))
			Expect(err).NotTo(HaveOccurred())
			Expect(*got.Content).To(Equal(prodContent))

			got, err = files.Get(ctx, appcfg.FileByDefEnv(defID, appcfg.EnvNameDefault))
			Expect(err).NotTo(HaveOccurred())
			Expect(*got.Content).To(Equal(defaultContent))
		})
	})

	Context("FilesByDef", func() {
		It("returns all files under the given defID", func() {
			defID := bson.NewObjectID()
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = appcfg.EnvNameDefault
			})
			newFile(func(acf *appcfg.AppConfigFile) {
				acf.DefID = defID
				acf.EnvName = "prod"
			})

			result, err := files.List(ctx, appcfg.FilesByDef(defID))
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
		})
	})
})
