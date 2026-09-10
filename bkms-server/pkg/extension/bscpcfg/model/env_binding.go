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

// EnvBinding 环境绑定配置
type EnvBinding struct {
	// AppID 所属 bkms 应用 ID
	// bscp app name = bkms appID
	AppID string `bson:"appID" validate:"required"`
	// EnvName 绑定的 bkms 环境名称（如 dev、prod）
	EnvName string `bson:"envName" validate:"required"`

	// BscpEnvID BSCP 环境 ID
	BscpEnvID string `bson:"bscpEnvID"`
	// BscpEnvName BSCP 环境名称
	BscpEnvName string `bson:"bscpEnvName"`
	// BscpAppID BSCP App ID
	BscpAppID string `bson:"bscpAppID"`

	// Operator 最近操作人
	Operator string `bson:"operator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
