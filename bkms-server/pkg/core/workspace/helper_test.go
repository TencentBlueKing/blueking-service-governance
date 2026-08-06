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

package workspace_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

var _ = Describe("ListSortByOpTime", func() {
	var ctx context.Context
	var wsStore bkmsworkspace.WorkspaceStore
	var opStore audit.OperationRecordStore

	BeforeEach(func() {
		var err error
		ctx = context.Background()

		wsStore, err = bkmsworkspace.NewWorkspaceStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		opStore, err = audit.NewOperationRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should sort workspaces by user operation time descending", func() {
		username := "test-user-" + stringx.Random(6)

		ws1 := dbfactory.Workspace(ctx, wsStore)
		ws2 := dbfactory.Workspace(ctx, wsStore)
		ws3 := dbfactory.Workspace(ctx, wsStore)
		defer func() {
			_ = wsStore.Delete(ctx, ws1.ID)
			_ = wsStore.Delete(ctx, ws2.ID)
			_ = wsStore.Delete(ctx, ws3.ID)
		}()

		now := time.Now()
		createOpRecord := func(wsID string, createdAt time.Time) {
			record := &audit.OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      username,
				AccessType:    audit.AccessTypeWeb,
				OperationType: audit.OperationTypeUpdate,
				ResourceType:  audit.ResourceTypeWorkspace,
				ResourceID:    wsID,
				Result:        audit.ResultSuccess,
				Group:         audit.OperationGroup{WorkspaceID: wsID},
				CreatedAt:     createdAt,
				UpdatedAt:     createdAt,
			}
			_, err := opStore.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())
		}

		createOpRecord(ws1.ID, now.Add(-3*time.Hour))
		createOpRecord(ws2.ID, now.Add(-1*time.Hour))
		createOpRecord(ws3.ID, now.Add(-2*time.Hour))

		result, err := bkmsworkspace.ListSortByOpTime(ctx, wsStore, opStore, username)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(result)).To(BeNumerically(">=", 3))

		idxMap := make(map[string]int)
		for i, r := range result {
			idxMap[r.ID] = i
		}
		Expect(idxMap[ws2.ID]).To(BeNumerically("<", idxMap[ws3.ID]))
		Expect(idxMap[ws3.ID]).To(BeNumerically("<", idxMap[ws1.ID]))
	})

	It("should place workspaces without operation records at the end", func() {
		username := "test-user-" + stringx.Random(6)

		wsWithOp := dbfactory.Workspace(ctx, wsStore)
		wsNoOp := dbfactory.Workspace(ctx, wsStore)
		defer func() {
			_ = wsStore.Delete(ctx, wsWithOp.ID)
			_ = wsStore.Delete(ctx, wsNoOp.ID)
		}()

		record := &audit.OperationRecord{
			ID:            bson.NewObjectID(),
			Username:      username,
			AccessType:    audit.AccessTypeWeb,
			OperationType: audit.OperationTypeCreate,
			ResourceType:  audit.ResourceTypeWorkspace,
			ResourceID:    wsWithOp.ID,
			Result:        audit.ResultSuccess,
			Group:         audit.OperationGroup{WorkspaceID: wsWithOp.ID},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_, err := opStore.Create(ctx, record)
		Expect(err).NotTo(HaveOccurred())

		result, err := bkmsworkspace.ListSortByOpTime(ctx, wsStore, opStore, username)
		Expect(err).NotTo(HaveOccurred())

		idxMap := make(map[string]int)
		for i, r := range result {
			idxMap[r.ID] = i
		}
		Expect(idxMap[wsWithOp.ID]).To(BeNumerically("<", idxMap[wsNoOp.ID]))
	})

	It("should only consider the specified user's operations", func() {
		user1 := "user1-" + stringx.Random(6)
		user2 := "user2-" + stringx.Random(6)

		wsA := dbfactory.Workspace(ctx, wsStore)
		wsB := dbfactory.Workspace(ctx, wsStore)
		defer func() {
			_ = wsStore.Delete(ctx, wsA.ID)
			_ = wsStore.Delete(ctx, wsB.ID)
		}()

		now := time.Now()
		// user1 最近操作了 wsA
		r1 := &audit.OperationRecord{
			ID: bson.NewObjectID(), Username: user1,
			AccessType: audit.AccessTypeWeb, OperationType: audit.OperationTypeUpdate,
			ResourceType: audit.ResourceTypeWorkspace, ResourceID: wsA.ID,
			Result: audit.ResultSuccess, Group: audit.OperationGroup{WorkspaceID: wsA.ID},
			CreatedAt: now, UpdatedAt: now,
		}
		_, err := opStore.Create(ctx, r1)
		Expect(err).NotTo(HaveOccurred())

		// user2 最近操作了 wsB（但对 user1 来说不可见）
		r2 := &audit.OperationRecord{
			ID: bson.NewObjectID(), Username: user2,
			AccessType: audit.AccessTypeWeb, OperationType: audit.OperationTypeUpdate,
			ResourceType: audit.ResourceTypeWorkspace, ResourceID: wsB.ID,
			Result: audit.ResultSuccess, Group: audit.OperationGroup{WorkspaceID: wsB.ID},
			CreatedAt: now.Add(time.Hour), UpdatedAt: now.Add(time.Hour),
		}
		_, err = opStore.Create(ctx, r2)
		Expect(err).NotTo(HaveOccurred())

		result, err := bkmsworkspace.ListSortByOpTime(ctx, wsStore, opStore, user1)
		Expect(err).NotTo(HaveOccurred())

		idxMap := make(map[string]int)
		for i, r := range result {
			idxMap[r.ID] = i
		}
		// user1 视角：wsA 有操作记录，wsB 没有，所以 wsA 排在前面
		Expect(idxMap[wsA.ID]).To(BeNumerically("<", idxMap[wsB.ID]))
	})
})
