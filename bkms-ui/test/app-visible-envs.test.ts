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

import { defineComponent, effectScope, nextTick } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EnvSelectPanel from '~/components/env-select-panel.vue';
import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
import { useDeployPrecheck } from '~/pages/application/detail/deploy/use-deploy-precheck';
import { useAppDetail } from '~/stores/app-detail';
import { useDeployEnvStore } from '~/stores/deploy-env';

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
  env('feature-a', { kind: 'feature', ownerAppID: 'app-a', sourceEnvID: 'prod' }),
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

async function openPanel(props: Record<string, unknown> = {}) {
  const result = render(EnvSelectPanel, {
    props: { enableVisibleEnvSettings: true, filterVisibleEnvs: true, ...props },
    global: {
      mocks: {
        $t: (text: string, params: unknown[] = []) => text.replace(/\{(\d+)\}/g, (_, i) => String(params[Number(i)])),
      },
      stubs: { Popover: PopoverStub, Select: true, StatusDotIcon: true, ColorIcon: true, OverflowTags: true },
      directives: { 'bk-tooltips': {} },
    },
  });
  await userEvent.click(screen.getByTestId('env-trigger'));
  await waitFor(() => expect(screen.getByRole('button', { name: '设置可见环境' })).toBeEnabled());
  return result;
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

  it('当用户搜索或仅显示已部署环境时，可见标准环境计数应保持不变', async () => {
    useAppDetail().appDetail = detail(['test', 'not-ready']);
    await openPanel();
    await userEvent.type(screen.getByPlaceholderText('请输入关键词'), 'no-match');
    await userEvent.click(screen.getByRole('checkbox', { name: '仅显示已部署环境' }));
    expect(screen.getByText('已按可见环境筛选，共 2 个')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    expect(screen.getByText('PROD')).toBeInTheDocument();
  });

  it('当保存已成功但刷新详情失败时，应提示刷新并保留已保存名单', async () => {
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
    expect(useAppDetail().appDetail?.visibleEnvNames).toEqual(['prod']);
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
    expect(panel.getByText('已按可见环境筛选，共 1 个')).toBeInTheDocument();
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
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.update).toHaveBeenCalledWith({ appID: 'app-a', visibleEnvNames: [] });
    pending.resolve({});
    await waitFor(() => expect(screen.queryByTestId('env-panel')).not.toBeInTheDocument());
  });

  it('无编辑权限时保留草稿以便重试', async () => {
    mocks.update.mockRejectedValueOnce(new Error('forbidden'));
    await openPanel();
    await userEvent.click(screen.getByRole('button', { name: '设置可见环境' }));
    await userEvent.click(screen.getByText('PROD'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(useAppDetail().savingVisibleEnvs).toBe(false));
    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(mocks.update).toHaveBeenLastCalledWith({ appID: 'app-a', visibleEnvNames: ['prod'] }));
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
