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

import type { Component } from 'vue';
import { defineComponent, effectScope, nextTick, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EnvGroupSelector from '~/components/env-group-selector.vue';
import EnvSelectPanel from '~/components/env-select-panel/env-select-panel.vue';
import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
import { useDeployPrecheck } from '~/pages/application/detail/deploy/use-deploy-precheck';
import { useAppDetail } from '~/stores/app-detail';
import { useDeployEnvStore } from '~/stores/deploy-env';

import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
import type { EnvOutput } from '~/@types/v1/env';

const mocks = vi.hoisted(() => ({
  update: vi.fn(),
  get: vi.fn(),
  initialGet: vi.fn(),
  list: vi.fn(),
  statuses: vi.fn(),
  message: vi.fn(),
  federation: vi.fn(),
  envVars: vi.fn(),
}));
vi.mock('~/api/modules/v1/app', () => ({
  AppService: { updateAppVisibleEnvs: mocks.update, getApp: mocks.get, getAppDeployStatuses: mocks.statuses },
}));
vi.mock('~/api/modules/v1/env', () => ({ EnvService: { listAppEnvs: mocks.list } }));
vi.mock('~/api/modules/bkmsserver', () => ({ ApiServerService: { GetApp: mocks.initialGet } }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'space-a' }) }));
vi.mock('~/pages/application/detail/deploy/use-federation-resource-precheck', () => ({
  useFederationResourcePrecheck: () => ({ check: mocks.federation }),
}));
vi.mock('~/pages/application/detail/deploy/use-env-var-precheck', () => ({
  useEnvVarPrecheck: () => ({ check: mocks.envVars }),
}));
vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  useI18n: () => ({
    t: (text: string, params: unknown[] = []) => text.replace(/\{(\d+)\}/g, (_, i) => String(params[Number(i)])),
    te: () => true,
  }),
}));
vi.mock('bkui-vue', async importOriginal => ({ ...(await importOriginal<object>()), Message: mocks.message }));

const env = (name: string, extra: Partial<EnvOutput> = {}): EnvOutput => ({
  id: name,
  name,
  displayName: name.toUpperCase(),
  kind: 'standard',
  type: 'test',
  status: 'Ready',
  ...extra,
});
const environments = [
  env('test'),
  env('prod', { type: 'production' }),
  env('feature-a', { kind: 'feature', ownerAppID: 'app-a', sourceEnvID: 'test' }),
  env('not-ready', { status: 'NotReady' }),
];
const detail = (visibleEnvNames: string[] = []) => ({ id: 'app-a', name: 'app-a', visibleEnvNames });
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}

// 保留 Popover 的外部点击关闭契约；在 Vue 更新后检查目标，覆盖浏览器事件冒泡间的 DOM 替换。
const PopoverStub = defineComponent({
  name: 'Popover',
  emits: ['after-show', 'after-hidden'],
  data: () => ({ open: false }),
  mounted() {
    document.addEventListener('click', this.handleDocumentClick);
  },
  beforeUnmount() {
    document.removeEventListener('click', this.handleDocumentClick);
  },
  methods: {
    handleDocumentClick(event: MouseEvent) {
      void nextTick(() => {
        if (this.open && this.$el && event.target instanceof Node && !this.$el.contains(event.target)) {
          this.hide();
        }
      });
    },
    hide() {
      this.open = false;
      this.$emit('after-hidden');
    },
    show() {
      this.open = true;
      this.$emit('after-show');
    },
  },
  template:
    '<div><div data-testid="env-trigger" @click="show"><slot /></div><div v-if="open" data-testid="env-panel"><slot name="content" /><button @click="hide">关闭下拉</button></div></div>',
});

beforeEach(() => {
  vi.clearAllMocks();
  setActivePinia(createPinia());
  const store = useAppDetail();
  store.updateAppID('app-a');
  store.appDetail = detail();
  mocks.initialGet.mockImplementation(() => Promise.resolve(store.appDetail));
  mocks.get.mockImplementation(() => Promise.resolve(store.appDetail));
  mocks.update.mockResolvedValue({});
  mocks.list.mockResolvedValue(environments);
  mocks.statuses.mockResolvedValue([]);
  mocks.federation.mockResolvedValue([]);
  mocks.envVars.mockResolvedValue({});
});

