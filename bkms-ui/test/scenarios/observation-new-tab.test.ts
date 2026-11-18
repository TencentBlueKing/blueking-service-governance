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
 * 场景级测试：应用管理观测数据的新标签页入口
 * ROUTES 将原有只读页面列为落选，本头记录新增交互的路径清单。
 * 覆盖（V = 6）
 * ① 初始视图 ② 切换环境时继续同步视图 ③ 在同一 APM 的有服务和无服务环境之间切换
 * ④ 无 APM 后切回并重建 iframe ⑤ 环境无 APM 实例 ⑥ 应用缺少观测配置
 * stub/mock：环境选择器与 iframe 模拟组件事件，APM 和服务接口及相关 store 用 mock，路由使用内存历史。
 */
import { nextTick, reactive } from 'vue';

import { getByRole, queryByRole } from '@testing-library/vue';
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import Observation from '~/pages/application/detail/observation.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

const monitorOrigin = 'https://monitor.example.com';
const linkName = '在新标签页打开';
const testEnv = { id: 'env-test', name: 'test' };
const prodEnv = { id: 'env-prod', name: 'prod' };
const deploySelection = reactive({ curEnvItem: testEnv });
const mocks = vi.hoisted(() => ({
  getEnvApm: vi.fn(),
  getServiceName: vi.fn(),
  sendSetParams: vi.fn(),
}));

vi.mock('bkui-vue', () => ({
  Button: { template: '<button><slot /></button>' },
  Exception: { template: '<div><slot name="description" /><slot /></div>' },
}));
vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: {
    GetApmServiceName: mocks.getServiceName,
    GetWorkspace: () => Promise.resolve({ bkSystems: { bkMonitorProjectID: '123' } }),
  },
}));
vi.mock('~/api/modules/v1', () => ({ BkintegrationsBkmonitorService: { getEnvApm: mocks.getEnvApm } }));
vi.mock('~/composables/use-env-manager', () => ({ envDetailLocation: vi.fn() }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/stores/deploy-env', () => ({ useDeployEnvStore: () => ({ currentEnv: testEnv.name }) }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'demo-space' }) }));
vi.mock('~/stores/trpc-deploy', () => ({ useTrpcDeployStore: () => deploySelection }));

const EnvSelectPanel = {
  name: 'EnvSelectPanel',
  emits: ['update:modelValue', 'update:item'],
  template: '<div />',
};
const MonitorIframe = {
  name: 'MonitorIframe',
  props: { url: { type: String, required: true } },
  emits: ['route-change'],
  methods: { sendSetParams: mocks.sendSetParams },
  template: '<iframe :src="url" />',
};

function monitorRouteChange(path: string, query: Record<string, string>) {
  const hash = `#${path}?${new URLSearchParams(query)}`;
  return { hash, href: `${monitorOrigin}/?bizId=123${hash}`, query };
}

async function mountObservation() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ component: Observation, path: '/' }],
  });
  await router.push({ path: '/', query: { env: testEnv.name } });
  const wrapper = mount(Observation, {
    global: {
      mocks: i18nGlobalMocks,
      plugins: [router],
      stubs: { EnvSelectPanel, MonitorIframe },
    },
  });
  await flushPromises();
  return {
    element: wrapper.get('div').element,
    async selectEnvironment(env: typeof testEnv) {
      const selector = wrapper.getComponent({ name: EnvSelectPanel.name });
      selector.vm.$emit('update:modelValue', env.name);
      selector.vm.$emit('update:item', env);
      await flushPromises();
    },
    wrapper,
  };
}

function readDestination(element: HTMLElement) {
  const link = getByRole(element, 'link', { name: linkName });
  const url = new URL(link.getAttribute('href') ?? '');
  const [path, hashQuery] = url.hash.split('?');
  return { link, path, query: new URLSearchParams(hashQuery), url };
}

