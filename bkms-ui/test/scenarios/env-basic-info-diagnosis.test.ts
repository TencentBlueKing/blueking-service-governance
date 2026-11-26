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
 * 真实环境基本信息页的诊断摘要场景。
 * 保留页面与环境详情 Store；隔离 HTTP、路由、表单和非诊断重子组件。
 * 覆盖未恢复问题计数、无告警、无报告、失败、加载、保存刷新及跨环境迟到响应。
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BasicInfo from '~/pages/env/basic-info.vue';
import { useEnvDetailStore } from '~/stores/env-detail';
import { useFrontendFeaturesStore } from '~/stores/frontend-features';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { CheckItemOutput, ClusterReportOutput } from '~/@types/v1/bkintegrations-kubeinsight';
import type { EnvDetailOutput } from '~/@types/v1/env';

const mocks = vi.hoisted(() => ({
  getReport: vi.fn<() => Promise<ClusterReportOutput | undefined>>(),
  getEnv: vi.fn<() => Promise<EnvDetailOutput>>(),
  updateEnvBasicInfo: vi.fn(),
  updateEnvCluster: vi.fn(),
  push: vi.fn(),
}));

vi.mock('~/api/modules/v1', () => ({
  BkintegrationsKubeinsightService: { getLatestEnvReport: mocks.getReport },
  EnvService: {
    getEnv: mocks.getEnv,
    updateEnvBasicInfo: mocks.updateEnvBasicInfo,
    updateEnvCluster: mocks.updateEnvCluster,
  },
}));
vi.mock('~/api/modules/v1/frontend-features', () => ({
  FrontendFeaturesService: { getFrontendFeatures: vi.fn() },
}));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1', workspaceDetail: { bkSystems: { bkBCSProjectID: 'project-1' } } }),
}));
vi.mock('~/components/cluster-selector/use-cluster-selector', async () => {
  const { ref } = await import('vue');
  return { default: () => ({ loading: ref(false), clusterData: ref([]), getClusterList: vi.fn() }) };
});
vi.mock('~/composables/use-env-manager', () => ({ envTypeAvatarColorMap: {}, envTypeTagClassMap: {} }));
vi.mock('~/common/util', () => ({
  formatTimeByTimezone: (value: string) => value,
  parseCss: (value: number | string) => (typeof value === 'number' ? `${value}px` : value),
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({ push: mocks.push, route: { params: { space: 'ws-1', envId: 'env-1' } } });
});

// 保留按钮的原生 disabled/click 语义；表单只隔离校验，不复刻页面的保存逻辑。
/* 场景测试的多组件契约桩集中在此文件。 */
/* eslint-disable vue/one-component-per-file */
vi.mock('bkui-vue', async () => {
  const { defineComponent, h } = await import('vue');
  const slotComponent = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('span', slots.default?.()),
  });
  const buttonComponent = defineComponent({
    props: { disabled: Boolean },
    setup:
      (props, { slots }) =>
      () =>
        h('button', { disabled: props.disabled }, slots.default?.()),
  });
  const formComponent = Object.assign(
    defineComponent({
      setup(_, { expose, slots }) {
        expose({ validate: () => Promise.resolve() });
        return () => h('div', slots.default?.());
      },
    }),
    { FormItem: slotComponent },
  );
  return {
    Button: buttonComponent,
    Form: formComponent,
    DatePicker: slotComponent,
    Input: slotComponent,
    Message: vi.fn(),
    OverflowTitle: slotComponent,
    Select: Object.assign(slotComponent, { Option: slotComponent }),
    Tag: slotComponent,
  };
});
vi.mock('bkui-vue/lib/icon', async () => {
  const { defineComponent } = await import('vue');
  return { ExclamationCircleShape: defineComponent({ template: '<i />' }) };
});

vi.mock('~/pages/env/cluster-components/cluster-components.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/env/components/delete-env-action.vue', () => ({
  default: { template: '<div />', methods: { reset: vi.fn() } },
}));
vi.mock('~/pages/env/components/env-category-description.vue', () => ({ default: { template: '<span />' } }));
vi.mock('~/pages/env/components/project-selector/deployed-apps-warning.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/env/components/project-selector/project-selector.vue', () => ({ default: { template: '<span />' } }));
vi.mock('~/components/user-display.vue', () => ({
  default: { props: ['value'], template: '<span>{{ value }}</span>' },
}));
vi.mock('~/components/cluster-selector/cluster-selector.vue', () => ({ default: { template: '<span />' } }));
vi.mock('~/components/namespace-selector/namespace-selector.vue', () => ({ default: { template: '<span />' } }));

function deferred<T>() {
  let resolve: (value: T) => void = () => {
    throw new Error('Promise resolver is not initialized');
  };
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}

function environment(id = 'env-1', hasCluster = true): EnvDetailOutput {
  return {
    id,
    name: id,
    displayName: `环境 ${id}`,
    type: 'test',
    creator: 'tester',
    cluster: hasCluster
      ? { clusterID: `cluster-${id}`, projectCode: 'project-1', namespace: 'namespace-1' }
      : undefined,
  };
}

