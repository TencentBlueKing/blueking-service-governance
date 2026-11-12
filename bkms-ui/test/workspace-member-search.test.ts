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

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia } from 'pinia';
import { afterEach, expect, it, vi } from 'vitest';
import WorkspaceBasic from '~/pages/basic/base-info/basic.vue';
import { useUserStore } from '~/stores/user';

import { i18nGlobalMocks, I18nTStub } from './helpers/mock-i18n';

const { listWorkspaceRoleMemberGroups } = vi.hoisted(() => ({
  listWorkspaceRoleMemberGroups: vi.fn(),
}));

vi.mock('@blueking/table', async () => (await import('./helpers/mock-table')).tableMockFactory());
vi.mock('vue-i18n', async () => (await import('./helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => (await import('./helpers/mock-router')).createRouterMock());
vi.mock('~/api/modules/v1', () => ({
  WorkspaceService: {
    getWorkspace: vi.fn().mockResolvedValue({ id: 'ws-1', state: 'Ready' }),
    listWorkspaceRoleMemberGroups,
  },
}));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({
    currentSpace: 'ws-1',
    spaceState: { Ready: 'Ready' },
    getRepositoryTypeName: () => '',
  }),
}));

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

it('matches the user ID before names resolve and the display name after the lookup completes', async () => {
  const userId = 'opaque-user-id';
  const searchName = '张三';
  listWorkspaceRoleMemberGroups.mockResolvedValue([{ roleCode: 'admin', members: [userId] }]);
  vi.stubEnv('DEV', true);
  vi.stubEnv('BK_API_URL_TMPL', 'https://{api_name}.example.test');
  vi.spyOn(console, 'error').mockImplementation(() => {});

  let completeLookup!: (response: Response) => void;
  const pendingLookup = new Promise<Response>(resolve => {
    completeLookup = resolve;
  });
  const fetchNames = vi.fn<typeof fetch>().mockReturnValueOnce(pendingLookup);
  vi.stubGlobal('fetch', fetchNames);

  const pinia = createPinia();
  useUserStore(pinia).setUserInfo({ user_id: 'viewer', tenant_id: 'tenant-a' });
  render(WorkspaceBasic, {
    global: {
      plugins: [pinia],
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
      directives: { 'bk-tooltips': {} },
    },
  });

  // 表格立即渲染 userid，后台发起一次批量姓名查询
  expect(await screen.findByText(userId)).toBeInTheDocument();
  expect(fetchNames).toHaveBeenCalledTimes(1);
  const user = userEvent.setup();
  const search = screen.getByPlaceholderText('搜索用户名');

  // 姓名未就绪：按姓名搜索无结果
  await user.type(search, searchName);
  expect(await screen.findByText('搜索结果为空')).toBeInTheDocument();

  // 按 userid 始终可命中
  await user.clear(search);
  await user.type(search, userId);
  await waitFor(() => expect(screen.queryByText('搜索结果为空')).not.toBeInTheDocument());
  expect(screen.getByText(userId)).toBeInTheDocument();

  // 姓名查询完成后，按姓名即可命中（无需刷新或重试）
  completeLookup(Response.json({ code: 0, data: [{ bk_username: userId, display_name: searchName }] }));
  await user.clear(search);
  await user.type(search, searchName);
  await waitFor(() => expect(screen.queryByText('搜索结果为空')).not.toBeInTheDocument());
  expect(screen.getByText(userId)).toBeInTheDocument();
  expect(fetchNames).toHaveBeenCalledTimes(1);
  expect(listWorkspaceRoleMemberGroups).toHaveBeenCalledTimes(1);
});
