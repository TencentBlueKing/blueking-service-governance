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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// fakeIAMServicer is an in-memory iamServicer used for LocalManager unit
// tests. It records the most recent invocation arguments per method and
// returns pre-configured stub values.
type fakeIAMServicer struct {
	// Pre-configured return values
	allowed              bool
	multiActionsResponse map[string]map[string]bool
	listRolesResp        []*role.Role
	listMembersResp      []string
	createWSAdminResp    *role.Role
	createBuiltinResp    []*role.Role
	returnedErr          error

	// Captured arguments
	gotUsername          string
	gotWorkspaceID       string
	gotAppID             string
	gotEnvID             string
	gotActionID          string
	gotIDsForMulti       []string
	gotActionIDsForMulti []string
	gotCreateWSData      bkiam.WorkspaceData
	gotCreateWSUsers     []string
	gotCreateBuiltinData bkiam.WorkspaceData
	gotUpdateWSData      bkiam.WorkspaceData
	gotUpdateBuiltinData bkiam.WorkspaceData
	gotRoleID            string
	gotRoleUsers         []string
	gotListRolesScope    role.PermissionScope
}

func newFakeIAMServicer() *fakeIAMServicer {
	return &fakeIAMServicer{}
}

func (f *fakeIAMServicer) CreateWorkspaceAdmin(
	_ context.Context, data bkiam.WorkspaceData, users []string,
) (*role.Role, error) {
	f.gotCreateWSData = data
	f.gotCreateWSUsers = users
	return f.createWSAdminResp, f.returnedErr
}

func (f *fakeIAMServicer) UpdateWorkspaceAdmin(_ context.Context, data bkiam.WorkspaceData) error {
	f.gotUpdateWSData = data
	return f.returnedErr
}

func (f *fakeIAMServicer) CreateWorkspaceScopeBuiltinRoles(
	_ context.Context, data bkiam.WorkspaceData,
) ([]*role.Role, error) {
	f.gotCreateBuiltinData = data
	return f.createBuiltinResp, f.returnedErr
}

func (f *fakeIAMServicer) UpdateWorkspaceScopeBuiltinRoles(
	_ context.Context, data bkiam.WorkspaceData,
) error {
	f.gotUpdateBuiltinData = data
	return f.returnedErr
}

func (f *fakeIAMServicer) AddRoleForUsers(_ context.Context, roleID string, users []string) error {
	f.gotRoleID = roleID
	f.gotRoleUsers = users
	return f.returnedErr
}

func (f *fakeIAMServicer) DeleteRoleForUsers(_ context.Context, roleID string, users []string) error {
	f.gotRoleID = roleID
	f.gotRoleUsers = users
	return f.returnedErr
}

func (f *fakeIAMServicer) ListRoles(
	_ context.Context, workspaceID string, scp role.PermissionScope,
) ([]*role.Role, error) {
	f.gotWorkspaceID = workspaceID
	f.gotListRolesScope = scp
	return f.listRolesResp, f.returnedErr
}

func (f *fakeIAMServicer) ListRoleMembers(_ context.Context, roleID string) ([]string, error) {
	f.gotRoleID = roleID
	return f.listMembersResp, f.returnedErr
}

func (f *fakeIAMServicer) DeleteAllRolesByWorkspaceID(_ context.Context, workspaceID string) error {
	f.gotWorkspaceID = workspaceID
	return f.returnedErr
}

func (f *fakeIAMServicer) WorkspaceCreateIsAllowed(username string) (bool, error) {
	f.gotUsername = username
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) WorkspaceActionIsAllowed(
	username, workspaceID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) WorkspacesMultiActionsAllowed(
	username string, workspaceIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	f.gotUsername = username
	f.gotIDsForMulti = workspaceIDs
	f.gotActionIDsForMulti = actionIDs
	return f.multiActionsResponse, f.returnedErr
}

