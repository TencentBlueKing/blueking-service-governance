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

package podcache

import "context"

// SyncOutcome 一次同步的结果分类，决定协调器是否推进水位与是否重试。
type SyncOutcome string

const (
	// SyncOutcomeSucceeded 同步成功，缓存已与集群对齐，水位推进到本次同步时间
	SyncOutcomeSucceeded SyncOutcome = "Succeeded"
	// SyncOutcomeClusterUnreachable 集群不可达，未获得完整数据，不得写入残缺数据、不得推进水位
	SyncOutcomeClusterUnreachable SyncOutcome = "ClusterUnreachable"
	// SyncOutcomeStorageFailed 集群数据已取到但写入共享缓存失败，缓存可能部分更新，不得推进水位
	SyncOutcomeStorageFailed SyncOutcome = "StorageFailed"
)

// NamespaceKey 缓存的隔离键，即一组（集群，命名空间）。
type NamespaceKey struct {
	// ClusterID 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
}

// SyncResult 一次同步的结果。
//
// Watermark 仅在 Outcome 为 SyncOutcomeSucceeded 时有意义，其余情况下协调器须保留原水位。
type SyncResult struct {
	// Outcome 结果分类
	Outcome SyncOutcome
	// Watermark 本次同步后的新水位，仅成功时有效
	Watermark *SyncWatermark
}

// SyncExecutor 对指定（集群，命名空间）执行一次同步，是协调器与执行器之间的唯一接口。
//
// 该接口把「何时同步、由谁同步」与「怎么同步」解耦，使 C2 协调器与 C3 执行器可分别独立
// 编译与单测。接口须同时适配常驻监听与周期增量两种实现路径（父需求 Q-004 / Q-005 未收敛）。
//
// 返回 error 表示执行器自身异常；可预期的失败通过 SyncResult.Outcome 表达，
// 其中集群不可达时协调器不推进水位。
type SyncExecutor interface {
	// Sync 对指定（集群，命名空间）执行一次同步
	Sync(ctx context.Context, key NamespaceKey) (*SyncResult, error)
}

// ActiveSetManager 维护需要持续同步的活跃（集群，命名空间）集合。
//
// 调用方：查询侧在两级路由判定应建立缓存时加入，部署完成事件在预热时加入。
type ActiveSetManager interface {
	// Join 把（集群，命名空间）加入活跃同步集合；重复加入幂等
	Join(ctx context.Context, key NamespaceKey) error

	// Leave 把（集群，命名空间）移出活跃同步集合；对不在集合中的 key 为空操作
	Leave(ctx context.Context, key NamespaceKey) error
}
