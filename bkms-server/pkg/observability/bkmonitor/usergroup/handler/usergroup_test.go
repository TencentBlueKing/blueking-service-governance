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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	bkmusergroup "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
	bkmserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup/serializer"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

const testUserGroupBkBizID int64 = -2001

func buildWorkspace() *workspace.Workspace {
	ws := &workspace.Workspace{ID: "ws-1"}
	ws.BkSystems.BkMonitorProjectID = "-2001"
	return ws
}

var _ = Describe("Usergroup Handler", func() {
	var (
		router                 *gin.Engine
		ws                     *workspace.Workspace
		validateWorkspaceCalls int
		expectedWorkspacePerm  *ginperm.Type
	)

	buildRequest := func(method, path string, body any) *http.Request {
		var payload []byte
		if body != nil {
			var err error
			payload, err = json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())
		}

		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "tester"}))
		return req
	}

	buildCurrentSaveBody := func(name, memberID string) map[string]any {
		return map[string]any{
			"name":     name,
			"channels": []string{"user"},
			"alertNotice": []map[string]any{{
				"time_range": "00:00--23:59",
				"notify_config": []map[string]any{
					{"level": 1, "type": []string{}, "notice_ways": []map[string]any{{"name": "weixin"}}},
					{"level": 2, "type": []string{}, "notice_ways": []map[string]any{{"name": "mail"}}},
					{"level": 3, "type": []string{}, "notice_ways": []map[string]any{{"name": "sms"}}},
				},
			}},
			"actionNotice": []map[string]any{{
				"time_range": "00:00--23:59",
				"notify_config": []map[string]any{
					{"phase": 1, "type": []string{}, "notice_ways": []map[string]any{{"name": "weixin"}}},
					{"phase": 2, "type": []string{}, "notice_ways": []map[string]any{{"name": "mail"}}},
					{"phase": 3, "type": []string{}, "notice_ways": []map[string]any{{"name": "sms"}}},
				},
			}},
			"users": []map[string]any{
				{"id": memberID, "type": "user", "display_name": memberID},
			},
		}
	}

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ws = buildWorkspace()
		validateWorkspaceCalls = 0
		expectedWorkspacePerm = nil
		bkmapi.ResetStubStateForTest()
		router = gin.New()
		router.Use(bkerrs.ErrorHandler())
		bkmusergroup.Register(router.Group(""), New(&storereg.Registry{}, bkmusergroup.New()))
		mockey.Mock(ginperm.ValidateWorkspaceByID).To(
			func(
				_ context.Context,
				_ *storereg.Registry,
				workspaceID string,
				permType ginperm.Type,
			) (*workspace.Workspace, error) {
				validateWorkspaceCalls++
				Expect(workspaceID).To(Equal("ws-1"))
				if expectedWorkspacePerm != nil {
					Expect(permType).To(Equal(*expectedWorkspacePerm))
				}
				return ws, nil
			},
		).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	Describe("main flows", func() {
		It("creates a user group from current payload", func() {
			permType := ginperm.TypeEdit
			expectedWorkspacePerm = &permType
			client := bkmapi.NewStub("tester")
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			body := buildCurrentSaveBody("ops", "tester")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodPost, "/workspaces/ws-1/bkmonitor/user-groups", body))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.SaveUserGroupResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Name).To(Equal("ops"))
			Expect(resp.Data.BkBizID).To(Equal(testUserGroupBkBizID))
			Expect(resp.Data.Channels).To(ContainElement("user"))
			Expect(resp.Data.DutyArranges).NotTo(BeEmpty())
			Expect(resp.Data.DutyArranges[0].NeedRotation).To(BeFalse())
			Expect(resp.Data.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(resp.Data.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(resp.Data.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(resp.Data.DutyArranges[0].Backups).To(Equal([]map[string]any{}))
			Expect(resp.Data.DutyArranges[0].Order).To(Equal(int64(1)))
			Expect(resp.Data.DutyArranges[0].GroupType).To(Equal("specified"))
			Expect(resp.Data.DutyArranges[0].GroupNumber).To(Equal(int64(0)))
			Expect(resp.Data.DutyArranges[0].Users).To(HaveLen(1))
			Expect(resp.Data.DutyArranges[0].Users[0].ID).To(Equal("tester"))
			Expect(resp.Data.AlertNotice).To(HaveLen(1))
			Expect(resp.Data.AlertNotice[0].NotifyConfig).To(HaveLen(3))
			Expect(resp.Data.ActionNotice).To(HaveLen(1))
			Expect(resp.Data.ActionNotice[0].NotifyConfig).To(HaveLen(3))
		})

		It("lists user groups", func() {
			permType := ginperm.TypeView
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups", nil))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.ListUserGroupsResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Count).To(Equal(int64(1)))
			Expect(resp.Data.Results).To(HaveLen(1))
			Expect(resp.Data.Results[0].Name).To(Equal("ops"))
		})

		It("lists all user groups with current API shape", func() {
			permType := ginperm.TypeView
			expectedWorkspacePerm = &permType
			client := bkmapi.NewStub("tester")
			var err error
			for i, name := range []string{"ops-a", "ops-b", "ops-c", "ops-d", "ops-e", "ops-f"} {
				_, err = client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
					ID:           lo.ToPtr(int64(1001 + i)),
					BkBizID:      testUserGroupBkBizID,
					Name:         name,
					Channels:     []string{"user"},
					AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
					ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
					Operator:     "tester",
				})
				Expect(err).NotTo(HaveOccurred())
			}
			_, err = client.SaveUserGroup(context.Background(), &bkmapi.SaveUserGroupReq{
				ID:           lo.ToPtr(int64(2000)),
				BkBizID:      testUserGroupBkBizID,
				Name:         "other",
				Channels:     []string{"user"},
				AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
				ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups", nil))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.ListUserGroupsResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Count).To(Equal(int64(7)))
			Expect(resp.Data.Results).To(HaveLen(7))
			Expect(lo.Map(resp.Data.Results, func(item *bkmapi.UserGroup, _ int) string {
				return item.Name
			})).To(ContainElements("ops-a", "ops-b", "ops-c", "ops-d", "ops-e", "ops-f", "other"))
		})

		It("gets a user group detail", func() {
			permType := ginperm.TypeView
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups/1001", nil))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.GetUserGroupResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal(int64(1001)))
			Expect(resp.Data.Name).To(Equal("ops"))
		})

		It("creates a user group", func() {
			permType := ginperm.TypeEdit
			expectedWorkspacePerm = &permType
			client := bkmapi.NewStub("tester")
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			body := map[string]any{
				"name":     "ops",
				"channels": []string{"user"},
				"alertNotice": []map[string]any{
					{"time_range": "00:00--23:59"},
				},
				"actionNotice": []map[string]any{
					{"time_range": "00:00--23:59"},
				},
				"users": []map[string]any{
					{"id": "tester", "type": "user", "display_name": "tester"},
				},
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodPost, "/workspaces/ws-1/bkmonitor/user-groups", body))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.SaveUserGroupResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Name).To(Equal("ops"))
			Expect(resp.Data.BkBizID).To(Equal(testUserGroupBkBizID))
		})

		It("updates a user group", func() {
			permType := ginperm.TypeEdit
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			body := map[string]any{
				"name":     "ops-updated",
				"channels": []string{"user"},
				"alertNotice": []map[string]any{
					{"time_range": "00:00--23:59"},
				},
				"actionNotice": []map[string]any{
					{"time_range": "00:00--23:59"},
				},
				"users": []map[string]any{
					{"id": "tester-2", "type": "user", "display_name": "tester-2"},
				},
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodPut, "/workspaces/ws-1/bkmonitor/user-groups/1001", body))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.SaveUserGroupResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal(int64(1001)))
			Expect(resp.Data.Name).To(Equal("ops-updated"))
		})

		It("updates a user group from current payload", func() {
			permType := ginperm.TypeEdit
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			body := buildCurrentSaveBody("ops-updated", "tester-2")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodPut, "/workspaces/ws-1/bkmonitor/user-groups/1001", body))

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp bkmserializer.SaveUserGroupResp
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal(int64(1001)))
			Expect(resp.Data.Name).To(Equal("ops-updated"))
			Expect(resp.Data.DutyArranges).NotTo(BeEmpty())
			Expect(resp.Data.DutyArranges[0].NeedRotation).To(BeFalse())
			Expect(resp.Data.DutyArranges[0].DutyTime).To(Equal([]map[string]any{}))
			Expect(resp.Data.DutyArranges[0].HandoffTime).To(Equal(map[string]any{}))
			Expect(resp.Data.DutyArranges[0].DutyUsers).To(Equal([][]bkmapi.UserGroupUser{}))
			Expect(resp.Data.DutyArranges[0].Backups).To(Equal([]map[string]any{}))
			Expect(resp.Data.DutyArranges[0].Order).To(Equal(int64(1)))
			Expect(resp.Data.DutyArranges[0].GroupType).To(Equal("specified"))
			Expect(resp.Data.DutyArranges[0].GroupNumber).To(Equal(int64(0)))
			Expect(resp.Data.DutyArranges[0].Users).To(HaveLen(1))
			Expect(resp.Data.DutyArranges[0].Users[0].ID).To(Equal("tester-2"))
		})

		It("deletes a user group", func() {
			permType := ginperm.TypeEdit
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodDelete, "/workspaces/ws-1/bkmonitor/user-groups/1001", nil))

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(MatchJSON(`{}`))
		})
	})

	Describe("error paths", func() {
		It("returns invalid request when create body is malformed", func() {
			body := map[string]any{
				"channels": []string{"user"},
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodPost, "/workspaces/ws-1/bkmonitor/user-groups", body))

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(validateWorkspaceCalls).To(Equal(0))
			var resp bkerrs.GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Error.Code).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(resp.Error.Message).To(ContainSubstring("bind json request"))
		})

		It("returns invalid request when group id is malformed", func() {
			rec := httptest.NewRecorder()
			router.ServeHTTP(
				rec,
				buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups/not-a-number", nil),
			)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			var resp bkerrs.GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Error.Code).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(resp.Error.Message).To(ContainSubstring("bind uri request"))
		})

		It("maps generic service errors to internal server error", func() {
			permType := ginperm.TypeView
			expectedWorkspacePerm = &permType
			mockey.Mock(bkmapi.NewMonitorClient).Return(nil, errors.New("boom")).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups", nil))

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			var resp bkerrs.GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Error.Code).To(Equal(bkerrs.ErrCodeInternalServerError))
		})

		It("maps user groups outside workspace to not found", func() {
			permType := ginperm.TypeView
			expectedWorkspacePerm = &permType
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
			mockey.Mock(bkmapi.NewMonitorClient).Return(client, nil).Build()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, buildRequest(http.MethodGet, "/workspaces/ws-1/bkmonitor/user-groups/1001", nil))

			Expect(rec.Code).To(Equal(http.StatusNotFound))
			var resp bkerrs.GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Error.Code).To(Equal(bkerrs.ErrCodeNotFound))
		})
	})

	Describe("wrapServiceError", func() {
		It("maps validator errors to invalid argument", func() {
			type invalidInput struct {
				Name string `validate:"required"`
			}

			rawErr := validator.New(validator.WithRequiredStructEnabled()).Struct(&invalidInput{})
			Expect(rawErr).To(HaveOccurred())

			wrapped := new(Handler).wrapServiceError(rawErr, "save user group")
			var bkErr *bkerrs.Error
			Expect(errors.As(wrapped, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		})
	})
})
