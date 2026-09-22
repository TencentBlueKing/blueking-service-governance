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

package app_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

var _ = Describe("Application store", func() {
	var db *mongo.Database
	var apps *bkmsapp.ApplicationStore
	var ctx context.Context
	var workspaceID string
	var appID string
	var appName string

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		apps = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)

		ctx = context.Background()
		appID = "test-appid-" + stringx.Random(6)
		appName = "test-app-" + stringx.Random(6)
		workspaceID = "test-workspace-" + stringx.Random(6)
	})

	Context("CreateApp", func() {
		It("should create and get application successfully", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			err := bkmsapp.CreateApp(ctx, db, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Get the application back from the database
			app, err := apps.Get(ctx, bkmsapp.ByID(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(app.WorkspaceID).To(Equal(workspaceID))
			Expect(app.Name).To(Equal(appName))
		})

		It("should return error when application with same name already exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			// First call should succeed
			err := bkmsapp.CreateApp(ctx, db, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same name should fail
			duplicateApp := &bkmsapp.Application{ID: appID + "-nodup", WorkspaceID: workspaceID, Name: appName}
			err = bkmsapp.CreateApp(ctx, db, duplicateApp)
			Expect(err).To(MatchError(bkmsapp.ErrAppAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("application with the same name already exists"))
		})

		It("should return error when application with same id already exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			// First call should succeed
			err := bkmsapp.CreateApp(ctx, db, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same id should fail
			duplicateApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName + "-nodup"}
			err = bkmsapp.CreateApp(ctx, db, duplicateApp)
			Expect(err).To(MatchError(bkmsapp.ErrAppAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("application with the same id already exists"))
		})
	})

	Context("DeleteIfExists", func() {
		It("delete if the application exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}
			Expect(bkmsapp.CreateApp(ctx, db, testApp)).To(Succeed())

			err := apps.DeleteIfExists(ctx, bkmsapp.ByName(workspaceID, appName))
			Expect(err).NotTo(HaveOccurred())

			_, err = apps.Get(ctx, bkmsapp.ByID(appID))
			Expect(err).To(MatchError(bkmsapp.ErrAppNotFound))
		})
	})

	Context("Get", func() {
		It("should return ErrAppNotFound when application does not exist", func() {
			nonExistentID := "non-existent-id-" + stringx.Random(6)
			app, err := apps.Get(ctx, bkmsapp.ByID(nonExistentID))
			Expect(err).To(MatchError(bkmsapp.ErrAppNotFound))
			Expect(app).To(BeNil())
		})

		It("should decrypt sensitive fields when getting application", func() {
			testApp := &bkmsapp.Application{
				ID:          appID,
				WorkspaceID: workspaceID,
				Name:        appName,
				HelmSpec: &bkmsapp.HelmSpec{
					HelmSource: &bkmsapp.HelmSource{
						RepoType: "HelmRepo",
						HelmRepoConfig: &bkmsapp.HelmRepoConfig{
							RepoURL:   "https://charts.example.com",
							ChartName: "test-chart",
							Username:  "testUser",
							Password:  "testPassword",
						},
					},
				},
			}
			Expect(bkmsapp.CreateApp(ctx, db, testApp)).To(Succeed())
			// 加密只作用于待写入的副本，调用方的对象保持明文
			Expect(testApp.HelmSpec.HelmSource.HelmRepoConfig.Password).To(Equal("testPassword"))

			// Get the application by ID
			app, err := apps.Get(ctx, bkmsapp.ByID(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
			Expect(app.HelmSpec).NotTo(BeNil())
			Expect(app.HelmSpec.HelmSource).NotTo(BeNil())
			Expect(app.HelmSpec.HelmSource.HelmRepoConfig).NotTo(BeNil())
			// Password should be decrypted back to original value
			Expect(app.HelmSpec.HelmSource.HelmRepoConfig.Username).To(Equal("testUser"))
			Expect(app.HelmSpec.HelmSource.HelmRepoConfig.Password).To(Equal("testPassword"))
		})
	})

	Context("GetAppsByIDs", func() {
		It("should return nil slice when ids is nil or empty", func() {
			result, err := bkmsapp.GetAppsByIDs(ctx, db, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())

			result, err = bkmsapp.GetAppsByIDs(ctx, db, []string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should load multiple apps in one query preserving id order and duplicates", func() {
			id2 := "test-appid-" + stringx.Random(6)
			name2 := "test-app-" + stringx.Random(6)
			err := bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName})
			Expect(err).NotTo(HaveOccurred())
			err = bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{ID: id2, WorkspaceID: workspaceID, Name: name2})
			Expect(err).NotTo(HaveOccurred())

			result, err := bkmsapp.GetAppsByIDs(ctx, db, []string{appID, id2, appID})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].Name).To(Equal(appName))
			Expect(result[1].Name).To(Equal(name2))
			Expect(result[2].Name).To(Equal(appName))
			Expect(result[0]).To(BeIdenticalTo(result[2]))
		})

		It("should return wrapped ErrAppNotFound when any id is missing", func() {
			err := bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName})
			Expect(err).NotTo(HaveOccurred())
			missing := "non-existent-id-" + stringx.Random(6)

			result, err := bkmsapp.GetAppsByIDs(ctx, db, []string{appID, missing})
			Expect(err).To(MatchError(bkmsapp.ErrAppNotFound))
			Expect(result).To(BeNil())
		})
	})

	Context("Update", func() {
		It("should update the helm source and keep the password readable", func() {
			app := &bkmsapp.Application{
				ID:          appID,
				WorkspaceID: workspaceID,
				Name:        appName,
				HelmSpec: &bkmsapp.HelmSpec{
					HelmSource: &bkmsapp.HelmSource{
						RepoType: "HelmRepo",
						HelmRepoConfig: &bkmsapp.HelmRepoConfig{
							RepoURL:   "https://charts.bitnami.com/bitnami",
							ChartName: "redis",
							Username:  "bitnami",
							Password:  "bitnami",
						},

						ValueFiles: []string{"my-values.yaml"},
					},
				},
			}
			Expect(bkmsapp.CreateApp(ctx, db, app)).To(Succeed())

			err := apps.Update(ctx, map[string]any{"helmSpec.helmSource": &bkmsapp.HelmSource{
				RepoType: "HelmRepo",
				HelmRepoConfig: &bkmsapp.HelmRepoConfig{
					RepoURL:   "https://charts.bitnami.cn/bitnami",
					ChartName: "nginx",
					Username:  "admin",
					Password:  "admin",
				},
			}}, bkmsapp.ByID(appID))
			Expect(err).NotTo(HaveOccurred())
			app, err = apps.Get(ctx, bkmsapp.ByID(appID))
			Expect(err).NotTo(HaveOccurred())

			expectedHelmSource := app.HelmSpec.HelmSource
			Expect(expectedHelmSource.RepoType).To(Equal(bkmsapp.HelmSourceRepoTypeHelm))
			Expect(expectedHelmSource.HelmRepoConfig.RepoURL).To(Equal("https://charts.bitnami.cn/bitnami"))
			Expect(expectedHelmSource.HelmRepoConfig.ChartName).To(Equal("nginx"))
			Expect(expectedHelmSource.HelmRepoConfig.Username).To(Equal("admin"))
			Expect(expectedHelmSource.HelmRepoConfig.Password).To(Equal("admin"))
		})
	})

	Context("RemoveVisibleEnvName", func() {
		It("should pull the env name only from apps in that workspace", func() {
			visible := map[string]any{"visibleEnvNames": []string{"test", "stag"}}
			app := dbfactory.ApplicationWithOpts(ctx, apps, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
			Expect(apps.Update(ctx, visible, bkmsapp.ByID(app.ID))).To(Succeed())

			otherWorkspaceID := "test-workspace-" + stringx.Random(6)
			otherApp := dbfactory.ApplicationWithOpts(
				ctx,
				apps,
				&dbfactory.ApplicationOpts{WorkspaceID: otherWorkspaceID},
			)
			Expect(apps.Update(ctx, visible, bkmsapp.ByID(otherApp.ID))).To(Succeed())

			err := bkmsapp.RemoveVisibleEnvName(ctx, db, workspaceID, "test")
			Expect(err).NotTo(HaveOccurred())

			app, err = apps.Get(ctx, bkmsapp.ByID(app.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(app.VisibleEnvNames).To(Equal([]string{"stag"}))

			otherApp, err = apps.Get(ctx, bkmsapp.ByID(otherApp.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherApp.VisibleEnvNames).To(Equal([]string{"test", "stag"}))
		})

		It("should be a no-op when no app lists the env name", func() {
			app := dbfactory.ApplicationWithOpts(ctx, apps, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
			Expect(
				apps.Update(ctx, map[string]any{"visibleEnvNames": []string{"stag"}}, bkmsapp.ByID(app.ID)),
			).To(Succeed())

			err := bkmsapp.RemoveVisibleEnvName(ctx, db, workspaceID, "test")
			Expect(err).NotTo(HaveOccurred())

			app, err = apps.Get(ctx, bkmsapp.ByID(app.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(app.VisibleEnvNames).To(Equal([]string{"stag"}))
		})
	})

	Context("List", func() {
		var workspace1ID, workspace2ID string
		var app1Name, app2Name, app3Name string

		BeforeEach(func() {
			workspace1ID = "workspace1-" + stringx.Random(6)
			workspace2ID = "workspace2-" + stringx.Random(6)
			app1Name = "app1-" + stringx.Random(6)
			app2Name = "app2-" + stringx.Random(6)
			app3Name = "app3-" + stringx.Random(6)

			// Create test applications
			Expect(bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{
				ID:          "id1-" + stringx.Random(6),
				WorkspaceID: workspace1ID,
				Name:        app1Name,
				Type:        bkmsapp.AppTypeHelm,
			})).To(Succeed())
			Expect(bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{
				ID:          "id2-" + stringx.Random(6),
				WorkspaceID: workspace1ID,
				Name:        app2Name,
				Type:        bkmsapp.AppTypeTRPC,
			})).To(Succeed())
			Expect(bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{
				ID:          "id3-" + stringx.Random(6),
				WorkspaceID: workspace2ID,
				Name:        app3Name,
				Type:        bkmsapp.AppTypeHelm,
			})).To(Succeed())
		})

		It("should keep old fields and new language/framework on round trip", func() {
			testApp := &bkmsapp.Application{
				ID:          appID,
				WorkspaceID: workspaceID,
				Name:        appName,
				Type:        bkmsapp.AppTypeTRPC,
				Language:    "go",
				Framework:   "trpc",
				TrpcSpec:    &bkmsapp.TrpcSpec{Language: "go"},
			}
			Expect(bkmsapp.CreateApp(ctx, db, testApp)).To(Succeed())

			got, err := apps.Get(ctx, bkmsapp.ByID(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Type).To(Equal(bkmsapp.AppTypeTRPC))
			Expect(got.Language).To(Equal("go"))
			Expect(got.Framework).To(Equal("trpc"))
			Expect(got.TrpcSpec).NotTo(BeNil())
			Expect(got.TrpcSpec.Language).To(Equal("go"))
		})

		It("should treat type=trpc as matching future bkmsApp+trpc rows", func() {
			legacyID := "legacy-" + stringx.Random(6)
			modernID := "modern-" + stringx.Random(6)
			Expect(bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{
				ID:          legacyID,
				WorkspaceID: workspaceID,
				Name:        "legacy-" + stringx.Random(6),
				Type:        bkmsapp.AppTypeTRPC,
			})).To(Succeed())
			Expect(bkmsapp.CreateApp(ctx, db, &bkmsapp.Application{
				ID:          modernID,
				WorkspaceID: workspaceID,
				Name:        "modern-" + stringx.Random(6),
				Type:        bkmsapp.AppTypeBKMSApp,
				Framework:   "trpc",
				Language:    "go",
			})).To(Succeed())

			result, err := apps.List(ctx, bkmsapp.InWorkspace(workspaceID), bkmsapp.OfType(bkmsapp.AppTypeTRPC))
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(result, func(a bkmsapp.Application, _ int) string { return a.ID })).
				To(ConsistOf(legacyID, modernID))
		})

		It("should recognize bkmsApp as an appmodel type", func() {
			Expect(bkmsapp.IsAppModelType(bkmsapp.AppTypeBKMSApp)).To(BeTrue())
			Expect(bkmsapp.IsAppModelType(bkmsapp.AppTypeTRPC)).To(BeTrue())
			Expect(bkmsapp.IsAppModelType(bkmsapp.AppTypeHelm)).To(BeFalse())
		})

		It("should filter applications by multiple criteria", func() {
			result, err := apps.List(ctx, bkmsapp.InWorkspace(workspace1ID), bkmsapp.OfType(bkmsapp.AppTypeHelm))
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].WorkspaceID).To(Equal(workspace1ID))
			Expect(result[0].Name).To(Equal(app1Name))
			Expect(result[0].Type).To(Equal(bkmsapp.AppTypeHelm))
		})

		It("should count applications by workspace IDs", func() {
			counts, err := bkmsapp.CountByWorkspaceIDs(
				ctx,
				db,
				[]string{workspace1ID, workspace2ID, "missing-workspace"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[workspace1ID]).To(Equal(2))
			Expect(counts[workspace2ID]).To(Equal(1))
			Expect(counts).NotTo(HaveKey("missing-workspace"))
		})
	})
})
