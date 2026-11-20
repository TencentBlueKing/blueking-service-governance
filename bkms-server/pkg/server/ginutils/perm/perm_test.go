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

package perm_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	infrasperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

type fakePermManager struct {
	infrasperm.StubAllowAnyManager

	allowWorkspaceEdit bool
	allowAppView       *set.StringSet
	allowAppEdit       *set.StringSet

	// allowedEnvChecks: "<action>:<envType>:<envID>"
	allowedEnvChecks *set.StringSet
	// allowedAppEnvChecks: "<action>:<appID>:<envType>:<envID>"
	allowedAppEnvChecks *set.StringSet
	// viewableEnvs: "<envType>:<envID>"
	viewableEnvs *set.StringSet
	// viewableAppEnvs: "<appID>:<envType>:<envID>"
	viewableAppEnvs *set.StringSet

	envCheckCalls         []string
	appEnvCheckCalls      []string
	filterEnvsCalls       int
	filterAppEnvsCalls    int
	recordedFilterEnvRefs []bkiam.EnvRef
}

func newFakePermManager() *fakePermManager {
	return &fakePermManager{
		allowAppView:        set.NewStringSet(),
		allowAppEdit:        set.NewStringSet(),
		allowedEnvChecks:    set.NewStringSet(),
		allowedAppEnvChecks: set.NewStringSet(),
		viewableEnvs:        set.NewStringSet(),
		viewableAppEnvs:     set.NewStringSet(),
	}
}

func envCheckKey(action, envType, envID string) string {
	return fmt.Sprintf("%s:%s:%s", action, envType, envID)
}

func appEnvCheckKey(action, appID, envType, envID string) string {
	return fmt.Sprintf("%s:%s:%s:%s", action, appID, envType, envID)
}

func envRefKey(envType, envID string) string {
	return fmt.Sprintf("%s:%s", envType, envID)
}

func appEnvRefKey(appID, envType, envID string) string {
	return fmt.Sprintf("%s:%s:%s", appID, envType, envID)
}

func (f *fakePermManager) HasEditWorkspacePerm(_ context.Context, workspaceID string) error {
	if !f.allowWorkspaceEdit {
		return pkgerrors.Errorf("denied edit workspace %s", workspaceID)
	}
	return nil
}

func (f *fakePermManager) HasViewAppPerm(_ context.Context, _, appID string) error {
	if !f.allowAppView.Has(appID) {
		return pkgerrors.Errorf("denied view app %s", appID)
	}
	return nil
}

func (f *fakePermManager) HasEditAppPerm(_ context.Context, _, appID string) error {
	if !f.allowAppEdit.Has(appID) {
		return pkgerrors.Errorf("denied edit app %s", appID)
	}
	return nil
}

func (f *fakePermManager) HasViewEnvPerm(_ context.Context, _, envType, envID string) error {
	key := envCheckKey("view", envType, envID)
	f.envCheckCalls = append(f.envCheckCalls, key)
	if !f.allowedEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) HasEditEnvPerm(_ context.Context, _, envType, envID string) error {
	key := envCheckKey("edit", envType, envID)
	f.envCheckCalls = append(f.envCheckCalls, key)
	if !f.allowedEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) HasDeleteEnvPerm(_ context.Context, _, envType, envID string) error {
	key := envCheckKey("delete", envType, envID)
	f.envCheckCalls = append(f.envCheckCalls, key)
	if !f.allowedEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) HasViewAppInEnvPerm(_ context.Context, _, appID, envType, envID string) error {
	key := appEnvCheckKey("view", appID, envType, envID)
	f.appEnvCheckCalls = append(f.appEnvCheckCalls, key)
	if !f.allowedAppEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) HasEditAppInEnvPerm(_ context.Context, _, appID, envType, envID string) error {
	key := appEnvCheckKey("edit", appID, envType, envID)
	f.appEnvCheckCalls = append(f.appEnvCheckCalls, key)
	if !f.allowedAppEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) HasDeployAppToEnvPerm(_ context.Context, _, appID, envType, envID string) error {
	key := appEnvCheckKey("deploy", appID, envType, envID)
	f.appEnvCheckCalls = append(f.appEnvCheckCalls, key)
	if !f.allowedAppEnvChecks.Has(key) {
		return pkgerrors.Errorf("denied %s", key)
	}
	return nil
}

