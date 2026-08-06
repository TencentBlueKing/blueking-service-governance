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

package types

// CreateGradeManagerReq 创建分级管理员的请求参数
type CreateGradeManagerReq struct {
	System              string               `json:"system"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Members             []string             `json:"members"`
	AuthorizationScopes []AuthorizationScope `json:"authorization_scopes"`
	SubjectScopes       []SubjectScope       `json:"subject_scopes"`
}

// UpdateGradeManagerReq 更新分级管理员的请求参数
type UpdateGradeManagerReq struct {
	System              string               `json:"system"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	AuthorizationScopes []AuthorizationScope `json:"authorization_scopes"`
	SubjectScopes       []SubjectScope       `json:"subject_scopes"`
}

// UserGroupParam 待创建的用户组参数
type UserGroupParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Readonly 用户组是否只读. 设置为 True 后，分级管理员无法在权限中心产品上删除该用户组
	Readonly bool `json:"readonly"`
}

// UserMemberParam 待新增的用户成员参数
type UserMemberParam struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// AddUserGroupMembersReq 添加用户组成员请求参数
type AddUserGroupMembersReq struct {
	Members   []UserMemberParam `json:"members"`
	ExpiredAt int               `json:"expired_at"`
}
