package polaris_test

import (
	"context"
	"errors"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("PolarisConfigService", func() {
	var (
		ctx               context.Context
		diApp             *fxtest.App
		appStore          bkmsapp.ApplicationStore
		envStore          bkmsenv.EnvironmentStore
		envService        *env.EnvService
		store             polaris.PolarisConfigStore
		appModelStore     appmodel.AppModelStore
		scopedEnvVarStore envvars.ScopedEnvVarStore
		appDepsVarReader  *depenvvars.Reader
		polarisVarReader  *polarisenvvars.Reader
		depSvcStore       depsvcmodel.ServiceStore
		depSvcInstStore   depsvcmodel.ServiceInstanceStore
		envStateManager   *polaris.PolarisEnvStateManager
		service           *polaris.PolarisConfigService
		app               *bkmsapp.Application
		environment       *bkmsenv.Environment
		otherEnvironment  *bkmsenv.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			appmodel.FxModule,
			envvars.FxModule,
			depsvcmodel.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&appStore,
				&envStore,
				&envService,
				&store,
				&appModelStore,
				&scopedEnvVarStore,
				&appDepsVarReader,
				&polarisVarReader,
				&depSvcStore,
				&depSvcInstStore,
				&envStateManager,
			),
		)
		diApp.RequireStart()

		reader := envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader)
		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			appModelStore,
			reader,
		)
		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		otherEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	Describe("Create", func() {
		It("should not create env states before deployment", func() {
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-create",
				Properties: polaris.Properties{
					InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080,
				},
				ScopeEnvNames: []string{environment.Name, otherEnvironment.Name, environment.Name},
			}
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).To(BeEmpty())
			Expect(stored.GetEnvState(environment.Name).AppliedFields).To(BeNil())
			Expect(stored.GetEnvState(environment.Name).UpdatedAt).To(BeZero())
			Expect(stored.EnvWeights).To(Equal(map[string]int32{
				environment.Name:      polaris.DefaultEnvWeight,
				otherEnvironment.Name: polaris.DefaultEnvWeight,
			}))
		})

		It("should create no env states for an empty scope", func() {
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-empty-scope",
				Properties: polaris.Properties{
					InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080,
				},
			}
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).To(BeEmpty())
			Expect(stored.EnvWeights).To(BeEmpty())
		})

		It("should create and link a managed Polaris service", func() {
			mockey.PatchConvey("create managed Polaris service", GinkgoT(), func() {
				serviceInstanceID := bson.NewObjectID()
				mockey.Mock((*polaris.PolarisPlatformManager).CreateService).To(func(
					_ *polaris.PolarisPlatformManager,
					_ context.Context,
					params *polaris.CreatePolarisServiceParams,
				) (*polaris.CreatePolarisServiceResult, error) {
					Expect(params.AppID).To(Equal(app.ID))
					Expect(params.WorkspaceID).To(Equal(app.WorkspaceID))
					return &polaris.CreatePolarisServiceResult{
						ServiceInstanceID: serviceInstanceID,
						Token:             "managed-token",
					}, nil
				}).Build()

				config := &polaris.PolarisConfig{
					AppID: app.ID,
					Properties: polaris.Properties{
						InstanceKey: "managed", PolarisName: "managed-service",
						PolarisNamespace: "Test", ServicePort: 8080, Operator: "owner",
					},
					ScopeEnvNames: []string{environment.Name},
				}
				Expect(service.Create(ctx, app, config, true)).To(Succeed())

				stored, err := store.Get(ctx, app.ID, config.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.DepSvcInstID).To(Equal(serviceInstanceID))
				Expect(stored.PolarisToken).To(Equal("managed-token"))
				Expect(stored.GetEnvState(environment.Name).AppliedFields).To(BeNil())
				Expect(stored.EnvWeights[environment.Name]).To(Equal(polaris.DefaultEnvWeight))
			})
		})
	})

	Describe("Update", func() {
		It("should wait for deploy before creating a state for a newly scoped environment", func() {
			config := newTestConfig(app.ID, "cfg-update", nil, nil)
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{environment.Name},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvStates).NotTo(HaveKey(environment.Name))
			Expect(updated.EnvWeights[environment.Name]).To(Equal(polaris.DefaultEnvWeight))
		})
	})

	Describe("Environment weights", func() {
		It("should persist a pending environment weight without touching deployment state", func() {
			config := newTestConfig(
				app.ID,
				"cfg-weight-pending-deploy",
				[]string{environment.Name},
				nil,
			)
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 0)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvWeights).To(Equal(map[string]int32{environment.Name: 0}))
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(BeNil())
			Consistently(func(g Gomega) {
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				g.Expect(getErr).NotTo(HaveOccurred())
				g.Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
			}).WithTimeout(200 * time.Millisecond).Should(Succeed())
		})

		It("should drop weight when an undeployed environment leaves scope", func() {
			config := newTestConfig(
				app.ID,
				"cfg-weight-drop-undeployed",
				[]string{environment.Name},
				nil,
			)
			config.EnvWeights = map[string]int32{environment.Name: 35}
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvWeights).NotTo(HaveKey(environment.Name))
		})

		It("should retain weight when a deployed environment leaves scope", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-weight-retain-deployed",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name: envState(applied),
				},
			)
			config.EnvWeights = map[string]int32{environment.Name: 35}
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvWeights[environment.Name]).To(Equal(int32(35)))
			Expect(updated.EnvStates).To(HaveKey(environment.Name))
		})

		It("should reuse retained weight when a deployed environment returns to scope", func() {
			applied := redeployFields("old-key", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-weight-readd",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name: envState(applied),
				},
			)
			config.EnvWeights = map[string]int32{environment.Name: 35}
			Expect(store.Create(ctx, config)).To(Succeed())

			removed, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{},
			})
			Expect(err).NotTo(HaveOccurred())

			readded, err := service.Update(ctx, app, removed, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{environment.Name},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(readded.EnvWeights[environment.Name]).To(Equal(int32(35)))
		})

		It("should patch a deployed environment even when other fields are pending modify", func() {
			applied := redeployFields("k1", "t1", 8080)
			staleApplied := redeployFields("old-key", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-weight-put",
				[]string{environment.Name, otherEnvironment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name:      envState(staleApplied),
					otherEnvironment.Name: envState(applied),
				},
			)
			config.EnvWeights = map[string]int32{otherEnvironment.Name: 20}
			Expect(store.Create(ctx, config)).To(Succeed())

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.EnvWeights[environment.Name]).To(BeZero())
				Expect(updated.EnvWeights[otherEnvironment.Name]).To(Equal(int32(20)))
				Expect(polaris.PolarisEnvStatus(
					updated, environment.Name, updated.GetEnvState(environment.Name),
				)).To(Equal(polaris.PolarisEnvStatusPendingModify))

				Eventually(func(g Gomega) {
					stored, getErr := store.Get(ctx, app.ID, config.Name)
					g.Expect(getErr).NotTo(HaveOccurred())
					g.Expect(stored.GetEnvState(environment.Name).LastError).NotTo(BeEmpty())
				}).WithTimeout(5 * time.Second).Should(Succeed())
			})
		})

		It("should update a deployed environment outside scope and trigger a patch", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-weight-put-pending-delete",
				nil,
				map[string]polaris.PolarisEnvState{
					environment.Name: envState(applied),
				},
			)
			config.EnvWeights = map[string]int32{environment.Name: 20}
			Expect(store.Create(ctx, config)).To(Succeed())

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 25)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.EnvWeights[environment.Name]).To(Equal(int32(25)))

				Eventually(func(g Gomega) {
					stored, getErr := store.Get(ctx, app.ID, config.Name)
					g.Expect(getErr).NotTo(HaveOccurred())
					g.Expect(stored.GetEnvState(environment.Name).LastError).NotTo(BeEmpty())
				}).WithTimeout(5 * time.Second).Should(Succeed())
			})
		})
	})

	Describe("Delete", func() {
		It("should delete the linked service before deleting the config", func() {
			mockey.PatchConvey("delete linked Polaris service", GinkgoT(), func() {
				serviceInstanceID := bson.NewObjectID()
				mockey.Mock((*polaris.PolarisPlatformManager).DeleteService).To(func(
					_ *polaris.PolarisPlatformManager,
					_ context.Context,
					params *polaris.DeleteServiceParams,
				) error {
					Expect(params.ServiceInstanceID).To(Equal(serviceInstanceID))
					Expect(params.AppID).To(Equal(app.ID))
					return nil
				}).Build()

				config := &polaris.PolarisConfig{
					AppID: app.ID, Name: "cfg-delete", DepSvcInstID: serviceInstanceID,
				}
				Expect(store.Create(ctx, config)).To(Succeed())
				Expect(service.Delete(ctx, app, config)).To(Succeed())
				_, err := store.Get(ctx, app.ID, config.Name)
				Expect(err).To(MatchError(polaris.ErrConfigNotFound))
			})
		})

		It("should keep the config when deleting the linked service fails", func() {
			mockey.PatchConvey("linked Polaris service deletion fails", GinkgoT(), func() {
				serviceInstanceID := bson.NewObjectID()
				mockey.Mock((*polaris.PolarisPlatformManager).DeleteService).
					Return(errors.New("some instances existed in service")).Build()

				config := &polaris.PolarisConfig{
					AppID: app.ID, Name: "cfg-delete-blocked", DepSvcInstID: serviceInstanceID,
				}
				Expect(store.Create(ctx, config)).To(Succeed())
				Expect(service.Delete(ctx, app, config)).To(MatchError(ContainSubstring("some instances existed")))
				_, err := store.Get(ctx, app.ID, config.Name)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