func (f *fakePermManager) FilterViewableEnvs(
	_ context.Context, _ string, envs []bkiam.EnvRef,
) (*set.StringSet, error) {
	f.filterEnvsCalls++
	f.recordedFilterEnvRefs = append(f.recordedFilterEnvRefs, envs...)
	allowed := set.NewStringSet()
	for _, ref := range envs {
		if f.viewableEnvs.Has(envRefKey(ref.Type, ref.ID)) {
			allowed.Add(ref.ID)
		}
	}
	return allowed, nil
}

func (f *fakePermManager) FilterViewableAppEnvs(
	_ context.Context, _ string, pairs []bkiam.AppEnvRef,
) (map[string]*set.StringSet, error) {
	f.filterAppEnvsCalls++
	result := map[string]*set.StringSet{}
	for _, pair := range pairs {
		if !f.viewableAppEnvs.Has(appEnvRefKey(pair.AppID, pair.Env.Type, pair.Env.ID)) {
			continue
		}
		if result[pair.AppID] == nil {
			result[pair.AppID] = set.NewStringSet()
		}
		result[pair.AppID].Add(pair.Env.ID)
	}
	return result, nil
}

func expectBKErrCode(err error, code bkerrs.ErrCode) {
	ExpectWithOffset(1, err).To(HaveOccurred())
	var bkErr *bkerrs.Error
	ExpectWithOffset(1, errors.As(err, &bkErr)).To(BeTrue(), "expected bkerrs.Error, got %T: %v", err, err)
	ExpectWithOffset(1, bkErr.Code()).To(Equal(code))
}

