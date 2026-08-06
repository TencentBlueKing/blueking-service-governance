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

// Package envvars 中的模型当前主要包含作用范围为空间级、环境类型级的公共类环境变量。
// TODO: 把目前由 env 模块管理的单一环境级环境变量也迁移到这里来统一管理。为什么必须要迁移？因为新的产品设计中
// 所有作用范围的环境变量会在同一个列表中展示，假如不迁移，整体的实现就无法统一，非常别扭。
package envvars

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ScopedEnvVar 作用域为公共类的环境变量。
// 当前需求上，同一变量在 (WorkspaceID, ScopeType, ScopeValue, Key) 维度上唯一。
type ScopedEnvVar struct {
	// ID 环境变量 ID
	ID bson.ObjectID `bson:"_id,omitempty"`

	// WorkspaceID 环境所属空间 ID
	WorkspaceID string `bson:"workspaceID"`

	// ScopeType 作用域类型，目前可选值为 workspace、envType、env
	ScopeType envvartypes.ScopeType `bson:"scopeType"`
	// ScopeValue 作用域值:
	// - 当 ScopeType 为 workspace 时，固定为空字符串
	// - 当 ScopeType 为 envType 时，可选值为 development、test、staging 或 production
	// - 当 ScopeType 为 env 时，值为具体的环境名称
	ScopeValue string `bson:"scopeValue"`

	// Key 环境变量名
	Key string `bson:"key"`
	// Value 环境变量值
	Value string `bson:"value"`
	// Description 描述
	Description string `bson:"description"`
	// IsBuiltin 是否内置，内置变量仅由系统写入，不允许用户修改
	IsBuiltin bool `bson:"isBuiltin"`
	// IsSensitive 是否敏感，敏感变量值不会以明文返回给前端/客户端
	IsSensitive bool `bson:"isSensitive"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
