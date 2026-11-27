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

package perm

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
)

var _ = Describe("StubAllowAnyManager", func() {
	var (
		stub *StubAllowAnyManager
		ctx  context.Context
	)

	BeforeEach(func() {
		stub = &StubAllowAnyManager{}
		ctx = context.Background()
	})

	Context("permission checks", func() {
		It("allows all permission checks", func() {
			Expect(stub.HasCreateWorkspacePerm(ctx)).To(Succeed())
			Expect(stub.HasViewWorkspacePerm(ctx, "any-ws")).To(Succeed())
			Expect(stub.HasCreateAppPerm(ctx, "any-ws")).To(Succeed())
			Expect(stub.HasDeleteAppPerm(ctx, "any-ws", "any-app")).To(Succeed())
			Expect(stub.HasDeployAppToEnvPerm(ctx, "any-ws", "any-app", "test", "any-env")).To(Succeed())
		})

		It("returns all requested resources as viewable", func() {
			ids := []string{"a", "b", "c"}
			workspaces, err := stub.FilterViewableWorkspaces(ctx, ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces.ToSlice()).To(ConsistOf("a", "b", "c"))

			apps, err := stub.FilterViewableApps(ctx, "any-ws", ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(apps.ToSlice()).To(ConsistOf("a", "b", "c"))
		})
	})

	Context("role management", func() {
		It("returns the four canned built-in roles with stable UUIDs from ListRoles", func() {
			roles, err := stub.ListRoles(ctx, "ws-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(roles).To(HaveLen(4))

			ids := make([]string, 0, len(roles))
			codes := make([]string, 0, len(roles))
			for _, r := range roles {
				ids = append(ids, r.ID)
				codes = append(codes, r.RoleCode)
				Expect(r.Scope.ResourceID).To(Equal("ws-1"))
			}
			Expect(ids).To(ConsistOf(
				stubAdminRoleID, stubDeveloperRoleID, stubSRERoleID, stubOperatorRoleID,
			))
			Expect(codes).To(ConsistOf(
				RoleCodeAdmin, RoleCodeDeveloper, RoleCodeSre, RoleCodeOperator,
			))
		})

		It("maps role codes to fixed user lists in ListRoleMembers", func() {
			members, err := stub.ListRoleMembers(ctx, stubAdminRoleID)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(ConsistOf("admin", "blueking"))

			members, err = stub.ListRoleMembers(ctx, stubDeveloperRoleID)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(ConsistOf("developer"))
		})

		It("reflects dynamic role membership changes in ListRoleMembers", func() {
			Expect(stub.AddRoleForUsers(ctx, stubAdminRoleID, []string{"alice"})).To(Succeed())

			members, err := stub.ListRoleMembers(ctx, stubAdminRoleID)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(ContainElement("alice"))
			Expect(members).To(ConsistOf("admin", "blueking", "alice"))

			Expect(stub.DeleteRoleForUsers(ctx, stubAdminRoleID, []string{"alice"})).To(Succeed())

			members, err = stub.ListRoleMembers(ctx, stubAdminRoleID)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(ConsistOf("admin", "blueking"))
		})

		It("keeps CreateWorkspaceAdmin / Update* methods effectively no-op", func() {
			Expect(stub.CreateWorkspaceAdmin(ctx, "ws", "name", []string{"u"}, "ci", "bcs", "repo")).To(Succeed())
			Expect(stub.CreateWorkspaceScopeBuiltinRoles(ctx, "ws", "name", "ci", "bcs", "repo")).To(Succeed())
			Expect(stub.DeleteAllRolesByWorkspaceID(ctx, "ws")).To(Succeed())
		})

		It("keeps custom roles in memory through create, update and delete", func() {
			data := bkiam.CustomRoleData{
				WorkspaceID: "ws-1",
				Permissions: []role.Permission{role.Perm.SpaceView},
				AppScope:    role.AppScope{Mode: role.AppScopeAll},
				EnvScope:    role.EnvScope{Mode: role.EnvScopeTypes, EnvTypes: []string{"test"}},
			}
			created, err := stub.CreateCustomRole(ctx, "non-prod", "", data, []string{"bob"})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.IsCustom()).To(BeTrue())
			Expect(stub.ListRoleMembers(ctx, created.ID)).To(ConsistOf("bob"))

			roles, err := stub.ListRoles(ctx, "ws-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(roles).To(HaveLen(5))
			otherRoles, err := stub.ListRoles(ctx, "ws-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(otherRoles).To(HaveLen(4))

			updated, err := stub.UpdateCustomRole(ctx, created.ID, "renamed", "", data)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.DisplayName).To(Equal("renamed"))

			got, err := stub.GetRole(ctx, "ws-1", created.RoleCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.DisplayName).To(Equal("renamed"))
			_, err = stub.GetRole(ctx, "ws-2", created.RoleCode)
			Expect(errors.Is(err, role.ErrRoleNotFound)).To(BeTrue())

			Expect(stub.DeleteCustomRole(ctx, created.ID)).To(Succeed())
			_, err = stub.GetRole(ctx, "ws-1", created.RoleCode)
			Expect(errors.Is(err, role.ErrRoleNotFound)).To(BeTrue())
		})
	})
})
