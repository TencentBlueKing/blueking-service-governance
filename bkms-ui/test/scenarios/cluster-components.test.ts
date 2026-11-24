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
 * 覆盖组件列表、分组与安装入口，以及列表/安装/卸载的环境维度请求路径。
 *
 * 列表为自定义 div 分组（非 vxe 表格），故无需 stub @blueking/table；
 * 安装侧滑内含动态表单 ComponentsConfig（按 addon schema 渲染，较重），stub 保留
 * `visible` 和 `confirm` 契约；请求路径断言经过真实 Service 和 Fetch 的参数替换。
 */
import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { v1Fetch } from '~/api/clients';
import ClusterComponents from '~/pages/env/cluster-components/cluster-components.vue';

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  listPortPools: vi.fn(),
  infoBox: vi.fn<(options: { onConfirm: () => Promise<void> }) => void>(),
}));

// 保留真实 Service 和 URL 参数替换，只隔离传输层，捕获实际发出的请求路径。
vi.mock('~/api/interceptors', () => ({
  fetch: mocks.request,
  interceptors: { response: { use: vi.fn() } },
}));
vi.mock('~/api/modules/v1', async () => ({
  ClusterAddonService: (await import('~/api/modules/v1/cluster-addon')).ClusterAddonService,
  PortPoolService: { listPortPools: mocks.listPortPools },
}));

// 隔离侧滑中的动态表单，保留确认事件以验证父组件发出的安装请求。
vi.mock('~/pages/env/cluster-components/install-sideslider.vue', () => ({
  default: {
    name: 'StubInstallSideslider',
    props: ['visible'],
    emits: ['confirm'],
    template: `<div v-if="visible">
      安装组件侧滑
      <button @click="$emit('confirm', { namespace: 'bk-system', chartVersion: '1.0.0', values: {} })">
        确认安装组件
      </button>
    </div>`,
  },
}));

import { i18nPlugin } from '../helpers/mock-i18n';

// i18n 不在测试范围：useI18n 直译、$t 直译
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

// Message / InfoBox 为命令式 API，mock 避免 jsdom 下弹窗副作用
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: vi.fn(),
  InfoBox: mocks.infoBox,
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

function renderPage(addons: unknown[] = [], autoExpandAppType?: string) {
  v1Fetch.config.prefix = '/api/bkms/v1/bkms-server';
  mocks.request.mockResolvedValue({ addons });
  mocks.listPortPools.mockResolvedValue({ portPools: [] });
  return render(ClusterComponents, {
    props: { envId: 'env-1', hasClusterConfig: true, autoExpandAppType },
    global: { plugins: [i18nPlugin] },
  });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe('集群组件列表（S17）', () => {
  it('使用环境组件接口，路径不遗留空间和集群占位符', async () => {
    renderPage();
    await screen.findByText('暂无组件');
    expect(mocks.request).toHaveBeenCalledWith(
      '/api/bkms/v1/bkms-server/envs/env-1/cluster-addons',
      expect.objectContaining({ method: 'GET', needRes: true }),
    );
  });

  it('确认安装时使用环境组件路径并将安装参数放入请求体', async () => {
    renderPage(oneAddon);
    await screen.findByText(/必选组件/);
    await userEvent.click(screen.getByText(/必选组件/));
    await userEvent.click(screen.getByText('安装'));
    await userEvent.click(await screen.findByText('确认安装组件'));

    expect(mocks.request).toHaveBeenCalledWith(
      '/api/bkms/v1/bkms-server/envs/env-1/cluster-addons/monitor',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ namespace: 'bk-system', chartVersion: '1.0.0', values: {} }),
      }),
    );
  });

  it('确认卸载时使用环境组件路径', async () => {
    renderPage([{ ...oneAddon[0], supportedActions: ['uninstall'], installInfo: { status: 'deployed' } }]);
    await screen.findByText(/必选组件/);
    await userEvent.click(screen.getByText(/必选组件/));
    await userEvent.click(screen.getByText('卸载'));
    await mocks.infoBox.mock.calls[0][0].onConfirm();

    expect(mocks.request).toHaveBeenCalledWith(
      '/api/bkms/v1/bkms-server/envs/env-1/cluster-addons/monitor',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });

  it('平台托管组件统一展示在 bkmsApp 分组，保留 Helm 和 Agones 分组', async () => {
    renderPage([
      { ...oneAddon[0], requiredForAppTypes: ['trpc', 'taf', 'bkmsApp'], optionalForAppTypes: ['helm', 'agones'] },
    ]);
    expect(await screen.findByText('平台托管应用')).toBeVisible();
    expect(screen.queryByText('tRPC 应用')).not.toBeInTheDocument();
    expect(screen.queryByText('TAF 应用')).not.toBeInTheDocument();
    expect(screen.getByText('Helm 应用')).toBeVisible();
    expect(screen.getByText('Agones 应用')).toBeVisible();
    await userEvent.click(screen.getByText('平台托管应用'));
    expect(await screen.findByText('监控插件')).toBeVisible();
  });

  it.each(['trpc', 'taf', 'bkmsApp'])('从 %s 应用跳转时自动展开统一分组', async appType => {
    renderPage([{ ...oneAddon[0], requiredForAppTypes: ['trpc', 'taf', 'bkmsApp'] }], appType);
    expect(await screen.findByText('平台托管应用')).toBeVisible();
    expect(await screen.findByText('监控插件')).toBeVisible();
  });

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
