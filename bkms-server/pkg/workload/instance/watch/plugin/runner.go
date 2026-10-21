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
	"log/slog"
	"reflect"
	"time"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// pluginFetchTimeout 单次插件拉取上限
// 插件与 Pod 事件 / 心跳共用 consume goroutine，超时后按本轮失败跳过，避免一次卡住拖死整条流
const pluginFetchTimeout = 5 * time.Second

const (
	// fetchStagePrime 建流时为预置实例建基线的那次拉取
	fetchStagePrime = "prime"
	// fetchStageRound 每个 tick 的周期拉取
	fetchStageRound = "round"
)

// Emit 把一条插件事件写给客户端；返回 error 表示流已不可写，Runner 立即中止本轮
type Emit func(event serializer.AppInstancePluginWatchEvent) error

// Runner 周期性驱动已注册插件，并把有变化的载荷推成独立事件
//
// 连接级对象：每条 Watch 请求新建一个。与 pushedInstances 同理，状态只在基础层
// consume 的单 goroutine 内读写，故不加锁；若以后把插件调用挪进独立 goroutine，这里必须先补锁
type Runner struct {
	plugins []Plugin
	// pushed 每个插件上次成功推给前端的载荷（含 Prime 记下的建流基线），索引为 pushed[插件名][实例 ID]
	// 它是判断「有没有变化」的唯一基准：本轮拉取失败不动它，页面因此保留上次已知值
	pushed map[string]map[string]any
	// fetchTimeout 单次 Fetch 上限；单测调小以免真实等待
	fetchTimeout time.Duration
}

// NewRunner 按注册顺序创建插件执行时；无插件时 Run 直接空转
func NewRunner(plugins ...Plugin) *Runner {
	return &Runner{
		plugins:      plugins,
		pushed:       make(map[string]map[string]any, len(plugins)),
		fetchTimeout: pluginFetchTimeout,
	}
}

// Prime 建流时拉一轮，把结果记为各实例的对比基线，不推事件
// 预置实例前端首包已带附属数据，按「首次见到且为空不推」判断会吞掉变空、重推未变的
// 代价：List 到建流之间恰好发生的变化，要等下次变化或重连才补上
func (r *Runner) Prime(ctx context.Context, snapshot []*serializer.AppInstanceOutputObj) {
	// 没有预置实例时无从建基线，省掉一轮拉取
	if len(snapshot) == 0 {
		return
	}

	views := toInstanceSnapshots(snapshot)

	for _, p := range r.plugins {
		// 拉取失败的插件不建基线，它的实例保持未记录，之后按首次见到处理
		payloads, ok := r.fetchOrSkip(ctx, p, fetchStagePrime, views)
		if !ok {
			continue
		}

		// 只记插件给了值的实例；没给的保持未记录，之后按首次见到处理
		baseline := make(map[string]any, len(views))
		for _, instance := range views {
			if payload, found := payloads[instance.ID]; found {
				baseline[instance.ID] = payload
			}
		}

		r.pushed[p.Name()] = baseline
	}
}

// Forget 清掉某实例在全部插件里的已推送记录，下一轮按首次见到重新判断
// 前端收到新行的 ADDED 时附属数据为空，此前记下的基线已不代表前端所见，不清掉的话
// 载荷没变就永远不推，这一行的附属数据会一直空着
func (r *Runner) Forget(instanceID string) {
	for _, pushed := range r.pushed {
		delete(pushed, instanceID)
	}
}

// Run 跑一轮：把存活实例快照交给每个插件，仅对相对上次推送有变化的实例 emit 事件
// 单个插件失败只跳过本轮该插件，既不拆流也不影响其它插件与 Pod 事件
func (r *Runner) Run(ctx context.Context, snapshot []*serializer.AppInstanceOutputObj, emit Emit) error {
	// 快照为空说明当前没有已知的存活实例，无从比对，省掉一轮无谓拉取
	if len(snapshot) == 0 {
		return nil
	}

	views := toInstanceSnapshots(snapshot)

	for _, p := range r.plugins {
		// 拉取失败（含超时）按「本轮不可用」处理：不推事件、不拆流、不动已推送记录
		payloads, ok := r.fetchOrSkip(ctx, p, fetchStageRound, views)
		if !ok {
			continue
		}

		if err := r.emitChanged(p.Name(), views, payloads, emit); err != nil {
			return err
		}
	}

	return nil
}

