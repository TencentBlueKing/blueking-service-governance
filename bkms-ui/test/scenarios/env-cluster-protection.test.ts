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
 * 场景级测试：环境管理 / 集群信息编辑保护（S9 的来源环境保护补充）
 * 路径清单：docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S9；本头记录新增交互路径。
 *
 * 覆盖：已部署应用拦截 / 未部署应用时使用最新集群配置 /
 * 获取详情失败 / 环境切换后的迟到响应 / 提交时后端拒绝 → V = 5。
 * mock 边界：API、详情与空间 store、集群诊断和组件配置等外围内容；
 * basic-info / BkmsContent / 拦截弹窗与列表 / bkui Form、Button 真实渲染。
 * 集群与命名空间选择器使用带可见输入值的契约 stub，以验证进入编辑及新旧配置来源。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';
import { MockedApiError } from '../helpers/mocked-api-error';

import type { EnvDetailOutput } from '~/@types/v1/env';

interface EnvStoreStub {
  currentEnv: EnvDetailOutput | null;
  syncSavedEnv: ReturnType<typeof vi.fn>;
  waitForCurrentEnv: ReturnType<typeof vi.fn>;
}

const mocks = vi.hoisted(() => ({
  getEnv: vi.fn(),
  message: vi.fn(),
  updateEnvCluster: vi.fn(),
  store: { current: undefined as EnvStoreStub | undefined },
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({
    route: { path: '/ws-1/env/env-a/basic', name: 'envBasicInfo', params: { space: 'ws-1' } },
  });
});
vi.mock('~/api/modules/v1', () => ({
  EnvService: { getEnv: mocks.getEnv, updateEnvCluster: mocks.updateEnvCluster },
  BkintegrationsKubeinsightService: {
    getLatestEnvReport: vi.fn().mockResolvedValue({ abnormalItems: [] }),
  },
}));
vi.mock('~/stores/frontend-features', () => ({
  useFrontendFeaturesStore: () => ({ enableKubeInsight: false }),
}));
vi.mock('~/stores/env-detail', async () => {
  const { reactive } = await import('vue');
  const store = reactive<EnvStoreStub>({
    currentEnv: null,
    syncSavedEnv: vi.fn().mockResolvedValue(undefined),
    waitForCurrentEnv: vi.fn().mockImplementation(async () => store.currentEnv),
  });
  mocks.store.current = store;
  return { useEnvDetailStore: () => store };
});
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1', workspaceDetail: { bkSystems: { bkBCSProjectID: 'project-1' } } }),
}));
vi.mock('~/components/cluster-selector/use-cluster-selector', async () => {
  const { ref } = await import('vue');
  return {
    default: () => ({ loading: ref(false), clusterData: ref([]), getClusterList: vi.fn().mockResolvedValue([]) }),
  };
});
vi.mock('~/pages/env/cluster-components/cluster-components.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/env/components/project-selector/project-selector.vue', () => ({
  default: { props: ['projectCode'], template: '<span>{{ projectCode }}</span>' },
}));
vi.mock('~/components/cluster-selector/cluster-selector.vue', () => ({
  default: {
    props: ['value'],
    template: '<input aria-label="集群选择器" :value="value" readonly />',
  },
}));
vi.mock('~/components/namespace-selector.vue', () => ({
  default: {
    props: ['value'],
    template: '<input aria-label="命名空间选择器" :value="value" readonly />',
  },
}));
vi.mock('~/components/field-item.vue', () => ({
  default: { props: ['fieldValue', 'value'], template: '<span>{{ fieldValue }} {{ value }}</span>' },
}));
vi.mock('~/components/cluster-health-score.vue', () => ({ default: { template: '<div />' } }));
vi.mock('bkui-vue', async importOriginal => ({
  ...((await importOriginal<object>()) as object),
  Message: mocks.message,
}));

const { default: BasicInfo } = await import('~/pages/env/basic-info.vue');

function makeEnv(id = 'env-a'): EnvDetailOutput {
  return {
    id,
    name: id,
    displayName: id === 'env-a' ? '来源环境A' : '来源环境B',
    type: 'test',
    appDeployStatuses: [],
    cluster: { clusterID: 'cluster-old', namespace: 'namespace-old', projectCode: 'project-1' },
  };
}

const deployedApp = { appID: 'app-id-2', appName: 'payment-service', appType: 'trpc', deployStatus: 'deployed' };

