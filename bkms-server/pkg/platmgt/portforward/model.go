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

// Package portforward 提供 port-forward 白名单管理能力，作为 platmgt 模块的一部分。
package portforward

// WhitelistEntry 定义 port-forward 白名单中的一条记录。
// 每条记录代表一个被允许 port-forward 的非正式环境。
type WhitelistEntry struct {
	// EnvID 仅允许非正式环境 ID（同时作为文档 _id）。
	EnvID string `bson:"_id" json:"envID"`
}
