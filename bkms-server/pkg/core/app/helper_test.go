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

package app_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

var _ = Describe("ListSortByOpTime", func() {
	var ctx context.Context
	var appStore *bkmsapp.ApplicationStoreMongo
	var opStore audit.OperationRecordStore
	var workspaceID string

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		workspaceID = "test-ws-" + stringx.Random(6)

		appStore, err = bkmsapp.NewApplicationStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		opStore, err = audit.NewOperationRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return nil when no apps exist", func() {
		result, err := bkmsapp.ListSortByOpTime(
			ctx, appStore, opStore, "nonexistent-ws", "some-user",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeNil())
	})

	It("should sort apps by user operation time descending", func() {
		username := "test-user-" + stringx.Random(6)

		app1 := &bkmsapp.Application{
			ID: "app-oldest-" + stringx.Random(6), WorkspaceID: workspaceID,
			Name: "app-oldest", DisplayName: "oldest",
		}
		app2 := &bkmsapp.Application{
			ID: "app-newest-" + stringx.Random(6), WorkspaceID: workspaceID,
			Name: "app-newest", DisplayName: "newest",
		}
		app3 := &bkmsapp.Application{
			ID: "app-middle-" + stringx.Random(6), WorkspaceID: workspaceID,
			Name: "app-middle", DisplayName: "middle",
		}

		Expect(appStore.CreateApp(ctx, app1)).To(Succeed())
		Expect(appStore.CreateApp(ctx, app2)).To(Succeed())
		Expect(appStore.CreateApp(ctx, app3)).To(Succeed())

		now := time.Now()
		createOpRecord := func(appID string, createdAt time.Time) {
			record := &audit.OperationRecord{
				ID: bson.NewObjectID(), Username: username,
				AccessType: audit.AccessTypeWeb, OperationType: audit.OperationTypeUpdate,
				ResourceType: audit.ResourceTypeApp, ResourceID: appID,
				Result:    audit.ResultSuccess,
				Group:     audit.OperationGroup{WorkspaceID: workspaceID, AppID: appID},
				CreatedAt: createdAt, UpdatedAt: createdAt,
			}
			_, err := opStore.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())
		}

		createOpRecord(app1.ID, now.Add(-3*time.Hour))
		createOpRecord(app2.ID, now.Add(-1*time.Hour))
		createOpRecord(app3.ID, now.Add(-2*time.Hour))

		result, err := bkmsapp.ListSortByOpTime(
			ctx, appStore, opStore, workspaceID, username,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(3))

		Expect(result[0].ID).To(Equal(app2.ID))
		Expect(result[1].ID).To(Equal(app3.ID))
		Expect(result[2].ID).To(Equal(app1.ID))
	})

	It("should place apps without operation records at the end", func() {
		username := "test-user-" + stringx.Random(6)

		appWithOp := &bkmsapp.Application{
			ID: "app-with-op-" + stringx.Random(6), WorkspaceID: workspaceID,
			Name: "app-with-op", DisplayName: "with op",
		}
		appNoOp := &bkmsapp.Application{
			ID: "app-no-op-" + stringx.Random(6), WorkspaceID: workspaceID,
			Name: "app-no-op", DisplayName: "no op",
		}

		Expect(appStore.CreateApp(ctx, appWithOp)).To(Succeed())
		Expect(appStore.CreateApp(ctx, appNoOp)).To(Succeed())

		record := &audit.OperationRecord{
			ID: bson.NewObjectID(), Username: username,
			AccessType: audit.AccessTypeWeb, OperationType: audit.OperationTypeCreate,
			ResourceType: audit.ResourceTypeApp, ResourceID: appWithOp.ID,
			Result:    audit.ResultSuccess,
			Group:     audit.OperationGroup{WorkspaceID: workspaceID, AppID: appWithOp.ID},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		_, err := opStore.Create(ctx, record)
		Expect(err).NotTo(HaveOccurred())

		result, err := bkmsapp.ListSortByOpTime(
			ctx, appStore, opStore, workspaceID, username,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(2))
		Expect(result[0].ID).To(Equal(appWithOp.ID))
		Expect(result[1].ID).To(Equal(appNoOp.ID))
	})
})
