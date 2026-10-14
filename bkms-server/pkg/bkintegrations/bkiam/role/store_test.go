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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("RoleStore", func() {
	var (
		ctx           context.Context
		roles         *RoleStore
		gradeManagers *GradeManagerStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		roles = testutil.NewStore(GinkgoT(), RoleMeta)
		gradeManagers = testutil.NewStore(GinkgoT(), GradeManagerMeta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(GradeManagerMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(RoleMeta)).To(Succeed())
	})

	Describe("WorkspaceGradeManager operations", func() {
		It("should reject duplicate workspace grade manager (unique index on workspaceID)", func() {
			Expect(gradeManagers.Create(ctx, &WorkspaceGradeManager{WorkspaceID: "ws-002", GradeManagerID: 2001})).
				To(Succeed())

			dup := &WorkspaceGradeManager{WorkspaceID: "ws-002", GradeManagerID: 2002}
			Expect(gradeManagers.Create(ctx, dup)).To(MatchError(store.ErrDuplicate))
		})
	})

	Describe("Role operations", func() {
		newRole := func(id, name, code, wsID string, isGM bool, scope PermissionScope, ugID int) *Role {
			return &Role{
				ID:             id,
				Name:           name,
				RoleCode:       code,
				Description:    "desc-" + id,
				WorkspaceID:    wsID,
				IsGradeManager: isGM,
				Scope:          scope,
				UserGroupID:    ugID,
			}
		}
		workspaceScope := func(wsID string) PermissionScope {
			return PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: wsID}
		}

		It("should reject duplicate role id (unique index on id)", func() {
			Expect(roles.Create(ctx, newRole("role-002", "n1", "developer", "ws-100", false,
				workspaceScope("ws-100"), 5002))).To(Succeed())

			dup := newRole("role-002", "n2", "sre", "ws-101", false, workspaceScope("ws-101"), 5003)
			Expect(roles.Create(ctx, dup)).To(MatchError(store.ErrDuplicate))
		})

		It("should delete roles by workspace id and user group ids", func() {
			for i, code := range []string{"developer", "sre", "operator"} {
				r := newRole(fmt.Sprintf("role-d%d", i+1), "n", code, "ws-200", false,
					workspaceScope("ws-200"), 6001+i)
				Expect(roles.Create(ctx, r)).To(Succeed())
			}
			// 其他 workspace 下相同用户组 ID 的角色不受影响
			Expect(roles.Create(ctx, newRole("role-other", "n", "developer", "ws-201", false,
				workspaceScope("ws-201"), 6001))).To(Succeed())

			deleted, err := roles.DeleteMany(ctx, ByUserGroupIDs("ws-200", []int{6001, 6002}))
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(Equal(int64(2)))

			// r1, r2 should be gone, r3 should remain
			Expect(roles.Exists(ctx, ByRoleID("role-d1"))).To(BeFalse())
			Expect(roles.Exists(ctx, ByRoleID("role-d2"))).To(BeFalse())
			Expect(roles.Exists(ctx, ByRoleID("role-d3"))).To(BeTrue())
			Expect(roles.Exists(ctx, ByRoleID("role-other"))).To(BeTrue())
		})

		It("should be a no-op when deleting roles with empty or nil user group ids", func() {
			Expect(roles.DeleteMany(ctx, ByUserGroupIDs("ws-300", []int{}))).To(BeZero())
			Expect(roles.DeleteMany(ctx, ByUserGroupIDs("ws-300", nil))).To(BeZero())
		})

		Context("List with various filters", func() {
			BeforeEach(func() {
				items := []*Role{
					newRole("role-l1", "admin role", "admin", "ws-list-A", true, workspaceScope("ws-list-A"), 7001),
					newRole("role-l2", "dev role", "developer", "ws-list-A", false, workspaceScope("ws-list-A"), 7002),
					newRole("role-l3", "sre role", "sre", "ws-list-B", false, workspaceScope("ws-list-B"), 7003),
				}
				for _, r := range items {
					Expect(roles.Create(ctx, r)).To(Succeed())
				}
			})

			It("should filter by isGradeManager", func() {
				got, err := roles.List(ctx, ByGradeManager(true))
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(1))
				Expect(got[0].ID).To(Equal("role-l1"))
			})

			It("should filter by scope (resourceType + resourceID)", func() {
				got, err := roles.List(ctx, ByScope(workspaceScope("ws-list-B")))
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(1))
				Expect(got[0].ID).To(Equal("role-l3"))
			})
		})
	})
})