// toInstanceSnapshots 抽出插件快照后再交给各插件 Fetch；各插件共用这一份，改字段碰不到调用方的投影
func toInstanceSnapshots(snapshot []*serializer.AppInstanceOutputObj) []InstanceSnapshot {
	views := make([]InstanceSnapshot, 0, len(snapshot))
	for _, inst := range snapshot {
		views = append(views, InstanceSnapshot{ID: inst.ID, IP: inst.IP})
	}

	return views
}

// fetchOrSkip 拉一轮并吞掉失败，Prime 与 Run 共用：失败只告警、记 metrics，返回 false 让调用方跳过该插件
// 失败被这里吞掉，不会体现在响应里，只能靠日志与 metrics 排查；stage 写进日志区分建基线还是周期轮次
// 客户端已断开导致的失败是正常收流，不是插件故障，不告警
func (r *Runner) fetchOrSkip(
	ctx context.Context,
	p Plugin,
	stage string,
	views []InstanceSnapshot,
) (map[string]any, bool) {
	payloads, err := r.fetch(ctx, p, views)
	if err != nil {
		if ctx.Err() == nil {
			log.WarnAttrs(ctx, "watch plugin fetch failed, skip this plugin",
				slog.String("plugin", p.Name()),
				slog.String("stage", stage),
				slog.String("err", err.Error()),
			)
		}

		metrics.InstanceWatchPluginFetch(p.Name(), false)

		return nil, false
	}

	metrics.InstanceWatchPluginFetch(p.Name(), true)

	return payloads, true
}

// fetch 带超时地拉一轮
func (r *Runner) fetch(
	ctx context.Context,
	p Plugin,
	snapshot []InstanceSnapshot,
) (map[string]any, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, r.fetchTimeout)
	defer cancel()

	payloads, err := p.Fetch(fetchCtx, snapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch plugin %s", p.Name())
	}

	return payloads, nil
}

// emitChanged 按快照顺序比对单个插件的本轮结果，仅差异才推
func (r *Runner) emitChanged(
	name string,
	snapshot []InstanceSnapshot,
	payloads map[string]any,
	emit Emit,
) error {
	prev := r.pushed[name]
	// 按本轮快照重建，DELETED 的实例不在快照里，随之出栈
	next := make(map[string]any, len(snapshot))

	for _, instance := range snapshot {
		payload, known := payloadThisRound(payloads, prev, instance.ID)
		if !known {
			continue
		}

		next[instance.ID] = payload

		if err := emitIfChanged(name, instance.ID, prev, payload, emit); err != nil {
			return err
		}
	}

	// 全部写成功才提交；中途失败丢弃 next，缓存停在上一轮
	r.pushed[name] = next

	return nil
}

// payloadThisRound 本轮该实例的载荷：插件给了用新值，没给则沿用上次，避免「没给」被下一轮当成变化
func payloadThisRound(payloads, prev map[string]any, id string) (any, bool) {
	if payload, ok := payloads[id]; ok {
		return payload, true
	}

	last, tracked := prev[id]
	return last, tracked
}

// emitIfChanged 相对上次有差异才写事件；写失败说明流已不可写，本轮剩余实例不再尝试
func emitIfChanged(name, instanceID string, prev map[string]any, payload any, emit Emit) error {
	last, tracked := prev[instanceID]
	if !needEmit(tracked, last, payload) {
		return nil
	}

	if err := emit(serializer.AppInstancePluginWatchEvent{
		Type:   EventTypePlugin,
		Plugin: name,
		Object: &serializer.AppInstancePluginObj{ID: instanceID, Data: payload},
	}); err != nil {
		return errors.Wrapf(err, "emit %s plugin event for instance %s", name, instanceID)
	}

	return nil
}

// needEmit 相对上次是否该写事件：已见过走深比较（载荷可能含 map，不能 ==）；首次且为空则不推
func needEmit(tracked bool, last, payload any) bool {
	if tracked {
		return !reflect.DeepEqual(last, payload)
	}

	return !isEmptyPayload(payload)
}

// isEmptyPayload 是否「没有数据」：只看 nil 和容器长度，不解释插件语义
func isEmptyPayload(payload any) bool {
	if payload == nil {
		return true
	}

	switch v := reflect.ValueOf(payload); v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return v.Len() == 0
	default:
		return false
	}
}
