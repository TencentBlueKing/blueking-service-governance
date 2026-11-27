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

package binding_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint/binding"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	bcsProjectID = "bcs-project-id"
	clusterA     = "BCS-K8S-00001"
	clusterB     = "BCS-K8S-00002"
	clusterC     = "BCS-K8S-00003"
	sharedID     = "BCS-K8S-00004"
)

var _ = Describe("Service", func() {
	var (
		ctx       context.Context
		db        *mongo.Database
		svc       *binding.Service
		envSvc    *env.EnvService
		apStore   *accesspoint.AccessPointStore
		bindStore *accesspoint.BindingStore
		cfgStore  *polaris.ConfigStore
		ws        *bkmsworkspace.Workspace
		app       *bkmsapp.Application

		// bcsClusters 模拟 BCS 返回的项目集群列表；bcsErr 非空时模拟查询失败；bcsQueried 记录查询过的项目 ID
		bcsClusters []bcs.Cluster
		bcsErr      error
		bcsQueried  []string
	)

	// boundClusters 返回绑定到接入点的集群
	boundClusters := func(p *accesspoint.PolarisAccessPoint) []string {
		bindings, err := bindStore.List(ctx, accesspoint.ByAccessPoint(ws.ID, p.ID.Hex()), store.Asc("clusterID"))
		Expect(err).NotTo(HaveOccurred())
		return lo.Map(bindings, func(b accesspoint.PolarisClusterBinding, _ int) string { return b.ClusterID })
	}

	createEnv := func(clusterID string) *envmodel.Environment {
		return dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
			WorkspaceID: ws.ID,
			Cluster:     &envmodel.BizCluster{ClusterID: clusterID, Namespace: "ns-" + stringx.Random(8)},
		})
	}

	replace := func(p *accesspoint.PolarisAccessPoint, clusterIDs ...string) error {
		_, _, err := svc.ReplaceClusters(ctx, ws, p.ID.Hex(), clusterIDs)
		return err
	}

	BeforeEach(func() {
		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "ws-admin"})
		diApp := fxtest.New(GinkgoT(), binding.FxModule, env.FxModule, fx.Populate(&svc, &envSvc))
		diApp.RequireStart()
		DeferCleanup(diApp.RequireStop)

		db = testutil.NewDB(GinkgoT())
		apStore = testutil.NewStore(GinkgoT(), accesspoint.AccessPointMeta)
		bindStore = testutil.NewStore(GinkgoT(), accesspoint.BindingMeta)
		cfgStore = testutil.NewStore(GinkgoT(), polaris.ConfigMeta)
		ws = &bkmsworkspace.Workspace{
			ID:        "ws-" + stringx.Random(8),
			BkSystems: bkmsworkspace.BkSystems{BkBCSProjectID: bcsProjectID},
		}
		appStore := testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		app = dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: ws.ID})
		setPolarisConfig(config.PolarisConfig{AllowWorkspaceAccessPoints: true})

		bcsClusters = []bcs.Cluster{
			{ID: clusterA, Name: "cluster-a", Status: "RUNNING"},
			{ID: clusterB, Name: "cluster-b", Status: "RUNNING"},
			{ID: clusterC, Name: "cluster-c", Status: "DELETING"},
			{ID: sharedID, Name: "shared", Status: "RUNNING", IsShared: true},
		}
		bcsErr, bcsQueried = nil, nil
		mockey.Mock(bcs.New).Return(bcs.Client(bcs.NewStub(auth.User{})), nil).Build()
		mockey.Mock((*bcs.StubApiClient).ListClustersByProject).To(
			func(_ *bcs.StubApiClient, _ context.Context, projectID string) ([]bcs.Cluster, error) {
				bcsQueried = append(bcsQueried, projectID)
				if projectID != bcsProjectID {
					return nil, nil
				}
				return bcsClusters, bcsErr
			},
		).Build()

		DeferCleanup(func() {
			_, _ = cfgStore.DeleteMany(ctx, polaris.ByApp(app.ID))
			_, _ = appStore.DeleteMany(ctx, bkmsapp.ByID(app.ID))
			_, _ = store.Of(db, envmodel.EnvMeta).DeleteMany(ctx, envmodel.InWorkspace(ws.ID))
		})
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(testutil.CleanupMetaColl(accesspoint.AccessPointMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(accesspoint.BindingMeta)).To(Succeed())
	})

	Describe("ReplaceClusters", func() {
		It("should bind candidate clusters to an access point", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)

			before, after, err := svc.ReplaceClusters(ctx, ws, p.ID.Hex(), []string{clusterB, clusterA})

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(BeEmpty())
			Expect(after).To(Equal([]string{clusterA, clusterB}))
			Expect(boundClusters(p)).To(Equal([]string{clusterA, clusterB}))
			b, err := bindStore.Get(ctx, accesspoint.ByCluster(ws.ID, clusterA))
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Creator).To(Equal("ws-admin"))
			Expect(b.Updater).To(Equal("ws-admin"))
		})

		It("should reject removing a cluster whose feature env is used by a polaris config", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, p.ID.Hex())
			source := createEnv(clusterA)
			featEnv := dbfactory.FeatEnv(ctx, envSvc, app, source)
			cfg := dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{featEnv},
			})

			err := replace(p, clusterB)

			var inUse *accesspoint.ClusterInUseError
			Expect(errors.As(err, &inUse)).To(BeTrue())
			apiErr := accesspoint.APIError(err, "bind").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(apiErr.Details()).To(Equal(bkerrs.Details{bkerrs.NewDetail(
				bkerrs.ErrDetailCodePolarisClusterInUse,
				apiErr.Error(),
				bkerrs.WithSystem(bkerrs.SystemName),
				bkerrs.WithModule("polaris"),
				bkerrs.WithExtras(map[string]string{
					"clusterIDs": fmt.Sprintf(`[%q]`, clusterA),
					"configs": fmt.Sprintf(`[{"clusterID":%q,"envName":%q,"appID":%q,"configName":%q}]`,
						clusterA, featEnv.Name, app.ID, cfg.Name),
				}),
			)}))
			Expect(boundClusters(p)).To(Equal([]string{clusterA, clusterB}))
		})

		It("should reject rebinding a cluster in use from another access point", func() {
			first := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			second := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, first.ID.Hex())
			dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: first.ID.Hex(), ScopeEnvs: []*envmodel.Environment{createEnv(clusterA)},
			})

			err := replace(second, clusterA)

			Expect(err).To(MatchError(accesspoint.ErrClusterInUse))
			Expect(boundClusters(first)).To(Equal([]string{clusterA}))
			Expect(boundClusters(second)).To(BeEmpty())
		})

		It("should allow keeping a cluster in use while changing other clusters", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, p.ID.Hex())
			dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{createEnv(clusterA)},
			})

			Expect(replace(p, clusterA, clusterC)).To(Succeed())

			Expect(boundClusters(p)).To(Equal([]string{clusterA, clusterC}))
		})

		It("should rebind an unused cluster and change the access point resolved for its envs", func() {
			first := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			second := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, first.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, first.ID.Hex())
			// 环境存在但没有被北极星配置使用
			createEnv(clusterA)

			before, after, err := svc.ReplaceClusters(ctx, ws, second.ID.Hex(), []string{clusterA})

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(BeEmpty())
			Expect(after).To(Equal([]string{clusterA}))
			Expect(boundClusters(first)).To(Equal([]string{clusterB}))
			Expect(boundClusters(second)).To(Equal([]string{clusterA}))
			ids, err := accesspoint.NewResolver(db).ResolveIDsByClusters(ctx, ws.ID, []string{clusterA})
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal(map[string]string{clusterA: second.ID.Hex()}))
		})

		It("should unbind all clusters with an empty list when they are not in use", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, p.ID.Hex())

			before, after, err := svc.ReplaceClusters(ctx, ws, p.ID.Hex(), []string{})

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(Equal([]string{clusterA, clusterB}))
			Expect(after).To(BeEmpty())
			Expect(boundClusters(p)).To(BeEmpty())
			// 没有待绑定的集群时不需要查询 BCS
			Expect(bcsQueried).To(BeEmpty())
		})

		It("should not delete a cluster rebound to another access point between checking and writing", func() {
			first := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			second := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, first.ID.Hex())
			// 占用检查查询北极星配置时，模拟另一个请求把集群换绑到 second
			mockey.Mock(polaris.ListByScopeEnvs).To(
				func(context.Context, *mongo.Database, string, []string) ([]polaris.PolarisConfig, error) {
					err := bindStore.Update(ctx, map[string]any{"accessPointID": second.ID.Hex()},
						accesspoint.ByCluster(ws.ID, clusterA),
					)
					return nil, err
				},
			).Build()

			Expect(replace(first)).To(Succeed())

			Expect(boundClusters(second)).To(Equal([]string{clusterA}))
		})
	})

	Describe("BindCluster", func() {
		It("should bind an unbound cluster with the normalized access point id", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)

			before, after, err := svc.BindCluster(ctx, ws, clusterB, strings.ToUpper(p.ID.Hex()))

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(BeEmpty())
			Expect(after).To(Equal(p.ID.Hex()))
			b, err := bindStore.Get(ctx, accesspoint.ByCluster(ws.ID, clusterB))
			Expect(err).NotTo(HaveOccurred())
			Expect(b.AccessPointID).To(Equal(p.ID.Hex()))
			Expect(b.Creator).To(Equal("ws-admin"))
		})

		It("should rebind an unused cluster without touching other clusters", func() {
			first := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			second := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, first.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, first.ID.Hex())
			createEnv(clusterA)

			before, after, err := svc.BindCluster(ctx, ws, clusterA, second.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(Equal(first.ID.Hex()))
			Expect(after).To(Equal(second.ID.Hex()))
			Expect(boundClusters(first)).To(Equal([]string{clusterB}))
			Expect(boundClusters(second)).To(Equal([]string{clusterA}))
		})

		DescribeTable("should reject changing a cluster whose env is used by a polaris config",
			func(toAnotherAccessPoint bool) {
				p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
				dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
				dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
					AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{createEnv(clusterA)},
				})
				target := ""
				if toAnotherAccessPoint {
					target = dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID).ID.Hex()
				}

				_, _, err := svc.BindCluster(ctx, ws, clusterA, target)

				var inUse *accesspoint.ClusterInUseError
				Expect(errors.As(err, &inUse)).To(BeTrue())
				Expect(inUse.ClusterIDs()).To(Equal([]string{clusterA}))
				Expect(boundClusters(p)).To(Equal([]string{clusterA}))
			},
			Entry("rebind", true),
			Entry("unbind", false),
		)

		It("should unbind an unused cluster with an empty access point id", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, p.ID.Hex())

			before, after, err := svc.BindCluster(ctx, ws, clusterA, "")

			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(Equal(p.ID.Hex()))
			Expect(after).To(BeEmpty())
			Expect(boundClusters(p)).To(Equal([]string{clusterB}))
		})

		It("should not write when the cluster is already in the requested state", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{createEnv(clusterA)},
			})

			before, after, err := svc.BindCluster(ctx, ws, clusterA, p.ID.Hex())
			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(Equal(p.ID.Hex()))
			Expect(after).To(Equal(p.ID.Hex()))
			b, err := bindStore.Get(ctx, accesspoint.ByCluster(ws.ID, clusterA))
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Updater).To(BeEmpty())

			before, after, err = svc.BindCluster(ctx, ws, clusterB, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(BeEmpty())
			Expect(after).To(BeEmpty())
		})

		It("should reject a cluster outside the candidates", func() {
			_, _, err := svc.BindCluster(ctx, ws, sharedID, dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID).ID.Hex())

			Expect(err).To(MatchError(accesspoint.ErrAccessPointInvalid))
			Expect(bindStore.Count(ctx, accesspoint.InWorkspace(ws.ID))).To(BeZero())
		})
	})

	Describe("ListClusters", func() {
		It("should list non-shared clusters with envs first and keep bindings of clusters without envs", func() {
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterC, p.ID.Hex())
			// clusterC 上有源环境与特性环境，特性环境被北极星配置使用
			source := createEnv(clusterC)
			featEnv := dbfactory.FeatEnv(ctx, envSvc, app, source)
			dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{featEnv},
			})
			// clusterB 上有环境但未被使用
			envB := createEnv(clusterB)
			// clusterA 上最后一个环境被删除，集群仍在列表中且绑定不变
			Expect(envSvc.Delete(ctx, createEnv(clusterA).ID)).To(Succeed())

			views, err := svc.ListClusters(ctx, ws)

			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(views, func(v binding.ClusterView, _ int) string { return v.Cluster.ID })).
				To(Equal([]string{clusterB, clusterC, clusterA}))

			Expect(lo.Map(views[0].Envs, func(e envmodel.Environment, _ int) string { return e.Name })).
				To(Equal([]string{envB.Name}))
			Expect(views[0].AccessPoint).To(BeNil())
			Expect(views[0].InUse).To(BeFalse())

			Expect(views[1].Cluster.Status).To(Equal("DELETING"))
			Expect(lo.Map(views[1].Envs, func(e envmodel.Environment, _ int) string { return e.Name })).
				To(ConsistOf(source.Name, featEnv.Name))
			Expect(views[1].AccessPoint.ID).To(Equal(p.ID))
			Expect(views[1].AccessPoint.Name).To(Equal(p.Name))
			Expect(views[1].InUse).To(BeTrue())

			Expect(views[2].Envs).To(BeEmpty())
			Expect(views[2].AccessPoint.ID).To(Equal(p.ID))
			Expect(views[2].InUse).To(BeFalse())
		})

		It("should not count envs of other workspaces", func() {
			otherWS := "ws-other-" + stringx.Random(6)
			dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
				WorkspaceID: otherWS,
				Cluster:     &envmodel.BizCluster{ClusterID: clusterA, Namespace: "ns-" + stringx.Random(8)},
			})
			DeferCleanup(func() {
				_, _ = store.Of(db, envmodel.EnvMeta).DeleteMany(ctx, envmodel.InWorkspace(otherWS))
			})

			views, err := svc.ListClusters(ctx, ws)

			Expect(err).NotTo(HaveOccurred())
			Expect(views).To(HaveLen(3))
			Expect(views).To(HaveEach(HaveField("Envs", BeEmpty())))
		})

		It("should fail instead of returning a partial list when bcs fails", func() {
			bcsErr = errors.New("bcs unavailable")
			p := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())

			_, err := svc.ListClusters(ctx, ws)

			Expect(errors.Is(err, bcsErr)).To(BeTrue())
		})
	})

	Describe("ValidateEnvClusterChange", func() {
		var (
			envA *envmodel.Environment
			p    *accesspoint.PolarisAccessPoint
		)

		BeforeEach(func() {
			p = dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
			dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterA, p.ID.Hex())
			envA = createEnv(clusterA)
		})

		It("should allow any cluster when the env is not used by polaris configs", func() {
			Expect(svc.ValidateEnvClusterChange(ctx, *envA, clusterB)).To(Succeed())
		})

		Context("when the env is used by a polaris config", func() {
			var cfg *polaris.PolarisConfig

			BeforeEach(func() {
				cfg = dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
					AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{envA},
				})
			})

			It("should reject moving to an unbound cluster", func() {
				err := svc.ValidateEnvClusterChange(ctx, *envA, clusterB)

				Expect(err).To(MatchError(accesspoint.ErrClusterNotBound))
				Expect(err.Error()).To(ContainSubstring(app.ID + "/" + cfg.Name))
				apiErr := accesspoint.APIError(err, "update env cluster").(*bkerrs.Error)
				Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			})

			It("should reject moving to a cluster bound to another access point", func() {
				other := dbfactory.PolarisAccessPoint(ctx, apStore, ws.ID)
				dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, other.ID.Hex())

				err := svc.ValidateEnvClusterChange(ctx, *envA, clusterB)

				Expect(err).To(MatchError(accesspoint.ErrAccessPointMismatch))
				apiErr := accesspoint.APIError(err, "update env cluster").(*bkerrs.Error)
				Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			})

			It("should allow moving to a cluster bound to the same access point", func() {
				dbfactory.PolarisClusterBinding(ctx, bindStore, ws.ID, clusterB, p.ID.Hex())

				Expect(svc.ValidateEnvClusterChange(ctx, *envA, clusterB)).To(Succeed())
			})
		})

		It("should skip the check when workspace access points are disabled", func() {
			dbfactory.PolarisConfig(ctx, cfgStore, &dbfactory.PolarisConfigOpts{
				AppID: app.ID, AccessPointID: p.ID.Hex(), ScopeEnvs: []*envmodel.Environment{envA},
			})
			setPolarisConfig(config.PolarisConfig{AllowWorkspaceAccessPoints: false})

			Expect(svc.ValidateEnvClusterChange(ctx, *envA, clusterB)).To(Succeed())
		})
	})
})
