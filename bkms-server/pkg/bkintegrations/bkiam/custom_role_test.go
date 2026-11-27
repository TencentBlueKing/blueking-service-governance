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

package bkiam

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	cloudapiiamtypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// scopeActions 返回授权中的 action ID
func scopeActions(s cloudapiiamtypes.AuthorizationScope) []string {
	return lo.Map(s.Actions, func(a cloudapiiamtypes.Action, _ int) string { return a.ID })
}

// allActions 返回全部授权中的 action ID
func allActions(scopes []cloudapiiamtypes.AuthorizationScope) []string {
	return lo.FlatMap(scopes, func(s cloudapiiamtypes.AuthorizationScope, _ int) []string { return scopeActions(s) })
}

// findScope 返回包含指定 action 的授权
func findScope(scopes []cloudapiiamtypes.AuthorizationScope, actionID string) cloudapiiamtypes.AuthorizationScope {
	s, ok := lo.Find(scopes, func(s cloudapiiamtypes.AuthorizationScope) bool {
		return lo.Contains(scopeActions(s), actionID)
	})
	Expect(ok).To(BeTrue(), "scope of action %s not found", actionID)
	return s
}

var _ = Describe("custom role auth scopes", func() {
	var originCfg *config.Config

	BeforeEach(func() {
		originCfg = config.G
		config.G = &config.Config{BkIAMSystemIDs: config.BkIAMSystemIDsConfig{Bkms: "bkms"}}
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("grants app-env actions with the app and env scopes in the same policy", func() {
		scopes := GenCustomRoleAuthScopes(CustomRoleData{
			WorkspaceID: "ws",
			Permissions: []role.Permission{
				role.Perm.SpaceView,
				role.Perm.AppView,
				role.Perm.EnvView,
				role.Perm.EnvDeploy,
			},
			AppScope: role.AppScope{Mode: role.AppScopeApps, AppIDs: []string{"app-a"}},
			Apps:     []ResourceRef{{ID: "app-a", Name: "A"}},
			EnvScope: role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"development", "test"}},
		})

		appEnv := findScope(scopes, "deploy_app_to_env")
		// 部署带上同一范围内的环境编辑权限
		Expect(scopeActions(appEnv)).To(ConsistOf("view_app_in_env", "edit_app_in_env", "deploy_app_to_env"))
		Expect(appEnv.Resources).To(HaveLen(2))
		app, env := appEnv.Resources[0], appEnv.Resources[1]
		Expect(app.Type).To(BeEquivalentTo("app"))
		Expect(app.Paths).To(HaveLen(1))
		Expect(app.Paths[0][1].ID).To(Equal("app-a"))
		Expect(env.Type).To(BeEquivalentTo("env"))
		Expect(lo.Map(env.Paths, func(p cloudapiiamtypes.ResourcePaths, _ int) string { return p[1].ID })).
			To(ConsistOf("development", "test"))
		Expect(env.Paths[0][1].Type).To(BeEquivalentTo("env_type"))

		// 只涉及单一资源的权限沿用原有 action
		Expect(scopeActions(findScope(scopes, "view_app"))).To(ConsistOf("view_app"))
		Expect(findScope(scopes, "view_app").Resources).To(HaveLen(1))
		Expect(scopeActions(findScope(scopes, "view_env"))).To(ConsistOf("view_env"))
		Expect(scopeActions(findScope(scopes, "view_workspace"))).To(ConsistOf("view_workspace"))
		// 部署不带上应用级的编辑权限
		Expect(allActions(scopes)).NotTo(ContainElement("edit_app"))
	})

	It("composes app-env view and edit actions from app and env permissions", func() {
		data := CustomRoleData{
			WorkspaceID: "ws",
			Permissions: []role.Permission{role.Perm.AppView, role.Perm.AppEdit},
			AppScope:    role.AppScope{Mode: role.AppScopeAll},
			EnvScope:    role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}},
		}
		// 仅有 app 侧权限时，不授予 app-env 联合 action，部分环境也不授予 edit_app
		Expect(allActions(GenCustomRoleAuthScopes(data))).To(ConsistOf("view_app"))

		// 同时具备 app 和 env 权限时，组合授予 view_app_in_env 和 edit_app_in_env
		data.Permissions = []role.Permission{
			role.Perm.AppView, role.Perm.EnvView, role.Perm.AppEdit, role.Perm.EnvEdit,
		}
		Expect(allActions(GenCustomRoleAuthScopes(data))).To(ConsistOf(
			"view_app", "view_env", "edit_env", "view_app_in_env", "edit_app_in_env",
		))

		// 覆盖全部环境时，才额外授予应用级的 edit_app
		data.EnvScope = role.EnvScope{Mode: role.EnvScopeAll}
		Expect(allActions(GenCustomRoleAuthScopes(data))).To(ContainElements("edit_app", "edit_app_in_env"))
	})

	It("uses the workspace node for all apps and all envs", func() {
		scopes := GenCustomRoleAuthScopes(CustomRoleData{
			WorkspaceID: "ws",
			Permissions: []role.Permission{role.Perm.AppEdit, role.Perm.EnvEdit},
			AppScope:    role.AppScope{Mode: role.AppScopeAll},
			EnvScope:    role.EnvScope{Mode: role.EnvScopeAll},
		})
		appEnv := findScope(scopes, "edit_app_in_env")
		for _, res := range appEnv.Resources {
			Expect(res.Paths).To(HaveLen(1))
			Expect(res.Paths[0]).To(HaveLen(1))
			Expect(res.Paths[0][0].Type).To(BeEquivalentTo("workspace"))
			Expect(res.Paths[0][0].ID).To(Equal("ws"))
		}
	})

	It("puts specified envs under their env type node", func() {
		scopes := GenCustomRoleAuthScopes(CustomRoleData{
			WorkspaceID: "ws",
			Permissions: []role.Permission{role.Perm.EnvView},
			AppScope:    role.AppScope{Mode: role.AppScopeAll},
			EnvScope:    role.EnvScope{Mode: role.EnvScopeEnvs, EnvIDs: []string{"oid-1"}},
			Envs:        []EnvResourceRef{{ID: "oid-1", Name: "测试", Type: "test"}},
		})
		path := findScope(scopes, "view_env").Resources[0].Paths[0]
		Expect(path).To(HaveLen(3))
		Expect(path[1].Type).To(BeEquivalentTo("env_type"))
		Expect(path[1].ID).To(Equal("test"))
		Expect(path[2].Type).To(BeEquivalentTo("env"))
		Expect(path[2].ID).To(Equal("oid-1"))
	})

	It("grants nothing on a resource whose specified scope became empty", func() {
		scopes := GenCustomRoleAuthScopes(CustomRoleData{
			WorkspaceID: "ws",
			Permissions: []role.Permission{
				role.Perm.SpaceView, role.Perm.AppView, role.Perm.EnvView, role.Perm.EnvDeploy,
			},
			AppScope: role.AppScope{Mode: role.AppScopeAll},
			EnvScope: role.EnvScope{Mode: role.EnvScopeEnvs},
		})
		// 环境都被删除后，不能授予没有路径的策略（可能被理解为不限制）
		Expect(allActions(scopes)).To(ConsistOf("view_workspace", "view_app"))
		for _, s := range scopes {
			for _, res := range s.Resources {
				Expect(res.Paths).NotTo(BeEmpty())
			}
		}
	})

	It("derives built-in role permissions from the role templates", func() {
		Expect(BuiltinRolePermissions(role.BuiltinRoleCode.Operator)).
			To(ConsistOf(role.Perm.SpaceView, role.Perm.AppView, role.Perm.AppObservationView, role.Perm.EnvView))
		Expect(BuiltinRolePermissions(role.BuiltinRoleCode.Developer)).To(ConsistOf(
			role.Perm.SpaceView, role.Perm.AppCreate, role.Perm.AppView, role.Perm.AppObservationView,
			role.Perm.AppEdit, role.Perm.AppDelete, role.Perm.EnvView, role.Perm.EnvDeploy,
		))
		Expect(BuiltinRolePermissions(role.BuiltinRoleCode.SRE)).NotTo(ContainElement(role.Perm.AppDelete))
		Expect(BuiltinRolePermissions(role.BuiltinRoleCode.Admin)).To(ContainElements(
			role.Perm.SpaceEdit, role.Perm.SpaceDelete, role.Perm.EnvDelete, role.Perm.EnvDeploy,
		))
	})

	Context("with the monitor space ready", func() {
		BeforeEach(func() {
			config.G.BkIAMSystemIDs.BkMonitor = "bk_monitorv3"
		})

		monitorData := func(envScope role.EnvScope, perms ...role.Permission) CustomRoleData {
			return CustomRoleData{
				WorkspaceID: "ws",
				Permissions: perms,
				AppScope:    role.AppScope{Mode: role.AppScopeAll},
				EnvScope:    envScope,
				BKMonitor: &CustomRoleMonitorData{
					SpaceID: "100", SpaceName: "ws",
					AllApms: envScope.IsAll(),
					Apms:    []ResourceRef{{ID: "11", Name: "apm-test"}},
				},
			}
		}

		It("grants business access and the APM apps of the envs in scope", func() {
			scopes := GenCustomRoleAuthScopes(monitorData(
				role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}},
				role.Perm.SpaceView, role.Perm.AppView, role.Perm.AppObservationView,
			))
			business := findScope(scopes, "view_business_v2")
			Expect(business.System).To(Equal("bk_monitorv3"))
			Expect(business.Resources[0].Paths[0][0].ID).To(Equal("100"))

			apm := findScope(scopes, "view_apm_application_v2")
			Expect(apm.Resources[0].Type).To(BeEquivalentTo("apm_application"))
			Expect(apm.Resources[0].Paths).To(HaveLen(1))
			Expect(apm.Resources[0].Paths[0]).To(HaveLen(2))
			Expect(apm.Resources[0].Paths[0][1].ID).To(Equal("11"))
		})

		It("grants all APM apps of the space for all envs", func() {
			scopes := GenCustomRoleAuthScopes(monitorData(
				role.EnvScope{Mode: role.EnvScopeAll},
				role.Perm.SpaceView, role.Perm.AppView, role.Perm.AppObservationView,
			))
			apm := findScope(scopes, "view_apm_application_v2")
			Expect(apm.Resources[0].Paths).To(HaveLen(1))
			Expect(apm.Resources[0].Paths[0]).To(HaveLen(1))
			Expect(apm.Resources[0].Paths[0][0].Type).To(BeEquivalentTo("space"))
		})

		It("grants nothing in monitor without the observation permission", func() {
			scopes := GenCustomRoleAuthScopes(monitorData(
				role.EnvScope{Mode: role.EnvScopeAll}, role.Perm.SpaceView, role.Perm.AppView, role.Perm.EnvView,
			))
			Expect(allActions(scopes)).NotTo(ContainElement("view_business_v2"))
			Expect(allActions(scopes)).NotTo(ContainElement("view_apm_application_v2"))
		})
	})
})
