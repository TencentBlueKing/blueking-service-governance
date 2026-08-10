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

package handler

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
)

type fakeDefaultAlertRoleManager struct {
	roles          []*role.Role
	members        map[string][]string
	listRolesCalls int
}

func (f *fakeDefaultAlertRoleManager) ListRoles(_ context.Context, _ string) ([]*role.Role, error) {
	f.listRolesCalls++
	return f.roles, nil
}

func (f *fakeDefaultAlertRoleManager) ListRoleMembers(_ context.Context, roleID string) ([]string, error) {
	return f.members[roleID], nil
}

type fakeDefaultAlertUserGroupService struct {
	findResp    *bkmapi.UserGroup
	findName    string
	saveResp    *bkmapi.UserGroupDetail
	savedParams *usergroup.SaveParams
}

func (f *fakeDefaultAlertUserGroupService) FindByName(
	_ context.Context, _ *workspace.Workspace, name, _ string,
) (*bkmapi.UserGroup, error) {
	f.findName = name
	return f.findResp, nil
}

func (f *fakeDefaultAlertUserGroupService) Save(
	_ context.Context, _ *workspace.Workspace, params *usergroup.SaveParams,
) (*bkmapi.UserGroupDetail, error) {
	f.savedParams = params
	return f.saveResp, nil
}

type concurrentRoleManager struct {
	roles          []*role.Role
	members        map[string][]string
	listRolesCalls int
	mu             sync.Mutex
	started        int
	allStarted     chan struct{}
}

func (m *concurrentRoleManager) ListRoles(_ context.Context, _ string) ([]*role.Role, error) {
	m.listRolesCalls++
	return m.roles, nil
}

func (m *concurrentRoleManager) ListRoleMembers(ctx context.Context, roleID string) ([]string, error) {
	m.mu.Lock()
	m.started++
	if m.started == 2 {
		close(m.allStarted)
	}
	m.mu.Unlock()

	select {
	case <-m.allStarted:
		return m.members[roleID], nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ = Describe("default alert user group resolution", func() {
	buildPermMgr := func() defaultAlertRoleManager {
		return &fakeDefaultAlertRoleManager{
			roles: []*role.Role{
				{ID: "role-developer", RoleCode: perm.RoleCodeDeveloper},
				{ID: "role-sre", RoleCode: perm.RoleCodeSre},
			},
			members: map[string][]string{
				"role-developer": {"dev-a", "shared"},
				"role-sre":       {"sre-a", "shared"},
			},
		}
	}

	It("reuses existing default alert user group without overwriting members", func() {
		groupSvc := &fakeDefaultAlertUserGroupService{
			findResp: &bkmapi.UserGroup{
				ID:   3001,
				Name: buildDefaultAlertUserGroupName("ws-1"),
			},
		}

		ids, err := resolveDefaultAlertNoticeGroupIDs(
			context.Background(),
			&workspace.Workspace{ID: "ws-1"},
			groupSvc,
			buildPermMgr(),
			"tester",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(ids).To(Equal([]int64{3001}))
		Expect(groupSvc.findName).To(Equal(buildDefaultAlertUserGroupName("ws-1")))
		Expect(groupSvc.savedParams).To(BeNil())
	})

	It("creates default alert user group when missing", func() {
		groupSvc := &fakeDefaultAlertUserGroupService{
			saveResp: &bkmapi.UserGroupDetail{UserGroup: bkmapi.UserGroup{ID: 4001}},
		}

		ids, err := resolveDefaultAlertNoticeGroupIDs(
			context.Background(),
			&workspace.Workspace{ID: "ws-1"},
			groupSvc,
			buildPermMgr(),
			"tester",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(ids).To(Equal([]int64{4001}))
		Expect(groupSvc.savedParams).NotTo(BeNil())
		Expect(groupSvc.savedParams.Name).To(Equal(buildDefaultAlertUserGroupName("ws-1")))
		Expect(groupSvc.savedParams.Users).To(Equal([]bkmapi.UserGroupUser{
			{ID: "dev-a", Type: "user"},
			{ID: "shared", Type: "user"},
			{ID: "sre-a", Type: "user"},
		}))
		Expect(groupSvc.savedParams.AlertNotice).To(Equal([]bkmapi.AlertNotice{{
			TimeRange: "00:00--23:59",
			NotifyConfig: []bkmapi.AlertNoticeConfig{
				{
					Level: 1,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
						{Name: "mail", Receivers: []string{}},
						{Name: "sms", Receivers: []string{}},
						{Name: "voice", Receivers: []string{}},
					},
				},
				{
					Level: 2,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
					},
				},
				{
					Level: 3,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
					},
				},
			},
		}}))
		Expect(groupSvc.savedParams.ActionNotice).To(Equal([]bkmapi.ActionNotice{{
			TimeRange: "00:00--23:59",
			NotifyConfig: []bkmapi.ActionNoticeConfig{
				{
					Phase: 1,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
						{Name: "mail", Receivers: []string{}},
						{Name: "sms", Receivers: []string{}},
						{Name: "voice", Receivers: []string{}},
					},
				},
				{
					Phase: 2,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
					},
				},
				{
					Phase: 3,
					Type:  []string{},
					NoticeWays: []bkmapi.NoticeWay{
						{Name: "weixin", Receivers: []string{}},
					},
				},
			},
		}}))
	})

	It("skips creating default alert user group when both roles have no members", func() {
		groupSvc := &fakeDefaultAlertUserGroupService{}
		permMgr := &fakeDefaultAlertRoleManager{
			roles: []*role.Role{
				{ID: "role-developer", RoleCode: perm.RoleCodeDeveloper},
				{ID: "role-sre", RoleCode: perm.RoleCodeSre},
			},
			members: map[string][]string{
				"role-developer": nil,
				"role-sre":       nil,
			},
		}

		ids, err := resolveDefaultAlertNoticeGroupIDs(
			context.Background(),
			&workspace.Workspace{ID: "ws-1"},
			groupSvc,
			permMgr,
			"tester",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(ids).To(BeNil())
		Expect(groupSvc.savedParams).To(BeNil())
	})

	It("lists roles once and fetches target role members concurrently", func() {
		var _ defaultAlertRoleManager = (*concurrentRoleManager)(nil)

		manager := &concurrentRoleManager{
			roles: []*role.Role{
				{ID: "role-developer", RoleCode: perm.RoleCodeDeveloper},
				{ID: "role-sre", RoleCode: perm.RoleCodeSre},
			},
			members: map[string][]string{
				"role-developer": {"dev-a"},
				"role-sre":       {"sre-a"},
			},
			allStarted: make(chan struct{}),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		members, err := listDefaultAlertGroupMembers(ctx, manager, "ws-1")

		Expect(err).NotTo(HaveOccurred())
		Expect(manager.listRolesCalls).To(Equal(1))
		Expect(members).To(Equal([]string{"dev-a", "sre-a"}))
	})
})
