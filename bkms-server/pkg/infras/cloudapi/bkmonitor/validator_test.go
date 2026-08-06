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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate", func() {
	It("accepts search user groups request with only bk biz ids", func() {
		err := Validate(&SearchUserGroupsReq{
			BkBizIDs: []int64{2},
		})

		Expect(err).NotTo(HaveOccurred())
	})

	It("requires ids or name for legacy search user groups validation", func() {
		err := (&SearchUserGroupsReq{
			BkBizIDs: []int64{-2001},
		}).Validate()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ids or name is required"))
	})

	It("accepts legacy search user groups validation when name is provided", func() {
		err := (&SearchUserGroupsReq{
			BkBizIDs: []int64{-2001},
			Name:     "ops",
		}).Validate()

		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("rejects invalid requests",
		func(req any) {
			Expect(Validate(req)).To(HaveOccurred())
		},
		Entry("search user groups without bk biz ids", &SearchUserGroupsReq{}),
		Entry("search user group detail without id", &SearchUserGroupDetailReq{}),
		Entry("save user group without required slices", &SaveUserGroupReq{
			BkBizID:  100,
			Name:     "ops",
			Operator: "tester",
		}),
		Entry("delete user group without ids", &DeleteUserGroupReq{
			BkBizIDs: []int64{-2001},
			Operator: "tester",
		}),
		Entry("delete user group without bk biz ids", &DeleteUserGroupReq{
			IDs:      []int64{1001},
			Operator: "tester",
		}),
		Entry("time series query without time range", &TimeSeriesUnifyQueryReq{}),
	)

	It("accepts valid requests", func() {
		Expect(Validate(&SearchUserGroupsReq{
			BkBizIDs: []int64{-2001},
		})).To(Succeed())
		Expect(Validate(&SearchUserGroupDetailReq{
			ID: 1001,
		})).To(Succeed())
		Expect(Validate(&SaveUserGroupReq{
			BkBizID:      -2001,
			Name:         "ops",
			Operator:     "tester",
			Channels:     []string{"user"},
			AlertNotice:  []AlertNotice{{TimeRange: "00:00--23:59"}},
			ActionNotice: []ActionNotice{{TimeRange: "00:00--23:59"}},
		})).To(Succeed())
		Expect(Validate(&DeleteUserGroupReq{
			IDs:      []int64{1001},
			BkBizIDs: []int64{-2001},
			Operator: "tester",
		})).To(Succeed())
		Expect(Validate(&TimeSeriesUnifyQueryReq{
			BkBizID:      -2001,
			QueryConfigs: []QueryConfig{{DataSourceLabel: "bk_monitor"}},
			StartTime:    1,
			EndTime:      2,
			Expression:   "a",
		})).To(Succeed())
	})
})