async function openPanel(props: Record<string, unknown> = {}, component: Component = EnvSelectPanel) {
  const result = render(component, {
    props: { enableVisibleEnvSettings: true, filterVisibleEnvs: true, ...props },
    global: {
      mocks: {
        $t: (text: string, params: unknown[] = []) => text.replace(/\{(\d+)\}/g, (_, i) => String(params[Number(i)])),
      },
      stubs: { Popover: PopoverStub, Select: true, StatusDotIcon: true, ColorIcon: true, OverflowTags: true },
      directives: {
        'bk-tooltips': {
          mounted(el, binding) {
            el.setAttribute('data-tip', binding.value?.content || '');
          },
          updated(el, binding) {
            el.setAttribute('data-tip', binding.value?.content || '');
          },
        },
      },
    },
  });
  await userEvent.click(screen.getByTestId('env-trigger'));
  await waitFor(() => expect(screen.getByRole('button', { name: '设置可见环境' })).toBeEnabled());
  return result;
}

async function openPollingPanel() {
  const panel = ref<InstanceType<typeof EnvSelectPanel>>();
  await openPanel(
    {},
    {
      components: { EnvSelectPanel },
      setup: () => ({ panel }),
      template: '<EnvSelectPanel ref="panel" />',
    },
  );
  return panel.value!;
}

