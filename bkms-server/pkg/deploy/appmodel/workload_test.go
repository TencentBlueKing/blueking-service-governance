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
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GetLatestWorkload", func() {
	var (
		ctx   context.Context
		db    *mongo.Database
		appID string
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		appID = "test-app-" + stringx.Random(6)
	})

	// createRecord 写入一条默认泳道的部署记录
	createRecord := func(createdAt time.Time, kind string, keys appmodel.ResourceKeys) {
		record := &appmodel.Record{
			WorkspaceID:  "test-workspace",
			AppID:        appID,
			EnvName:      "prod",
			ClusterID:    "BCS-K8S-00000",
			Namespace:    "demo-ns",
			WorkloadKind: kind,
			ResourceKeys: keys,
			Status:       appmodel.StatusDeployed,
			CreatedAt:    createdAt,
		}
		Expect(store.Of(db, appmodel.RecordMeta).Create(ctx, record)).To(Succeed())
	}

	It("returns ErrDeployRecordNotFound when the env has no deploy record", func() {
		kind, name, err := appmodel.GetLatestWorkload(ctx, db, appID, "prod", "")
		Expect(errors.Is(err, appmodel.ErrDeployRecordNotFound)).To(BeTrue())
		Expect(kind).To(BeEmpty())
		Expect(name).To(BeEmpty())
	})

	It("returns the main workload of the latest deploy record", func() {
		now := time.Now()
		createRecord(now.Add(-time.Hour), "Deployment", appmodel.ResourceKeys{{Kind: "Deployment", Name: "old"}})
		createRecord(now, "GameDeployment", appmodel.ResourceKeys{
			{Kind: "Service", Name: "demo-svc"},
			{Kind: "GameDeployment", Name: "demo"},
		})

		kind, name, err := appmodel.GetLatestWorkload(ctx, db, appID, "prod", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(kind).To(Equal("GameDeployment"))
		Expect(name).To(Equal("demo"))
	})

	It("returns an error when the record has no matching workload resource", func() {
		createRecord(time.Now(), "GameDeployment", appmodel.ResourceKeys{{Kind: "Service", Name: "demo-svc"}})

		kind, name, err := appmodel.GetLatestWorkload(ctx, db, appID, "prod", "")
		Expect(err).To(MatchError(ContainSubstring("has no workload")))
		Expect(errors.Is(err, appmodel.ErrDeployRecordNotFound)).To(BeFalse())
		Expect(kind).To(BeEmpty())
		Expect(name).To(BeEmpty())
	})
})
