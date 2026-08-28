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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

var testParams = RunParams{DeployID: "deploy-1", ResourceVersion: "100"}

// stubPodWatch 同时充当 PodWatcher 与它返回的 watch.Interface，省掉一层假 watcher
// 用例先把事件塞进 ch，close(ch) 即模拟集群中断
type stubPodWatch struct {
	ch  chan k8swatch.Event
	err error
}

func newStubPodWatch() *stubPodWatch {
	return &stubPodWatch{ch: make(chan k8swatch.Event, 8)}
}

func (s *stubPodWatch) Watch(context.Context, string, metav1.ListOptions) (k8swatch.Interface, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s, nil
}

func (s *stubPodWatch) Stop() {}

func (s *stubPodWatch) ResultChan() <-chan k8swatch.Event { return s.ch }

func (s *stubPodWatch) send(evType k8swatch.EventType, obj *unstructured.Unstructured) {
	s.ch <- k8swatch.Event{Type: evType, Object: obj}
}

// stubPlugin 桩插件：按第几次调用返回不同载荷，同时记录每次收到的快照
// 它也是「新增一个插件只需实现接口 + 注册，不动基础层」的证明
// 调用发生在 Manager 的 goroutine 上，故计数用 atomic，快照与调用信号走 channel
type stubPlugin struct {
	name      string
	calls     atomic.Int64
	fetch     func(call int) (map[string]any, error)
	visited   chan int
	snapshots chan []string
}

func newStubPlugin(name string, fetch func(call int) (map[string]any, error)) *stubPlugin {
	return &stubPlugin{
		name:      name,
		fetch:     fetch,
		visited:   make(chan int, 64),
		snapshots: make(chan []string, 64),
	}
}

func (s *stubPlugin) Name() string { return s.name }

func (s *stubPlugin) Fetch(
	_ context.Context,
	snapshot []plugin.InstanceSnapshot,
) (map[string]any, error) {
	ids := make([]string, 0, len(snapshot))
	for _, instance := range snapshot {
		ids = append(ids, instance.ID)
	}
	s.snapshots <- ids

	call := int(s.calls.Add(1))
	s.visited <- call

	return s.fetch(call)
}

func validPod(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": name, "creationTimestamp": "2026-05-29T00:00:00Z"},
		"spec":     map[string]any{"containers": []any{map[string]any{"image": "example:v1"}}},
		"status":   map[string]any{"podIP": "127.0.0.1", "phase": "Running"},
	}}
}

// badPod 缺 spec.containers，投影必失败；保留 name 才能确认它没被推出去
func badPod(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": name},
	}}
}

// sseEvent 通用事件信封：Pod 事件与附属数据事件共用一条流但字段不同，object 先留原样
type sseEvent struct {
	Type   string          `json:"type"`
	Reason string          `json:"reason"`
	Plugin string          `json:"plugin"`
	Object json.RawMessage `json:"object"`
}

// instance 把 Pod 事件的 object 解成实例投影
func (e sseEvent) instance() *serializer.AppInstanceOutputObj {
	var obj *serializer.AppInstanceOutputObj
	Expect(json.Unmarshal(e.Object, &obj)).To(Succeed())

	return obj
}

// pluginPayload 把附属数据事件的 object 解成实例 ID 与载荷
func (e sseEvent) pluginPayload() (string, []string) {
	var obj struct {
		ID   string   `json:"id"`
		Data []string `json:"data"`
	}
	Expect(json.Unmarshal(e.Object, &obj)).To(Succeed())

	return obj.ID, obj.Data
}

// runToEnd 跑完已入队的事件并收流；调用前必须 close(ch)，否则会一直等下去
func runToEnd(m *Manager) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() { done <- m.Run(context.Background(), rec, testParams) }()
	Eventually(done).Should(Receive(BeNil()))

	return rec
}

