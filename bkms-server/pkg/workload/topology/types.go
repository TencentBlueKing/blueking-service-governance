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
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ======================== 常量定义 ========================

// RefreshStatus 刷新状态枚举，用于 ResourceSnapshot.RefreshStatus 字段，标识资源范围的刷新进度
const (
	// RefreshStatusProgressing 刷新中，Refresher 开始刷新时设置
	RefreshStatusProgressing = "progressing"
	// RefreshStatusSuccess 刷新成功，数据可用于拓扑构图
	RefreshStatusSuccess = "success"
	// RefreshStatusFailed 刷新失败，保留上一版可用数据
	RefreshStatusFailed = "failed"
)

// TargetNameWildcard 通配符常量，用于 ResourceRelation.TargetName 字段，
// 表示目标名称为通配匹配（如 label_selector 关系中指向所有匹配的 Pod）
const TargetNameWildcard = "*"

// SourceType 资源来源类型枚举，用于 ResourceEntry.SourceType 字段，
// 描述资源是通过什么途径被纳入当前拓扑范围的
const (
	// SourceTypeHelmManifest 来源于 Helm Release Manifest 解析（Helm 部署场景）
	SourceTypeHelmManifest = "helm_manifest"
	// SourceTypeAppModelDeploy 来源于 AppModel 部署记录中的 ResourceKeys
	SourceTypeAppModelDeploy = "appmodel_deploy"
	// SourceTypeOwnerReference 通过 ownerReferences 自动发现的子资源（如 ReplicaSet、Pod）
	SourceTypeOwnerReference = "owner_reference"
)

// RelationType 关系类型，同时用于存储层（ResourceRelation.RelationType 字段）和
// API 层（EdgeReason.Type 字段），描述两个资源之间的关系是如何被发现/推导出来的
// RelationType 拥有更详细的信息，主要用于存储 / 日志 / 调试场景
type RelationType string

const (
	// RelationTypeOwnerReference 通过 metadata.ownerReferences 发现（如 Deployment→ReplicaSet）
	RelationTypeOwnerReference RelationType = "owner_reference"
	// RelationTypeLabelSelector 通过 spec.selector 标签匹配发现（如 Service→Pod）
	RelationTypeLabelSelector RelationType = "label_selector"
	// RelationTypeVolumeMount 通过 spec.volumes 挂载引用发现（如 Deployment→ConfigMap/Secret）
	RelationTypeVolumeMount RelationType = "volume_mount"
	// RelationTypeBackendRef 通过 Ingress backend 或 HTTPRoute 引用发现（如 Ingress→Service）
	RelationTypeBackendRef RelationType = "backend_ref"
	// RelationTypeEnvRef 通过环境变量 valueFrom 引用发现（如 Deployment→ConfigMap/Secret）
	RelationTypeEnvRef RelationType = "env_ref"
	// RelationTypeScaleTargetRef 通过 HPA 的 spec.scaleTargetRef 发现（如 HPA→Deployment）
	RelationTypeScaleTargetRef RelationType = "scale_target_ref"
	// RelationTypeServiceAccountRef 通过 spec.template.spec.serviceAccountName 发现（如 Deployment→ServiceAccount）
	RelationTypeServiceAccountRef RelationType = "service_account_ref"
	// RelationTypeAppRoot 虚拟根节点到顶层资源 MANAGES 边的 reason.type（仅 API 层使用）
	RelationTypeAppRoot RelationType = "app_root"
)

// EdgeRelation 边关系类型，用于 Edge.Relation 字段。
// 主边（IsPrimary=true）用于组织树结构，建议实线展示；
// 辅助边（IsPrimary=false）用于横向依赖，建议虚线展示
// EdgeRelation 拥有更高的可读性，主要用于 API / 前端页面展示
type EdgeRelation string

