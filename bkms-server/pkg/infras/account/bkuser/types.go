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

package bkuser

const (
	bkUserGatewayName = "bk-user"

	// UserStatusEnabled means the bk-user account is active for tenant access.
	UserStatusEnabled = "enabled"
)

// User is the subset of bk-user user fields currently consumed by BKMS.
type User struct {
	TenantID    string `json:"tenant_id"`
	BkUsername  string `json:"bk_username"`
	LoginName   string `json:"login_name"`
	DisplayName string `json:"display_name"`
	TimeZone    string `json:"time_zone"`
	Language    string `json:"language"`
	Status      string `json:"status"`
}

type getUserResponse struct {
	Data  User `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
