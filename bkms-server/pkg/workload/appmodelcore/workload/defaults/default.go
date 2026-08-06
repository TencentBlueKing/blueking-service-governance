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

package defaults

// UpdateStrategy 的默认值
const (
	// 与 GameDeployment 默认值保持一致

	// MaxUnavailable 默认最大不可用副本数量
	MaxUnavailable = "25%"
	// MaxSurge 默认最大超出所需副本的数量
	MaxSurge = "25%"
)

// PodDeletionCost 默认的 pod 删除成本，按 GameDeployment 约定为 1024
const PodDeletionCost = 1024

// WorkloadMainContainerName 默认的工作负载主容器名称
const WorkloadMainContainerName = "main"
