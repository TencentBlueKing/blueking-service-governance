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

package portforward

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Store port-forward 白名单存储
type Store = store.GenericStore[WhitelistEntry]

// Meta port-forward 白名单的存储元数据，每个被允许的环境 ID 对应集合中的一条独立文档（_id = envID）
var Meta = store.Meta[WhitelistEntry]{Collection: "plat_admin_port_forward_whitelist"}

// ByEnvIDs 按环境 ID 列表匹配白名单记录
func ByEnvIDs(envIDs []string) store.Opt {
	return store.Where("_id", bson.M{"$in": envIDs})
}
