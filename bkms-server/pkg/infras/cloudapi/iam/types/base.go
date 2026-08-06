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

// Package types 定义蓝鲸 IAM 网关 HTTP 客户端通用的请求 / 响应基础类型
package types

// ResourceType 资源类型
type ResourceType string

const (
	// BKMS 资源类型

	// WorkspaceResourceType 表示空间资源类型
	WorkspaceResourceType ResourceType = "workspace"
	// AppResourceType 表示应用资源类型
	AppResourceType ResourceType = "app"
	// EnvResourceType 表示环境资源类型
	EnvResourceType ResourceType = "env"

	// BKCI 资源类型

	// ProjectResourceType 表示项目资源类型
	ProjectResourceType ResourceType = "project"
	// PipelineResourceType 表示流水线资源类型
	PipelineResourceType ResourceType = "pipeline"
	// RepertoryResourceType 表示代码仓库资源类型
	RepertoryResourceType ResourceType = "repertory"
	// CredentialResourceType 表示凭证资源类型
	CredentialResourceType ResourceType = "credential"

	// BCS 资源类型

	// BCSProjectResourceType 表示 BCS 项目资源类型
	BCSProjectResourceType ResourceType = "project"
	// ClusterResourceType 表示集群资源类型
	ClusterResourceType ResourceType = "cluster"
	// NamespaceResourceType 表示命名空间资源类型
	NamespaceResourceType ResourceType = "namespace"

	// BKMonitor 资源类型

	// SpaceResourceType 表示监控空间资源类型
	SpaceResourceType ResourceType = "space"
	// DashboardResourceType 表示仪表盘资源类型
	DashboardResourceType ResourceType = "grafana_dashboard"
	// APMResourceType 表示 APM 资源类型
	APMResourceType ResourceType = "apm_application"

	// BKLog 资源类型

	// IndicesResourceType 表示索引集资源类型
	IndicesResourceType ResourceType = "indices"
	// CollectionResourceType 表示采集项资源类型
	CollectionResourceType ResourceType = "collection"
	// ESSourceResourceType 表示 ES 源资源类型
	ESSourceResourceType ResourceType = "es_source"

	// BKRepo 资源类型

	// BKRepoProjectResourceType 表示 bk-repo 项目资源类型
	BKRepoProjectResourceType ResourceType = "project"
	// RepositoryResourceType 表示仓库资源类型
	RepositoryResourceType ResourceType = "repo"
	// RepoNodeResourceType 表示节点资源类型
	RepoNodeResourceType ResourceType = "node"
)

// ResourcePath 资源路径
type ResourcePath struct {
	System string       `json:"system"`
	Type   ResourceType `json:"type"`
	ID     string       `json:"id"`
	Name   string       `json:"name"`
}

// ResourcePaths 资源拓扑
type ResourcePaths []ResourcePath

// Resource 表示权限描述的资源
type Resource struct {
	System string          `json:"system"`
	Type   ResourceType    `json:"type"`
	Paths  []ResourcePaths `json:"paths"`
}

// Action 表示资源动作, 如 create_app
type Action struct {
	ID string `json:"id"`
}

// AuthorizationScope 授权范围
type AuthorizationScope struct {
	System    string     `json:"system"`
	Actions   []Action   `json:"actions"`
	Resources []Resource `json:"resources"`
}

// SubjectScope 主体范围
type SubjectScope struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
