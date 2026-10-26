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

package appmodel_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("CheckIfTrafficLaneDeployed", func() {
	DescribeTable("checks the latest record of the lane",
		func(status appmodel.Status, deployed bool) {
			ctx := context.Background()
			db := testutil.NewDB(GinkgoT())
			appID := "lane-check-" + stringx.Random(6)
			records := store.Of(db, appmodel.RecordMeta)
			DeferCleanup(func() {
				_, _ = records.DeleteMany(context.Background(), store.Where("appID", appID))
			})
			Expect(records.Create(ctx, &appmodel.Record{
				AppID: appID, EnvName: "prod", TrafficLaneName: "base", Status: status,
			})).To(Succeed())

			err := appmodel.CheckIfTrafficLaneDeployed(ctx, db, appID, "prod", "base")
			if deployed {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring("not deployed")))
			}
		},
		Entry("deployed", appmodel.StatusDeployed, true),
		// 回滚后工作负载仍在运行
		Entry("rolled back batch release", appmodel.StatusRolledBack, true),
		Entry("deploying", appmodel.StatusDeploying, false),
		Entry("failed", appmodel.StatusFailed, false),
	)
})
