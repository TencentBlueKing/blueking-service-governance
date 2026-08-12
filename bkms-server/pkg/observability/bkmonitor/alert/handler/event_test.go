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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
)

var _ = Describe("collectRemoteStrategyIDsForAppAlerts", func() {
	It("filters by alert display name while keeping local display name mapping", func() {
		matchedID := bson.NewObjectID()
		otherID := bson.NewObjectID()

		ids, remoteToBKMS := collectRemoteStrategyIDsForAppAlerts([]alertstrategy.AlertStrategy{
			{
				ID:          matchedID,
				DisplayName: "CPU 过高",
				RemoteRefs: []alertstrategy.RemoteStrategyRef{
					{EnvName: "prod", RemoteStrategyID: 1001},
				},
			},
			{
				ID:          otherID,
				DisplayName: "Memory 过高",
				RemoteRefs: []alertstrategy.RemoteStrategyRef{
					{EnvName: "prod", RemoteStrategyID: 1002},
					{EnvName: "stag", RemoteStrategyID: 1003},
				},
			},
		}, "CPU")

		Expect(ids).To(Equal([]int64{1001}))
		Expect(remoteToBKMS).To(HaveKey(int64(1001)))
		Expect(remoteToBKMS[1001].StrategyID).To(Equal(matchedID.Hex()))
		Expect(remoteToBKMS[1001].AlertDisplayName).To(Equal("CPU 过高"))
		Expect(remoteToBKMS).NotTo(HaveKey(1002))
		Expect(remoteToBKMS).NotTo(HaveKey(1003))
	})

	It("deduplicates a shared remote strategy id across multiple environments", func() {
		ruleID := bson.NewObjectID()

		ids, remoteToBKMS := collectRemoteStrategyIDsForAppAlerts([]alertstrategy.AlertStrategy{{
			ID:          ruleID,
			DisplayName: "容器异常重启",
			RemoteRefs: []alertstrategy.RemoteStrategyRef{
				{EnvName: "teamdev", RemoteStrategyID: 1001},
				{EnvName: "pre-release", RemoteStrategyID: 1001},
			},
		}}, "")

		Expect(ids).To(Equal([]int64{1001}))
		Expect(remoteToBKMS).To(HaveKey(int64(1001)))
		Expect(remoteToBKMS[1001].StrategyID).To(Equal(ruleID.Hex()))
		Expect(remoteToBKMS[1001].AlertDisplayName).To(Equal("容器异常重启"))
	})
})

var _ = Describe("extractAlertDisplayNameFromRemoteName", func() {
	It("strips app suffix from remote strategy name", func() {
		Expect(extractAlertDisplayNameFromRemoteName("CPU 使用率过高【demo-app】")).To(Equal("CPU 使用率过高"))
		Expect(extractAlertDisplayNameFromRemoteName("Already Plain")).To(Equal("Already Plain"))
	})
})
