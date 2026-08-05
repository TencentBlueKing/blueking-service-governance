package migrate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/migrate"
)

const tkeRouteEniName = "tkeRouteEni"

var _ = Describe("tkeRouteEni component migration", func() {
	var (
		ctx           context.Context
		diApp         *fxtest.App
		appStore      bkmsapp.ApplicationStore
		appModelStore appmodel.AppModelStore
		appSpecStore  appspec.AppSpecStore
		wsCompStore   workspace.WorkspaceCompsStore
		compDefStore  component.ComponentDefStore
		migrator      *migrate.Migrator
	)

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupCollection(appmodel.CollectionName)).To(Succeed())
		Expect(testutil.CleanupCollection("app_specs")).To(Succeed())
		Expect(testutil.CleanupCollection("workspace_components")).To(Succeed())
		Expect(testutil.CleanupCollection("component_defs")).To(Succeed())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			appspec.FxModule,
			workspace.FxModule,
			component.FxModule,
			fx.Populate(&appStore, &appModelStore, &appSpecStore, &wsCompStore, &compDefStore),
		)
		diApp.RequireStart()

		migrator = migrate.New(
			database.Client().Database(database.Name()),
			appStore, appSpecStore, appModelStore, wsCompStore, compDefStore,
		)
		Expect(compDefStore.Create(ctx, &component.ComponentDef{
			Name:    tkeRouteEniName,
			Version: component.DefaultComponentDefVersion,
			Patchers: []string{`spec.template.metadata:
  annotations:
    tke.cloud.tencent.com/networks: tke-route-eni
`},
			Creator: "admin",
			Updater: "admin",
		})).To(Succeed())
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	createAppWithDirectMount := func() *bkmsapp.Application {
		app := dbfactory.Application(ctx, appStore)
		Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Workload: appmodel.Workload{Type: appmodel.WorkloadTypeTrpc, Name: app.Name},
		})).To(Succeed())
		Expect(appModelStore.AddComponent(ctx, app.ID, &component.Component{
			Name: "tkerouteeni-aaaaa",
			ComponentInst: component.ComponentInst{
				Type: tkeRouteEniName, Version: component.DefaultComponentDefVersion,
			},
		})).To(Succeed())
		return app
	}

	It("dry-run reports without writing", func() {
		app := createAppWithDirectMount()
		result, err := migrator.Run(ctx, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.AppSpecWrites).To(Equal(1))
		Expect(result.AppComponentsRemoved).To(Equal(1))

		am, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(am.Components).To(HaveLen(1))
	})

	It("migrates direct mounts to AppSpec and removes components", func() {
		app := createAppWithDirectMount()
		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.AppSpecWrites).To(Equal(1))
		Expect(result.AppComponentsRemoved).To(Equal(1))

		spec, err := appSpecStore.Get(ctx, app.ID, appspec.DefaultEnvName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*spec.TkeRouteEni.Enabled).To(BeTrue())

		am, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(am.Components).To(BeEmpty())
	})

	It("skips AppSpec write when already enabled", func() {
		app := createAppWithDirectMount()
		Expect(appspec.SetDefaultSection(
			ctx, appSpecStore, appModelStore, app.ID,
			appspec.TkeRouteEniSection,
			&appspec.TkeRouteEniSpec{Enabled: lo.ToPtr(true)},
			appspec.SectionWriteModeReplace,
		)).To(Succeed())

		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.AppSpecWrites).To(BeZero())
		Expect(result.AppComponentsRemoved).To(Equal(1))

		am, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(am.Components).To(BeEmpty())
	})

	It("expands workspace environment scope and cleans refs", func() {
		app := dbfactory.Application(ctx, appStore)
		Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Workload: appmodel.Workload{Type: appmodel.WorkloadTypeTrpc, Name: app.Name},
		})).To(Succeed())

		wsComp := &workspace.Component{
			ComponentInst: component.ComponentInst{
				Type: tkeRouteEniName, Version: component.DefaultComponentDefVersion,
			},
			Name:          "tkerouteeni-ws001",
			WorkspaceID:   app.WorkspaceID,
			ScopeType:     component.ScopeTypeEnvironment,
			ScopeEnvNames: []string{"prod", "stag"},
		}
		Expect(wsCompStore.Add(ctx, wsComp)).To(Succeed())
		Expect(appModelStore.AddComponent(ctx, app.ID, &component.Component{
			Name:         "ref-tkerouteeni",
			ComponentRef: component.ComponentRef{RefWorkspaceCompName: wsComp.Name},
		})).To(Succeed())

		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.AppSpecWrites).To(Equal(2))
		Expect(result.WorkspaceComponentsRemoved).To(Equal(1))

		for _, envName := range []string{"prod", "stag"} {
			spec, getErr := appSpecStore.Get(ctx, app.ID, envName)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(*spec.TkeRouteEni.Enabled).To(BeTrue())
		}
		_, err = wsCompStore.GetByName(ctx, app.WorkspaceID, wsComp.Name)
		Expect(err).To(MatchError(workspace.ErrComponentNotFound))
	})

	It("matches workspace refs by workspaceID+name, not name alone", func() {
		const sharedName = "tkerouteeni-shared"

		appA := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
			WorkspaceID: "ws-a",
		})
		appB := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
			WorkspaceID: "ws-b",
		})
		for _, app := range []*bkmsapp.Application{appA, appB} {
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{Type: appmodel.WorkloadTypeTrpc, Name: app.Name},
			})).To(Succeed())
		}

		Expect(wsCompStore.Add(ctx,
			&workspace.Component{
				ComponentInst: component.ComponentInst{
					Type: tkeRouteEniName, Version: component.DefaultComponentDefVersion,
				},
				Name: sharedName, WorkspaceID: "ws-a",
				ScopeType: component.ScopeTypeEnvironment, ScopeEnvNames: []string{"prod"},
			},
			&workspace.Component{
				ComponentInst: component.ComponentInst{
					Type: tkeRouteEniName, Version: component.DefaultComponentDefVersion,
				},
				Name: sharedName, WorkspaceID: "ws-b",
				ScopeType: component.ScopeTypeEnvironment, ScopeEnvNames: []string{"stag"},
			},
		)).To(Succeed())

		Expect(appModelStore.AddComponent(ctx, appA.ID, &component.Component{
			Name:         "ref-a",
			ComponentRef: component.ComponentRef{RefWorkspaceCompName: sharedName},
		})).To(Succeed())
		Expect(appModelStore.AddComponent(ctx, appB.ID, &component.Component{
			Name:         "ref-b",
			ComponentRef: component.ComponentRef{RefWorkspaceCompName: sharedName},
		})).To(Succeed())

		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.AppSpecWrites).To(Equal(2))
		Expect(result.WorkspaceComponentsRemoved).To(Equal(2))

		specA, err := appSpecStore.Get(ctx, appA.ID, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(*specA.TkeRouteEni.Enabled).To(BeTrue())
		_, err = appSpecStore.Get(ctx, appA.ID, "stag")
		Expect(err).To(MatchError(appspec.ErrAppSpecNotFound))

		specB, err := appSpecStore.Get(ctx, appB.ID, "stag")
		Expect(err).NotTo(HaveOccurred())
		Expect(*specB.TkeRouteEni.Enabled).To(BeTrue())
		_, err = appSpecStore.Get(ctx, appB.ID, "prod")
		Expect(err).To(MatchError(appspec.ErrAppSpecNotFound))
	})
})
