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

var _ = Describe("Env AppSpec", func() {
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

	It("merges env spec into the default spec", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Replicas: lo.ToPtr(int32(2)),
			UpdateStrategy: &appmodel.UpdateStrategy{
				MaxUnavailable: lo.ToPtr("25%"),
				MaxSurge:       lo.ToPtr("1"),
			},
			Workload: appmodel.Workload{
				Resources: map[string]string{
					"cpu":    "100m-200m",
					"memory": "256Mi-512Mi",
				},
			},
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

		err = SetEnv(ctx, appSpecStore, &AppSpec{
			AppID:   app.ID,
			EnvName: "stag",
			Resources: &ResourcesSpec{
				Replicas:       lo.ToPtr(int32(5)),
				MemoryRequests: lo.ToPtr("1Gi"),
				MemoryLimits:   lo.ToPtr("2Gi"),
			},
			UpdateStrategy: &UpdateStrategySpec{
				MaxSurge: lo.ToPtr("3"),
			},
			DevMode: &DevModeSpec{
				Enabled: lo.ToPtr(false),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		effective, err := GetEnvEffective(ctx, appSpecStore, appModelStore, app.ID, "stag")
		Expect(err).NotTo(HaveOccurred())
		Expect(*effective.Resources.Replicas).To(Equal(int32(5)))
		Expect(*effective.Resources.CPURequests).To(Equal("100m"))
		Expect(*effective.Resources.CPULimits).To(Equal("200m"))
		Expect(*effective.Resources.MemoryRequests).To(Equal("1Gi"))
		Expect(*effective.Resources.MemoryLimits).To(Equal("2Gi"))
		Expect(*effective.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*effective.UpdateStrategy.MaxSurge).To(Equal("3"))
		Expect(effective.DevMode).NotTo(BeNil())
		Expect(*effective.DevMode.Enabled).To(BeFalse())
		Expect(effective.DevMode.WorkPath).To(BeNil())
		Expect(effective.DevMode.MountPath).To(BeNil())
	})

	It("falls back to the default spec when env override is missing", func() {
		app := dbfactory.Application(ctx, appStore)
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID:    app.ID,
			Replicas: lo.ToPtr(int32(4)),
			Workload: appmodel.Workload{
				Resources: map[string]string{
					"cpu": "250m-500m",
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		err = SetDefault(ctx, appSpecStore, appModelStore, app.ID, &AppSpec{
			Resources: &ResourcesSpec{
				Replicas:    lo.ToPtr(int32(4)),
				CPURequests: lo.ToPtr("250m"),
				CPULimits:   lo.ToPtr("500m"),
			},
			DevMode: &DevModeSpec{Enabled: lo.ToPtr(true)},
		})
		Expect(err).NotTo(HaveOccurred())

		effective, err := GetEnvEffective(ctx, appSpecStore, appModelStore, app.ID, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(effective.EnvName).To(Equal("prod"))
		Expect(*effective.Resources.Replicas).To(Equal(int32(4)))
		Expect(*effective.Resources.CPURequests).To(Equal("250m"))
		Expect(*effective.Resources.CPULimits).To(Equal("500m"))
		Expect(effective.DevMode).NotTo(BeNil())
		Expect(*effective.DevMode.Enabled).To(BeTrue())
	})

	It("returns a validation sentinel on invalid env spec", func() {
		err := SetEnv(context.Background(), nil, &AppSpec{
			AppID:   "app-id",
			EnvName: "stag",
			Resources: &ResourcesSpec{
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("1Gi"),
			},
		})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, ErrAppSpecValidation)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("resource_request_lte_limit"))
	})
})
