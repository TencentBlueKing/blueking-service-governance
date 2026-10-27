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

// Package hostport manages app-level random HostPort port mappings for federated clusters.
package hostport

import "time"

// HostPortConfig is the app-level HostPort mapping document (1:1 with app).
type HostPortConfig struct {
	AppID string `bson:"appID"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID  string                      `json:"-" bson:"tenantID"`
	Ports     []int32                     `bson:"ports"`
	EnvStates map[string]HostPortEnvState `bson:"envStates"`
	CreatedAt time.Time                   `bson:"createdAt"`
	UpdatedAt time.Time                   `bson:"updatedAt"`
}

// HostPortEnvState records the last successfully deployed HostPort ports for one env.
type HostPortEnvState struct {
	AppliedPorts []int32   `bson:"appliedPorts"`
	UpdatedAt    time.Time `bson:"updatedAt"`
}

// EnvStateView is the computed pending-deploy view for one federated environment.
type EnvStateView struct {
	AppliedPorts       []int32
	PendingAddPorts    []int32
	PendingRemovePorts []int32
}

// HostPorts is the API-facing aggregate of declared ports and federated env pending state.
type HostPorts struct {
	Ports     []int32
	EnvStates map[string]EnvStateView
}