enableAutoUnmount(afterEach);
afterEach(() => vi.unstubAllEnvs());
beforeEach(() => {
  vi.stubEnv('BK_MONITOR', monitorOrigin);
  vi.resetAllMocks();
  deploySelection.curEnvItem = testEnv;
  mocks.getEnvApm.mockImplementation(({ envID }: { envID: string }) =>
    Promise.resolve({ name: envID === testEnv.id ? 'test-apm' : 'prod-apm' }),
  );
  mocks.getServiceName.mockImplementation(({ envName }: { envName: string }) =>
    Promise.resolve({ serviceName: `${envName}-service` }),
  );
});

describe('观测数据：在新标签页打开', () => {
  it('当用户从初始视图打开新标签页时，应保留业务 ID 与观测参数', async () => {
    const { element } = await mountObservation();
    const { link, path, query, url } = readDestination(element);

    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'));
    expect(link).not.toHaveAttribute('title');
    expect(url.origin).toBe(monitorOrigin);
    expect(Object.fromEntries(url.searchParams)).toEqual({ bizId: '123' });
    expect(path).toBe('#/apm/service');
    expect(Object.fromEntries(query)).toMatchObject({
      dashboardId: 'service-default-overview',
      'filter-app_name': 'test-apm',
      'filter-service_name': 'test-service',
      from: 'now-1h',
      sceneId: 'apm_service',
      to: 'now',
    });
  });

  it('当用户切换环境并调整视图时，应保留目标环境与最新页签和时间', async () => {
    const { element, selectEnvironment, wrapper } = await mountObservation();
    const iframe = wrapper.getComponent({ name: MonitorIframe.name });
    const initialSrc = iframe.props('url');
    const view = {
      dashboardId: 'service-trace-overview',
      'filter-app_name': 'test-apm',
      'filter-service_name': 'test-service',
      from: '1700000000000',
      sceneType: 'trace',
      to: '1700003600000',
    };
    const viewWithInstance = { ...view, 'filter-bk_instance_id': 'test-pod' };
    iframe.vm.$emit('route-change', monitorRouteChange('/apm/service/trace', viewWithInstance));
    await flushPromises();
    const { path, query } = readDestination(element);
    expect(path).toBe('#/apm/service/trace');
    expect(Object.fromEntries(query)).toMatchObject(viewWithInstance);
    expect(iframe.props('url')).toBe(initialSrc);

    await selectEnvironment(prodEnv);
    expect(mocks.sendSetParams).toHaveBeenCalledWith({
      'filter-app_name': 'prod-apm',
      'filter-service_name': 'prod-service',
    });

    const prodFilters = {
      'filter-app_name': 'prod-apm',
      'filter-service_name': 'prod-service',
    };
    const pendingDestination = readDestination(element);
    expect(pendingDestination.path).toBe('#/apm/service/trace');
    expect(Object.fromEntries(pendingDestination.query)).toMatchObject({ ...view, ...prodFilters });
    expect(pendingDestination.query.has('filter-bk_instance_id')).toBe(false);

    iframe.vm.$emit('route-change', monitorRouteChange('/apm/service', { ...viewWithInstance, from: 'now-12h' }));
    const latestView = { dashboardId: 'service-log-overview', from: 'now-6h', sceneType: 'log' };
    iframe.vm.$emit(
      'route-change',
      monitorRouteChange('/apm/service/log', { ...viewWithInstance, ...latestView, 'filter-app_name': 'prod-apm' }),
    );
    await nextTick();
    const destinationWithoutAck = readDestination(element);
    expect(destinationWithoutAck.path).toBe('#/apm/service/log');
    expect(Object.fromEntries(destinationWithoutAck.query)).toMatchObject({ ...view, ...prodFilters, ...latestView });
    expect(destinationWithoutAck.query.has('filter-bk_instance_id')).toBe(false);

    const acknowledgedRoute = monitorRouteChange('/apm/service/trace', {
      ...view,
      ...prodFilters,
      'filter-bk_instance_id': 'prod-pod',
      from: 'now-3h',
    });
    iframe.vm.$emit('route-change', acknowledgedRoute);
    await flushPromises();
    expect(readDestination(element).url.hash).toBe(acknowledgedRoute.hash);
    expect(iframe.props('url')).toBe(initialSrc);
  });

  it('当用户在同一 APM 的有服务和无服务环境之间切换时，应使用目标环境的过滤条件', async () => {
    mocks.getEnvApm.mockResolvedValue({ name: 'shared-apm' });
    const { element, selectEnvironment, wrapper } = await mountObservation();
    const iframe = wrapper.getComponent({ name: MonitorIframe.name });
    const filters = { 'filter-app_name': 'shared-apm', 'filter-service_name': 'test-service' };
    mocks.getServiceName.mockResolvedValueOnce({ serviceName: '' });
    await selectEnvironment(prodEnv);

    iframe.vm.$emit('route-change', monitorRouteChange('/apm/service', { ...filters, from: 'now-6h' }));
    await nextTick();
    expect(Object.fromEntries(readDestination(element).query)).toMatchObject({
      ...filters,
      'filter-service_name': '',
      from: 'now-6h',
    });

    const applicationRoute = monitorRouteChange('/apm/application', {
      app_name: 'shared-apm',
      from: 'now-3h',
    });
    iframe.vm.$emit('route-change', applicationRoute);
    await flushPromises();
    expect(readDestination(element).url.hash).toBe(applicationRoute.hash);

    await selectEnvironment(testEnv);
    iframe.vm.$emit('route-change', applicationRoute);
    await nextTick();
    const destination = readDestination(element);
    expect(destination.path).toBe('#/apm/service');
    expect(Object.fromEntries(destination.query)).toMatchObject({ ...filters, from: 'now-3h' });
  });

  it('当用户从无 APM 环境切回有 APM 的环境时，应打开重新加载的观测视图', async () => {
    const { element, selectEnvironment, wrapper } = await mountObservation();
    const iframe = wrapper.getComponent({ name: MonitorIframe.name });
    const view = { from: 'now-6h', to: 'now' };
    iframe.vm.$emit('route-change', monitorRouteChange('/apm/service/trace', view));
    await flushPromises();
    expect(readDestination(element).path).toBe('#/apm/service/trace');

    mocks.getEnvApm.mockResolvedValueOnce(null);
    await selectEnvironment(prodEnv);
    expect(wrapper.findComponent({ name: MonitorIframe.name }).exists()).toBe(false);
    expect(queryByRole(element, 'link', { name: linkName })).toBeNull();

    await selectEnvironment(testEnv);
    const rebuiltIframe = wrapper.getComponent({ name: MonitorIframe.name });
    const destination = readDestination(element);
    expect(rebuiltIframe.element).not.toBe(iframe.element);
    expect(destination.path).toBe('#/apm/service');
    expect(destination.url.hash).toBe(new URL(rebuiltIframe.props('url')).hash);
    expect(Object.fromEntries(destination.query)).toMatchObject(view);
  });

  it.each([
    { configMissing: false, message: '当前环境未创建 APM 实例', state: '未创建 APM 实例的环境' },
    { configMissing: true, message: '尚未开启观测功能', state: '缺少观测配置的应用' },
  ])('当用户查看$state时，应隐藏新标签页入口', async ({ configMissing, message }) => {
    if (configMissing) mocks.getServiceName.mockRejectedValue({ error: { details: [{ code: 'APM_CONFIG_MISSING' }] } });
    else mocks.getEnvApm.mockResolvedValue(null);
    const { element, wrapper } = await mountObservation();

    expect(wrapper.findComponent({ name: MonitorIframe.name }).exists()).toBe(false);
    expect(queryByRole(element, 'link', { name: linkName })).toBeNull();
    expect(wrapper.text()).toContain(message);
  });
});
