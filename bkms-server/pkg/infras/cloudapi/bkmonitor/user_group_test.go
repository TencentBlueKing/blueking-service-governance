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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResetStubStateForTest", func() {
	Describe("stub state reset", func() {
		BeforeEach(func() {
			ResetStubStateForTest()
		})

		It("clears dynamically created apm apps", func() {
			client := NewStub("tester")

			_, err := client.CreateApmApp(
				context.Background(),
				-2001,
				"project-a",
				"temp-env",
				"desc",
				"tester",
				"ws-1",
			)
			Expect(err).NotTo(HaveOccurred())

			apps, err := client.ListApmApp(context.Background(), -2001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(ContainElement(HaveField("AppName", "temp-env")))

			ResetStubStateForTest()

			apps, err = client.ListApmApp(context.Background(), -2001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).NotTo(ContainElement(HaveField("AppName", "temp-env")))
		})

		It("restores custom user groups to default stub data", func() {
			client := NewStub("tester")
			groupID := int64(9999)

			_, err := client.SaveUserGroup(context.Background(), &SaveUserGroupReq{
				ID:       &groupID,
				BkBizID:  -1,
				Name:     "temp",
				Channels: []string{"user"},
				AlertNotice: []AlertNotice{
					{TimeRange: "00:00--23:59"},
				},
				ActionNotice: []ActionNotice{
					{TimeRange: "00:00--23:59"},
				},
				Operator: "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := client.SearchUserGroupDetail(context.Background(), &SearchUserGroupDetailReq{ID: groupID})
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.BkBizID).To(Equal(int64(-1)))
			Expect(detail.Name).To(Equal("temp"))

			ResetStubStateForTest()

			detail, err = client.SearchUserGroupDetail(context.Background(), &SearchUserGroupDetailReq{ID: groupID})
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.ID).To(Equal(groupID))
			Expect(detail.BkBizID).To(Equal(stubDefaultUserGroupBkBizID))
			Expect(detail.Name).To(Equal(stubDefaultUserGroupName))
		})
	})
})

var _ = Describe("SaveUserGroupReq", func() {
	It("omits id when marshalling create requests", func() {
		payload, err := json.Marshal(&SaveUserGroupReq{
			BkBizID:  -120,
			Name:     "temp",
			Channels: []string{"user"},
			AlertNotice: []AlertNotice{
				{TimeRange: "00:00--23:59"},
			},
			ActionNotice: []ActionNotice{
				{TimeRange: "00:00--23:59"},
			},
			Operator: "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(string(payload)).NotTo(ContainSubstring(`"id"`))
	})
})

var _ = Describe("StubClient SaveUserGroup", func() {
	It("derives top level users from duty arranges", func() {
		client := NewStub("tester")
		groupID := int64(1001)

		detail, err := client.SaveUserGroup(context.Background(), &SaveUserGroupReq{
			ID:       &groupID,
			BkBizID:  -1,
			Name:     "temp",
			Channels: []string{"user"},
			AlertNotice: []AlertNotice{
				{TimeRange: "00:00--23:59"},
			},
			ActionNotice: []ActionNotice{
				{TimeRange: "00:00--23:59"},
			},
			DutyArranges: []DutyArrange{{
				Users: []UserGroupUser{{
					ID:          "tester",
					Type:        "user",
					DisplayName: "Tester",
				}},
			}},
			Operator: "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(detail.Users).To(HaveLen(1))
		Expect(detail.Users[0].ID).To(Equal("tester"))
	})
})
