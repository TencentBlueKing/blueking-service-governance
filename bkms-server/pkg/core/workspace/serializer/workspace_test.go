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

package serializer_test

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("Workspace serializer", func() {
	// -----------------------------------------------------------------------
	// FromModel 转换测试
	// -----------------------------------------------------------------------
	Describe("BkSystemsOutputObj FromModel", func() {
		It("maps all BkSystems fields to output object", func() {
			bk := workspace.BkSystems{
				BkCIProjectID:             "ci-123",
				BkCIProjectUID:            "uid-abc",
				BkBCSProjectID:            "bcs-456",
				BkBCSProjectCode:          "bkce",
				BkLogProjectID:            "log-789",
				BkMonitorProjectID:        "mon-012",
				BkRepoProjectID:           "repo-345",
				BkCCBizID:                 "999",
				Level2BizID:               "888",
				ObsProductID:              "obs-1",
				ObsProductName:            "观测产品",
				IsBoundExistedBKCIProject: true,
			}

			output := new(serializer.BkSystemsOutputObj).FromModel(bk)

			Expect(output).To(Equal(&serializer.BkSystemsOutputObj{
				BkCIProjectID:             "ci-123",
				BkCIProjectUID:            "uid-abc",
				BkBCSProjectID:            "bcs-456",
				BkBCSProjectCode:          "bkce",
				BkLogProjectID:            "log-789",
				BkMonitorProjectID:        "mon-012",
				BkRepoProjectID:           "repo-345",
				BkCCBizID:                 "999",
				Level2BizID:               "888",
				ObsProductID:              "obs-1",
				ObsProductName:            "观测产品",
				IsBoundExistedBKCIProject: true,
			}))
		})
	})

	Describe("WorkspaceInfoOutputObj FromModel", func() {
		It("maps workspace fields to info output object", func() {
			now := time.Now()
			ws := workspace.Workspace{
				ID:          "my-ws",
				DisplayName: "我的空间",
				Description: "描述",
				BkSystems: workspace.BkSystems{
					BkCIProjectID: "ci-1",
				},
				State:     workspace.StateReady,
				Creator:   "user1",
				CreatedAt: now,
				Updater:   "user2",
				UpdatedAt: now,
			}

			output := new(serializer.WorkspaceInfoOutputObj).FromModel(ws)

			Expect(output.ID).To(Equal("my-ws"))
			Expect(output.DisplayName).To(Equal("我的空间"))
			Expect(output.Description).To(Equal("描述"))
			Expect(output.State).To(Equal("Ready"))
			Expect(output.Creator).To(Equal("user1"))
			Expect(output.CreatedAt).To(Equal(now))
			Expect(output.Updater).To(Equal("user2"))
			Expect(output.UpdatedAt).To(Equal(now))
			Expect(output.BkSystems).NotTo(BeNil())
			Expect(output.BkSystems.BkCIProjectID).To(Equal("ci-1"))
		})
	})

	Describe("WorkspaceDetailOutputObj FromModel", func() {
		It("maps workspace and image registry to detail output", func() {
			now := time.Now()
			ws := workspace.Workspace{
				ID:                "ws-detail",
				DisplayName:       "详情空间",
				Description:       "详情描述",
				ImageRegistryType: registry.ImageRegistryTypeExternal,
				BkSystems: workspace.BkSystems{
					BkCIProjectID: "ci-2",
				},
				State:     workspace.StateDisabled,
				Creator:   "creator",
				CreatedAt: now,
				Updater:   "updater",
				UpdatedAt: now,
			}
			ir := &registry.ImageRegistry{
				Registry: "mirrors.tencent.com/bkpaas",
				Username: "admin",
				Password: "secret",
			}

			output := new(serializer.WorkspaceDetailOutputObj).FromModel(ws, ir)

			Expect(output.ID).To(Equal("ws-detail"))
			Expect(output.DisplayName).To(Equal("详情空间"))
			Expect(output.Description).To(Equal("详情描述"))
			Expect(output.ImageRegistryType).To(Equal("external"))
			Expect(output.State).To(Equal("Disabled"))
			Expect(output.Creator).To(Equal("creator"))
			Expect(output.Updater).To(Equal("updater"))
			Expect(output.BkSystems).NotTo(BeNil())
			Expect(output.BkSystems.BkCIProjectID).To(Equal("ci-2"))
			Expect(output.ImageRegistry).NotTo(BeNil())
			Expect(output.ImageRegistry.Registry).To(Equal("mirrors.tencent.com/bkpaas"))
			Expect(output.ImageRegistry.Username).To(Equal("admin"))
			Expect(output.ImageRegistry.Password).To(Equal("secret"))
		})

		It("handles nil image registry", func() {
			ws := workspace.Workspace{
				ID:                "ws-no-ir",
				ImageRegistryType: registry.ImageRegistryTypeBuiltin,
				State:             workspace.StateReady,
			}

			output := new(serializer.WorkspaceDetailOutputObj).FromModel(ws, nil)

			Expect(output.ImageRegistryType).To(Equal("builtin"))
			Expect(output.ImageRegistry).NotTo(BeNil())
			Expect(output.ImageRegistry.Registry).To(BeEmpty())
		})
	})

	// -----------------------------------------------------------------------
	// int64 json:",string" 序列化/反序列化测试
	// -----------------------------------------------------------------------
	Describe("int64 JSON string tag", func() {
		DescribeTable("serializes int64 as JSON string",
			func(obj interface{}, fieldName, expectedRaw string) {
				data, err := json.Marshal(obj)
				Expect(err).NotTo(HaveOccurred())

				var raw map[string]json.RawMessage
				Expect(json.Unmarshal(data, &raw)).To(Succeed())
				Expect(string(raw[fieldName])).To(Equal(expectedRaw))
			},
			Entry("CreateWorkspaceInput bkCCBizID",
				serializer.CreateWorkspaceInput{ID: "ws1", DisplayName: "n", BkCCBizID: 1234567890},
				"bkCCBizID", `1234567890`),
			Entry("UserStatisticsOutputObj workspaceCount",
				serializer.UserStatisticsOutputObj{WorkspaceCount: 10},
				"workspaceCount", `"10"`),
			Entry("UserStatisticsOutputObj appCount",
				serializer.UserStatisticsOutputObj{AppCount: 20},
				"appCount", `"20"`),
			Entry("UserStatisticsOutputObj envCount",
				serializer.UserStatisticsOutputObj{EnvCount: 30},
				"envCount", `"30"`),
			Entry("RoleMemberGroupOutputObj userGroupID",
				serializer.RoleMemberGroupOutputObj{UserGroupID: 1234567890123},
				"userGroupID", `"1234567890123"`),
		)

		It("deserializes JSON string to int64 for CreateWorkspaceInput bkCCBizID", func() {
			jsonStr := `{"id":"ws1","displayName":"名称","bkCCBizID":999888777}`
			var parsed serializer.CreateWorkspaceInput
			Expect(json.Unmarshal([]byte(jsonStr), &parsed)).To(Succeed())
			Expect(parsed.BkCCBizID).To(Equal(int64(999888777)))
		})

		It("serializes nested UserWorkspaceStatisticsOutputObj int64 fields as strings", func() {
			output := serializer.UserStatisticsOutputObj{
				WorkspaceCount: 10,
				AppCount:       20,
				EnvCount:       30,
				WorkspaceStatistics: []*serializer.UserWorkspaceStatisticsOutputObj{
					{WorkspaceID: "ws1", AppCount: 5, EnvCount: 8},
				},
			}

			data, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())

			var raw map[string]json.RawMessage
			Expect(json.Unmarshal(data, &raw)).To(Succeed())

			var stats []map[string]json.RawMessage
			Expect(json.Unmarshal(raw["workspaceStatistics"], &stats)).To(Succeed())
			Expect(stats).To(HaveLen(1))
			Expect(string(stats[0]["appCount"])).To(Equal(`"5"`))
			Expect(string(stats[0]["envCount"])).To(Equal(`"8"`))
		})
	})

	Describe("CreateWorkspace", func() {
		DescribeTable("input deserialization",
			func(jsonStr string, target, expected interface{}) {
				Expect(json.Unmarshal([]byte(jsonStr), target)).To(Succeed())
				Expect(target).To(Equal(expected))
			},
			Entry(
				"CreateWorkspaceInput with all fields",
				`{"id":"my-workspace","displayName":"我的工作空间","description":"测试描述","bkCIProjectID":"ci-proj","bkCCBizID":100,"managers":["mgr1","mgr2"],"imageRegistry":{"registry":"mirrors.tencent.com/bkpaas","username":"user","password":"pass"}}`,
				&serializer.CreateWorkspaceInput{},
				&serializer.CreateWorkspaceInput{
					ID:            "my-workspace",
					DisplayName:   "我的工作空间",
					Description:   "测试描述",
					BkCIProjectID: "ci-proj",
					BkCCBizID:     100,
					Managers:      []string{"mgr1", "mgr2"},
					ImageRegistry: &serializer.ImageRegistryInput{
						Registry: "mirrors.tencent.com/bkpaas",
						Username: "user",
						Password: "pass",
					},
				},
			),
			Entry("SetWorkspaceStateInput",
				`{"state":"Disabled"}`,
				&serializer.SetWorkspaceStateInput{},
				&serializer.SetWorkspaceStateInput{State: "Disabled"}),
			Entry("AddWorkspaceUserInput",
				`{"userIDs":["u1","u2","u3"]}`,
				&serializer.AddWorkspaceUserInput{},
				&serializer.AddWorkspaceUserInput{UserIDs: []string{"u1", "u2", "u3"}}),
			Entry("UpdateWorkspaceInfoInput",
				`{"displayName":"新名称","description":"新描述"}`,
				&serializer.UpdateWorkspaceInfoInput{},
				&serializer.UpdateWorkspaceInfoInput{DisplayName: "新名称", Description: "新描述"}),
		)

		DescribeTable("input validation",
			func(input serializer.CreateWorkspaceInput, expectedErrSubstrings []string) {
				err := binding.Validator.ValidateStruct(input)
				if len(expectedErrSubstrings) == 0 {
					Expect(err).NotTo(HaveOccurred())
					return
				}

				Expect(err).To(HaveOccurred())
				for _, expected := range expectedErrSubstrings {
					Expect(err.Error()).To(ContainSubstring(expected))
				}
			},
			Entry("valid input", serializer.CreateWorkspaceInput{
				ID:          "my-workspace",
				DisplayName: "My Workspace",
			}, nil),
			Entry("too long workspace id", serializer.CreateWorkspaceInput{
				ID:          "my-workspace-abcdefghijklmnop",
				DisplayName: "My Workspace",
			}, []string{
				"CreateWorkspaceInput.ID",
				"failed on the 'workspace_id' tag",
			}),
		)
	})
})
