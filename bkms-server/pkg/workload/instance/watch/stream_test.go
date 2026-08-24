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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
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

func failingPolaris(context.Context) ([]*polaris.PolarisServiceInstances, error) {
	return nil, errors.New("polaris down")
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

// stubPolaris 按第几次调用返回不同的北极星结果，并把每次调用送进 visited 供测试做同步
// 拉取发生在 Manager 的 goroutine 上，故计数用 atomic
type stubPolaris struct {
	calls   atomic.Int64
	result  func(call int) ([]*polaris.PolarisServiceInstances, error)
	visited chan int
}

func newStubPolaris(result func(call int) ([]*polaris.PolarisServiceInstances, error)) *stubPolaris {
	return &stubPolaris{result: result, visited: make(chan int, 64)}
}

func (s *stubPolaris) list(context.Context) ([]*polaris.PolarisServiceInstances, error) {
	call := int(s.calls.Add(1))
	s.visited <- call

	return s.result(call)
}

// polarisSvc 构造匹配 validPod 的单条北极星注册；healthy 用来制造前后两轮的差异
func polarisSvc(healthy bool) []*polaris.PolarisServiceInstances {
	return []*polaris.PolarisServiceInstances{{
		ServiceNamespace: "Production",
		ServiceName:      "demo-app",
		ServicePort:      8080,
		Instances: []*polarisInfra.Instance{
			{IP: "127.0.0.1", Port: 8080, Weight: 100, IsHealthy: healthy, EnableHealthCheck: true},
		},
	}}
}

// runToEnd 跑完已入队的事件并收流；调用前必须 close(ch)，否则会一直等下去
func runToEnd(m *Manager) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() { done <- m.Run(context.Background(), rec, testParams) }()
	Eventually(done).Should(Receive(BeNil()))

	return rec
}

// runUntilResynced 用极短补拉周期跑一条 Watch，等补拉到第 untilCall 次后中断集群通道并收流
// 补拉在 select 分支里串行执行，因此收到第 N 次调用信号时，第 N-1 轮的推送必然已写完
func runUntilResynced(w *stubPodWatch, stub *stubPolaris, untilCall int) *httptest.ResponseRecorder {
	m := NewManager(w, stub.list)
	m.tickInterval = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() { done <- m.Run(context.Background(), rec, testParams) }()

	for range untilCall {
		Eventually(stub.visited).Should(Receive())
	}
	close(w.ch)
	Eventually(done).Should(Receive(BeNil()))

	return rec
}

func parseSSE(rec *httptest.ResponseRecorder) []serializer.AppInstanceWatchEvent {
	var events []serializer.AppInstanceWatchEvent
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var ev serializer.AppInstanceWatchEvent
		Expect(json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev)).To(Succeed())
		events = append(events, ev)
	}

	return events
}

