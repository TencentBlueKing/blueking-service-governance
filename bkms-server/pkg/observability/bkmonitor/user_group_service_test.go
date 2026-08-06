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

package bkmonitor

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

var _ = Describe("UserGroupService", func() {
	// ==================== BuildUserGroupName ====================

	Describe("BuildUserGroupName", func() {
		// 验证告警组名称格式化
		It("format env name to user group name", func() {
			Expect(BuildUserGroupName("test-1")).To(Equal("【APM】test-1 告警组"))
		})
	})

	// ==================== ApplyMemberDiff ====================

	Describe("ApplyMemberDiff", func() {
		// 新增用户：追加不存在的成员，已存在的不重复
		It("append new users and skip existing ones", func() {
			users := []bkmapi.UserGroupUser{
				{ID: "alice", Type: "user", DisplayName: "alice"},
				{ID: "bk_biz_maintainer", Type: "group", DisplayName: "运维人员"},
			}
			got, changed := ApplyMemberDiff(users, []string{"bob", "alice"}, nil)

			Expect(changed).To(BeTrue())
			Expect(got).To(HaveLen(3))
			Expect(got[2]).To(Equal(bkmapi.UserGroupUser{
				ID: "bob", Type: "user", DisplayName: "bob",
			}))
		})

		// 移除用户：仅移除 type=user，保留 type=group 默认成员
		It("remove user-type members and keep group-type defaults", func() {
			users := []bkmapi.UserGroupUser{
				{ID: "alice", Type: "user"},
				{ID: "bob", Type: "user"},
				{ID: "bk_biz_maintainer", Type: "group"},
			}
			got, changed := ApplyMemberDiff(users, nil, []string{"alice", "bk_biz_maintainer"})

			Expect(changed).To(BeTrue())
			Expect(got).To(HaveLen(2))
			Expect(got[0].ID).To(Equal("bob"))
			// group 类型不受 remove 影响
			Expect(got[1].ID).To(Equal("bk_biz_maintainer"))
		})

		// 冲突场景：同一用户同时出现在 add 和 remove 中，remove 优先
		It("remove takes priority over add on conflict", func() {
			users := []bkmapi.UserGroupUser{
				{ID: "alice", Type: "user"},
			}
			got, changed := ApplyMemberDiff(users, []string{"alice"}, []string{"alice"})

			Expect(changed).To(BeTrue())
			Expect(got).To(BeEmpty())
		})

		// 空操作：无增无删时列表不变
		It("no-op when both add and remove are nil", func() {
			users := []bkmapi.UserGroupUser{
				{ID: "alice", Type: "user"},
			}
			got, changed := ApplyMemberDiff(users, nil, nil)

			Expect(changed).To(BeFalse())
			Expect(got).To(HaveLen(1))
		})
	})

	Describe("resolveBkMonitorProjectID", func() {
		It("rejects whitespace padded project ids", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = " -100 "

			_, err := ws.ResolveBkMonitorProjectID()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bkMonitorProjectID must be a valid int64"))
		})
	})

	// ==================== GetByEnv ====================

	Describe("GetByEnv", func() {
		// client 工厂失败时错误被正确包装上抛
		It("return wrapped error when client factory fails", func() {
			svc := &UserGroupService{
				newClient: func(_ string) (bkmapi.Client, error) {
					return nil, errors.New("mock client error")
				},
			}
			detail, err := svc.GetByEnv(context.Background(), 123, "test-1", "tester")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("new bkmonitor client"))
			Expect(detail).To(BeNil())
		})
	})

	// ==================== SyncMembersForEnvWithRetry ====================

	Describe("SyncMembersForEnvWithRetry", func() {
		// --- 公共测试辅助 ---

		buildWs := func() *workspace.Workspace {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			return ws
		}

		matchedGroups := func() []*bkmapi.UserGroup {
			return []*bkmapi.UserGroup{
				{ID: 9001, BkBizID: -100, Name: BuildUserGroupName("test-1")},
			}
		}

		detail := func() *bkmapi.UserGroupDetail {
			d := &bkmapi.UserGroupDetail{}
			d.ID = 9001
			d.BkBizID = -100
			d.Name = BuildUserGroupName("test-1")
			d.DutyArranges = []bkmapi.DutyArrange{{
				Users: []bkmapi.UserGroupUser{
					{ID: "bk_biz_maintainer", Type: "group"},
				},
			}}
			return d
		}

		mockClient := &bkmapi.ApiClient{}
		factory := func(_ string) (bkmapi.Client, error) { return mockClient, nil }

		// mock time.Sleep 避免真实等待
		BeforeEach(func() {
			mockey.Mock(time.Sleep).To(func(_ time.Duration) {}).Build()
		})
		AfterEach(func() {
			mockey.UnPatchAll()
		})

		// 告警组已存在，首次即成功保存成员
		It("save members on first attempt when group exists", func() {
			mockey.PatchConvey("first attempt success", GinkgoT(), func() {
				var saveCalls int32
				mockey.Mock((*bkmapi.ApiClient).SearchUserGroups).
					Return(matchedGroups(), nil).Build()
				mockey.Mock((*bkmapi.ApiClient).SearchUserGroupDetail).
					Return(detail(), nil).Build()
				mockey.Mock((*bkmapi.ApiClient).SaveUserGroup).To(
					func(_ *bkmapi.ApiClient, _ context.Context, _ *bkmapi.SaveUserGroupReq,
					) (*bkmapi.UserGroupDetail, error) {
						atomic.AddInt32(&saveCalls, 1)
						return detail(), nil
					}).Build()

				svc := &UserGroupService{
					newClient: factory,
					permMgr:   &perm.StubAllowAnyManager{},
				}
				svc.SyncMembersForEnvWithRetry(
					context.Background(), buildWs(), "test-1", "tester",
				)

				Expect(atomic.LoadInt32(&saveCalls)).To(Equal(int32(1)))
			})
		})

		// 告警组尚未创建，前两次 NotFound 后重试成功
		It("retry on NotFound and succeed on later attempt", func() {
			mockey.PatchConvey("notfound then success", GinkgoT(), func() {
				var searchCalls, saveCalls int32
				mockey.Mock((*bkmapi.ApiClient).SearchUserGroups).To(
					func(_ *bkmapi.ApiClient, _ context.Context, _ *bkmapi.SearchUserGroupsReq,
					) ([]*bkmapi.UserGroup, error) {
						n := atomic.AddInt32(&searchCalls, 1)
						if n < 3 {
							return nil, nil
						}
						return matchedGroups(), nil
					}).Build()
				mockey.Mock((*bkmapi.ApiClient).SearchUserGroupDetail).
					Return(detail(), nil).Build()
				mockey.Mock((*bkmapi.ApiClient).SaveUserGroup).To(
					func(_ *bkmapi.ApiClient, _ context.Context, _ *bkmapi.SaveUserGroupReq,
					) (*bkmapi.UserGroupDetail, error) {
						atomic.AddInt32(&saveCalls, 1)
						return detail(), nil
					}).Build()

				svc := &UserGroupService{
					newClient: factory,
					permMgr:   &perm.StubAllowAnyManager{},
				}
				svc.SyncMembersForEnvWithRetry(
					context.Background(), buildWs(), "test-1", "tester",
				)

				Expect(atomic.LoadInt32(&searchCalls)).To(Equal(int32(3)))
				Expect(atomic.LoadInt32(&saveCalls)).To(Equal(int32(1)))
			})
		})

		// 遇到非 NotFound 错误时立即终止
		It("stop retrying immediately on non-NotFound error", func() {
			mockey.PatchConvey("non-notfound error stops retry", GinkgoT(), func() {
				var searchCalls, saveCalls int32
				mockey.Mock((*bkmapi.ApiClient).SearchUserGroups).To(
					func(_ *bkmapi.ApiClient, _ context.Context, _ *bkmapi.SearchUserGroupsReq,
					) ([]*bkmapi.UserGroup, error) {
						atomic.AddInt32(&searchCalls, 1)
						return nil, errors.New("mock network error")
					}).Build()
				mockey.Mock((*bkmapi.ApiClient).SaveUserGroup).To(
					func(_ *bkmapi.ApiClient, _ context.Context, _ *bkmapi.SaveUserGroupReq,
					) (*bkmapi.UserGroupDetail, error) {
						atomic.AddInt32(&saveCalls, 1)
						return nil, nil
					}).Build()

				svc := &UserGroupService{
					newClient: factory,
					permMgr:   &perm.StubAllowAnyManager{},
				}
				svc.SyncMembersForEnvWithRetry(
					context.Background(), buildWs(), "test-1", "tester",
				)

				Expect(atomic.LoadInt32(&searchCalls)).To(Equal(int32(1)))
				Expect(atomic.LoadInt32(&saveCalls)).To(Equal(int32(0)))
			})
		})
	})
})
