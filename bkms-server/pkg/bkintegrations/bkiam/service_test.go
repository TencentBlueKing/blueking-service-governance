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
	"context"

	iamsdk "github.com/TencentBlueKing/iam-go-sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	cloudapiiam "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam"
	cloudapiiamtypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// fakeIAMClient is an in-memory IAMClient used for IAMService unit tests.
//
// It records the most recent invocation arguments per method and returns
// pre-configured stub values. By default every operation succeeds with
// canned IDs; tests may override individual fields to simulate failures
// or to inspect call arguments.
type fakeIAMClient struct {
	// Pre-configured return values
	gradeManagerID int
	userGroupID    int
	createGMErr    error
	listMembers    []cloudapiiamtypes.UserMember
	noUserGroups   bool

	// Captured arguments
	gotCreateGMUsers      []string
	gotAddGMMembers       []string
	gotDeleteGMMembers    []string
	gotAddUGMembers       []string
	gotAddUGExpiredAt     int
	gotDeleteUGMembers    []string
	gotGrantedScopesCalls int
	gotDeletedUserGroups  []int
	gotDeletedGM          int

	// Counters for action checks
	isAllowedReturn      bool
	multiActionsResponse map[string]map[string]bool
}

var _ cloudapiiam.IAMClient = (*fakeIAMClient)(nil)

// newFakeIAMClient builds a fakeIAMClient with sensible defaults.
func newFakeIAMClient() *fakeIAMClient {
	return &fakeIAMClient{
		gradeManagerID:  100,
		userGroupID:     1000,
		listMembers:     []cloudapiiamtypes.UserMember{{ID: "alice"}, {ID: "bob"}},
		isAllowedReturn: true,
	}
}

func (f *fakeIAMClient) CreateGradeManager(
	_ context.Context, _, _ string, members []string, _ []cloudapiiamtypes.AuthorizationScope,
) (*int, error) {
	if f.createGMErr != nil {
		return nil, f.createGMErr
	}
	f.gotCreateGMUsers = append([]string(nil), members...)
	id := f.gradeManagerID
	return &id, nil
}

func (f *fakeIAMClient) UpdateGradeManager(
	_ context.Context, _ int, _, _ string, _ []cloudapiiamtypes.AuthorizationScope,
) error {
	return nil
}

func (f *fakeIAMClient) DeleteGradeManager(_ context.Context, gradeManagerID int) error {
	f.gotDeletedGM = gradeManagerID
	return nil
}

func (f *fakeIAMClient) GetGradeManagerByName(_ context.Context, _ string) (*int, error) {
	id := f.gradeManagerID
	return &id, nil
}

func (f *fakeIAMClient) AddGradeManagerMembers(_ context.Context, _ int, members []string) error {
	f.gotAddGMMembers = append([]string(nil), members...)
	return nil
}

func (f *fakeIAMClient) DeleteGradeManagerMembers(_ context.Context, _ int, members []string) error {
	f.gotDeleteGMMembers = append([]string(nil), members...)
	return nil
}

func (f *fakeIAMClient) CreateUserGroups(
	_ context.Context, _ int, groups ...cloudapiiamtypes.UserGroupParam,
) ([]cloudapiiamtypes.UserGroup, error) {
	if f.noUserGroups {
		return nil, nil
	}
	out := make([]cloudapiiamtypes.UserGroup, len(groups))
	for i, g := range groups {
		out[i] = cloudapiiamtypes.UserGroup{
			ID: f.userGroupID + i, Name: g.Name, Description: g.Description, Readonly: g.Readonly,
		}
	}
	return out, nil
}

func (f *fakeIAMClient) DeleteUserGroup(_ context.Context, userGroupID int) error {
	f.gotDeletedUserGroups = append(f.gotDeletedUserGroups, userGroupID)
	return nil
}

