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

package backends

// UserInfo 表示一个通过认证后的用户信息。
type UserInfo struct {
	// ID 为用户的唯一标记。
	ID string
	// TenantID 是认证上游返回的登录态所属租户，部分后端可能为空。
	TenantID string
}

// apigwUserInfoResponse 是 bk-login 网关 userinfo 接口的响应结构。
type apigwUserInfoResponse struct {
	Data struct {
		BkUsername  string `json:"bk_username"`
		TenantID    string `json:"tenant_id"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
