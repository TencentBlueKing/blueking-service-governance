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

// Package plugin 定义实例 Watch 附属数据插件的接口与执行时
//
// Watch 分两层：基础层只做 Pod 的 list/watch 与投影推送，与任何附属数据源无关；
// 附属数据（北极星、后续的 CPU / 告警等）一律作为插件接入，由本包周期性驱动，
// 结果以独立事件推出，不再嵌进 Pod 投影
package plugin

import "context"

// EventTypePlugin 附属数据事件的 type 取值；恒为 PLUGIN，新增插件只增 plugin 取值、不扩事件类型
// 基础层产出的 Pod 事件类型见 instance/watch.EventType
const EventTypePlugin = "PLUGIN"

// InstanceSnapshot 插件看到的实例快照：只含匹配附属数据需要的标识与 IP
// 不把 API 投影对象传给插件，避免插件依赖 serializer，也避免改到 pushed 里的对象
type InstanceSnapshot struct {
	ID string
	IP string
}

// Plugin 为流内实例补充附属数据
//
// 契约（由 Runner 保证，插件不必自己实现）：
//   - 每个周期收到「本连接当前存活的全量实例快照」，而不是本周期有变动的那几个。
//     Pod 不变、仅附属数据变化时也要能推出更新，因此必须给全量
//   - 快照从投影对象抽出，改顶层字段碰不到 pushed 缓存；各插件共用同一份
//   - 插件只负责返回「当前值」，不维护历史、不判断变化；变化比对与推送由 Runner 统一做
//   - 返回值按实例 ID 索引，未命中的实例可以不出现在 map 里
//   - 每次 Fetch 带超时；超时与返回 error 同一口径（跳过本轮）。插件必须尊重 ctx 取消
//
// 插件侧唯一的额外要求：集合类载荷必须定序，否则 Runner 的深比较会把顺序抖动误判成变化
type Plugin interface {
	// Name 插件名，写入事件的 plugin 字段，供前端区分附属数据来源
	Name() string
	// Fetch 按快照返回各实例当前的附属数据；返回 error 表示本轮不可用，Runner 跳过本轮
	Fetch(ctx context.Context, snapshot []InstanceSnapshot) (map[string]any, error)
}
