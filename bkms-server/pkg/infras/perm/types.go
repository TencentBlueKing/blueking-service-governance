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

// Package perm 提供 bkms-server 进程内的权限管理器入口。
//
// 本包是权限管理能力的业务侧入口（L3）：对外暴露 v2 风格的 Manager 接口，
// 方法签名完全使用 pkg/bkintegrations/bkiam 中的纯 Go DTO，不引用任何生成的 PB 模块。
//
// 调用链：
//
//	业务代码 -> perm.Manager (LocalManager) -> iam.IAMService -> cloudapi/iam.IAMClient -> 蓝鲸 IAM 网关
package perm

// ResourceType 蓝鲸 IAM 中的资源类型
type ResourceType string

const (
	// WorkspaceResourceType 资源类型 - 工作空间
	WorkspaceResourceType ResourceType = "workspace"
	// AppResourceType 资源类型 - 应用
	AppResourceType ResourceType = "app"
	// EnvResourceType 资源类型 - 环境
	EnvResourceType ResourceType = "env"
)
