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

package polaris_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

var _ = Describe("InstanceMatcher", func() {
	It("matches instances by pod IP and service port", func() {
		svcInstances := []*polaris.PolarisServiceInstances{
			{
				ServiceNamespace: "Production",
				ServiceName:      "svc-a",
				ServicePort:      8080,
				Instances: []*polarisInfra.Instance{
					{
						IP:                "127.0.0.1",
						Port:              8080,
						Weight:            100,
						StaticWeight:      80,
						IsHealthy:         true,
						EnableHealthCheck: true,
						Metadata:          map[string]string{"k": "v"},
					},
					{IP: "127.0.0.1", Port: 9090},
				},
			},
		}

		infos := polaris.NewInstanceMatcher(svcInstances).ForIP("127.0.0.1")

		Expect(infos).To(HaveLen(1))
		Expect(infos[0].ServiceNamespace).To(Equal("Production"))
		Expect(infos[0].ServiceName).To(Equal("svc-a"))
		Expect(infos[0].Port).To(Equal(uint32(8080)))
		Expect(infos[0].Weight).To(Equal(int64(100)))
		Expect(infos[0].StaticWeight).To(Equal(int64(80)))
		Expect(infos[0].Metadata).To(Equal(map[string]string{"k": "v"}))
	})

	// 北极星侧返回顺序不保证稳定，Watch 补拉要靠前后两次结果比对差异
	It("orders matched instances by service coordinates regardless of input order", func() {
		svcInstances := []*polaris.PolarisServiceInstances{
			{
				ServiceNamespace: "Production",
				ServiceName:      "svc-b",
				ServicePort:      8080,
				Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 8080}},
			},
			{
				ServiceNamespace: "Development",
				ServiceName:      "svc-a",
				ServicePort:      9090,
				Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 9090}},
			},
			{
				ServiceNamespace: "Production",
				ServiceName:      "svc-a",
				ServicePort:      8080,
				Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 8080}},
			},
		}

		infos := polaris.NewInstanceMatcher(svcInstances).ForIP("127.0.0.1")

		Expect(infos).To(HaveLen(3))
		Expect(infos[0].ServiceNamespace).To(Equal("Development"))
		Expect(infos[0].ServiceName).To(Equal("svc-a"))
		Expect(infos[1].ServiceNamespace).To(Equal("Production"))
		Expect(infos[1].ServiceName).To(Equal("svc-a"))
		Expect(infos[2].ServiceNamespace).To(Equal("Production"))
		Expect(infos[2].ServiceName).To(Equal("svc-b"))
	})

	It("returns an empty slice rather than nil when nothing matches", func() {
		infos := polaris.NewInstanceMatcher(nil).ForIP("10.0.0.1")

		Expect(infos).NotTo(BeNil())
		Expect(infos).To(BeEmpty())
	})
})
