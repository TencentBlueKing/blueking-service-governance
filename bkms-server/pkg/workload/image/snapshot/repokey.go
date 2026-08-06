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

package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
)

// GenerateRepoKey 根据仓库地址和凭证生成仓库实例唯一标识
// 使用 SHA256({registryAddress} + {username} + {password}) 的完整哈希值
func GenerateRepoKey(registryAddress, username, password string) string {
	data := registryAddress + username + password
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
