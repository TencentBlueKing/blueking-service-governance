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

package depsvcredis

const (
	configKeyCreateTicketID  = "createTicketID"
	configKeyDisableTicketID = "disableTicketID"
	configKeyDestroyTicketID = "destroyTicketID"
	configKeyClusterID       = "clusterID"
	configKeyClusterName     = "clusterName"
	configKeyClusterType     = "clusterType"
	configKeyDomain          = "domain"
	configKeyPort            = "port"
	configKeyBkBizID         = "bkBizID"
)

// Redis 实例 Credentials 落库键，绑定 EnvVars 可通过 ${{env.KEY}} 引用。
const (
	CredHost = "REDIS_HOST"
	CredPort = "REDIS_PORT"
	CredPwd  = "REDIS_PWD"
)
