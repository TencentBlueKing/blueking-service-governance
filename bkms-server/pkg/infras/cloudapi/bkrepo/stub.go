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

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// StubApiClient 测试用的蓝盾制品库 API 客户端实现，返回模拟数据
type StubApiClient struct {
	operator string
}

// NewStub 创建 StubApiClient
func NewStub(operator string) *StubApiClient {
	return &StubApiClient{operator: operator}
}

// ------------------------------------------ 蓝盾制品库用户管理 API ------------------------------------------

// CreateUserToProject 模拟创建用户并绑定为项目管理员，总是返回成功
func (s *StubApiClient) CreateUserToProject(
	ctx context.Context, projectID, username, _ string, _ []string,
) error {
	log.Infof(ctx, "Stub: CreateUserToProject request: projectID=%s, username=%s", projectID, username)
	return nil
}

// ------------------------------------------ 蓝盾制品库项目管理 API ------------------------------------------

// CreateProject 模拟创建制品库项目，总是返回成功
func (s *StubApiClient) CreateProject(ctx context.Context, projectID string) error {
	log.Infof(ctx, "Stub: CreateProject request: %s", projectID)
	return nil
}

// ------------------------------------------ 蓝盾制品库仓库管理 API ------------------------------------------

// CreateRepository 模拟创建制品库仓库，总是返回成功
func (s *StubApiClient) CreateRepository(
	ctx context.Context, projectID, repoName, repoType, _ string, isPublic bool,
) error {
	log.Infof(
		ctx, "Stub: CreateRepository request: projectID=%s, repoName=%s, repoType=%s, isPublic=%v",
		projectID, repoName, repoType, isPublic,
	)
	return nil
}
