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

package polaris

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	polarisaddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisinfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	watchplugin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

// svcInstances 构造一条注册在 8080 端口的北极星服务；port 用来制造端口不匹配的场景
func svcInstances(ip string, port uint32) []*polarisaddon.PolarisServiceInstances {
	return []*polarisaddon.PolarisServiceInstances{{
		ServiceNamespace: "Production",
		ServiceName:      "demo-app",
		ServicePort:      8080,
		Instances: []*polarisinfra.Instance{
			{IP: ip, Port: port, Weight: 100, IsHealthy: true, EnableHealthCheck: true},
		},
	}}
}

func snapshotOf(id, ip string) []watchplugin.InstanceSnapshot {
	return []watchplugin.InstanceSnapshot{{ID: id, IP: ip}}
}

// polarisInfos 把插件载荷取回成北极星投影列表
func polarisInfos(payload any) []*serializer.PolarisInstanceInfoOutputObj {
	infos, ok := payload.([]*serializer.PolarisInstanceInfoOutputObj)
	Expect(ok).To(BeTrue())

	return infos
}

var _ = Describe("Plugin", func() {
	ctx := context.Background()

	It("returns the polaris state of instances matched by IP and port", func() {
		p := New(func(context.Context) ([]*polarisaddon.PolarisServiceInstances, error) {
			return svcInstances("10.0.0.1", 8080), nil
		})

		payloads, err := p.Fetch(ctx, snapshotOf("pod-1", "10.0.0.1"))

		Expect(err).NotTo(HaveOccurred())
		Expect(p.Name()).To(Equal("polaris"))

		infos := polarisInfos(payloads["pod-1"])
		Expect(infos).To(HaveLen(1))
		Expect(infos[0].ServiceName).To(Equal("demo-app"))
		Expect(infos[0].IsHealthy).To(BeTrue())
		Expect(infos[0].Weight).To(Equal(int64(100)))
	})

	// 没命中就给空载荷：端口对不上（同 IP 注册的其它端口不该混进来）与整个应用没配北极星是同一口径
	// 空切片而不是缺 key，也不是 nil：与「配置被删后成功拉回空」同形，Runner 据此判断变化
	DescribeTable("returns an empty payload when nothing matches",
		func(svcs []*polarisaddon.PolarisServiceInstances) {
			p := New(func(context.Context) ([]*polarisaddon.PolarisServiceInstances, error) {
				return svcs, nil
			})

			payloads, err := p.Fetch(ctx, snapshotOf("pod-1", "10.0.0.1"))

			Expect(err).NotTo(HaveOccurred())
			Expect(payloads).To(HaveKey("pod-1"))
			Expect(polarisInfos(payloads["pod-1"])).To(BeEmpty())
		},
		Entry("port does not match", svcInstances("10.0.0.1", 9090)),
		Entry("nothing is registered", nil),
	)

	// 拉取失败直接抛给 Runner：由它跳过本轮，页面保留上次已知状态
	It("propagates the fetch error", func() {
		p := New(func(context.Context) ([]*polarisaddon.PolarisServiceInstances, error) {
			return nil, errors.New("polaris down")
		})

		_, err := p.Fetch(ctx, snapshotOf("pod-1", "10.0.0.1"))

		Expect(err).To(MatchError(ContainSubstring("polaris down")))
	})
})
