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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// EnvInfo 环境关联信息
type EnvInfo struct {
	EnvID bson.ObjectID `bson:"envID"`

	EnvName string `bson:"envName"`
}

// ApmInstConfig 记录蓝鲸监控 APM 实例的配置信息，
// 包括实例标识（ApmID、Name）、访问凭证（Token）以及关联的环境列表。
type ApmInstConfig struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// WorkspaceID 工作空间ID
	WorkspaceID string `bson:"workspaceID" validate:"required"`
	// ApmID APM ID
	ApmID int64 `bson:"apmID" validate:"required"`
	// Name APM名称
	Name string `bson:"name" validate:"required"`
	// Token APM token
	Token string `bson:"token" validate:"required"`
	// AssociatedEnvs 关联环境
	AssociatedEnvs []EnvInfo `bson:"associatedEnvs"`

	// Creator 创建人
	Creator string `bson:"creator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
