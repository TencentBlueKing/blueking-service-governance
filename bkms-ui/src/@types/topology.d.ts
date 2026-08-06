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

import {
  type EdgeReasonOutputObj,
  type ResourceTopologyDataOutputObj,
  type TopologyEdgeOutputObj,
  type TopologyNodeDetailOutputObj,
  type TopologyNodeEventOutputObj,
  type TopologyNodeOutputObj,
} from './v1/topology';

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type EdgeReason = EdgeReasonOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ResourceTopologyData = ResourceTopologyDataOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type TopologyEdge = TopologyEdgeOutputObj;

/** 拓扑相关 (占位) */
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type TopologyNode = TopologyNodeOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type TopologyNodeDetail = TopologyNodeDetailOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type TopologyNodeEvent = TopologyNodeEventOutputObj;
