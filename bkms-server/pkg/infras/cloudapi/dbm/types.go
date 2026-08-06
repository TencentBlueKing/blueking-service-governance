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

package dbm

// TicketType DBM 工单类型
type TicketType string

const (
	// Redis 创建工单类型
	// TicketTypeRedisClusterApply 集群创建
	TicketTypeRedisClusterApply TicketType = "REDIS_CLUSTER_APPLY"
	// TicketTypeRedisInsApply 主从创建
	TicketTypeRedisInsApply TicketType = "REDIS_INS_APPLY"

	// Redis 禁用工单类型
	// TicketTypeRedisProxyClose 集群禁用
	TicketTypeRedisProxyClose TicketType = "REDIS_PROXY_CLOSE"
	// TicketTypeRedisClose 主从禁用
	TicketTypeRedisClose TicketType = "REDIS_CLOSE"

	// Redis 删除工单类型
	// TicketTypeRedisDestroy 集群删除
	TicketTypeRedisDestroy TicketType = "REDIS_DESTROY"
	// TicketTypeRedisInstanceDestroy 主从删除
	TicketTypeRedisInstanceDestroy TicketType = "REDIS_INSTANCE_DESTROY"
)

// TicketStatus DBM 工单状态
type TicketStatus string

const (
	// TicketStatusPending 待处理
	TicketStatusPending TicketStatus = "PENDING"
	// TicketStatusRunning 处理中
	TicketStatusRunning TicketStatus = "RUNNING"
	// TicketStatusSucceeded 成功
	TicketStatusSucceeded TicketStatus = "SUCCEEDED"
	// TicketStatusFailed 失败
	TicketStatusFailed TicketStatus = "FAILED"
	// TicketStatusTerminated 终止
	TicketStatusTerminated TicketStatus = "TERMINATED"
)

// IsTerminal 判断工单是否已终态（SUCCEEDED / FAILED / TERMINATED）
func (s TicketStatus) IsTerminal() bool {
	switch s {
	case TicketStatusSucceeded, TicketStatusFailed, TicketStatusTerminated:
		return true
	default:
		return false
	}
}

// TicketInfo 工单状态信息
type TicketInfo struct {
	ID     int          `json:"id"`
	Status TicketStatus `json:"status"`
}

// ClusterOnlineStatuses 集群在线状态集合，在线状态的集群删除前需先禁用
var ClusterOnlineStatuses = map[string]bool{
	"online":    true,
	"running":   true,
	"available": true,
	"normal":    true,
	"ready":     true,
}

// IsOnlinePhase 判断集群是否处于在线状态（在线状态的集群删除前需先禁用）
func IsOnlinePhase(status string) bool {
	return ClusterOnlineStatuses[status]
}
