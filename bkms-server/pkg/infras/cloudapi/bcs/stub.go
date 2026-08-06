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

// Package bcs provides api client to bcs（蓝鲸容器服务）
package bcs

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// stubProjects 本地开发时返回的固定项目列表
var stubProjects = []Project{
	{
		ID:          "stub0001stub0001stub0001stub0001",
		Code:        "stub-project-a",
		Name:        "Stub 项目 A",
		Description: "本地开发用 stub 项目 A",
		Kind:        "k8s",
		BizID:       "100001",
		IsOffline:   false,
	},
	{
		ID:          "stub0002stub0002stub0002stub0002",
		Code:        "stub-project-b",
		Name:        "Stub 项目 B",
		Description: "本地开发用 stub 项目 B",
		Kind:        "k8s",
		BizID:       "100002",
		IsOffline:   false,
	},
}

// stubClusters 本地开发时返回的固定集群列表
var stubClusters = []Cluster{
	{
		ID:          "XXX-K8S-00001",
		Name:        "stub-cluster-default",
		Type:        "single",
		Environment: "test",
		IsShared:    false,
		Description: "本地开发用 stub 集群",
		Status:      "RUNNING",
	},
}

// stubNamespaces 本地开发时返回的固定命名空间列表
var stubNamespaces = []Namespace{
	{Name: "default", Status: "Active"},
	{Name: "kube-system", Status: "Active"},
}

// StubApiClient 测试用的 BCS API 客户端实现，返回模拟数据
type StubApiClient struct {
	user auth.User
}

// NewStub 创建 StubApiClient
func NewStub(user auth.User) *StubApiClient {
	return &StubApiClient{user: user}
}

// ListAuthorizedProjects 模拟获取有权限的项目列表，返回 stubProjects
func (s *StubApiClient) ListAuthorizedProjects(ctx context.Context) ([]Project, error) {
	log.Infof(ctx, "Stub: ListAuthorizedProjects request: user=%s", s.user.ID)
	return stubProjects, nil
}

// GetProject 模拟根据项目 id 获取项目详情
func (s *StubApiClient) GetProject(ctx context.Context, id string) (*Project, error) {
	log.Infof(ctx, "Stub: GetProject request: id=%s", id)
	for i := range stubProjects {
		if stubProjects[i].ID == id || stubProjects[i].Code == id {
			p := stubProjects[i]
			return &p, nil
		}
	}

	// 默认返回第一个 stub 项目
	return &Project{
		ID:          "stub0001stub0001stub0001stub0001",
		Code:        id,
		Name:        fmt.Sprintf("Stub 项目 (%s)", id),
		Description: "本地开发用 stub 默认项目",
		Kind:        "k8s",
		BizID:       "100001",
		IsOffline:   false,
	}, nil
}

// ListClustersByProject 模拟获取项目下的集群列表，返回 stubClusters
func (s *StubApiClient) ListClustersByProject(ctx context.Context, projectID string) ([]Cluster, error) {
	log.Infof(ctx, "Stub: ListClustersByProject request: projectID=%s", projectID)
	return stubClusters, nil
}

// ListNamespacesByCluster 模拟获取集群下的命名空间列表，返回 stubNamespaces
func (s *StubApiClient) ListNamespacesByCluster(
	ctx context.Context,
	projectID, clusterID string,
) ([]Namespace, error) {
	log.Infof(ctx, "Stub: ListNamespacesByCluster request: projectID=%s, clusterID=%s", projectID, clusterID)

	return stubNamespaces, nil
}

// CreateWebConsole 模拟创建 web console 会话，返回固定的 URL
func (s *StubApiClient) CreateWebConsole(
	ctx context.Context, projectID, clusterID, namespace, podName, containerName, command string,
) (string, error) {
	log.Infof(
		ctx, "Stub: CreateWebConsole request: projectID=%s, clusterID=%s, namespace=%s, podName=%s, containerName=%s",
		projectID, clusterID, namespace, podName, containerName,
	)
	return fmt.Sprintf(
		"https://stub-bcs.example.com/web-console?project=%s&cluster=%s&ns=%s&pod=%s",
		projectID, clusterID, namespace, podName,
	), nil
}
