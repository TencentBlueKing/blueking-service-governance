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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
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
	batchAllowed         map[string]bool
	batchPairsAllowed    map[string]map[string]bool
	customRoleResp       *role.Role

	// Captured arguments
	gotUsername          string
	gotWorkspaceID       string
	gotAppID             string
	gotEnvID             string
	gotEnvType           string
	gotEnvRefs           []bkiam.EnvRef
	gotAppEnvRefs        []bkiam.AppEnvRef
	gotCustomRoleData    bkiam.CustomRoleData
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

func (f *fakeIAMServicer) WorkspaceCreateIsAllowed(_ context.Context, username string) (bool, error) {
	f.gotUsername = username
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) WorkspaceActionIsAllowed(
	_ context.Context, username, workspaceID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) WorkspacesMultiActionsAllowed(
	_ context.Context, username string, workspaceIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	f.gotUsername = username
	f.gotIDsForMulti = workspaceIDs
	f.gotActionIDsForMulti = actionIDs
	return f.multiActionsResponse, f.returnedErr
}

func (f *fakeIAMServicer) AppCreateIsAllowed(_ context.Context, username, workspaceID string) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) AppActionIsAllowed(
	_ context.Context, username, workspaceID, appID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotAppID = appID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) AppsMultiActionsAllowed(
	_ context.Context, username, workspaceID string, appIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotIDsForMulti = appIDs
	f.gotActionIDsForMulti = actionIDs
	return f.multiActionsResponse, f.returnedErr
}

func (f *fakeIAMServicer) EnvCreateIsAllowed(_ context.Context, username, workspaceID string) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) EnvActionIsAllowed(
	_ context.Context, username, workspaceID, envType, envID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotEnvType = envType
	f.gotEnvID = envID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) AppEnvActionIsAllowed(
	_ context.Context, username, workspaceID, appID, envType, envID, actionID string,
) (bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotAppID = appID
	f.gotEnvType = envType
	f.gotEnvID = envID
	f.gotActionID = actionID
	return f.allowed, f.returnedErr
}

func (f *fakeIAMServicer) EnvsActionAllowed(
	_ context.Context, username, workspaceID string, envs []bkiam.EnvRef, actionID string,
) (map[string]bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotEnvRefs = envs
	f.gotActionID = actionID
	return f.batchAllowed, f.returnedErr
}

func (f *fakeIAMServicer) AppEnvPairsActionAllowed(
	_ context.Context, username, workspaceID string, pairs []bkiam.AppEnvRef, actionID string,
) (map[string]map[string]bool, error) {
	f.gotUsername = username
	f.gotWorkspaceID = workspaceID
	f.gotAppEnvRefs = pairs
	f.gotActionID = actionID
	return f.batchPairsAllowed, f.returnedErr
}

func (f *fakeIAMServicer) GetRole(_ context.Context, roleID string) (*role.Role, error) {
	f.gotRoleID = roleID
	return f.customRoleResp, f.returnedErr
}

func (f *fakeIAMServicer) CreateCustomRole(
	_ context.Context, _, _ string, data bkiam.CustomRoleData,
) (*role.Role, error) {
	f.gotCustomRoleData = data
	return f.customRoleResp, f.returnedErr
}

func (f *fakeIAMServicer) UpdateCustomRole(
	_ context.Context, roleID, _, _ string, data bkiam.CustomRoleData,
) (*role.Role, error) {
	f.gotRoleID = roleID
	f.gotCustomRoleData = data
	return f.customRoleResp, f.returnedErr
}

