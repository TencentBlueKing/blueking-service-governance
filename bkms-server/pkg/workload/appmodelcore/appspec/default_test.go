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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Default AppSpec", func() {
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

	It("lazily creates the default spec from AppModel", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Replicas: lo.ToPtr(int32(3)),
			UpdateStrategy: &appmodel.UpdateStrategy{
				MaxUnavailable: lo.ToPtr("25%"),
				MaxSurge:       lo.ToPtr("0"),
			},
			Workload: appmodel.Workload{
				Resources: map[string]string{
					"cpu":    "100m-200m",
					"memory": "256Mi",
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		spec, err := GetDefault(ctx, appSpecStore, appModelStore, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(*spec.Resources.Replicas).To(Equal(int32(3)))
		Expect(*spec.Resources.CPURequests).To(Equal("100m"))
		Expect(*spec.Resources.CPULimits).To(Equal("200m"))
		Expect(*spec.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*spec.Resources.MemoryLimits).To(Equal("256Mi"))
		Expect(*spec.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*spec.UpdateStrategy.MaxSurge).To(Equal("0"))
		Expect(spec.DevMode).To(BeNil())

		saved, err := appSpecStore.Get(ctx, app.ID, DefaultEnvName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.Replicas).To(Equal(int32(3)))
	})

	It("sets the default spec and syncs AppModel-backed sections", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID: app.ID,
			UpdateStrategy: &appmodel.UpdateStrategy{
				Type: "RollingUpdate",
			},
			Workload: appmodel.Workload{},
		})
		Expect(err).NotTo(HaveOccurred())

		err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
			Resources: &ResourcesSpec{
				Replicas:       lo.ToPtr(int32(6)),
				CPURequests:    lo.ToPtr("300m"),
				CPULimits:      lo.ToPtr("500m"),
				MemoryRequests: lo.ToPtr("256Mi"),
			},
			UpdateStrategy: &UpdateStrategySpec{
				MaxUnavailable: lo.ToPtr("10%"),
				MaxSurge:       lo.ToPtr("2"),
			},
			DevMode: &DevModeSpec{
				Enabled: lo.ToPtr(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		savedSpec, err := appSpecStore.Get(ctx, app.ID, DefaultEnvName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*savedSpec.Resources.Replicas).To(Equal(int32(6)))
		Expect(*savedSpec.UpdateStrategy.MaxUnavailable).To(Equal("10%"))
		Expect(*savedSpec.UpdateStrategy.MaxSurge).To(Equal("2"))
		Expect(savedSpec.DevMode).NotTo(BeNil())
		Expect(*savedSpec.DevMode.Enabled).To(BeTrue())
		Expect(savedSpec.DevMode.WorkPath).To(BeNil())
		Expect(savedSpec.DevMode.MountPath).To(BeNil())

		savedModel, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(*savedModel.Replicas).To(Equal(int32(6)))
		Expect(savedModel.Workload.Resources).To(Equal(map[string]string{
			"cpu":    "300m-500m",
			"memory": "256Mi-256Mi",
		}))
		Expect(savedModel.UpdateStrategy.Type).To(Equal("RollingUpdate"))
		Expect(*savedModel.UpdateStrategy.MaxUnavailable).To(Equal("10%"))
		Expect(*savedModel.UpdateStrategy.MaxSurge).To(Equal("2"))
	})

	It("returns a validation sentinel on invalid default spec", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Workload: appmodel.Workload{},
		})
		Expect(err).NotTo(HaveOccurred())

		err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
			DevMode: &DevModeSpec{
				Enabled:  lo.ToPtr(true),
				WorkPath: lo.ToPtr("/tmp/dev-mode"),
			},
		})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, ErrAppSpecValidation)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("WorkPath"))
	})

	It("rejects CPU requests larger than limits", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Workload: appmodel.Workload{},
		})
		Expect(err).NotTo(HaveOccurred())

		err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
			Resources: &ResourcesSpec{
				CPURequests: lo.ToPtr("1500m"),
				CPULimits:   lo.ToPtr("1200m"),
			},
		})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, ErrAppSpecValidation)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("CPURequests"))
		Expect(err.Error()).To(ContainSubstring("resource_request_lte_limit"))
	})

	It("sets the default spec with partial values", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID: app.ID,
			UpdateStrategy: &appmodel.UpdateStrategy{
				Type: "RollingUpdate",
			},
			Workload: appmodel.Workload{
				Resources: map[string]string{
					"cpu":    "100m-200m",
					"memory": "256Mi",
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
			UpdateStrategy: &UpdateStrategySpec{
				MaxUnavailable: lo.ToPtr("35%"),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		spec, err := GetDefault(ctx, appSpecStore, appModelStore, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(*spec.UpdateStrategy.MaxUnavailable).To(Equal("35%"))
		// SetDefault 默认就是全量写入，其他未提供值的将被复位为 nil
		Expect(spec.UpdateStrategy.MaxSurge).To(BeNil())
		Expect(spec.Resources).To(BeNil())

		savedModel, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(savedModel.Replicas).To(BeNil())
		Expect(savedModel.UpdateStrategy.Type).To(Equal("RollingUpdate"))
		Expect(*savedModel.UpdateStrategy.MaxUnavailable).To(Equal("35%"))
		Expect(savedModel.UpdateStrategy.MaxSurge).To(BeNil())
		Expect(savedModel.Workload.Resources).To(BeNil())
	})
})
