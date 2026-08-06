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

import { type RouteRecordName } from 'vue-router';

import type { Config } from './interceptors';

export interface IQueue {
  config?: Config;
  controller: AbortController;
  id: string;
  request: Promise<unknown>;
  routeName?: RouteName;
}

export type RouteName = null | RouteRecordName | undefined;

// 请求队列
const requestQueue: Array<IQueue> = [];
// 添加队列
function addQueue(data: IQueue) {
  const index = requestQueue.findIndex(q => q.id === data?.id);
  if (index === -1) {
    requestQueue.push(data);
  }
}
// 取消队列请求
async function cancelRequest(id?: string | string[]) {
  let queues = requestQueue.filter(queue => !queue.config?.irrevocable); // 过滤配置了不可取消请求的配置
  if (id?.length) {
    const ids = Array.isArray(id) ? id : [id];
    queues = queues.filter(queue => ids.includes(queue.id));
  }
  queues.forEach(queue => queue.controller?.abort());
  await Promise.all(queues.map(item => item.request)).catch(() => {});
  clearQueue(id);
}
// 清空队列（不取消请求）
function clearQueue(id?: string | string[]) {
  if (id?.length) {
    const ids = Array.isArray(id) ? id : [id];
    ids.forEach(id => {
      const index = requestQueue.findIndex(queue => queue.id === id);
      index > -1 && requestQueue.splice(index, 1);
    });
  } else {
    requestQueue.length = 0;
  }
}
// 移除队列
function removeQueue(id: string) {
  const index = requestQueue.findIndex(q => q.id === id);
  if (index > -1) {
    requestQueue.splice(index, 1);
  }
}

export {
  addQueue,
  cancelRequest,
  clearQueue,
  removeQueue,
  requestQueue, // 内部数据结构，只读模式
};
