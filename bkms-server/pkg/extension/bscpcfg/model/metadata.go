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

// Package model 定义了应用配置管理相关的纯数据模型。
package model

import (
	"time"
)

// Metadata 应用配置管理元信息（全局共享，一个 App 一条记录）。
type Metadata struct {
	// AppID bkms 应用 ID（唯一键，一个 App 只有一条记录）
	AppID string `bson:"appID" validate:"required"`
	// BscpBizID BSCP 业务 ID
	BscpBizID string `bson:"bscpBizID" validate:"required"`
	// WorkloadName 指定被注入 bscp 配置的目标 workload 名称
	WorkloadName string `bson:"workloadName"`
	// WorkloadKind 目标工作负载类型
	WorkloadKind string `bson:"workloadKind"`
	// MountPath 配置文件在容器中的挂载路径（所有环境共享同一路径）
	MountPath string `bson:"mountPath"`

	// CredentialID BSCP Credential ID（每个业务下唯一，名称固定为 bkms-credential）
	CredentialID string `bson:"credentialID"`
	// CredentialName BSCP Credential 名称
	CredentialName string `bson:"credentialName"`
	// Token BSCP Credential 的访问令牌（用于 sidecar 拉取配置）
	Token string `bson:"token"`
	// FeedAddr BSCP 服务订阅地址（sidecar 连接的 feed server 地址）
	FeedAddr string `bson:"feedAddr"`
	// PostHookID BSCP 后置脚本 ID
	PostHookID string `bson:"postHookID,omitempty"`

	// Operator 最近操作人
	Operator string `bson:"operator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// MetadataUpdate 定义了更新 Metadata 时允许修改的数据
type MetadataUpdate struct {
	// MountPath 更新挂载路径（nil 表示不更新）
	MountPath *string
	// CredentialID 更新 credential ID（nil 表示不更新）
	CredentialID *string
	// CredentialName 更新 credential 名称（nil 表示不更新）
	CredentialName *string
	// Token 更新 token（nil 表示不更新）
	Token *string
	// WorkloadName 更新目标 workload 名称（nil 表示不更新）
	WorkloadName *string
	// WorkloadKind 更新目标工作负载类型（nil 表示不更新）
	WorkloadKind *string
}

// ApplyTo 将更新数据应用到 Metadata 对象上
func (u *MetadataUpdate) ApplyTo(m *Metadata) {
	if u == nil || m == nil {
		return
	}
	if u.MountPath != nil {
		m.MountPath = *u.MountPath
	}
	if u.WorkloadName != nil {
		m.WorkloadName = *u.WorkloadName
	}
	if u.CredentialID != nil {
		m.CredentialID = *u.CredentialID
	}
	if u.CredentialName != nil {
		m.CredentialName = *u.CredentialName
	}
	if u.Token != nil {
		m.Token = *u.Token
	}
	if u.WorkloadKind != nil {
		m.WorkloadKind = *u.WorkloadKind
	}
}
