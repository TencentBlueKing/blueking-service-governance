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

import { ref } from 'vue';

import { defineStore } from 'pinia';
import { ApiServerService } from '~/api/modules/bkmsserver';

import type { AppDetailOutputObj } from '~/@types/app';
import type { IAppType } from '~/composables/app-type';

// 保留 re-export，兼容已有引用
export type { AppType, IAppType } from '~/composables/app-type';

export const useAppDetail = defineStore('appDetail', () => {
  const app = ref('');
  const appType = ref<IAppType>('');
  const appID = ref('');
  // 应用详情
  const appDetail = ref<AppDetailOutputObj | null>(null);
  const loading = ref(false);
  // 缓存请求 Promise,key 为 appID
  const fetchPromiseCache = new Map<string, Promise<AppDetailOutputObj | null>>();
  // 瞬态导航标记:用于跨路由跳转时,传递「去配置」的来源(source)
  const pendingBuilderSource = ref<null | string>(null);

  // 更新当前app缓存
  function updateAppName(appName: string) {
    app.value = appName;
  }

  // 更新当前app类型
  function updateAppType(type: IAppType) {
    appType.value = type;
  }

  // 更新当前 appID
  function updateAppID(id: string) {
    appID.value = id;
  }

  // 获取并保存应用详情
  async function fetchAppDetail(appId?: string) {
    const targetAppId = appId || appID.value;

    if (!targetAppId) return null;

    // 如果正在请求同一应用,复用 Promise 避免重复请求
    const cachedPromise = fetchPromiseCache.get(targetAppId);
    if (cachedPromise) {
      return cachedPromise;
    }

    // 创建新请求
    const fetchPromise = (async () => {
      try {
        loading.value = true;
        const res = await ApiServerService.GetApp({
          appID: targetAppId,
        });

        if (res) {
          appDetail.value = res;
          updateAppName(res.name || '');
          updateAppID(res.id || '');
          updateAppType((res.type || '') as IAppType);
          return res;
        }
        return null;
      } catch {
        appDetail.value = null;
        return null;
      } finally {
        loading.value = false;
        // 请求完成后立即清除缓存
        fetchPromiseCache.delete(targetAppId);
      }
    })();

    // 缓存当前请求, 仅防止同时发起的重复请求
    fetchPromiseCache.set(targetAppId, fetchPromise);
    return fetchPromise;
  }

  // 设置瞬态导航标记
  function setPendingBuilderSource(source: null | string) {
    pendingBuilderSource.value = source;
  }

  // 消费瞬态导航标记:返回当前标记并立即清空,保证「读后即清」只生效一次
  function consumePendingBuilderSource(): null | string {
    const source = pendingBuilderSource.value;
    pendingBuilderSource.value = null;
    return source;
  }

  // 重置所有应用相关状态
  function reset() {
    app.value = '';
    appType.value = '';
    appID.value = '';
    appDetail.value = null;
    // 重置时一并清空瞬态导航标记
    pendingBuilderSource.value = null;
  }

  return {
    appID,
    app,
    appType,
    appDetail,
    loading,
    pendingBuilderSource,
    updateAppID,
    updateAppName,
    updateAppType,
    fetchAppDetail,
    setPendingBuilderSource,
    consumePendingBuilderSource,
    reset,
  };
});
