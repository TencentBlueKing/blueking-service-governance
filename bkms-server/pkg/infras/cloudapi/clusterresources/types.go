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

package clusterresources

import "time"

// ListEventParams 查询事件参数
type ListEventParams struct {
	Namespace     string   // 命名空间
	ComponentName string   // 组件名称
	ResourceKinds []string // 资源类型，支持多个值，如：[Deployment, Pod]
	ResourceNames []string // 资源名称，支持多个，如：[nginx-ingress, nginx-ingress-2695bd-58877d456b]
	Level         string   // 事件级别
	StartedAt     int64    // 开始时间戳
	EndedAt       int64    // 结束时间戳
	Page          int      // 页码
	PageSize      int      // 每页大小
}

// EventEntry 事件条目
type EventEntry struct {
	ClusterID     string    // BCS 集群 ID
	Namespace     string    // 命名空间
	Level         string    // 事件级别
	Content       string    // 事件内容
	Type          string    // 事件类型
	ComponentName string    // 组件名称
	ResourceKind  string    // 关联的资源类型，如：Deployment, Pod，Node 等
	ResourcesName string    // 关联的资源名称，如：nginx-ingress-2695bd-58877d456b
	CreatedAt     time.Time // 事件创建时间
}

// PaginatedEvents 分页事件查询结果
type PaginatedEvents struct {
	Count int64
	Data  []EventEntry
}
