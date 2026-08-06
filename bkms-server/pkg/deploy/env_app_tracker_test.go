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

package deploy

import (
	"context"
	"fmt"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
)

var _ = Describe("isEmptyOrBaselineLane", func() {
	var (
		ctx         context.Context
		workspaceID string
		envName     string
	)

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "test-workspace"
		envName = "staging"
	})

	It("should return true when traffic lane name is empty", func() {
		result := isEmptyOrBaselineLane(ctx, workspaceID, envName, "")
		Expect(result).To(BeTrue())
	})

	It("should return true when traffic lane is baseline lane", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockTrafficManagerAndGetBaseline("baseline")

			result := isEmptyOrBaselineLane(ctx, workspaceID, envName, "baseline")
			Expect(result).To(BeTrue())
		})
	})

	It("should return false when traffic lane is not baseline lane", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockTrafficManagerAndGetBaseline("baseline")

			result := isEmptyOrBaselineLane(ctx, workspaceID, envName, "feature-lane")
			Expect(result).To(BeFalse())
		})
	})

	It("should return false when get baseline lane fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockClient := &trafficmanager.StubTrafficManager{}
			mockey.Mock(trafficmanager.New).Return(mockClient).Build()
			mockey.Mock((*trafficmanager.StubTrafficManager).GetBaselineTrafficLane).
				Return(nil, fmt.Errorf("rpc error")).Build()

			result := isEmptyOrBaselineLane(ctx, workspaceID, envName, "some-lane")
			Expect(result).To(BeFalse())
		})
	})

	It("should return false when baseline lane is nil", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockClient := &trafficmanager.StubTrafficManager{}
			mockey.Mock(trafficmanager.New).Return(mockClient).Build()
			mockey.Mock((*trafficmanager.StubTrafficManager).GetBaselineTrafficLane).
				Return((*trafficmanager.TrafficLane)(nil), nil).Build()

			result := isEmptyOrBaselineLane(ctx, workspaceID, envName, "some-lane")
			Expect(result).To(BeFalse())
		})
	})
})

var _ = Describe("TrackEnvAddApp", func() {
	var (
		ctx      context.Context
		diApp    *fxtest.App
		envStore envmodel.EnvironmentStore
		envSvc   *bkmsenv.EnvService
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsenv.FxModule,
			fx.Populate(
				&envStore,
				&envSvc,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	It("should add app to env idempotently", func() {
		env := dbfactory.Env(ctx, envSvc, "workspace-track-add")
		appID := "app-track-add"

		TrackEnvAddApp(ctx, envStore, env.WorkspaceID, env.Name, appID)
		TrackEnvAddApp(ctx, envStore, env.WorkspaceID, env.Name, appID)

		got, err := envStore.Get(ctx, env.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.AppIDs).To(Equal([]string{appID}))
	})
})
