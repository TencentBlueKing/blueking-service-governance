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
	"net/http"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

const (
	// streamTickInterval 一次 tick 同时驱动 SSE 心跳与全部插件的一轮拉取
	// 15s 取自当前唯一插件（北极星 SDK）的缓存量级，补得更密只会空转
	// 多插件暂共用这一周期；有独立节奏需求时再按插件拆 ticker
	streamTickInterval = 15 * time.Second
	// streamMaxAge 单条 SSE 连接的硬性上限；到期推 ENDED，不向客户端承诺更长连接
	streamMaxAge = 2 * time.Minute
	// seedListTimeout 建流时预置 List 的上限；它在写 SSE 响应头之前，取一个前端可接受的建流等待
	// 超时按预置失败降级，不拖慢建流
	seedListTimeout = 5 * time.Second
	// seedListResourceVersion 预置 List 从 apiserver watch cache 读，不穿透 etcd
	// 也是唯一允许与 continue 翻页同时出现的非空位点，实例超过一页时不会被 apiserver 拒绝
	seedListResourceVersion = "0"
	// ENDED 事件的 reason，前端据此识别集群中断并重连
	clusterWatchEndedReason = "cluster watch interrupted"
	// watchTimeoutEndedReason 连接达到 maxAge 时的 ENDED reason；前端同样重新 List 再建 Watch
	watchTimeoutEndedReason = "watch timeout"
)

// errClusterInterrupted 已成流后集群 Watch 断开；内部信号，对外表现为推 ENDED 后收流
var errClusterInterrupted = errors.New("cluster watch interrupted")

// errStreamTimedOut 已成流后到达连接硬上限；内部信号，对外表现为推 ENDED 后收流
var errStreamTimedOut = errors.New("watch timeout")

// errSkipEvent 该事件不投影给前端，但流继续
var errSkipEvent = errors.New("skip watch event")

// Manager 将集群 Pod Watch 投影为 SSE 事件，是 Watch 的基础层
// 只管 Pod 的 list/watch 与投影推送，不感知任何附属数据源；附属数据由插件层补充
// 连接级对象：每条 Watch 请求新建一个，字段只在 consume 的单 goroutine 内读写，故不加锁
type Manager struct {
	podSource PodSource
	// pluginRunner 附属数据插件执行时；每个 tick 驱动一轮，产出独立的 PLUGIN 事件
	pluginRunner *plugin.Runner
	// tickInterval 心跳与插件层共用的周期；单测调小以免真实等待
	tickInterval time.Duration
	// maxAge 本条 SSE 的最长存活时间；单测调小以免真实等待
	maxAge time.Duration
	// seedListTimeout 建流时预置 List 的上限
	seedListTimeout time.Duration
}

// NewManager 创建 Watch 投影流 Manager；plugins 为附属数据插件，按注册顺序驱动
// 基础层不认识具体插件，注册在 handler 组装时完成
func NewManager(podSource PodSource, plugins ...plugin.Plugin) *Manager {
	return &Manager{
		podSource:       podSource,
		pluginRunner:    plugin.NewRunner(plugins...),
		tickInterval:    streamTickInterval,
		maxAge:          streamMaxAge,
		seedListTimeout: seedListTimeout,
	}
}

// Run 从续传位点起把集群 Pod 变更投影为 SSE，直到客户端断开、集群中断或达到连接硬上限
// 返回 error 只代表流未建立，调用方按普通 HTTP 错误返回；已成流后的中断先推 ENDED 再收流
func (m *Manager) Run(ctx context.Context, w http.ResponseWriter, p RunParams) error {
	stream, err := newSSEStream(ctx, w)
	if err != nil {
		return err
	}

	// 预置插件层快照：前端首包已有这些实例，Watch 只推位点之后的增量
	// 不预置的话 Pod 一直不动时快照恒为空，北极星等附属数据的变化永远推不出去
	// 放在建 Watch 之前：集群 Watch 的结果通道无缓冲，建好后迟迟不消费会把事件堵在读端
	// 这里只有一次 List、不拉插件，建流前最多多等 seedListTimeout
	pushed := m.seedLiveInstances(ctx, p)

	// 前端首包里的附属数据可能已降级为空，也可能早于本连接的拉取，不能拿来当对比基线
	// 标为待同步：首个 tick 拿到值后对它们无条件推一次，之后再按深比较只推变化
	m.pluginRunner.MarkUnsynced(pushed.ids()...)

	// 再建立集群 Watch；此时尚未写响应头，失败仍可返回普通 HTTP 错误
	// TimeoutSeconds 与 SSE 硬上限对齐，避免集群侧 Watch 比本条连接活得更久
	timeoutSeconds := m.watchTimeoutSeconds()
	watcher, err := m.podSource.Watch(ctx, p.Namespace, metav1.ListOptions{
		LabelSelector:   p.LabelSelector,
		ResourceVersion: p.ResourceVersion,
		TimeoutSeconds:  &timeoutSeconds,
	})
	if err != nil {
		return wrapWatchStartErr(err, p.ResourceVersion)
	}
	defer watcher.Stop()

	// Watch 建立成功才写 SSE 头，避免失败响应被当成 200 事件流
	stream.writeHeaders()

	// 成流后才计入活跃连接；建流失败没有 SSE，不涨 Gauge
	metrics.InstanceWatchStarted()
	defer metrics.InstanceWatchFinished()

	err = m.consume(ctx, stream, watcher, pushed, p.DeployID)

	// 已成流后的集群中断或到达硬上限：补一条 ENDED 供前端识别并重连
	if reason, ok := endedReason(err); ok {
		err = stream.writeEvent(serializer.AppInstanceWatchEvent{
			Type:   string(EventEnded),
			Reason: reason,
		}, string(EventEnded))
	}

	// 响应头已写出，改不了 HTTP 状态码，流内错误只能落日志，否则写失败会彻底无声
	if err != nil {
		log.WarnAttrs(ctx, "instance watch stream ended with error",
			slog.String("namespace", p.Namespace),
			slog.String("label_selector", p.LabelSelector),
			slog.String("deploy_id", p.DeployID),
			slog.String("err", err.Error()),
		)
	}

	return nil
}

