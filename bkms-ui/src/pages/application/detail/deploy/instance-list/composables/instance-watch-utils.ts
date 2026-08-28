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

import type { InstanceWatchEvent } from '../types';
import type { AppInstanceOutputObj } from '~/@types/v1/instance';

export type RestartSortOrder = 'asc' | 'desc' | null;

interface ParsedSseBlock {
  event?: InstanceWatchEvent;
  heartbeat: boolean;
}

/** 从当前缓冲区中提取完整 SSE 事件块，并保留未结束的尾部。 */
export function extractSseEventBlocks(buffer: string) {
  const blocks: string[] = [];
  let rest = buffer;
  let boundaryIndex = rest.search(/\r?\n\r?\n/);

  while (boundaryIndex >= 0) {
    blocks.push(rest.slice(0, boundaryIndex));
    const separator = rest.slice(boundaryIndex).match(/^\r?\n\r?\n/)?.[0] || '';
    rest = rest.slice(boundaryIndex + separator.length);
    boundaryIndex = rest.search(/\r?\n\r?\n/);
  }

  return { blocks, rest };
}

/** 从已经筛选、排序的全量实例集合中截取当前页。 */
export function paginateInstances(instances: AppInstanceOutputObj[], current: number, limit: number) {
  const start = (current - 1) * limit;
  return instances.slice(start, start + limit);
}

/** 解析一个完整 SSE 事件块；注释行只作为心跳，不参与业务事件解析。 */
export function parseInstanceSseBlock(eventBlock: string): ParsedSseBlock {
  const dataLines: string[] = [];
  let heartbeat = false;

  for (const line of eventBlock.split(/\r?\n/)) {
    if (line.startsWith(':')) {
      heartbeat = true;
    } else if (line.startsWith('data:')) {
      // SSE 允许一条事件包含多行 data，拼接后再统一解析 JSON。
      dataLines.push(line.slice('data:'.length).trimStart());
    }
  }

  if (dataLines.length === 0) return { heartbeat };

  const parsed = JSON.parse(dataLines.join('\n')) as InstanceWatchEvent;
  if (!parsed || typeof parsed !== 'object' || typeof parsed.type !== 'string') {
    throw new Error('invalid instance watch event');
  }

  return { event: parsed, heartbeat };
}

/** 将单条 Watch 事件归并到实例快照中。 */
export function reduceInstanceWatchEvent(
  instances: AppInstanceOutputObj[],
  event: InstanceWatchEvent,
): AppInstanceOutputObj[] {
  if (event.type === 'PLUGIN') {
    // 插件事件不能创建实例；当前只认识 polaris，真实空数组也必须覆盖旧值。
    if (event.plugin !== 'polaris' || !event.object?.id || !Array.isArray(event.object.data)) return instances;
    const index = instances.findIndex(item => item.id === event.object?.id);
    if (index < 0) return instances;

    const next = [...instances];
    next[index] = {
      ...next[index],
      polarisInfos: event.object.data,
    };
    return next;
  }

  // ENDED 只负责驱动连接生命周期，不改变页面已有快照。
  if (event.type === 'ENDED') return instances;

  const instanceID = event.object?.id;
  if (!instanceID) return instances;

  if (event.type === 'DELETED') {
    // 服务端对 DELETED 只保证 object.id，因此删除逻辑不能读取其他字段。
    return instances.filter(item => item.id !== instanceID);
  }

  const index = instances.findIndex(item => item.id === instanceID);
  if (event.type === 'ADDED') {
    if (index < 0) {
      // Pod 事件的 polarisInfos 不可信，新行等待后续 PLUGIN 事件补齐北极星数据。
      return [...instances, { ...event.object, polarisInfos: [] }];
    }

    // 重复 ADDED 按防御性更新处理，但保留已经由 PLUGIN 写入的北极星信息。
    const next = [...instances];
    next[index] = {
      ...next[index],
      ...event.object,
      polarisInfos: next[index].polarisInfos || [],
    };
    return next;
  }

  if (event.type === 'MODIFIED' && index >= 0) {
    // MODIFIED 只更新 K8s 投影，禁止用事件中的空 polarisInfos 覆盖插件数据。
    const next = [...instances];
    next[index] = {
      ...next[index],
      ...event.object,
      polarisInfos: next[index].polarisInfos || [],
    };
    return next;
  }

  return instances;
}

/** 在分页前对全量实例按重启次数做稳定排序，缺失值始终置后。 */
export function sortInstancesByRestart(instances: AppInstanceOutputObj[], order: RestartSortOrder) {
  if (!order) return instances;

  const orderFactor = order === 'asc' ? 1 : -1;
  return instances
    .map((item, index) => ({ index, item }))
    .sort((a, b) => {
      const valueA = Number(a.item.restartCount);
      const valueB = Number(b.item.restartCount);
      const validA = Number.isFinite(valueA);
      const validB = Number.isFinite(valueB);
      if (!validA) return validB ? 1 : a.index - b.index;
      if (!validB) return -1;
      const result = (valueA - valueB) * orderFactor;
      // 重启次数相同时回退到原始下标，保证 Watch 更新前后排序稳定。
      return result === 0 ? a.index - b.index : result;
    })
    .map(item => item.item);
}
