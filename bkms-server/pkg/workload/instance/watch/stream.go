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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

const (
	// streamTickInterval 一次 tick 同时驱动 SSE 心跳与北极星补拉
	// 15s 取自北极星 SDK 的缓存量级，补得更密只会空转
	streamTickInterval = 15 * time.Second
	// streamMaxAge 单条 SSE 连接的硬性上限；到期推 ENDED，不向客户端承诺更长连接
	streamMaxAge = 2 * time.Minute
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

// Manager 将集群 Pod Watch 投影为 SSE 事件
// 连接级对象：每条 Watch 请求新建一个，字段只在 consume 的单 goroutine 内读写，故不加锁
type Manager struct {
	podWatcher  PodWatcher
	listPolaris PolarisLister
	// tickInterval 心跳与北极星补拉共用的周期；单测调小以免真实等待
	tickInterval time.Duration
	// maxAge 本条 SSE 的最长存活时间；单测调小以免真实等待
	maxAge time.Duration
	// polarisCache 最近一轮北极星拉取结果，供密集的 Pod 事件复用，见 attachPolaris
	// nil 表示从未成功缓存；成功拉取即使空结果也写成空切片，与 nil 区分
	polarisCache []*polaris.PolarisServiceInstances
	// polarisCachedAt polarisCache 的取回时刻；配合 polarisCache == nil 判断是否该真拉
	polarisCachedAt time.Time
}

// NewManager 创建 Watch 投影流 Manager
func NewManager(podWatcher PodWatcher, listPolaris PolarisLister) *Manager {
	return &Manager{
		podWatcher:   podWatcher,
		listPolaris:  listPolaris,
		tickInterval: streamTickInterval,
		maxAge:       streamMaxAge,
	}
}

// Run 从续传位点起把集群 Pod 变更投影为 SSE，直到客户端断开、集群中断或达到连接硬上限
// 返回 error 只代表流未建立，调用方按普通 HTTP 错误返回；已成流后的中断先推 ENDED 再收流
func (m *Manager) Run(ctx context.Context, w http.ResponseWriter, p RunParams) error {
	stream, err := newSSEStream(ctx, w)
	if err != nil {
		return err
	}

	// 先建立集群 Watch；此时尚未写响应头，失败仍可返回普通 HTTP 错误
	// TimeoutSeconds 与 SSE 硬上限对齐，避免集群侧 Watch 比本条连接活得更久
	timeoutSeconds := m.watchTimeoutSeconds()
	watcher, err := m.podWatcher.Watch(ctx, p.Namespace, metav1.ListOptions{
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

	err = m.consume(ctx, stream, watcher, p.DeployID)

	// 已成流后的集群中断或到达硬上限：补一条 ENDED 供前端识别并重连
	if reason, ok := endedReason(err); ok {
		err = stream.writeEvent(serializer.AppInstanceWatchEvent{
			Type:   string(EventEnded),
			Reason: reason,
		})
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

// consume 消费集群事件并写 SSE；ctx 取消安静退出，通道关闭视为集群中断，到期视为超时
func (m *Manager) consume(
	ctx context.Context,
	stream *sseStream,
	watcher k8swatch.Interface,
	deployID string,
) error {
	ticker := time.NewTicker(m.tickInterval)
	defer ticker.Stop()

	maxAge := time.NewTimer(m.maxAge)
	defer maxAge.Stop()

	// 连接级记录，存已推送出去的投影；随本次 consume 结束一起释放
	pushed := newPushedInstances()

	for {
		select {
		case <-ctx.Done():
			// 客户端断开不必再写 ENDED，由前端 onerror/onclose 重连
			return nil

		case <-maxAge.C:
			return errStreamTimedOut

		case <-ticker.C:
			// 先补拉北极星，只对有变化的实例补推 MODIFIED；拉取失败只跳过本轮
			if err := m.resyncPolaris(ctx, stream, pushed); err != nil {
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
			err := m.handleEvent(ctx, stream, pushed, ev, deployID)
			if err != nil && !errors.Is(err, errSkipEvent) {
				return err
			}
		}
	}
}

// handleEvent 把单条集群事件投影为 SSE；BOOKMARK 跳过，Error 视为集群中断
func (m *Manager) handleEvent(
	ctx context.Context,
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
	event, err := m.projectEvent(ctx, ev, deployID)
	if err != nil {
		return err
	}

	if err = stream.writeEvent(event); err != nil {
		return err
	}

	// 推送成功后才记录：没推出去的实例不参与后续北极星补拉比对
	pushed.track(event)

	return nil
}

// projectEvent 把集群对象投影为平台事件；无法识别或投影失败则跳过，不推 skipped
func (m *Manager) projectEvent(
	ctx context.Context,
	ev k8swatch.Event,
	deployID string,
) (serializer.AppInstanceWatchEvent, error) {
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

	// 北极星失败不拆流，K8s 字段照常推
	m.attachPolaris(ctx, instance)

	eventType := EventModified
	if ev.Type == k8swatch.Added {
		eventType = EventAdded
	}

	return serializer.AppInstanceWatchEvent{Type: string(eventType), Object: instance}, nil
}
