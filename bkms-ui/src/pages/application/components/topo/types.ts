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

import type { ResourceCategory } from './constants';
import type { TopologyNode } from '~/@types/topology';

export interface CategoryGroup {
  id: ResourceCategory;
  kinds: KindGroup[];
  label: string;
}

export interface ContextMenuEvent {
  action: string;
  nodeData: TopoNodeData;
  nodeId: string;
}

export interface ContextMenuItem {
  disabled?: boolean;
  id: string;
  label: string;
  tip?: string;
}

export interface KindGroup {
  kind: string;
  nodes: TopologyNode[];
}

export type NodeStatus = 'all' | 'error' | 'healthy' | 'unknown' | 'warning';

/** 侧栏状态统计与筛选的固定顺序：正常 → 异常 → 告警 → 未知 */
export const TOPO_NODE_STATUS_ORDER: NodeStatus[] = ['all', 'healthy', 'error', 'warning', 'unknown'];

export type StatusCounts = Record<NodeStatus, number>;

export interface TopoNodeData extends TopologyNode {
  /** 节点是否处于折叠状态 */
  collapsed: boolean;
  /** 节点是否有子节点 */
  hasChildren?: boolean;
  /** 归一化后的节点状态 */
  nodeStatus: NodeStatus;
}
