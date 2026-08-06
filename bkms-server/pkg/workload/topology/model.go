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

// Package topology 提供资源拓扑图的领域模型、存储和构建能力
package topology

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ======================== 持久化实体（MongoDB） ========================

// ResourceSnapshot 资源拓扑快照，表示一个 appID + envName + trafficLaneName 作用域下当前可见的资源集合和扩展关系
type ResourceSnapshot struct {
	ID              bson.ObjectID      `bson:"_id,omitempty"`
	AppID           string             `bson:"appID"`
	EnvName         string             `bson:"envName"`
	TrafficLaneName string             `bson:"trafficLaneName"`
	ClusterID       string             `bson:"clusterID"`
	Namespace       string             `bson:"namespace"`
	ReleaseName     string             `bson:"releaseName,omitempty"`
	DataVersion     int64              `bson:"dataVersion"`
	RefreshStatus   string             `bson:"refreshStatus"`
	RefreshedAt     time.Time          `bson:"refreshedAt,omitempty"`
	WarningSummary  string             `bson:"warningSummary,omitempty"`
	Resources       []ResourceEntry    `bson:"resources"`
	Relations       []ResourceRelation `bson:"relations"`
	CreatedAt       time.Time          `bson:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
}

// ResourceEntry 资源条目，表示资源范围内的单个 Kubernetes 资源身份
type ResourceEntry struct {
	// Kind 资源类型，如 Deployment、Service、ConfigMap 等
	Kind string `bson:"kind"`
	// APIVersion 资源的 API 版本，如 apps/v1、v1 等
	APIVersion string `bson:"apiVersion,omitempty"`
	// Namespace 资源所在的命名空间，集群级别资源为空
	Namespace string `bson:"namespace"`
	// Name 资源名称
	Name string `bson:"name"`
	// IsManaged 是否为应用直接管理的资源（true 表示由 Helm/AppModel 直接声明）
	IsManaged bool `bson:"isManaged"`
	// SourceType 资源来源类型，取值参考 SourceType 常量（如 helm_manifest、owner_reference 等）
	SourceType string `bson:"sourceType"`
	// SourceReason 资源来源的补充说明，描述该资源被纳入范围的具体原因
	SourceReason string `bson:"sourceReason,omitempty"`
}

// ResourceRelation 扩展关系，表示两个资源之间的可复用关系线索
type ResourceRelation struct {
	// RelationType 关系类型，取值参考 RelationType 常量（如 owner_reference、label_selector 等）
	RelationType RelationType `bson:"relationType"`
	// SourceKind 关系源端资源的 Kind
	SourceKind string `bson:"sourceKind"`
	// SourceNamespace 关系源端资源的命名空间
	SourceNamespace string `bson:"sourceNamespace"`
	// SourceName 关系源端资源的名称
	SourceName string `bson:"sourceName"`
	// TargetKind 关系目标端资源的 Kind
	TargetKind string `bson:"targetKind"`
	// TargetNamespace 关系目标端资源的命名空间
	TargetNamespace string `bson:"targetNamespace"`
	// TargetName 关系目标端资源的名称
	TargetName string `bson:"targetName"`
	// SourceFieldPath 源端触发关系的字段路径，如 spec.template.spec.volumes[0].configMap.name
	SourceFieldPath string `bson:"sourceFieldPath,omitempty"`
	// TargetFieldPath 目标端被引用的字段路径
	TargetFieldPath string `bson:"targetFieldPath,omitempty"`
	// Summary 关系的可读摘要描述
	Summary string `bson:"summary,omitempty"`
	// MatchedLabels 标签选择器场景下匹配到的标签键值对
	MatchedLabels map[string]string `bson:"matchedLabels,omitempty"`
}

// ResourceKeyEntry 资源引用条目（来自 AppModel 部署记录中的 ResourceKeys）
type ResourceKeyEntry struct {
	// Kind 资源类型，如：GameDeployment, Service, ConfigMap
	Kind string
	// Name 资源名称
	Name string
}

// RefreshArgs 刷新任务参数
type RefreshArgs struct {
	AppID           string
	EnvName         string
	TrafficLaneName string
	ClusterID       string
	Namespace       string
	// ReleaseName Helm Release 名称（Helm 部署场景使用）
	ReleaseName string
	// ResourceKeys 部署关联的资源引用列表（AppModel 部署场景使用）
	ResourceKeys []ResourceKeyEntry
	// LabelSelector 标签选择器（AppModel 部署场景使用，用于关联 Pod）
	LabelSelector map[string]string
}

// SnapshotKey 返回刷新参数的作用域键
func (a RefreshArgs) SnapshotKey() string {
	return fmt.Sprintf("%s/%s/%s", a.AppID, a.EnvName, a.TrafficLaneName)
}
