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

import { defineComponent, ref } from 'vue';

import { fireEvent, render, screen, waitFor } from '@testing-library/vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Home from '~/pages/home/home.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { AppDeployedEnvOutputObj } from '~/@types/v1/workspace';

const { listWorkspacesOverview } = vi.hoisted(() => ({ listWorkspacesOverview: vi.fn() }));

vi.mock('~/api/modules/v1', async importOriginal => ({
  ...(await importOriginal<object>()),
  WorkspaceService: { listWorkspacesOverview },
}));
vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async importOriginal => ({
  ...(await importOriginal<object>()),
  useRouter: () => ({ push: vi.fn() }),
}));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({}) }));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ list: [], handleGetWorkspaceList: vi.fn() }),
}));
vi.mock('~/stores/user', () => ({ useUserStore: () => ({ userInfo: {} }) }));
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Popover: defineComponent({
    props: { disabled: Boolean },
    setup: () => ({ open: ref(false) }),
    template: '<div @mouseenter="open = !disabled"><slot /><slot v-if="open" name="content" /></div>',
  }),
  Tag: { template: '<span data-testid="env-tag"><slot /></span>' },
}));

const development = { id: 'dev', name: 'dev', displayName: '开发环境', type: 'development', kind: 'standard' };
const production = { id: 'prod', name: 'prod', displayName: '生产环境', type: 'production', kind: 'standard' };
const feature = { id: 'feat-demo', name: 'feat-demo', displayName: '功能联调', type: 'development', kind: 'feature' };

async function hoverTag(label: string) {
  const tag = visibleTags().find(tag => tag.textContent?.trim() === label);
  if (!tag?.parentElement) throw new Error(`找不到标签 ${label}`);
  await fireEvent.mouseEnter(tag.parentElement);
  return tag;
}

async function renderHome(deployedEnvs: AppDeployedEnvOutputObj[], columnWidth = 800) {
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(columnWidth);
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(70);
  listWorkspacesOverview.mockResolvedValue([
    { id: 'workspace-demo', displayName: '应用空间', apps: [{ id: 'app-demo', name: '示例应用', deployedEnvs }] },
  ]);
  render(Home, {
    global: {
      directives: { 'bk-tooltips': {} },
      mocks: i18nGlobalMocks,
      stubs: { EmptyHome: true, Skeleton: { template: '<div><slot /></div>' }, TeamSpace: true },
    },
  });
  await screen.findByRole('button', { name: '示例应用' });
}

function visibleTags() {
  return screen.getAllByTestId('env-tag').filter(tag => !tag.closest('.invisible'));
}

afterEach(() => vi.restoreAllMocks());

describe('首页已部署环境', () => {
  it('保留标准环境分组并在末尾展示同类型的特性环境', async () => {
    await renderHome([feature, production, development]);

    expect(visibleTags().map(tag => tag.textContent?.trim())).toEqual(['生产（1）', '开发（1）', '特性（1）']);
    expect(await hoverTag('特性（1）')).toHaveClass('bg-[#E2F5F7]', 'text-[#3A9EAA]');
    expect(screen.getByText('功能联调')).toBeInTheDocument();
    expect(screen.queryByText('feat-demo')).not.toBeInTheDocument();
    expect(await hoverTag('开发（1）')).toHaveClass('env-tag-development');
    expect(screen.getByText('开发环境')).toBeInTheDocument();
  });

  it('只有特性环境且缺少展示名时，悬浮明细仍显示内部名', async () => {
    await renderHome([{ ...feature, displayName: undefined }]);

    expect(visibleTags().map(tag => tag.textContent?.trim())).toEqual(['特性（1）']);
    await hoverTag('特性（1）');
    expect(screen.getByText('feat-demo')).toBeInTheDocument();
  });

  it('窄列的折叠标签保留特性分组及环境明细', async () => {
    await renderHome([development, production, feature], 190);

    await waitFor(() => {
      expect(visibleTags().map(tag => tag.textContent?.trim())).toEqual(['开发（1）', '生产（1）', '+1']);
    });
    await hoverTag('+1');
    await hoverTag('特性（1）');
    expect(screen.getByText('功能联调')).toBeInTheDocument();
  });
});
