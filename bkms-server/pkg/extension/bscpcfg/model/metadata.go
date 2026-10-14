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

// Metadata 应用配置管理元信息（全局共享，一个 App 一条记录）
type Metadata struct {
	// AppID bkms 应用 ID（唯一键，一个 App 只有一条记录）
	AppID string `bson:"appID" validate:"required"`
	// PostHookID BSCP 后置脚本 ID
	PostHookID string `bson:"postHookID,omitempty"`

	// WorkloadKind 目标工作负载类型
	WorkloadKind string `bson:"workloadKind"`
	// WorkloadName 指定被注入 bscp 配置的目标 workload 名称
	WorkloadName string `bson:"workloadName"`
	// MountPath 配置文件在容器中的挂载路径（所有环境共享同一路径）
	MountPath string `bson:"mountPath"`

	// Enable 是否启用 BSCP 配置管理
	// 查询不到 Metadata 或为 false 表示未开启
	Enable bool `bson:"enable"`
	// Operator 最近操作人
	Operator string `bson:"operator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
