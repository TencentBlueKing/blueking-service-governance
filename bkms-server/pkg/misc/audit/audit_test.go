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

package audit

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Audit", func() {
	Describe("Attribute", func() {
		It("displays port-forward attribute name", func() {
			Expect(AttributePortForward.DisplayName()).To(Equal("端口转发"))
		})
	})

	var store OperationRecordStore
	var ctx context.Context

	BeforeEach(func() {
		var err error
		store, err = NewOperationRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
	})

	AfterEach(func() {
		// Clean up test database
		err := testutil.CleanupCollection(collectionName)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("AddOperationRecord", func() {
		var (
			username      string
			operationType OperationType
			resourceType  ResourceType
			resourceID    string
		)

		BeforeEach(func() {
			username = "blueking"
			operationType = OperationTypeCreate
			resourceType = ResourceTypeApp
			resourceID = bson.NewObjectID().Hex()
		})

		Context("basic operation record", func() {
			It("should create operation record and return ID", func() {
				id, err := AddOperationRecord(ctx, username, operationType, resourceType, resourceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())

				// Verify record exists
				record, err := store.Get(ctx, id)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.Username).To(Equal(username))
				Expect(record.OperationType).To(Equal(operationType))
				Expect(record.ResourceType).To(Equal(resourceType))
				Expect(record.ResourceID).To(Equal(resourceID))
				Expect(record.Result).To(Equal(ResultSuccess))
				Expect(record.AccessType).To(Equal(AccessTypeWeb))
			})
		})

		Context("with options", func() {
			It("should create record with multiple options", func() {
				attribute := Attribute("multi-option-app")
				dataBefore := map[string]any{"status": "old"}
				dataAfter := map[string]any{"status": "new"}

				id, err := AddOperationRecord(
					ctx, username, OperationTypeUpdate, resourceType, resourceID,
					WithAccessType(AccessTypeClient),
					WithAttribute(attribute),
					WithResult(ResultSuccess),
					WithDataBefore(dataBefore),
					WithDataAfter(dataAfter),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())

				record, err := store.Get(ctx, id)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.AccessType).To(Equal(AccessTypeClient))
				Expect(record.Attribute).To(Equal(attribute))
				Expect(record.Result).To(Equal(ResultSuccess))
				Expect(record.Data.Before).NotTo(BeNil())
				Expect(record.Data.After).NotTo(BeNil())
			})

			It("should create record with group information", func() {
				workspaceID := "workspace-test-" + stringx.Random(6)
				appID := "app-test-" + stringx.Random(6)
				envName := "production"

				id, err := AddOperationRecord(
					ctx, username, OperationTypeDeploy, resourceType, resourceID,
					WithWorkspaceID(workspaceID),
					WithAppID(appID),
					WithEnvName(envName),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())

				record, err := store.Get(ctx, id)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.Group.WorkspaceID).To(Equal(workspaceID))
				Expect(record.Group.AppID).To(Equal(appID))
				Expect(record.Group.EnvName).To(Equal(envName))
			})

			It("should create record with complete group using WithGroup", func() {
				group := OperationGroup{
					WorkspaceID: "workspace-test-" + stringx.Random(6),
					AppID:       "app-test-" + stringx.Random(6),
					EnvName:     bson.NewObjectID().Hex(),
				}

				id, err := AddOperationRecord(
					ctx, username, OperationTypeBuild, resourceType, resourceID,
					WithGroup(group),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())

				record, err := store.Get(ctx, id)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.Group.WorkspaceID).To(Equal(group.WorkspaceID))
				Expect(record.Group.AppID).To(Equal(group.AppID))
				Expect(record.Group.EnvName).To(Equal(group.EnvName))
			})
		})
	})
})