function renderPage(env = environment()) {
  useEnvDetailStore().primeCurrentEnv(env, 'ws-1');
  return render(BasicInfo, {
    props: { env: env.id, workspace: 'ws-1' },
    global: {
      mocks: i18nGlobalMocks,
      stubs: {
        Skeleton: { props: ['loading'], template: '<div><slot v-if="!loading" /><slot v-else name="loading" /></div>' },
        BkmsContent: {
          props: ['title', 'showEditIcon'],
          emits: ['edit'],
          template:
            '<section><h2>{{ title }}</h2><button v-if="showEditIcon" @click="$emit(\'edit\')">编辑集群资源</button><slot /></section>',
        },
        FieldItem: {
          props: ['fieldValue', 'value'],
          template: '<span>{{ fieldValue }}：{{ value }}<slot name="value" /></span>',
        },
      },
      directives: { bkloading: {}, 'bk-tooltips': {} },
    },
  });
}

function report(items: CheckItemOutput[] = []): ClusterReportOutput {
  return { abnormalItems: items };
}

function summary() {
  return screen.getByRole('status', { name: '集群诊断' });
}

beforeEach(() => {
  setActivePinia(createPinia());
  vi.resetAllMocks();
  mocks.getReport.mockResolvedValue(report());
  mocks.getEnv.mockResolvedValue(environment());
  mocks.updateEnvBasicInfo.mockResolvedValue({});
  mocks.updateEnvCluster.mockResolvedValue({});
});

