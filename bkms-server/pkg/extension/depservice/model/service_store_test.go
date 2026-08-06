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

package model

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test ServiceStoreMongo", func() {
	var mongoClient *mongo.Client

	var ctx context.Context

	var serviceStore *ServiceStoreMongo

	BeforeEach(func() {
		var err error
		var dbName string

		mongoClient, dbName = database.Client(), database.Name()

		ctx = context.Background()

		serviceStore, err = NewServiceStoreMongo(mongoClient, dbName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := serviceStore.DeleteAll(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Test ServiceStoreMongo methods", func() {
		var svc Service
		var systemPlan ServicePlan
		var userPlan ServicePlan

		BeforeEach(func() {
			systemPlan = ServicePlan{
				Name:         "default",
				ProviderType: ProviderTypeSystemAllocated,
				Config:       map[string]any{"baseUrl": "http://test-polaris.example.com:8080"},
			}
			userPlan = ServicePlan{
				Name:         "custom",
				ProviderType: ProviderTypeUserDefined,
				Config:       map[string]any{},
			}
			plans := []ServicePlan{systemPlan, userPlan}

			svcName := "test-svc-" + stringx.Random(6)
			svc = Service{Name: svcName, DisplayName: svcName, Plans: plans}
			// test: 成功创建 service
			err := serviceStore.Create(ctx, &svc)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// test: 成功删除 service
			err := serviceStore.Delete(ctx, svc.Name)
			Expect(err).NotTo(HaveOccurred())
			_, err = serviceStore.Get(ctx, svc.Name)
			Expect(AsNotFoundError(err)).To(Equal(true))
		})

		It("test get service", func() {
			svcFromDB, err := serviceStore.Get(ctx, svc.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(svcFromDB.Name).To(Equal(svc.Name))
			Expect(svcFromDB.Plans[0].Name).To(Equal(systemPlan.Name))
			// test: 此处验证了 serviceFromDBValue 和 servicePrepDBValue 的正确性
			Expect(svcFromDB.Plans[0].Config["baseUrl"]).To(Equal("http://test-polaris.example.com:8080"))
			Expect(svcFromDB.Plans[1].Name).To(Equal(userPlan.Name))
		})

		It("test upsert service", func() {
			newDisplayName := "new-display-name-" + stringx.Random(6)
			planName := "foo-" + stringx.Random(6)
			newSvc := Service{Name: svc.Name, DisplayName: newDisplayName, Plans: []ServicePlan{{Name: planName}}}

			err := serviceStore.Upsert(ctx, &newSvc)
			Expect(err).NotTo(HaveOccurred())

			dbSvc, err := serviceStore.Get(ctx, svc.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(dbSvc.DisplayName).To(Equal(newDisplayName))
			Expect(dbSvc.Plans).To(HaveLen(1))
			Expect(dbSvc.Plans[0].Name).To(Equal(planName))
		})

		It("test list services", func() {
			svcList, err := serviceStore.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(svcList).To(HaveLen(1))
			Expect(svcList[0].Name).To(Equal(svc.Name))
		})
	})
})
