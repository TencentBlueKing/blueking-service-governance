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

package customrole

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// recordingManager 在进程内保存自定义角色，并记录每次重新授权使用的数据
type recordingManager struct {
	perm.StubAllowAnyManager
	updates []bkiam.CustomRoleData
	// updateErr 不为空时，修改直接返回该错误
	updateErr error
}

func (m *recordingManager) UpdateCustomRole(
	ctx context.Context, roleID, displayName, description string, data bkiam.CustomRoleData,
) (*role.Role, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updates = append(m.updates, data)
	return m.StubAllowAnyManager.UpdateCustomRole(ctx, roleID, displayName, description, data)
}

var _ = Describe("custom role scope maintenance", func() {
	var (
		ctx      context.Context
		db       *mongo.Database
		envSvc   *bkmsenv.EnvService
		mgr      *recordingManager
		ws       *workspace.Workspace
		appA     *bkmsapp.Application
		appB     *bkmsapp.Application
		testEnv  *envmodel.Environment
		prodEnv  *envmodel.Environment
		roleData bkiam.CustomRoleData
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		diApp := fxtest.New(GinkgoT(), bkmsenv.FxModule, fx.Populate(&envSvc))
		diApp.RequireStart()
		DeferCleanup(diApp.RequireStop)

		db = testutil.NewDB(GinkgoT())
		ws = dbfactory.Workspace(ctx, testutil.NewStore(GinkgoT(), workspace.Meta))
		apps := testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		appA = dbfactory.ApplicationWithOpts(ctx, apps, &dbfactory.ApplicationOpts{WorkspaceID: ws.ID})
		appB = dbfactory.ApplicationWithOpts(ctx, apps, &dbfactory.ApplicationOpts{WorkspaceID: ws.ID})
		testEnv = dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{WorkspaceID: ws.ID, Type: "test"})
		prodEnv = dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{WorkspaceID: ws.ID, Type: "production"})
		DeferCleanup(func() {
			Expect(testutil.CleanupMetaColl(envmodel.EnvMeta)).To(Succeed())
			Expect(testutil.CleanupMetaColl(bkmsapp.AppMeta)).To(Succeed())
			Expect(testutil.CleanupMetaColl(workspace.Meta)).To(Succeed())
		})

		mgr = &recordingManager{}
		roleData = bkiam.CustomRoleData{
			WorkspaceID: ws.ID,
			Permissions: []role.Permission{
				role.Perm.SpaceView,
				role.Perm.AppView,
				role.Perm.EnvView,
				role.Perm.EnvDeploy,
			},
			AppScope: role.AppScope{Mode: role.AppScopeApps, AppIDs: []string{appA.ID, appB.ID}},
			EnvScope: role.EnvScope{Mode: role.EnvScopeEnvs, EnvIDs: []string{testEnv.ID.Hex(), prodEnv.ID.Hex()}},
		}
	})

	Context("BuildData", func() {
		It("loads names of specified apps and envs with their env types", func() {
			data, err := BuildData(
				ctx,
				db,
				ws,
				[]role.Permission{role.Perm.EnvDeploy},
				roleData.AppScope,
				roleData.EnvScope,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.Permissions).To(ContainElements(role.Perm.SpaceView, role.Perm.AppView, role.Perm.EnvDeploy))
			Expect(data.Apps).To(HaveLen(2))
			Expect(data.Envs).To(ConsistOf(
				bkiam.EnvResourceRef{ID: testEnv.ID.Hex(), Name: testEnv.DisplayName, Type: "test"},
				bkiam.EnvResourceRef{ID: prodEnv.ID.Hex(), Name: prodEnv.DisplayName, Type: "production"},
			))
		})

		It("rejects apps or envs outside the workspace", func() {
			otherEnv := dbfactory.Env(ctx, envSvc, "other-ws")
			_, err := BuildData(ctx, db, ws, nil, roleData.AppScope,
				role.EnvScope{Mode: role.EnvScopeEnvs, EnvIDs: []string{otherEnv.ID.Hex()}})
			Expect(errors.Is(err, role.ErrInvalidCustomRole)).To(BeTrue())

			_, err = BuildData(
				ctx,
				db,
				ws,
				nil,
				role.AppScope{Mode: role.AppScopeApps, AppIDs: []string{"no-such-app"}},
				roleData.EnvScope,
			)
			Expect(errors.Is(err, role.ErrInvalidCustomRole)).To(BeTrue())
		})
	})

	Context("pruning and reapplying", func() {
		var r *role.Role

		BeforeEach(func() {
			var err error
			r, err = mgr.CreateCustomRole(ctx, "partial", "", roleData, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes a deleted app from specified apps and regrants", func() {
			Expect(PruneApp(ctx, db, mgr, ws.ID, appA.ID)).To(Succeed())
			got, err := mgr.GetRole(ctx, ws.ID, r.RoleCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AppScope.AppIDs).To(Equal([]string{appB.ID}))
			Expect(mgr.updates).To(HaveLen(1))
			Expect(mgr.updates[0].Apps).To(HaveLen(1))
			Expect(mgr.updates[0].Apps[0].ID).To(Equal(appB.ID))
		})

		It("leaves roles that do not reference the deleted resource untouched", func() {
			unrelatedEnv := &envmodel.Environment{ID: bson.NewObjectID(), WorkspaceID: ws.ID, Type: "staging"}
			Expect(PruneApp(ctx, db, mgr, ws.ID, "unrelated")).To(Succeed())
			Expect(PruneEnv(ctx, db, mgr, unrelatedEnv)).To(Succeed())
			Expect(mgr.updates).To(BeEmpty())
		})

		It("keeps an emptied scope empty instead of treating it as all", func() {
			Expect(PruneEnv(ctx, db, mgr, testEnv)).To(Succeed())
			Expect(PruneEnv(ctx, db, mgr, prodEnv)).To(Succeed())
			got, err := mgr.GetRole(ctx, ws.ID, r.RoleCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.EnvScope.Mode).To(Equal(role.EnvScopeEnvs))
			Expect(got.EnvScope.EnvIDs).To(BeEmpty())
		})

		It("regrants with the current env type after the type changes", func() {
			Expect(
				envSvc.Update(ctx, testEnv.ID, &envmodel.EnvironmentUpdateData{Type: lo.ToPtr("staging")}),
			).To(Succeed())
			after := *testEnv
			after.Type = "staging"
			Expect(ReapplyEnv(ctx, db, mgr, testEnv, &after)).To(Succeed())
			Expect(mgr.updates).To(HaveLen(1))
			Expect(mgr.updates[0].Envs).To(ContainElement(
				bkiam.EnvResourceRef{ID: testEnv.ID.Hex(), Name: testEnv.DisplayName, Type: "staging"},
			))
		})

		It("returns the error when reapplying a role fails", func() {
			mgr.updateErr = errors.New("iam unavailable")
			Expect(PruneApp(ctx, db, mgr, ws.ID, appA.ID)).NotTo(Succeed())
			Expect(mgr.updates).To(BeEmpty())
		})
	})

	Context("monitor grants", func() {
		var apmTest, apmProd *bkmonitor.ApmInstConfig

		BeforeEach(func() {
			Expect(store.Of(db, workspace.Meta).Update(
				ctx, map[string]any{"bkSystems.bkMonitorProjectID": "100"}, workspace.ByWorkspaceID(ws.ID),
			)).To(Succeed())
			ws.BkSystems.BkMonitorProjectID = "100"
			roleData.Permissions = append(roleData.Permissions, role.Perm.AppObservationView)
			apms := store.Of(db, bkmonitor.ApmInstConfigMeta)
			apmTest = &bkmonitor.ApmInstConfig{
				WorkspaceID: ws.ID, ApmID: 11, Name: "apm-test", Token: "t",
				AssociatedEnvs: []bkmonitor.EnvInfo{{EnvID: testEnv.ID, EnvName: testEnv.Name}},
			}
			apmProd = &bkmonitor.ApmInstConfig{
				WorkspaceID: ws.ID, ApmID: 12, Name: "apm-prod", Token: "t",
				AssociatedEnvs: []bkmonitor.EnvInfo{{EnvID: prodEnv.ID, EnvName: prodEnv.Name}},
			}
			Expect(apms.Create(ctx, apmTest)).To(Succeed())
			Expect(apms.Create(ctx, apmProd)).To(Succeed())
			DeferCleanup(func() {
				Expect(testutil.CleanupMetaColl(bkmonitor.ApmInstConfigMeta)).To(Succeed())
			})
		})

		It("grants the APM apps bound to the envs in scope", func() {
			data, err := BuildData(ctx, db, ws, roleData.Permissions, roleData.AppScope,
				role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(data.BKMonitor).NotTo(BeNil())
			Expect(data.BKMonitor.SpaceID).To(Equal("100"))
			Expect(data.BKMonitor.AllApms).To(BeFalse())
			Expect(data.BKMonitor.Apms).To(ConsistOf(bkiam.ResourceRef{ID: "11", Name: "apm-test"}))

			data, err = BuildData(ctx, db, ws, roleData.Permissions, roleData.AppScope,
				role.EnvScope{Mode: role.EnvScopeAll})
			Expect(err).NotTo(HaveOccurred())
			Expect(data.BKMonitor.AllApms).To(BeTrue())
		})

		It("grants no APM apps without the observation permission", func() {
			perms := lo.Without(roleData.Permissions, role.Perm.AppObservationView)
			data, err := BuildData(ctx, db, ws, perms, roleData.AppScope, role.EnvScope{Mode: role.EnvScopeAll})
			Expect(err).NotTo(HaveOccurred())
			// 仍返回监控数据，修改角色时据此撤销旧的监控授权
			Expect(data.BKMonitor).NotTo(BeNil())
			Expect(data.BKMonitor.AllApms).To(BeFalse())
			Expect(data.BKMonitor.Apms).To(BeEmpty())
		})

		It("grants nothing in monitor before the monitor space is ready", func() {
			ws.BkSystems.BkMonitorProjectID = ""
			data, err := BuildData(ctx, db, ws, roleData.Permissions, roleData.AppScope, roleData.EnvScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.BKMonitor).To(BeNil())
		})

		It("reapplies roles covering an env after its APM binding changes", func() {
			_, err := mgr.CreateCustomRole(ctx, "partial", "", roleData, nil)
			Expect(err).NotTo(HaveOccurred())
			allData := roleData
			allData.EnvScope = role.EnvScope{Mode: role.EnvScopeAll}
			_, err = mgr.CreateCustomRole(ctx, "all-envs", "", allData, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(ReapplyEnvApm(ctx, db, mgr, testEnv)).To(Succeed())
			// 环境范围为全部的角色授予的是空间下全部 APM 应用，不需要重新授权
			Expect(mgr.updates).To(HaveLen(1))
			Expect(mgr.updates[0].BKMonitor.Apms).To(ConsistOf(
				bkiam.ResourceRef{ID: "11", Name: "apm-test"},
				bkiam.ResourceRef{ID: "12", Name: "apm-prod"},
			))
		})

		It("updates APM grants of env-type-scoped roles when env type changes or env is deleted", func() {
			testTypeData := roleData
			testTypeData.EnvScope = role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}}
			_, err := mgr.CreateCustomRole(ctx, "test-only", "", testTypeData, nil)
			Expect(err).NotTo(HaveOccurred())

			prodTypeData := roleData
			prodTypeData.EnvScope = role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"production"}}
			_, err = mgr.CreateCustomRole(ctx, "prod-only", "", prodTypeData, nil)
			Expect(err).NotTo(HaveOccurred())

			By("changing testEnv from test to production revokes apm-test from test-only and grants it to prod-only")
			Expect(
				envSvc.Update(ctx, testEnv.ID, &envmodel.EnvironmentUpdateData{Type: lo.ToPtr("production")}),
			).To(Succeed())
			after := *testEnv
			after.Type = "production"
			Expect(ReapplyEnv(ctx, db, mgr, testEnv, &after)).To(Succeed())
			Expect(mgr.updates).To(HaveLen(2))
			Expect(mgr.updates[0].BKMonitor.Apms).To(BeEmpty())
			Expect(mgr.updates[1].BKMonitor.Apms).To(ConsistOf(
				bkiam.ResourceRef{ID: "11", Name: "apm-test"},
				bkiam.ResourceRef{ID: "12", Name: "apm-prod"},
			))

			By("deleting prodEnv excludes its APM even before the env document is removed from DB")
			mgr.updates = nil
			Expect(PruneEnv(ctx, db, mgr, prodEnv)).To(Succeed())
			Expect(mgr.updates).To(HaveLen(1))
			Expect(mgr.updates[0].BKMonitor.Apms).To(ConsistOf(
				bkiam.ResourceRef{ID: "11", Name: "apm-test"},
			))
		})
	})

	Context("coverage", func() {
		It("checks env coverage by all, types and specified envs", func() {
			Expect(CoversEnv(role.EnvScope{Mode: role.EnvScopeAll}, prodEnv)).To(BeTrue())
			nonProd := role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}}
			Expect(CoversEnv(nonProd, testEnv)).To(BeTrue())
			Expect(CoversEnv(nonProd, prodEnv)).To(BeFalse())
			specified := role.EnvScope{Mode: role.EnvScopeEnvs, EnvIDs: []string{prodEnv.ID.Hex()}}
			Expect(CoversEnv(specified, prodEnv)).To(BeTrue())
			Expect(CoversEnv(specified, testEnv)).To(BeFalse())
		})

		It("treats built-in roles as covering all apps and envs", func() {
			developer := &role.Role{RoleCode: role.BuiltinRoleCode.Developer}
			operator := &role.Role{RoleCode: role.BuiltinRoleCode.Operator}
			Expect(CanDeploy(developer, prodEnv)).To(BeTrue())
			Expect(CanDeploy(operator, prodEnv)).To(BeFalse())
		})

		It("merges scopes of multiple roles for display", func() {
			merged := MergeScopes([]*role.Role{
				{
					RoleCode: "custom-aaa111",
					AppScope: &role.AppScope{Mode: role.AppScopeApps, AppIDs: []string{"a"}},
					EnvScope: &role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}},
				},
				{
					RoleCode: "custom-bbb222",
					AppScope: &role.AppScope{Mode: role.AppScopeApps, AppIDs: []string{"b", "a"}},
					EnvScope: &role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"development"}},
				},
			})
			Expect(merged.AllApps).To(BeFalse())
			Expect(merged.AppIDs).To(Equal([]string{"a", "b"}))
			Expect(merged.EnvTypes).To(Equal([]string{"development", "test"}))

			merged = MergeScopes([]*role.Role{{RoleCode: role.BuiltinRoleCode.Operator}})
			Expect(merged.AllApps).To(BeTrue())
			Expect(merged.AllEnvs).To(BeTrue())
		})
	})
})