const (
	// EdgeRelationManages 管理关系（主边），APP 根节点到顶层资源、或未知父子关系的默认值
	EdgeRelationManages EdgeRelation = "MANAGES"
	// EdgeRelationCreates 创建关系（主边），所有 ownerReferences 关联的父子资源统一使用
	// 如 Deployment→ReplicaSet、ReplicaSet→Pod、StatefulSet→Pod、DaemonSet→Pod、Job→Pod、CronJob→Job
	EdgeRelationCreates EdgeRelation = "CREATES"
	// EdgeRelationSelects 选择关系（辅助边），如 Service 通过 labelSelector 选择 Pod
	EdgeRelationSelects EdgeRelation = "SELECTS"
	// EdgeRelationMounts 挂载关系（辅助边），如 Deployment 通过 volumes 挂载 ConfigMap/Secret
	EdgeRelationMounts EdgeRelation = "MOUNTS"
	// EdgeRelationRoutes 路由关系（辅助边），如 Ingress 通过 backend 路由到 Service
	EdgeRelationRoutes EdgeRelation = "ROUTES_TO"
	// EdgeRelationScales 扩缩关系（辅助边），如 HPA 通过 scaleTargetRef 关联到 Deployment
	EdgeRelationScales EdgeRelation = "SCALES"
	// EdgeRelationReferences 引用关系（辅助边），
	// 如 Deployment 通过 env valueFrom/envFrom 引用 ConfigMap/Secret，
	// 或通过 serviceAccountName 引用 ServiceAccount。通过 EdgeReason.Type 区分具体引用途径
	EdgeRelationReferences EdgeRelation = "REFERENCES"
)

// 虚拟根节点相关常量，Builder.buildAppRootNodeAndEdges 使用，为拓扑树提供统一根入口
const (
	// NodeKindApp 虚拟根节点的 Kind，不对应实际 K8s 资源
	NodeKindApp = "App"
)

// ======================== API 响应模型 ========================

// Graph 拓扑图，拓扑主 API 的顶层响应结构
type Graph struct {
	// Metadata 拓扑元信息（appID、envName、clusterID 等）
	Metadata Metadata
	// Nodes 拓扑图中的所有节点
	Nodes []Node
	// Edges 拓扑图中的所有边
	Edges []Edge
	// RootID 根节点 ID（APP 虚拟根节点）
	RootID string
	// GeneratedAt 拓扑图生成时间（RFC3339 格式）
	GeneratedAt string
	// IsPartial 是否为不完整拓扑（部分资源获取失败时为 true）
	IsPartial bool
	// Warnings 构建过程中产生的警告信息
	Warnings []string
	// DataVersion 数据版本号，与 ResourceSnapshot.DataVersion 对应
	DataVersion int64
}

// Metadata 拓扑元信息
type Metadata struct {
	AppID           string
	EnvName         string
	TrafficLaneName string
	ClusterID       string
	Namespace       string
}

// Node 拓扑节点
type Node struct {
	ID          string
	Kind        string
	Namespace   string
	Name        string
	DisplayName string
	// Status 节点综合状态码（对应 k8sstatus.Result.Code），枚举值见 k8sstatus 包常量
	Status string
	// Reason 状态补充说明（对应 k8sstatus.Result.Message），解释"为什么是这个状态"，可能为空字符串
	Reason    string
	IsManaged bool
	Extras    map[string]string
}