func (f *fakeIAMClient) GrantUserGroupPolicies(
	_ context.Context, _ int, _ []cloudapiiamtypes.AuthorizationScope,
) error {
	f.gotGrantedScopesCalls++
	return nil
}

func (f *fakeIAMClient) AddUserGroupMembers(
	_ context.Context, _ int, members []string, expiredAt int,
) error {
	f.gotAddUGMembers = append([]string(nil), members...)
	f.gotAddUGExpiredAt = expiredAt
	return nil
}

func (f *fakeIAMClient) DeleteUserGroupMembers(_ context.Context, _ int, members []string) error {
	f.gotDeleteUGMembers = append([]string(nil), members...)
	return nil
}

func (f *fakeIAMClient) ListUserGroupMembers(
	_ context.Context, _ int,
) ([]cloudapiiamtypes.UserMember, error) {
	return f.listMembers, nil
}

func (f *fakeIAMClient) IsAllowed(_ iamsdk.Request) (bool, error) {
	return f.isAllowedReturn, nil
}

func (f *fakeIAMClient) BatchResourceMultiActionsAllowed(
	_ iamsdk.MultiActionRequest, resourcesList []iamsdk.Resources,
) (map[string]map[string]bool, error) {
	if f.multiActionsResponse != nil {
		return f.multiActionsResponse, nil
	}
	results := make(map[string]map[string]bool, len(resourcesList))
	for _, resources := range resourcesList {
		if len(resources) == 0 {
			continue
		}
		results[resources[0].ID] = map[string]bool{}
	}
	return results, nil
}

// fakeRoleStore is an in-memory role.RoleStore implementation used to drive
// IAMService specs without requiring a real MongoDB.
type fakeRoleStore struct {
	gradeManagers map[string]*role.WorkspaceGradeManager
	rolesByID     map[string]*role.Role
	roles         []*role.Role

	listErr error
}

var _ role.RoleStore = (*fakeRoleStore)(nil)

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{
		gradeManagers: map[string]*role.WorkspaceGradeManager{},
		rolesByID:     map[string]*role.Role{},
	}
}

func (s *fakeRoleStore) CreateWorkspaceGradeManager(
	_ context.Context, wgm *role.WorkspaceGradeManager,
) (*role.WorkspaceGradeManager, error) {
	if _, ok := s.gradeManagers[wgm.WorkspaceID]; ok {
		return nil, errors.New("duplicate grade manager")
	}
	s.gradeManagers[wgm.WorkspaceID] = wgm
	return wgm, nil
}

func (s *fakeRoleStore) GetWorkspaceGradeManager(
	_ context.Context, workspaceID string,
) (*role.WorkspaceGradeManager, error) {
	wgm, ok := s.gradeManagers[workspaceID]
	if !ok {
		return nil, errors.Errorf("workspace(%s) grade manager not found", workspaceID)
	}
	return wgm, nil
}

func (s *fakeRoleStore) DeleteWorkspaceGradeManager(_ context.Context, workspaceID string) error {
	delete(s.gradeManagers, workspaceID)
	return nil
}

func (s *fakeRoleStore) CreateRole(_ context.Context, r *role.Role) (*role.Role, error) {
	s.rolesByID[r.ID] = r
	s.roles = append(s.roles, r)
	return r, nil
}

func (s *fakeRoleStore) GetRoleByID(_ context.Context, roleID string) (*role.Role, error) {
	r, ok := s.rolesByID[roleID]
	if !ok {
		return nil, errors.Errorf("role(%s) not found", roleID)
	}
	return r, nil
}

func (s *fakeRoleStore) DeleteRolesByUserGroupIDs(
	_ context.Context, workspaceID string, userGroupIDs []int,
) error {
	keep := s.roles[:0]
	idSet := map[int]struct{}{}
	for _, id := range userGroupIDs {
		idSet[id] = struct{}{}
	}
	for _, r := range s.roles {
		if r.WorkspaceID == workspaceID {
			if _, ok := idSet[r.UserGroupID]; ok {
				delete(s.rolesByID, r.ID)
				continue
			}
		}
		keep = append(keep, r)
	}
	s.roles = keep
	return nil
}

