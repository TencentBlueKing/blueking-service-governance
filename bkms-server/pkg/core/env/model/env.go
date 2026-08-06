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

package model

import (
	"cmp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// EnvStatus 环境状态
type EnvStatus string

const (
	// EnvStatusReady 环境就绪
	EnvStatusReady EnvStatus = "Ready"
	// EnvStatusNotReady 环境未就绪
	EnvStatusNotReady EnvStatus = "NotReady"
)

// EnvironmentKind 环境类别
type EnvironmentKind string

const (
	// EnvironmentKindStandard 标准环境
	EnvironmentKindStandard EnvironmentKind = "standard"
	// EnvironmentKindFeature 特性环境，一定隶属于一个特定应用
	EnvironmentKindFeature EnvironmentKind = "feature"
)

// Environment 环境信息
type Environment struct {
	// ID 环境 ID
	ID bson.ObjectID `bson:"_id,omitempty"`

	// Name 环境名称, 即英文标识
	Name string `bson:"name"`
	// DisplayName 环境展示名称, 通常记录环境中文名
	DisplayName string `bson:"displayName"`
	// Type 环境类型, 可选值 development、test、staging 或 production
	Type string `bson:"type"`

	// WorkspaceID 环境所属空间 ID
	WorkspaceID string `bson:"workspaceID"`

	// Kind 环境类别。历史数据缺失时视为标准环境
	Kind EnvironmentKind `bson:"kind,omitempty"`

	// OwnerAppID 特性环境所属应用 ID，仅特性环境使用
	OwnerAppID string `bson:"ownerAppID,omitempty"`
	// SourceEnvID 特性环境来源环境 ID，仅特性环境使用
	SourceEnvID bson.ObjectID `bson:"sourceEnvID,omitempty"`

	// Cluster 业务集群信息
	Cluster BizCluster `bson:"cluster"`

	// Description 环境描述
	Description string `bson:"description"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`

	// AppIDs 应用 ID 列表，可能包含部署成功、部署失败、部署中的 AppID
	// 第一次尝试部署时加入，手动卸载时移除
	AppIDs []string `bson:"appIDs"`

	// TODO 增加 TrafficLaneIDs, ComponentIDs 等关联信息. 可能会用于环境列表展示和详情的关联查询

	// Status 环境状态. 暂时不持久化到 db 中, 先根据 Cluster 计算
	Status EnvStatus `bson:"-"`
}

// GetKind 返回环境类别；兼容历史记录缺失 kind 的情况。
func (e Environment) GetKind() EnvironmentKind {
	return cmp.Or(e.Kind, EnvironmentKindStandard)
}

// IsFeatureEnv 判断是否为特性环境。
func (e Environment) IsFeatureEnv() bool {
	return e.GetKind() == EnvironmentKindFeature
}

// BizCluster 业务集群信息
type BizCluster struct {
	// ProjectCode 蓝鲸 BCS 项目 Code
	ProjectCode string `bson:"projectCode"`
	// ClusterID 蓝鲸 BCS 集群 ID
	ClusterID string `bson:"clusterID"`
	// ClusterType 蓝鲸 BCS 集群类型, 如：single 等
	ClusterType string `bson:"clusterType"`
	// Namespace 蓝鲸 BCS 命名空间
	Namespace string `bson:"namespace"`
}