/** 组件契约与 store 单测：名单语义、异步隔离、草稿与选择联动；仅替换 Popover 定位动画和图标。 */
describe('应用可见环境：名单语义与保存', () => {
  it.each(['trpc', 'taf'] as const)('当 %s 应用预检名单外环境时，应阻止部署且本应用特性环境仍可通过', async type => {
    const store = useAppDetail();
    store.updateAppType(type);
    store.appDetail = detail(['test']);
    const scope = effectScope();
    const precheck = scope.run(() => useDeployPrecheck())!;
    try {
      expect(await precheck.precheck(environments[1])).toBe(false);
      expect(mocks.federation).not.toHaveBeenCalled();
      expect(mocks.envVars).not.toHaveBeenCalled();
      expect(await precheck.precheck(environments[2])).toBe(true);
      expect(mocks.envVars).toHaveBeenCalledWith('feature-a');
    } finally {
      scope.stop();
    }
  });
  it('只放行名单内标准环境和本应用特性环境，空名单恢复不限制', () => {
    const store = useAppDetail();
    store.appDetail = detail(['test']);
    const { isEnvVisible } = useAppVisibleEnvs();
    expect(environments.filter(isEnvVisible).map(item => item.name)).toEqual(['test', 'feature-a']);
    expect(isEnvVisible(env('foreign-feature', { kind: 'feature', ownerAppID: 'other-app' }))).toBe(false);
    store.appDetail = detail();
    expect(isEnvVisible(env('new-env'))).toBe(true);
    store.appDetail = detail(['test', 'prod']);
    expect(isEnvVisible(env('new-env'))).toBe(false);
    store.updateAppID('app-b');
    expect(isEnvVisible(env('prod'))).toBe(true);
  });

  it('整体替换名单并重新获取详情；恢复全部明确提交空数组', async () => {
    const store = useAppDetail();
    await store.updateVisibleEnvs([]);
    expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: [] });
    expect(mocks.get).toHaveBeenCalledWith({ appID: 'app-a' }, { interceptorErr: false });
  });

  it('保存失败不修改原名单；详情刷新失败保留已保存名单', async () => {
    const store = useAppDetail();
    store.appDetail = detail(['prod']);
    mocks.update.mockRejectedValueOnce(new Error('forbidden'));
    await expect(store.updateVisibleEnvs(['test'])).rejects.toThrow('forbidden');
    expect(store.appDetail?.visibleEnvNames).toEqual(['prod']);
    mocks.get.mockRejectedValueOnce(new Error('offline'));
    expect(await store.updateVisibleEnvs(['test'])).toBe('refresh-failed');
    expect(store.appDetail?.visibleEnvNames).toEqual(['test']);
  });

  it('应用 A→B→A 后不接受先前保存的迟到响应', async () => {
    const store = useAppDetail();
    const pending = deferred<object>();
    mocks.update.mockReturnValueOnce(pending.promise);
    const saving = store.updateVisibleEnvs(['prod']);
    store.updateAppID('app-b');
    store.updateAppID('app-a');
    store.appDetail = detail(['test']);
    pending.resolve({});
    expect(await saving).toBe('stale');
    expect(store.appDetail?.visibleEnvNames).toEqual(['test']);
    expect(mocks.get).not.toHaveBeenCalled();
  });

  it.each(['切换应用', '清空应用', '重置 store'])('%s 后无新请求时，立即释放详情 loading', async action => {
    const store = useAppDetail();
    const pending = deferred<ReturnType<typeof detail>>();
    mocks.initialGet.mockReturnValueOnce(pending.promise);
    const fetching = store.fetchAppDetail();
    expect(store.loading).toBe(true);

    if (action === '重置 store') store.reset();
    else store.updateAppID(action === '切换应用' ? 'app-b' : '');
    expect.soft(store.loading).toBe(false);

    pending.resolve(detail());
    expect(await fetching).toBeNull();
    expect(store.loading).toBe(false);
  });

  it('旧应用详情请求结束时，不释放新应用请求的 loading', async () => {
    const store = useAppDetail();
    const previous = deferred<ReturnType<typeof detail>>();
    const latest = deferred<ReturnType<typeof detail>>();
    mocks.initialGet.mockReturnValueOnce(previous.promise).mockReturnValueOnce(latest.promise);
    const previousFetching = store.fetchAppDetail();
    store.updateAppID('app-b');
    const latestFetching = store.fetchAppDetail();
    expect(store.loading).toBe(true);

    previous.resolve(detail());
    expect(await previousFetching).toBeNull();
    expect(store.loading).toBe(true);

    latest.resolve({ ...detail(), id: 'app-b' });
    await latestFetching;
    expect(store.loading).toBe(false);
    expect(store.appDetail?.id).toBe('app-b');
  });

  it('保存之前的详情请求晚返回也不会恢复旧名单', async () => {
    const store = useAppDetail();
    const pending = deferred<ReturnType<typeof detail>>();
    mocks.initialGet.mockReturnValueOnce(pending.promise);
    const fetching = store.fetchAppDetail();
    await store.updateVisibleEnvs(['test']);
    pending.resolve(detail(['prod']));
    await fetching;
    expect(store.appDetail?.visibleEnvNames).toEqual(['test']);
  });
});

