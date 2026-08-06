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

package polaris_test

import (
	"context"
	"errors"

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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
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
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name, otherEnvironment.Name, environment.Name},
			}
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).To(BeEmpty())
			Expect(stored.GetEnvState(environment.Name).AppliedFields).To(BeNil())
			Expect(stored.GetEnvState(environment.Name).UpdatedAt).To(BeZero())
		})

		It("should create no env states for an empty scope", func() {
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-empty-scope",
				Properties: polaris.Properties{
					InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080,
				},
				ScopeType: component.ScopeTypeEnvironment,
			}
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).To(BeEmpty())
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
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{environment.Name},
				}
				Expect(service.Create(ctx, app, config, true)).To(Succeed())

				stored, err := store.Get(ctx, app.ID, config.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.DepSvcInstID).To(Equal(serviceInstanceID))
				Expect(stored.PolarisToken).To(Equal("managed-token"))
				Expect(stored.GetEnvState(environment.Name).AppliedFields).To(BeNil())
			})
		})
	})

	Describe("Update", func() {
		It("should wait for deploy before creating a state for a newly scoped environment", func() {
			config := newTestConfig(app.ID, "cfg-update", nil, nil)
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				Scope: &polaris.PatchPolarisScope{
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{environment.Name},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvStates).NotTo(HaveKey(environment.Name))
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
