package taf_test

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf"
)

var _ = Describe("TAF application service", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var appConfigFileStore appcfg.AppConfigFileStore
	var appConfigFileVersionStore appcfg.AppConfigFileVersionStore
	var envStore envmodel.EnvironmentStore
	var envService *bkmsenv.EnvService
	var workspaceStore bkmsworkspace.WorkspaceStore
	var ruleStore appdefaults.RuleStore
	var appModelStore appmodel.AppModelStore
	var appSpecStore appspec.AppSpecStore
	var workspace *bkmsworkspace.Workspace
	var environment *envmodel.Environment
	var application *bkmsapp.Application

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			bkmsenv.FxModule,
			bkmsworkspace.FxModule,
			appmodel.FxModule,
			appspec.FxModule,
			appdefaults.FxModule,
			fx.Populate(
				&appStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&envStore,
				&envService,
				&workspaceStore,
				&ruleStore,
				&appModelStore,
				&appSpecStore,
			),
		)
		diApp.RequireStart()
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		environment = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
	})

	AfterEach(func() {
		if application != nil {
			_ = appSpecStore.DeleteByApp(ctx, application.ID)
			_ = appModelStore.DeleteAppModel(ctx, application.ID)
			_, _ = appConfigFileStore.DeleteByApp(ctx, application.ID)
			_ = appStore.DeleteAppByName(ctx, application.WorkspaceID, application.Name)
		}
		Expect(ruleStore.Drop(ctx)).To(Succeed())
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		Expect(workspaceStore.Delete(ctx, workspace.ID)).To(Succeed())
		diApp.RequireStop()
	})

	It("initializes AppModel and AppSpecs from platform defaults and workspace rules", func() {
		err := ruleStore.Create(ctx, &appdefaults.Rule{
			WorkspaceID: workspace.ID,
			ConfigType:  appspec.AppSpecSectionResources,
			EnvType:     environment.Type,
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
			EnvType:     environment.Type,
			Spec: &appspec.AppSpec{
				DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		name := "taf-defaults-" + stringx.Random(6)
		application = &bkmsapp.Application{
			ID:          name + stringx.Random(6),
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeTAF,
		}
		err = taf.NewService(
			appModelStore,
			appSpecStore,
			ruleStore,
			envStore,
			appConfigFileStore,
			appConfigFileVersionStore,
			appStore,
		).Create(ctx, application, &taf.CreateParams{
			TafConfig: &taf.TafConfigParams{
				FileName: "taf.conf",
				FilePath: "/etc/taf",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		persistedModel, err := appModelStore.GetAppModel(ctx, application.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persistedModel.Workload.Type).To(Equal(appmodel.WorkloadTypeTaf))
		Expect(*persistedModel.Replicas).To(Equal(int32(1)))
		Expect(persistedModel.Workload.Resources).To(Equal(map[string]string{
			"cpu":    "1-2",
			"memory": "2Gi-4Gi",
		}))
		Expect(*persistedModel.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*persistedModel.UpdateStrategy.MaxSurge).To(Equal("25%"))

		defaultSpec, err := appSpecStore.Get(ctx, application.ID, appspec.DefaultEnvName)
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

		envSpec, err := appSpecStore.Get(ctx, application.ID, environment.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(*envSpec.Resources.Replicas).To(Equal(int32(2)))
		Expect(*envSpec.Resources.CPURequests).To(Equal("500m"))
		Expect(*envSpec.DevMode.Enabled).To(BeTrue())
		Expect(envSpec.UpdateStrategy).To(BeNil())
	})

	It("persists platform defaults when the workspace has no rules", func() {
		rules, err := ruleStore.List(ctx, workspace.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(BeEmpty())

		name := "taf-without-defaults-" + stringx.Random(6)
		application = &bkmsapp.Application{
			ID:          name + stringx.Random(6),
			Name:        name,
			DisplayName: name,
			WorkspaceID: workspace.ID,
			Type:        bkmsapp.AppTypeTAF,
		}
		err = taf.NewService(
			appModelStore,
			appSpecStore,
			ruleStore,
			envStore,
			appConfigFileStore,
			appConfigFileVersionStore,
			appStore,
		).Create(ctx, application, &taf.CreateParams{
			TafConfig: &taf.TafConfigParams{
				FileName: "taf.conf",
				FilePath: "/etc/taf",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		persistedModel, err := appModelStore.GetAppModel(ctx, application.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persistedModel.Workload.Type).To(Equal(appmodel.WorkloadTypeTaf))
		Expect(*persistedModel.Replicas).To(Equal(int32(1)))
		Expect(persistedModel.Workload.Resources).To(Equal(map[string]string{
			"cpu":    "1-2",
			"memory": "2Gi-4Gi",
		}))
		Expect(*persistedModel.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*persistedModel.UpdateStrategy.MaxSurge).To(Equal("25%"))

		defaultSpec, err := appSpecStore.Get(ctx, application.ID, appspec.DefaultEnvName)
		Expect(err).NotTo(HaveOccurred())
		Expect(defaultSpec.AppID).To(Equal(application.ID))
		Expect(defaultSpec.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(*defaultSpec.Resources.Replicas).To(Equal(int32(1)))
		Expect(*defaultSpec.UpdateStrategy.MaxUnavailable).To(Equal("25%"))

		_, err = appSpecStore.Get(ctx, application.ID, environment.Name)
		Expect(errors.Is(err, appspec.ErrAppSpecNotFound)).To(BeTrue())
	})
})