func (s *fakeRoleStore) ListRoles(
	_ context.Context, params *role.RoleQueryParams,
) ([]*role.Role, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]*role.Role, 0, len(s.roles))
	for _, r := range s.roles {
		if params.WorkspaceID != nil && *params.WorkspaceID != r.WorkspaceID {
			continue
		}
		if params.IsGradeManager != nil && *params.IsGradeManager != r.IsGradeManager {
			continue
		}
		if params.Scope != nil &&
			(params.Scope.ResourceType != r.Scope.ResourceType ||
				params.Scope.ResourceID != r.Scope.ResourceID) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

var _ = Describe("IAMService", func() {
	const workspaceID = "ws-001"

	var (
		ctx       context.Context
		client    *fakeIAMClient
		store     *fakeRoleStore
		svc       *IAMService
		originCfg *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		client = newFakeIAMClient()
		store = newFakeRoleStore()
		svc = NewIAMService(client, store)

		// IAMService reads config.G.BkIAMSystemIDs.Bkms when building IAM
		// requests; populate a minimal config and restore on AfterEach.
		originCfg = config.G
		config.G = &config.Config{
			BkIAMSystemIDs: config.BkIAMSystemIDsConfig{
				Bkms:      "bkms",
				BkCI:      "bk_ci",
				BCS:       "bk_bcs",
				BkMonitor: "bk_monitor",
				BkLog:     "bk_log",
				BkRepo:    "bk_repo",
				BSCP:      "bk_bscp",
				BkCC:      "bk_cmdb",
			},
		}
	})

	AfterEach(func() {
		config.G = originCfg
	})

	Context("workspace admin lifecycle", func() {
		It("creates a workspace admin role with grade manager and members", func() {
			data := WorkspaceData{WorkspaceID: workspaceID, WorkspaceName: "ws-001-name"}
			users := []string{"alice", "bob"}

			r, err := svc.CreateWorkspaceAdmin(ctx, data, users)
			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
			Expect(r.RoleCode).To(Equal(role.BuiltinRoleCode.Admin))
			Expect(r.IsGradeManager).To(BeTrue())
			Expect(r.WorkspaceID).To(Equal(workspaceID))
			Expect(r.Scope.ResourceType).To(Equal(role.WorkspaceResourceType))

			// grade manager record should be persisted
			wgm, err := store.GetWorkspaceGradeManager(ctx, workspaceID)
			Expect(err).NotTo(HaveOccurred())
			Expect(wgm.GradeManagerID).To(Equal(client.gradeManagerID))

			// members should be passed through to user group
			Expect(client.gotAddUGMembers).To(Equal(users))
			Expect(client.gotAddUGExpiredAt).To(Equal(cloudapiiam.NeverExpireTimestamp))
		})

		It("returns the existing admin role when called twice (idempotent)", func() {
			data := WorkspaceData{WorkspaceID: workspaceID}
			first, err := svc.CreateWorkspaceAdmin(ctx, data, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			// Second call must short-circuit and return the same role.
			second, err := svc.CreateWorkspaceAdmin(ctx, data, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())
			Expect(second.ID).To(Equal(first.ID))
		})

		It("propagates grade manager creation errors", func() {
			client.createGMErr = errors.New("iam down")
			_, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, nil)
			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err).Error()).To(ContainSubstring("iam down"))
		})

		It("returns an error when IAM creates no user groups", func() {
			client.noUserGroups = true
			_, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty result"))
		})

		It("updates workspace admin scopes against an existing grade manager", func() {
			_, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())
			grantsBefore := client.gotGrantedScopesCalls

			err = svc.UpdateWorkspaceAdmin(ctx, WorkspaceData{
				WorkspaceID: workspaceID,
				BKCI:        &BKCIOptions{ProjectID: "p1", ProjectName: "p1-name"},
			})
			Expect(err).NotTo(HaveOccurred())
			// Update should grant policies on the admin user group at least once.
			Expect(client.gotGrantedScopesCalls).To(BeNumerically(">", grantsBefore))
		})
	})

	Context("workspace builtin roles", func() {
		It("creates only missing builtin roles and returns the full set", func() {
			_, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			roles, err := svc.CreateWorkspaceScopeBuiltinRoles(ctx, WorkspaceData{WorkspaceID: workspaceID})
			Expect(err).NotTo(HaveOccurred())
			// Should have the builtin set excluding the admin grade manager role.
			Expect(len(roles)).To(Equal(len(role.WorkspaceScopeBuiltinRoles)))

			// Calling it again should be a no-op (no extra role rows created).
			rolesAgain, err := svc.CreateWorkspaceScopeBuiltinRoles(ctx, WorkspaceData{WorkspaceID: workspaceID})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rolesAgain)).To(Equal(len(roles)))
		})
	})

	Context("role list / get / members", func() {
		It("lists workspace-scope roles filtered by scope", func() {
			_, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			scope := role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID}
			roles, err := svc.ListRoles(ctx, workspaceID, scope)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(roles)).To(BeNumerically(">=", 1))
		})

		It("lists role members from the underlying user group", func() {
			r, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			members, err := svc.ListRoleMembers(ctx, r.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(Equal([]string{"alice", "bob"}))
		})

		It("adds users to a grade-manager role both at GM and UG layers", func() {
			r, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.AddRoleForUsers(ctx, r.ID, []string{"carol"})).To(Succeed())
			Expect(client.gotAddGMMembers).To(Equal([]string{"carol"}))
			Expect(client.gotAddUGMembers).To(Equal([]string{"carol"}))
		})

		It("removes users from a grade-manager role both at GM and UG layers", func() {
			r, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.DeleteRoleForUsers(ctx, r.ID, []string{"alice"})).To(Succeed())
			Expect(client.gotDeleteGMMembers).To(Equal([]string{"alice"}))
			Expect(client.gotDeleteUGMembers).To(Equal([]string{"alice"}))
		})
	})

	Context("permission checks", func() {
		It("delegates WorkspaceCreateIsAllowed to the IAM client", func() {
			allowed, err := svc.WorkspaceCreateIsAllowed("alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue())

			client.isAllowedReturn = false
			allowed, err = svc.WorkspaceCreateIsAllowed("alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("returns batched results for AppsMultiActionsAllowed", func() {
			client.multiActionsResponse = map[string]map[string]bool{
				"app-1": {"view_app": true},
				"app-2": {"view_app": false},
			}
			res, err := svc.AppsMultiActionsAllowed(
				"alice", workspaceID, []string{"app-1", "app-2"}, []string{"view_app"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(res["app-1"]["view_app"]).To(BeTrue())
			Expect(res["app-2"]["view_app"]).To(BeFalse())
		})
	})

	Context("workspace teardown", func() {
		It("deletes user groups, role rows, grade manager and its record", func() {
			r, err := svc.CreateWorkspaceAdmin(ctx, WorkspaceData{WorkspaceID: workspaceID}, []string{"alice"})
			Expect(err).NotTo(HaveOccurred())
			expectedUGID := r.UserGroupID

			Expect(svc.DeleteAllRolesByWorkspaceID(ctx, workspaceID)).To(Succeed())

			// User group should have been deleted via the IAM client.
			Expect(client.gotDeletedUserGroups).To(ContainElement(expectedUGID))
			// Grade manager record should have been removed from local store.
			_, err = store.GetWorkspaceGradeManager(ctx, workspaceID)
			Expect(err).To(HaveOccurred())
		})
	})
})