// eventTypes 事件类型序列，用来一次性断言整条流的形状
func eventTypes(events []serializer.AppInstanceWatchEvent) []string {
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

		err := NewManager(&stubPodWatch{err: errors.New("dial cluster")}, failingPolaris).
			Run(context.Background(), rec, testParams)

		Expect(err).To(HaveOccurred())
		Expect(rec.Header().Get("Content-Type")).To(BeEmpty())
	})

	// 位点过期尚未成流：必须返回哨兵而不是写成 200 SSE，handler 才能映射 409
	It("returns ErrResourceVersionGone when the watch start is expired", func() {
		rec := httptest.NewRecorder()
		expired := pkgerrors.Wrap(k8serrors.NewResourceExpired("too old"), "watch pods")

		err := NewManager(&stubPodWatch{err: expired}, failingPolaris).
			Run(context.Background(), rec, testParams)

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

		err := NewManager(&stubPodWatch{err: gone}, failingPolaris).
			Run(context.Background(), rec, testParams)

		Expect(err).To(MatchError(ErrResourceVersionGone))
		Expect(rec.Header().Get("Content-Type")).To(BeEmpty())
	})

	// 连接硬上限到期：通道仍开着也要推 ENDED，前端按断流口径重新 List
	It("ends the stream with ENDED when the max age is reached", func() {
		w := newStubPodWatch()
		m := NewManager(w, failingPolaris)
		m.maxAge = 20 * time.Millisecond

		rec := httptest.NewRecorder()
		done := make(chan error, 1)
		go func() { done <- m.Run(context.Background(), rec, testParams) }()
		Eventually(done).Should(Receive(BeNil()))

		events := parseSSE(rec)
		Expect(eventTypes(events)).To(Equal([]string{"ENDED"}))
		Expect(events[0].Reason).To(Equal(watchTimeoutEndedReason))
	})

	// 坏 Pod / BOOKMARK 跳过、北极星失败降级空数组、DELETED 只留 id
	// 通道关闭视为集群中断：补 ENDED 后收流，不再向调用方返回错误
	It("skips bad events, degrades polaris, and ends the stream with ENDED", func() {
		w := newStubPodWatch()
		w.send(k8swatch.Added, badPod("bad-pod"))
		w.send(k8swatch.Bookmark, validPod("pod-1"))
		w.send(k8swatch.Added, validPod("pod-2"))
		w.send(k8swatch.Deleted, validPod("pod-gone"))
		close(w.ch)

		rec := runToEnd(NewManager(w, failingPolaris))
		Expect(rec.Header().Get("Content-Type")).To(Equal("text/event-stream"))

		events := parseSSE(rec)
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "DELETED", "ENDED"}))
		Expect(events[0].Object.ID).To(Equal("pod-2"))
		// 北极星拉不到降级为空数组而不是 null
		Expect(events[0].Object.PolarisInfos).To(Equal([]*serializer.PolarisInstanceInfoOutputObj{}))
		// DELETED 只保证 id，ENDED 不带 object
		Expect(events[1].Object.ID).To(Equal("pod-gone"))
		Expect(events[1].Object.IP).To(BeEmpty())
		Expect(events[2].Object).To(BeNil())
	})

	// Pod 事件很密：一个 Pod 从 Pending 到 Ready 就有多条 MODIFIED，滚动更新会瞬间涌入几百条
	// 每条都查一次北极星配置库既压不住，也拿不到更新的数据（SDK 自身有 15s 缓存）
	// 这里用默认 15s 的 tick，测试期间不会发生周期补拉，拉取次数只反映 Pod 事件
	It("reuses the last polaris result across pod events within one tick", func() {
		w := newStubPodWatch()
		stub := newStubPolaris(func(int) ([]*polaris.PolarisServiceInstances, error) {
			return polarisSvc(true), nil
		})
		for _, name := range []string{"pod-1", "pod-2", "pod-3"} {
			w.send(k8swatch.Added, validPod(name))
		}
		close(w.ch)

		events := parseSSE(runToEnd(NewManager(w, stub.list)))

		// 三条事件只拉了一次，且复用缓存没让后两条丢掉北极星信息
		Expect(int(stub.calls.Load())).To(Equal(1))
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "ADDED", "ADDED", "ENDED"}))
		for i := range 3 {
			Expect(events[i].Object.PolarisInfos).To(HaveLen(1))
		}
	})

	// Pod 没动、只有北极星健康位变化时也要能推出去；同一状态再补一轮则不再重复推
	It("pushes MODIFIED when only polaris changed, and stays quiet when it does not", func() {
		w := newStubPodWatch()
		stub := newStubPolaris(func(call int) ([]*polaris.PolarisServiceInstances, error) {
			// 第 1 次是 Pod 事件的合并，之后的补拉都改成不健康
			return polarisSvc(call == 1), nil
		})
		w.send(k8swatch.Added, validPod("pod-1"))

		// 等到第 3 次调用（第 2 轮补拉）开始，确保第 1 轮补拉的推送已经落地
		events := parseSSE(runUntilResynced(w, stub, 3))

		// 只有一条补推：第 2 轮北极星与上次推送的一致，不再重复推同样的状态
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "MODIFIED", "ENDED"}))
		Expect(events[0].Object.PolarisInfos[0].IsHealthy).To(BeTrue())
		Expect(events[1].Object.PolarisInfos[0].IsHealthy).To(BeFalse())
		// K8s 展示字段沿用上次已知投影，不因补拉丢失
		Expect(events[1].Object.Image).To(Equal("example:v1"))
	})

	// 补拉失败只跳过本轮：不拆流，也不能推空 polarisInfos 把页面上已有的状态清掉
	It("skips the round without clearing polaris when the refetch fails", func() {
		w := newStubPodWatch()
		stub := newStubPolaris(func(call int) ([]*polaris.PolarisServiceInstances, error) {
			if call == 1 {
				return polarisSvc(true), nil
			}

			return nil, errors.New("polaris down")
		})
		w.send(k8swatch.Added, validPod("pod-1"))

		events := parseSSE(runUntilResynced(w, stub, 3))

		// 中间没有任何 MODIFIED，前端本地保留的仍是上一次的非空 polarisInfos
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "ENDED"}))
		Expect(events[0].Object.PolarisInfos).To(HaveLen(1))
	})

	// 补拉只认已成功推送过的实例：投影失败的从未被记录，已 DELETED 的要被移出
	// 留一个存活实例，既能触发补拉，又能反衬出被跳过和已删的两个不该被推
	It("never resyncs instances that were skipped or already deleted", func() {
		w := newStubPodWatch()
		stub := newStubPolaris(func(call int) ([]*polaris.PolarisServiceInstances, error) {
			// 前两次先给空，补拉时才有注册；已删实例若仍参与比对就会误推 MODIFIED
			if call <= 2 {
				return nil, nil
			}

			return polarisSvc(true), nil
		})
		w.send(k8swatch.Added, badPod("bad-pod"))
		w.send(k8swatch.Added, validPod("pod-alive"))
		w.send(k8swatch.Added, validPod("pod-gone"))
		w.send(k8swatch.Deleted, validPod("pod-gone"))

		events := parseSSE(runUntilResynced(w, stub, 4))

		// 只有两条 ADDED，即 bad-pod 没被推出去
		Expect(eventTypes(events)).To(Equal([]string{"ADDED", "ADDED", "DELETED", "MODIFIED", "ENDED"}))

		// 补拉只推还活着的那个：bad-pod 没被记录过，pod-gone 已被移出
		Expect(events[3].Object.ID).To(Equal("pod-alive"))
		Expect(events[3].Object.PolarisInfos).To(HaveLen(1))
	})
})
