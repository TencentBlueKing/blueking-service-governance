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

// Package actions defines IAM action ID constants for the various business
// systems integrated with bkms (BKMS / BKCI / BCS / BKMonitor / BKLog / BKRepo).
//
// This package contains only constant declarations and depends on the
// standard library only.
package actions

// BKMS 资源操作

// WorkspaceAction 空间操作
var WorkspaceAction = struct{ Create, View, Edit, Delete string }{
	// 创建空间
	Create: "create_workspace",
	// 查看空间
	View: "view_workspace",
	// 编辑空间
	Edit: "edit_workspace",
	// 删除空间
	Delete: "delete_workspace",
}

// AppAction 应用操作
var AppAction = struct{ View, Edit, Delete, Create string }{
	// 查看应用
	View: "view_app",
	// 编辑应用
	Edit: "edit_app",
	// 删除应用
	Delete: "delete_app",
	// 创建应用
	Create: "create_app",
}

// EnvAction 环境操作
var EnvAction = struct{ View, Edit, Delete, Create, Deploy string }{
	// 查看环境
	View: "view_env",
	// 编辑环境
	Edit: "edit_env",
	// 删除环境
	Delete: "delete_env",
	// 创建环境
	Create: "create_env",
	// 部署环境
	Deploy: "deploy_env",
}

// BKCI 资源操作

// ProjectAction 项目操作
var ProjectAction = struct{ Visit, View, Edit, List, Enable string }{
	// 访问项目
	Visit: "project_visit",
	// 查看项目
	View: "project_view",
	// 编辑项目
	Edit: "project_edit",
	// 列举项目
	List: "project_list",
	// 启用/禁用项目
	Enable: "project_enable",
}

// PipelineAction 流水线操作
var PipelineAction = struct{ Create, List, View, Edit, Delete, Execute string }{
	// 创建流水线
	Create: "pipeline_create",
	// 列举流水线
	List: "pipeline_list",
	// 查看流水线
	View: "pipeline_view",
	// 编辑流水线
	Edit: "pipeline_edit",
	// 删除流水线
	Delete: "pipeline_delete",
	// 执行流水线
	Execute: "pipeline_execute",
}

// RepertoryAction 代码仓库操作. Repertory 采用了 BKCI 项目里的命名
var RepertoryAction = struct{ Create, List, View, Edit, Use, Delete string }{
	// 关联(创建)代码仓库
	Create: "repertory_create",
	// 列举代码仓库
	List: "repertory_list",
	// 查看代码仓库
	View: "repertory_view",
	// 编辑代码仓库
	Edit: "repertory_edit",
	// 使用代码仓库
	Use: "repertory_use",
	// 删除代码仓库
	Delete: "repertory_delete",
}

// CredentialAction 凭据操作
var CredentialAction = struct{ Create, List, View, Edit, Delete, Use string }{
	// 创建(添加)凭据
	Create: "credential_create",
	// 列举凭据
	List: "credential_list",
	// 查看凭据
	View: "credential_view",
	// 编辑凭据
	Edit: "credential_edit",
	// 删除凭据
	Delete: "credential_delete",
	// 使用凭据
	Use: "credential_use",
}

// BCS 资源操作

// BCSProjectAction 项目操作
var BCSProjectAction = struct{ View, Edit string }{
	// 查看项目
	View: "project_view",
	// 编辑项目
	Edit: "project_edit",
}

// ClusterAction 集群操作
var ClusterAction = struct{ Create, View, Manage, Delete string }{
	// 创建集群
	Create: "cluster_create",
	// 查看集群
	View: "cluster_view",
	// 管理集群
	Manage: "cluster_manage",
	// 删除集群
	Delete: "cluster_delete",
}

// ClusterScopedAction 集群域资源操作
var ClusterScopedAction = struct{ Create, View, Update, Delete string }{
	// 资源创建
	Create: "cluster_scoped_create",
	// 资源查看
	View: "cluster_scoped_view",
	// 资源更新
	Update: "cluster_scoped_update",
	// 资源删除
	Delete: "cluster_scoped_delete",
}

// NamespaceAction 命名空间操作
var NamespaceAction = struct{ Create, List, View, Update, Delete string }{
	// 创建命名空间
	Create: "namespace_create",
	// 列举命名空间
	List: "namespace_list",
	// 查看命名空间
	View: "namespace_view",
	// 更新命名空间
	Update: "namespace_update",
	// 删除命名空间
	Delete: "namespace_delete",
}