describe('可见环境下拉交互', () => {
  it('已有当前应用详情时，面板初始化应复用详情和可见名单', async () => {
    const store = useAppDetail();
    store.appDetail = detail(['test']);
    const previousDetail = store.appDetail;
    mocks.initialGet.mockResolvedValue(detail(['test']));
    await openPanel();
    expect(mocks.initialGet).not.toHaveBeenCalled();
    expect(store.appDetail).toBe(previousDetail);
    expect(screen.queryByText('PROD')).not.toBeInTheDocument();
  });

  it('已有当前应用详情时，面板刷新不应因详情接口异常丢失名单限制', async () => {
    const store = useAppDetail();
    store.appDetail = detail(['test']);
    const { isRestricted, checkDeployEnv } = useAppVisibleEnvs();
    const panel = await openPollingPanel();
    const previousDetail = store.appDetail;
    mocks.initialGet.mockClear();
    mocks.initialGet.mockRejectedValue(new TypeError('offline'));
    await panel.refresh();
    expect(isRestricted.value).toBe(true);
    expect(store.appDetail).toBe(previousDetail);
    expect(checkDeployEnv(environments[1])).toBe(false);
    expect(screen.queryByText('PROD')).not.toBeInTheDocument();
    expect(mocks.initialGet).not.toHaveBeenCalled();
  });

  it.each([false, true])('并发加载环境时，仅最新请求更新列表和结束 loading（最新先返回：%s）', async latestFirst => {
    const panel = await openPollingPanel();
    const previous = deferred<EnvOutput[]>();
    const latest = deferred<EnvOutput[]>();
    mocks.list.mockReturnValueOnce(previous.promise).mockReturnValueOnce(latest.promise);
    const previousRefresh = panel.refreshEnvList();
    const latestRefresh = panel.refreshEnvList();
    await nextTick();
    const settingsButton = screen.getByRole('button', { name: '设置可见环境' });
    expect(settingsButton).toBeDisabled();

    if (latestFirst) {
      latest.resolve([env('latest')]);
      await latestRefresh;
      await nextTick();
      expect(settingsButton).toBeEnabled();
      previous.resolve([env('previous')]);
    } else {
      previous.resolve([env('previous')]);
      await previousRefresh;
      await nextTick();
      expect(settingsButton).toBeDisabled();
      latest.resolve([env('latest')]);
    }

    await Promise.all([previousRefresh, latestRefresh]);
    await nextTick();
    expect(settingsButton).toBeEnabled();
    expect(screen.getByText('LATEST')).toBeVisible();
    expect(screen.queryByText('PREVIOUS')).not.toBeInTheDocument();
  });

  it.each(['未加载', '属于其他应用'])('应用详情%s时，面板应获取当前应用详情后再使用可见名单', async state => {
    const store = useAppDetail();
    store.appDetail = state === '未加载' ? null : { ...detail(['prod']), id: 'app-b' };
    mocks.initialGet.mockResolvedValue(detail(['test']));
    await openPanel();
    expect(mocks.initialGet).toHaveBeenCalledExactlyOnceWith({ appID: 'app-a' });
    expect(store.appDetail?.visibleEnvNames).toEqual(['test']);
    expect(screen.queryByText('PROD')).not.toBeInTheDocument();
  });

  it.each([
    ['设置可见环境', false],
    ['设置可见环境', true],
    ['保存', false],
    ['保存', true],
  ] as const)('%s 期间并发刷新部署状态（刷新先返回：%s），操作仍应完成并采用最新状态', async (action, refreshFirst) => {
    useAppDetail().appDetail = detail(['test']);
    const panel = await openPollingPanel();
    if (action === '保存') {
      await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    }
    const pending = deferred<AppDeployedEnvOutputObj[]>();
    const refresh = deferred<AppDeployedEnvOutputObj[]>();
    mocks.statuses.mockReturnValueOnce(pending.promise).mockReturnValueOnce(refresh.promise);
    await userEvent.click(screen.getByRole('button', { name: action }));
    const refreshing = panel.refreshDeployStatuses();
    if (refreshFirst) {
      refresh.resolve([{ id: 'prod', deployStatus: 'deployed' }]);
      await refreshing;
      pending.resolve([]);
    } else {
      pending.resolve([]);
      await flushPromises();
      expect(mocks.update).not.toHaveBeenCalled();
      refresh.resolve([{ id: 'prod', deployStatus: 'deployed' }]);
      await refreshing;
    }
    await flushPromises();
    if (action === '保存') {
      expect(mocks.update).toHaveBeenCalledExactlyOnceWith({ appID: 'app-a', visibleEnvNames: ['test', 'prod'] });
    } else {
      expect(screen.getByRole('button', { name: '保存' })).toBeVisible();
      const row = screen.getByText('PROD').closest('[data-tip]')!;
      expect(within(row as HTMLElement).getByRole('checkbox')).toBeChecked();
      expect(within(row as HTMLElement).getByRole('checkbox')).toBeDisabled();
    }
  });

  it('保存被并发刷新顶替后，最新状态请求失败应保留草稿并允许重试', async () => {
    useAppDetail().appDetail = detail(['test']);
    const panel = await openPollingPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByText('PROD'));
    const pending = deferred<AppDeployedEnvOutputObj[]>();
    mocks.statuses.mockReturnValueOnce(pending.promise).mockRejectedValueOnce(new TypeError('offline'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    expect(await panel.refreshDeployStatuses()).toBe(false);
    pending.resolve([]);
    await flushPromises();
    expect(mocks.update).not.toHaveBeenCalled();
    const row = screen.getByText('PROD').closest('[data-tip]')!;
    expect(within(row as HTMLElement).getByRole('checkbox')).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledExactlyOnceWith({ appID: 'app-a', visibleEnvNames: ['test', 'prod'] }),
    );
  });

  it('设置面板复用父组件状态，进入设置和保存前仍刷新部署状态', async () => {
    useAppDetail().appDetail = detail(['test']);
    await openPanel();
    const requestsBeforeEdit = mocks.statuses.mock.calls.length;
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await screen.findByRole('button', { name: '保存' });
    expect(mocks.statuses).toHaveBeenCalledTimes(requestsBeforeEdit + 1);

    mocks.statuses.mockResolvedValue([{ id: 'prod', deployStatus: 'deployed' }]);
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: ['test', 'prod'] }),
    );
    expect(mocks.statuses).toHaveBeenCalledTimes(requestsBeforeEdit + 2);
  });

  it('进入编辑态前等待部署状态，避免慢请求期间把已部署环境作为可取消项', async () => {
    useAppDetail().appDetail = detail(['test']);
    const pending = deferred<{ deployStatus: string; id: string }[]>();
    mocks.statuses.mockReturnValue(pending.promise);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    expect(screen.queryByRole('button', { name: '保存' })).not.toBeInTheDocument();
    pending.resolve([{ id: 'prod', deployStatus: 'deployed' }]);
    const row = await screen.findByText('PROD');
    expect(within(row.closest('[data-tip]') as HTMLElement).getByRole('checkbox')).toBeDisabled();
  });

  it('部署中、失败、成功及 Helm 进行中的环境必须可见，分组取消只移除可选环境', async () => {
    const names = ['deploying', 'failed', 'deployed', 'helm-pending', 'free'];
    mocks.list.mockResolvedValue(names.map(name => env(name)));
    mocks.statuses.mockResolvedValue([
      { id: 'deploying', deployStatus: 'deploying' },
      { id: 'failed', deployStatus: 'failed' },
      { id: 'deployed', deployStatus: 'deployed' },
      { id: 'deployed', deployStatus: 'uninstalled', trafficLaneName: 'other-lane' },
      { id: 'helm-pending', deployStatus: 'pending-install' },
    ]);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const group = within(screen.getByRole('group', { name: '测试' }));
    for (const name of names.slice(0, -1)) {
      const row = group.getByText(name.toUpperCase()).closest('[data-tip]')!;
      expect(row).toHaveAttribute('data-tip', '该环境已有部署，必须保持可见，不能取消');
      const checkbox = within(row as HTMLElement).getByRole('checkbox');
      expect(checkbox).toBeChecked();
      expect(checkbox).toBeDisabled();
      await userEvent.click(row);
      expect(checkbox).toBeChecked();
    }
    await userEvent.click(group.getAllByRole('checkbox')[0]);
    expect(group.getByText('FREE').closest('[data-tip]')?.querySelector('input[type="checkbox"]')).not.toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: names.slice(0, -1) }),
    );
  });

  it('特性环境的父环境自动选中并禁用，不能通过单项或整组取消', async () => {
    useAppDetail().appDetail = detail(['test']);
    mocks.list.mockResolvedValue([
      env('test'),
      env('prod', { type: 'production' }),
      env('free', { type: 'production' }),
      env('feature-a', { kind: 'feature', ownerAppID: 'app-a', sourceEnvID: 'prod' }),
    ]);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const row = screen.getByText('PROD').closest('[data-tip]')!;
    expect(row).toHaveAttribute('data-tip', '该环境是特性环境的父环境，必须保持可见，不能取消');
    const checkbox = within(row as HTMLElement).getByRole('checkbox');
    expect(checkbox).toBeChecked();
    expect(checkbox).toBeDisabled();
    await userEvent.click(row);
    await userEvent.click(screen.getByText('FREE'));
    await userEvent.click(within(screen.getByRole('group', { name: '生产' })).getAllByRole('checkbox')[0]);
    expect(checkbox).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: ['test', 'prod'] }),
    );
  });

  it('未限制可见环境时，设置面板默认全选；取消部分勾选后保存具体名单', async () => {
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const panel = within(screen.getByTestId('env-panel'));
    const checkboxes = panel.getAllByRole('checkbox').filter(input => !(input as HTMLInputElement).disabled);
    expect(checkboxes.every(input => (input as HTMLInputElement).checked)).toBe(true);
    await userEvent.click(panel.getByText('PROD'));
    await userEvent.click(panel.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: ['test', 'not-ready'] }),
    );
  });

  it('当点击设置或取消切换编辑态时，应保持下拉打开，点击外部时才关闭', async () => {
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await nextTick();
    expect(screen.getByRole('button', { name: '保存' })).toBeVisible();
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    await nextTick();
    expect(screen.getByRole('button', { name: '设置可见环境' })).toBeVisible();
    await userEvent.click(document.body);
    await waitFor(() => expect(screen.queryByTestId('env-panel')).not.toBeInTheDocument());
  });
  it('当用户对部分勾选的分类全选时，应连同未就绪环境保存具体名单', async () => {
    useAppDetail().appDetail = detail(['test']);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const group = within(screen.getByRole('group', { name: '测试' }));
    const checkboxes = group.getAllByRole('checkbox');
    expect(checkboxes[1]).toBeChecked();
    expect(checkboxes[2]).not.toBeChecked();
    await userEvent.click(checkboxes[0]);
    expect(checkboxes.every(input => (input as HTMLInputElement).checked)).toBe(true);
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: ['test', 'not-ready'] }),
    );
  });

  it('当用户搜索或仅显示已部署环境时，可见环境计数应包含特性环境且保持不变', async () => {
    useAppDetail().appDetail = detail(['test', 'test-2', 'staging']);
    mocks.list.mockResolvedValue([
      env('test'),
      env('test-2'),
      env('staging', { type: 'staging' }),
      env('prod', { type: 'production' }),
      env('feature-a', { kind: 'feature', ownerAppID: 'app-a', sourceEnvID: 'test' }),
      env('feature-b', { kind: 'feature', ownerAppID: 'app-a', sourceEnvID: 'test-2' }),
    ]);
    await openPanel();
    expect(screen.getByText('FEATURE-A')).toBeInTheDocument();
    expect(screen.getByText('FEATURE-B')).toBeInTheDocument();
    expect(screen.getByText('已按可见环境筛选，共 5 个')).toBeInTheDocument();
    await userEvent.type(screen.getByPlaceholderText('请输入关键词'), 'no-match');
    await userEvent.click(screen.getByRole('checkbox', { name: '仅显示已部署环境' }));
    expect(screen.getByText('已按可见环境筛选，共 5 个')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    expect(screen.getByText('PROD')).toBeInTheDocument();
  });

  it('当保存已成功但刷新详情失败时，应提示刷新并保留已保存名单', async () => {
    useAppDetail().appDetail = detail(['test']);
    mocks.get.mockRejectedValueOnce(new Error('offline'));
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByText('PROD'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith(
        expect.objectContaining({
          theme: 'warning',
          message: '配置已保存，获取最新配置失败，请刷新页面重试',
        }),
      ),
    );
    expect(useAppDetail().appDetail?.visibleEnvNames).toEqual(['test', 'prod']);
  });

  it('当名单内没有可部署环境时，应清空目标且保留配置入口', async () => {
    useAppDetail().appDetail = detail(['not-ready']);
    mocks.list.mockResolvedValue(environments.filter(item => item.kind !== 'feature'));
    const view = await openPanel({ modelValue: 'prod', initFirstEnvWhenEmpty: true });
    expect(view.emitted()['update:modelValue']).toContainEqual(['']);
    expect(screen.getByRole('button', { name: '设置可见环境' })).toBeEnabled();
  });

  it.each([
    { visible: [], expected: 'prod' },
    { visible: ['test'], expected: 'test' },
  ])('缓存 prod、名单 $visible 时，初始化单选为 $expected', async ({ visible, expected }) => {
    useDeployEnvStore().updateCurrentEnv('prod');
    useAppDetail().appDetail = detail(visible);
    const view = await openPanel({ initFirstEnvWhenEmpty: true });
    expect(view.emitted()['update:modelValue']).toContainEqual([expected]);
  });

  it('总览下钻到隐藏环境时保留查看目标，部署候选仍按名单过滤', async () => {
    useAppDetail().appDetail = detail(['test']);
    const view = await openPanel({ modelValue: 'prod', preserveMissingModelValue: true });
    expect(view.emitted()['update:modelValue']).toContainEqual(['prod']);
    expect(within(screen.getByTestId('env-panel')).queryByText('PROD')).not.toBeInTheDocument();
  });
  it('隐藏标准环境但保留其特性环境；查看历史时保留全部候选', async () => {
    useAppDetail().appDetail = detail(['test']);
    const view = await openPanel();
    const panel = within(screen.getByTestId('env-panel'));
    expect(panel.queryByText('PROD')).not.toBeInTheDocument();
    expect(panel.getByText('FEATURE-A')).toBeInTheDocument();
    expect(panel.getByText('已按可见环境筛选，共 2 个')).toBeInTheDocument();
    await view.rerender({ filterVisibleEnvs: false });
    expect(panel.getByText('PROD')).toBeInTheDocument();
  });

  it('编辑显示全部标准环境，包括未就绪环境，取消或关闭不会保存', async () => {
    useAppDetail().appDetail = detail(['test']);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const panel = within(screen.getByTestId('env-panel'));
    expect(panel.getByText('PROD')).toBeInTheDocument();
    expect(panel.getByText('NOT-READY')).toBeInTheDocument();
    expect(panel.queryByText('FEATURE-A')).not.toBeInTheDocument();
    await userEvent.click(panel.getByText('PROD'));
    await userEvent.click(panel.getByRole('button', { name: '取消' }));
    expect(mocks.update).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByRole('button', { name: '关闭下拉' }));
    await userEvent.click(screen.getByTestId('env-trigger'));
    expect(screen.getByRole('button', { name: '设置可见环境' })).toBeInTheDocument();
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it('恢复全部后提交空数组；保存中阻止重复提交', async () => {
    useAppDetail().appDetail = detail(['test']);
    const pending = deferred<object>();
    mocks.update.mockReturnValueOnce(pending.promise);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByRole('button', { name: '恢复全部' }));
    const checkboxes = within(screen.getByTestId('env-panel'))
      .getAllByRole('checkbox')
      .filter(input => !(input as HTMLInputElement).disabled);
    expect(checkboxes.every(input => (input as HTMLInputElement).checked)).toBe(true);
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: [] });
    pending.resolve({});
    await waitFor(() => expect(screen.queryByTestId('env-panel')).not.toBeInTheDocument());
  });

  it.each(['点击外部', '取消'])('保存前刷新部署状态期间%s，不提交已失效的草稿', async action => {
    useAppDetail().appDetail = detail(['test']);
    mocks.list.mockResolvedValue(environments.filter(item => item.kind !== 'feature'));
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    const pending = deferred<AppDeployedEnvOutputObj[]>();
    mocks.statuses.mockReturnValueOnce(pending.promise);
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    expect(mocks.update).not.toHaveBeenCalled();
    if (action === '点击外部') {
      await userEvent.click(document.body);
      await waitFor(() => expect(screen.queryByTestId('env-panel')).not.toBeInTheDocument());
    } else {
      await userEvent.click(screen.getByRole('button', { name: action }));
    }
    pending.resolve([]);
    await flushPromises();
    expect(mocks.update).not.toHaveBeenCalled();
    expect(useAppDetail().appDetail?.visibleEnvNames).toEqual(['test']);
  });

  it('提交后刷新详情期间关闭再打开下拉，旧保存会话不关闭新面板', async () => {
    useAppDetail().appDetail = detail(['test']);
    const pending = deferred<ReturnType<typeof detail>>();
    mocks.get.mockReturnValueOnce(pending.promise);
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByText('PROD'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByRole('button', { name: '关闭下拉' }));
    await userEvent.click(screen.getByTestId('env-trigger'));
    pending.resolve(detail(['test', 'prod']));
    await flushPromises();
    expect(screen.getByTestId('env-panel')).toBeInTheDocument();
  });

  it('无编辑权限时保留草稿以便重试', async () => {
    useAppDetail().appDetail = detail(['test']);
    mocks.update.mockRejectedValueOnce(new Error('forbidden'));
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByText('PROD'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(useAppDetail().savingVisibleEnvs).toBe(false));
    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.update).toHaveBeenLastCalledWith({ appID: 'app-a', visibleEnvNames: ['test', 'prod'] }),
    );
  });

  it('保存后单选回退可用环境；部署表单清空目标', async () => {
    const view = await openPanel({ modelValue: 'prod', initFirstEnvWhenEmpty: true });
    useAppDetail().appDetail = detail(['test']);
    useAppDetail().visibleEnvsVersion += 1;
    await waitFor(() => expect(view.emitted()['update:modelValue']).toContainEqual(['test']));
    await view.rerender({ modelValue: 'prod', initFirstEnvWhenEmpty: false });
    useAppDetail().visibleEnvsVersion += 1;
    await waitFor(() => expect(view.emitted()['update:modelValue']).toContainEqual(['']));
  });

  it('多环境被全部隐藏后保持空选择', async () => {
    const view = await openPanel({
      mode: 'multi',
      multiSelectable: true,
      modelValues: ['prod'],
      initFirstEnvWhenEmpty: true,
    });
    useAppDetail().appDetail = detail(['test']);
    useAppDetail().visibleEnvsVersion += 1;
    await waitFor(() => expect(view.emitted()['update:modelValues']).toContainEqual([[]]));
  });
});

