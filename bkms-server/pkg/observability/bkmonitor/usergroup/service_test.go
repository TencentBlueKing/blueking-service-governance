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
	stderrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

const testUserGroupBkBizID int64 = -2001

func buildWorkspace() *workspace.Workspace {
	ws := &workspace.Workspace{ID: "ws-1"}
	ws.BkSystems.BkMonitorProjectID = "-2001"
	return ws
}

func buildWorkspaceWithBkMonitorProjectID(projectID string) *workspace.Workspace {
	ws := &workspace.Workspace{ID: "ws-1"}
	ws.BkSystems.BkMonitorProjectID = projectID
	return ws
}

func buildSaveParams() *SaveParams {
	return &SaveParams{
		ID:       1001,
		Name:     "edited",
		Channels: []string{"user"},
		AlertNotice: []bkmapi.AlertNotice{
			{TimeRange: "00:00--23:59"},
		},
		ActionNotice: []bkmapi.ActionNotice{
			{TimeRange: "00:00--23:59"},
		},
		Users: []bkmapi.UserGroupUser{{
			ID:          "tester",
			Type:        "user",
			DisplayName: "tester",
		}},
		Operator: "tester",
	}
}

var _ = Describe("Usergroup Service", func() {
	var (
		ws  *workspace.Workspace
		svc *Service
	)

	BeforeEach(func() {
		ws = buildWorkspace()
		bkmapi.ResetStubStateForTest()
	})

	Describe("List", func() {
		It("uses workspace bk biz id when listing user groups", func() {
			client := bkmapi.NewStub("tester")
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "ops",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			got, err := svc.List(context.Background(), ws, "tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].ID).To(Equal(int64(1001)))
			Expect(got[0].BkBizID).To(Equal(testUserGroupBkBizID))
			Expect(got[0].Name).To(Equal("ops"))
		})

		It("wraps client factory errors", func() {
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return nil, errors.New("boom") },
			}

			_, err := svc.List(context.Background(), ws, "tester")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("new bkmonitor client"))
		})
	})

	Describe("FindByName", func() {
		It("uses workspace bk biz id and group name when searching user groups", func() {
			client := bkmapi.NewStub("tester")
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "ops",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1002)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "other",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			got, err := svc.FindByName(context.Background(), ws, "ops", "tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
			Expect(got.ID).To(Equal(int64(1001)))
			Expect(got.Name).To(Equal("ops"))
		})

		It("returns nil when the target user group does not exist", func() {
			client := bkmapi.NewStub("tester")
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			got, err := svc.FindByName(context.Background(), ws, "missing", "tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeNil())
		})
	})

	Describe("resolve bkmonitor space id", func() {
		DescribeTable(
			"rejects invalid workspace monitor project ids",
			func(ws *workspace.Workspace, expected string) {
				_, err := ws.ResolveBkMonitorProjectID()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expected))
			},
			Entry("nil workspace", (*workspace.Workspace)(nil), "workspace is nil"),
			Entry(
				"empty project id",
				buildWorkspaceWithBkMonitorProjectID(""),
				"bkMonitorProjectID is empty",
			),
			Entry(
				"whitespace padded project id",
				buildWorkspaceWithBkMonitorProjectID(" -2001 "),
				"bkMonitorProjectID must be a valid int64",
			),
			Entry("non numeric project id", buildWorkspaceWithBkMonitorProjectID("not-a-number"),
				"bkMonitorProjectID must be a valid int64"),
		)

		DescribeTable(
			"accepts valid int64 monitor space ids",
			func(raw string, expected int64) {
				id, err := buildWorkspaceWithBkMonitorProjectID(raw).ResolveBkMonitorProjectID()

				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(expected))
			},
			Entry("negative project id", "-2001", int64(-2001)),
			Entry("zero project id", "0", int64(0)),
			Entry("positive project id", "2001", int64(2001)),
		)
	})

	Describe("Get", func() {
		It("rejects cross-biz user groups", func() {
			client := bkmapi.NewStub("tester")
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1001)),
				BkBizID:      -9999,
				Name:         "foreign",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			_, err = svc.Get(context.Background(), ws, 1001, "tester")

			Expect(err).To(HaveOccurred())
			Expect(stderrors.Is(err, ErrUserGroupNotInWorkspace)).To(BeTrue())
		})
	})

	Describe("Save", func() {
		It("passes workspace bk biz id and default timezone to remote save", func() {
			client := bkmapi.NewStub("tester")
			factoryCalls := 0
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "edited",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) {
					factoryCalls++
					return client, nil
				},
			}

			detail, err := svc.Save(context.Background(), ws, buildSaveParams())

			Expect(err).NotTo(HaveOccurred())
			Expect(detail).NotTo(BeNil())
			Expect(detail.BkBizID).To(Equal(testUserGroupBkBizID))
			Expect(detail.Timezone).To(Equal(saveUserGroupDefaultTimezone))
			Expect(detail.DutyArranges).To(HaveLen(1))
			Expect(detail.DutyArranges[0].GroupType).To(Equal("specified"))
			Expect(detail.DutyArranges[0].NeedRotation).To(BeFalse())
			Expect(detail.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(detail.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(detail.DutyArranges[0].Backups).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].Users).To(HaveLen(1))
			Expect(detail.DutyArranges[0].Users[0].ID).To(Equal("tester"))
			Expect(factoryCalls).To(Equal(1))
		})

		It("normalizes nested notice slices before calling bkmonitor", func() {
			params := &SaveParams{
				Name:     "ops",
				Channels: []string{"user"},
				AlertNotice: []bkmapi.AlertNotice{{
					TimeRange: "00:00--23:59",
					NotifyConfig: []bkmapi.AlertNoticeConfig{{
						Level:      1,
						NoticeWays: []bkmapi.NoticeWay{{Name: "weixin"}},
					}},
				}},
				ActionNotice: []bkmapi.ActionNotice{{
					TimeRange: "00:00--23:59",
					NotifyConfig: []bkmapi.ActionNoticeConfig{{
						Phase:      1,
						NoticeWays: []bkmapi.NoticeWay{{Name: "mail"}},
					}},
				}},
				Users: []bkmapi.UserGroupUser{{
					ID:   "tester",
					Type: "user",
				}},
				Operator: "tester",
			}

			req := buildSaveUserGroupReq(testUserGroupBkBizID, nil, params)

			Expect(req.AlertNotice[0].NotifyConfig[0].Type).To(Equal([]string{}))
			Expect(req.AlertNotice[0].NotifyConfig[0].NoticeWays).To(HaveLen(1))
			Expect(req.AlertNotice[0].NotifyConfig[0].NoticeWays[0].Receivers).To(Equal([]string{}))
			Expect(req.ActionNotice[0].NotifyConfig[0].Type).To(Equal([]string{}))
			Expect(req.ActionNotice[0].NotifyConfig[0].NoticeWays).To(HaveLen(1))
			Expect(req.ActionNotice[0].NotifyConfig[0].NoticeWays[0].Receivers).To(Equal([]string{}))
		})

		It("rewrites existing groups into the simplified monitor-compatible payload on update", func() {
			client := bkmapi.NewStub("tester")
			factoryCalls := 0
			groupID := int64(1001)
			effectiveTime := "2026-08-06 00:00:00"
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           &groupID,
				BkBizID:      testUserGroupBkBizID,
				Name:         "ops",
				Timezone:     "UTC",
				Channels:     []string{"user"},
				Desc:         "before",
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				DutyArranges: []bkmapi.DutyArrange{{
					ID:            88,
					Hash:          "arrange-hash",
					GroupType:     "specified",
					GroupNumber:   0,
					UserGroupID:   groupID,
					NeedRotation:  false,
					DutyTime:      []map[string]any{},
					EffectiveTime: &effectiveTime,
					HandoffTime:   map[string]any{},
					DutyUsers:     [][]bkmapi.UserGroupUser{},
					Users: []bkmapi.UserGroupUser{{
						ID:          "legacy",
						Type:        "user",
						DisplayName: "legacy",
					}},
					Backups: []map[string]any{},
					Order:   1,
				}},
				MentionList: []bkmapi.UserGroupUser{{
					ID:          "observer",
					Type:        "user",
					DisplayName: "observer",
				}},
				DutyNotice:  &bkmapi.DutyNotice{},
				MentionType: 2,
				Path:        "biz/path",
				Operator:    "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) {
					factoryCalls++
					return client, nil
				},
			}

			detail, err := svc.Save(context.Background(), ws, buildSaveParams())

			Expect(err).NotTo(HaveOccurred())
			Expect(detail.Timezone).To(Equal(saveUserGroupDefaultTimezone))
			Expect(detail.Path).To(BeEmpty())
			Expect(detail.MentionType).To(Equal(int64(0)))
			Expect(detail.MentionList).To(BeNil())
			Expect(detail.DutyArranges).To(HaveLen(1))
			Expect(detail.DutyArranges[0].ID).To(Equal(int64(0)))
			Expect(detail.DutyArranges[0].Hash).To(BeEmpty())
			Expect(detail.DutyArranges[0].UserGroupID).To(Equal(int64(0)))
			Expect(detail.DutyArranges[0].EffectiveTime).To(BeNil())
			Expect(detail.DutyArranges[0].NeedRotation).To(BeFalse())
			Expect(detail.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(detail.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(detail.DutyArranges[0].Backups).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].GroupType).To(Equal("specified"))
			Expect(detail.DutyArranges[0].GroupNumber).To(Equal(int64(0)))
			Expect(detail.DutyArranges[0].Users).To(HaveLen(1))
			Expect(detail.DutyArranges[0].Users[0].ID).To(Equal("tester"))
			Expect(factoryCalls).To(Equal(1))
		})

		It("rewrites existing duty groups into the simplified monitor-compatible payload on update", func() {
			client := bkmapi.NewStub("tester")
			groupID := int64(1001)
			hitFirstDuty := true
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           &groupID,
				BkBizID:      testUserGroupBkBizID,
				Name:         "duty-group",
				Timezone:     "UTC",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				NeedDuty:     true,
				DutyRules:    []int64{9001},
				DutyNotice: &bkmapi.DutyNotice{
					HitFirstDuty: &hitFirstDuty,
				},
				DutyArranges: []bkmapi.DutyArrange{{
					GroupType:    "auto",
					GroupNumber:  2,
					NeedRotation: true,
					DutyTime:     []map[string]any{{"type": "daily"}},
					HandoffTime:  map[string]any{"time": "09:00"},
					DutyUsers: [][]bkmapi.UserGroupUser{{
						{ID: "legacy-duty", Type: "user", DisplayName: "legacy-duty"},
					}},
					Backups: []map[string]any{},
					Order:   1,
				}},
				Operator: "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			detail, err := svc.Save(context.Background(), ws, buildSaveParams())

			Expect(err).NotTo(HaveOccurred())
			Expect(detail.NeedDuty).To(BeFalse())
			Expect(detail.DutyRules).To(BeNil())
			Expect(detail.DutyNotice).To(BeNil())
			Expect(detail.DutyArranges).To(HaveLen(1))
			Expect(detail.DutyArranges[0].GroupType).To(Equal("specified"))
			Expect(detail.DutyArranges[0].GroupNumber).To(Equal(int64(0)))
			Expect(detail.DutyArranges[0].NeedRotation).To(BeFalse())
			Expect(detail.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(detail.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(detail.DutyArranges[0].Backups).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].Users).To(HaveLen(1))
			Expect(detail.DutyArranges[0].Users[0].ID).To(Equal("tester"))
		})

		It("rewrites legacy duty-like groups into the simplified monitor-compatible payload on update", func() {
			client := bkmapi.NewStub("tester")
			groupID := int64(1001)
			dutyRuleID := int64(9001)
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           &groupID,
				BkBizID:      testUserGroupBkBizID,
				Name:         "legacy-duty-group",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				DutyArranges: []bkmapi.DutyArrange{{
					GroupType:    "specified",
					GroupNumber:  0,
					DutyRuleID:   &dutyRuleID,
					NeedRotation: false,
					DutyTime:     []map[string]any{{"type": "weekly"}},
					HandoffTime:  map[string]any{"hour": 9},
					DutyUsers: [][]bkmapi.UserGroupUser{{
						{ID: "legacy", Type: "user", DisplayName: "legacy"},
					}},
					Order: 1,
				}},
				Operator: "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) { return client, nil },
			}

			detail, err := svc.Save(context.Background(), ws, buildSaveParams())

			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DutyArranges).To(HaveLen(1))
			Expect(detail.DutyArranges[0].DutyRuleID).To(BeNil())
			Expect(detail.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(detail.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(detail.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(detail.DutyArranges[0].Users).To(HaveLen(1))
			Expect(detail.DutyArranges[0].Users[0].ID).To(Equal("tester"))
		})

		DescribeTable(
			"rejects invalid required fields before calling remote client",
			func(mutate func(*SaveParams), expected string) {
				client := bkmapi.NewStub("tester")
				factoryCalls := 0
				svc = &Service{
					newClient: func(string) (bkmapi.MonitorClient, error) {
						factoryCalls++
						return client, nil
					},
				}
				params := buildSaveParams()
				mutate(params)

				_, err := svc.Save(context.Background(), ws, params)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expected))
				Expect(factoryCalls).To(Equal(0))
			},
			Entry("empty name", func(params *SaveParams) {
				params.Name = ""
			}, "field Name failed on required"),
			Entry("missing channels", func(params *SaveParams) {
				params.Channels = nil
			}, "field Channels failed on min"),
			Entry("missing alert notice", func(params *SaveParams) {
				params.AlertNotice = nil
			}, "field AlertNotice failed on min"),
			Entry("missing action notice", func(params *SaveParams) {
				params.ActionNotice = nil
			}, "field ActionNotice failed on min"),
			Entry("missing users", func(params *SaveParams) {
				params.Users = nil
			}, "field Users failed on min"),
			Entry("empty operator", func(params *SaveParams) {
				params.Operator = ""
			}, "field Operator failed on required"),
		)

		It("rejects nil request before calling remote client", func() {
			client := bkmapi.NewStub("tester")
			factoryCalls := 0
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) {
					factoryCalls++
					return client, nil
				},
			}

			_, err := svc.Save(context.Background(), ws, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request is nil"))
			Expect(factoryCalls).To(Equal(0))
		})
	})

	Describe("Delete", func() {
		It("calls remote delete with workspace bk biz id", func() {
			client := bkmapi.NewStub("tester")
			factoryCalls := 0
			_, err := client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(1001)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "ops",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			svc = &Service{
				newClient: func(string) (bkmapi.MonitorClient, error) {
					factoryCalls++
					return client, nil
				},
			}

			err = svc.Delete(context.Background(), ws, 1001, "tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(factoryCalls).To(Equal(1))
			detail, err := client.SearchUserGroupDetail(
				context.Background(),
				&bkmapi.SearchUserGroupDetailReq{ID: 1001},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.Name).NotTo(Equal("ops"))
		})
	})
})
