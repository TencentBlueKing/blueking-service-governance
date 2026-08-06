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

package serializer

// --- BCS URI 参数 ---

// BCSProjectURIInput 根据项目 ID 获取项目详情
type BCSProjectURIInput struct {
	ProjectID string `uri:"projectID" binding:"required,len=32"`
}

// BCSProjectClustersURIInput 获取项目下的集群列表
type BCSProjectClustersURIInput struct {
	ProjectID string `uri:"projectID" binding:"required,len=32"`
}

// BCSClusterNamespacesURIInput 获取集群下的命名空间列表
type BCSClusterNamespacesURIInput struct {
	ProjectID string `uri:"projectID" binding:"required,len=32"`
	ClusterID string `uri:"clusterID" binding:"required,min=13"`
}

// --- BCS Output ---

// BCSProjectOutput BCS 项目输出
type BCSProjectOutput struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Code             string `json:"code"`
	Description      string `json:"description"`
	BizID            string `json:"bizID"`
	IsOffline        bool   `json:"isOffline"`
	IsBoundWorkspace bool   `json:"isBoundWorkspace"`
}

// ListBCSAuthorizedProjectsOutput 获取有权限的 BCS 项目列表的响应
type ListBCSAuthorizedProjectsOutput struct {
	Data []*BCSProjectOutput `json:"data"`
}

// GetBCSProjectOutput 获取 BCS 项目详情的响应
type GetBCSProjectOutput struct {
	Data *BCSProjectOutput `json:"data"`
}

// ClusterOutput 集群输出
type ClusterOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Environment string `json:"environment"`
	IsShared    bool   `json:"isShared"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ListClustersByProjectOutput 获取项目下集群列表的响应
type ListClustersByProjectOutput struct {
	Data []*ClusterOutput `json:"data"`
}

// NamespaceOutput 命名空间输出
type NamespaceOutput struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ListNamespacesByClusterOutput 获取集群下命名空间列表的响应
type ListNamespacesByClusterOutput struct {
	Data []*NamespaceOutput `json:"data"`
}
