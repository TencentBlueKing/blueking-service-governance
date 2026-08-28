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

package plugin

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// fakePlugin 每轮按调用次数返回预置载荷，用来驱动 Runner 的比对逻辑
type fakePlugin struct {
	name    string
	calls   int
	rounds  []map[string]any
	failAll bool
	mutate  func([]InstanceSnapshot)
}

func (f *fakePlugin) Name() string { return f.name }

func (f *fakePlugin) Fetch(_ context.Context, snapshot []InstanceSnapshot) (map[string]any, error) {
	if f.mutate != nil {
		f.mutate(snapshot)
	}

	if f.failAll {
		return nil, errors.New("fetch failed")
	}

	round := f.rounds[min(f.calls, len(f.rounds)-1)]
	f.calls++

	return round, nil
}

// blockUntilCancelPlugin 卡住直到 ctx 取消，用来验证 Runner 的拉取超时
type blockUntilCancelPlugin struct {
	name string
}

func (p *blockUntilCancelPlugin) Name() string { return p.name }

func (p *blockUntilCancelPlugin) Fetch(
	ctx context.Context,
	_ []InstanceSnapshot,
) (map[string]any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func snapshotOf(ids ...string) []*serializer.AppInstanceOutputObj {
	items := make([]*serializer.AppInstanceOutputObj, 0, len(ids))
	for _, id := range ids {
		items = append(items, &serializer.AppInstanceOutputObj{ID: id})
	}

	return items
}

// collect 把一轮产出的事件收进切片，供断言推送顺序与内容
func collect(events *[]serializer.AppInstancePluginWatchEvent) Emit {
	return func(event serializer.AppInstancePluginWatchEvent) error {
		*events = append(*events, event)

		return nil
	}
}

var _ = Describe("Runner", func() {
	ctx := context.Background()

	// 载荷里可能含 map（如北极星的 metadata），比对走深比较而不是 ==：内容相同就不该重复推
	It("pushes only instances whose payload changed", func() {
		p := &fakePlugin{name: "stub", rounds: []map[string]any{
			{"a": map[string]string{"k": "v1"}, "b": map[string]string{"k": "v1"}},
			{"a": map[string]string{"k": "v2"}, "b": map[string]string{"k": "v1"}},
			{"a": map[string]string{"k": "v2"}, "b": map[string]string{"k": "v1"}},
		}}
		runner := NewRunner(p)
		snapshot := snapshotOf("a", "b")

		var first, second, third []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshot, collect(&first))).To(Succeed())
		Expect(runner.Run(ctx, snapshot, collect(&second))).To(Succeed())
		Expect(runner.Run(ctx, snapshot, collect(&third))).To(Succeed())

		// 首轮两个实例都是新值，第二轮只有 a 变了，第三轮内容全都没变
		Expect(first).To(HaveLen(2))
		Expect(first[0].Object.ID).To(Equal("a"))
		Expect(first[0].Type).To(Equal(EventTypePlugin))
		Expect(first[0].Plugin).To(Equal("stub"))
		Expect(second).To(HaveLen(1))
		Expect(second[0].Object.ID).To(Equal("a"))
		Expect(second[0].Object.Data).To(Equal(map[string]string{"k": "v2"}))
		Expect(third).To(BeEmpty())
	})

	// 从来没有过数据的实例不推空载荷；一旦推过非空，转空是真实变化必须推出去
	It("suppresses the first empty payload but pushes a later emptying", func() {
		p := &fakePlugin{name: "stub", rounds: []map[string]any{
			{"a": []string{}},
			{"a": []string{"v1"}},
			{"a": []string{}},
		}}
		runner := NewRunner(p)

		var events []serializer.AppInstancePluginWatchEvent
		for range 3 {
			Expect(runner.Run(ctx, snapshotOf("a"), collect(&events))).To(Succeed())
		}

		Expect(events).To(HaveLen(2))
		Expect(events[0].Object.Data).To(Equal([]string{"v1"}))
		Expect(events[1].Object.Data).To(Equal([]string{}))
	})

	// 拉取失败跳过本轮：不推事件，也不能把已推送记录清掉，否则下一轮会重推同一份数据
	It("keeps the pushed record when a fetch fails", func() {
		p := &fakePlugin{name: "stub", rounds: []map[string]any{{"a": []string{"v1"}}}}
		runner := NewRunner(p)

		var first []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&first))).To(Succeed())

		p.failAll = true
		var failed []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&failed))).To(Succeed())

		p.failAll = false
		var recovered []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&recovered))).To(Succeed())

		Expect(first).To(HaveLen(1))
		Expect(failed).To(BeEmpty())
		// 恢复后载荷没变，不该因为失败轮而重推
		Expect(recovered).To(BeEmpty())
	})

	// 一个插件失败不影响其它插件本轮的推送
	It("isolates plugin failures", func() {
		broken := &fakePlugin{name: "broken", failAll: true}
		healthy := &fakePlugin{name: "healthy", rounds: []map[string]any{{"a": []string{"v1"}}}}
		runner := NewRunner(broken, healthy)

		var events []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&events))).To(Succeed())

		Expect(events).To(HaveLen(1))
		Expect(events[0].Plugin).To(Equal("healthy"))
	})

	// 实例从快照消失（已 DELETED）后记录随之出栈：再次出现时按新值重推，长连接不累积
	It("drops records for instances that left the snapshot", func() {
		p := &fakePlugin{name: "stub", rounds: []map[string]any{{"a": []string{"v1"}}}}
		runner := NewRunner(p)

		var first []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&first))).To(Succeed())
		// a 离开快照这一轮不产生事件，同时它的记录被裁掉
		var gone []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("b"), collect(&gone))).To(Succeed())

		var again []serializer.AppInstancePluginWatchEvent
		Expect(runner.Run(ctx, snapshotOf("a"), collect(&again))).To(Succeed())

		Expect(first).To(HaveLen(1))
		Expect(again).To(HaveLen(1))
	})

	// 超时与拉取失败同一口径：不推事件、不拆流，让心跳和 Pod 事件能继续
	It("treats a timed-out fetch as a skippable failure", func() {
		p := &blockUntilCancelPlugin{name: "slow"}
		runner := NewRunner(p)
		runner.fetchTimeout = 20 * time.Millisecond

		var events []serializer.AppInstancePluginWatchEvent
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx, snapshotOf("a"), collect(&events))
		}()

		Eventually(errCh, time.Second).Should(Receive(BeNil()))
		Expect(events).To(BeEmpty())
	})

	// 插件快照是抽出的值拷贝，改 IP 碰不到调用方的投影对象
	It("does not let plugin mutations reach the caller snapshot", func() {
		mut := &fakePlugin{
			name:   "mut",
			rounds: []map[string]any{{"a": []string{"x"}}},
			mutate: func(snapshot []InstanceSnapshot) {
				snapshot[0].IP = "hacked"
			},
		}
		snapshot := snapshotOf("a")
		snapshot[0].IP = "10.0.0.1"

		var events []serializer.AppInstancePluginWatchEvent
		Expect(NewRunner(mut).Run(ctx, snapshot, collect(&events))).To(Succeed())

		Expect(snapshot[0].IP).To(Equal("10.0.0.1"))
	})
})
