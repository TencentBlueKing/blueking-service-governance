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

package usergroup

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/lock"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

var _ = Describe("default alert group resolution", func() {
	buildGroupService := func(client bkmapi.MonitorClient) *Service {
		return &Service{
			newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
		}
	}

	buildPermMgr := func() *perm.StubAllowAnyManager {
		return &perm.StubAllowAnyManager{}
	}

	BeforeEach(func() {
		redis.InitClientForTest()
	})

	It("reuses existing default alert user group without overwriting members", func() {
		mockey.PatchConvey("reuse existing group without perm lookup", GinkgoT(), func() {
			bkmapi.ResetStubStateForTest()
			client := bkmapi.NewStub("tester")
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(3001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         buildDefaultAlertUserGroupName("ws-1"),
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			groupSvc := buildGroupService(client)
			permMgr := buildPermMgr()

			var listRolesCalls int32
			mockey.Mock((*perm.StubAllowAnyManager).ListRoles).To(
				func(_ *perm.StubAllowAnyManager, _ context.Context, _ string) ([]*role.Role, error) {
					atomic.AddInt32(&listRolesCalls, 1)
					return nil, nil
				},
			).Build()

			ids, err := ResolveDefaultAlertNoticeGroupIDs(
				context.Background(),
				buildWorkspace(),
				groupSvc,
				permMgr,
				"tester",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal([]int64{3001}))
			Expect(atomic.LoadInt32(&listRolesCalls)).To(Equal(int32(0)))
			groups, err := groupSvc.List(context.Background(), buildWorkspace(), "tester")
			Expect(err).NotTo(HaveOccurred())
			matchedGroups := lo.Filter(groups, func(group *bkmapi.UserGroup, _ int) bool {
				return group != nil && group.Name == buildDefaultAlertUserGroupName("ws-1")
			})
			Expect(matchedGroups).To(HaveLen(1))
			Expect(matchedGroups[0].ID).To(Equal(int64(3001)))
		})
	})

	It("creates default alert user group when missing", func() {
		bkmapi.ResetStubStateForTest()
		client := bkmapi.NewStub("tester")
		groupSvc := buildGroupService(client)
		ws := buildWorkspace()

		ids, err := ResolveDefaultAlertNoticeGroupIDs(
			context.Background(),
			ws,
			groupSvc,
			buildPermMgr(),
			"tester",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(ids).To(HaveLen(1))
		detail, err := client.SearchUserGroupDetail(context.Background(), &bkmapi.SearchUserGroupDetailReq{ID: ids[0]})
		Expect(err).NotTo(HaveOccurred())
		Expect(detail).NotTo(BeNil())
		Expect(detail.BkBizID).To(Equal(testUserGroupBkBizID))
		Expect(detail.DutyArranges).To(HaveLen(1))
		Expect(detail.DutyArranges[0].Users).To(Equal([]bkmapi.UserGroupUser{
			{ID: "developer", Type: "user"},
			{ID: "sre", Type: "user"},
		}))
		Expect(detail.AlertNotice).To(Equal([]bkmapi.AlertNotice{{
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
		Expect(detail.ActionNotice).To(Equal([]bkmapi.ActionNotice{{
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

	It("reuses the group created by another holder of the workspace lock", func() {
		bkmapi.ResetStubStateForTest()
		redis.InitClientForTest()
		client := bkmapi.NewStub("tester")
		groupSvc := buildGroupService(client)
		ws := buildWorkspace()
		lockKey := buildDefaultAlertUserGroupLockKey(ws.ID)
		groupLock := lock.NewRedisLock(lockKey, 1)
		Expect(groupLock.Acquire(context.Background())).To(BeTrue())

		createDone := make(chan error, 1)
		go func() {
			time.Sleep(50 * time.Millisecond)
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(4001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         buildDefaultAlertUserGroupName(ws.ID),
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			groupLock.Release(context.Background())
			createDone <- err
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ids, err := ResolveDefaultAlertNoticeGroupIDs(
			ctx,
			ws,
			groupSvc,
			buildPermMgr(),
			"tester",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(ids).To(Equal([]int64{4001}))
		Expect(<-createDone).NotTo(HaveOccurred())
		groups, err := groupSvc.List(context.Background(), ws, "tester")
		Expect(err).NotTo(HaveOccurred())
		matchedGroups := lo.Filter(groups, func(group *bkmapi.UserGroup, _ int) bool {
			return group != nil && group.Name == buildDefaultAlertUserGroupName(ws.ID)
		})
		Expect(matchedGroups).To(HaveLen(1))
	})

	It("skips creating default alert user group when both roles have no members", func() {
		mockey.PatchConvey("skip create when target roles have no members", GinkgoT(), func() {
			bkmapi.ResetStubStateForTest()
			client := bkmapi.NewStub("tester")
			groupSvc := buildGroupService(client)
			permMgr := buildPermMgr()

			mockey.Mock((*perm.StubAllowAnyManager).ListRoleMembers).To(
				func(_ *perm.StubAllowAnyManager, _ context.Context, roleID string) ([]string, error) {
					switch roleID {
					case "developer-role-id", "sre-role-id":
						return nil, nil
					default:
						return nil, nil
					}
				},
			).Build()

			ids, err := ResolveDefaultAlertNoticeGroupIDs(
				context.Background(),
				buildWorkspace(),
				groupSvc,
				permMgr,
				"tester",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(BeNil())
			group, err := groupSvc.FindByName(
				context.Background(),
				buildWorkspace(),
				buildDefaultAlertUserGroupName("ws-1"),
				"tester",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(group).To(BeNil())
		})
	})

	It("lists roles once and fetches target role members concurrently", func() {
		mockey.PatchConvey("list role members concurrently", GinkgoT(), func() {
			manager := buildPermMgr()
			roles, err := manager.ListRoles(context.Background(), "ws-1")
			Expect(err).NotTo(HaveOccurred())

			expectedMembersByRoleID := map[string][]string{}
			for _, roleInfo := range roles {
				if roleInfo == nil {
					continue
				}
				if roleInfo.RoleCode != perm.RoleCodeDeveloper && roleInfo.RoleCode != perm.RoleCodeSre {
					continue
				}
				members, listErr := manager.ListRoleMembers(context.Background(), roleInfo.ID)
				Expect(listErr).NotTo(HaveOccurred())
				expectedMembersByRoleID[roleInfo.ID] = members
			}

			var listRolesCalls int32
			var started int32
			allStarted := make(chan struct{})

			mockey.Mock((*perm.StubAllowAnyManager).ListRoles).To(
				func(_ *perm.StubAllowAnyManager, _ context.Context, _ string) ([]*role.Role, error) {
					atomic.AddInt32(&listRolesCalls, 1)
					return roles, nil
				},
			).Build()
			mockey.Mock((*perm.StubAllowAnyManager).ListRoleMembers).To(
				func(_ *perm.StubAllowAnyManager, ctx context.Context, roleID string) ([]string, error) {
					if atomic.AddInt32(&started, 1) == 2 {
						close(allStarted)
					}
					select {
					case <-allStarted:
						return expectedMembersByRoleID[roleID], nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
			).Build()

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			members, err := listDefaultAlertGroupMembers(ctx, manager, "ws-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(atomic.LoadInt32(&listRolesCalls)).To(Equal(int32(1)))
			Expect(members).To(Equal([]string{"developer", "sre"}))
		})
	})

	It("times out when the workspace lock stays unavailable", func() {
		bkmapi.ResetStubStateForTest()
		redis.InitClientForTest()
		client := bkmapi.NewStub("tester")
		groupSvc := buildGroupService(client)
		ws := buildWorkspace()
		lockKey := buildDefaultAlertUserGroupLockKey(ws.ID)
		groupLock := lock.NewRedisLock(lockKey, 1)
		Expect(groupLock.Acquire(context.Background())).To(BeTrue())
		defer groupLock.Release(context.Background())

		originalWaitTimeout := defaultAlertUserGroupLockWaitTimeout
		defaultAlertUserGroupLockWaitTimeout = 150 * time.Millisecond
		defer func() {
			defaultAlertUserGroupLockWaitTimeout = originalWaitTimeout
		}()

		done := make(chan error, 1)
		go func() {
			_, err := ResolveDefaultAlertNoticeGroupIDs(
				context.Background(),
				ws,
				groupSvc,
				buildPermMgr(),
				"tester",
			)
			done <- err
		}()

		Eventually(done, time.Second).Should(Receive(MatchError(ContainSubstring(
			"wait default alert user group lock",
		))))
	})
})
