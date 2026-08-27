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

package hostport_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
)

var _ = Describe("EnvStateManager", func() {
	var (
		ctx        context.Context
		diApp      *fxtest.App
		appStore   bkmsapp.ApplicationStore
		envService *env.EnvService
		store      hostport.HostPortStore
		manager    *hostport.EnvStateManager
		app        *bkmsapp.Application
		federation *envmodel.Environment
		nonFedEnv  *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		suffix := stringx.Random(6)
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			hostport.FxModule,
			fx.Populate(&appStore, &envService, &store, &manager),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		federation = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: app.WorkspaceID,
			AppIDs:      []string{app.ID},
			Cluster: &envmodel.BizCluster{
				ProjectCode:  "proj",
				ClusterID:    "BCS-FED-1",
				ClusterType:  "single",
				Namespace:    "ns-fed-" + suffix,
				IsFederation: true,
			},
		})
		nonFedEnv = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: app.WorkspaceID,
			AppIDs:      []string{app.ID},
			Cluster: &envmodel.BizCluster{
				ProjectCode: "proj",
				ClusterID:   "BCS-SINGLE-1",
				ClusterType: "single",
				Namespace:   "ns-single-" + suffix,
			},
		})
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	Describe("ReconcileAfterDeploy", func() {
		It("is a no-op for non-federation environments", func() {
			Expect(manager.ReconcileAfterDeploy(ctx, app, nonFedEnv, []int32{80})).To(Succeed())

			_, err := store.Get(ctx, app.ID)
			Expect(err).To(MatchError(hostport.ErrConfigNotFound))
		})

		It("persists the provided applied ports for federation environments", func() {
			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, []int32{8080, 80})).To(Succeed())

			config, err := store.Get(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.EnvStates[federation.Name].AppliedPorts).To(Equal([]int32{80, 8080}))
		})

		It("records empty applied ports from the deploy snapshot", func() {
			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, []int32{80})).To(Succeed())
			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, nil)).To(Succeed())

			config, err := store.Get(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.EnvStates[federation.Name].AppliedPorts).To(Equal([]int32{}))
		})

		It("does not re-list declared ports (avoids mid-deploy race)", func() {
			_, err := store.ReplacePorts(ctx, app.ID, []int32{80})
			Expect(err).NotTo(HaveOccurred())
			// Snapshot at build/inject time was only 80; declared ports changed after inject.
			_, err = store.ReplacePorts(ctx, app.ID, []int32{80, 443})
			Expect(err).NotTo(HaveOccurred())

			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, []int32{80})).To(Succeed())

			config, err := store.Get(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.EnvStates[federation.Name].AppliedPorts).To(Equal([]int32{80}))
		})
	})

	Describe("ReconcileAfterUninstall", func() {
		It("removes the env snapshot for federation environments", func() {
			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, []int32{80})).To(Succeed())
			Expect(manager.ReconcileAfterUninstall(ctx, app, federation)).To(Succeed())

			config, err := store.Get(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			_, ok := config.EnvStates[federation.Name]
			Expect(ok).To(BeFalse())
		})

		It("is a no-op for non-federation environments", func() {
			Expect(manager.ReconcileAfterDeploy(ctx, app, federation, []int32{80})).To(Succeed())
			Expect(manager.ReconcileAfterUninstall(ctx, app, nonFedEnv)).To(Succeed())

			config, err := store.Get(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			_, ok := config.EnvStates[federation.Name]
			Expect(ok).To(BeTrue())
		})
	})
})