// runUntilFetched 用极短 tick 跑一条 Watch，等插件被调用到第 untilCall 次后中断集群通道并收流
// 插件层在 select 分支里串行执行，因此收到第 N 次调用信号时，第 N-1 轮的推送必然已写完
func runUntilFetched(w *stubPodWatch, p *stubPlugin, untilCall int) *httptest.ResponseRecorder {
	m := NewManager(w, p)
	m.tickInterval = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() { done <- m.Run(context.Background(), rec, testParams) }()

	for range untilCall {
		Eventually(p.visited).Should(Receive())
	}
	close(w.ch)
	Eventually(done).Should(Receive(BeNil()))

	return rec
}

func parseSSE(rec *httptest.ResponseRecorder) []sseEvent {
	var events []sseEvent
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var ev sseEvent
		Expect(json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev)).To(Succeed())
		events = append(events, ev)
	}

	return events
}

// eventTypes 事件类型序列，用来一次性断言整条流的形状
func eventTypes(events []sseEvent) []string {
	types := make([]string, 0, len(events))
	for _, ev := range events {
		types = append(types, ev.Type)
	}

	return types
}

var _ = Describe("Manager", func() {
	// 建立失败还没成流：不能写成 200 SSE，调用方应按普通 HTTP 错误返回
	It("does not write SSE headers when cluster Watch cannot start", func() {
		rec := httptest.NewRecorder()

		err := NewManager(&stubPodWatch{err: errors.New("dial cluster")}).
			Run(context.Background(), rec, testParams)

		Expect(err).To(HaveOccurred())
		Expect(rec.Header().Get("Content-Type")).To(BeEmpty())
	})

	// 位点过期尚未成流：必须返回哨兵而不是写成 200 SSE，handler 才能映射 409
	It("returns ErrResourceVersionGone when the watch start is expired", func() {
		rec := httptest.NewRecorder()
		expired := pkgerrors.Wrap(k8serrors.NewResourceExpired("too old"), "watch pods")

		err := NewManager(&stubPodWatch{err: expired}).Run(context.Background(), rec, testParams)

		Expect(err).To(MatchError(ErrResourceVersionGone))
		Expect(rec.Header().Get("Content-Type")).To(BeEmpty())
	})

	It("returns ErrResourceVersionGone when the watch start is gone", func() {
		rec := httptest.NewRecorder()
		gone := &k8serrors.StatusError{ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Reason:  metav1.StatusReasonGone,
			Code:    int32(http.StatusGone),
			Message: "gone",
		}}

		err := NewManager(&stubPodWatch{err: gone}).Run(context.Background(), rec, testParams)

		Expect(err).To(MatchError(ErrResourceVersionGone))
		Expect(rec.Header().Get("Content-Type")).To(BeEmpty())
	})

	// 连接硬上限到期：通道仍开着也要推 ENDED，前端按断流口径重新 List
	It("ends the stream with ENDED when the max age is reached", func() {
		w := newStubPodWatch()
		m := NewManager(w)
		m.maxAge = 20 * time.Millisecond

		rec := httptest.NewRecorder()
		done := make(chan error, 1)
		go func() { done <- m.Run(context.Background(), rec, testParams) }()
		Eventually(done).Should(Receive(BeNil()))

		events := parseSSE(rec)
		Expect(eventTypes(events)).To(Equal([]string{"ENDED"}))
		Expect(events[0].Reason).To(Equal(watchTimeoutEndedReason))
	})

	// 坏 Pod / BOOKMARK 跳过、DELETED 只留 id；通道关闭视为集群中断，补 ENDED 后收流
	// Pod 事件不再承载附属数据：polarisInfos 恒为空数组，前端只能从 PLUGIN 事件取
	It("skips bad events and never carries plugin data on pod events", func() {
		w := newStubPodWatch()
		w.send(k8swatch.Added, badPod("bad-pod"))
		w.send(k8swatch.Bookmark, validPod("pod-1"))
		w.send(k8swatch.Added, validPod("pod-2"))
		w.send(k8swatch.Deleted, validPod("pod-gone"))
		close(w.ch)

		rec := runToEnd(NewManager(w))
		Expect(rec.Header().Get("Content-Type")).To(Equal("text/event-stream"))

		events := parseSSE(rec)
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "DELETED", "ENDED"}))
		Expect(events[0].instance().ID).To(Equal("pod-2"))
		// 空数组而不是 null，也不会出现北极星数据
		Expect(events[0].instance().PolarisInfos).To(Equal([]*serializer.PolarisInstanceInfoOutputObj{}))
		// DELETED 只保证 id，ENDED 不带 object
		Expect(events[1].instance().ID).To(Equal("pod-gone"))
		Expect(events[1].instance().IP).To(BeEmpty())
		Expect(string(events[2].Object)).To(Equal("null"))
	})

	// Pod 事件不再触发附属数据拉取：默认 15s tick 下三条事件期间插件一次都不该被调到
	// 这是分层后「拉取次数不因 Pod 事件放大」的直接体现
	It("does not fetch plugins on pod events", func() {
		w := newStubPodWatch()
		p := newStubPlugin("stub", func(int) (map[string]any, error) { return nil, nil })
		for _, name := range []string{"pod-1", "pod-2", "pod-3"} {
			w.send(k8swatch.Added, validPod(name))
		}
		close(w.ch)

		events := parseSSE(runToEnd(NewManager(w, p)))

		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "ADDED", "ADDED", "ENDED"}))
		Expect(int(p.calls.Load())).To(BeZero())
	})

	// 插件每轮拿到的是本连接当前存活的全量实例，而不是本轮有变动的那几个
	// Pod 完全没动、只有附属数据变化时也必须能推出去
	It("hands the full set of live instances to plugins and pushes only changes", func() {
		w := newStubPodWatch()
		p := newStubPlugin("stub", func(call int) (map[string]any, error) {
			// 第 1 轮给两个实例同样的载荷，第 2 轮只改 pod-1
			if call == 1 {
				return map[string]any{"pod-1": []string{"v1"}, "pod-2": []string{"v1"}}, nil
			}

			return map[string]any{"pod-1": []string{"v2"}, "pod-2": []string{"v1"}}, nil
		})
		w.send(k8swatch.Added, validPod("pod-1"))
		w.send(k8swatch.Added, validPod("pod-2"))

		// 等到第 3 次调用开始，确保第 2 轮的推送已落地
		events := parseSSE(runUntilFetched(w, p, 3))

		// 插件两次都看到全部存活实例，且按 ID 定序
		Expect(p.snapshots).To(Receive(Equal([]string{"pod-1", "pod-2"})))
		Expect(eventTypes(events)).To(Equal([]string{
			"ADDED", "ADDED", "PLUGIN", "PLUGIN", "PLUGIN", "ENDED",
		}))

		// 首轮两个实例各推一条，第 2 轮只有 pod-1 变了
		id, data := events[4].pluginPayload()
		Expect(id).To(Equal("pod-1"))
		Expect(data).To(Equal([]string{"v2"}))
		Expect(events[4].Plugin).To(Equal("stub"))
	})

	// 快照只认已成功推送过的实例：投影失败的从未被记录，已 DELETED 的要被移出
	// 留一个存活实例，既能触发插件调用，又能反衬出被跳过和已删的两个不该进快照
	It("never hands skipped or deleted instances to plugins", func() {
		w := newStubPodWatch()
		p := newStubPlugin("stub", func(call int) (map[string]any, error) {
			// 前两轮先不给数据，之后才有；已删实例若仍在快照里就会被误推
			if call <= 2 {
				return nil, nil
			}

			return map[string]any{"pod-alive": []string{"v1"}}, nil
		})
		w.send(k8swatch.Added, badPod("bad-pod"))
		w.send(k8swatch.Added, validPod("pod-alive"))
		w.send(k8swatch.Added, validPod("pod-gone"))
		w.send(k8swatch.Deleted, validPod("pod-gone"))

		events := parseSSE(runUntilFetched(w, p, 4))

		// 只有两条 ADDED，即 bad-pod 没被推出去
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "ADDED", "DELETED", "PLUGIN", "ENDED"}))

		// 插件只看到还活着的那个：bad-pod 没被记录过，pod-gone 已被移出
		Expect(p.snapshots).To(Receive(Equal([]string{"pod-alive"})))
		id, _ := events[3].pluginPayload()
		Expect(id).To(Equal("pod-alive"))
	})
})