beforeEach(() => {
  vi.clearAllMocks();
  mocks.store.current!.currentEnv = makeEnv();
  mocks.getEnv.mockResolvedValue(makeEnv());
  mocks.updateEnvCluster.mockResolvedValue({});
});

async function clickClusterEdit() {
  // 基础信息与集群资源分别有一个「编辑」按钮，按页面可见顺序选择集群资源入口。
  await userEvent.click(screen.getAllByRole('button', { name: '编辑' })[1]);
}

async function renderPage() {
  const view = render(BasicInfo, {
    props: { env: 'env-a', workspace: 'ws-1' },
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
      directives: { 'bk-tooltips': {}, bkloading: {} },
      stubs: { Skeleton: { template: '<div><slot /></div>' } },
    },
  });
  await screen.findByText('来源环境A');
  return view;
}

describe('环境管理：集群信息编辑保护', () => {
  it('当用户编辑仅已部署应用的环境时，应保留原有应用拦截提示', async () => {
    mocks.getEnv.mockResolvedValue({ ...makeEnv(), appDeployStatuses: [deployedApp] });
    await renderPage();
    await clickClusterEdit();

    expect(await screen.findByText('无法修改集群信息')).toBeVisible();
    expect(screen.getByText('payment-service')).toBeVisible();
    expect(screen.queryByText('特性环境')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('集群选择器')).not.toBeInTheDocument();
  });

  it('当用户编辑未部署应用的环境时，应使用最新详情中的集群和命名空间进入编辑', async () => {
    mocks.getEnv.mockResolvedValue({
      ...makeEnv(),
      cluster: { clusterID: 'cluster-latest', namespace: 'namespace-latest', projectCode: 'project-1' },
    });
    await renderPage();
    await clickClusterEdit();

    expect(await screen.findByLabelText('集群选择器')).toHaveValue('cluster-latest');
    expect(screen.getByLabelText('命名空间选择器')).toHaveValue('namespace-latest');
    expect(screen.queryByText('无法修改集群信息')).not.toBeInTheDocument();
  });

  it('当用户编辑时获取最新详情失败时，应提示失败并阻止进入编辑', async () => {
    mocks.getEnv.mockRejectedValue(new MockedApiError());
    await renderPage();
    await clickClusterEdit();

    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'error' })));
    expect(screen.queryByLabelText('集群选择器')).not.toBeInTheDocument();
    expect(mocks.updateEnvCluster).not.toHaveBeenCalled();
  });

  it('当用户发起编辑后切换环境时，应丢弃旧环境的迟到检查结果', async () => {
    let finishCheck: (detail: EnvDetailOutput) => void = () => {};
    mocks.getEnv.mockImplementationOnce(() => new Promise<EnvDetailOutput>(resolve => (finishCheck = resolve)));
    const view = await renderPage();
    await clickClusterEdit();
    expect(mocks.getEnv).toHaveBeenCalledWith({ envID: 'env-a' });

    mocks.store.current!.currentEnv = makeEnv('env-b');
    await view.rerender({ env: 'env-b', workspace: 'ws-1' });
    await screen.findByText('来源环境B');
    finishCheck({ ...makeEnv(), appDeployStatuses: [deployedApp] });
    // 新环境检查成功进入编辑，是旧检查已经结束且页面仍可操作的正向信号。
    await clickClusterEdit();
    await waitFor(() => expect(mocks.getEnv).toHaveBeenLastCalledWith({ envID: 'env-b' }));
    expect(await screen.findByLabelText('集群选择器')).toBeVisible();
    expect(screen.queryByText('payment-service')).not.toBeInTheDocument();
    expect(screen.queryByText('无法修改集群信息')).not.toBeInTheDocument();
  });

  it('当用户通过预检查但保存时被后端拒绝时，应保留编辑且不提示成功', async () => {
    mocks.updateEnvCluster.mockRejectedValue(new MockedApiError());
    await renderPage();
    await clickClusterEdit();
    await screen.findByLabelText('集群选择器');
    await userEvent.click(screen.getByRole('button', { name: '确定' }));

    await waitFor(() => expect(mocks.updateEnvCluster).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByRole('button', { name: '确定' })).toBeEnabled());
    expect(screen.getByLabelText('集群选择器')).toBeVisible();
    expect(mocks.message).not.toHaveBeenCalledWith(expect.objectContaining({ theme: 'success' }));
    expect(mocks.store.current!.syncSavedEnv).not.toHaveBeenCalled();
  });
});
