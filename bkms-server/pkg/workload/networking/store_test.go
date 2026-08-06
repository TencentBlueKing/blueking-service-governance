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

package networking

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test ServiceStoreMongo", func() {
	var ctx context.Context
	var mongoClient *mongo.Client
	var store ServiceStore
	var dbName string

	BeforeEach(func() {
		var err error

		mongoClient, dbName = database.Client(), database.Name()

		ctx = context.Background()

		store, err = NewServiceStoreMongo(mongoClient, dbName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Test ServiceStoreMongo method", func() {
		appID1 := "test-app-" + stringx.Random(6)
		svc1 := &Service{
			AppID:              appID1,
			Name:               "test-service1-" + stringx.Random(6),
			Selector:           map[string]string{"app": appID1},
			Ports:              []ServicePort{{Name: "http", Protocol: ProtocolTCP, Port: 80, TargetPort: "8080"}},
			TrafficLaneEnabled: true,
		}

		appID2 := "test-app-" + stringx.Random(6)
		svc2 := &Service{
			AppID:              appID2,
			Name:               "test-service2-" + stringx.Random(6),
			Selector:           map[string]string{"app": appID2},
			Ports:              []ServicePort{{Name: "udp", Protocol: ProtocolUDP, Port: 53, TargetPort: "53"}},
			TrafficLaneEnabled: false,
		}

		BeforeEach(func() {
			var err error
			// test: create service
			err = store.Create(ctx, svc1)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, svc2)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			// test: delete service
			err := store.Delete(ctx, appID1, svc1.Name)
			Expect(err).NotTo(HaveOccurred())
			err = store.Delete(ctx, appID2, svc2.Name)
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteAll(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("create fail when appID is missing", func() {
			svc := &Service{
				Name: "test-service-" + stringx.Random(6),
			}

			err := store.Create(ctx, svc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation failed"))
		})

		It("list services by appID successfully", func() {
			services, _ := store.ListByApp(ctx, appID1)
			Expect(services).To(HaveLen(1))
			Expect(services[0].Name).To(Equal(svc1.Name))
			Expect(services[0].Selector).To(Equal(svc1.Selector))
			Expect(services[0].Ports).To(HaveLen(1))
			Expect(services[0].TrafficLaneEnabled).To(BeTrue())

			services, _ = store.ListByApp(ctx, appID2)
			Expect(services).To(HaveLen(1))
			Expect(services[0].Name).To(Equal(svc2.Name))
			Expect(services[0].Selector).To(Equal(svc2.Selector))
			Expect(services[0].Ports).To(HaveLen(1))
			Expect(services[0].TrafficLaneEnabled).To(BeFalse())
		})

		It("group services by appID successfully", func() {
			grouped, _ := store.GroupByAppID(ctx, []string{appID1, appID2})
			Expect(grouped).To(HaveLen(2))

			Expect(grouped[appID1]).To(HaveLen(1))
			Expect(grouped[appID1][0].Name).To(Equal(svc1.Name))
			Expect(grouped[appID2]).To(HaveLen(1))
			Expect(grouped[appID2][0].Name).To(Equal(svc2.Name))
		})

		Context("Test Update", func() {
			It("update selector and ports successfully", func() {
				randValue := stringx.Random(6)
				randPortName := stringx.Random(6)

				err := store.Update(
					ctx,
					svc1.AppID,
					svc1.Name,
					&SvcUpdateData{Selector: map[string]string{"foo": randValue}, Ports: []ServicePort{
						{
							Name:       randPortName,
							Protocol:   ProtocolTCP,
							Port:       8000,
							TargetPort: "5000",
						},
					}},
				)
				Expect(err).NotTo(HaveOccurred())

				svc, _ := store.Get(ctx, svc1.AppID, svc1.Name)
				Expect(svc.Selector).To(Equal(map[string]string{"foo": randValue}))
				Expect(svc.Ports[0].Name).To(Equal(randPortName))
				Expect(svc.Ports[0].TargetPort).To(Equal("5000"))
			})

			It("update trafficLaneEnabled successfully", func() {
				trafficLaneEnabled := true
				err := store.Update(
					ctx,
					svc2.AppID,
					svc2.Name,
					&SvcUpdateData{TrafficLaneEnabled: &trafficLaneEnabled},
				)
				Expect(err).NotTo(HaveOccurred())

				svc, _ := store.Get(ctx, svc2.AppID, svc2.Name)
				Expect(svc.TrafficLaneEnabled).To(BeTrue())
				Expect(svc.Selector).To(Equal(svc2.Selector))
				Expect(svc.Ports).To(Equal(svc2.Ports))
			})
		})
	})
})
