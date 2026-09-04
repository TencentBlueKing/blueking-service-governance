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

var _ = Describe("Service", func() {
	var (
		ctx        context.Context
		diApp      *fxtest.App
		appStore   bkmsapp.ApplicationStore
		envService *env.EnvService
		store      hostport.HostPortStore
		manager    *hostport.EnvStateManager
		svc        *hostport.Service
		app        *bkmsapp.Application
		fedEnv     *envmodel.Environment
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
			fx.Provide(hostport.NewService),
			fx.Populate(&appStore, &envService, &store, &manager, &svc),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		fedEnv = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
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

	Describe("ListPorts", func() {
		It("returns empty ports when no config exists", func() {
			ports, err := svc.ListPorts(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{}))
		})
	})

	Describe("ReplacePorts", func() {
		It("replaces ports through the service layer", func() {
			ports, err := svc.ReplacePorts(ctx, app.ID, []int32{8080, 80})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{80, 8080}))

			ports, err = svc.ReplacePorts(ctx, app.ID, []int32{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{}))
		})
	})

	Describe("GetHostPorts", func() {
		It("returns ports with federated env states only", func() {
			_, err := svc.ReplacePorts(ctx, app.ID, []int32{8080, 80})
			Expect(err).NotTo(HaveOccurred())

			result, err := svc.GetHostPorts(ctx, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Ports).To(Equal([]int32{80, 8080}))
			Expect(result.EnvStates).To(HaveKey(fedEnv.Name))
			Expect(result.EnvStates).NotTo(HaveKey(nonFedEnv.Name))
			Expect(result.EnvStates[fedEnv.Name].PendingAddPorts).To(Equal([]int32{80, 8080}))
		})
	})

	Describe("ListFederatedEnvStates", func() {
		It("returns empty states without error when config is missing", func() {
			views, err := svc.ListFederatedEnvStates(ctx, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(views).To(HaveKey(fedEnv.Name))
			Expect(views).NotTo(HaveKey(nonFedEnv.Name))
			Expect(views[fedEnv.Name].PendingAddPorts).To(BeEmpty())
			Expect(views[fedEnv.Name].PendingRemovePorts).To(BeEmpty())
			Expect(views[fedEnv.Name].AppliedPorts).To(BeEmpty())
		})

		It("marks newly declared ports as pending add until deploy reconcile", func() {
			_, err := svc.ReplacePorts(ctx, app.ID, []int32{8080})
			Expect(err).NotTo(HaveOccurred())

			views, err := svc.ListFederatedEnvStates(ctx, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(views[fedEnv.Name].PendingAddPorts).To(Equal([]int32{8080}))
			Expect(views[fedEnv.Name].PendingRemovePorts).To(BeEmpty())
		})

		It("computes pending add and remove against applied snapshot", func() {
			_, err := svc.ReplacePorts(ctx, app.ID, []int32{80})
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.ReconcileAfterDeploy(ctx, app, fedEnv, []int32{80})).To(Succeed())

			_, err = svc.ReplacePorts(ctx, app.ID, []int32{443})
			Expect(err).NotTo(HaveOccurred())

			views, err := svc.ListFederatedEnvStates(ctx, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(views[fedEnv.Name].AppliedPorts).To(Equal([]int32{80}))
			Expect(views[fedEnv.Name].PendingAddPorts).To(Equal([]int32{443}))
			Expect(views[fedEnv.Name].PendingRemovePorts).To(Equal([]int32{80}))
		})
	})
})
