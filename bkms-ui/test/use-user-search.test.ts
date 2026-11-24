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

import { defineComponent } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import useUserSearch from '~/composables/use-user-search';
import { useFrontendFeaturesStore } from '~/stores/frontend-features';
import { useUserStore } from '~/stores/user';

import type { VueWrapper } from '@vue/test-utils';
import type { ISearchItem } from 'bkui-vue/lib/search-select/utils';

const field: ISearchItem = { id: 'username', name: '操作人' };

function deferred<T>() {
  let resolve: (value: T) => void = () => {
    throw new Error('Promise resolver is not initialized');
  };
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}

function usersResponse(id: string, name: string) {
  return Response.json({
    code: 0,
    data: [
      {
        bk_username: id,
        display_name: name,
        full_name: name,
        login_name: id,
        owner_tenant_id: 'tenant-a',
        status: 'normal',
      },
    ],
  });
}

describe('useUserSearch', () => {
  let host: VueWrapper;
  let pinia: ReturnType<typeof createPinia>;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubEnv('DEV', false);
    vi.stubEnv('BK_API_URL_TMPL', 'https://{api_name}.example.test');
    pinia = createPinia();
    setActivePinia(pinia);
    useFrontendFeaturesStore().enableMultiTenantMode = true;
    useUserStore().setUserInfo({ user_id: 'viewer', tenant_id: 'tenant-a' });
  });

  afterEach(() => {
    host?.unmount();
    vi.useRealTimers();
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  function menuList() {
    let getMenuList: ReturnType<typeof useUserSearch>['searchSelectBind']['value']['getMenuList'];
    const searchHost = defineComponent({
      setup() {
        const bind = useUserSearch().searchSelectBind.value;
        getMenuList = 'getMenuList' in bind ? bind.getMenuList : undefined;
        return {};
      },
      template: '<div />',
    });
    host = mount(searchHost, { global: { plugins: [pinia] } });
    if (!getMenuList) throw new Error('多租户远程搜索未启用');
    return getMenuList;
  }

  it('防抖结束后返回当前关键词的候选人', async () => {
    const fetchUsers = vi.fn<typeof fetch>().mockResolvedValue(usersResponse('bob', 'Bob'));
    vi.stubGlobal('fetch', fetchUsers);
    const getMenuList = menuList();

    const pending = getMenuList(field, 'bob');
    await vi.advanceTimersByTimeAsync(300);
    await expect(pending).resolves.toEqual([{ id: 'bob', name: 'Bob' }]);
    expect(String(fetchUsers.mock.calls[0]?.[0])).toContain('keyword=bob');
  });

  it('旧关键词在新关键词的防抖等待期内返回时，不写入候选菜单', async () => {
    const alice = deferred<Response>();
    const bob = deferred<Response>();
    const fetchUsers = vi.fn<typeof fetch>().mockReturnValueOnce(alice.promise).mockReturnValueOnce(bob.promise);
    vi.stubGlobal('fetch', fetchUsers);
    const getMenuList = menuList();

    let aliceSettled = false;
    void getMenuList(field, 'alice').then(() => {
      aliceSettled = true;
    });
    await vi.advanceTimersByTimeAsync(300);
    expect(fetchUsers).toHaveBeenCalledTimes(1);
    expect(String(fetchUsers.mock.calls[0]?.[0])).toContain('keyword=alice');

    const bobMenu = getMenuList(field, 'bob');
    alice.resolve(usersResponse('alice', 'Alice'));
    await flushPromises();
    expect(aliceSettled).toBe(false);
    expect(fetchUsers).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(300);
    expect(fetchUsers).toHaveBeenCalledTimes(2);
    expect(String(fetchUsers.mock.calls[1]?.[0])).toContain('keyword=bob');
    bob.resolve(usersResponse('bob', 'Bob'));
    await expect(bobMenu).resolves.toEqual([{ id: 'bob', name: 'Bob' }]);
    await flushPromises();
    expect(aliceSettled).toBe(false);
  });
});
