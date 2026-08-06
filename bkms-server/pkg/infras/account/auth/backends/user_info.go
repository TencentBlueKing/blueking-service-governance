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

import "github.com/pkg/errors"

func parseUserInfo(url string, respData map[string]any) (*UserInfo, error) {
	data, ok := respData["data"].(map[string]any)
	if !ok {
		return nil, errors.Errorf("failed to get user info from %s, response data not json format", url)
	}
	username, ok := data["username"].(string)
	if !ok || username == "" {
		return nil, errors.Errorf("failed to get user info from %s, username not found", url)
	}
	return &UserInfo{ID: username}, nil
}
