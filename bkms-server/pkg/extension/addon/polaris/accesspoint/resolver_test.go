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

package accesspoint_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
)

var _ = Describe("Resolver", func() {
	var (
		ctx         context.Context
		resolver    accesspoint.Resolver
		apStore     *accesspoint.AccessPointStore
		bindStore   *accesspoint.BindingStore
		svcStore    *depsvcmodel.ServiceStore
		workspaceID string
	)

	// createAccessPoint 在 wsID 下创建一个自建接入点
	createAccessPoint := func(wsID string) *accesspoint.PolarisAccessPoint {
		p := &accesspoint.PolarisAccessPoint{
			WorkspaceID: wsID,
			Name:        "tse-" + stringx.Random(6),
			ConsoleURL:  "http://console.example.com",
			ClientHosts: []string{"10.0.0.1", "polaris.example.com"},
			Token:       "secret-token",
		}
		Expect(apStore.Create(ctx, p)).To(Succeed())
		return p
	}

	bindCluster := func(wsID, clusterID, accessPointID string) {
		Expect(bindStore.Create(ctx, &accesspoint.PolarisClusterBinding{
			WorkspaceID:   wsID,
			ClusterID:     clusterID,
			AccessPointID: accessPointID,
		})).To(Succeed())
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		app := fxtest.New(GinkgoT(), accesspoint.FxModule, fx.Populate(&resolver))
		app.RequireStart()
		DeferCleanup(app.RequireStop)

		apStore = testutil.NewStore(GinkgoT(), accesspoint.AccessPointMeta)
		bindStore = testutil.NewStore(GinkgoT(), accesspoint.BindingMeta)
		svcStore = testutil.NewStore(GinkgoT(), depsvcmodel.ServiceMeta)
		workspaceID = "ws-" + stringx.Random(8)

		setPolarisConfig(config.PolarisConfig{
			Address:                    "10.1.0.1:8091, 10.1.0.2:8091",
			AllowWorkspaceAccessPoints: false,
		})
		_, err := svcStore.DeleteMany(ctx, depsvcmodel.ByName("polaris"))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_, err := svcStore.DeleteMany(ctx, depsvcmodel.ByName("polaris"))
		Expect(err).NotTo(HaveOccurred())
		Expect(testutil.CleanupMetaColl(accesspoint.AccessPointMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(accesspoint.BindingMeta)).To(Succeed())
	})

	Context("ResolveByID", func() {
		It("should assemble the platform access point from config and the polaris dep service", func() {
			Expect(svcStore.Create(ctx, &depsvcmodel.Service{
				Name: "polaris",
				Plans: []depsvcmodel.ServicePlan{{
					Name:         "default",
					ProviderType: depsvcmodel.ProviderTypeSystemAllocated,
					Config:       map[string]any{"baseUrl": "http://polaris-openapi.example.com"},
				}},
			})).To(Succeed())

			ap, err := resolver.ResolveByID(ctx, workspaceID, accesspoint.PlatformID)

			Expect(err).NotTo(HaveOccurred())
			Expect(ap).To(Equal(&accesspoint.AccessPoint{
				ID:             accesspoint.PlatformID,
				Kind:           accesspoint.KindPlatform,
				Name:           accesspoint.PlatformName,
				OpenAPIBaseURL: "http://polaris-openapi.example.com",
				SDKAddresses:   []string{"10.1.0.1:8091", "10.1.0.2:8091"},
				Capabilities:   accesspoint.CapabilitiesOf(accesspoint.KindPlatform),
			}))
		})

		It("should leave the platform OpenAPI address empty when the polaris dep service is absent", func() {
			setPolarisConfig(config.PolarisConfig{JoinPoint: "default", AllowWorkspaceAccessPoints: false})

			ap, err := resolver.ResolveByID(ctx, workspaceID, accesspoint.PlatformID)

			Expect(err).NotTo(HaveOccurred())
			Expect(ap.OpenAPIBaseURL).To(BeEmpty())
			Expect(ap.JoinPoint).To(Equal("default"))
			Expect(ap.SDKAddresses).To(BeEmpty())
		})

		It("should resolve a workspace access point with the decrypted token", func() {
			p := createAccessPoint(workspaceID)

			ap, err := resolver.ResolveByID(ctx, workspaceID, p.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			Expect(ap).To(Equal(&accesspoint.AccessPoint{
				ID:             p.ID.Hex(),
				Kind:           accesspoint.KindWorkspace,
				WorkspaceID:    workspaceID,
				Name:           p.Name,
				ConsoleURL:     "http://console.example.com",
				OpenAPIBaseURL: "http://10.0.0.1:8090",
				SDKAddresses:   []string{"10.0.0.1:8091", "polaris.example.com:8091"},
				ClientHosts:    []string{"10.0.0.1", "polaris.example.com"},
				Token:          "secret-token",
				Capabilities:   accesspoint.CapabilitiesOf(accesspoint.KindWorkspace),
			}))
		})

		It("should not resolve an access point of another workspace", func() {
			p := createAccessPoint("ws-other-" + stringx.Random(6))

			_, err := resolver.ResolveByID(ctx, workspaceID, p.ID.Hex())

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})

		It("should return not found for a malformed id", func() {
			_, err := resolver.ResolveByID(ctx, workspaceID, "not-an-object-id")

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})

		It("should return not found for the platform access point when workspace access points are allowed", func() {
			setPolarisConfig(config.PolarisConfig{Address: "10.1.0.1:8091", AllowWorkspaceAccessPoints: true})

			_, err := resolver.ResolveByID(ctx, workspaceID, accesspoint.PlatformID)

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})

	Context("ResolveByCluster", func() {
		BeforeEach(func() {
			setPolarisConfig(config.PolarisConfig{Address: "10.1.0.1:8091", AllowWorkspaceAccessPoints: true})
		})

		It("should ignore bindings and return the platform when workspace access points are disabled", func() {
			setPolarisConfig(config.PolarisConfig{AllowWorkspaceAccessPoints: false})
			p := createAccessPoint(workspaceID)
			bindCluster(workspaceID, "BCS-K8S-00001", p.ID.Hex())

			ap, err := resolver.ResolveByCluster(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).NotTo(HaveOccurred())
			Expect(ap.ID).To(Equal(accesspoint.PlatformID))
		})

		It("should return ErrClusterNotBound without platform fallback when the cluster is not bound", func() {
			_, err := resolver.ResolveByCluster(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).To(MatchError(accesspoint.ErrClusterNotBound))
		})

		It("should not use a binding of the same cluster in another workspace", func() {
			p := createAccessPoint("ws-other-" + stringx.Random(6))
			bindCluster(p.WorkspaceID, "BCS-K8S-00001", p.ID.Hex())

			_, err := resolver.ResolveByCluster(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).To(MatchError(accesspoint.ErrClusterNotBound))
		})

		It("should resolve the workspace access point bound to the cluster", func() {
			p := createAccessPoint(workspaceID)
			bindCluster(workspaceID, "BCS-K8S-00001", p.ID.Hex())

			ap, err := resolver.ResolveByCluster(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).NotTo(HaveOccurred())
			Expect(ap.ID).To(Equal(p.ID.Hex()))
			Expect(ap.Kind).To(Equal(accesspoint.KindWorkspace))
		})

		It("should fail when the bound access point has been deleted", func() {
			bindCluster(workspaceID, "BCS-K8S-00001", bson.NewObjectID().Hex())

			_, err := resolver.ResolveByCluster(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})

	Context("ResolveForPolarisConfig", func() {
		var app *bkmsapp.Application

		BeforeEach(func() {
			appStore := testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
			app = dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
			DeferCleanup(func() {
				_, err := appStore.DeleteMany(ctx, bkmsapp.ByID(app.ID))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should treat a legacy config without access point as platform", func() {
			// 存量配置只来自内部版，内部版不允许空间自建接入点（BeforeEach 的默认配置）
			cfg := &polaris.PolarisConfig{AppID: app.ID}

			ap, err := resolver.ResolveForPolarisConfig(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(ap.ID).To(Equal(accesspoint.PlatformID))
			Expect(ap.Kind).To(Equal(accesspoint.KindPlatform))
		})

		It("should resolve the access point fixed on the config", func() {
			p := createAccessPoint(workspaceID)
			cfg := &polaris.PolarisConfig{AppID: app.ID, AccessPointID: p.ID.Hex()}

			ap, err := resolver.ResolveForPolarisConfig(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(ap.ID).To(Equal(p.ID.Hex()))
			Expect(ap.Token).To(Equal("secret-token"))
		})

		It("should not resolve an access point outside the app's workspace", func() {
			p := createAccessPoint("ws-other-" + stringx.Random(6))
			cfg := &polaris.PolarisConfig{AppID: app.ID, AccessPointID: p.ID.Hex()}

			_, err := resolver.ResolveForPolarisConfig(ctx, cfg)

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})
})
