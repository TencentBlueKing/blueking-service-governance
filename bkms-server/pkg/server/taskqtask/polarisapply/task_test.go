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

package polarisapply

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("Polaris dynamic apply task", func() {
	var (
		ctx               context.Context
		diApp             *fxtest.App
		appStore          bkmsapp.ApplicationStore
		envStore          bkmsenv.EnvironmentStore
		envService        *env.EnvService
		store             polaris.PolarisConfigStore
		appModelStore     appmodel.AppModelStore
		scopedEnvVarStore envvars.ScopedEnvVarStore
		depSvcStore       depsvcmodel.ServiceStore
		depSvcInstStore   depsvcmodel.ServiceInstanceStore
		depSvcBindStore   depsvcmodel.ServiceBindingStore
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
			polaris.FxModule,
			fx.Populate(
				&appStore,
				&envStore,
				&envService,
				&store,
				&appModelStore,
				&scopedEnvVarStore,
				&depSvcStore,
				&depSvcInstStore,
				&depSvcBindStore,
				&envStateManager,
			),
		)
		diApp.RequireStart()

		registry := &storereg.Registry{
			AppStore:           appStore,
			EnvStore:           envStore,
			AppModelStore:      appModelStore,
			ScopedEnvVarStore:  scopedEnvVarStore,
			PolarisConfigStore: store,
			AppDepsVarReader:   depenvvars.NewReader(depSvcInstStore, depSvcBindStore),
			PolarisVarReader:   polarisenvvars.NewReader(store),
		}
		registryMock := mockey.Mock(storereg.G).Return(registry).Build()
		DeferCleanup(registryMock.UnPatch)

		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			appModelStore,
			envvars.NewUnifiedEnvVarsReader(
				scopedEnvVarStore,
				depenvvars.NewReader(depSvcInstStore, depSvcBindStore),
				polarisenvvars.NewReader(store),
			),
			Enqueue,
		)
		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		otherEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	runHandler := func(args Args) error {
		payload, err := json.Marshal(args)
		Expect(err).NotTo(HaveOccurred())
		task := asynq.NewTask(DynamicApplyTask.Name(), payload)
		return DynamicApplyTask.Handler()(ctx, task)
	}

	createDeployedConfig := func(name string, envNames []string) *polaris.PolarisConfig {
		applied := redeployFields("k1", "t1", 8080)
		states := make(map[string]polaris.PolarisEnvState, len(envNames))
		for _, envName := range envNames {
			states[envName] = envState(applied)
		}
		config := newTestConfig(app.ID, name, envNames, states)
		Expect(store.Create(ctx, config)).To(Succeed())
		return config
	}

	Describe("Update enqueue", func() {
		It("should enqueue one asynq task per ready environment", func() {
			config := createDeployedConfig("cfg-enqueue", []string{environment.Name, otherEnvironment.Name})
			var envNames []string
			mockey.PatchConvey("enqueue succeeds", GinkgoT(), func() {
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, task *taskq.Task, _ ...asynq.Option,
				) error {
					Expect(task).NotTo(BeNil())
					envNames = append(envNames, "called")
					return nil
				}).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(envNames).To(HaveLen(2))
			})
		})

		It("should keep enqueuing other envs, record lastError, and fail Update", func() {
			config := createDeployedConfig(
				"cfg-enqueue-partial",
				[]string{environment.Name, otherEnvironment.Name},
			)
			mockey.PatchConvey("first env enqueue fails", GinkgoT(), func() {
				var calls int
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, _ *taskq.Task, _ ...asynq.Option,
				) error {
					calls++
					if calls == 1 {
						return errors.New("asynq unavailable")
					}
					return nil
				}).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).To(MatchError(ContainSubstring("enqueue polaris dynamic apply")))
				Expect(err).To(MatchError(ContainSubstring(environment.Name)))
				Expect(calls).To(Equal(2))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(ContainSubstring("asynq unavailable"))
				Expect(stored.GetEnvState(otherEnvironment.Name).LastError).To(BeEmpty())
			})
		})
	})

	Describe("handler", func() {
		It("should stop retry without recording lastError when the app no longer exists", func() {
			err := runHandler(Args{
				AppID: "missing-app", ConfigName: "cfg", EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should stop retry and record lastError when the app model no longer exists", func() {
			config := createDeployedConfig("cfg-missing-model", []string{environment.Name})
			err := runHandler(Args{
				AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
			stored, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.GetEnvState(environment.Name).LastError).To(ContainSubstring("get app model"))
		})

		It("should mark lastError as exhausted when retries are exhausted", func() {
			config := createDeployedConfig("cfg-retry-exhausted", []string{environment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			mockey.PatchConvey("retries exhausted", GinkgoT(), func() {
				mockey.Mock((*polaris.CRApplier).Apply).Return(errors.New("upsert polaris CR failed")).Build()
				mockey.Mock(asynq.GetRetryCount).Return(10, true).Build()
				mockey.Mock(asynq.GetMaxRetry).Return(10, true).Build()

				err := runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
				})
				Expect(err).To(MatchError("upsert polaris CR failed"))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(Equal(
					"upsert polaris CR failed (retry 11/11, retries exhausted)",
				))
			})
		})
	})
})

func TestFormatLastError(t *testing.T) {
	err := stderrors.New("upsert polaris CR failed")
	tests := []struct {
		name      string
		exhausted bool
		attempt   int
		total     int
		want      string
	}{
		{"first failed attempt", false, 1, 11, "upsert polaris CR failed (retry 1/11)"},
		{"later failed attempt", false, 3, 11, "upsert polaris CR failed (retry 3/11)"},
		{"retries exhausted", true, 11, 11, "upsert polaris CR failed (retry 11/11, retries exhausted)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLastError(err, tt.attempt, tt.total, tt.exhausted)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func newTestConfig(
	appID, name string,
	scopeEnvNames []string,
	envStates map[string]polaris.PolarisEnvState,
) *polaris.PolarisConfig {
	return &polaris.PolarisConfig{
		AppID: appID,
		Name:  name,
		Properties: polaris.Properties{
			InstanceKey:      "k1",
			PolarisName:      "polaris-service",
			PolarisNamespace: "Test",
			PolarisToken:     "t1",
			ServicePort:      8080,
		},
		ScopeEnvNames: scopeEnvNames,
		EnvStates:     envStates,
	}
}

func redeployFields(instanceKey, token string, servicePort int32) *polaris.RedeployRequiredFields {
	return &polaris.RedeployRequiredFields{
		InstanceKey:  instanceKey,
		PolarisToken: token,
		ServicePort:  servicePort,
	}
}

func envState(appliedFields *polaris.RedeployRequiredFields) polaris.PolarisEnvState {
	return polaris.PolarisEnvState{AppliedFields: appliedFields}
}
