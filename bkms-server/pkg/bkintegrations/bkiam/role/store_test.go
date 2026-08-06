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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("RoleStoreMongo", func() {
	var (
		ctx   context.Context
		store *RoleStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Cleanup test data from previous runs
		Expect(testutil.CleanupCollection(gradeManagerCollName)).To(Succeed())
		Expect(testutil.CleanupCollection(roleCollName)).To(Succeed())

		var err error
		store, err = NewRoleStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})

	Describe("WorkspaceGradeManager operations", func() {
		It("should create a workspace grade manager successfully", func() {
			wgm := &WorkspaceGradeManager{WorkspaceID: "ws-001", GradeManagerID: 1001}

			created, err := store.CreateWorkspaceGradeManager(ctx, wgm)
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.WorkspaceID).To(Equal("ws-001"))
			Expect(created.GradeManagerID).To(Equal(1001))
		})

		It("should reject duplicate workspace grade manager (unique index on workspaceID)", func() {
			wgm := &WorkspaceGradeManager{WorkspaceID: "ws-002", GradeManagerID: 2001}
			_, err := store.CreateWorkspaceGradeManager(ctx, wgm)
			Expect(err).NotTo(HaveOccurred())

			dup := &WorkspaceGradeManager{WorkspaceID: "ws-002", GradeManagerID: 2002}
			_, err = store.CreateWorkspaceGradeManager(ctx, dup)
			Expect(err).To(HaveOccurred())
		})

		It("should get a workspace grade manager by workspace id", func() {
			wgm := &WorkspaceGradeManager{WorkspaceID: "ws-003", GradeManagerID: 3001}
			_, err := store.CreateWorkspaceGradeManager(ctx, wgm)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.GetWorkspaceGradeManager(ctx, "ws-003")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
			Expect(got.GradeManagerID).To(Equal(3001))
		})

		It("should return error when workspace grade manager not found", func() {
			got, err := store.GetWorkspaceGradeManager(ctx, "ws-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(got).To(BeNil())
		})

		It("should delete a workspace grade manager by workspace id", func() {
			wgm := &WorkspaceGradeManager{WorkspaceID: "ws-004", GradeManagerID: 4001}
			_, err := store.CreateWorkspaceGradeManager(ctx, wgm)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.DeleteWorkspaceGradeManager(ctx, "ws-004")).To(Succeed())

			_, err = store.GetWorkspaceGradeManager(ctx, "ws-004")
			Expect(err).To(HaveOccurred())
		})

		It("should be a no-op when deleting a non-existent workspace grade manager", func() {
			Expect(store.DeleteWorkspaceGradeManager(ctx, "ws-not-exist")).To(Succeed())
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

		It("should create a role successfully", func() {
			r := newRole("role-001", "n1", "developer", "ws-100", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-100"}, 5001)

			created, err := store.CreateRole(ctx, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).To(Equal("role-001"))
		})

		It("should reject duplicate role id (unique index on id)", func() {
			r := newRole("role-002", "n1", "developer", "ws-100", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-100"}, 5002)
			_, err := store.CreateRole(ctx, r)
			Expect(err).NotTo(HaveOccurred())

			dup := newRole("role-002", "n2", "sre", "ws-101", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-101"}, 5003)
			_, err = store.CreateRole(ctx, dup)
			Expect(err).To(HaveOccurred())
		})

		It("should get a role by id", func() {
			r := newRole("role-003", "n3", "operator", "ws-100", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-100"}, 5004)
			_, err := store.CreateRole(ctx, r)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.GetRoleByID(ctx, "role-003")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
			Expect(got.RoleCode).To(Equal("operator"))
			Expect(got.UserGroupID).To(Equal(5004))
		})

		It("should return error when role not found", func() {
			got, err := store.GetRoleByID(ctx, "role-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(got).To(BeNil())
		})

		It("should delete roles by workspace id and user group ids", func() {
			r1 := newRole("role-d1", "n", "developer", "ws-200", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-200"}, 6001)
			r2 := newRole("role-d2", "n", "sre", "ws-200", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-200"}, 6002)
			r3 := newRole("role-d3", "n", "operator", "ws-200", false,
				PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-200"}, 6003)
			_, err := store.CreateRole(ctx, r1)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.CreateRole(ctx, r2)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.CreateRole(ctx, r3)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.DeleteRolesByUserGroupIDs(ctx, "ws-200", []int{6001, 6002})).To(Succeed())

			// r1, r2 should be gone, r3 should remain
			_, err = store.GetRoleByID(ctx, "role-d1")
			Expect(err).To(HaveOccurred())
			_, err = store.GetRoleByID(ctx, "role-d2")
			Expect(err).To(HaveOccurred())
			got, err := store.GetRoleByID(ctx, "role-d3")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
		})

		It("should be a no-op when deleting roles with empty user group ids", func() {
			Expect(store.DeleteRolesByUserGroupIDs(ctx, "ws-300", []int{})).To(Succeed())
		})

		Context("ListRoles with various filters", func() {
			BeforeEach(func() {
				items := []*Role{
					newRole("role-l1", "admin role", "admin", "ws-list-A", true,
						PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-list-A"}, 7001),
					newRole("role-l2", "dev role", "developer", "ws-list-A", false,
						PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-list-A"}, 7002),
					newRole("role-l3", "sre role", "sre", "ws-list-B", false,
						PermissionScope{ResourceType: WorkspaceResourceType, ResourceID: "ws-list-B"}, 7003),
				}
				for _, r := range items {
					_, err := store.CreateRole(ctx, r)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should list all roles when query params is empty", func() {
				got, err := store.ListRoles(ctx, &RoleQueryParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(3))
			})

			It("should filter by workspace id", func() {
				wsID := "ws-list-A"
				got, err := store.ListRoles(ctx, &RoleQueryParams{WorkspaceID: &wsID})
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(2))
			})

			It("should filter by isGradeManager", func() {
				isGM := true
				got, err := store.ListRoles(ctx, &RoleQueryParams{IsGradeManager: &isGM})
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(1))
				Expect(got[0].ID).To(Equal("role-l1"))
			})

			It("should filter by scope (resourceType + resourceID)", func() {
				got, err := store.ListRoles(ctx, &RoleQueryParams{
					Scope: &PermissionScope{
						ResourceType: WorkspaceResourceType,
						ResourceID:   "ws-list-B",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(1))
				Expect(got[0].ID).To(Equal("role-l3"))
			})

			It("should return empty slice when nothing matches", func() {
				wsID := "ws-not-exist"
				got, err := store.ListRoles(ctx, &RoleQueryParams{WorkspaceID: &wsID})
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(BeEmpty())
			})
		})
	})
})
