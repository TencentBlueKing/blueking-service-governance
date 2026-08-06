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

// Package serializer defines Gin input and output serializers for topology APIs.
package serializer

import (
	"strconv"
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// AppEnvURIInput is the path input for APIs scoped by application and environment.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// TopologyNodeURIInput is the path input for topology node APIs.
type TopologyNodeURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 拓扑节点 ID（base64url 无填充编码）
	NodeID string `uri:"nodeID" binding:"required,min=1"`
}

// TrafficLaneQueryInput is the query input for APIs that support traffic lane filtering.
type TrafficLaneQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// ListTopologyNodeEventsQueryInput is the query input for listing topology node events.
type ListTopologyNodeEventsQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 事件级别（可选过滤参数，可选值：Normal, Warning）
	Level string `form:"level"`
	// 起始时间戳（可选过滤参数，如：1772223278）
	StartedAt int64 `form:"startedAt"`
	// 结束时间戳（可选过滤参数，如：1772223278）
	EndedAt int64 `form:"endedAt"`
	// 分页页码（从 1 开始）
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// GetResourceTopologyOutput is the JSON response for querying resource topology.
type GetResourceTopologyOutput struct {
	// 拓扑数据
	Data *ResourceTopologyDataOutputObj `json:"data"`
}

// ResourceTopologyDataOutputObj is the JSON representation of a resource topology graph.
type ResourceTopologyDataOutputObj struct {
	// 拓扑元信息
	Metadata *TopologyMetadataOutputObj `json:"metadata"`
	// 拓扑节点列表
	Nodes []*TopologyNodeOutputObj `json:"nodes"`
	// 拓扑边列表
	Edges []*TopologyEdgeOutputObj `json:"edges"`
	// 拓扑根节点 ID（base64url 无填充编码）
	RootID string `json:"rootID"`
	// 生成时间（ISO 8601 格式）
	GeneratedAt string `json:"generatedAt"`
	// 是否为部分拓扑
	IsPartial bool `json:"isPartial"`
	// 警告信息列表
	Warnings []string `json:"warnings"`
	// 数据版本号
	DataVersion string `json:"dataVersion"`
}

// FromModel fills output fields from a topology graph.
func (o *ResourceTopologyDataOutputObj) FromModel(graph *topology.Graph) *ResourceTopologyDataOutputObj {
	nodes := make([]*TopologyNodeOutputObj, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes = append(nodes, new(TopologyNodeOutputObj).FromModel(node))
	}

	edges := make([]*TopologyEdgeOutputObj, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		edges = append(edges, new(TopologyEdgeOutputObj).FromModel(edge))
	}

	*o = ResourceTopologyDataOutputObj{
		Metadata:    new(TopologyMetadataOutputObj).FromModel(graph.Metadata),
		Nodes:       nodes,
		Edges:       edges,
		RootID:      graph.RootID,
		GeneratedAt: graph.GeneratedAt,
		IsPartial:   graph.IsPartial,
		Warnings:    graph.Warnings,
		DataVersion: strconv.FormatInt(graph.DataVersion, 10),
	}
	return o
}

// TopologyMetadataOutputObj is the JSON representation of topology metadata.
type TopologyMetadataOutputObj struct {
	// 应用 ID
	AppID string `json:"appID"`
	// 环境名称
	EnvName string `json:"envName"`
	// 泳道名称
	TrafficLaneName string `json:"trafficLaneName"`
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 主命名空间
	Namespace string `json:"namespace"`
}

// FromModel fills output fields from topology metadata.
func (o *TopologyMetadataOutputObj) FromModel(metadata topology.Metadata) *TopologyMetadataOutputObj {
	*o = TopologyMetadataOutputObj{
		AppID:           metadata.AppID,
		EnvName:         metadata.EnvName,
		TrafficLaneName: metadata.TrafficLaneName,
		ClusterID:       metadata.ClusterID,
		Namespace:       metadata.Namespace,
	}
	return o
}

// TopologyNodeOutputObj is the JSON representation of a topology node.
type TopologyNodeOutputObj struct {
	// 拓扑节点 ID（base64url 无填充编码，内部格式 {kind}/{namespace}/{name}）
	ID string `json:"id"`
	// 资源类型
	Kind string `json:"kind"`
	// 命名空间
	Namespace string `json:"namespace"`
	// 资源名称
	Name string `json:"name"`
	// 显示名称
	DisplayName string `json:"displayName"`
	// 资源状态（如 Running、Deployed、Healthy、Degraded）
	Status string `json:"status"`
	// 状态补充说明（对应 k8sstatus.Result.Message），可能为空字符串
	Reason string `json:"reason"`
	// 是否为部署直接管理的资源
	IsManaged bool `json:"isManaged"`
	// 类型专属扩展字段（key 为字段名，value 为字符串值）
	Extras map[string]string `json:"extras"`
}