// Node extras key 常量定义，用于 Node.Extras map 的 key。
// 不同 Kind 可提供的 extras key 如下：
//
//	Pod:                             image, ip
//	Deployment/StatefulSet/DaemonSet: image, readyReplicas, replicas
//	Service:                         ports, selector, clusterIP, type
//	Ingress:                         host
//	ReplicaSet:                      image, readyReplicas, replicas
const (
	// ExtrasKeyImage 容器镜像地址（Pod/Deployment/StatefulSet/DaemonSet/ReplicaSet）
	ExtrasKeyImage = "image"
	// ExtrasKeyPorts Service 暴露的端口列表
	ExtrasKeyPorts = "ports"
	// ExtrasKeySelector Service 的 spec.selector 标签选择器
	ExtrasKeySelector = "selector"
	// ExtrasKeyHost Ingress 的主机名
	ExtrasKeyHost = "host"
	// ExtrasKeyClusterIP Service 的 ClusterIP 地址
	ExtrasKeyClusterIP = "clusterIP"
	// ExtrasKeyServiceType Service 类型（ClusterIP/NodePort/LoadBalancer）
	ExtrasKeyServiceType = "type"
	// ExtrasKeyReplicas 期望副本数（Deployment/StatefulSet/DaemonSet/ReplicaSet）
	ExtrasKeyReplicas = "replicas"
	// ExtrasKeyReadyReplicas 就绪副本数（Deployment/StatefulSet/DaemonSet/ReplicaSet）
	ExtrasKeyReadyReplicas = "readyReplicas"
	// ExtrasKeyPodIP Pod 的 IP 地址
	ExtrasKeyPodIP = "ip"
	// ExtrasKeyNodeName Pod 所在的节点名称
	ExtrasKeyNodeName = "nodeName"
	// ExtrasKeyRestartCount Pod 容器的累计重启次数
	ExtrasKeyRestartCount = "restartCount"
	// ExtrasKeyReady Pod 的就绪状态（true/false）
	ExtrasKeyReady = "ready"
	// ExtrasKeyPhase Pod 的生命周期阶段（Running/Pending/Succeeded/Failed/Unknown）
	ExtrasKeyPhase = "phase"
	// ExtrasKeyAvailableReplicas Deployment 可用的副本数
	ExtrasKeyAvailableReplicas = "availableReplicas"
	// ExtrasKeyStrategy Deployment 的更新策略类型（RollingUpdate/Recreate）
	ExtrasKeyStrategy = "strategy"
	// ExtrasKeyOwnerDeployment ReplicaSet 所属的 Deployment 名称
	ExtrasKeyOwnerDeployment = "ownerDeployment"
	// ExtrasKeyKeys ConfigMap/Secret 的键列表（逗号分隔）
	ExtrasKeyKeys = "keys"
	// ExtrasKeyDataSize ConfigMap 的 data 字段总大小（字节数）
	ExtrasKeyDataSize = "dataSize"
	// ExtrasKeyBinaryDataSize ConfigMap 的 binaryData 字段总大小（字节数）
	ExtrasKeyBinaryDataSize = "binaryDataSize"
	// ExtrasKeySecretType Secret 的类型（Opaque/kubernetes.io/tls 等）
	ExtrasKeySecretType = "secretType"
	// ExtrasKeySecrets ServiceAccount 引用的 secrets 名称列表（逗号分隔）
	ExtrasKeySecrets = "secrets"
	// ExtrasKeyAutomountToken ServiceAccount 的 automountServiceAccountToken 设置
	ExtrasKeyAutomountToken = "automountToken"
	// ExtrasKeyRules Ingress 的路由规则列表
	ExtrasKeyRules = "rules"
	// ExtrasKeyTLS Ingress 的 TLS 配置
	ExtrasKeyTLS = "tls"
)

// Edge 拓扑边，描述两个节点之间的关系。
//
// 示例（Service→Pod 的 SELECTS 辅助边）：
//
//	Edge{
//	    ID:        "U2VydmljZS9kZWZhdWx0L25naW54LXN2Yy0-...",
//	    SourceID:  "U2VydmljZS9kZWZhdWx0L25naW54LXN2Yw",
//	    TargetID:  "UG9kL2RlZmF1bHQvbmdpbngtN2Y0YmNmOTZkNi02enR4aw",
//	    Relation:  "SELECTS",
//	    IsPrimary: false,
//	    Reason:    EdgeReason{Type: "label_selector", ...},
//	}
type Edge struct {
	// ID 边的唯一标识，由 EncodeEdgeID 生成（base64url 无填充编码，明文格式: {srcKind/ns/name}->{tgtKind/ns/name}:{relation}）
	ID string
	// SourceID 源节点 ID（与 Node.ID 对应），表示关系的发起方。示例: Service 节点的编码 ID
	SourceID string
	// TargetID 目标节点 ID（与 Node.ID 对应），表示关系的指向方。示例: Pod 节点的编码 ID
	TargetID string
	// Relation 关系类型，取值为 EdgeRelation 常量（MANAGES/CREATES/SELECTS/MOUNTS/ROUTES_TO/SCALES/REFERENCES）
	Relation EdgeRelation
	// IsPrimary 是否为主边。主边构成树结构（实线展示），辅助边表达横向依赖（虚线展示）
	IsPrimary bool
	// Reason 边产生的原因，解释"为什么存在这条边"，前端可在悬浮提示/详情面板中展示
	Reason EdgeReason
}

