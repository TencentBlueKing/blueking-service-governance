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

import { getCurrentInstance, onUnmounted } from 'vue';

type EventCallback<T = any> = (data: T) => void;

// 全局事件监听器存储
const listeners = new Map<string, EventCallback[]>();

export function useEventBus() {
  // 记录当前组件注册的监听器,用于自动清理
  const componentListeners: Array<{ callback: EventCallback; event: string }> = [];
  const instance = getCurrentInstance();

  /**
   * 注册事件监听
   */
  const on = <T = any>(event: string, callback: EventCallback<T>) => {
    if (!listeners.has(event)) {
      listeners.set(event, []);
    }
    listeners.get(event)?.push(callback);

    // 如果在组件内调用,记录下来以便自动清理
    if (instance) {
      componentListeners.push({ event, callback });
    }
  };

  /**
   * 触发事件
   */
  const emit = <T = any>(event: string, data?: T) => {
    listeners.get(event)?.forEach(callback => callback(data));
  };

  /**
   * 移除特定事件的特定回调
   */
  const off = <T = any>(event: string, callback?: EventCallback<T>) => {
    if (!callback) {
      // 如果没有指定回调,移除该事件的所有监听器
      listeners.delete(event);
      return;
    }

    const callbacks = listeners.get(event);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
      // 如果该事件没有监听器了,删除该事件
      if (callbacks.length === 0) {
        listeners.delete(event);
      }
    }
  };

  /**
   * 清空所有事件监听器
   */
  const clear = () => {
    listeners.clear();
  };

  // 组件卸载时,自动清理该组件注册的监听器
  if (instance) {
    onUnmounted(() => {
      componentListeners.forEach(({ event, callback }) => {
        off(event, callback);
      });
      componentListeners.length = 0;
    });
  }

  return { on, emit, off, clear };
}