// FromModel fills output fields from a topology node.
func (o *TopologyNodeOutputObj) FromModel(node topology.Node) *TopologyNodeOutputObj {
	*o = TopologyNodeOutputObj{
		ID:          node.ID,
		Kind:        node.Kind,
		Namespace:   node.Namespace,
		Name:        node.Name,
		DisplayName: node.DisplayName,
		Status:      node.Status,
		Reason:      node.Reason,
		IsManaged:   node.IsManaged,
		Extras:      node.Extras,
	}
	return o
}

// TopologyEdgeOutputObj is the JSON representation of a topology edge.
type TopologyEdgeOutputObj struct {
	// 边 ID（base64url 无填充编码）
	ID string `json:"id"`
	// 源拓扑节点 ID（base64url 编码）
	SourceID string `json:"sourceID"`
	// 目标拓扑节点 ID（base64url 编码）
	TargetID string `json:"targetID"`
	// 关系类型（MANAGES、OWNS、CREATES、SELECTS、MOUNTS、ROUTES_TO 等）
	Relation topology.EdgeRelation `json:"relation"`
	// 是否为主边（形成树结构）
	IsPrimary bool `json:"isPrimary"`
	// 关系原因
	Reason *EdgeReasonOutputObj `json:"reason"`
}

// FromModel fills output fields from a topology edge.
func (o *TopologyEdgeOutputObj) FromModel(edge topology.Edge) *TopologyEdgeOutputObj {
	*o = TopologyEdgeOutputObj{
		ID:        edge.ID,
		SourceID:  edge.SourceID,
		TargetID:  edge.TargetID,
		Relation:  edge.Relation,
		IsPrimary: edge.IsPrimary,
		Reason:    new(EdgeReasonOutputObj).FromModel(edge.Reason),
	}
	return o
}

// EdgeReasonOutputObj is the JSON representation of a topology edge reason.
type EdgeReasonOutputObj struct {
	// 判定类型（owner_reference、label_selector、volume_mount、backend_ref、helm_manifest）
	Type topology.RelationType `json:"type"`
	// 可读摘要
	Summary string `json:"summary"`
	// 匹配的标签（适用于 label_selector）
	MatchedLabels map[string]string `json:"matchedLabels"`
	// 源字段路径
	SourceFieldPath string `json:"sourceFieldPath"`
	// 目标字段路径
	TargetFieldPath string `json:"targetFieldPath"`
}

// FromModel fills output fields from a topology edge reason.
func (o *EdgeReasonOutputObj) FromModel(reason topology.EdgeReason) *EdgeReasonOutputObj {
	*o = EdgeReasonOutputObj{
		Type:            reason.Type,
		Summary:         reason.Summary,
		MatchedLabels:   reason.MatchedLabels,
		SourceFieldPath: reason.SourceFieldPath,
		TargetFieldPath: reason.TargetFieldPath,
	}
	return o
}

// GetTopologyNodeDetailOutput is the JSON response for querying topology node detail.
type GetTopologyNodeDetailOutput struct {
	// 拓扑节点详情数据
	Data *TopologyNodeDetailOutputObj `json:"data"`
}

// TopologyNodeDetailOutputObj is the JSON representation of topology node detail.
type TopologyNodeDetailOutputObj struct {
	// 拓扑节点 ID（base64url 无填充编码）
	ID string `json:"id"`
	// 资源类型
	Kind string `json:"kind"`
	// 命名空间
	Namespace string `json:"namespace"`
	// 资源名称
	Name string `json:"name"`
	// 创建时间（ISO 8601 格式）
	CreatedAt string `json:"createdAt"`
	// 类型专属扩展字段（复用 kindExtrasProviders 注册表）
	Extras map[string]string `json:"extras"`
	// 资源 conditions 列表
	Conditions []*TopologyNodeConditionOutputObj `json:"conditions"`
}

// FromModel fills output fields from topology node detail.
func (o *TopologyNodeDetailOutputObj) FromModel(detail *topology.NodeDetail) *TopologyNodeDetailOutputObj {
	conditions := make([]*TopologyNodeConditionOutputObj, 0, len(detail.Conditions))
	for _, condition := range detail.Conditions {
		conditions = append(conditions, new(TopologyNodeConditionOutputObj).FromModel(condition))
	}

	*o = TopologyNodeDetailOutputObj{
		ID:         detail.ID,
		Kind:       detail.Kind,
		Namespace:  detail.Namespace,
		Name:       detail.Name,
		CreatedAt:  detail.CreatedAt,
		Extras:     detail.Extras,
		Conditions: conditions,
	}
	return o
}

