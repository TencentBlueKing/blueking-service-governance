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

import { computed, onUnmounted } from 'vue';

import { debounce } from 'lodash-es';
import { getBkUserApiUrl } from '~/common/bk-user-api';
import { useUserStore } from '~/stores/user';

import type { ICommonItem, ISearchItem } from 'bkui-vue/lib/search-select/utils';

/** bk-user search 接口返回的用户结构 */
interface BkUserSearchItem {
  bk_username: string;
  display_name: string;
  full_name: string;
  login_name: string;
  owner_tenant_id: string;
  status: string;
}

/** search 接口响应（部分环境无 code 字段，解析需兼容） */
interface BkUserSearchResponse {
  code?: number;
  data?: BkUserSearchItem[];
  message?: string;
}

/**
 * 多租户环境下的人员远程搜索
 * 通过 bk-user 网关的 tenant users search 接口按关键词搜索人员，
 * 为 SearchSelect 提供搜索配置装饰器与 getMenuList hook：
 * - 下拉展示 display_name，实际查询值使用 bk_username
 * 非多租户环境 searchSelectBind 为空对象、目标字段不标记 async，保持纯文本输入现状
 * @param fieldId 接入远程搜索的筛选项 id（默认 username）
 */
export default function useUserSearch(fieldId = 'username') {
  const userStore = useUserStore();

  /** 是否多租户：已配置用户网关且租户非 default（与 user-name store 的 canResolve 逻辑一致） */
  const isMultiTenant = computed(() => {
    const tenantId = userStore.tenantId;
    return !!getBkUserApiUrl() && !!tenantId && tenantId !== 'default';
  });

  /** 请求序号：递增即令所有在途请求过期，用于丢弃乱序返回的旧响应 */
  let requestSeq = 0;

  async function fetchUsers(keyword: string): Promise<BkUserSearchItem[]> {
    const apiBaseUrl = getBkUserApiUrl();
    const tenantId = userStore.tenantId;
    const response = await fetch(
      `${apiBaseUrl}/api/v3/open-web/tenant/users/-/search/?keyword=${encodeURIComponent(keyword)}`,
      {
        credentials: 'include',
        headers: { 'X-Bk-Tenant-Id': tenantId },
      },
    );
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const res: BkUserSearchResponse = await response.json();
    if (res?.code !== undefined && res.code !== 0) {
      throw new Error(res.message || `code ${res.code}`);
    }
    if (!Array.isArray(res?.data)) {
      return [];
    }
    return res.data;
  }

  /**
   * 执行搜索并通过 resolve 返回候选列表。
   * 过期响应（requestSeq 已被更新请求或清空输入递增）静默丢弃、不 resolve：
   * 组件内每次 setMenuList 调用独立管理 loading，loading 由最新一次调用复位，
   * 过期请求若 resolve 反而会把已更新的菜单列表覆写为空/旧结果
   */
  async function searchUsers(keyword: string, resolve: (value: ICommonItem[]) => void): Promise<void> {
    const seq = ++requestSeq;
    try {
      const users = await fetchUsers(keyword);
      if (seq !== requestSeq) return;
      resolve(users.map(user => ({ id: user.bk_username, name: user.display_name })));
    } catch (err) {
      // 搜索失败降级为空列表，不阻塞页面；过期请求的失败同样不覆写新结果
      console.error('搜索人员失败:', err);
      if (seq === requestSeq) resolve([]);
    }
  }

  // 防抖包装：延迟期内调用返回 undefined，而 SearchSelect 内部会对返回值直接 .catch()，
  // 因此通过 resolve 回调 + Promise 包装保证每次调用都同步返回 Promise
  const debouncedSearch = debounce((keyword: string, resolve: (value: ICommonItem[]) => void) => {
    searchUsers(keyword, resolve);
  }, 300);

  /** 对接 SearchSelect :get-menu-list（由 searchSelectBind 注入，仅多租户时生效） */
  function handleGetMenuList(item: ISearchItem, keyword: string): Promise<ICommonItem[]> {
    return new Promise(resolve => {
      // 非目标筛选项或空关键词：立即返回空列表，不发请求；
      // 同时递增 requestSeq 使在途请求过期、取消挂起的防抖调用，
      // 避免清空输入后旧响应仍写入菜单列表
      if (item?.id !== fieldId || !keyword.trim()) {
        requestSeq++;
        debouncedSearch.cancel();
        resolve([]);
        return;
      }
      debouncedSearch(keyword, resolve);
    });
  }

  /**
   * 装饰搜索配置：把远程/本地行为标记集中在一处
   * - 目标字段（fieldId）多租户时标记 async: true 走远程搜索
   * - 其余字段显式 async: false——一旦 SearchSelect 传入 getMenuList，
   *   async 为 undefined 的项都会被当作远程处理，必须显式关闭
   */
  function decorateSearchData(items: ISearchItem[]): ISearchItem[] {
    return items.map(item => {
      if (item.id !== fieldId) return { ...item, async: false };
      return isMultiTenant.value ? { ...item, async: true } : item;
    });
  }

  /** 多租户时注入远程搜索 hook，非多租户为空对象（零行为变化） */
  const searchSelectBind = computed(() => (isMultiTenant.value ? { getMenuList: handleGetMenuList } : {}));

  onUnmounted(() => debouncedSearch.cancel());

  return {
    isMultiTenant,
    decorateSearchData,
    searchSelectBind,
  };
}
