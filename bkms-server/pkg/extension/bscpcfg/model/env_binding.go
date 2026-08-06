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

// ServiceRef 绑定的下发服务引用项
type ServiceRef struct {
	// ID 下发服务 ID
	ID string `bson:"id" json:"id"`

	// Name 下发服务名称（如 bkms-order-svc-dev）
	Name string `bson:"name" json:"name"`
}

// EnvBinding 环境绑定配置（一个 app+env 一条记录）。
type EnvBinding struct {
	// AppID 所属 bkms 应用 ID（关联 Metadata.AppID）
	AppID string `bson:"appID" validate:"required"`
	// EnvName 绑定的环境名称（如 dev、prod）
	EnvName string `bson:"envName" validate:"required"`
	// Services 绑定的下发服务列表
	// ps: 至少包含默认服务，可额外绑定公共服务
	Services []ServiceRef `bson:"bscpApps" validate:"required,min=1"`
	// DefaultServiceID 创建时自动生成的默认 file Service ID
	// ps: 不可移除，更新 Services 时必须包含此 ID
	DefaultServiceID string `bson:"defaultBscpAppID"`

	// Operator 最近操作人
	Operator string `bson:"operator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// EnvBindingUpdate 定义了更新 EnvBinding 时允许修改的数据
type EnvBindingUpdate struct {
	// Services 更新绑定的下发服务列表
	// ps: nil 表示不更新，非 nil 时全量替换，但必须包含 DefaultServiceID
	Services *[]ServiceRef
}