// TopologyNodeConditionOutputObj is the JSON representation of a topology node condition.
type TopologyNodeConditionOutputObj struct {
	// condition 类型
	Type string `json:"type"`
	// condition 状态（True/False/Unknown）
	Status string `json:"status"`
	// condition 原因
	Reason string `json:"reason"`
	// condition 消息
	Message string `json:"message"`
	// 上次状态变更时间（ISO 8601 格式）
	LastTransitionTime string `json:"lastTransitionTime"`
}

// FromModel fills output fields from a topology condition.
func (o *TopologyNodeConditionOutputObj) FromModel(condition topology.Condition) *TopologyNodeConditionOutputObj {
	*o = TopologyNodeConditionOutputObj{
		Type:               condition.Type,
		Status:             condition.Status,
		Reason:             condition.Reason,
		Message:            condition.Message,
		LastTransitionTime: condition.LastTransitionTime,
	}
	return o
}

// ListTopologyNodeEventsOutput is the JSON response for listing topology node events.
type ListTopologyNodeEventsOutput struct {
	// 分页事件数据
	Data *PaginatedTopologyNodeEventsOutputObj `json:"data"`
}

// PaginatedTopologyNodeEventsOutputObj is the JSON representation of paginated topology node events.
type PaginatedTopologyNodeEventsOutputObj struct {
	// 事件总数
	Count int64 `json:"count,string"`
	// 事件列表（按时间倒序排列）
	Results []TopologyNodeEventOutputObj `json:"results"`
}

// FromModel fills output fields from paginated cluster events.
func (o *PaginatedTopologyNodeEventsOutputObj) FromModel(
	events *clusterresources.PaginatedEvents,
) *PaginatedTopologyNodeEventsOutputObj {
	results := make([]TopologyNodeEventOutputObj, 0, len(events.Data))
	for _, event := range events.Data {
		results = append(results, new(TopologyNodeEventOutputObj).FromModel(event))
	}

	return &PaginatedTopologyNodeEventsOutputObj{
		Count:   events.Count,
		Results: results,
	}
}

// TopologyNodeEventOutputObj is the JSON representation of a topology node event.
type TopologyNodeEventOutputObj struct {
	// BCS 集群 ID
	ClusterID string `json:"clusterID"`
	// 命名空间
	Namespace string `json:"namespace"`
	// 事件级别（如：Normal, Warning）
	Level string `json:"level"`
	// 事件内容
	Content string `json:"content"`
	// 事件类型（如：Completed, Pulled, Started 等）
	Type string `json:"type"`
	// 组件名称
	ComponentName string `json:"componentName"`
	// 关联的资源类型，如：Deployment, Pod，Node 等
	ResourceKind string `json:"resourceKind"`
	// 关联的资源名称，如：nginx-ingress-2695bd-58877d456b
	ResourcesName string `json:"resourcesName"`
	// 事件创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// FromModel fills output fields from a cluster event entry.
func (o TopologyNodeEventOutputObj) FromModel(event clusterresources.EventEntry) TopologyNodeEventOutputObj {
	return TopologyNodeEventOutputObj{
		ClusterID:     event.ClusterID,
		Namespace:     event.Namespace,
		Level:         event.Level,
		Content:       event.Content,
		Type:          event.Type,
		ComponentName: event.ComponentName,
		ResourceKind:  event.ResourceKind,
		ResourcesName: event.ResourcesName,
		CreatedAt:     event.CreatedAt.UTC(),
	}
}

// GetTopologyNodeManifestOutput is the JSON response for querying topology node manifest.
type GetTopologyNodeManifestOutput struct {
	// Manifest 数据
	Data *TopologyNodeManifestOutputObj `json:"data"`
}

// TopologyNodeManifestOutputObj is the JSON representation of a topology node manifest.
type TopologyNodeManifestOutputObj struct {
	// YAML/JSON 字符串内容
	Content string `json:"content"`
	// 格式（yaml 或 json）
	Format string `json:"format"`
	// 是否被截断
	Truncated bool `json:"truncated"`
}

// FromModel fills output fields from topology node manifest.
func (o *TopologyNodeManifestOutputObj) FromModel(manifest *topology.NodeManifest) *TopologyNodeManifestOutputObj {
	*o = TopologyNodeManifestOutputObj{
		Content:   manifest.Content,
		Format:    manifest.Format,
		Truncated: manifest.Truncated,
	}
	return o
}
