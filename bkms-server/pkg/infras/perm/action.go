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

package perm

// BKMS 资源操作，对应蓝鲸 IAM 中的动作 ID。

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

// AppEnvAction 应用在环境中的操作，同时关联应用和环境两种资源。
//
// 同一条授权里应用和环境必须同时满足，因此不同角色的应用范围和环境范围不会拼接。
var AppEnvAction = struct{ View, Edit, Deploy string }{
	// 查看应用在环境中的数据
	View: "view_app_in_env",
	// 编辑应用在环境中的配置（不发布）
	Edit: "edit_app_in_env",
	// 部署应用到环境
	Deploy: "deploy_app_to_env",
}