func (f *fakeIAMServicer) AppCreateIsAllowed(username, workspaceID string) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) AppActionIsAllowed(
	username, workspaceID, appID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotAppID = appID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) AppsMultiActionsAllowed(
	username, workspaceID string, appIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotIDsForMulti = appIDs
	f.gotActionIDsForMulti = actionIDs
	return f.multiActionsResponse, f.returnedErr
}

func (f *fakeIAMServicer) EnvCreateIsAllowed(username, workspaceID string) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) EnvActionIsAllowed(
	username, workspaceID, envID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotEnvID = envID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

// ctxWithUser returns a context that carries the given username, mimicking
// the auth middleware that injects auth.User into the request context.
func ctxWithUser(username string) context.Context {
	return context.WithValue(context.Background(), ctxkey.AuthUser, auth.User{ID: username})
}

var _ = Describe("LocalManager", func() {
	const (
		testUser    = "alice"
		testWS      = "ws-1"
		testApp     = "app-1"
		testEnv     = "env-1"
		bkCIID      = "bkci-proj"
		bcsID       = "bcs-proj"
		bkRepoID    = "bkrepo-proj"
		displayName = "WS-1-Display"
	)

	var (
		fake *fakeIAMServicer
		mgr  *LocalManager
		ctx  context.Context
	)

	BeforeEach(func() {
		fake = newFakeIAMServicer()
		mgr = &LocalManager{svc: fake}
		ctx = ctxWithUser(testUser)
	})

	Context("HasCreateWorkspacePerm", func() {
		It("returns nil when IAMService grants the permission and forwards username from ctx", func() {
			fake.allowed = true
			Expect(mgr.HasCreateWorkspacePerm(ctx)).To(Succeed())
			Expect(fake.gotUsername).To(Equal(testUser))
		})

		It("returns a no-permission error when IAMService denies", func() {
			fake.allowed = false
			err := mgr.HasCreateWorkspacePerm(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no permission to create workspace"))
		})

		It("wraps IAMService transport errors", func() {
			fake.returnedErr = errors.New("upstream boom")
			err := mgr.HasCreateWorkspacePerm(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("upstream boom"))
		})

		It("fails fast when ctx has no auth user", func() {
			err := mgr.HasCreateWorkspacePerm(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get auth user"))
		})
	})

	Context("HasViewAppPerm", func() {
		It("forwards workspace id, app id and view action to IAMService", func() {
			fake.allowed = true
			Expect(mgr.HasViewAppPerm(ctx, testWS, testApp)).To(Succeed())
			Expect(fake.gotUsername).To(Equal(testUser))
			Expect(fake.gotWorkspaceID).To(Equal(testWS))
			Expect(fake.gotAppID).To(Equal(testApp))
			Expect(fake.gotActionID).To(Equal(AppAction.View))
		})

		It("returns a deny error mentioning workspace and app id", func() {
			fake.allowed = false
			err := mgr.HasViewAppPerm(ctx, testWS, testApp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(testApp))
			Expect(err.Error()).To(ContainSubstring(testWS))
		})
	})

	Context("HasDeployEnvPerm", func() {
		It("forwards workspace, env and the deploy action to IAMService", func() {
			fake.allowed = true
			Expect(mgr.HasDeployEnvPerm(ctx, testWS, testEnv)).To(Succeed())
			Expect(fake.gotEnvID).To(Equal(testEnv))
			Expect(fake.gotActionID).To(Equal(EnvAction.Deploy))
		})
	})

	Context("FilterViewableWorkspaces", func() {
		It("returns the subset of workspaces that the IAMService marks as viewable", func() {
			fake.multiActionsResponse = map[string]map[string]bool{
				"ws-a": {WorkspaceAction.View: true},
				"ws-b": {WorkspaceAction.View: false},
				"ws-c": {WorkspaceAction.View: true},
			}
			result, err := mgr.FilterViewableWorkspaces(ctx, []string{"ws-a", "ws-b", "ws-c"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ToSlice()).To(ConsistOf("ws-a", "ws-c"))
			Expect(fake.gotActionIDsForMulti).To(ConsistOf(WorkspaceAction.View))
			Expect(fake.gotIDsForMulti).To(ConsistOf("ws-a", "ws-b", "ws-c"))
		})
	})

	Context("CreateWorkspaceAdmin", func() {
		It("packs split args into iam.WorkspaceData with the legacy field-mapping quirk", func() {
			fake.createWSAdminResp = &role.Role{ID: "r-admin"}
			err := mgr.CreateWorkspaceAdmin(
				ctx, testWS, displayName, []string{"u1", "u2"},
				bkCIID, bcsID, bkRepoID,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.gotCreateWSUsers).To(Equal([]string{"u1", "u2"}))
			Expect(fake.gotCreateWSData.WorkspaceID).To(Equal(testWS))
			Expect(fake.gotCreateWSData.WorkspaceName).To(Equal(displayName))

			By("BKCI options")
			Expect(fake.gotCreateWSData.BKCI).NotTo(BeNil())
			Expect(fake.gotCreateWSData.BKCI.ProjectID).To(Equal(bkCIID))
			Expect(fake.gotCreateWSData.BKCI.ProjectName).To(Equal(bkCIID))

			By("BCS options preserve the legacy quirk: ProjectName takes bkCIProjectID, NOT bcsProjectID")
			Expect(fake.gotCreateWSData.BCS).NotTo(BeNil())
			Expect(fake.gotCreateWSData.BCS.ProjectID).To(Equal(bcsID))
			Expect(fake.gotCreateWSData.BCS.ProjectName).To(Equal(bkCIID))

			By("BKRepo options")
			Expect(fake.gotCreateWSData.BKRepo).NotTo(BeNil())
			Expect(fake.gotCreateWSData.BKRepo.ProjectID).To(Equal(bkRepoID))
			Expect(fake.gotCreateWSData.BKRepo.ProjectName).To(Equal(bkRepoID))
		})
	})

	Context("ListRoles", func() {
		It("fixes scope to WorkspaceResourceType and forwards workspace id", func() {
			fake.listRolesResp = []*role.Role{{ID: "r1"}, {ID: "r2"}}
			roles, err := mgr.ListRoles(ctx, testWS)
			Expect(err).NotTo(HaveOccurred())
			Expect(roles).To(HaveLen(2))
			Expect(fake.gotWorkspaceID).To(Equal(testWS))
			Expect(fake.gotListRolesScope.ResourceType).To(Equal(role.WorkspaceResourceType))
			Expect(fake.gotListRolesScope.ResourceID).To(Equal(testWS))
		})
	})

	Context("GetRole", func() {
		It("returns the role with matching RoleCode from the list", func() {
			fake.listRolesResp = []*role.Role{
				{ID: "r1", RoleCode: RoleCodeAdmin},
				{ID: "r2", RoleCode: RoleCodeDeveloper},
			}
			r, err := mgr.GetRole(ctx, testWS, RoleCodeDeveloper)
			Expect(err).NotTo(HaveOccurred())
			Expect(r.ID).To(Equal("r2"))
		})

		It("returns 'role not found' when no role matches", func() {
			fake.listRolesResp = []*role.Role{{ID: "r1", RoleCode: RoleCodeAdmin}}
			_, err := mgr.GetRole(ctx, testWS, RoleCodeOperator)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role not found"))
		})
	})

	Context("ListRoleMembers", func() {
		It("forwards roleID to IAMService and returns the members slice", func() {
			fake.listMembersResp = []string{"u1", "u2"}
			members, err := mgr.ListRoleMembers(ctx, "role-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(Equal([]string{"u1", "u2"}))
			Expect(fake.gotRoleID).To(Equal("role-x"))
		})
	})
})
