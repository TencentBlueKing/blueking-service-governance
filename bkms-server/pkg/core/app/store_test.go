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
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ApplicationStoreMongo", func() {
	var appStore *bkmsapp.ApplicationStoreMongo
	var ctx context.Context
	var workspaceID string
	var appID string
	var appName string

	BeforeEach(func() {
		var err error

		appStore, err = bkmsapp.NewApplicationStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		appID = "test-appid-" + stringx.Random(6)
		appName = "test-app-" + stringx.Random(6)
		workspaceID = "test-workspace-" + stringx.Random(6)
	})

	Context("CreateApp", func() {
		It("should create and get application successfully", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Get the application back from the database
			app, err := appStore.GetApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(app.WorkspaceID).To(Equal(workspaceID))
			Expect(app.Name).To(Equal(appName))
		})

		It("should return error when application with same name already exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			// First call should succeed
			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same name should fail
			duplicateApp := &bkmsapp.Application{ID: appID + "-nodup", WorkspaceID: workspaceID, Name: appName}
			err = appStore.CreateApp(ctx, duplicateApp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("application with the same name already exists"))
		})
		It("should return error when application with same id already exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}

			// First call should succeed
			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same id should fail
			duplicateApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName + "-nodup"}
			err = appStore.CreateApp(ctx, duplicateApp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("application with the same id already exists"))
		})
	})

	Context("DeleteApp", func() {
		It("delete if the application exists", func() {
			testApp := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}
			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			err = appStore.DeleteAppByName(ctx, workspaceID, appName)
			Expect(err).NotTo(HaveOccurred())

			app, _ := appStore.GetApp(ctx, appID)
			Expect(app).To(BeNil())
		})

		It("delete if the application does not exist", func() {
			err := appStore.DeleteAppByName(ctx, workspaceID, appName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("GetApp", func() {
		It("should get application by ID successfully", func() {
			testApp := &bkmsapp.Application{
				ID:          appID,
				WorkspaceID: workspaceID,
				Name:        appName,
				DisplayName: "Test Application",
			}
			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Get the application by ID
			app, err := appStore.GetApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
			Expect(app.ID).To(Equal(appID))
			Expect(app.WorkspaceID).To(Equal(workspaceID))
			Expect(app.Name).To(Equal(appName))
			Expect(app.DisplayName).To(Equal("Test Application"))
		})

		It("should return ErrAppNotFound when application does not exist", func() {
			nonExistentID := "non-existent-id-" + stringx.Random(6)
			app, err := appStore.GetApp(ctx, nonExistentID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(bkmsapp.ErrAppNotFound))
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
			err := appStore.CreateApp(ctx, testApp)
			Expect(err).NotTo(HaveOccurred())

			// Get the application by ID
			app, err := appStore.GetApp(ctx, appID)
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
			apps, err := appStore.GetAppsByIDs(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(BeNil())

			apps, err = appStore.GetAppsByIDs(ctx, []string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(BeNil())
		})

		It("should load multiple apps in one query preserving id order and duplicates", func() {
			id2 := "test-appid-" + stringx.Random(6)
			name2 := "test-app-" + stringx.Random(6)
			err := appStore.CreateApp(ctx, &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName})
			Expect(err).NotTo(HaveOccurred())
			err = appStore.CreateApp(ctx, &bkmsapp.Application{ID: id2, WorkspaceID: workspaceID, Name: name2})
			Expect(err).NotTo(HaveOccurred())

			apps, err := appStore.GetAppsByIDs(ctx, []string{appID, id2, appID})
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(HaveLen(3))
			Expect(apps[0].Name).To(Equal(appName))
			Expect(apps[1].Name).To(Equal(name2))
			Expect(apps[2].Name).To(Equal(appName))
			Expect(apps[0]).To(BeIdenticalTo(apps[2]))
		})

		It("should return wrapped ErrAppNotFound when any id is missing", func() {
			err := appStore.CreateApp(ctx, &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName})
			Expect(err).NotTo(HaveOccurred())
			missing := "non-existent-id-" + stringx.Random(6)

			apps, err := appStore.GetAppsByIDs(ctx, []string{appID, missing})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, bkmsapp.ErrAppNotFound)).To(BeTrue())
			Expect(apps).To(BeNil())
		})
	})

	Context("UpdateDisplayName", func() {
		It("should update the display name of an application", func() {
			app := &bkmsapp.Application{ID: appID, WorkspaceID: workspaceID, Name: appName}
			_ = appStore.CreateApp(ctx, app)

			newDisplayName := "new " + stringx.Random(6)
			err := appStore.UpdateDisplayName(ctx, app, newDisplayName)
			Expect(err).NotTo(HaveOccurred())
			app, _ = appStore.GetApp(ctx, appID)
			Expect(app.DisplayName).To(Equal(newDisplayName))
		})
	})

	Context("UpdateHelmSpec", func() {
		It("should update the helm spec of an application", func() {
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
			_ = appStore.CreateApp(ctx, app)

			err := appStore.UpdateHelmSource(ctx, app, &bkmsapp.HelmSource{
				RepoType: "HelmRepo",
				HelmRepoConfig: &bkmsapp.HelmRepoConfig{
					RepoURL:   "https://charts.bitnami.cn/bitnami",
					ChartName: "nginx",
					Username:  "admin",
					Password:  "admin",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			app, err = appStore.GetApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())

			expectedHelmSource := app.HelmSpec.HelmSource
			Expect(expectedHelmSource.RepoType).To(Equal(bkmsapp.HelmSourceRepoTypeHelm))
			Expect(expectedHelmSource.HelmRepoConfig.RepoURL).To(Equal("https://charts.bitnami.cn/bitnami"))
			Expect(expectedHelmSource.HelmRepoConfig.ChartName).To(Equal("nginx"))
			Expect(expectedHelmSource.HelmRepoConfig.Username).To(Equal("admin"))
			Expect(expectedHelmSource.HelmRepoConfig.Password).To(Equal("admin"))
		})
	})

	Context("ListApps", func() {
		var workspace1ID, workspace2ID string
		var app1Name, app2Name, app3Name string

		BeforeEach(func() {
			workspace1ID = "workspace1-" + stringx.Random(6)
			workspace2ID = "workspace2-" + stringx.Random(6)
			app1Name = "app1-" + stringx.Random(6)
			app2Name = "app2-" + stringx.Random(6)
			app3Name = "app3-" + stringx.Random(6)

			// Create test applications
			_ = appStore.CreateApp(ctx, &bkmsapp.Application{
				ID:          "id1-" + stringx.Random(6),
				WorkspaceID: workspace1ID,
				Name:        app1Name,
				Type:        bkmsapp.AppTypeHelm,
			})
			_ = appStore.CreateApp(ctx, &bkmsapp.Application{
				ID:          "id2-" + stringx.Random(6),
				WorkspaceID: workspace1ID,
				Name:        app2Name,
				Type:        bkmsapp.AppTypeTRPC,
			})
			_ = appStore.CreateApp(ctx, &bkmsapp.Application{
				ID:          "id3-" + stringx.Random(6),
				WorkspaceID: workspace2ID,
				Name:        app3Name,
				Type:        bkmsapp.AppTypeHelm,
			})
		})

		It("should list all applications when no filter is provided", func() {
			apps, err := appStore.ListApps(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(BeNumerically(">=", 3))
		})

		It("should filter applications by workspaceID", func() {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{WorkspaceID: workspace1ID})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(Equal(2))
			for _, app := range apps {
				Expect(app.WorkspaceID).To(Equal(workspace1ID))
			}
		})

		It("should filter applications by appName", func() {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{AppName: app1Name})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(Equal(1))
			Expect(apps[0].Name).To(Equal(app1Name))
		})

		It("should filter applications by appType", func() {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{AppType: bkmsapp.AppTypeHelm})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(BeNumerically(">=", 2))
			for _, app := range apps {
				Expect(app.Type).To(Equal(bkmsapp.AppTypeHelm))
			}
		})

		It("should filter applications by multiple criteria", func() {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{
				WorkspaceID: workspace1ID,
				AppType:     bkmsapp.AppTypeHelm,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(Equal(1))
			Expect(apps[0].WorkspaceID).To(Equal(workspace1ID))
			Expect(apps[0].Name).To(Equal(app1Name))
			Expect(apps[0].Type).To(Equal(bkmsapp.AppTypeHelm))
		})

		It("should return empty list when no applications match the filter", func() {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{
				WorkspaceID: "non-existent-workspace",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(Equal(0))
		})

		It("should count applications by workspace IDs", func() {
			counts, err := appStore.CountByWorkspaceIDs(ctx, []string{workspace1ID, workspace2ID, "missing-workspace"})
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[workspace1ID]).To(Equal(2))
			Expect(counts[workspace2ID]).To(Equal(1))
			Expect(counts).NotTo(HaveKey("missing-workspace"))
		})
	})
})
