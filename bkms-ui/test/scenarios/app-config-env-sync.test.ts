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

import { reactive } from 'vue';

import { flushPromises, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DeployConfig from '~/pages/application/detail/app-config/deploy-config.vue';

import type { EnvOutput } from '~/@types/v1/env';

const STORAGE_KEY = 'bkms_app_config_deploy_env';
const ENVS: EnvOutput[] = [
  { name: 'dev', displayName: 'Development', type: 'development' },
  { name: 'prod', displayName: 'Production', type: 'production' },
];
const appState = reactive({ appID: 'app-a', runtime: null });
const spaceState = reactive({ currentSpace: 'space-a' });
const routeState = reactive<{ query: Record<string, string | string[]> }>({ query: {} });
const mocks = vi.hoisted(() => ({
  listEnvs: vi.fn(),
  overview: vi.fn(),
  replace: vi.fn(),
  resources: vi.fn(),
  strategy: vi.fn(),
  metadata: vi.fn(),
  network: vi.fn(),
}));

vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => appState }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => spaceState }));
vi.mock('vue-router', () => ({ useRoute: () => routeState, useRouter: () => ({ replace: mocks.replace }) }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('~/composables/use-app-runtime', () => ({ isRuntimeModSupported: () => false }));
vi.mock('~/api/modules/v1', () => ({
  EnvService: { listAppEnvs: mocks.listEnvs },
  AppSpecService: { getAppSpecOverview: mocks.overview },
}));
vi.mock('bkui-vue', () => ({ Popover: { template: '<div><slot /></div>' } }));
vi.mock('~/components/skeleton/skeleton-layout', () => ({
  default: { shape: { template: '<div />' }, formItem: { template: '<div />' } },
}));
vi.mock('~/components/skeleton/skeleton.vue', () => ({
  default: { name: 'Skeleton', template: '<div><slot /></div>' },
}));
vi.mock('~/pages/application/detail/app-config/env-perspective-select.vue', () => ({
  default: { name: 'EnvPerspectiveSelect', props: ['envList', 'modelValue', 'modifiedEnvNames'], template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/resources-form.vue', () => ({
  default: {
    name: 'ResourcesForm',
    props: ['currentEnv'],
    template: '<div />',
    methods: { handleEnvChange: (...args: unknown[]) => mocks.resources(...args) },
  },
}));
vi.mock('~/pages/application/detail/app-config/components/update-strategy-form.vue', () => ({
  default: {
    name: 'UpdateStrategyForm',
    props: ['currentEnv'],
    template: '<div />',
    methods: { handleEnvChange: (...args: unknown[]) => mocks.strategy(...args) },
  },
}));
vi.mock('~/pages/application/detail/app-config/components/metadata-config.vue', () => ({
  default: {
    name: 'MetadataConfig',
    props: ['currentEnv'],
    template: '<div />',
    methods: { handleEnvChange: (...args: unknown[]) => mocks.metadata(...args) },
  },
}));
vi.mock('~/pages/application/detail/app-config/components/network-access.vue', () => ({
  default: {
    name: 'NetworkAccess',
    props: ['currentEnv'],
    template: '<div />',
    methods: { handleEnvChange: (...args: unknown[]) => mocks.network(...args), isSaving: () => false },
  },
}));
vi.mock('~/pages/application/detail/app-config/components/program-config.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/lifecycle.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/app-config/components/health-probe.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/dev-mode-form.vue', () => ({
  default: { template: '<div />' },
}));

let wrapper: ReturnType<typeof shallowMount<typeof DeployConfig>> | undefined;

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}

async function openPage(selections: Record<string, string> = {}) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(selections));
  wrapper = shallowMount(DeployConfig, {
    global: {
      mocks: { $t: (key: string) => key },
      stubs: { ResourcesForm: false, UpdateStrategyForm: false, MetadataConfig: false, NetworkAccess: false },
    },
  });
  await flushPromises();
  return wrapper;
}

function readStoredSelections(): Record<string, string> {
  return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
}

function selectedEnv() {
  return wrapper!.get('[data-testid="deploy-config-content"]').attributes('data-env-name');
}

async function selectEnv(envName: string) {
  wrapper!.getComponent({ name: 'EnvPerspectiveSelect' }).vm.$emit('change', envName);
  await flushPromises();
}

beforeEach(() => {
  vi.resetAllMocks();
  localStorage.clear();
  appState.appID = 'app-a';
  spaceState.currentSpace = 'space-a';
  routeState.query = {};
  mocks.listEnvs.mockResolvedValue(ENVS);
  mocks.overview.mockResolvedValue({ configuredEnvs: [] });
  mocks.resources.mockResolvedValue(true);
  mocks.strategy.mockResolvedValue(true);
  mocks.metadata.mockResolvedValue(true);
  mocks.network.mockResolvedValue(true);
  mocks.replace.mockImplementation(({ query }) => {
    routeState.query = query;
    return Promise.resolve();
  });
});

afterEach(() => {
  wrapper?.unmount();
  wrapper = undefined;
});

