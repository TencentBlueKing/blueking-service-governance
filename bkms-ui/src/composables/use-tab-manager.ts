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

import { onBeforeUnmount } from 'vue';

/**
 * 提供打开新标签页的功能，如果标签页已存在则聚焦到已有标签页
 */
export function useTabManager() {
  // 存储已打开的窗口引用
  const openedWindows = new Map<string, Window>();

  /**
   * 打开新标签页或聚焦已存在的标签页
   * @param url - 要打开的 URL
   * @param key - 用于标识标签页的唯一键，如果不提供则使用 URL 作为键
   * @returns Promise<Window | null> - 返回打开的窗口对象，如果打开失败则返回 null
   */
  const openTab = async (url: string, key?: string): Promise<null | Window> => {
    const windowKey = key || url;
    const existingWindow = openedWindows.get(windowKey);

    // 检查是否已有打开的窗口且未关闭
    if (existingWindow && !existingWindow.closed) {
      existingWindow.focus();
      return existingWindow;
    }

    // 打开新窗口
    const newWindow = window.open(url, '_blank');
    if (newWindow) {
      openedWindows.set(windowKey, newWindow);

      // 监听窗口关闭事件，清理引用
      const checkWindowClosed = () => {
        if (newWindow.closed) {
          openedWindows.delete(windowKey);
        } else {
          setTimeout(checkWindowClosed, 1000);
        }
      };
      checkWindowClosed();
    }

    return newWindow;
  };

  /**
   * 关闭指定的标签页
   * @param key - 标签页的唯一键
   */
  const closeTab = (key: string) => {
    const window = openedWindows.get(key);
    if (window && !window.closed) {
      window.close();
    }
    openedWindows.delete(key);
  };

  /**
   * 检查指定标签页是否已打开
   * @param key - 标签页的唯一键
   * @returns boolean - 是否已打开
   */
  const isTabOpen = (key: string): boolean => {
    const window = openedWindows.get(key);
    return window ? !window.closed : false;
  };

  /**
   * 清理所有已打开的标签页引用
   */
  const clearAllTabs = () => {
    openedWindows.clear();
  };

  // 组件卸载时清理所有窗口引用
  onBeforeUnmount(() => {
    clearAllTabs();
  });

  return {
    openTab,
    closeTab,
    isTabOpen,
    clearAllTabs,
  };
}
