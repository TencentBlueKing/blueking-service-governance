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

package bkci

import "time"

// Project 蓝盾项目
type Project struct {
	// ID 项目 UID，格式为 32 位字符串
	ID string `bson:"id"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// Code 项目 Code，格式为可读唯一字符串，如：bkms
	Code string `bson:"code"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Creator 项目创建人
	Creator string `bson:"creator"`
	// CreatedAt 项目创建时间
	CreatedAt time.Time `bson:"createdAt"`
}

// PipelineTemplate 流水线模板
type PipelineTemplate struct {
	// ID 模板 UID，格式如：0348a1df-44de-29b2-1c94-2d42841c009d
	ID string `bson:"id"`
	// Type 模板类型，全局唯一
	Type string `bson:"type"`
	// Version 模板版本，格式为 semver
	Version string `bson:"version"`
	// Name 模板名称（会作为初始化出的流水线名称）
	Name string `bson:"name"`
	// Description 模板描述（会作为初始化出的流水线描述）
	Description string `bson:"description"`
	// Stages 模板阶段配置
	Stages []map[string]any `bson:"stages"`
}

// Pipeline 流水线
type Pipeline struct {
	// ID 流水线 UID，格式类似于：p-5df30e9fe868af903dff8d375dd7b463
	ID string `bson:"id"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// Type 流水线类型，同工作空间下唯一
	Type string `bson:"type"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// ProjectCode 蓝盾项目 Code
	ProjectCode string `bson:"projectCode"`
	// Name 流水线名称
	Name string `bson:"name"`
	// Description 流水线描述
	Description string `bson:"description"`
	// TemplateVersion 已应用的内置流水线模板版本，格式为 semver
	TemplateVersion string `bson:"templateVersion,omitempty"`
	// CallbackCredentialID 触发专用流水线关联的蓝盾回调凭证 ID；仅 build-trigger-* 类型有值
	CallbackCredentialID string `bson:"callbackCredentialID,omitempty"`
	// Creator 流水线创建人
	Creator string `bson:"creator"`
	// CreatedAt 流水线创建时间
	CreatedAt time.Time `bson:"createdAt"`
}

// Repository 代码库在蓝盾侧的同步记录
type Repository struct {
	// ID 仓库 Hash ID，格式类似于：Zr3Dx
	ID string `bson:"id"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// Alias 仓库别名，格式类似于：bkms
	Alias string `bson:"alias"`
	// URL 仓库地址，格式类似于：https://git.example.com/bkms/bkms.git
	URL string `bson:"url"`
	// Type 蓝盾返回的代码库类型
	Type string `bson:"type"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// ProjectCode 蓝盾项目 Code
	ProjectCode string `bson:"projectCode"`
	// CredentialID 蓝盾凭证 ID，OAuth 关联时为空
	CredentialID string `bson:"credentialID,omitempty"`
	// Creator 仓库创建人
	Creator string `bson:"creator"`
	// CreatedAt 仓库创建时间
	CreatedAt time.Time `bson:"createdAt"`
}

// BuildNodeBinding 工作空间关联的蓝盾第三方构建机，一个工作空间最多一条
type BuildNodeBinding struct {
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// ProjectCode 构建机所在的蓝盾项目 Code
	ProjectCode string `bson:"projectCode"`
	// NodeHashID 蓝盾节点 HashId
	NodeHashID string `bson:"nodeHashID"`
	// AgentHashID 蓝盾 Agent HashId，下发流水线调度第三方构建机时使用
	AgentHashID string `bson:"agentHashID"`
	// DisplayName 关联时的节点名称快照，节点别名为空时取主机名
	DisplayName string `bson:"displayName"`
	// IP 关联时的节点 IP 快照
	IP string `bson:"ip"`
	// OSName 关联时的操作系统快照
	OSName string `bson:"osName"`
	// Updater 最后操作人
	Updater string `bson:"updater"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
