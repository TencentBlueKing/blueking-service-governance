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

import { createPinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useUserStore } from '~/stores/user';
import { useUserNameStore } from '~/stores/user-name';

const initialTime = Date.parse('2026-01-01T00:00:00Z');
const fetchMock = vi.fn<typeof fetch>();

function createStore(tenantId = 'tenant-a') {
  const pinia = createPinia();
  useUserStore(pinia).setUserInfo({ user_id: 'viewer', tenant_id: tenantId });
  return useUserNameStore(pinia);
}

function displayInfo(...users: { bk_username: string; display_name: string }[]) {
  return Response.json({ code: 0, data: users });
}

beforeEach(() => {
  vi.stubEnv('BK_API_URL_TMPL', 'https://{api_name}.example.test');
  vi.stubGlobal('fetch', fetchMock);
  vi.spyOn(Date, 'now').mockReturnValue(initialTime);
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('user name resolution', () => {
  it('skips resolution for the default tenant', async () => {
    const store = createStore('default');

    await store.resolveUserNames(['member']);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it.each([
    { failure: 'HTTP error', response: () => new Response(null, { status: 503 }) },
    { failure: 'business error', response: () => Response.json({ code: 1, data: [] }) },
    { failure: 'missing data', response: () => Response.json({ code: 0 }) },
  ])('preserves cached names and allows immediate retry after $failure', async ({ response }) => {
    const store = createStore();
    fetchMock.mockResolvedValueOnce(displayInfo({ bk_username: 'member', display_name: 'Old name' }));
    await store.resolveUserNames(['member']);

    vi.mocked(Date.now).mockReturnValue(initialTime + 5 * 60_000);
    fetchMock.mockResolvedValueOnce(response());
    await store.resolveUserNames(['member', 'missing']);
    expect(store.getDisplayName('member')).toBe('Old name');
    expect(store.getDisplayName('missing')).toBe('');

    fetchMock.mockResolvedValueOnce(displayInfo({ bk_username: 'member', display_name: 'New name' }));
    await store.resolveUserNames(['member', 'missing']);
    expect(store.getDisplayName('member')).toBe('New name');
    expect(store.getDisplayName('missing')).toBe('missing');
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('replaces an expired name when a successful batch omits that user', async () => {
    const store = createStore();
    fetchMock
      .mockResolvedValueOnce(displayInfo({ bk_username: 'member', display_name: 'Old name' }))
      .mockResolvedValueOnce(displayInfo())
      .mockResolvedValueOnce(displayInfo({ bk_username: 'member', display_name: 'New name' }));

    await store.resolveUserNames(['member']);
    vi.mocked(Date.now).mockReturnValue(initialTime + 5 * 60_000);
    await store.resolveUserNames(['member']);
    expect(store.getDisplayName('member')).toBe('member');

    // 查不到的降级值在短缓存期内不再发起请求
    await store.resolveUserNames(['member']);
    expect(fetchMock).toHaveBeenCalledTimes(2);

    vi.mocked(Date.now).mockReturnValue(initialTime + 5 * 60_000 + 30_000);
    await store.resolveUserNames(['member']);
    expect(store.getDisplayName('member')).toBe('New name');
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('retries only unresolved users after a partial failure', async () => {
    const store = createStore();
    fetchMock
      .mockResolvedValueOnce(displayInfo({ bk_username: 'first', display_name: 'First' }))
      .mockResolvedValueOnce(new Response(null, { status: 503 }))
      .mockResolvedValueOnce(displayInfo({ bk_username: 'second', display_name: 'Second' }));

    await Promise.all([store.resolveUserNames(['first']), store.resolveUserNames(['second'])]);
    expect(store.getDisplayName('first')).toBe('First');
    expect(store.getDisplayName('second')).toBe('');

    await store.resolveUserNames(['first', 'second']);
    expect(store.getDisplayName('second')).toBe('Second');
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.lastCall?.[0]).toEqual(expect.stringMatching(/bk_usernames=second$/));
  });
});
