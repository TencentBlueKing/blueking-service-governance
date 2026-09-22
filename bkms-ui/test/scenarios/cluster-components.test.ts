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
 * 场景级测试：集群组件安装与配置（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S17）
 *
 * 覆盖三组用户可感知行为：
 *   空态（无组件） / 分组展开显示组件与安装入口 / 点击安装弹出侧滑 → V = 3
 *
 * 列表为自定义 div 分组（非 vxe 表格），故无需 stub @blueking/table；
 * 安装侧滑内含动态表单 ComponentsConfig（按 addon schema 渲染，较重），stub 为仅按
 * `visible` 渲染标题的占位组件，以验证「点击安装 → 父组件 isShowInstallSideslider 置真
 * → 侧滑可见」这一用户可感知行为（侧滑内容的动态表单由 ComponentsConfig 单测负责）。
 */
import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ClusterComponents from '~/pages/env/cluster-components/cluster-components.vue';

const mocks = vi.hoisted(() => ({
  listClusterAddons: vi.fn(),
  listPortPools: vi.fn(),
}));

vi.mock('~/api/modules/v1', () => ({
  ClusterAddonService: { listClusterAddons: mocks.listClusterAddons },
  PortPoolService: { listPortPools: mocks.listPortPools },
}));

// 安装侧滑（含动态表单 ComponentsConfig）较重，stub 为仅按 visible 渲染标题的占位。
vi.mock('~/pages/env/cluster-components/install-sideslider.vue', () => ({
  default: {
    name: 'StubInstallSideslider',
    props: ['visible'],
    template: '<div v-if="visible">安装组件侧滑</div>',
  },
}));

import { i18nPlugin } from '../helpers/mock-i18n';

// i18n 不在测试范围：useI18n 直译、$t 直译
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

// Message / InfoBox 为命令式 API，mock 避免 jsdom 下弹窗副作用
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: vi.fn(),
  InfoBox: vi.fn(),
}));

vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock();
});

const oneAddon = [
  {
    name: 'monitor',
    displayName: '监控插件',
    description: '监控组件',
    requiredForAppTypes: ['default'],
    optionalForAppTypes: [],
    installInfo: { status: '' },
    supportedActions: ['install'],
  },
];

function renderPage(addons: unknown[] = []) {
  mocks.listClusterAddons.mockResolvedValue({ addons });
  mocks.listPortPools.mockResolvedValue({ portPools: [] });
  return render(ClusterComponents, {
    props: { envId: 'env-1', hasClusterConfig: true },
    global: { plugins: [i18nPlugin] },
  });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe('集群组件列表（S17）', () => {
  it('当集群无组件时，应显示「暂无组件」空态', async () => {
    renderPage([]);
    expect(await screen.findByText('暂无组件')).toBeTruthy();
  });

  it('当列表有组件时，分组默认收起；点击分组头部可展开显示组件名与「安装」入口', async () => {
    const { getByText, queryByText } = renderPage(oneAddon);
    // 等待列表加载完成后再定位分组头部
    await screen.findByText(/必选组件/);
    // 收起态：组件名不渲染
    expect(queryByText('监控插件')).toBeNull();
    // 以头部行的语义文本为锚点点击（事件冒泡至头部行 @click），避免依赖 cursor-pointer 样式类
    await userEvent.click(getByText(/必选组件/));
    expect(await screen.findByText('监控插件')).toBeTruthy();
    expect(getByText('安装')).toBeTruthy();
  });

  it('当用户点击「安装」时，应弹出安装组件侧滑', async () => {
    const { getByText } = renderPage(oneAddon);
    await screen.findByText(/必选组件/);
    await userEvent.click(getByText(/必选组件/));
    const installBtn = await screen.findByText('安装');
    await userEvent.click(installBtn);
    expect(await screen.findByText('安装组件侧滑')).toBeTruthy();
  });
});
