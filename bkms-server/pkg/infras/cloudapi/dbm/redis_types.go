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

import (
	"strconv"
	"strings"
)

// ClusterType DBM Redis 集群类型（cluster_type 枚举）
type ClusterType string

const (
	// Redis 集群部署
	// ClusterTypeTwemproxyRedis 对应页面 "Tendis Cache" 集群"
	ClusterTypeTwemproxyRedis ClusterType = "TwemproxyRedisInstance"
	// ClusterTypeTwemproxyTendisSSD 对应页面 "TendisSSD 集群"
	ClusterTypeTwemproxyTendisSSD ClusterType = "TwemproxyTendisSSDInstance"
	// ClusterTypePredixyRedisCluster 对应页面 "原生 Redis Cluster"
	ClusterTypePredixyRedisCluster ClusterType = "PredixyRedisCluster"
	// ClusterTypePredixyTendisplus 对应页面 "Tendisplus 集群"
	ClusterTypePredixyTendisplus ClusterType = "PredixyTendisplusCluster"

	// Redis 主从部署
	// ClusterTypeRedisInstance 对应页面 "主从部署"
	ClusterTypeRedisInstance ClusterType = "RedisInstance"
)

// IsProxyClusterType 判断是否为含 Proxy 的集群类型（区别于主从实例 RedisInstance）
func IsProxyClusterType(clusterType ClusterType) bool {
	switch clusterType {
	case ClusterTypeTwemproxyRedis, ClusterTypeTwemproxyTendisSSD,
		ClusterTypePredixyRedisCluster, ClusterTypePredixyTendisplus:
		return true
	default:
		return false
	}
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	ID          int         `json:"id"`
	Domain      string      `json:"domain"`
	Port        int         `json:"port"`
	Status      string      `json:"status"`
	ClusterType ClusterType `json:"cluster_type"`
	BkBizID     int         `json:"bk_biz_id"`
}

// DisableTicketType 根据集群类型返回对应的禁用工单类型
func DisableTicketType(clusterType ClusterType) TicketType {
	if clusterType == ClusterTypeRedisInstance {
		return TicketTypeRedisClose
	}
	return TicketTypeRedisProxyClose
}

// DeleteTicketType 根据集群类型返回对应的删除工单类型
func DeleteTicketType(clusterType ClusterType) TicketType {
	if clusterType == ClusterTypeRedisInstance {
		return TicketTypeRedisInstanceDestroy
	}
	return TicketTypeRedisDestroy
}

// ------------------------------------------ Redis 工单参数 ------------------------------------------

