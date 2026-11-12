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

import BkUserDisplayName from '@blueking/bk-user-display-name';
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getUserDisplayConfig } from '~/common/bk-user-api';
import UserDisplay from '~/components/user-display.vue';

const gateway = 'https://bk-user-web.example.test/prod';
const fetchUser = vi.fn<typeof fetch>();

function mountDisplay(props: InstanceType<typeof UserDisplay>['$props'], tenantId: string | undefined) {
  const config = getUserDisplayConfig(tenantId);
  if (config) BkUserDisplayName.configure(config);
  return mount(UserDisplay, { props });
}

function userResponse(displayName: string) {
  return Response.json({ data: { display_name: displayName } });
}

enableAutoUnmount(afterEach);

beforeEach(() => {
  vi.stubEnv('DEV', false);
  vi.stubEnv('BK_API_URL_TMPL', 'https://{api_name}.example.test');
  vi.stubGlobal('fetch', fetchUser);
  fetchUser.mockReset();
  BkUserDisplayName.userCache.clear();
  BkUserDisplayName.tenantId = '';
  BkUserDisplayName.apiBaseUrl = '';
});

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('UserDisplay', () => {
  it.each(['system', 'tenant-a'])('looks up the original user ID in tenant %s', async tenantId => {
    fetchUser.mockResolvedValueOnce(userResponse('张三'));
    const wrapper = mountDisplay({ value: 'opaque-user-id' }, tenantId);

    await flushPromises();

    expect(wrapper.text()).toBe('张三');
    expect(fetchUser).toHaveBeenCalledExactlyOnceWith(
      `${gateway}/api/v3/open-web/tenant/users/opaque-user-id/display_info/`,
      { credentials: 'include', headers: { 'X-Bk-Tenant-Id': tenantId } },
    );
  });

  it('uses the local user gateway proxy in development', async () => {
    vi.stubEnv('DEV', true);
    fetchUser.mockResolvedValueOnce(userResponse('张三'));
    const wrapper = mountDisplay({ value: 'local-id' }, 'tenant-a');

    await flushPromises();

    expect(wrapper.text()).toBe('张三');
    expect(fetchUser).toHaveBeenCalledWith(
      '/api-bk-user-selector/api/v3/open-web/tenant/users/local-id/display_info/',
      expect.any(Object),
    );
  });

  it('preserves person order and the requested separator, and skips blank IDs', async () => {
    fetchUser.mockImplementation(input => {
      const name = String(input).includes('/first/') ? '张三' : '李四';
      return Promise.resolve(userResponse(name));
    });
    const wrapper = mountDisplay({ value: ['first', '', 'second'], separator: ' / ' }, 'tenant-a');

    await flushPromises();

    expect(wrapper.text()).toBe('张三 / 李四');
    expect(fetchUser).toHaveBeenCalledTimes(2);
  });

  it.each([undefined, 'default'])('preserves the original text when tenant_id is %s', async tenantId => {
    const wrapper = mountDisplay({ value: ['first', 'second'], separator: ' / ' }, tenantId);
    expect(wrapper.text()).toBe('first / second');
    expect(fetchUser).not.toHaveBeenCalled();
  });

  it('keeps plain values as-is without a lookup', () => {
    const wrapper = mountDisplay({ value: 'system', plain: true }, 'tenant-a');

    expect(wrapper.text()).toBe('system');
    expect(fetchUser).not.toHaveBeenCalled();
  });

  it('uses the supplied empty text for blank values without a lookup', async () => {
    const wrapper = mountDisplay({ value: null, emptyText: '无' }, 'tenant-a');
    expect(wrapper.text()).toBe('无');

    await wrapper.setProps({ value: [' ', ''] });
    expect(wrapper.text()).toBe('无');
    expect(fetchUser).not.toHaveBeenCalled();
  });

  it('keeps the original ID when the gateway is unconfigured', () => {
    vi.stubEnv('BK_API_URL_TMPL', '');
    const wrapper = mountDisplay({ value: 'original-id' }, 'tenant-a');

    expect(wrapper.text()).toBe('original-id');
    expect(fetchUser).not.toHaveBeenCalled();
  });

  it('keeps the original IDs when a user is missing or the request fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    fetchUser
      .mockResolvedValueOnce(Response.json({ message: 'User not found' }, { status: 404 }))
      .mockRejectedValueOnce(new Error('Network unavailable'));
    const wrapper = mountDisplay({ value: ['missing-id', 'unreachable-id'], separator: ' / ' }, 'tenant-a');

    await flushPromises();

    expect(wrapper.text()).toBe('missing-id / unreachable-id');
    expect(fetchUser).toHaveBeenCalledTimes(2);
  });

  it('keeps the new name when an earlier lookup completes last', async () => {
    let finishEarlier!: (response: Response) => void;
    const earlier = new Promise<Response>(resolve => {
      finishEarlier = resolve;
    });
    fetchUser.mockReturnValueOnce(earlier).mockResolvedValueOnce(userResponse('新用户'));
    const wrapper = mountDisplay({ value: 'previous-id' }, 'tenant-a');

    await wrapper.setProps({ value: 'new-id' });
    await flushPromises();
    expect(wrapper.text()).toBe('新用户');

    finishEarlier(userResponse('旧用户'));
    await flushPromises();
    expect(wrapper.text()).toBe('新用户');
  });
});