// watchTimeoutSeconds 把 maxAge 换成 k8s TimeoutSeconds；不足 1s 时取 1，避免单测毫秒级 maxAge 传 0
func (m *Manager) watchTimeoutSeconds() int64 {
	seconds := int64(m.maxAge / time.Second)
	seconds = lo.Ternary(seconds < 1, 1, seconds)
	return seconds
}

// wrapWatchStartErr 建流失败尚未写头；位点过期单独成哨兵，其余原样包装
func wrapWatchStartErr(err error, resourceVersion string) error {
	if k8serrors.IsGone(err) || k8serrors.IsResourceExpired(err) {
		return errors.Wrapf(ErrResourceVersionGone, "watch pods from resourceVersion %s: %s", resourceVersion, err)
	}

	return errors.Wrapf(err, "watch pods from resourceVersion %s", resourceVersion)
}

// endedReason 已成流后需要补 ENDED 的内部信号；客户端断开不在此列
func endedReason(err error) (string, bool) {
	switch {
	case errors.Is(err, errClusterInterrupted):
		return clusterWatchEndedReason, true
	case errors.Is(err, errStreamTimedOut):
		return watchTimeoutEndedReason, true
	default:
		return "", false
	}
}

// seedLiveInstances 取当前存活实例作为插件层的初始快照，只记录不推事件
// 结果与续传位点可能有出入，由后续事件纠正：ADDED 经 Runner.MarkUnsynced 重新同步，DELETED 出栈；
// 偶尔收不到 DELETED 的残留实例只会产生前端忽略的 PLUGIN
// 拉取失败或超时只告警并返回空记录，不拆流
func (m *Manager) seedLiveInstances(ctx context.Context, p RunParams) *pushedInstances {
	pushed := newPushedInstances()

	listCtx, cancel := context.WithTimeout(ctx, m.seedListTimeout)
	defer cancel()

	pods, err := m.podSource.List(listCtx, p.Namespace, metav1.ListOptions{
		LabelSelector:   p.LabelSelector,
		ResourceVersion: seedListResourceVersion,
	})
	if err != nil {
		// 客户端已断开导致的失败是正常收流，不告警
		if ctx.Err() == nil {
			log.WarnAttrs(ctx, "list live pods for watch plugin seed failed, plugins only see pushed instances",
				slog.String("namespace", p.Namespace),
				slog.String("label_selector", p.LabelSelector),
				slog.String("err", err.Error()),
			)
		}

		return pushed
	}

	// 与 Pod 事件同一套投影口径：投影失败的坏 Pod 不进快照，插件也就不会为它补数据
	for _, pod := range pods.Items {
		instance, pErr := new(serializer.AppInstanceOutputObj).FromPodManifest(pod.Object, p.DeployID)
		if pErr != nil {
			continue
		}

		pushed.seed(instance)
	}

	return pushed
}

