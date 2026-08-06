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

// Resp IAM 网关标准返回
type Resp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// CreateUserGroupsResp 创建用户组返回结果
type CreateUserGroupsResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []int  `json:"data"`
}

// GradeManager 分级管理员
type GradeManager struct {
	ID          int `mapstructure:"id"`
	Name        string
	Description string
}

// GradeManagerData 分级管理员查询结果数据
type GradeManagerData struct {
	Count   int            `mapstructure:"count"`
	Results []GradeManager `mapstructure:"results"`
}

// UserGroup 用户组
type UserGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Readonly 用户组是否只读. 设置为 True 后，分级管理员无法在权限中心产品上删除该用户组
	Readonly bool `json:"readonly"`
}

// UserMember 用户组成员
type UserMember struct {
	Type      string `mapstructure:"type"`
	ID        string `mapstructure:"id"`
	ExpiredAt int    `mapstructure:"expired_at"`
}

// UserMemberData 用户组成员查询结果数据
type UserMemberData struct {
	Count   int          `mapstructure:"count"`
	Results []UserMember `mapstructure:"results"`
}
