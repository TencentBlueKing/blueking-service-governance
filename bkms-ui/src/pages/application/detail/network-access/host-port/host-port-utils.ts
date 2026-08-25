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

import type { EnvOutput } from '~/@types/v1/env';
import type { HostPortEnvStateOutput } from '~/@types/v1/hostport';

/** key 为环境名；仅包含联邦环境（与 HostPortsOutput.envStates 一致） */
export type HostPortEnvStates = Record<string, HostPortEnvStateOutput>;

/** 生效范围展示项：集合来自 envStates key，展示信息来自 listAppEnvs */
export interface ScopeEnv {
  /** 展示名：优先 listAppEnvs.displayName，否则回退环境名 */
  displayName: string;
  /** 环境名（envStates 的 key） */
  name: string;
  /** 环境分类，用于展示类型 Tag */
  type?: string;
}

/** 环境分类展示顺序，与环境列表、组件可用环境保持一致 */
const ENV_TYPE_ORDER = ['development', 'test', 'staging', 'production'];

/** 由 listAppEnvs 构建 name → EnvOutput，供生效范围补齐 displayName / type */
export function buildEnvInfoMap(envs: EnvOutput[]) {
  const map = new Map<string, EnvOutput>();
  envs.forEach(env => {
    if (env.name) map.set(env.name, env);
  });
  return map;
}

/** 按环境类型顺序 + 展示名，对生效范围列表排序 */
export function compareScopeEnv(a: ScopeEnv, b: ScopeEnv) {
  const orderDiff = getEnvTypeWeight(a.type) - getEnvTypeWeight(b.type);
  if (orderDiff !== 0) return orderDiff;
  return a.displayName.localeCompare(b.displayName);
}

/** 统计有待部署变更（待新增或待删除端口非空）的环境数；环境集合取自 envStates 的 key */
export function countPendingEnvs(envStates: HostPortEnvStates) {
  return Object.values(envStates).filter(state => hasEnvPendingChange(state)).length;
}

/** 环境类型排序权重；未知类型排在标准类型之后 */
export function getEnvTypeWeight(type?: string) {
  if (!type) return ENV_TYPE_ORDER.length;
  const index = ENV_TYPE_ORDER.indexOf(type);
  return index === -1 ? ENV_TYPE_ORDER.length : index;
}

/** 是否「待部署」：pendingAddPorts 或 pendingRemovePorts 非空 */
export function hasEnvPendingChange(state?: HostPortEnvStateOutput | null) {
  if (!state) return false;
  return (state.pendingAddPorts?.length ?? 0) > 0 || (state.pendingRemovePorts?.length ?? 0) > 0;
}

/** 草稿相对已保存声明是否有增删（顺序无关；与服务端升序去重语义一致） */
export function hasPortListChanged(savedPorts: number[], draftPorts: number[]) {
  const savedSet = new Set(savedPorts);
  if (savedSet.size !== draftPorts.length) return true;
  return draftPorts.some(port => !savedSet.has(port));
}

/**
 * BCS 随机 HostPort webhook 注入到容器的环境变量命名。
 * 非纯数字端口推导不出变量名，返回空串。
 */
export function hostPortEnvName(port: string) {
  return parseContainerPort(port) != null ? `BCS_RANDHOSTPORT_FOR_CONTAINER_PORT_${port.trim()}` : '';
}

/**
 * 解析容器端口：trim 后须为 1–65535 的纯数字，否则返回 null。
 * 校验规则与环境变量名推导共用。
 */
export function parseContainerPort(value: unknown): null | number {
  const text = String(value ?? '').trim();
  if (!/^\d+$/.test(text)) return null;
  const port = Number(text);
  return port >= 1 && port <= 65535 ? port : null;
}