// CreateRedisParams DBM 创建 Redis 工单参数（聚合 cluster 与 master_slave 两种部署模式所需字段）
type CreateRedisParams struct {
	// 业务 ID
	BkBizID int `json:"bk_biz_id"`
	// 单据类型（REDIS_CLUSTER_APPLY / REDIS_INS_APPLY）
	TicketType TicketType `json:"ticket_type"`
	// 云区域 ID
	BkCloudID int `json:"bk_cloud_id"`
	// 业务英文缩写
	DBAppAbbr string `json:"db_app_abbr"`
	// 集群类型，见 ClusterType* 常量
	ClusterType ClusterType `json:"cluster_type"`

	// --- 通用字段 ---
	// 版本号（如 Redis-6）
	DBVersion string `json:"db_version"`
	// 城市代码，可选，默认 ""
	CityCode string `json:"city_code,omitempty"`
	// 容灾级别，可选，默认 NONE，见 DisasterTolerance* 常量
	DisasterToleranceLevel string `json:"disaster_tolerance_level,omitempty"`
	// 主机来源：resource_pool
	IPSource string `json:"ip_source"`
	// 资源池申请规格（ip_source = resource_pool 时必填）
	ResourceSpec *ResourceSpec `json:"resource_spec"`

	// --- 集群部署（REDIS_CLUSTER_APPLY）专用字段 ---
	// 集群名称（英文、数字及下划线）
	ClusterName string `json:"cluster_name"`
	// 集群别名（一般为中文别名）
	ClusterAlias string `json:"cluster_alias"`
	// 集群接入层端口
	ProxyPort int `json:"proxy_port"`
	// Proxy 访问密码，可选，不传则系统随机生成
	ProxyPwd string `json:"proxy_pwd,omitempty"`
	// 集群分片数（PredixyRedisCluster / PredixyTendisplusCluster 要求 ≥ 3）
	ClusterShardNum int `json:"cluster_shard_num"`

	// --- 主从部署（REDIS_INS_APPLY）专用字段 ---
	// 集群起始端口
	Port int `json:"port"`
	// Redis 访问密码，不传则系统随机生成
	RedisPwd string `json:"redis_pwd"`
	// 是否为追加部署（false 新建，true 追加到已有机器）
	AppendApply bool `json:"append_apply"`
	// 主从集群信息列表，每个元素代表一个主从集群
	Infos []RedisInsInfo `json:"infos,omitempty"`

	// --- 其他 ---
	// 单据备注, 可选
	Remark string `json:"remark,omitempty"`
	// 是否忽略重复单据校验，默认 false
	IgnoreDuplication bool `json:"ignore_duplication,omitempty"`
	// 幂等键（可选，用于平台侧去重）
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// DisableRedisParams DBM 禁用 Redis 工单参数
type DisableRedisParams struct {
	// 业务 ID
	BkBizID int `json:"bk_biz_id"`
	// 单据类型（REDIS_CLOSE / REDIS_PROXY_CLOSE，可通过 DisableTicketType(clusterType) 获取）
	TicketType TicketType `json:"ticket_type"`
	// 待禁用的集群 ID
	ClusterID int `json:"cluster_id"`
	// 是否强制禁用，默认 false
	Force bool `json:"force"`
}

// DeleteRedisParams DBM 删除 Redis 工单参数
//   - REDIS_DESTROY（集群删除）：单集群操作，使用 ClusterID
//   - REDIS_INSTANCE_DESTROY（主从删除）：多集群操作，使用 ClusterIDs
type DeleteRedisParams struct {
	// 业务 ID
	BkBizID int `json:"bk_biz_id"`
	// 单据类型（REDIS_DESTROY / REDIS_INSTANCE_DESTROY，可通过 DeleteTicketType(clusterType) 获取）
	TicketType TicketType `json:"ticket_type"`
	// 集群 ID（REDIS_DESTROY 使用）
	ClusterID int `json:"cluster_id,omitempty"`
	// 集群 ID 列表（REDIS_INSTANCE_DESTROY 使用）
	ClusterIDs []int `json:"cluster_ids,omitempty"`
	// 是否强制删除，默认 false
	Force bool `json:"force"`
}

// ------------------------------------------ Redis 工单内部类型 ------------------------------------------

// createTicketPayload 创建 Redis 工单顶层请求体
type createTicketPayload struct {
	BkBizID           int        `json:"bk_biz_id"`
	TicketType        TicketType `json:"ticket_type"`
	Details           any        `json:"details"`
	Remark            string     `json:"remark,omitempty"`
	IgnoreDuplication bool       `json:"ignore_duplication"`
}

// redisClusterApplyDetails REDIS_CLUSTER_APPLY（Redis 集群部署）单据 details
type redisClusterApplyDetails struct {
	BkCloudID              int           `json:"bk_cloud_id"`
	DBAppAbbr              string        `json:"db_app_abbr"`
	ClusterName            string        `json:"cluster_name"`
	ClusterAlias           string        `json:"cluster_alias,omitempty"`
	ClusterType            ClusterType   `json:"cluster_type"`
	DBVersion              string        `json:"db_version"`
	ProxyPort              int           `json:"proxy_port"`
	ProxyPwd               string        `json:"proxy_pwd,omitempty"`
	CityCode               string        `json:"city_code"`
	DisasterToleranceLevel string        `json:"disaster_tolerance_level"`
	ClusterShardNum        int           `json:"cluster_shard_num"`
	IPSource               string        `json:"ip_source"`
	ResourceSpec           *ResourceSpec `json:"resource_spec"`
}

// redisInsApplyDetails REDIS_INS_APPLY（Redis 主从节点部署）单据 details
type redisInsApplyDetails struct {
	BkCloudID              int            `json:"bk_cloud_id"`
	DBAppAbbr              string         `json:"db_app_abbr"`
	ClusterType            ClusterType    `json:"cluster_type"`
	DBVersion              string         `json:"db_version,omitempty"`
	Port                   int            `json:"port,omitempty"`
	RedisPwd               string         `json:"redis_pwd,omitempty"`
	CityCode               string         `json:"city_code"`
	DisasterToleranceLevel string         `json:"disaster_tolerance_level"`
	AppendApply            bool           `json:"append_apply"`
	IPSource               string         `json:"ip_source"`
	Infos                  []RedisInsInfo `json:"infos"`
	ResourceSpec           *ResourceSpec  `json:"resource_spec"`
}

// disableTicketPayload 禁用 Redis 工单顶层请求体
type disableTicketPayload struct {
	BkBizID    int                 `json:"bk_biz_id"`
	TicketType TicketType          `json:"ticket_type"`
	Details    disableRedisDetails `json:"details"`
}

// disableRedisDetails REDIS_CLOSE / REDIS_PROXY_CLOSE 单据 details
type disableRedisDetails struct {
	ClusterID int  `json:"cluster_id"`
	Force     bool `json:"force"`
}

// deleteTicketPayload 删除 Redis 工单顶层请求体
type deleteTicketPayload struct {
	BkBizID    int        `json:"bk_biz_id"`
	TicketType TicketType `json:"ticket_type"`
	Details    any        `json:"details"`
}

// deleteRedisClusterDetails REDIS_DESTROY 单据 details（单集群操作）
type deleteRedisClusterDetails struct {
	ClusterID int  `json:"cluster_id"`
	Force     bool `json:"force"`
}

// deleteRedisInstanceDetails REDIS_INSTANCE_DESTROY 单据 details（多集群操作）
type deleteRedisInstanceDetails struct {
	ClusterIDs []int `json:"cluster_ids"`
	Force      bool  `json:"force"`
}

// filterClustersOpts DBM 过滤集群接口（/dbbase/filter_clusters/）的查询参数。
// 仅设置非空字段会被下发，空字段交由 DBM 使用默认值。
type filterClustersOpts struct {
	// BkBizID 业务 ID，0 表示不限制
	BkBizID int
	// ClusterIDs 集群 ID 列表，多个以逗号分隔（如 "1,2,3"）
	ClusterIDs []int
	// ClusterType 集群类型，多个以逗号分隔（如 "tendbha,tendbsingle"）
	ClusterType string
	// DBType DB 类型（如 mysql、redis、mongodb）
	DBType string
	// Name 集群名称/别名模糊查询
	Name string
	// ExactDomain 域名精确查询
	ExactDomain string
	// Limit 分页限制，0 表示不下发（DBM 默认 -1 不分页返回全部）
	Limit int
	// Offset 分页起始
	Offset int
}

// toQueryParams 将查询结构体转换为下发给 DBM 的 query 参数，空字段不下发。
func (q *filterClustersOpts) toQueryParams() map[string]string {
	params := make(map[string]string)
	if q == nil {
		return params
	}
	if q.BkBizID != 0 {
		params["bk_biz_id"] = strconv.Itoa(q.BkBizID)
	}
	if len(q.ClusterIDs) > 0 {
		clusterIDs := make([]string, len(q.ClusterIDs))
		for i, id := range q.ClusterIDs {
			clusterIDs[i] = strconv.Itoa(id)
		}
		params["cluster_ids"] = strings.Join(clusterIDs, ",")
	}
	if q.ClusterType != "" {
		params["cluster_type"] = q.ClusterType
	}
	if q.DBType != "" {
		params["db_type"] = q.DBType
	}
	if q.Name != "" {
		params["name"] = q.Name
	}
	if q.ExactDomain != "" {
		params["exact_domain"] = q.ExactDomain
	}
	if q.Limit != 0 {
		params["limit"] = strconv.Itoa(q.Limit)
	}
	if q.Offset != 0 {
		params["offset"] = strconv.Itoa(q.Offset)
	}
	return params
}

// filterClusterItem DBM 过滤集群接口返回的集群项
type filterClusterItem struct {
	ID                int         `json:"id"`
	Status            string      `json:"status"`
	ClusterName       string      `json:"cluster_name"`
	ClusterAccessPort int         `json:"cluster_access_port"`
	MasterDomain      string      `json:"master_domain"`
	BkBizID           int         `json:"bk_biz_id"`
	ClusterType       ClusterType `json:"cluster_type"`
}

// toClusterInfo 将 DBM 集群项转换为对外暴露的 ClusterInfo
func toClusterInfo(item filterClusterItem) *ClusterInfo {
	return &ClusterInfo{
		ID:          item.ID,
		Domain:      item.MasterDomain,
		Port:        item.ClusterAccessPort,
		Status:      item.Status,
		ClusterType: item.ClusterType,
		BkBizID:     item.BkBizID,
	}
}

// ------------------------------------------ Redis 资源规格 ------------------------------------------

// LocationSpec 地域约束
type LocationSpec struct {
	// 城市代码（如 sz）
	City string `json:"city,omitempty"`
	// 园区 ID 列表
	SubZoneIDs []int `json:"sub_zone_ids,omitempty"`
}

// ResourceSpecItem 单角色资源规格
type ResourceSpecItem struct {
	// 规格 ID
	SpecID int `json:"spec_id"`
	// 申请机器数量（backend_group.count 表示机器组数，每组含 1 master + 1 slave）
	Count int `json:"count"`
	// 地域约束，可选
	LocationSpec *LocationSpec `json:"location_spec,omitempty"`
}

// ResourceSpec 资源池申请规格（ip_source = resource_pool 时必填）
type ResourceSpec struct {
	// Proxy 节点规格（仅集群部署需要）
	Proxy *ResourceSpecItem `json:"proxy,omitempty"`
	// 后端主从节点组规格
	BackendGroup *ResourceSpecItem `json:"backend_group"`
}

// RedisInsInfo master_slave 模式的实例信息（REDIS_INS_APPLY 的 infos 元素）
type RedisInsInfo struct {
	// 集群名称（英文、数字及下划线）
	ClusterName string `json:"cluster_name"`
	// DB 数量
	Databases int `json:"databases"`
}

// 容灾级别（disaster_tolerance_level）枚举值
const (
	// DisasterToleranceNone 无（默认）
	DisasterToleranceNone = "NONE"
	// DisasterToleranceSameSubzoneCross 指定园区
	DisasterToleranceSameSubzoneCross = "SAME_SUBZONE_CROSS_SWTICH"
	// DisasterToleranceSameSubzone 指定园区（无机架要求）
	DisasterToleranceSameSubzone = "SAME_SUBZONE"
	// DisasterToleranceCrossSubzone 跨园区
	DisasterToleranceCrossSubzone = "CROS_SUBZONE"
	// DisasterToleranceCrossRack 不限园区
	DisasterToleranceCrossRack = "CROSS_RACK"
	// DisasterToleranceMaxEachZoneEqual 每个 subzone 尽量均匀分布
	DisasterToleranceMaxEachZoneEqual = "MAX_EACH_ZONE_EQUAL"
)

// 主机来源（ip_source）枚举值
const (
	// IPSourceResourcePool 资源池
	IPSourceResourcePool = "resource_pool"
)
