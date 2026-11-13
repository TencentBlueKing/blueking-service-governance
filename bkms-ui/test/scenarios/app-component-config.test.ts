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

/**
 * 场景级测试：应用组件配置的可用环境
 *
 * 覆盖可见及折叠环境的名称和标识、同名环境与全局组件筛选、两处筛选状态同步。
 * mock 应用详情与环境接口，stub 组件编辑侧滑、顶部搜索控件和浮层定位。
 */
import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import ModulesConfig from '~/pages/application/detail/modules-config.vue';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

const mocks = vi.hoisted(() => ({
  fetchAppDetail: vi.fn(),
  listAppEnvs: vi.fn(),
}));

vi.mock('~/api/modules/v1', () => ({ AppComponentsService: {} }));
vi.mock('~/api/modules/v1/env', () => ({ EnvService: { listAppEnvs: mocks.listAppEnvs } }));
vi.mock('~/api/modules/bkmsserver', () => ({ ApiServerService: {} }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'space-a' }) }));
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

vi.mock('~/components/modules/components-sideSlider.vue', () => ({ default: { template: '<div />' } }));

vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({
    appID: 'app-a',
    loading: false,
    fetchAppDetail: mocks.fetchAppDetail,
    appDetail: {
      appModelSpec: {
        components: [
          {
            name: 'scoped-component',
            type: 'deploy',
            scopeType: 'environment',
            scopeEnvNames: ['feat-first', 'dev', 'feat-overflow', 'archived'],
          },
          { name: 'standard-component', type: 'deploy', scopeType: 'environment', scopeEnvNames: ['staging'] },
          { name: 'global-component', type: 'deploy', scopeType: 'global' },
        ],
      },
    },
  }),
}));

beforeAll(installVxeShims);
beforeEach(() => {
  mocks.fetchAppDetail.mockResolvedValue(undefined);
  mocks.listAppEnvs.mockReset().mockResolvedValue([
    { name: 'feat-first', displayName: '联调环境', kind: 'feature', type: 'development' },
    { name: 'dev', displayName: '开发环境', kind: 'standard', type: 'development' },
    { name: 'feat-overflow', kind: 'feature', type: 'development' },
    { name: 'staging', displayName: '联调环境', kind: 'standard', type: 'staging' },
  ]);
});

function renderPage() {
  return render(ModulesConfig, {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
      directives: { 'bk-tooltips': {} },
      stubs: {
        Popover: {
          props: { trigger: { type: String, default: 'hover' } },
          data: () => ({ isOpen: false }),
          methods: {
            hide() {
              this.isOpen = false;
            },
            updatePopover() {},
          },
          template: `<div>
            <span @mouseenter="trigger === 'hover' && (isOpen = true)"
              @click="trigger === 'click' && (isOpen = true)"><slot /></span>
            <div v-if="isOpen"><slot name="content" /></div>
          </div>`,
        },
        SearchSelect: {
          props: ['data', 'modelValue'],
          emits: ['update:modelValue'],
          template: `<div aria-label="顶部搜索">
            <template v-for="item in data" :key="item.id">
              <button v-for="option in item.children" :key="option.id"
                :aria-pressed="modelValue.some(filter => filter.id === item.id && filter.values.some(value => value.id === option.id))"
                @click="$emit('update:modelValue', [{ id: item.id, values: [option] }])">{{ option.name }}</button>
            </template>
          </div>`,
        },
      },
    },
  });
}

describe('应用组件配置：可用环境', () => {
  it('当用户查看可用环境时，应显示环境名称、特性标识及缺失名称的回退值', async () => {
    renderPage();

    expect(await screen.findByText('特性')).toBeVisible();
    expect(screen.getAllByText('联调环境')).toHaveLength(3);
    expect(screen.getAllByText('开发环境')).toHaveLength(2);
    expect(screen.getAllByText('所有环境')).toHaveLength(2);
    expect(mocks.listAppEnvs).toHaveBeenCalledExactlyOnceWith({ appID: 'app-a' });

    await userEvent.hover(screen.getByText('+2'));
    expect(await screen.findByText('feat-overflow')).toBeVisible();
    expect(screen.getAllByText('特性')).toHaveLength(2);
    expect(screen.getAllByText('archived')).toHaveLength(2);
  });

  it('当用户在顶部选择特性环境时，应匹配对应组件并同步列头筛选', async () => {
    renderPage();
    expect(await screen.findByText('global-component')).toBeVisible();

    await userEvent.click(await screen.findByRole('button', { name: '联调环境（特性）' }));
    expect(screen.getByText('scoped-component')).toBeVisible();
    expect(screen.queryByText('standard-component')).not.toBeInTheDocument();
    expect(screen.queryByText('global-component')).not.toBeInTheDocument();

    await userEvent.click(screen.getByText('可用环境'));
    expect(await screen.findByRole('checkbox', { name: '联调环境（特性）' })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: '联调环境' })).not.toBeChecked();
  });

  it('当用户在列头选择标准环境时，应排除同名特性环境并同步顶部筛选', async () => {
    renderPage();
    expect(await screen.findByText('特性')).toBeVisible();

    await userEvent.click(screen.getByText('可用环境'));
    await userEvent.click(await screen.findByRole('checkbox', { name: '联调环境' }));
    await userEvent.click(screen.getByRole('button', { name: '确定' }));

    expect(screen.getByText('standard-component')).toBeVisible();
    expect(screen.queryByText('scoped-component')).not.toBeInTheDocument();
    expect(screen.queryByText('global-component')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '联调环境', pressed: true })).toBeVisible();
    expect(screen.getByRole('button', { name: '联调环境（特性）', pressed: false })).toBeVisible();
  });

  it('当用户选择所有环境时，应筛出全局组件', async () => {
    renderPage();
    expect(await screen.findByText('scoped-component')).toBeVisible();

    await userEvent.click(screen.getByRole('button', { name: '所有环境' }));

    expect(screen.getByText('global-component')).toBeVisible();
    expect(screen.queryByText('scoped-component')).not.toBeInTheDocument();
    expect(screen.queryByText('standard-component')).not.toBeInTheDocument();
  });
});
