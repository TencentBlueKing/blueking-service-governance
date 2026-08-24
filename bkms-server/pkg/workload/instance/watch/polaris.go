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

package watch

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// resyncPolaris 周期补拉北极星，只对 polarisInfos 相对上次推送有变化的实例补推 MODIFIED
// 拉取失败跳过本轮：不推事件、不拆流，页面保留上次已知的北极星状态，不会被清成空数组
// 只遍历已推送过的实例，因此既不会补出从未成功推送过的 ADDED，也不会推已 DELETED 的实例
func (m *Manager) resyncPolaris(
	ctx context.Context,
	stream *sseStream,
	pushed *pushedInstances,
) error {
	// 还没成功推送过任何实例时无从比对，省掉一次无谓拉取
	if pushed.empty() {
		return nil
	}

	// 拉取失败跳过本轮：不推事件、不拆流，页面保留上次已知的北极星状态
	svcInstances, ok := m.fetchPolaris(ctx)
	if !ok {
		return nil
	}

	for _, instance := range pushed.sorted() {
		// 拿一个只带 IP 的探针过一遍合并算法，用与 Pod 事件完全相同的匹配规则重算北极星投影
		// 走同一个函数是为了让字段口径与排序一致：两条口径算出不同结果就会被误判成变化
		probe := &serializer.AppInstanceOutputObj{
			IP:           instance.IP,
			PolarisInfos: []*serializer.PolarisInstanceInfoOutputObj{},
		}
		serializer.MergePolarisInfoToAppInstances([]*serializer.AppInstanceOutputObj{probe}, svcInstances)

		if slices.EqualFunc(instance.PolarisInfos, probe.PolarisInfos, samePolarisInfo) {
			continue
		}

		// K8s 展示字段沿用上次已知投影，只换北极星部分后整条重推
		instance.PolarisInfos = probe.PolarisInfos
		if err := stream.writeEvent(serializer.AppInstanceWatchEvent{
			Type: string(EventModified), Object: instance,
		}); err != nil {
			return err
		}
	}

	return nil
}

// attachPolaris 给单条投影挂北极星；拉取失败时保持空数组，不返回 error
// 一个 tick 周期内复用上一轮结果，不为每条 Pod 事件重拉，理由见下方注释
func (m *Manager) attachPolaris(ctx context.Context, instance *serializer.AppInstanceOutputObj) {
	// 先写成空数组，避免 JSON 出现 polarisInfos: null
	instance.PolarisInfos = []*serializer.PolarisInstanceInfoOutputObj{}

	// Pod 事件很密：一个 Pod 从 Pending 到 Ready 就有多条 MODIFIED，滚动更新会瞬间涌入几百条
	// 而每次拉取都要查一次北极星配置库，且北极星 SDK 自身有 15s 缓存，比这更密的拉取
	// 拿回来的是同一份数据。因此缓存未过期就直接复用，新实例的北极星信息由下一轮补拉补上
	svcInstances := m.polarisCache
	if m.polarisCache == nil || time.Since(m.polarisCachedAt) >= m.tickInterval {
		fresh, ok := m.fetchPolaris(ctx)
		if !ok {
			// 拉不到北极星不阻塞该 Pod 推送：保持空数组，与「未注册北极星」同形，由前端展示为未知
			return
		}

		svcInstances = fresh
	}

	// 按 Pod IP + 服务端口匹配；未命中时仍保持上面的空数组
	serializer.MergePolarisInfoToAppInstances([]*serializer.AppInstanceOutputObj{instance}, svcInstances)
}

// fetchPolaris 真拉一轮该应用环境的北极星实例，成功后刷新连接级缓存
// 拉取失败按「本轮不可用」处理而不是流错误：调用方跳过本轮，既不推事件也不拆流
// 失败不覆盖缓存，避免把上一轮拿到的真实状态冲成空
func (m *Manager) fetchPolaris(ctx context.Context) ([]*polaris.PolarisServiceInstances, bool) {
	svcInstances, err := m.listPolaris(ctx)
	if err != nil {
		// 失败被上层吞掉，不会体现在响应里，只能靠日志排查
		log.WarnAttrs(ctx, "list polaris service instances failed, fallback to unknown polaris state",
			slog.String("err", err.Error()),
		)

		metrics.InstanceWatchPolarisRefetch(false)

		return nil, false
	}

	// nil 归一成空切片，让 polarisCache == nil 只表示从未成功缓存
	if svcInstances == nil {
		svcInstances = []*polaris.PolarisServiceInstances{}
	}

	m.polarisCache, m.polarisCachedAt = svcInstances, time.Now()

	metrics.InstanceWatchPolarisRefetch(true)

	return svcInstances, true
}

// samePolarisInfo 逐字段比较单条北极星投影；Metadata 是 map，不能直接用 ==
// 单拎出来是为了给上面的 slices.EqualFunc 当比较函数，内联成匿名函数会淹掉那段循环
func samePolarisInfo(a, b *serializer.PolarisInstanceInfoOutputObj) bool {
	return a.ServiceNamespace == b.ServiceNamespace &&
		a.ServiceName == b.ServiceName &&
		a.IP == b.IP &&
		a.Port == b.Port &&
		a.IsHealthy == b.IsHealthy &&
		a.Weight == b.Weight &&
		a.IsIsolated == b.IsIsolated &&
		a.EnableHealthCheck == b.EnableHealthCheck &&
		maps.Equal(a.Metadata, b.Metadata)
}