describe('环境分组选择器部署状态来源', () => {
  function renderSelector(props: { deployStatuses?: AppDeployedEnvOutputObj[]; showDeployIcon?: boolean } = {}) {
    return render(EnvGroupSelector, {
      props: { envList: [env('test')], showDeployIcon: true, ...props },
      global: {
        mocks: { $t: (text: string) => text },
        stubs: {
          ColorIcon: {
            props: ['icon'],
            template: '<span data-testid="deploy-icon" :data-icon="icon" />',
          },
        },
        directives: { 'bk-tooltips': {} },
      },
    });
  }

  it('外部空列表也不自行请求，图标随外部 ID 或名称状态更新', async () => {
    const view = renderSelector({ deployStatuses: [] });
    expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'status-unknown');
    await view.rerender({ deployStatuses: [{ id: 'test', deployStatus: 'deployed' }] });
    expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'normal');
    await view.rerender({ deployStatuses: [{ name: 'test', deployStatus: 'failed' }] });
    expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'abnormal');
    expect(mocks.statuses).not.toHaveBeenCalled();
  });

  it('未传部署状态时保持自行请求的行为', async () => {
    mocks.statuses.mockResolvedValue([{ id: 'test', deployStatus: 'deployed' }]);
    renderSelector();
    await waitFor(() => expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'normal'));
    expect(mocks.statuses).toHaveBeenCalledExactlyOnceWith({ appID: 'app-a' });
  });

  it('切换为外部状态后忽略迟到响应，移除外部状态时重新请求', async () => {
    const pending = deferred<AppDeployedEnvOutputObj[]>();
    const nextPending = deferred<AppDeployedEnvOutputObj[]>();
    mocks.statuses.mockReturnValueOnce(pending.promise).mockReturnValueOnce(nextPending.promise);
    const view = renderSelector();
    await view.rerender({ deployStatuses: [{ id: 'test', deployStatus: 'failed' }] });
    pending.resolve([{ id: 'test', deployStatus: 'deployed' }]);
    await flushPromises();
    expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'abnormal');

    await view.rerender({ deployStatuses: undefined });
    expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'status-unknown');
    expect(mocks.statuses).toHaveBeenCalledTimes(2);
    nextPending.resolve([{ name: 'test', deployStatus: 'deploying' }]);
    await waitFor(() => expect(screen.getByTestId('deploy-icon')).toHaveAttribute('data-icon', 'loading'));
  });
});
