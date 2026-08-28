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
	"cmp"
	"slices"

	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

// MatchedInstance 按 Pod IP + 服务端口命中的一条北极星实例
// List 合并与 Watch 插件共用；API 投影由 serializer.PolarisInstanceInfoOutputObj.FromModel 完成
type MatchedInstance struct {
	ServiceNamespace  string
	ServiceName       string
	IP                string
	Port              uint32
	IsHealthy         bool
	Weight            int64
	IsIsolated        bool
	EnableHealthCheck bool
	Metadata          map[string]string
}

// InstanceMatcher 按实例 IP 索引北极星结果，供按 Pod 查询时复用
type InstanceMatcher struct {
	byIP map[string][]ipMatch
}

// ipMatch 一条北极星实例与其所属服务的配对，命中 IP 后再过滤端口
type ipMatch struct {
	svc  *PolarisServiceInstances
	inst *polarisInfra.Instance
}

// NewInstanceMatcher 按 IP 建索引；svc / inst 为 nil 的条目跳过
func NewInstanceMatcher(svcInstances []*PolarisServiceInstances) *InstanceMatcher {
	byIP := make(map[string][]ipMatch)
	for _, svc := range svcInstances {
		if svc == nil {
			continue
		}

		for _, inst := range svc.Instances {
			if inst == nil {
				continue
			}

			byIP[inst.IP] = append(byIP[inst.IP], ipMatch{svc: svc, inst: inst})
		}
	}

	return &InstanceMatcher{byIP: byIP}
}

// ForIP 返回该 Pod IP 下端口匹配的实例，按服务坐标定序
// 未命中是空切片而不是 nil，与「配置被删后成功拉回空」同形
func (m *InstanceMatcher) ForIP(ip string) []*MatchedInstance {
	if m == nil {
		return []*MatchedInstance{}
	}

	infos := buildMatchedInstances(m.byIP[ip])
	if infos == nil {
		return []*MatchedInstance{}
	}

	return infos
}

// buildMatchedInstances 过滤端口并按 (serviceNamespace, serviceName, port) 定序
// 北极星侧返回顺序不保证稳定；Watch 周期比对靠前后两次结果，顺序漂移会被误判成变化
func buildMatchedInstances(matches []ipMatch) []*MatchedInstance {
	var infos []*MatchedInstance
	for _, item := range matches {
		if int64(item.inst.Port) != int64(item.svc.ServicePort) {
			continue
		}

		infos = append(infos, &MatchedInstance{
			ServiceNamespace:  item.svc.ServiceNamespace,
			ServiceName:       item.svc.ServiceName,
			IP:                item.inst.IP,
			Port:              item.inst.Port,
			IsHealthy:         item.inst.IsHealthy,
			Weight:            int64(item.inst.Weight),
			IsIsolated:        item.inst.IsIsolated,
			EnableHealthCheck: item.inst.EnableHealthCheck,
			Metadata:          item.inst.Metadata,
		})
	}

	slices.SortFunc(infos, func(a, b *MatchedInstance) int {
		return cmp.Or(
			cmp.Compare(a.ServiceNamespace, b.ServiceNamespace),
			cmp.Compare(a.ServiceName, b.ServiceName),
			cmp.Compare(a.Port, b.Port),
		)
	})

	return infos
}
