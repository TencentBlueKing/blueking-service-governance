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

package appspec_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	lifecyclesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/lifecycle"
)

var _ = Describe("AppSpec Sections", func() {
	Context("default scope", func() {
		var ctx context.Context
		var diApp *fxtest.App
		var appStore bkmsapp.ApplicationStore
		var appModelStore appmodel.AppModelStore
		var appSpecStore AppSpecStore

		BeforeEach(func() {
			ctx = context.Background()
			diApp = fxtest.New(
				GinkgoT(),
				bkmsapp.FxModule,
				appmodel.FxModule,
				FxModule,
				fx.Populate(&appStore, &appModelStore, &appSpecStore),
			)
			diApp.RequireStart()
		})

		AfterEach(func() {
			diApp.RequireStop()
		})

		It("patches only the default resources section", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID: app.ID,
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type:           "RollingUpdate",
					MaxUnavailable: lo.ToPtr("25%"),
					MaxSurge:       lo.ToPtr("1"),
				},
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Resources: &ResourcesSpec{
					Replicas:       lo.ToPtr(int32(2)),
					CPURequests:    lo.ToPtr("100m"),
					CPULimits:      lo.ToPtr("200m"),
					MemoryRequests: lo.ToPtr("256Mi"),
					MemoryLimits:   lo.ToPtr("512Mi"),
				},
				UpdateStrategy: &UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("25%"),
					MaxSurge:       lo.ToPtr("1"),
				},
				DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefaultSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				ResourcesSection,
				&ResourcesSpec{MemoryRequests: lo.ToPtr("1Gi")},
				SectionWriteModePatch,
			)
			Expect(err).NotTo(HaveOccurred())

			resourcesSpec, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, ResourcesSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*resourcesSpec.CPURequests).To(Equal("100m"))
			Expect(*resourcesSpec.MemoryRequests).To(Equal("1Gi"))
			Expect(*resourcesSpec.MemoryLimits).To(Equal("512Mi"))

			// Other sections should be unaffected.
			updateStrategy, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, UpdateStrategySection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*updateStrategy.MaxUnavailable).To(Equal("25%"))
			Expect(*updateStrategy.MaxSurge).To(Equal("1"))

			savedModel, err := appModelStore.GetAppModel(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(savedModel.UpdateStrategy.Type).To(Equal("RollingUpdate"))
			Expect(*savedModel.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
			Expect(*savedModel.UpdateStrategy.MaxSurge).To(Equal("1"))
			Expect(savedModel.Workload.Resources).To(Equal(map[string]string{
				"cpu":    "100m-200m",
				"memory": "1Gi-512Mi",
			}))
		})

		It("returns cloned section values instead of internal pointers", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Resources: &ResourcesSpec{
					Replicas:       lo.ToPtr(int32(2)),
					CPURequests:    lo.ToPtr("100m"),
					MemoryRequests: lo.ToPtr("256Mi"),
				},
			})
			Expect(err).NotTo(HaveOccurred())

			resourcesSpec, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, ResourcesSection)
			Expect(err).NotTo(HaveOccurred())

			resourcesSpec.Replicas = lo.ToPtr(int32(9))
			resourcesSpec.MemoryRequests = lo.ToPtr("2Gi")

			reloadedResources, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, ResourcesSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*reloadedResources.Replicas).To(Equal(int32(2)))
			Expect(*reloadedResources.MemoryRequests).To(Equal("256Mi"))
		})

		It("replaces only the default update strategy section", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID: app.ID,
				UpdateStrategy: &appmodel.UpdateStrategy{
					Type:           "RollingUpdate",
					MaxUnavailable: lo.ToPtr("20%"),
					MaxSurge:       lo.ToPtr("2"),
				},
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Resources: &ResourcesSpec{
					Replicas: lo.ToPtr(int32(3)),
				},
				UpdateStrategy: &UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("20%"),
					MaxSurge:       lo.ToPtr("2"),
				},
				DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefaultSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				UpdateStrategySection,
				&UpdateStrategySpec{MaxUnavailable: lo.ToPtr("35%")},
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			updateStrategy, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, UpdateStrategySection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*updateStrategy.MaxUnavailable).To(Equal("35%"))
			Expect(updateStrategy.MaxSurge).To(BeNil())

			savedModel, err := appModelStore.GetAppModel(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(savedModel.UpdateStrategy.Type).To(Equal("RollingUpdate"))
			Expect(*savedModel.UpdateStrategy.MaxUnavailable).To(Equal("35%"))
			Expect(savedModel.UpdateStrategy.MaxSurge).To(BeNil())
		})

		It("patches dev mode without overwriting default paths", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefaultSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				DevModeSection,
				&DevModeSpec{Enabled: lo.ToPtr(false)},
				SectionWriteModePatch,
			)
			Expect(err).NotTo(HaveOccurred())

			devModeSpec, err := GetDefaultSection(ctx, appSpecStore, appModelStore, app.ID, DevModeSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*devModeSpec.Enabled).To(BeFalse())
			Expect(devModeSpec.WorkPath).To(BeNil())
			Expect(devModeSpec.MountPath).To(BeNil())
		})
	})

	Context("env scope", func() {
		var ctx context.Context
		var diApp *fxtest.App
		var appStore bkmsapp.ApplicationStore
		var appModelStore appmodel.AppModelStore
		var appSpecStore AppSpecStore

		BeforeEach(func() {
			ctx = context.Background()
			diApp = fxtest.New(
				GinkgoT(),
				bkmsapp.FxModule,
				appmodel.FxModule,
				FxModule,
				fx.Populate(&appStore, &appModelStore, &appSpecStore),
			)
			diApp.RequireStart()
		})

		AfterEach(func() {
			diApp.RequireStop()
		})

		It("patches only one env section and preserves the others", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Resources: &ResourcesSpec{
					Replicas:    lo.ToPtr(int32(2)),
					CPURequests: lo.ToPtr("100m"),
					CPULimits:   lo.ToPtr("200m"),
				},
				UpdateStrategy: &UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("25%"),
					MaxSurge:       lo.ToPtr("1"),
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnv(ctx, appSpecStore, &AppSpec{
				AppID:   app.ID,
				EnvName: "stag",
				Resources: &ResourcesSpec{
					Replicas:       lo.ToPtr(int32(5)),
					MemoryRequests: lo.ToPtr("1Gi"),
					MemoryLimits:   lo.ToPtr("2Gi"),
				},
				UpdateStrategy: &UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("10%"),
					MaxSurge:       lo.ToPtr("3"),
				},
				DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				DevModeSection,
				&DevModeSpec{Enabled: lo.ToPtr(false)},
				SectionWriteModePatch,
			)
			Expect(err).NotTo(HaveOccurred())

			rawDevMode, err := GetEnvSection(ctx, appSpecStore, app.ID, "stag", DevModeSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*rawDevMode.Enabled).To(BeFalse())
			Expect(rawDevMode.WorkPath).To(BeNil())
			Expect(rawDevMode.MountPath).To(BeNil())

			effectiveResources, err := GetEnvEffectiveSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				"stag",
				ResourcesSection,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(*effectiveResources.Replicas).To(Equal(int32(5)))
			Expect(*effectiveResources.CPURequests).To(Equal("100m"))
			Expect(*effectiveResources.MemoryRequests).To(Equal("1Gi"))
		})

		It("creates a new env override from one section write", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"prod",
				ResourcesSection,
				&ResourcesSpec{Replicas: lo.ToPtr(int32(4))},
				SectionWriteModePatch,
			)
			Expect(err).NotTo(HaveOccurred())

			rawResources, err := GetEnvSection(ctx, appSpecStore, app.ID, "prod", ResourcesSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*rawResources.Replicas).To(Equal(int32(4)))

			rawUpdateStrategy, err := GetEnvSection(ctx, appSpecStore, app.ID, "prod", UpdateStrategySection)
			Expect(err).NotTo(HaveOccurred())
			Expect(rawUpdateStrategy).To(BeNil())
		})

		It("replaces only the target env section", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnv(ctx, appSpecStore, &AppSpec{
				AppID:   app.ID,
				EnvName: "stag",
				UpdateStrategy: &UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("20%"),
					MaxSurge:       lo.ToPtr("2"),
				},
				DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				UpdateStrategySection,
				&UpdateStrategySpec{MaxUnavailable: lo.ToPtr("30%")},
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			rawUpdateStrategy, err := GetEnvSection(ctx, appSpecStore, app.ID, "stag", UpdateStrategySection)
			Expect(err).NotTo(HaveOccurred())
			Expect(*rawUpdateStrategy.MaxUnavailable).To(Equal("30%"))
			Expect(rawUpdateStrategy.MaxSurge).To(BeNil())
		})

		It("removes env resources override and falls back to default values", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Resources: &ResourcesSpec{Replicas: lo.ToPtr(int32(2)), CPURequests: lo.ToPtr("100m")},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				ResourcesSection,
				&ResourcesSpec{Replicas: lo.ToPtr(int32(5)), CPURequests: lo.ToPtr("300m")},
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			effectiveResources, err := GetEnvEffectiveSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				"stag",
				ResourcesSection,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(*effectiveResources.Replicas).To(Equal(int32(5)))
			Expect(*effectiveResources.CPURequests).To(Equal("300m"))

			// Delete the env override by replacing with nil.
			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				ResourcesSection,
				nil,
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			rawResources, err := GetEnvSection(ctx, appSpecStore, app.ID, "stag", ResourcesSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(rawResources).To(BeNil())

			effectiveResources, err = GetEnvEffectiveSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				"stag",
				ResourcesSection,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(*effectiveResources.Replicas).To(Equal(int32(2)))
			Expect(*effectiveResources.CPURequests).To(Equal("100m"))
		})

		It("keeps an empty env lifecycle override instead of falling back to default values", func() {
			app := dbfactory.Application(ctx, appStore)
			err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
				AppID:    app.ID,
				Workload: appmodel.Workload{},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
				Lifecycle: &LifecycleSpec{
					PostStart: &lifecyclesection.Handler{
						Type:         appmodel.LifecycleTypeExec,
						SleepSeconds: lo.ToPtr(int64(10)),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				LifecycleSection,
				&LifecycleSpec{},
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			rawLifecycle, err := GetEnvSection(ctx, appSpecStore, app.ID, "stag", LifecycleSection)
			Expect(err).NotTo(HaveOccurred())
			Expect(rawLifecycle).NotTo(BeNil())
			Expect(rawLifecycle.PostStart).To(BeNil())
			Expect(rawLifecycle.PreStop).To(BeNil())
			Expect(rawLifecycle.TerminationGracePeriodSeconds).To(BeNil())

			effectiveLifecycle, err := GetEnvEffectiveSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				"stag",
				LifecycleSection,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(effectiveLifecycle).NotTo(BeNil())
			Expect(effectiveLifecycle.PostStart).To(BeNil())

			err = SetEnvSection(
				ctx,
				appSpecStore,
				app.ID,
				"stag",
				LifecycleSection,
				nil,
				SectionWriteModeReplace,
			)
			Expect(err).NotTo(HaveOccurred())

			effectiveLifecycle, err = GetEnvEffectiveSection(
				ctx,
				appSpecStore,
				appModelStore,
				app.ID,
				"stag",
				LifecycleSection,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(effectiveLifecycle.PostStart).NotTo(BeNil())
			Expect(*effectiveLifecycle.PostStart.SleepSeconds).To(Equal(int64(10)))
		})
	})
})
