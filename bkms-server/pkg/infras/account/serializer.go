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

package account

// CurrentUserOutput 是获取当前用户接口的响应，结构不包裹在 data 字段中
type CurrentUserOutput struct {
	// 用户 ID
	UserID string `json:"userID"`
	// 用户所属租户，认证上游未返回租户时为空（如单租户环境）
	TenantID string `json:"tenantID"`
	// 用户时区（IANA 时区名），认证上游未返回时为 Asia/Shanghai
	TimeZone string `json:"timeZone"`
	// 用户管理个人中心地址，未配置用户管理地址时为空
	PersonalCenterURL string `json:"personalCenterURL"`
}

// LegacyErrorOutput 是 account 接口沿用的历史错误响应格式
type LegacyErrorOutput struct {
	// 错误信息
	Message string `json:"message"`
	// 错误码，固定为 "1000"
	Code string `json:"code"`
}
