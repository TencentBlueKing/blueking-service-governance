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
	"cmp"
	"slices"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// pushedInstances 记录本连接内每个实例最后一次推给前端的投影，供北极星补拉比对与重推
// 只收成功推送过的实例，DELETED 后移除；请求级内存，不跨连接、不跨副本共享
//
// 驻留量的上界是该应用环境当前存活的实例数，与一次 List 全量同量级，连接一关整体释放
// 不是持久化存储，与 appmodel / image 下那几个写 MongoDB 的 SnapshotStore 无关
//
// 有意不加锁：全部读写都发生在 consume 的 for-select 循环里，Pod 事件与周期补拉是
// 同一 goroutine 的两个 case，天然互斥。若以后把补拉挪进独立 goroutine，这里必须先补锁
type pushedInstances struct {
	items map[string]*serializer.AppInstanceOutputObj
}

// newPushedInstances 创建一条 Watch 连接的已推送投影记录
func newPushedInstances() *pushedInstances {
	return &pushedInstances{items: make(map[string]*serializer.AppInstanceOutputObj)}
}

// track 按已推送出去的事件更新记录；DELETED 移除该实例，其余覆盖为最新一次投影
// 实例随 DELETED 出栈，因此长连接里的占用不会随事件数累积，只随存活实例数波动
func (p *pushedInstances) track(event serializer.AppInstanceWatchEvent) {
	if event.Object == nil {
		return
	}

	if event.Type == string(EventDeleted) {
		delete(p.items, event.Object.ID)
		return
	}

	p.items[event.Object.ID] = event.Object
}

// empty 是否尚无任何已推送实例
func (p *pushedInstances) empty() bool {
	return len(p.items) == 0
}

// sorted 按实例 ID 定序返回全部记录，使一轮补拉的推送顺序稳定
func (p *pushedInstances) sorted() []*serializer.AppInstanceOutputObj {
	items := make([]*serializer.AppInstanceOutputObj, 0, len(p.items))
	for _, item := range p.items {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b *serializer.AppInstanceOutputObj) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return items
}
