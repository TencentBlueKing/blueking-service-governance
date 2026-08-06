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

import "context"

// Client 蓝盾制品库 API 客户端接口
type Client interface {
	// CreateUserToProject 创建用户（公共账号）并绑定为项目管理员
	CreateUserToProject(ctx context.Context, projectID, username, password string, associatedUsers []string) error
	// CreateProject 创建制品库项目
	CreateProject(ctx context.Context, projectID string) error
	// CreateRepository 创建制品库仓库
	// repoType 可选值：GENERIC, DOCKER, MAVEN, NPM, PYPI, HELM, RPM, COMPOSER
	CreateRepository(ctx context.Context, projectID, repoName, repoType, description string, isPublic bool) error
}