describe('环境基本信息：诊断摘要', () => {
  it('按未恢复问题条目统计致命和预警，忽略 recordCount 与恢复条目', async () => {
    mocks.getReport.mockResolvedValue(
      report([
        { level: 'RISK', recordCount: 15 },
        { level: 'RISK', recordCount: 8, recovered: false },
        { level: 'WARN', recordCount: 20 },
        { level: 'RISK', recovered: true },
        { level: 'WARN', recovered: true },
        { level: 'RECOVERED', recordCount: 99 },
        { level: 'INFO' },
      ]),
    );
    renderPage();
    const details = await screen.findByRole('button', { name: '查看详情' });
    expect(summary()).toHaveTextContent(/致命\s*2/);
    expect(summary()).toHaveTextContent(/预警\s*1/);
    await fireEvent.click(details);
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'clusterHealthDiagnosis',
      params: { envId: 'env-1', space: 'ws-1' },
    });
  });

  it.each([
    { level: 'RISK', risk: 1, warn: 0 },
    { level: 'WARN', risk: 0, warn: 1 },
  ])('仅有 $level 时另一等级显示零', async ({ level, risk, warn }) => {
    mocks.getReport.mockResolvedValue(report([{ level }]));
    renderPage();
    await screen.findByRole('button', { name: '查看详情' });
    expect(summary()).toHaveTextContent(new RegExp(`致命\\s*${risk}`));
    expect(summary()).toHaveTextContent(new RegExp(`预警\\s*${warn}`));
  });

  it.each([
    {
      label: '全部恢复',
      items: [{ level: 'RISK', recovered: true }, { level: 'WARN', recovered: true }, { level: 'RECOVERED' }],
    },
    { label: '空报告', items: [] },
    { label: '仅 INFO', items: [{ level: 'INFO', recordCount: 100 }] },
  ])('$label 显示未发现异常，查看报告沿用诊断详情路由', async ({ items }) => {
    mocks.getReport.mockResolvedValue(report(items));
    renderPage();
    await screen.findByText('未发现异常');
    const button = within(summary()).getByRole('button', { name: '查看报告' });
    expect(button).toBeEnabled();
    expect(within(summary()).queryByText(/致命/)).not.toBeInTheDocument();
    await fireEvent.click(button);
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'clusterHealthDiagnosis',
      params: { envId: 'env-1', space: 'ws-1' },
    });
  });

  it.each([404, 500])('%s 错误隐藏整个诊断摘要和查看链接，不误显示健康', async status => {
    mocks.getReport.mockRejectedValue({ status, error: { message: '报告服务不可用' } });
    renderPage();
    await flushPromises();
    expect(screen.queryByRole('status', { name: '集群诊断' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /查看详情|查看报告/ })).not.toBeInTheDocument();
    expect(screen.queryByText('未发现异常')).not.toBeInTheDocument();
  });

  it('无集群配置时不请求报告，也不把未知状态显示为健康', async () => {
    renderPage(environment('env-1', false));
    await flushPromises();
    expect(screen.queryByRole('status', { name: '集群诊断' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /查看详情|查看报告/ })).not.toBeInTheDocument();
    expect(mocks.getReport).not.toHaveBeenCalled();
    expect(screen.queryByText('未发现异常')).not.toBeInTheDocument();
  });

  it('成功响应没有报告对象时隐藏摘要，与空异常数组的健康报告区分', async () => {
    mocks.getReport.mockResolvedValue(undefined);
    renderPage();
    await flushPromises();
    expect(screen.queryByRole('status', { name: '集群诊断' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /查看详情|查看报告/ })).not.toBeInTheDocument();
    expect(screen.queryByText('未发现异常')).not.toBeInTheDocument();
  });

  it('报告加载完成前显示忙碌状态，不提前显示未发现异常', async () => {
    const pending = deferred<ClusterReportOutput>();
    mocks.getReport.mockReturnValue(pending.promise);
    renderPage();
    expect(summary()).toHaveAttribute('aria-busy', 'true');
    expect(screen.queryByText('未发现异常')).not.toBeInTheDocument();
    pending.resolve(report());
    await screen.findByText('未发现异常');
    expect(summary()).toHaveAttribute('aria-busy', 'false');
  });

  it('保存后刷新报告期间清除旧计数，成功后展示新的健康状态', async () => {
    const refresh = deferred<ClusterReportOutput>();
    mocks.getReport.mockResolvedValueOnce(report([{ level: 'RISK' }])).mockReturnValueOnce(refresh.promise);
    renderPage();
    await screen.findByRole('button', { name: '查看详情' });
    await fireEvent.click(screen.getByRole('button', { name: /^编辑$/ }));
    await fireEvent.click(screen.getByRole('button', { name: /^确定$/ }));
    await waitFor(() => expect(mocks.getReport).toHaveBeenCalledTimes(2));
    expect(summary()).toHaveAttribute('aria-busy', 'true');
    expect(summary()).not.toHaveTextContent(/致命\s*1/);
    refresh.resolve(report());
    await screen.findByText('未发现异常');
  });

  it.each([
    { previous: '告警', items: [{ level: 'RISK' }] },
    { previous: '健康', items: [] },
  ])('原有$previous结果刷新失败时隐藏摘要，不残留上次报告', async ({ items }) => {
    mocks.getReport
      .mockResolvedValueOnce(report(items))
      .mockRejectedValueOnce({ status: 500, error: { message: '刷新报告失败' } });
    renderPage();
    await waitFor(() => expect(summary()).toHaveAttribute('aria-busy', 'false'));
    await fireEvent.click(screen.getByRole('button', { name: /^编辑$/ }));
    await fireEvent.click(screen.getByRole('button', { name: /^确定$/ }));
    await waitFor(() => expect(mocks.getReport).toHaveBeenCalledTimes(2));
    await flushPromises();
    expect(screen.queryByRole('status', { name: '集群诊断' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /查看详情|查看报告/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/致命\s*1/)).not.toBeInTheDocument();
    expect(screen.queryByText('未发现异常')).not.toBeInTheDocument();
  });

  it('切换环境时清除旧计数，新环境等待期间不沿用旧报告', async () => {
    const nextReport = deferred<ClusterReportOutput>();
    mocks.getReport.mockResolvedValueOnce(report([{ level: 'WARN' }])).mockReturnValueOnce(nextReport.promise);
    const { rerender } = renderPage();
    await screen.findByRole('button', { name: '查看详情' });
    useEnvDetailStore().primeCurrentEnv(environment('env-2'), 'ws-1');
    await rerender({ env: 'env-2', workspace: 'ws-1' });
    await waitFor(() => expect(mocks.getReport).toHaveBeenCalledTimes(2));
    expect(summary()).toHaveAttribute('aria-busy', 'true');
    expect(summary()).not.toHaveTextContent(/预警\s*1/);
    nextReport.resolve(report());
    await screen.findByText('未发现异常');
  });

  it('旧环境迟到响应不覆盖新环境报告，链接也指向新环境', async () => {
    const oldReport = deferred<ClusterReportOutput>();
    const newReport = deferred<ClusterReportOutput>();
    mocks.getReport.mockReturnValueOnce(oldReport.promise).mockReturnValueOnce(newReport.promise);
    const { rerender } = renderPage();
    useEnvDetailStore().primeCurrentEnv(environment('env-2'), 'ws-1');
    await rerender({ env: 'env-2', workspace: 'ws-1' });
    await waitFor(() => expect(mocks.getReport).toHaveBeenCalledTimes(2));
    newReport.resolve(report());
    await screen.findByText('未发现异常');
    oldReport.resolve(report([{ level: 'RISK' }]));
    await flushPromises();
    expect(summary()).toHaveTextContent('未发现异常');
    expect(summary()).not.toHaveTextContent(/致命/);
    await fireEvent.click(within(summary()).getByRole('button', { name: '查看报告' }));
    expect(mocks.push).toHaveBeenLastCalledWith({
      name: 'clusterHealthDiagnosis',
      params: { envId: 'env-2', space: 'ws-1' },
    });
  });

  it('开关关闭时隐藏整个摘要并停止初始及保存后的报告请求', async () => {
    useFrontendFeaturesStore().enableKubeInsight = false;
    renderPage();
    await screen.findByText('集群：cluster-env-1');
    expect(screen.queryByRole('status', { name: '集群诊断' })).not.toBeInTheDocument();
    await fireEvent.click(screen.getByRole('button', { name: /^编辑$/ }));
    await fireEvent.click(screen.getByRole('button', { name: /^确定$/ }));
    await waitFor(() => expect(mocks.updateEnvBasicInfo).toHaveBeenCalled());
    await flushPromises();
    expect(mocks.getReport).not.toHaveBeenCalled();
  });
});
