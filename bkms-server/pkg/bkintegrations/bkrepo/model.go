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

package bkrepo

import "time"

// Project 蓝盾制品库项目
type Project struct {
	// ID 项目 UID，内容与蓝盾项目的 Code 相同，可读 & 唯一，如：bkms
	ID string `bson:"id"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Username 公共账号用户名
	Username string `bson:"username"`
	// Password 公共账号密码（加密存储）
	Password string `bson:"password"`
	// Creator 项目创建人
	Creator string `bson:"creator"`
	// CreatedAt 项目创建时间
	CreatedAt time.Time `bson:"createdAt"`
}

// Repository 蓝盾制品库仓库（如 Docker 镜像、HELM 仓库等）
// 注：同一项目下仓库访问凭证是相同的，存储在 Project 中
type Repository struct {
	// ProjectID 项目 ID
	ProjectID string `bson:"projectID"`
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Name 仓库名称
	Name string `bson:"name"`
	// Type 仓库类型，目前可选值有：DOCKER, HELM
	Type RepoType `bson:"type"`
	// IsPublic 仓库是否公开（即可任意人访问）
	IsPublic bool `bson:"isPublic"`
	// Creator 仓库创建人
	Creator string `bson:"creator"`
	// CreatedAt 仓库创建时间
	CreatedAt time.Time `bson:"createdAt"`
}
