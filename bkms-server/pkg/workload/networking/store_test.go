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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Networking service store", func() {
	var (
		ctx      context.Context
		services *ServiceStore
		appID1   string
		appID2   string
		svc1     *Service
		svc2     *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		services = testutil.NewStore(GinkgoT(), ServiceMeta)

		appID1 = "test-app-" + stringx.Random(6)
		svc1 = &Service{
			AppID:              appID1,
			Name:               "test-service1-" + stringx.Random(6),
			Selector:           map[string]string{"app": appID1},
			Ports:              []ServicePort{{Name: "http", Protocol: ProtocolTCP, Port: 80, TargetPort: "8080"}},
			TrafficLaneEnabled: true,
		}
		appID2 = "test-app-" + stringx.Random(6)
		svc2 = &Service{
			AppID:              appID2,
			Name:               "test-service2-" + stringx.Random(6),
			Selector:           map[string]string{"app": appID2},
			Ports:              []ServicePort{{Name: "udp", Protocol: ProtocolUDP, Port: 53, TargetPort: "53"}},
			TrafficLaneEnabled: false,
		}
		Expect(services.Create(ctx, svc1)).To(Succeed())
		Expect(services.Create(ctx, svc2)).To(Succeed())
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(ServiceMeta)).To(Succeed())
	})

	It("create fail when appID is missing", func() {
		err := services.Create(ctx, &Service{Name: "test-service-" + stringx.Random(6)})
		Expect(err).To(MatchError(ContainSubstring("validation failed")))
	})

	It("create fail when the name already exists in the app", func() {
		err := services.Create(ctx, &Service{AppID: appID1, Name: svc1.Name})
		Expect(err).To(MatchError(ErrServiceAlreadyExists))
	})

	It("group services by appID successfully", func() {
		grouped, err := GroupByAppID(ctx, services, []string{appID1, appID2, "app-without-services"})
		Expect(err).NotTo(HaveOccurred())
		Expect(grouped).To(HaveLen(2))

		Expect(grouped[appID1]).To(HaveLen(1))
		Expect(grouped[appID1][0].Name).To(Equal(svc1.Name))
		Expect(grouped[appID2]).To(HaveLen(1))
		Expect(grouped[appID2][0].Name).To(Equal(svc2.Name))

		grouped, err = GroupByAppID(ctx, services, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(grouped).To(BeEmpty())
	})

	Context("SvcUpdateData", func() {
		It("update selector and ports successfully", func() {
			randValue := stringx.Random(6)
			randPortName := stringx.Random(6)

			data := &SvcUpdateData{Selector: map[string]string{"foo": randValue}, Ports: []ServicePort{
				{
					Name:       randPortName,
					Protocol:   ProtocolTCP,
					Port:       8000,
					TargetPort: "5000",
				},
			}}
			Expect(services.Update(ctx, data.ToBSON(), ByAppAndName(svc1.AppID, svc1.Name))).To(Succeed())

			svc, err := services.Get(ctx, ByAppAndName(svc1.AppID, svc1.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Selector).To(Equal(map[string]string{"foo": randValue}))
			Expect(svc.Ports[0].Name).To(Equal(randPortName))
			Expect(svc.Ports[0].TargetPort).To(Equal("5000"))
			Expect(svc.TrafficLaneEnabled).To(BeTrue())
		})

		It("update trafficLaneEnabled while ignoring empty selector and ports", func() {
			trafficLaneEnabled := true
			data := &SvcUpdateData{Selector: map[string]string{}, TrafficLaneEnabled: &trafficLaneEnabled}
			Expect(services.Update(ctx, data.ToBSON(), ByAppAndName(svc2.AppID, svc2.Name))).To(Succeed())

			svc, err := services.Get(ctx, ByAppAndName(svc2.AppID, svc2.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.TrafficLaneEnabled).To(BeTrue())
			Expect(svc.Selector).To(Equal(svc2.Selector))
			Expect(svc.Ports).To(Equal(svc2.Ports))
		})

		It("returns not found when the service does not exist", func() {
			trafficLaneEnabled := true
			data := &SvcUpdateData{TrafficLaneEnabled: &trafficLaneEnabled}
			err := services.Update(ctx, data.ToBSON(), ByAppAndName(appID1, "missing"))
			Expect(err).To(MatchError(ErrServiceNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
		})
	})
})
