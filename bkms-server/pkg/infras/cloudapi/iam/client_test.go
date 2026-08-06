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

package iam

import (
	"context"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

var _ = Describe("BKIAMClient", func() {
	var (
		mockCtrl  *gomock.Controller
		mockOp    *MockOperation
		ctx       context.Context
		originCfg *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockOp = NewMockOperation(mockCtrl)
		ctx = context.Background()

		// IAM client 内部依赖 config.G 中的 IAMSystemIDs / BkApp / BkPlatUrls
		// 在测试前设置一份最小可用配置，并在测试结束后还原。
		originCfg = config.G
		config.G = &config.Config{
			BkApp:          config.BkAppConfig{Code: "bkms", Secret: "secret"},
			BkPlatUrls:     config.BkPlatUrlsConfig{BkApiUrlTmpl: "http://{api_name}.apigw.example.com"},
			BkApiStages:    config.BkApiStagesConfig{BkIAM: "test"},
			BkIAMSystemIDs: config.BkIAMSystemIDsConfig{Bkms: "bkms"},
		}
	})

	AfterEach(func() {
		config.G = originCfg
		mockCtrl.Finish()
	})

	// expectStandardChain 预设最常见的链式调用：
	// op.SetContext(ctx).SetResult(&resp).Request() 三连，最终 Request 返回 (nil, nil)。
	// fillResp 用于在 SetResult 阶段往 result 写入桩数据。
	expectStandardChain := func(fillResp func(target any)) {
		mockOp.EXPECT().FullName().Return("test_operation").AnyTimes()
		mockOp.EXPECT().SetContext(gomock.Any()).Return(mockOp)
		mockOp.EXPECT().SetResult(gomock.Any()).DoAndReturn(func(target any) define.Operation {
			fillResp(target)
			return mockOp
		})
		mockOp.EXPECT().Request().Return(nil, nil)
	}

	newClientFromMock := func() IAMClient {
		client, err := newIAMClient(MockBkApiClient{op: mockOp})
		Expect(err).To(BeNil())
		return client
	}

	Describe("gateway url", func() {
		It("builds iam gateway url with stage", func() {
			gatewayURL, err := BuildIAMGatewayURL("http://{api_name}.apigw.example.com", "test")
			Expect(err).To(BeNil())
			Expect(gatewayURL).To(Equal("http://bk-iam.apigw.example.com/test"))
		})

		It("uses default stage when stage is empty", func() {
			gatewayURL, err := BuildIAMGatewayURL("http://{api_name}.apigw.example.com", "")
			Expect(err).To(BeNil())
			Expect(gatewayURL).To(Equal("http://bk-iam.apigw.example.com/prod"))
		})
	})

	Describe("grade manager", func() {
		It("creates grade manager and returns id", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
				resp.Message = "ok"
				resp.Data = map[string]any{"id": 1000}
			})

			id, err := newClientFromMock().CreateGradeManager(
				ctx, "test-grade-manager", "admin", []string{"foo"}, nil,
			)
			Expect(err).To(BeNil())
			Expect(*id).To(Equal(1000))
		})

		It("returns existing grade manager id when create conflicts", func() {
			const existingGradeManagerID = 1000

			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = ConflictCode
				resp.Message = "already exists"
			})
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
				resp.Message = "ok"
				resp.Data = map[string]any{
					"count": 1,
					"results": []map[string]any{
						{"id": existingGradeManagerID, "name": "test-grade-manager"},
					},
				}
			})

			id, err := newClientFromMock().CreateGradeManager(
				ctx, "test-grade-manager", "admin", []string{"foo"}, nil,
			)
			Expect(err).To(BeNil())
			Expect(*id).To(Equal(existingGradeManagerID))
		})

		It("gets grade manager id by name", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
				resp.Message = "ok"
				resp.Data = map[string]any{
					"count": 2,
					"results": []map[string]any{
						{"id": 500, "name": "foo-grade-manager"},
						{"id": 1000, "name": "bar-grade-manager"},
					},
				}
			})

			id, err := newClientFromMock().GetGradeManagerByName(ctx, "bar-grade-manager")
			Expect(err).To(BeNil())
			Expect(*id).To(Equal(1000))
		})

		It("returns error when grade manager name not found", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
				resp.Message = "ok"
				resp.Data = map[string]any{
					"count":   0,
					"results": []map[string]any{},
				}
			})

			_, err := newClientFromMock().GetGradeManagerByName(ctx, "missing-grade-manager")
			Expect(err).NotTo(BeNil())
		})

		It("updates grade manager successfully", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().UpdateGradeManager(ctx, 1000, "name", "desc", nil)
			Expect(err).To(BeNil())
		})

		It("deletes grade manager successfully", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().DeleteGradeManager(ctx, 500)
			Expect(err).To(BeNil())
		})

		It("returns error when delete grade manager fails", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 1000
				resp.Message = "delete failed"
			})

			err := newClientFromMock().DeleteGradeManager(ctx, 500)
			Expect(err).NotTo(BeNil())
		})

		It("adds grade manager members", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().AddGradeManagerMembers(ctx, 1000, []string{"foo", "bar"})
			Expect(err).To(BeNil())
		})

		It("deletes grade manager members", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().DeleteGradeManagerMembers(ctx, 1000, []string{"foo", "bar"})
			Expect(err).To(BeNil())
		})
	})

	Describe("user group", func() {
		It("creates user groups and returns ids in order", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.CreateUserGroupsResp)
				resp.Code = 0
				resp.Data = []int{3}
			})

			groups, err := newClientFromMock().CreateUserGroups(
				ctx,
				1000,
				types.UserGroupParam{Name: "app-manager", Readonly: true},
			)
			Expect(err).To(BeNil())
			Expect(groups).To(HaveLen(1))
			Expect(groups[0].Name).To(Equal("app-manager"))
			Expect(groups[0].ID).To(Equal(3))
			Expect(groups[0].Readonly).To(BeTrue())
		})

		It("returns error when user group id count mismatches", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.CreateUserGroupsResp)
				resp.Code = 0
				resp.Data = []int{3, 4}
			})

			groups, err := newClientFromMock().CreateUserGroups(
				ctx, 1000, types.UserGroupParam{Name: "app-manager", Readonly: true},
			)
			Expect(err).NotTo(BeNil())
			Expect(groups).To(BeNil())
		})

		It("deletes user group", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().DeleteUserGroup(ctx, 1000)
			Expect(err).To(BeNil())
		})

		It("grants user group policies", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().GrantUserGroupPolicies(ctx, 1, []types.AuthorizationScope{{}})
			Expect(err).To(BeNil())
		})

		It("adds user group members with never-expire timestamp", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().AddUserGroupMembers(
				ctx, 1000, []string{"foo", "bar"}, NeverExpireTimestamp,
			)
			Expect(err).To(BeNil())
		})

		It("deletes user group members", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
			})

			err := newClientFromMock().DeleteUserGroupMembers(ctx, 1000, []string{"foo", "bar"})
			Expect(err).To(BeNil())
		})

		It("lists user group members", func() {
			expectStandardChain(func(target any) {
				resp := target.(*types.Resp)
				resp.Code = 0
				resp.Data = map[string]any{
					"count": 2,
					"results": []map[string]any{
						{"id": "foo-user-group-member"},
						{"id": "bar-user-group-member"},
					},
				}
			})

			members, err := newClientFromMock().ListUserGroupMembers(ctx, 1000)
			Expect(err).To(BeNil())
			Expect(members).To(HaveLen(2))
			Expect(members[0].ID).To(Equal("foo-user-group-member"))
		})
	})
})