// NamespaceScopedAction 命名空间域操作
var NamespaceScopedAction = struct{ Create, View, Update, Delete string }{
	// 资源创建
	Create: "namespace_scoped_create",
	// 资源查看
	View: "namespace_scoped_view",
	// 资源更新
	Update: "namespace_scoped_update",
	// 资源删除
	Delete: "namespace_scoped_delete",
}

// BKMonitor 资源操作

// SpaceAction 空间操作
var SpaceAction = struct{ View string }{
	// 业务访问
	View: "view_business_v2",
}

// DashboardAction 仪表盘操作
var DashboardAction = struct{ New, Manage, View, Edit string }{
	// 新建仪表盘
	New: "new_dashboard",
	// 仪表盘配置管理
	Manage: "manage_datasource_v2",
	// 仪表盘查看
	View: "view_single_dashboard",
	// 仪表盘管理
	Edit: "edit_single_dashboard",
}

// EventAction 事件操作
var EventAction = struct{ View, Manage string }{
	// 事件中心查看
	View: "view_event_v2",
	// 事件中心管理
	Manage: "manage_event_v2",
}

// MetricAction 指标操作
var MetricAction = struct{ Explore string }{
	// 指标检索
	Explore: "explore_metric_v2",
}

// APMAction APM 应用操作
var APMAction = struct{ View, Manage string }{
	// APM 应用查看
	View: "view_apm_application_v2",
	// APM 应用管理
	Manage: "manage_apm_application_v2",
}

// AlertRuleAction 告警策略操作
var AlertRuleAction = struct{ View, Manage string }{
	// 告警策略查看
	View: "view_rule_v2",
	// 告警策略管理
	Manage: "manage_rule_v2",
}

// NotifyTeamAction 告警组操作
var NotifyTeamAction = struct{ View, Manage string }{
	// 告警组查看
	View: "view_notify_team_v2",
	// 告警组管理
	Manage: "manage_notify_team_v2",
}

// SilenceAction 屏蔽操作
var SilenceAction = struct{ View, Manage string }{
	// 屏蔽查看
	View: "view_downtime_v2",
	// 屏蔽管理
	Manage: "manage_downtime_v2",
}

// ConfigAction 配置操作
var ConfigAction = struct{ Export, Import string }{
	// 配置导出
	Export: "export_config_v2",
	// 配置导入
	Import: "import_config_v2",
}

// BKLog 资源操作

// IndicesAction 索引集操作
var IndicesAction = struct{ Create, Search, Manage string }{
	// 索引集配置新建
	Create: "create_indices_v2",
	// 日志检索
	Search: "search_log_v2",
	// 索引集配置管理
	Manage: "manage_indices_v2",
}

// CollectionAction 采集操作
var CollectionAction = struct{ Create, View, Manage string }{
	// 采集新建
	Create: "create_collection_v2",
	// 采集查看
	View: "view_collection_v2",
	// 采集管理
	Manage: "manage_collection_v2",
}

// ESSourceAction ES 源操作
var ESSourceAction = struct{ Create, Manage string }{
	// ES 源配置新建
	Create: "create_es_source_v2",
	// ES 源配置管理
	Manage: "manage_es_source_v2",
}

// ExtractConfigAction 日志提取配置操作
var ExtractConfigAction = struct{ Manage string }{
	// 日志提取配置管理
	Manage: "manage_extract_config_v2",
}

// BKRepo 资源操作

// BKRepoProjectAction 项目操作
var BKRepoProjectAction = struct{ View, Manage, Edit string }{
	// 项目查看
	View: "project_view",
	// 项目管理
	Manage: "project_manage",
	// 项目更新
	Edit: "project_edit",
}

// RepositoryAction 仓库操作
var RepositoryAction = struct{ Create, View, Manage, Edit, Delete string }{
	// 仓库创建
	Create: "repo_create",
	// 仓库查看
	View: "repo_view",
	// 仓库管理
	Manage: "repo_manage",
	// 仓库更新
	Edit: "repo_edit",
	// 仓库删除
	Delete: "repo_delete",
}

// RepoNodeAction 仓库节点操作
var RepoNodeAction = struct{ Create, Write, Edit, View, Download, Delete string }{
	// 节点创建
	Create: "node_create",
	// 节点编辑
	Write: "node_write",
	// 节点更新
	Edit: "node_edit",
	// 查看节点
	View: "node_view",
	// 下载节点
	Download: "node_download",
	// 节点删除
	Delete: "node_delete",
}
