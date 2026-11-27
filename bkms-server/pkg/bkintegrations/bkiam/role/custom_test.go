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

package role

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("NormalizePermissions", func() {
	allApps := AppScope{Mode: AppScopeAll}
	allEnvs := EnvScope{Mode: EnvScopeAll}
	someApps := AppScope{Mode: AppScopeApps, AppIDs: []string{"app-a"}}
	nonProd := EnvScope{Mode: EnvScopeTypes, EnvTypes: []string{"development", "test", "staging"}}

	It("always includes base view permissions in canonical order", func() {
		perms, err := NormalizePermissions([]Permission{Perm.EnvDeploy}, allApps, nonProd)
		Expect(err).NotTo(HaveOccurred())
		Expect(perms).To(Equal([]Permission{
			Perm.SpaceView, Perm.AppView, Perm.EnvView, Perm.EnvDeploy,
		}))
	})

	It("returns the three base view permissions for a blank role", func() {
		perms, err := NormalizePermissions(nil, allApps, allEnvs)
		Expect(err).NotTo(HaveOccurred())
		Expect(perms).To(Equal([]Permission{Perm.SpaceView, Perm.AppView, Perm.EnvView}))
	})

	DescribeTable("allows all workspace-scoped permissions regardless of partial scopes",
		func(p Permission, appScope AppScope, envScope EnvScope) {
			_, err := NormalizePermissions([]Permission{p}, appScope, envScope)
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("edit space with partial scopes", Perm.SpaceEdit, someApps, nonProd),
		Entry("delete space with partial scopes", Perm.SpaceDelete, someApps, nonProd),
		Entry("create app with partial scopes", Perm.AppCreate, someApps, nonProd),
		Entry("edit app with partial scopes", Perm.AppEdit, someApps, nonProd),
		Entry("delete app with partial scopes", Perm.AppDelete, someApps, nonProd),
		Entry("create env with partial scopes", Perm.EnvCreate, someApps, nonProd),
		Entry("edit env with partial scopes", Perm.EnvEdit, someApps, nonProd),
		Entry("delete env with partial scopes", Perm.EnvDelete, someApps, nonProd),
	)

	DescribeTable("rejects invalid permissions",
		func(p Permission) {
			_, err := NormalizePermissions([]Permission{p}, allApps, allEnvs)
			Expect(errors.Is(err, ErrInvalidCustomRole)).To(BeTrue())
		},
		Entry("create workspace", Perm.SpaceCreate),
		Entry("unknown permission", Permission("app.unknown")),
	)

	DescribeTable("rejects invalid scopes",
		func(appScope AppScope, envScope EnvScope) {
			_, err := NormalizePermissions(nil, appScope, envScope)
			Expect(errors.Is(err, ErrInvalidCustomRole)).To(BeTrue())
		},
		Entry("empty specified apps", AppScope{Mode: AppScopeApps}, allEnvs),
		Entry("empty env types", allApps, EnvScope{Mode: EnvScopeTypes}),
		Entry("unknown env type", allApps, EnvScope{Mode: EnvScopeTypes, EnvTypes: []string{"prod"}}),
		Entry("empty specified envs", allApps, EnvScope{Mode: EnvScopeEnvs}),
		Entry("unknown app scope mode", AppScope{Mode: "some"}, allEnvs),
	)
})

var _ = Describe("GenCustomRoleCode", func() {
	It("generates base36 custom role codes recognized as custom", func() {
		code := GenCustomRoleCode()
		Expect(code).To(MatchRegexp(`^custom-[0-9a-z]{1,6}$`))
		Expect((&Role{RoleCode: code}).IsCustom()).To(BeTrue())
		Expect((&Role{RoleCode: BuiltinRoleCode.Admin}).IsCustom()).To(BeFalse())
	})
})
