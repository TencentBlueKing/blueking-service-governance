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
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
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

		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			appModelStore,
			envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
			nil,
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

		It("should reject an empty operator", func() {
			config := newTestConfig(app.ID, "cfg-operator-empty", nil, nil)
			config.DepSvcInstID = bson.NewObjectID()
			config.Operator = "zhangsan"
			Expect(store.Create(ctx, config)).To(Succeed())

			empty := ""
			_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Operator: &empty})
			Expect(err).To(MatchError(polaris.ErrOperatorEmpty))

			stored, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.Operator).To(Equal("zhangsan"))
		})

		It("should reject operator updates for imported polaris services", func() {
			config := newTestConfig(app.ID, "cfg-operator-imported", nil, nil)
			config.Operator = "zhangsan"
			Expect(store.Create(ctx, config)).To(Succeed())

			operator := "lisi"
			_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Operator: &operator})
			Expect(err).To(MatchError(polaris.ErrOperatorNotManaged))

			stored, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.Operator).To(Equal("zhangsan"))
		})

		It("should sync polaris owners then persist operator for managed configs", func() {
			mockey.PatchConvey("update managed polaris owners", GinkgoT(), func() {
				config := newTestConfig(app.ID, "cfg-operator-managed", nil, nil)
				config.DepSvcInstID = bson.NewObjectID()
				config.Operator = "zhangsan"
				Expect(store.Create(ctx, config)).To(Succeed())

				mockey.Mock((*polaris.PolarisPlatformManager).UpdateServiceOwners).To(func(
					_ *polaris.PolarisPlatformManager,
					_ context.Context,
					got *polaris.PolarisConfig,
					owners string,
				) error {
					Expect(got.Name).To(Equal(config.Name))
					Expect(owners).To(Equal("lisi"))
					return nil
				}).Build()

				operator := "lisi"
				updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Operator: &operator})
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Operator).To(Equal("lisi"))
			})
		})

		It("should not enqueue dynamic apply when only operator is updated", func() {
			mockey.PatchConvey("operator-only update skips dynamic apply", GinkgoT(), func() {
				enqueued := 0
				service = polaris.NewPolarisConfigService(
					store,
					polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
					envStateManager,
					envStore,
					appModelStore,
					envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
					func(_ context.Context, _, _, _ string) error {
						enqueued++
						return nil
					},
				)

				applied := redeployFields("k1", "t1", 8080)
				config := newTestConfig(
					app.ID,
					"cfg-operator-no-apply",
					[]string{environment.Name},
					map[string]polaris.PolarisEnvState{
						environment.Name: envState(applied),
					},
				)
				config.DepSvcInstID = bson.NewObjectID()
				config.Operator = "zhangsan"
				Expect(store.Create(ctx, config)).To(Succeed())

				mockey.Mock((*polaris.PolarisPlatformManager).UpdateServiceOwners).Return(nil).Build()

				operator := "lisi"
				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Operator: &operator})
				Expect(err).NotTo(HaveOccurred())
				Expect(enqueued).To(Equal(0))
			})
		})

		It("should not persist operator when polaris owner sync fails", func() {
			mockey.PatchConvey("polaris owners update fails", GinkgoT(), func() {
				config := newTestConfig(app.ID, "cfg-operator-sync-fail", nil, nil)
				config.DepSvcInstID = bson.NewObjectID()
				config.Operator = "zhangsan"
				Expect(store.Create(ctx, config)).To(Succeed())

				mockey.Mock((*polaris.PolarisPlatformManager).UpdateServiceOwners).Return(
					errors.New("invalid owners"),
				).Build()

				operator := "lisi"
				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Operator: &operator})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid owners"))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.Operator).To(Equal("zhangsan"))
			})
		})
	})

	Describe("Weight factor", func() {
		It("should persist the switch on create", func() {
			config := newTestConfig(app.ID, "cfg-weight-factor-create", []string{environment.Name}, nil)
			config.EnableWeightFactor = true
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnableWeightFactor).To(BeTrue())
			// 动态权重开关不随环境加入 scope 预建，缺省即关闭
			Expect(stored.EnvDynamicWeights).To(BeEmpty())
		})

		It("should default the switch to off when create omits it", func() {
			config := newTestConfig(app.ID, "cfg-weight-factor-default", []string{environment.Name}, nil)
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnableWeightFactor).To(BeFalse())
		})

		It("should not enqueue dynamic apply when only the weight factor is updated", func() {
			enqueued := 0
			service = polaris.NewPolarisConfigService(
				store,
				polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
				envStateManager,
				envStore,
				appModelStore,
				envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
				func(_ context.Context, _, _, _ string) error {
					enqueued++
					return nil
				},
			)

			applied := redeployFields("k1", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-weight-factor-no-apply",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)
			Expect(store.Create(ctx, config)).To(Succeed())

			enabled := true
			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				EnableWeightFactor: &enabled,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnableWeightFactor).To(BeTrue())
			// 该字段不参与 CR 组装，单独更新不应空转下发
			Expect(enqueued).To(Equal(0))
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(Equal(applied))
		})

		It("should keep the environment switches when turned off", func() {
			config := newTestConfig(
				app.ID,
				"cfg-weight-factor-off",
				[]string{environment.Name, otherEnvironment.Name},
				nil,
			)
			config.EnableWeightFactor = true
			config.EnvDynamicWeights = map[string]bool{
				environment.Name:      true,
				otherEnvironment.Name: true,
			}
			Expect(store.Create(ctx, config)).To(Succeed())

			disabled := false
			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				EnableWeightFactor: &disabled,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnableWeightFactor).To(BeFalse())
			// 环境级开关是用户意图的记录，关闭配置级开关不改写它们
			Expect(updated.EnvDynamicWeights).To(Equal(map[string]bool{
				environment.Name:      true,
				otherEnvironment.Name: true,
			}))
		})

		It("should restore the environment switches when turned back on", func() {
			config := newTestConfig(app.ID, "cfg-weight-factor-restore", []string{environment.Name}, nil)
			config.EnvDynamicWeights = map[string]bool{environment.Name: true}
			Expect(store.Create(ctx, config)).To(Succeed())

			enabled := true
			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				EnableWeightFactor: &enabled,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnableWeightFactor).To(BeTrue())
			Expect(updated.EnvDynamicWeights).To(Equal(map[string]bool{environment.Name: true}))
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

			updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 0, nil)

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

		It("should return a patch error without persisting weight when other fields are pending modify", func() {
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

				updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 0, nil)
				Expect(err).To(MatchError(ContainSubstring("patch env weight")))
				Expect(updated).To(BeNil())
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.EnvWeights).NotTo(HaveKey(environment.Name))
				Expect(stored.EnvWeights[otherEnvironment.Name]).To(Equal(int32(20)))
				Expect(polaris.PolarisEnvStatus(
					stored, environment.Name, stored.GetEnvState(environment.Name),
				)).To(Equal(polaris.PolarisEnvStatusPendingModify))
				Expect(stored.GetEnvState(environment.Name).LastError).To(BeEmpty())
			})
		})

		It("should return a patch error without changing an out-of-scope deployed weight", func() {
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

				updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 25, nil)
				Expect(err).To(MatchError(ContainSubstring("patch env weight")))
				Expect(updated).To(BeNil())
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.EnvWeights[environment.Name]).To(Equal(int32(20)))
				Expect(stored.GetEnvState(environment.Name).LastError).To(BeEmpty())
			})
		})
	})

	Describe("Environment dynamic weights", func() {
		It("should persist the switch of an undeployed environment without any cluster call", func() {
			config := newTestConfig(app.ID, "cfg-dynamic-weight-pending", []string{environment.Name}, nil)
			config.EnableWeightFactor = true
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 80, lo.ToPtr(true))

			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvWeights[environment.Name]).To(Equal(int32(80)))
			Expect(updated.EnvDynamicWeights).To(Equal(map[string]bool{environment.Name: true}))
			Expect(polaris.PolarisEnvStatus(
				updated, environment.Name, updated.GetEnvState(environment.Name),
			)).To(Equal(polaris.PolarisEnvStatusPendingCreate))
		})

		It("should keep the switch untouched when the request omits it", func() {
			config := newTestConfig(app.ID, "cfg-dynamic-weight-omitted", []string{environment.Name}, nil)
			config.EnableWeightFactor = true
			config.EnvDynamicWeights = map[string]bool{environment.Name: true}
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 60, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvWeights[environment.Name]).To(Equal(int32(60)))
			Expect(updated.EnvDynamicWeights[environment.Name]).To(BeTrue())
		})

		It("should drop the switch when an undeployed environment leaves scope", func() {
			config := newTestConfig(app.ID, "cfg-dynamic-weight-drop", []string{environment.Name}, nil)
			config.EnableWeightFactor = true
			config.EnvDynamicWeights = map[string]bool{environment.Name: true}
			Expect(store.Create(ctx, config)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.EnvDynamicWeights).NotTo(HaveKey(environment.Name))
		})

		It("should retain and reuse the switch when a deployed environment leaves and returns to scope", func() {
			// 快照与当前配置不一致，回到 scope 时不会触发动态下发
			applied := redeployFields("old-key", "t1", 8080)
			config := newTestConfig(
				app.ID,
				"cfg-dynamic-weight-retain",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)
			config.EnableWeightFactor = true
			config.EnvDynamicWeights = map[string]bool{environment.Name: true}
			Expect(store.Create(ctx, config)).To(Succeed())

			removed, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(removed.EnvDynamicWeights[environment.Name]).To(BeTrue())

			readded, err := service.Update(ctx, app, removed, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{environment.Name},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(readded.EnvDynamicWeights[environment.Name]).To(BeTrue())
		})

		// 已部署环境走即时 Patch。集群侧的断言在带 k8s 标签的用例里，本机无集群时会跳过，
		// 所以这里直接拦下 applier，校验 service 算出的下发值。
		patchedDynamicWeight := func(
			configName string, weightFactor bool, stored, requested *bool,
		) bool {
			applied := redeployFields("k1", "t1", 8080)
			config := newTestConfig(
				app.ID,
				configName,
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)
			config.EnableWeightFactor = weightFactor
			if stored != nil {
				config.EnvDynamicWeights = map[string]bool{environment.Name: *stored}
			}
			Expect(store.Create(ctx, config)).To(Succeed())

			var patched bool
			mockey.Mock((*polaris.CRApplier).PatchWeight).To(func(
				_ *polaris.CRApplier,
				_ context.Context,
				_ *bkmsapp.Application,
				_ *bkmsenv.Environment,
				_ *polaris.PolarisConfig,
				_ int32,
				dynamicWeight bool,
			) error {
				patched = dynamicWeight
				return nil
			}).Build()

			_, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 40, requested)
			Expect(err).NotTo(HaveOccurred())
			return patched
		}

		It("should patch the requested switch while the weight factor is on", func() {
			mockey.PatchConvey("capture the patched dynamic weight", GinkgoT(), func() {
				Expect(patchedDynamicWeight(
					"cfg-dynamic-weight-patch-on", true, nil, lo.ToPtr(true),
				)).To(BeTrue())
			})
		})

		It("should patch the requested switch even while the weight factor is off", func() {
			mockey.PatchConvey("capture the patched dynamic weight", GinkgoT(), func() {
				Expect(patchedDynamicWeight(
					"cfg-dynamic-weight-patch-gated", false, nil, lo.ToPtr(true),
				)).To(BeTrue())
			})
		})

		It("should patch the stored switch when the request omits it", func() {
			mockey.PatchConvey("capture the patched dynamic weight", GinkgoT(), func() {
				Expect(patchedDynamicWeight(
					"cfg-dynamic-weight-patch-omitted", true, lo.ToPtr(true), nil,
				)).To(BeTrue())
			})
		})

		It("should not pre-create a default switch for an environment joining scope", func() {
			config := newTestConfig(app.ID, "cfg-dynamic-weight-no-default", []string{environment.Name}, nil)
			config.EnableWeightFactor = true
			Expect(service.Create(ctx, app, config, false)).To(Succeed())

			updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
				ScopeEnvNames: []string{environment.Name, otherEnvironment.Name},
			})
			Expect(err).NotTo(HaveOccurred())
			// 权重补默认值，动态权重开关缺省即关闭，不预建条目
			Expect(updated.EnvWeights).To(HaveKey(otherEnvironment.Name))
			Expect(updated.EnvDynamicWeights).To(BeEmpty())
		})
	})

	Describe("Immediate register mode", func() {
		BeforeEach(func() {
			// 即时下发要读取应用模型来渲染环境变量
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })
		})

		newImmediateConfig := func(name string, scopeEnvNames []string) *polaris.PolarisConfig {
			config := newTestConfig(app.ID, name, scopeEnvNames, nil)
			config.RegisterMode = polaris.RegisterModeImmediate
			return config
		}

		It("should not write per-env lastError when reading the app model fails", func() {
			Expect(appModelStore.DeleteAppModel(ctx, app.ID)).To(Succeed())
			config := newImmediateConfig("cfg-immediate-no-model", []string{environment.Name, otherEnvironment.Name})

			err := service.Create(ctx, app, config, false)
			Expect(err).To(MatchError(polaris.ErrClusterSyncFailed))
			Expect(err).To(MatchError(ContainSubstring("get app model")))

			stored, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.EnvStates).To(BeEmpty())
		})

		It("should keep the config and record the failure on every scoped environment", func() {
			config := newImmediateConfig("cfg-immediate-create", []string{environment.Name, otherEnvironment.Name})

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				err := service.Create(ctx, app, config, false)
				Expect(err).To(MatchError(polaris.ErrClusterSyncFailed))
				Expect(err).To(MatchError(ContainSubstring(environment.Name)))
				Expect(err).To(MatchError(ContainSubstring(otherEnvironment.Name)))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				for _, envName := range []string{environment.Name, otherEnvironment.Name} {
					state := stored.GetEnvState(envName)
					Expect(state.LastError).To(ContainSubstring("test discovery error"))
					Expect(state.AppliedFields).To(BeNil())
					Expect(polaris.PolarisEnvStatus(stored, envName, state)).
						To(Equal(polaris.PolarisEnvStatusPendingCreate))
				}
			})
		})

		It("should not touch the cluster when the scope is empty", func() {
			config := newImmediateConfig("cfg-immediate-empty-scope", nil)

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				Expect(service.Create(ctx, app, config, false)).To(Succeed())
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(BeEmpty())
			})
		})

		It("should drop the weight of an environment leaving scope even when it was applied", func() {
			config := newImmediateConfig("cfg-immediate-release-weight", []string{environment.Name})
			config.EnvWeights = map[string]int32{environment.Name: 35}
			config.EnvStates = map[string]polaris.PolarisEnvState{
				environment.Name: envState(redeployFields("k1", "t1", 8080)),
			}
			Expect(store.Create(ctx, config)).To(Succeed())

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					ScopeEnvNames: []string{},
				})
				// 删除集群资源失败，环境记录保留下来供下次保存时重试
				Expect(err).To(MatchError(polaris.ErrClusterSyncFailed))
				Expect(updated.EnvWeights).NotTo(HaveKey(environment.Name))
				Expect(updated.GetEnvState(environment.Name).LastError).
					To(ContainSubstring("test discovery error"))
			})
		})

		It("should refuse to delete the config while its cluster resources remain", func() {
			config := newImmediateConfig("cfg-immediate-delete-blocked", []string{environment.Name})
			Expect(store.Create(ctx, config)).To(Succeed())

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				Expect(service.Delete(ctx, app, config)).To(MatchError(polaris.ErrClusterSyncFailed))
				_, err := store.Get(ctx, app.ID, config.Name)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should leave on_deploy configs on the asynchronous path", func() {
			config := newTestConfig(app.ID, "cfg-on-deploy-create", []string{environment.Name}, nil)

			mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				// on_deploy 配置在部署前无需下发，创建不会触碰集群，也就不会失败
				Expect(service.Create(ctx, app, config, false)).To(Succeed())
				Consistently(func(g Gomega) {
					stored, getErr := store.Get(ctx, app.ID, config.Name)
					g.Expect(getErr).NotTo(HaveOccurred())
					g.Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
				}).WithTimeout(200 * time.Millisecond).Should(Succeed())
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