var _ = Describe("Ginutils Perm", func() {
	var (
		ctx      context.Context
		db       *mongo.Database
		diApp    *fxtest.App
		envSvc   *bkmsenv.EnvService
		wsStore  *bkmsworkspace.WorkspaceStore
		appStore *bkmsapp.ApplicationStore
		envStore *envmodel.EnvironmentStore
		registry *storereg.Registry

		fakeMgr   *fakePermManager
		permPatch *mockey.Mocker

		ws        *bkmsworkspace.Workspace
		appA      *bkmsapp.Application
		appB      *bkmsapp.Application
		devEnv    *envmodel.Environment
		prodEnv   *envmodel.Environment
		featDevA  *envmodel.Environment
		featProdA *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		wsStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		envStore = testutil.NewStore(GinkgoT(), envmodel.EnvMeta)

		diApp = fxtest.New(GinkgoT(), bkmsenv.FxModule, fx.Populate(&envSvc))
		diApp.RequireStart()

		registry = &storereg.Registry{DB: db}

		fakeMgr = newFakePermManager()
		permPatch = mockey.Mock(infrasperm.NewManager).Return(fakeMgr).Build()

		ws = dbfactory.Workspace(ctx, wsStore)
		appA = dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: ws.ID})
		appB = dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: ws.ID})
		devEnv = dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
			WorkspaceID: ws.ID,
			Type:        "development",
		})
		prodEnv = dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
			WorkspaceID: ws.ID,
			Type:        "production",
		})
		featDevA = dbfactory.FeatEnv(ctx, envSvc, appA, devEnv)
		featProdA = dbfactory.FeatEnv(ctx, envSvc, appA, prodEnv)
	})

	AfterEach(func() {
		permPatch.UnPatch()
		Expect(testutil.CleanupMetaColl(bkmsworkspace.Meta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(bkmsapp.AppMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(envmodel.EnvMeta)).To(Succeed())
		diApp.RequireStop()
	})

	Describe("Feature environment authorization", func() {
		It("authorizes a feature env by its source env's current ID and type", func() {
			// 修改来源环境类型后，特性环境鉴权应使用来源环境的当前类型而非创建时的旧类型
			newType := "staging"
			Expect(envSvc.Update(ctx, devEnv.ID, &envmodel.EnvironmentUpdateData{Type: &newType})).To(Succeed())

			fakeMgr.allowedEnvChecks.Add(envCheckKey("view", "staging", devEnv.ID.Hex()))
			got, err := ginperm.ValidateEnvByID(ctx, registry, featDevA.ID.Hex(), ginperm.TypeView)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(featDevA.ID))
			Expect(fakeMgr.envCheckCalls).To(Equal([]string{envCheckKey("view", "staging", devEnv.ID.Hex())}))

			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("deploy", appA.ID, "staging", devEnv.ID.Hex()))
			Expect(ginperm.ValidateAppEnv(ctx, registry, appA, featDevA, ginperm.TypeDeploy)).To(Succeed())
			Expect(fakeMgr.appEnvCheckCalls).To(Equal([]string{
				appEnvCheckKey("deploy", appA.ID, "staging", devEnv.ID.Hex()),
			}))
		})

		It("fails closed on view/edit of an orphan feature env and requires workspace edit to delete it", func() {
			// 独立删除来源标准环境，使 featDevA 成为孤儿特性环境
			Expect(envStore.DeleteIfExists(ctx, store.ByID(devEnv.ID))).To(Succeed())

			_, err := ginperm.ValidateEnvByID(ctx, registry, featDevA.ID.Hex(), ginperm.TypeView)
			expectBKErrCode(err, bkerrs.ErrCodeInternalServerError)

			_, err = ginperm.ValidateEnvByID(ctx, registry, featDevA.ID.Hex(), ginperm.TypeEdit)
			expectBKErrCode(err, bkerrs.ErrCodeInternalServerError)

			err = ginperm.ValidateAppEnv(ctx, registry, appA, featDevA, ginperm.TypeView)
			expectBKErrCode(err, bkerrs.ErrCodeInternalServerError)

			// 删除孤儿特性环境回退为空间编辑权限：受限角色被拒绝，空间管理员允许清理
			_, err = ginperm.ValidateEnvByID(ctx, registry, featDevA.ID.Hex(), ginperm.TypeDelete)
			expectBKErrCode(err, bkerrs.ErrCodeIAMNoPermission)

			fakeMgr.allowWorkspaceEdit = true
			got, err := ginperm.ValidateEnvByID(ctx, registry, featDevA.ID.Hex(), ginperm.TypeDelete)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(featDevA.ID))
		})
	})

	Describe("App-env validation", func() {
		It("rejects feature envs and cross-workspace envs in ValidateAppEnvByID", func() {
			_, err := ginperm.ValidateAppEnvByID(ctx, registry, appA, featDevA.ID.Hex(), ginperm.TypeDeploy)
			expectBKErrCode(err, bkerrs.ErrCodeNotFound)

			otherWS := dbfactory.Workspace(ctx, wsStore)
			otherEnv := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{WorkspaceID: otherWS.ID})
			_, err = ginperm.ValidateAppEnvByID(ctx, registry, appA, otherEnv.ID.Hex(), ginperm.TypeDeploy)
			expectBKErrCode(err, bkerrs.ErrCodeNotFound)

			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("deploy", appA.ID, devEnv.Type, devEnv.ID.Hex()))
			got, err := ginperm.ValidateAppEnvByID(ctx, registry, appA, devEnv.ID.Hex(), ginperm.TypeDeploy)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(devEnv.ID))
		})

		It("isolates feature envs by owner app in ValidateAppEnvByName", func() {
			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("edit", appA.ID, devEnv.Type, devEnv.ID.Hex()))
			gotApp, gotEnv, err := ginperm.ValidateAppEnvByName(
				ctx, registry, appA.ID, featDevA.Name, ginperm.TypeEdit,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotApp.ID).To(Equal(appA.ID))
			Expect(gotEnv.ID).To(Equal(featDevA.ID))

			// appB 不能访问 appA 拥有的特性环境
			_, _, err = ginperm.ValidateAppEnvByName(ctx, registry, appB.ID, featDevA.Name, ginperm.TypeView)
			expectBKErrCode(err, bkerrs.ErrCodeNotFound)
		})

		It("enforces TypeView, TypeEdit, and TypeDeploy semantics in ValidateAppInAllEnvs", func() {
			fakeMgr.allowAppView.Add(appA.ID)
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, devEnv.Type, devEnv.ID.Hex()))

			// TypeView：仅可见 dev、不可见 prod 时拒绝
			_, err := ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeView)
			expectBKErrCode(err, bkerrs.ErrCodeIAMNoPermission)

			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, prodEnv.Type, prodEnv.ID.Hex()))
			_, err = ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeView)
			Expect(err).NotTo(HaveOccurred())

			// TypeEdit：只校验 edit_app，不触发逐环境 edit_app_in_env
			_, err = ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeEdit)
			expectBKErrCode(err, bkerrs.ErrCodeIAMNoPermission)

			fakeMgr.allowAppEdit.Add(appA.ID)
			_, err = ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeEdit)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeMgr.appEnvCheckCalls).To(BeEmpty())

			// TypeDeploy：校验 edit_app 加上每个标准环境的 deploy_app_to_env
			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("deploy", appA.ID, devEnv.Type, devEnv.ID.Hex()))
			_, err = ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeDeploy)
			expectBKErrCode(err, bkerrs.ErrCodeIAMNoPermission)

			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("deploy", appA.ID, prodEnv.Type, prodEnv.ID.Hex()))
			_, err = ginperm.ValidateAppInAllEnvs(ctx, registry, appA.ID, ginperm.TypeDeploy)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deduplicates source env checks and skips unknown env names in ValidateAppEnvNames", func() {
			fakeMgr.allowedAppEnvChecks.Add(appEnvCheckKey("edit", appA.ID, devEnv.Type, devEnv.ID.Hex()))

			// devEnv 与 featDevA 共用同一鉴权环境，只检查一次；不存在的环境名跳过
			err := ginperm.ValidateAppEnvNames(
				ctx, registry, appA,
				[]string{devEnv.Name, featDevA.Name, "deleted-env"},
				ginperm.TypeEdit,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeMgr.appEnvCheckCalls).To(Equal([]string{
				appEnvCheckKey("edit", appA.ID, devEnv.Type, devEnv.ID.Hex()),
			}))
		})

		It("requires view permission on every standard env in ValidateWorkspaceViewInAllEnvs", func() {
			fakeMgr.viewableEnvs.Add(envRefKey(devEnv.Type, devEnv.ID.Hex()))
			_, err := ginperm.ValidateWorkspaceViewInAllEnvs(ctx, registry, ws.ID)
			expectBKErrCode(err, bkerrs.ErrCodeIAMNoPermission)

			fakeMgr.viewableEnvs.Add(envRefKey(prodEnv.Type, prodEnv.ID.Hex()))
			gotWS, err := ginperm.ValidateWorkspaceViewInAllEnvs(ctx, registry, ws.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotWS.ID).To(Equal(ws.ID))
		})
	})

	Describe("Batch filtering", func() {
		It("skips orphan feature envs in FilterViewableEnvs without failing the batch", func() {
			Expect(envStore.DeleteIfExists(ctx, store.ByID(prodEnv.ID))).To(Succeed())
			fakeMgr.viewableEnvs.Add(envRefKey(devEnv.Type, devEnv.ID.Hex()))

			result, err := ginperm.FilterViewableEnvs(
				ctx, registry, ws.ID,
				[]envmodel.Environment{*devEnv, *featDevA, *featProdA},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(result, func(e envmodel.Environment, _ int) string { return e.Name })).
				To(ConsistOf(devEnv.Name, featDevA.Name))
			Expect(fakeMgr.filterEnvsCalls).To(Equal(1))
			Expect(fakeMgr.recordedFilterEnvRefs).To(Equal([]bkiam.EnvRef{
				{ID: devEnv.ID.Hex(), Type: devEnv.Type},
			}))
		})

		It("filters multiple apps in one IAM request and skips orphan feature envs", func() {
			Expect(envStore.DeleteIfExists(ctx, store.ByID(prodEnv.ID))).To(Succeed())
			stagingEnv := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
				WorkspaceID: ws.ID,
				Type:        "staging",
			})

			// appA 仅可见 dev（含 featDevA），appB 仅可见 staging；验证多应用切片偏移与单次批量调用
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, devEnv.Type, devEnv.ID.Hex()))
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appB.ID, stagingEnv.Type, stagingEnv.ID.Hex()))

			viewable, err := ginperm.FilterViewableAppEnvNamesByApp(
				ctx, registry, ws.ID,
				map[string][]envmodel.Environment{
					appA.ID: {*devEnv, *featDevA, *featProdA, *stagingEnv},
					appB.ID: {*devEnv, *stagingEnv},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeMgr.filterAppEnvsCalls).To(Equal(1))
			Expect(viewable[appA.ID].ToSlice()).To(ConsistOf(devEnv.Name, featDevA.Name))
			Expect(viewable[appB.ID].ToSlice()).To(ConsistOf(stagingEnv.Name))
		})

		It("filters multi-app items in one IAM request via FilterByViewableAppEnvsByApp", func() {
			type statusRow struct {
				EnvName string
				Tag     string
			}

			// appA 仅可见 dev（含 featDevA），appB 仅可见 prod
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, devEnv.Type, devEnv.ID.Hex()))
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appB.ID, prodEnv.Type, prodEnv.ID.Hex()))

			filtered, err := ginperm.FilterByViewableAppEnvsByApp(
				ctx, registry, ws.ID,
				map[string][]statusRow{
					appA.ID: {
						{EnvName: devEnv.Name, Tag: "v1-dev"},
						{EnvName: devEnv.Name, Tag: "v1-dev-lane"},
						{EnvName: featDevA.Name, Tag: "v1-feat-dev"},
						{EnvName: prodEnv.Name, Tag: "v1-prod"},
						{EnvName: featProdA.Name, Tag: "v1-feat-prod"},
						{EnvName: "", Tag: "global"},
					},
					appB.ID: {
						{EnvName: devEnv.Name, Tag: "v2-dev"},
						{EnvName: prodEnv.Name, Tag: "v2-prod"},
						// featDevA 归属 appA，不应暴露给 appB
						{EnvName: featDevA.Name, Tag: "cross-app-feat"},
						{EnvName: "non-existent", Tag: "ghost"},
					},
				},
				func(r statusRow) string { return r.EnvName },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeMgr.filterAppEnvsCalls).To(Equal(1))
			Expect(lo.Map(filtered[appA.ID], func(r statusRow, _ int) string { return r.Tag })).
				To(Equal([]string{"v1-dev", "v1-dev-lane", "v1-feat-dev", "global"}))
			Expect(lo.Map(filtered[appB.ID], func(r statusRow, _ int) string { return r.Tag })).
				To(Equal([]string{"v2-prod"}))
		})

		It("tracks partial vs full visibility and returns non-nil empty slice in ListViewableAppEnvNames", func() {
			// 无任何可见环境时必须返回非 nil 空切片，避免 MongoDB $in: nil 报错
			names, all, err := ginperm.ListViewableAppEnvNames(ctx, registry, appA)
			Expect(err).NotTo(HaveOccurred())
			Expect(names).NotTo(BeNil())
			Expect(names).To(BeEmpty())
			Expect(all).To(BeFalse())

			// 仅可见 dev（含其特性环境）
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, devEnv.Type, devEnv.ID.Hex()))
			names, all, err = ginperm.ListViewableAppEnvNames(ctx, registry, appA)
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(ConsistOf(devEnv.Name, featDevA.Name))
			Expect(all).To(BeFalse())

			// 补齐 prod 权限后全部可见
			fakeMgr.viewableAppEnvs.Add(appEnvRefKey(appA.ID, prodEnv.Type, prodEnv.ID.Hex()))
			names, all, err = ginperm.ListViewableAppEnvNames(ctx, registry, appA)
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(ConsistOf(devEnv.Name, prodEnv.Name, featDevA.Name, featProdA.Name))
			Expect(all).To(BeTrue())
		})
	})
})
