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

// Package trafficmanager provides functionality for managing traffic in the BKMS system.
// FIXME 2026.6 trafficmanager 切换为空实现，未来重新支持泳道功能后会评估是否直接合并到 bkms-server 中
package trafficmanager

import "context"

// TrafficManager 管理 bkms-server 内部泳道信息。
type TrafficManager interface {
	ListTrafficLanes(ctx context.Context, workspaceID, envName string) ([]*TrafficLane, error)
	GetBaselineTrafficLane(ctx context.Context, workspaceID, envName string) (*TrafficLane, error)
	GetTrafficLane(ctx context.Context, workspaceID, envName, name string) (*TrafficLane, error)
}

// New 创建本地空实现。
func New() TrafficManager {
	return &StubTrafficManager{}
}
