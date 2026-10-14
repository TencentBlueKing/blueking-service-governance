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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Service store", func() {
	var ctx context.Context
	var services *ServiceStore
	var svc Service
	var systemPlan ServicePlan
	var userPlan ServicePlan

	BeforeEach(func() {
		ctx = context.Background()
		services = testutil.NewStore(GinkgoT(), ServiceMeta)

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
		Expect(services.Create(ctx, &svc)).To(Succeed())
	})

	AfterEach(func() {
		// test: 成功删除 service
		Expect(services.Delete(ctx, ByName(svc.Name))).To(Succeed())
		_, err := services.Get(ctx, ByName(svc.Name))
		Expect(err).To(MatchError(ErrServiceNotFound))
		Expect(AsNotFoundError(err)).To(BeTrue())
	})

	It("test get service", func() {
		svcFromDB, err := services.Get(ctx, ByName(svc.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(svcFromDB.Name).To(Equal(svc.Name))
		Expect(svcFromDB.Plans[0].Name).To(Equal(systemPlan.Name))
		// test: 此处验证了 plan config 编解码钩子的正确性
		Expect(svcFromDB.Plans[0].Config["baseUrl"]).To(Equal("http://test-polaris.example.com:8080"))
		Expect(svcFromDB.Plans[1].Name).To(Equal(userPlan.Name))
		// 调用方对象中的 config 不受编码影响
		Expect(svc.Plans[0].Config).To(Equal(systemPlan.Config))
	})

	It("stores plan config as a json string", func() {
		var raw Service
		Expect(services.Collection().FindOne(ctx, bson.M{"name": svc.Name}).Decode(&raw)).To(Succeed())
		Expect(
			raw.Plans[0].Config,
		).To(HaveKeyWithValue(configValueKey, `{"baseUrl":"http://test-polaris.example.com:8080"}`))
	})

	It("rejects duplicate service names", func() {
		Expect(services.Create(ctx, &Service{Name: svc.Name, Plans: []ServicePlan{userPlan}})).
			To(MatchError(store.ErrDuplicate))
	})

	It("test upsert service", func() {
		newDisplayName := "new-display-name-" + stringx.Random(6)
		planName := "foo-" + stringx.Random(6)
		newSvc := Service{Name: svc.Name, DisplayName: newDisplayName, Plans: []ServicePlan{{Name: planName}}}

		_, err := services.Upsert(ctx, &newSvc, ByName(newSvc.Name))
		Expect(err).NotTo(HaveOccurred())

		dbSvc, err := services.Get(ctx, ByName(svc.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(dbSvc.DisplayName).To(Equal(newDisplayName))
		Expect(dbSvc.Plans).To(HaveLen(1))
		Expect(dbSvc.Plans[0].Name).To(Equal(planName))
	})

	It("test list services", func() {
		svcList, err := services.List(ctx, ByName(svc.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(svcList).To(HaveLen(1))
		Expect(svcList[0].Name).To(Equal(svc.Name))
		// List 同样会解码 plan config
		Expect(svcList[0].Plans[0].Config["baseUrl"]).To(Equal("http://test-polaris.example.com:8080"))
	})
})
