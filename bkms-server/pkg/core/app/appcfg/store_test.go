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

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/dbutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("AppConfigFileStoreMongo", func() {
	var store *appcfg.AppConfigFileStoreMongo
	var defStore *appcfg.AppConfigFileDefStoreMongo
	var versionStore *appcfg.AppConfigFileVersionStoreMongo
	var ctx context.Context

	var appID string
	// another app ID for testing
	var appID2 string
	var testAppConfigFile appcfg.AppConfigFile

	BeforeEach(func() {
		var err error

		store, err = appcfg.NewAppConfigFileStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		defStore, err = appcfg.NewAppConfigFileDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		versionStore, err = appcfg.NewAppConfigFileVersionStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

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

	Context("Add", func() {
		It("should add an app config file successfully", func() {
			oid, err := store.Add(ctx, testAppConfigFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(oid).NotTo(Equal(bson.NilObjectID))

			// Call List to verify the app config file was added
			appConfigFiles, err := store.List(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(dbutil.EqualIgnoringID(appConfigFiles[0], testAppConfigFile)).To(BeTrue())
		})

		It("should return error when adding a duplicate app config file with same defID+envName", func() {
			defID := bson.NewObjectID()
			dup := testAppConfigFile
			dup.DefID = defID
			dup.EnvName = "prod"

			_, err := store.Add(ctx, dup)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Add(ctx, dup)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("app config file already exists"))
		})
	})

	Context("IsOwnedByApp", func() {
		It("should return true for owned", func() {
			oid, _ := store.Add(ctx, testAppConfigFile)

			owned, _ := store.IsOwnedByApp(ctx, oid, appID)
			Expect(owned).To(BeTrue())
			owned, _ = store.IsOwnedByApp(ctx, oid, appID2)
			Expect(owned).To(BeFalse())
		})
	})

	Context("List", func() {
		It("should return empty list for non-existent app", func() {
			appConfigFiles, err := store.List(ctx, "non-existent-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(BeEmpty())
		})
		It("should filter by type", func() {
			_, err := store.Add(ctx, testAppConfigFile)
			Expect(err).NotTo(HaveOccurred())

			appConfigFiles, err := store.List(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(appConfigFiles)).To(Equal(1))

			By("filter with another type should return nothing")
			appConfigFiles, err = store.List(ctx, appID, appcfg.AcfFilterType("overlay"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(BeEmpty())
		})
		It("should filter by envName", func() {
			// Create app-level default config (envName = "")
			appLevelContent := "config: app-level"
			appLevelFile := appcfg.AppConfigFile{
				AppID:   appID,
				EnvName: appcfg.EnvNameDefault,
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Content:           &appLevelContent,
				},
			}
			_, err := store.Add(ctx, appLevelFile)
			Expect(err).NotTo(HaveOccurred())

			// Create prod environment config
			prodContent := "config: prod"
			prodFile := appcfg.AppConfigFile{
				AppID:   appID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Content:           &prodContent,
				},
			}
			_, err = store.Add(ctx, prodFile)
			Expect(err).NotTo(HaveOccurred())

			By("list all should return 2 files")
			appConfigFiles, err := store.List(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(2))

			By("filter by app-level (empty envName) should return 1 file")
			appConfigFiles, err = store.List(ctx, appID, appcfg.AcfFilterEnvName(appcfg.EnvNameDefault))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(appConfigFiles[0].EnvName).To(Equal(appcfg.EnvNameDefault))

			By("filter by prod envName should return 1 file")
			appConfigFiles, err = store.List(ctx, appID, appcfg.AcfFilterEnvName("prod"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(HaveLen(1))
			Expect(appConfigFiles[0].EnvName).To(Equal("prod"))

			By("filter by non-existent envName should return empty")
			appConfigFiles, err = store.List(ctx, appID, appcfg.AcfFilterEnvName("staging"))
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfigFiles).To(BeEmpty())
		})
	})

	Context("Update", func() {
		It("should updateFieldsWithVersionCheck an existing app config file", func() {
			// Add an initial app config file
			oid, _ := store.Add(ctx, testAppConfigFile)

			// Update the app config file, set the ID field
			testAppConfigFile.ID = oid
			newContent := `global:
 image: myapp:v2.0
 replicas: 5`
			testAppConfigFile.Content = &newContent
			count, err := store.Update(ctx, testAppConfigFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			// Verify the update
			dbObj, err := store.GetByID(ctx, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(*dbObj.Content).To(Equal(newContent))
		})

		It("should reject stale updates when expected version mismatches", func() {
			testAppConfigFile.CurrentVersion = 0
			oid, err := store.Add(ctx, testAppConfigFile)
			Expect(err).NotTo(HaveOccurred())

			testAppConfigFile.ID = oid

			latest := testAppConfigFile
			latest.CurrentVersion = 1
			latestContent := "latest-content"
			latest.Content = &latestContent
			count, err := store.UpdateIfVersionMatches(ctx, latest, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			stale := testAppConfigFile
			stale.CurrentVersion = 1
			staleContent := "stale-content"
			stale.Content = &staleContent
			count, err = store.UpdateIfVersionMatches(ctx, stale, 0)
			Expect(errors.Is(err, appcfg.ErrAppConfigFileVersionConflict)).To(BeTrue())
			Expect(count).To(Equal(int64(0)))
		})
	})

	Context("DeleteByID", func() {
		It("should remove an existing app config file", func() {
			oid, _ := store.Add(ctx, testAppConfigFile)

			// Verify the app config file was added
			dbObj, err := store.GetByID(ctx, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(dbutil.EqualIgnoringID(dbObj, &testAppConfigFile)).To(BeTrue())

			// Remove the app config file
			count, err := store.DeleteByID(ctx, appID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			// Verify the app config file was removed
			_, err = store.GetByID(ctx, oid)
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
		It("should error when deleting a referenced normal app config file", func() {
			// Add a normal app config file
			normalFile := testAppConfigFile
			normalID, err := store.Add(ctx, normalFile)
			Expect(err).NotTo(HaveOccurred())

			// Add an overlay app config file that references the normal file
			_, err = appcfg.NewAppConfigFileService(store, defStore, versionStore).Create(
				ctx,
				appcfg.CreateCfgFileParams{
					AppID:               appID,
					EnvName:             appcfg.EnvNameDefault,
					Name:                "test-overlay",
					Type:                appcfg.AppConfigFileTypeOverlay,
					ContentSourceType:   appcfg.ContentSourceTypeLocal,
					Format:              appcfg.FileFormatYAML,
					BaseAppConfigFileID: &normalID,
					Creator:             appcfg.CfgSystemUser,
					Description:         appcfg.CfgSystemVersionDescription,
				},
			)
			Expect(err).NotTo(HaveOccurred())

			// Try to delete the referenced normal app config file
			count, err := store.DeleteByID(ctx, appID, normalID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file is referenced by other files"))
			Expect(count).To(Equal(int64(0)))
		})
	})

	Context("GetByDefIDAndEnv", func() {
		It("should return the file matching defID and envName", func() {
			defID := bson.NewObjectID()
			acf := testAppConfigFile
			acf.DefID = defID
			acf.EnvName = "prod"
			_, err := store.Add(ctx, acf)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.GetByDefIDAndEnv(ctx, defID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.DefID).To(Equal(defID))
			Expect(got.EnvName).To(Equal("prod"))
		})

		It("should return error when no record matches", func() {
			_, err := store.GetByDefIDAndEnv(ctx, bson.NewObjectID(), "non-existent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should distinguish between different envNames under the same defID", func() {
			defID := bson.NewObjectID()

			defaultContent := "config: default"
			defaultFile := appcfg.AppConfigFile{
				AppID:   appID,
				DefID:   defID,
				EnvName: appcfg.EnvNameDefault,
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Content:           &defaultContent,
				},
			}
			_, err := store.Add(ctx, defaultFile)
			Expect(err).NotTo(HaveOccurred())

			prodContent := "config: prod"
			prodFile := appcfg.AppConfigFile{
				AppID:   appID,
				DefID:   defID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Content:           &prodContent,
				},
			}
			_, err = store.Add(ctx, prodFile)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.GetByDefIDAndEnv(ctx, defID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(*got.Content).To(Equal("config: prod"))

			got, err = store.GetByDefIDAndEnv(ctx, defID, appcfg.EnvNameDefault)
			Expect(err).NotTo(HaveOccurred())
			Expect(*got.Content).To(Equal("config: default"))
		})
	})

	Context("ListByDefID", func() {
		It("should return all files under the given defID", func() {
			defID := bson.NewObjectID()

			defaultFile := testAppConfigFile
			defaultFile.DefID = defID
			defaultFile.EnvName = appcfg.EnvNameDefault
			_, err := store.Add(ctx, defaultFile)
			Expect(err).NotTo(HaveOccurred())

			prodFile := testAppConfigFile
			prodFile.DefID = defID
			prodFile.EnvName = "prod"
			_, err = store.Add(ctx, prodFile)
			Expect(err).NotTo(HaveOccurred())

			result, err := store.ListByDefID(ctx, defID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
		})

		It("should return empty list when no files exist for defID", func() {
			result, err := store.ListByDefID(ctx, bson.NewObjectID())
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should not include files from other defIDs", func() {
			defID1 := bson.NewObjectID()
			defID2 := bson.NewObjectID()

			file1 := testAppConfigFile
			file1.DefID = defID1
			_, err := store.Add(ctx, file1)
			Expect(err).NotTo(HaveOccurred())

			file2 := testAppConfigFile
			file2.DefID = defID2
			_, err = store.Add(ctx, file2)
			Expect(err).NotTo(HaveOccurred())

			result, err := store.ListByDefID(ctx, defID1)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].DefID).To(Equal(defID1))
		})
	})

	Context("DeleteByDefID", func() {
		It("should delete all files under the given defID", func() {
			defID := bson.NewObjectID()

			file1 := testAppConfigFile
			file1.DefID = defID
			file1.EnvName = appcfg.EnvNameDefault
			_, err := store.Add(ctx, file1)
			Expect(err).NotTo(HaveOccurred())

			file2 := testAppConfigFile
			file2.DefID = defID
			file2.EnvName = "prod"
			_, err = store.Add(ctx, file2)
			Expect(err).NotTo(HaveOccurred())

			count, err := store.DeleteByDefID(ctx, defID)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(2)))

			remaining, err := store.ListByDefID(ctx, defID)
			Expect(err).NotTo(HaveOccurred())
			Expect(remaining).To(BeEmpty())
		})

		It("should return 0 when no files match", func() {
			count, err := store.DeleteByDefID(ctx, bson.NewObjectID())
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(0)))
		})

		It("should not affect files from other defIDs", func() {
			defID1 := bson.NewObjectID()
			defID2 := bson.NewObjectID()

			file1 := testAppConfigFile
			file1.DefID = defID1
			_, err := store.Add(ctx, file1)
			Expect(err).NotTo(HaveOccurred())

			file2 := testAppConfigFile
			file2.DefID = defID2
			_, err = store.Add(ctx, file2)
			Expect(err).NotTo(HaveOccurred())

			count, err := store.DeleteByDefID(ctx, defID1)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))

			remaining, err := store.ListByDefID(ctx, defID2)
			Expect(err).NotTo(HaveOccurred())
			Expect(remaining).To(HaveLen(1))
		})
	})
})
