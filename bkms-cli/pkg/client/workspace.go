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

package client

// Workspace 工作空间
type Workspace struct {
	ID          string `json:"id" yaml:"id"`
	DisplayName string `json:"displayName" yaml:"displayName"`
	Description string `json:"description" yaml:"description"`
	State       string `json:"state" yaml:"state"`
	Creator     string `json:"creator" yaml:"creator"`
}

// ListWorkspacesRespData 获取工作空间列表返回数据
type ListWorkspacesRespData struct {
	Data []Workspace `json:"data"`
}

// GetWorkspaceRespData 获取工作空间详情返回数据
type GetWorkspaceRespData struct {
	Data Workspace `json:"data"`
}
