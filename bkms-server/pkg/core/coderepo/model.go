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

package coderepo

import "time"

// CodeRepository 工作空间的代码库
type CodeRepository struct {
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Alias 别名，工作空间内唯一，也是蓝盾代码库别名
	Alias string `bson:"alias"`
	// URL 仓库 HTTP(S) 地址
	URL string `bson:"url"`
	// SourceType 接入方式名
	SourceType string `bson:"sourceType"`
	// BasicAuth 仓库账号密码，仅 basic 方式有值
	BasicAuth *BasicAuth `bson:"basicAuth,omitempty"`
	// Creator 创建人
	Creator string `bson:"creator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// BasicAuth 仓库账号密码
type BasicAuth struct {
	// Username 仓库账号
	Username string `bson:"username"`
	// Password 仓库密码，加密存储
	Password string `bson:"password" audit:"mask"`
}