func (f *fakeIAMServicer) DeleteCustomRole(_ context.Context, roleID string) error {
	f.gotRoleID = roleID
	return f.returnedErr
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
		testEnvType = "test"
		bkCIID      = "bkci-proj"
		bcsID       = "bcs-proj"
		bkRepoID    = "bkrepo-proj"
		displayName = "WS-1-Display"
	)

	var (
		fake        *fakeIAMServicer
		mgr         *LocalManager
		ctx         context.Context
		originalCfg *config.Config
	)

	BeforeEach(func() {
		fake = newFakeIAMServicer()
		mgr = &LocalManager{svc: fake}
		ctx = ctxWithUser(testUser)
		originalCfg = config.G
		config.G = &config.Config{}
	})

	AfterEach(func() {
		config.G = originalCfg
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

	Context("app env permissions", func() {
		It("checks the app and the env with the app-env action", func() {
			fake.allowed = true
			Expect(mgr.HasDeployAppToEnvPerm(ctx, testWS, testApp, testEnvType, testEnv)).To(Succeed())
			Expect(fake.gotAppID).To(Equal(testApp))
			Expect(fake.gotEnvType).To(Equal(testEnvType))
			Expect(fake.gotEnvID).To(Equal(testEnv))
			Expect(fake.gotActionID).To(Equal(AppEnvAction.Deploy))

			Expect(mgr.HasViewAppInEnvPerm(ctx, testWS, testApp, testEnvType, testEnv)).To(Succeed())
			Expect(fake.gotActionID).To(Equal(AppEnvAction.View))
			Expect(mgr.HasEditAppInEnvPerm(ctx, testWS, testApp, testEnvType, testEnv)).To(Succeed())
			Expect(fake.gotActionID).To(Equal(AppEnvAction.Edit))
		})

		It("returns a deny error mentioning the app and the env", func() {
			fake.allowed = false
			err := mgr.HasDeployAppToEnvPerm(ctx, testWS, testApp, testEnvType, testEnv)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(testApp))
			Expect(err.Error()).To(ContainSubstring(testEnv))
		})

		It("filters viewable envs with the view env action", func() {
			fake.batchAllowed = map[string]bool{"env-a": false, "env-b": true}
			refs := []bkiam.EnvRef{{ID: "env-a", Type: "test"}, {ID: "env-b", Type: "production"}}
			result, err := mgr.FilterViewableEnvs(ctx, testWS, refs)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ToSlice()).To(ConsistOf("env-b"))
			Expect(fake.gotActionID).To(Equal(EnvAction.View))
		})

		It("filters multiple (app, env) pairs in one batch call", func() {
			fake.batchPairsAllowed = map[string]map[string]bool{
				"app-1": {"env-a": true, "env-b": false},
				"app-2": {"env-b": true},
			}
			pairs := []bkiam.AppEnvRef{
				{AppID: "app-1", Env: bkiam.EnvRef{ID: "env-a", Type: "test"}},
				{AppID: "app-1", Env: bkiam.EnvRef{ID: "env-b", Type: "production"}},
				{AppID: "app-2", Env: bkiam.EnvRef{ID: "env-b", Type: "production"}},
			}
			result, err := mgr.FilterViewableAppEnvs(ctx, testWS, pairs)
			Expect(err).NotTo(HaveOccurred())
			Expect(result["app-1"].ToSlice()).To(ConsistOf("env-a"))
			Expect(result["app-2"].ToSlice()).To(ConsistOf("env-b"))
			Expect(fake.gotAppEnvRefs).To(Equal(pairs))
			Expect(fake.gotActionID).To(Equal(AppEnvAction.View))
		})

		It("returns an empty set without calling IAM when there are no envs", func() {
			result, err := mgr.FilterViewableAppEnvs(ctx, testWS, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
			Expect(fake.gotUsername).To(BeEmpty())
		})
	})

	Context("custom roles", func() {
		It("forwards updates to the IAM service", func() {
			fake.customRoleResp = &role.Role{ID: "r-1", WorkspaceID: testWS}
			data := bkiam.CustomRoleData{WorkspaceID: testWS}
			_, err := mgr.UpdateCustomRole(ctx, "r-1", "name", "", data)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.gotRoleID).To(Equal("r-1"))
			Expect(fake.gotCustomRoleData).To(Equal(data))
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

	Context("independent bkrepo", func() {
		BeforeEach(func() {
			config.G.FeatureIntegrations.EnableIndependentBkRepo = true
		})

		It("does not grant bkrepo scopes to workspace admin", func() {
			fake.createWSAdminResp = &role.Role{ID: "r-admin"}
			Expect(mgr.CreateWorkspaceAdmin(
				ctx, testWS, displayName, []string{"u1"}, bkCIID, bcsID, bkRepoID,
			)).To(Succeed())
			Expect(fake.gotCreateWSData.BKCI).NotTo(BeNil())
			Expect(fake.gotCreateWSData.BKRepo).To(BeNil())
		})

		It("does not grant bkrepo scopes to builtin roles", func() {
			Expect(mgr.CreateWorkspaceScopeBuiltinRoles(
				ctx, testWS, displayName, bkCIID, bcsID, bkRepoID,
			)).To(Succeed())
			Expect(fake.gotCreateBuiltinData.BKCI).NotTo(BeNil())
			Expect(fake.gotCreateBuiltinData.BKRepo).To(BeNil())
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

		It("returns role.ErrRoleNotFound when no role matches", func() {
			fake.listRolesResp = []*role.Role{{ID: "r1", RoleCode: RoleCodeAdmin}}
			_, err := mgr.GetRole(ctx, testWS, RoleCodeOperator)
			Expect(errors.Is(err, role.ErrRoleNotFound)).To(BeTrue())
		})

		It("returns role.ErrRoleNotFound when GetRoleByID finds a role in another workspace", func() {
			fake.customRoleResp = &role.Role{ID: "r-other", WorkspaceID: "other-ws"}
			_, err := mgr.GetRoleByID(ctx, testWS, "r-other")
			Expect(errors.Is(err, role.ErrRoleNotFound)).To(BeTrue())
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

		It("hides the platform account from the members", func() {
			originCfg := config.G
			config.G = &config.Config{PlatformAccount: config.PlatformAccountConfig{
				Enabled: true, Username: "bkms-platform", AccessToken: "token",
			}}
			DeferCleanup(func() { config.G = originCfg })

			fake.listMembersResp = []string{"u1", "bkms-platform", "u2"}
			members, err := mgr.ListRoleMembers(ctx, "role-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(members).To(Equal([]string{"u1", "u2"}))
		})
	})
})