// EdgeReason 边关系原因，解释边是如何被推导出来的。
//
// 不同场景的示例：
//
// 1. Service→Pod 的 SELECTS 边（label_selector）：
//
//	EdgeReason{
//	    Type:            "label_selector",
//	    Summary:         "Service/nginx-svc selects Pod by labels: app=nginx",
//	    MatchedLabels:   map[string]string{"app": "nginx"},
//	    SourceFieldPath: "spec.selector",
//	    TargetFieldPath: "metadata.labels",
//	}
//
// 2. Deployment→ReplicaSet 的 CREATES 边（owner_reference）：
//
//	EdgeReason{
//	    Type:    "owner_reference",
//	    Summary: "matched by metadata.ownerReferences",
//	}
//
// 3. Ingress→Service 的 ROUTES_TO 边（backend_ref）：
//
//	EdgeReason{
//	    Type:            "backend_ref",
//	    Summary:         "Ingress/my-ingress routes to Service/my-svc",
//	    SourceFieldPath: "spec.rules[0].http.paths[0].backend.service.name",
//	    TargetFieldPath: "metadata.name",
//	}
type EdgeReason struct {
	// Type 判定类型，取值为 RelationType 常量（owner_reference/label_selector/volume_mount/backend_ref/app_root 等）
	Type RelationType
	// Summary 可读摘要，简短描述关系来源。示例: "Service/nginx-svc selects Pod by labels: app=nginx"
	Summary string
	// MatchedLabels 匹配的标签 KV 对，仅在 Type 为 label_selector 时填充。示例: {"app": "nginx", "tier": "backend"}
	MatchedLabels map[string]string
	// SourceFieldPath 源资源中产生此关系的字段路径，可选。示例: "spec.selector"、"spec.rules[0].http.paths[0].backend.service.name"
	SourceFieldPath string
	// TargetFieldPath 目标资源中被引用的字段路径，可选。示例: "metadata.labels"、"metadata.name"
	TargetFieldPath string
}

// ErrNodeNotInSnapshot 节点 ID 未通过快照范围校验（不属于当前部署的资源快照范围）
var ErrNodeNotInSnapshot = errors.New("node does not belong to the current resource snapshot")

// NodeDetail 节点详情领域模型
type NodeDetail struct {
	// ID 节点 ID（base64url 编码）
	ID string
	// Kind 资源类型
	Kind string
	// Namespace 命名空间
	Namespace string
	// Name 资源名称
	Name string
	// CreatedAt 创建时间（ISO 8601 格式）
	CreatedAt string
	// Extras 类型专属扩展字段（复用 kindExtrasProviders）
	Extras map[string]string
	// Conditions 资源 conditions 列表
	Conditions []Condition
}

// Condition K8s 资源条件
type Condition struct {
	// Type condition 类型（如 Ready、Available、Progressing）
	Type string
	// Status condition 状态（True/False/Unknown）
	Status string
	// Reason condition 原因
	Reason string
	// Message condition 消息
	Message string
	// LastTransitionTime 上次状态变更时间（ISO 8601 格式）
	LastTransitionTime string
}

// NodeManifest 节点 Manifest 领域模型
type NodeManifest struct {
	// Content YAML/JSON 字符串内容
	Content string
	// Format 格式（yaml 或 json）
	Format string
	// Truncated 是否被截断
	Truncated bool
}

// ======================== ID 编码/解码辅助函数 ========================

// ResourceKey 将 kind、namespace、name 拼接为 "kind/namespace/name" 格式的资源唯一键，
// 用于 clusterResources map key、staticSet key、Node ID 明文等所有需要唯一标识资源的场景
func ResourceKey(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

// EncodeNodeID 将节点的明文 ID（{kind}/{namespace}/{name}）编码为 base64url 无填充格式
func EncodeNodeID(kind, namespace, name string) string {
	plaintext := ResourceKey(kind, namespace, name)
	return base64.RawURLEncoding.EncodeToString([]byte(plaintext))
}

// DecodeNodeID 将 base64url 无填充编码的节点 ID 解码为 kind, namespace, name
func DecodeNodeID(encoded string) (kind, namespace, name string, err error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", "", errors.Wrap(err, "decode node ID")
	}

	parts := strings.SplitN(string(decoded), "/", 3)
	if len(parts) != 3 {
		return "", "", "", errors.Errorf(
			"invalid node ID format, expected 3 parts separated by '/', got %d", len(parts),
		)
	}
	return parts[0], parts[1], parts[2], nil
}

// EncodeEdgeID 将边的明文 ID（{sourceNodeID}->{targetNodeID}:{relation}）编码为 base64url 无填充格式
func EncodeEdgeID(
	sourceKind, sourceNS, sourceName, targetKind, targetNS, targetName string,
	relation EdgeRelation,
) string {
	sourceNodePlain := ResourceKey(sourceKind, sourceNS, sourceName)
	targetNodePlain := ResourceKey(targetKind, targetNS, targetName)
	plaintext := fmt.Sprintf("%s->%s:%s", sourceNodePlain, targetNodePlain, relation)
	return base64.RawURLEncoding.EncodeToString([]byte(plaintext))
}