// consume 消费集群事件并写 SSE；ctx 取消安静退出，通道关闭视为集群中断，到期视为超时
// pushed 是连接级记录，存已推送出去（含建流时预置）的投影；随本次 consume 结束一起释放
func (m *Manager) consume(
	ctx context.Context,
	stream *sseStream,
	watcher k8swatch.Interface,
	pushed *pushedInstances,
	deployID string,
) error {
	ticker := time.NewTicker(m.tickInterval)
	defer ticker.Stop()

	maxAge := time.NewTimer(m.maxAge)
	defer maxAge.Stop()

	for {
		select {
		case <-ctx.Done():
			// 客户端断开不必再写 ENDED，由前端 onerror/onclose 重连
			return nil

		case <-maxAge.C:
			return errStreamTimedOut

		case <-ticker.C:
			// 先跑一轮插件层，只对附属数据有变化的实例推 PLUGIN；单个插件失败只跳过本轮
			if err := m.runPlugins(ctx, stream, pushed); err != nil {
				return err
			}

			// 再写一次 SSE 注释心跳，避免空闲连接被代理断开
			if err := stream.writeHeartbeat(); err != nil {
				return err
			}

		case ev, ok := <-watcher.ResultChan():
			if !ok {
				return errClusterInterrupted
			}

			// 投影失败或 BOOKMARK 用 errSkipEvent 吞掉，不拆流
			err := m.handleEvent(stream, pushed, ev, deployID)
			if err != nil && !errors.Is(err, errSkipEvent) {
				return err
			}
		}
	}
}

// handleEvent 把单条集群事件投影为 SSE；BOOKMARK 跳过，Error 视为集群中断
func (m *Manager) handleEvent(
	stream *sseStream,
	pushed *pushedInstances,
	ev k8swatch.Event,
	deployID string,
) error {
	// BOOKMARK 只推进位点，不向页面推实例事件
	if ev.Type == k8swatch.Bookmark {
		return errSkipEvent
	}

	// 集群侧 Error 事件与通道关闭同样按中断处理
	if ev.Type == k8swatch.Error {
		return errClusterInterrupted
	}

	// 投影失败返回 errSkipEvent，调用方继续消费后续事件
	event, err := m.projectEvent(ev, deployID)
	if err != nil {
		return err
	}

	if err = stream.writeEvent(event, event.Type); err != nil {
		return err
	}

	// 推送成功后才记录：没推出去的实例不进插件层快照，也就不会被补出附属数据事件
	pushed.track(event)

	// 前端按 ADDED 新建的行附属数据为空，重复 ADDED 则保留旧行的附属数据；两种情况服务端都分不清，
	// 已推送记录不再等于页面上显示的值，标为待同步，下一轮拿到值后直接推一次
	if event.Type == string(EventAdded) {
		m.pluginRunner.MarkUnsynced(event.Object.ID)
	}

	return nil
}

// runPlugins 把本连接存活实例的快照交给插件层跑一轮，有变化的实例推独立 PLUGIN 事件
// 快照只含建流时预置或已成功推送、且未 DELETED 的实例，因此不会补出投影失败的实例，也不会推已删除的
// 插件调用留在 consume 的同一 goroutine 内顺序执行：pushedInstances 与 Runner 的免锁前提
// 正是「Pod 事件与周期任务是同一 goroutine 的两个 case」
func (m *Manager) runPlugins(ctx context.Context, stream *sseStream, pushed *pushedInstances) error {
	// 应用环境下没有存活实例、也没推送过任何实例时快照必为空，省掉一次排序与遍历
	if pushed.empty() {
		return nil
	}

	return m.pluginRunner.Run(
		ctx,
		pushed.sorted(),
		func(event serializer.AppInstancePluginWatchEvent) error {
			return stream.writeEvent(event, event.Type)
		},
	)
}

// projectEvent 把集群对象投影为平台事件；无法识别或投影失败则跳过，不推 skipped
func (m *Manager) projectEvent(ev k8swatch.Event, deployID string) (serializer.AppInstanceWatchEvent, error) {
	// dynamic client Watch 只应给出 Unstructured，其他类型不投影
	obj, ok := ev.Object.(*unstructured.Unstructured)
	if !ok {
		return serializer.AppInstanceWatchEvent{}, errSkipEvent
	}

	podName := mapx.GetStr(obj.Object, "metadata.name")
	if podName == "" {
		// 没有 name 无法定位实例，跳过
		return serializer.AppInstanceWatchEvent{}, errSkipEvent
	}

	// DELETED 只保证 object.id，不投影其余字段、不拉北极星
	if ev.Type == k8swatch.Deleted {
		return serializer.AppInstanceWatchEvent{
			Type:   string(EventDeleted),
			Object: &serializer.AppInstanceOutputObj{ID: podName},
		}, nil
	}

	// ADDED/MODIFIED 对齐 List 的 AppInstanceOutputObj；单个坏 Pod 跳过本事件
	instance, err := new(serializer.AppInstanceOutputObj).FromPodManifest(obj.Object, deployID)
	if err != nil {
		return serializer.AppInstanceWatchEvent{}, errSkipEvent
	}

	// Pod 事件不承载附属数据：置空数组而不是留 nil，避免 JSON 出现 polarisInfos: null
	// 北极星等附属信息由插件层以独立 PLUGIN 事件推送，前端不得从这里读取
	instance.PolarisInfos = []*serializer.PolarisInstanceInfoOutputObj{}

	eventType := EventModified
	if ev.Type == k8swatch.Added {
		eventType = EventAdded
	}

	return serializer.AppInstanceWatchEvent{Type: string(eventType), Object: instance}, nil
}
