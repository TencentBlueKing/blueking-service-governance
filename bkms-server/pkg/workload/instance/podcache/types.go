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

// Package podcache 定义 Pod 共享缓存的数据结构与同步侧接口契约。
//
// 本包只承载跨子需求的契约，不含任何缓存读写、筛选排序、单写者协调的实现：
// 同步写入由 C3 承接，协调与预热由 C2 承接，查询侧读取与两级路由由 C5 承接。
// 依赖方向是单向的——worker 侧与查询侧都 import 本包，本包不 import 任何 handler / store。
//
// 设计说明见 design_notes/pod_instance_cache.md；
// 下游发现投影字段不足时必须回到该文档走契约变更流程，禁止在实现子需求中私自扩字段。
package podcache

import "time"

// MongoDB 集合与索引设计约束。
//
// 本包只给出约束，索引的实际创建由 C3 通过 golang-migrate 迁移落地（见 db/AGENTS.md），
// 届时须按仓库惯例把索引注释写到对应的 store 源文件里：
//
//   - pod_cache_projections
//     唯一：clusterID + namespace + name（同步为幂等 upsert，保证同一 Pod 不产生重复文档）
//     普通：clusterID + namespace（查询侧按隔离键取整个命名空间的投影集合）
//   - pod_cache_sync_watermarks
//     唯一：clusterID + namespace（每个隔离键至多一条水位）
//
// 两个集合都不建 TTL 索引：缓存的生命周期由 60 分钟空闲释放与对账清理控制，
// 而非由文档过期控制，否则会与陈旧度判定的语义冲突。
const (
	// PodProjectionCollection Pod 缓存投影集合名
	PodProjectionCollection = "pod_cache_projections"
	// SyncWatermarkCollection 同步水位集合名
	SyncWatermarkCollection = "pod_cache_sync_watermarks"
)

// PodProjection 共享缓存中的单个 Pod 投影对象。
//
// 字段集合恰好支撑 serializer.AppInstanceOutputObj 的全部字段，以及关键字、状态、排序
// 三类查询所需，不多不少：
//   - Name / PodIP / NodeIP 参与关键字子串匹配
//   - Status 参与状态筛选，取值为归一后的实例状态封闭枚举
//   - Name / StartTime / RestartCount 参与排序（分别对应 name / age / restartCount）
//   - Labels 供查询侧按部署记录的 LabelSelector 匹配
//
// 禁止写入投影范围之外的任何内容，尤其是环境变量、Secret 引用与完整 Pod manifest：
// 前者是敏感数据，后者会突破同步进程 500MB 常驻内存与缓存 1GB 存储的容量约束。
//
// 缓存数据可重建，丢失不影响业务正确性。
type PodProjection struct {
	// ClusterID 所属集群，与 Namespace 共同构成缓存隔离键
	ClusterID string `json:"clusterID" bson:"clusterID"`
	// Namespace 所属命名空间
	Namespace string `json:"namespace" bson:"namespace"`
	// Name Pod 名称，在（集群，命名空间）内唯一，对应实例 ID
	Name string `json:"name" bson:"name"`

	// Labels Pod 标签，供 LabelSelector 匹配
	Labels map[string]string `json:"labels" bson:"labels"`
	// PodIP Pod IP，参与关键字匹配
	PodIP string `json:"podIP" bson:"podIP"`
	// NodeIP Pod 所在节点 IP，参与关键字匹配
	NodeIP string `json:"nodeIP" bson:"nodeIP"`
	// Image 首个业务容器的镜像
	Image string `json:"image" bson:"image"`
	// RestartCount 容器重启次数，取各容器最大值，参与排序
	RestartCount int64 `json:"restartCount" bson:"restartCount"`
	// Status 实例状态，取值为 podstatus.Normalize 归一后的封闭枚举，参与状态筛选
	Status string `json:"status" bson:"status"`
	// Message 状态详情，来自 status.message，为空时回退 status.reason
	Message string `json:"message" bson:"message"`
	// Ready 就绪状况，对应 Ready condition，输出为 AppInstanceOutputObj.IsHealthy
	Ready bool `json:"ready" bson:"ready"`
	// StartTime Pod 创建时间，查询侧据此计算存在时长，参与排序
	StartTime time.Time `json:"startTime" bson:"startTime"`
}

// SyncWatermark 单个（集群，命名空间）的同步水位，供查询侧做陈旧度判定。
//
// 只有同步成功才推进 LastSyncAt；集群不可达或存储失败时保持原值，
// 使查询侧能在陈旧度阈值内感知到同步已中断并降级直连。水位缺失一律视为不可信。
// 陈旧度阈值取值由 C5 按父需求 Q-006 确定，本契约不预设具体数值。
type SyncWatermark struct {
	// ClusterID 所属集群，与 Namespace 共同构成归属键
	ClusterID string `json:"clusterID" bson:"clusterID"`
	// Namespace 所属命名空间
	Namespace string `json:"namespace" bson:"namespace"`
	// LastSyncAt 最后一次成功同步的时间
	LastSyncAt time.Time `json:"lastSyncAt" bson:"lastSyncAt"`
}