describe('部署配置与配置文件共用环境记忆', () => {
  it('没有记忆时显示默认配置，有记忆时恢复对应环境', async () => {
    await openPage();
    expect(selectedEnv()).toBe('__default__');
    wrapper!.unmount();

    await openPage({ 'space-a:app-a': 'prod' });
    expect(selectedEnv()).toBe('prod');
    expect(mocks.resources).toHaveBeenLastCalledWith(expect.objectContaining({ name: 'prod', isDefault: false }));
  });

  it('手动选择环境写入共用记忆，选择默认仅清理当前应用', async () => {
    await openPage({ 'space-a:app-b': 'prod' });
    await selectEnv('dev');
    expect(selectedEnv()).toBe('dev');
    expect(readStoredSelections()).toEqual({ 'space-a:app-a': 'dev', 'space-a:app-b': 'prod' });

    await selectEnv('__default__');
    expect(selectedEnv()).toBe('__default__');
    expect(readStoredSelections()).toEqual({ 'space-a:app-b': 'prod' });
  });

  it('路由环境优先于共用记忆，使用后只移除 envName 参数', async () => {
    routeState.query = { envName: 'dev', activeTab: 'deploy-config' };
    await openPage({ 'space-a:app-a': 'prod' });
    expect(selectedEnv()).toBe('dev');
    expect(readStoredSelections()['space-a:app-a']).toBe('dev');
    expect(mocks.replace).toHaveBeenCalledWith({ query: { activeTab: 'deploy-config' } });
  });

  it('任一配置区取消环境切换后保留原环境和记忆', async () => {
    await openPage({ 'space-a:app-a': 'prod' });
    mocks.resources.mockResolvedValueOnce(false);
    await selectEnv('dev');
    expect(selectedEnv()).toBe('prod');
    expect(readStoredSelections()['space-a:app-a']).toBe('prod');
    expect(mocks.network).toHaveBeenLastCalledWith(expect.objectContaining({ name: 'prod' }));
  });

  it('环境列表请求失败仅本地显示默认，保留共用记忆', async () => {
    mocks.listEnvs.mockRejectedValueOnce(new Error('network unavailable'));
    await openPage({ 'space-a:app-a': 'prod' });
    expect(selectedEnv()).toBe('__default__');
    expect(readStoredSelections()['space-a:app-a']).toBe('prod');
  });

  it('成功获取环境列表并确认环境不存在时清理失效记忆', async () => {
    await openPage({ 'space-a:app-a': 'removed', 'space-b:app-a': 'prod' });
    expect(selectedEnv()).toBe('__default__');
    expect(readStoredSelections()).toEqual({ 'space-b:app-a': 'prod' });
  });

  it('切换应用或空间时分别恢复各自环境，不覆盖其他记忆', async () => {
    const selections = { 'space-a:app-a': 'prod', 'space-a:app-b': 'dev', 'space-b:app-b': 'prod' };
    await openPage(selections);
    appState.appID = 'app-b';
    await flushPromises();
    expect(selectedEnv()).toBe('dev');

    spaceState.currentSpace = 'space-b';
    await flushPromises();
    expect(selectedEnv()).toBe('prod');
    expect(readStoredSelections()).toEqual(selections);
  });

  it('切换应用后旧环境确认迟到时不覆盖新应用环境或记忆', async () => {
    const selections = { 'space-a:app-a': 'prod', 'space-a:app-b': 'prod' };
    await openPage(selections);
    const oldConfirmation = deferred<boolean>();
    mocks.resources.mockReturnValueOnce(oldConfirmation.promise);
    await selectEnv('dev');

    appState.appID = 'app-b';
    await flushPromises();
    oldConfirmation.resolve(true);
    await flushPromises();
    expect(selectedEnv()).toBe('prod');
    expect(readStoredSelections()).toEqual(selections);
  });

  it('A→B→A 后最初 A 的迟到列表不覆盖最新 A，也不清理其记忆', async () => {
    const oldEnvs = deferred<EnvOutput[]>();
    mocks.listEnvs.mockReturnValueOnce(oldEnvs.promise);
    const selections = { 'space-a:app-a': 'prod', 'space-a:app-b': 'dev' };
    await openPage(selections);
    appState.appID = 'app-b';
    await flushPromises();
    appState.appID = 'app-a';
    await flushPromises();
    expect(selectedEnv()).toBe('prod');

    oldEnvs.resolve([]);
    await flushPromises();
    expect(selectedEnv()).toBe('prod');
    expect(readStoredSelections()).toEqual(selections);
  });

  it('页签卸载后迟到的初始化不写环境记忆或消费路由参数', async () => {
    const oldEnvs = deferred<EnvOutput[]>();
    mocks.listEnvs.mockReturnValueOnce(oldEnvs.promise);
    routeState.query = { envName: 'dev' };
    await openPage({ 'space-a:app-a': 'prod' });
    wrapper!.unmount();
    wrapper = undefined;
    oldEnvs.resolve(ENVS);
    await flushPromises();
    expect(readStoredSelections()['space-a:app-a']).toBe('prod');
    expect(mocks.replace).not.toHaveBeenCalled();
  });
});
