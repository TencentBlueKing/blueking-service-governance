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

package lifecycle_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

var _ = Describe("Lifecycle", func() {
	var ctx context.Context
	var db *mongo.Database
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace
	var svc *lifecycle.Service
	var createdAppIDs []string

	newApp := func(framework, language string) *bkmsapp.Application {
		name := "lc-" + framework + "-" + stringx.Random(6)
		app := &bkmsapp.Application{
			ID:          name,
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeBKMSApp,
			Framework:   framework,
			Language:    language,
		}
		createdAppIDs = append(createdAppIDs, app.ID)
		return app
	}

	frameworkDefs := func(appID string) []appcfg.AppConfigFileDef {
		defs, err := appcfg.ListDefsByApp(ctx, store.Of(db, appcfg.AppConfigFileDefMeta), appID,
			appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework))
		Expect(err).NotTo(HaveOccurred())
		return defs
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		svc = lifecycle.NewService(db)
		createdAppIDs = nil
	})

	AfterEach(func() {
		for _, appID := range createdAppIDs {
			_, _ = store.Of(db, appspec.AppSpecMeta).DeleteMany(ctx, appspec.ByApp(appID))
			_ = appmodel.DeleteAppModel(ctx, db, appID)
			_, _ = store.Of(db, appcfg.AppConfigFileMeta).DeleteMany(ctx, appcfg.FilesByApp(appID))
			_, _ = store.Of(db, appcfg.AppConfigFileDefMeta).DeleteMany(ctx, appcfg.DefsByApp(appID))
			_, _ = store.Of(db, appcfg.AppConfigFileVersionMeta).DeleteMany(ctx, appcfg.VersionsByApp(appID))
			_ = store.Of(db, bkmsapp.AppMeta).DeleteIfExists(ctx, bkmsapp.ByID(appID))
		}
		Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspace.ID)).To(Succeed())
	})

	Describe("Create", func() {
		DescribeTable("writes only the new model fields",
			func(framework, language string, format appcfg.FileFormat) {
				app := newApp(framework, language)
				err := svc.Create(ctx, app, &lifecycle.CreateParams{
					Command: []string{"/app/server"},
					EnvVars: []appmodel.Variable{{Key: "FOO", Value: "bar"}},
					FrameworkFile: &lifecycle.FrameworkFileParams{
						Name: "server.conf", MountDir: "/etc/server/", Content: "",
					},
				})
				Expect(err).NotTo(HaveOccurred())

				persisted, err := store.Of(db, bkmsapp.AppMeta).Get(ctx, bkmsapp.ByID(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(persisted.Type).To(Equal(bkmsapp.AppTypeBKMSApp))
				Expect(persisted.Framework).To(Equal(framework))
				Expect(persisted.Language).To(Equal(language))
				Expect(persisted.TrpcSpec).To(BeNil())

				model, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(model.Workload.Command).To(Equal([]string{"/app/server"}))
				Expect(model.Workload.EnvVars).To(HaveLen(1))
				Expect(model.Workload.FrameworkConfig).To(BeEmpty())
				Expect(model.Workload.FrameworkConfigVersion).To(Equal(appruntime.FrameworkConfigVersionV1))
				Expect(model.Workload.Type).To(BeEmpty())
				Expect(model.Workload.TrpcConfig).To(Equal(appmodel.TrpcConfig{}))
				Expect(model.Workload.TafConfig).To(Equal(appmodel.TafConfig{}))

				defs := frameworkDefs(app.ID)
				Expect(defs).To(HaveLen(1))
				Expect(defs[0].Name).To(Equal("server.conf"))
				Expect(defs[0].MountDir).To(Equal("/etc/server"))
				file, err := store.Of(db, appcfg.AppConfigFileMeta).Get(ctx,
					appcfg.FileByDefEnv(defs[0].ID, appcfg.EnvNameDefault))
				Expect(err).NotTo(HaveOccurred())
				Expect(file.Format).To(Equal(format))

				_, err = store.Of(db, appspec.AppSpecMeta).Get(ctx, appspec.ByAppEnv(app.ID, ""))
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("trpc + go", appruntime.FrameworkTRPC, appruntime.LanguageGo, appcfg.FileFormatYAML),
			Entry("trpc + cpp", appruntime.FrameworkTRPC, appruntime.LanguageCpp, appcfg.FileFormatYAML),
			Entry("taf + cpp", appruntime.FrameworkTAF, appruntime.LanguageCpp, appcfg.FileFormatTAF),
		)

		DescribeTable("creates a blank app without a framework config file",
			func(language string) {
				app := newApp(appruntime.FrameworkBlank, language)
				Expect(svc.Create(ctx, app, &lifecycle.CreateParams{Command: []string{"/app/server"}})).To(Succeed())

				Expect(frameworkDefs(app.ID)).To(BeEmpty())
				model, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(model.Workload.FrameworkConfig).To(BeEmpty())
				Expect(model.Workload.FrameworkConfigVersion).To(Equal(appruntime.FrameworkConfigVersionV1))
			},
			Entry("go", appruntime.LanguageGo),
			Entry("cpp", appruntime.LanguageCpp),
			Entry("nodejs", appruntime.LanguageNodeJS),
			Entry("python", appruntime.LanguagePython),
		)

		DescribeTable("rejects invalid params without writing anything",
			func(mutate func(app *bkmsapp.Application, params *lifecycle.CreateParams)) {
				app := newApp(appruntime.FrameworkTRPC, appruntime.LanguageGo)
				params := &lifecycle.CreateParams{
					FrameworkFile: &lifecycle.FrameworkFileParams{Name: "trpc.yaml", MountDir: "/etc/trpc"},
				}
				mutate(app, params)

				err := svc.Create(ctx, app, params)
				Expect(err).To(MatchError(lifecycle.ErrInvalidParams))
				_, err = store.Of(db, bkmsapp.AppMeta).Get(ctx, bkmsapp.ByID(app.ID))
				Expect(err).To(MatchError(bkmsapp.ErrAppNotFound))
				_, err = store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).To(HaveOccurred())
				Expect(frameworkDefs(app.ID)).To(BeEmpty())
			},
			Entry("legacy type", func(app *bkmsapp.Application, _ *lifecycle.CreateParams) {
				app.Type = bkmsapp.AppTypeTRPC
			}),
			Entry("unsupported combination", func(app *bkmsapp.Application, _ *lifecycle.CreateParams) {
				app.Framework = appruntime.FrameworkTAF
			}),
			Entry("language only open to blank", func(app *bkmsapp.Application, _ *lifecycle.CreateParams) {
				app.Language = appruntime.LanguagePython
			}),
			Entry("missing language", func(app *bkmsapp.Application, _ *lifecycle.CreateParams) {
				app.Language = ""
			}),
			Entry("unknown frameworkConfig key", func(_ *bkmsapp.Application, params *lifecycle.CreateParams) {
				params.FrameworkConfig = map[string]any{"fileName": "trpc.yaml"}
			}),
			Entry("missing framework config file", func(_ *bkmsapp.Application, params *lifecycle.CreateParams) {
				params.FrameworkFile = nil
			}),
			Entry("empty framework config file name", func(_ *bkmsapp.Application, params *lifecycle.CreateParams) {
				params.FrameworkFile.Name = ""
			}),
			Entry("framework config file for blank", func(app *bkmsapp.Application, _ *lifecycle.CreateParams) {
				app.Framework = appruntime.FrameworkBlank
			}),
		)
	})

	Describe("UpdateStartupConfig", func() {
		var app *bkmsapp.Application

		BeforeEach(func() {
			app = newApp(appruntime.FrameworkBlank, appruntime.LanguageGo)
			Expect(svc.Create(ctx, app, &lifecycle.CreateParams{
				Command: []string{"/app/server"},
				Args:    []string{"--port", "8080"},
			})).To(Succeed())
		})

		It("replaces the start config and keeps frameworkConfig when it is omitted", func() {
			oldModel, newModel, err := svc.UpdateStartupConfig(
				ctx,
				app,
				&lifecycle.UpdateStartupConfigParams{ShCommand: "./start.sh"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldModel.Workload.Command).To(Equal([]string{"/app/server"}))
			Expect(newModel.Workload.Command).To(BeEmpty())
			Expect(newModel.Workload.Args).To(BeEmpty())
			Expect(newModel.Workload.ShCommand).To(Equal("./start.sh"))

			persisted, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(persisted.Workload.ShCommand).To(Equal("./start.sh"))
			Expect(persisted.Workload.FrameworkConfigVersion).To(Equal(appruntime.FrameworkConfigVersionV1))
		})

		It("rejects an invalid frameworkConfig without changing the model", func() {
			_, _, err := svc.UpdateStartupConfig(ctx, app, &lifecycle.UpdateStartupConfigParams{
				ShCommand:       "./start.sh",
				FrameworkConfig: map[string]any{"image": "nginx"},
			})
			Expect(err).To(MatchError(lifecycle.ErrInvalidParams))

			persisted, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(persisted.Workload.Command).To(Equal([]string{"/app/server"}))
		})

		It("rejects legacy apps", func() {
			legacy := *app
			legacy.Type = bkmsapp.AppTypeTRPC
			_, _, err := svc.UpdateStartupConfig(ctx, &legacy, &lifecycle.UpdateStartupConfigParams{})
			Expect(err).To(MatchError(lifecycle.ErrInvalidParams))
		})
	})

	Describe("UpdateSpecCompat", func() {
		var app *bkmsapp.Application

		BeforeEach(func() {
			app = newApp(appruntime.FrameworkTRPC, appruntime.LanguageGo)
			Expect(svc.Create(ctx, app, &lifecycle.CreateParams{
				Command:       []string{"/app/server"},
				FrameworkFile: &lifecycle.FrameworkFileParams{Name: "trpc_go.yaml", MountDir: "/usr/local/trpc/bin"},
			})).To(Succeed())
		})

		It("updates the start config and syncs the framework file location for bkmsApp apps", func() {
			oldModel, newModel, err := svc.UpdateSpecCompat(ctx, app, &lifecycle.SpecCompatParams{
				Framework: appruntime.FrameworkTRPC,
				Args:      []string{"--debug"},
				Language:  appruntime.LanguageGo,
				FileName:  "trpc_go.yaml",
				FilePath:  "/data/conf/",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(oldModel.Workload.Command).To(Equal([]string{"/app/server"}))
			Expect(newModel.Workload.Command).To(BeEmpty())
			Expect(newModel.Workload.Args).To(Equal([]string{"--debug"}))

			defs := frameworkDefs(app.ID)
			Expect(defs).To(HaveLen(1))
			Expect(defs[0].Name).To(Equal("trpc_go.yaml"))
			Expect(defs[0].MountDir).To(Equal("/data/conf"))
		})

		It("treats legacy apps the same way and leaves legacy fields untouched", func() {
			model, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
			Expect(err).NotTo(HaveOccurred())
			model.Workload.Type = appmodel.WorkloadTypeTrpc
			model.Workload.TrpcConfig = appmodel.TrpcConfig{Language: appruntime.LanguageGo, FileContent: "legacy"}
			Expect(appmodel.UpdateAppModel(ctx, db, model)).To(Succeed())
			legacy := *app
			legacy.Type, legacy.Framework, legacy.Language = bkmsapp.AppTypeTRPC, "", ""
			legacy.TrpcSpec = &bkmsapp.TrpcSpec{Language: appruntime.LanguageGo}

			_, newModel, err := svc.UpdateSpecCompat(ctx, &legacy, &lifecycle.SpecCompatParams{
				Framework: appruntime.FrameworkTRPC,
				ShCommand: "./start.sh",
				Language:  appruntime.LanguageGo,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(newModel.Workload.ShCommand).To(Equal("./start.sh"))
			Expect(newModel.Workload.Type).To(Equal(appmodel.WorkloadTypeTrpc))
			Expect(newModel.Workload.TrpcConfig).To(Equal(
				appmodel.TrpcConfig{Language: appruntime.LanguageGo, FileContent: "legacy"},
			))
			Expect(frameworkDefs(app.ID)[0].Name).To(Equal("trpc_go.yaml"))
		})

		It("accepts any language for legacy apps without a recorded language and does not write it", func() {
			legacy := *app
			legacy.Type, legacy.Framework, legacy.Language = bkmsapp.AppTypeTRPC, "", ""

			_, newModel, err := svc.UpdateSpecCompat(ctx, &legacy, &lifecycle.SpecCompatParams{
				Framework: appruntime.FrameworkTRPC,
				ShCommand: "./start.sh",
				Language:  appruntime.LanguageCpp,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(newModel.Workload.ShCommand).To(Equal("./start.sh"))
			Expect(newModel.Workload.TrpcConfig.Language).To(BeEmpty())
		})

		DescribeTable("rejects requests without changing the model",
			func(params lifecycle.SpecCompatParams) {
				_, _, err := svc.UpdateSpecCompat(ctx, app, &params)
				Expect(err).To(MatchError(lifecycle.ErrInvalidParams))

				persisted, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(persisted.Workload.Command).To(Equal([]string{"/app/server"}))
			},
			Entry("changing language", lifecycle.SpecCompatParams{
				Framework: appruntime.FrameworkTRPC, Language: appruntime.LanguageCpp,
			}),
			Entry("framework mismatch", lifecycle.SpecCompatParams{Framework: appruntime.FrameworkTAF}),
		)
	})
})
