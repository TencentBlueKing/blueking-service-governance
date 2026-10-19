package trpc_test

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc"
)

var _ = Describe("tRPC application service", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appStore *bkmsapp.ApplicationStore
	var envService *bkmsenv.EnvService
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var ruleStore *appdefaults.RuleStore
	var db *mongo.Database
	var workspace *bkmsworkspace.Workspace
	var environment *envmodel.Environment
	var application *bkmsapp.Application

	BeforeEach(func() {
		ctx = context.Background()
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			appcfg.FxModule,
			bkmsenv.FxModule,
			fx.Populate(
				&envService,
			),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		ruleStore = testutil.NewStore(GinkgoT(), appdefaults.RuleMeta)
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		environment = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
	})

	AfterEach(func() {
		if application != nil {
			_, _ = store.Of(db, appspec.AppSpecMeta).DeleteMany(ctx, appspec.ByApp(application.ID))
			_ = appmodel.DeleteAppModel(ctx, db, application.ID)
			_, _ = store.Of(database.DB(), appcfg.AppConfigFileMeta).
				DeleteMany(ctx, appcfg.FilesByApp(application.ID))
			_ = appStore.DeleteIfExists(ctx, bkmsapp.ByName(application.WorkspaceID, application.Name))
		}
		Expect(testutil.CleanupMetaColl(appdefaults.RuleMeta)).To(Succeed())
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspace.ID)).To(Succeed())
		diApp.RequireStop()
	})

	It("initializes AppModel and AppSpecs from platform defaults and workspace rules", func() {
		err := ruleStore.Create(ctx, &appdefaults.Rule{
			WorkspaceID: workspace.ID,
			ConfigType:  appspec.AppSpecSectionResources,
			EnvTypes:    []string{environment.Type},
			Spec: &appspec.AppSpec{
				Resources: &appspec.ResourcesSpec{
					Replicas:       lo.ToPtr(int32(2)),
					CPURequests:    lo.ToPtr("500m"),
					CPULimits:      lo.ToPtr("1"),
					MemoryRequests: lo.ToPtr("1Gi"),
					MemoryLimits:   lo.ToPtr("2Gi"),
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		err = ruleStore.Create(ctx, &appdefaults.Rule{
			WorkspaceID: workspace.ID,
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvTypes:    []string{environment.Type},
			Spec: &appspec.AppSpec{
				DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		name := "trpc-defaults-" + stringx.Random(6)
		application = &bkmsapp.Application{
			ID:          name + stringx.Random(6),
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeTRPC,
		}
		err = trpc.NewService(db).Create(ctx, application, &trpc.CreateParams{
			TrpcConfig: &trpc.TrpcConfigParams{
				Language: "go",
				FileName: "trpc.yaml",
				FilePath: "/etc/trpc",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		persistedModel, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(application.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(persistedModel.Workload.Type).To(Equal(appmodel.WorkloadTypeTrpc))
		Expect(persistedModel.Workload.TrpcConfig.Language).To(Equal("go"))
		Expect(persistedModel.Workload.TrpcConfig.FileName).To(BeEmpty())
		Expect(persistedModel.Workload.TrpcConfig.FilePath).To(BeEmpty())
		defs, listErr := appcfg.ListDefsByApp(
			ctx,
			store.Of(db, appcfg.AppConfigFileDefMeta),
			application.ID,
			appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
		)
		Expect(listErr).NotTo(HaveOccurred())
		Expect(defs).To(HaveLen(1))
		Expect(defs[0].Name).To(Equal("trpc.yaml"))
		Expect(defs[0].MountDir).To(Equal("/etc/trpc"))
		Expect(*persistedModel.Replicas).To(Equal(int32(1)))
		Expect(persistedModel.Workload.Resources).To(Equal(map[string]string{
			"cpu":    "1-2",
			"memory": "2Gi-4Gi",
		}))
		Expect(*persistedModel.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*persistedModel.UpdateStrategy.MaxSurge).To(Equal("25%"))

		defaultSpec, err := store.Of(db, appspec.AppSpecMeta).
			Get(ctx, appspec.ByAppEnv(application.ID, appspec.DefaultEnvName))
		Expect(err).NotTo(HaveOccurred())
		Expect(*defaultSpec.Resources.Replicas).To(Equal(int32(1)))
		Expect(*defaultSpec.Resources.CPURequests).To(Equal("1"))
		Expect(*defaultSpec.Resources.CPULimits).To(Equal("2"))
		Expect(*defaultSpec.Resources.MemoryRequests).To(Equal("2Gi"))
		Expect(*defaultSpec.Resources.MemoryLimits).To(Equal("4Gi"))
		Expect(*defaultSpec.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*defaultSpec.UpdateStrategy.MaxSurge).To(Equal("25%"))
		Expect(defaultSpec.Labels).To(BeNil())
		Expect(defaultSpec.Annotations).To(BeNil())

		envSpec, err := store.Of(db, appspec.AppSpecMeta).Get(ctx, appspec.ByAppEnv(application.ID, environment.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(*envSpec.Resources.Replicas).To(Equal(int32(2)))
		Expect(*envSpec.Resources.CPURequests).To(Equal("500m"))
		Expect(*envSpec.DevMode.Enabled).To(BeTrue())
		Expect(envSpec.UpdateStrategy).To(BeNil())
	})

	It("persists platform defaults when the workspace has no rules", func() {
		rules, err := ruleStore.List(ctx, appdefaults.ByWorkspace(workspace.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(BeEmpty())

		name := "trpc-without-defaults-" + stringx.Random(6)
		application = &bkmsapp.Application{
			ID:          name + stringx.Random(6),
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeTRPC,
		}
		err = trpc.NewService(db).Create(ctx, application, &trpc.CreateParams{
			TrpcConfig: &trpc.TrpcConfigParams{
				Language: "go",
				FileName: "trpc.yaml",
				FilePath: "/etc/trpc",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		persistedModel, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(application.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(persistedModel.Workload.Type).To(Equal(appmodel.WorkloadTypeTrpc))
		Expect(*persistedModel.Replicas).To(Equal(int32(1)))
		Expect(persistedModel.Workload.Resources).To(Equal(map[string]string{
			"cpu":    "1-2",
			"memory": "2Gi-4Gi",
		}))
		Expect(*persistedModel.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*persistedModel.UpdateStrategy.MaxSurge).To(Equal("25%"))

		defaultSpec, err := store.Of(db, appspec.AppSpecMeta).
			Get(ctx, appspec.ByAppEnv(application.ID, appspec.DefaultEnvName))
		Expect(err).NotTo(HaveOccurred())
		Expect(defaultSpec.AppID).To(Equal(application.ID))
		Expect(defaultSpec.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(*defaultSpec.Resources.Replicas).To(Equal(int32(1)))
		Expect(*defaultSpec.UpdateStrategy.MaxUnavailable).To(Equal("25%"))

		_, err = store.Of(db, appspec.AppSpecMeta).Get(ctx, appspec.ByAppEnv(application.ID, environment.Name))
		Expect(errors.Is(err, appspec.ErrAppSpecNotFound)).To(BeTrue())
	})

	It("updates the framework def and leaves the model file location empty", func() {
		name := "trpc-update-" + stringx.Random(6)
		application = &bkmsapp.Application{
			ID:          name + stringx.Random(6),
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeTRPC,
		}
		svc := trpc.NewService(db)
		err := svc.Create(ctx, application, &trpc.CreateParams{
			TrpcConfig: &trpc.TrpcConfigParams{
				Language: "go",
				FileName: "trpc.yaml",
				FilePath: "/etc/trpc",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		_, updated, err := svc.Update(ctx, application, &trpc.UpdateParams{
			Command: []string{"/bin/app"},
			TrpcConfig: &trpc.TrpcConfigParams{
				Language: "cpp",
				FileName: "trpc.yaml",
				FilePath: "/usr/local/trpc/bin/",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Workload.Command).To(Equal([]string{"/bin/app"}))
		Expect(updated.Workload.TrpcConfig.Language).To(Equal("cpp"))
		Expect(updated.Workload.TrpcConfig.FileName).To(BeEmpty())
		Expect(updated.Workload.TrpcConfig.FilePath).To(BeEmpty())

		persisted, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(application.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(persisted.Workload.TrpcConfig.FileName).To(BeEmpty())
		Expect(persisted.Workload.TrpcConfig.FilePath).To(BeEmpty())

		defs, err := appcfg.ListDefsByApp(
			ctx,
			store.Of(db, appcfg.AppConfigFileDefMeta),
			application.ID,
			appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(defs).To(HaveLen(1))
		Expect(defs[0].Name).To(Equal("trpc.yaml"))
		Expect(defs[0].MountDir).To(Equal("/usr/local/trpc/bin"))
	})
})
