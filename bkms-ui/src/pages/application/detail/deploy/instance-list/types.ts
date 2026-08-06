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

import { type AppInstanceOutputObj } from '~/@types/v1/instance';

/** 实例操作上下文（各 Action 组件共享） */
export interface InstanceActionContext {
  timer?: { start: () => void; stop: () => void };
  clearSelections: () => void;
  getEnvName: (context?: { instance?: AppInstanceOutputObj }) => string;
  getSelectedInstances: () => AppInstanceOutputObj[];
  refreshData: () => Promise<void> | void;
}

/** 环境数据加载完成载荷 */
export interface InstanceDataLoadedPayload {
  envName: string;
  instances: AppInstanceOutputObj[];
  total: number;
}

/** 行操作类型 */
export type InstanceRowAction = 'gray' | 'log' | 'login' | 'monitor' | 'weight';

/** 行操作事件载荷 */
export interface InstanceRowActionPayload {
  action: InstanceRowAction;
  envName: string;
  instance: AppInstanceOutputObj;
}

/** 环境选中变化载荷 */
export interface InstanceSelectionChangePayload {
  envName: string;
  selections: AppInstanceOutputObj[];
}

/** 实例表格暴露方法 */
export interface InstanceTableExpose {
  isAllSelected?: boolean;
  isCollapsed?: boolean;
  isCrossPageSelection?: boolean;
  selectedCount?: number;
  clearSelections: () => void;
  getSelections: () => AppInstanceOutputObj[];
  getTotal: () => number;
  loadInstances: () => Promise<void>;
  resetPage: (current?: number) => void;
}

/** 实例表格模式 */
export type InstanceTableMode = 'multiEnv' | 'single';
