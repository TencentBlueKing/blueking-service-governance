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
 * 场景级测试：创建特性环境的命名空间选择（本头即新增交互的路径清单）。
 * 覆盖：默认共享 / 独立创建及折叠保留 / 联邦来源切换 / 重开重置与离开确认 /
 * 创建失败重试 / 请求期间防重 / 联邦默认来源 / 来源选择器与侧栏数据不一致 /
 * 组件风险确认、配置跳转及刷新 / 折叠后清单变化重新展开 / 组件信息失败时独立创建仍可提交 / 过期响应 / 成功态提交快照 /
 * 创建中关闭及重开 / 来源列表迟到保留填写 / 局部刷新不改全局详情。
 * stub/mock：API、应用上下文；来源选择器模拟列表加载与选择事件，侧栏仅省略过渡动画，
 * tooltip 指令以 title 暴露契约。Form、Input、Radio 和未保存修改检测使用真实实现。
 */
import { defineComponent, h, ref } from 'vue';
import type { SetupContext } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CreateFeatureEnv from '~/pages/application/detail/deploy/create-feature-env.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { MockedApiError } from '../helpers/mocked-api-error';

import type { AppDetailOutputObj } from '~/@types/v1/app';
import type { EnvOutput } from '~/@types/v1/env';

const mocks = vi.hoisted(() => ({
  listAppEnvs: vi.fn(),
  createFeatureEnv: vi.fn(),
  getApp: vi.fn(),
  fetchAppDetail: vi.fn(),
  infoBox: vi.fn(),
  created: vi.fn(),
  deploy: vi.fn(),
  resolve: vi.fn(),
  appContext: { appID: 'app-id-1', app: 'app-a', appDetail: null as AppDetailOutputObj | null },
  currentEnv: { name: 'prod' },
  sourceEnvs: [] as EnvOutput[],
  sourceListWait: undefined as Promise<void> | undefined,
}));

vi.mock('vue-i18n', async () => {
  const { ref } = await import('vue');
  const { i18nMockFactory, i18nGlobalMocks } = await import('../helpers/mock-i18n');
  return { ...(await i18nMockFactory()), useI18n: () => ({ locale: ref('zh-CN'), t: i18nGlobalMocks.$t }) };
});
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  const routerMock = await createRouterMock({
    route: {
      params: { space: 'ws-1', name: 'app-a', type: 'trpc' },
      query: { activeTab: 'overview', view: 'feature-envs', envVars: 'old-env', envName: 'prod' },
    },
  });
  return {
    ...routerMock,
    useRouter: () => ({ ...routerMock.useRouter(), resolve: mocks.resolve }),
  };
});
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  InfoBox: mocks.infoBox,
}));
vi.mock('~/api/modules/v1', () => ({
  AppService: { getApp: mocks.getApp },
  EnvService: { listAppEnvs: mocks.listAppEnvs, createFeatureEnv: mocks.createFeatureEnv },
}));
vi.mock('~/stores/app-detail', async () => {
  const { reactive } = await import('vue');
  mocks.appContext = reactive(mocks.appContext);
  return { useAppDetail: () => Object.assign(mocks.appContext, { fetchAppDetail: mocks.fetchAppDetail }) };
});
vi.mock('~/stores/trpc-deploy', () => ({ useTrpcDeployStore: () => ({ curEnvItem: mocks.currentEnv }) }));
vi.mock('~/components/env-select-panel/env-select-panel.vue', async () => {
  const { defineComponent, onMounted } = await import('vue');
  return {
    default: defineComponent({
      props: { modelValue: { type: String, default: '' } },
      emits: ['update:modelValue', 'update:item', 'update:envList'],
      setup(_, { emit }) {
        onMounted(async () => {
          await mocks.sourceListWait;
          emit('update:envList', mocks.sourceEnvs);
        });
        return {
          envs: mocks.sourceEnvs,
          selectEnv(name: string) {
            emit('update:modelValue', name);
            emit(
              'update:item',
              mocks.sourceEnvs.find(env => env.name === name),
            );
          },
        };
      },
      template: `<select aria-label="来源环境" :value="modelValue" @change="selectEnv($event.target.value)">
        <option v-for="env in envs" :key="env.id" :value="env.name">{{ env.displayName }}</option>
      </select>`,
    }),
  };
});

const standardEnv: EnvOutput = {
  id: 'prod-id',
  name: 'prod',
  displayName: '生产环境',
  kind: 'standard',
  status: 'Ready',
  cluster: { clusterID: 'cluster-1', namespace: 'source-ns', isFederation: false },
};
const federationEnv: EnvOutput = {
  ...standardEnv,
  id: 'federation-id',
  name: 'prod-gz',
  displayName: '生产环境 - 广州',
  cluster: { clusterID: 'federation-cluster', namespace: 'federation-ns', isFederation: true },
};
const createdEnv: EnvOutput = {
  id: 'feature-id',
  name: 'feat-app-a-1',
  displayName: '订单联调',
  cluster: { namespace: 'source-ns' },
};

const appWithComponents: AppDetailOutputObj = {
  id: 'app-id-1',
  appModelSpec: {
    components: [
      { name: 'redis', type: 'redis' },
      { refWorkspaceCompName: 'mysql', scopeType: 'selected', scopeEnvNames: ['other-env'] },
    ],
  },
};

// 保留富文本中的真实交互节点，便于验证组件配置和独立创建链接。
const I18nTWithSlots = {
  props: {
    keypath: { type: String, default: '' },
    tag: { type: String, default: 'span' },
  },
  setup(props: { keypath: string; tag: string }, { slots }: SetupContext) {
    return () => {
      const children = slots.default?.() ?? [];
      return h(
        props.tag,
        props.keypath.split(/(\{\d+\})/).map(part => {
          const match = part.match(/^\{(\d+)\}$/);
          return match ? children[Number(match[1])] : part;
        }),
      );
    };
  },
};

const SidesliderStub = defineComponent({
  props: { isShow: Boolean, beforeClose: { type: Function, required: true } },
  emits: ['closed', 'update:isShow'],
  watch: {
    isShow(value: boolean) {
      if (!value) this.$emit('closed');
    },
  },
  methods: {
    async closeFromHeader() {
      if (await this.beforeClose()) this.$emit('update:isShow', false);
    },
  },
  template: `<div v-if="isShow"><button @click="closeFromHeader">头部关闭</button>
    <slot name="header" /><slot /></div>`,
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.appContext.appID = 'app-id-1';
  mocks.appContext.app = 'app-a';
  mocks.appContext.appDetail = { id: 'app-id-1', visibleEnvNames: ['prod'] };
  mocks.currentEnv.name = 'prod';
  mocks.sourceEnvs = [standardEnv, federationEnv];
  mocks.sourceListWait = undefined;
  mocks.listAppEnvs.mockResolvedValue(mocks.sourceEnvs);
  mocks.createFeatureEnv.mockResolvedValue(createdEnv);
  mocks.getApp.mockReset();
  mocks.getApp.mockResolvedValue({ id: 'app-id-1', appModelSpec: { components: [] } });
  mocks.fetchAppDetail.mockReset();
  // 真实 store 刷新失败会清空全局详情；局部组件查询不得调用此方法。
  mocks.fetchAppDetail.mockImplementation(async () => {
    mocks.appContext.appDetail = null;
    return null;
  });
  mocks.resolve.mockReturnValue({ href: '/component-config' });
  mocks.infoBox.mockImplementation((options: { onConfirm: () => void }) => options.onConfirm());
  vi.spyOn(window, 'open').mockReturnValue(null);
});

afterEach(() => {
  vi.restoreAllMocks();
});

async function fillDisplayName() {
  await userEvent.type(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名'), '订单联调');
}

async function openCreateForm({ waitForSource = true } = {}) {
  const harness = defineComponent({
    components: { CreateFeatureEnv },
    setup() {
      return { isShow: ref(false), created: mocks.created, deploy: mocks.deploy };
    },
    template: `<button @click="isShow = true">打开创建侧栏</button>
      <CreateFeatureEnv v-model:is-show="isShow" @created="created" @deploy="deploy" />`,
  });
  const view = render(harness, {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTWithSlots },
      stubs: { Sideslider: SidesliderStub, Popover: { template: '<div><slot /></div>' } },
      directives: {
        'bk-tooltips': {
          mounted(el: HTMLElement, { value }: { value: { content: string; disabled: boolean } }) {
            el.title = value.disabled ? '' : value.content;
          },
          updated(el: HTMLElement, { value }: { value: { content: string; disabled: boolean } }) {
            el.title = value.disabled ? '' : value.content;
          },
        },
      },
    },
  });
  await userEvent.click(screen.getByRole('button', { name: '打开创建侧栏' }));
  if (waitForSource) {
    await waitFor(() => expect(screen.getByRole('combobox', { name: '来源环境' })).toHaveValue(mocks.currentEnv.name));
  }
  await flushPromises();
  return view;
}

describe('特性环境：创建时选择命名空间', () => {
  it('当用户使用默认设置创建时，应显式提交共享命名空间并展示实际命名空间', async () => {
    await openCreateForm();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
    expect(screen.getByText('命名空间：沿用来源环境（默认）')).toBeInTheDocument();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith({
        appID: 'app-id-1',
        displayName: '订单联调',
        sourceEnvID: 'prod-id',
        nsType: 'shared',
      }),
    );
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(screen.getByText('source-ns')).toBeInTheDocument();
    expect(mocks.created).toHaveBeenCalledWith(createdEnv);
    expect(mocks.deploy).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '立即部署' }));
    expect(mocks.deploy).toHaveBeenCalledWith(createdEnv);
  });

  it('当用户选择独立创建再折叠高级设置时，应保留选择并提交独立命名空间', async () => {
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
    expect(screen.getByTitle('共享命名空间仅支持 AppModel 应用，使用组件的应用无法部署。')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByText('命名空间：独立创建')).toBeInTheDocument();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith(expect.objectContaining({ nsType: 'independent' })),
    );
  });

  it('当用户从独立创建切换到联邦来源时，应改为共享并禁用独立创建且切回后保持共享', async () => {
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod-gz');
    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeDisabled();
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
    expect(
      screen.getByTitle('来源环境（生产环境 - 广州）绑定了联邦集群，暂不支持创建独立的特性环境。'),
    ).toBeInTheDocument();
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod');
    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeEnabled();
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
  });

  it('当选择器返回联邦来源而侧栏仍缓存普通集群时，应按实际选择禁用独立创建并提交共享', async () => {
    mocks.listAppEnvs.mockResolvedValue([
      standardEnv,
      {
        ...federationEnv,
        displayName: '旧生产环境 - 广州',
        cluster: { ...federationEnv.cluster, isFederation: false },
      },
    ]);
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod-gz');

    const independentRadio = screen.getByRole('radio', { name: /独立创建/ });
    expect(independentRadio).toBeDisabled();
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
    expect(
      screen.getByTitle('来源环境（生产环境 - 广州）绑定了联邦集群，暂不支持创建独立的特性环境。'),
    ).toBeInTheDocument();
    await userEvent.click(independentRadio);
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith(
        expect.objectContaining({ sourceEnvID: 'federation-id', nsType: 'shared' }),
      ),
    );
  });

  it('当选择器返回普通来源而侧栏仍缓存联邦集群时，应允许按实际选择独立创建', async () => {
    mocks.sourceEnvs = [standardEnv, { ...federationEnv, cluster: { ...federationEnv.cluster, isFederation: false } }];
    mocks.listAppEnvs.mockResolvedValue([standardEnv, federationEnv]);
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod-gz');

    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeEnabled();
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith(
        expect.objectContaining({ sourceEnvID: 'federation-id', nsType: 'independent' }),
      ),
    );
  });

  it('当默认来源为联邦环境且选择器仅同步列表时，应禁用独立创建并使用当前展示名提示', async () => {
    mocks.currentEnv.name = 'prod-gz';
    mocks.listAppEnvs.mockResolvedValue([
      standardEnv,
      {
        ...federationEnv,
        displayName: '旧生产环境 - 广州',
        cluster: { ...federationEnv.cluster, isFederation: false },
      },
    ]);
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));

    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeDisabled();
    expect(
      screen.getByTitle('来源环境（生产环境 - 广州）绑定了联邦集群，暂不支持创建独立的特性环境。'),
    ).toBeInTheDocument();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith(
        expect.objectContaining({ sourceEnvID: 'federation-id', nsType: 'shared' }),
      ),
    );
  });

  it('当用户只修改命名空间后取消并重新打开时，应提示未保存修改且恢复共享和折叠状态', async () => {
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.queryByRole('button', { name: '创建' })).not.toBeInTheDocument());
    await userEvent.click(screen.getByRole('button', { name: '打开创建侧栏' }));
    expect(await screen.findByText('命名空间：沿用来源环境（默认）')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    expect(mocks.infoBox).toHaveBeenCalledTimes(1);
  });

  it('当用户创建失败后重试时，应保留表单与命名空间选择并且只在成功后通知刷新', async () => {
    mocks.createFeatureEnv.mockRejectedValueOnce(new MockedApiError());
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeChecked();
    expect(mocks.created).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenNthCalledWith(2, expect.objectContaining({ nsType: 'independent' }));
    expect(mocks.created).toHaveBeenCalledTimes(1);
  });

  it('当用户在创建请求尚未完成时重复点击创建，应只发起一次请求', async () => {
    let resolveCreate!: (value: EnvOutput) => void;
    mocks.createFeatureEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    const createButton = screen.getByRole('button', { name: '创建' });
    await userEvent.click(createButton);
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    await userEvent.click(createButton);
    expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1);
    resolveCreate(createdEnv);
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
  });

  it('当创建期间经头部关闭侧栏后后台成功时，应通知刷新且不重新展示成功页', async () => {
    let resolveCreate!: (value: EnvOutput) => void;
    mocks.createFeatureEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByRole('button', { name: '头部关闭' }));
    await waitFor(() => expect(screen.queryByRole('button', { name: '创建' })).not.toBeInTheDocument());

    resolveCreate(createdEnv);
    await flushPromises();
    expect(mocks.created).toHaveBeenCalledExactlyOnceWith(createdEnv);
    expect(screen.queryByText('环境创建成功')).not.toBeInTheDocument();
    expect(mocks.deploy).not.toHaveBeenCalled();
  });

  it('当关闭重开后上次创建成功时，应刷新列表但保留新表单内容和未保存标记', async () => {
    let resolveCreate!: (value: EnvOutput) => void;
    mocks.createFeatureEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByRole('button', { name: '头部关闭' }));
    await userEvent.click(screen.getByRole('button', { name: '打开创建侧栏' }));
    await userEvent.type(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名'), '新一轮联调');

    resolveCreate(createdEnv);
    await flushPromises();
    expect(mocks.created).toHaveBeenCalledExactlyOnceWith(createdEnv);
    expect(screen.queryByText('环境创建成功')).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('新一轮联调');
    await userEvent.click(screen.getByRole('button', { name: '头部关闭' }));
    expect(mocks.infoBox).toHaveBeenCalledTimes(2);
  });

  it('当切换应用后旧应用的创建请求成功时，不应通知当前应用刷新或展示旧成功页', async () => {
    let resolveCreate!: (value: EnvOutput) => void;
    mocks.createFeatureEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    mocks.getApp.mockResolvedValue({ id: 'app-id-2', appModelSpec: { components: [] } });
    mocks.appContext.appID = 'app-id-2';
    await waitFor(() => expect(mocks.getApp).toHaveBeenLastCalledWith({ appID: 'app-id-2' }));

    resolveCreate(createdEnv);
    await flushPromises();
    expect(mocks.created).not.toHaveBeenCalled();
    expect(screen.queryByText('环境创建成功')).not.toBeInTheDocument();
  });

  it('当用户当前环境绑定联邦集群时，应保留该来源并支持默认共享创建', async () => {
    mocks.currentEnv.name = 'prod-gz';
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() =>
      expect(mocks.createFeatureEnv).toHaveBeenCalledWith(
        expect.objectContaining({
          sourceEnvID: 'federation-id',
          nsType: 'shared',
        }),
      ),
    );
  });

  it('当应用配置了任意组件时，应展开风险提示并在确认前阻止共享创建，失败后可保留确认重试', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    mocks.createFeatureEnv.mockRejectedValueOnce(new MockedApiError());
    await openCreateForm();

    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'true');
    expect(
      screen.getByText(/应用使用了组件（.*redis.*mysql.*），沿用来源环境创建的特性环境将无法部署。/),
    ).toBeInTheDocument();
    const acknowledgement = screen.getByRole('checkbox', { name: '我已知晓风险，仍要沿用来源环境创建' });
    const createButton = screen.getByRole('button', { name: '创建' });
    expect(acknowledgement).not.toBeChecked();
    expect(createButton).toBeDisabled();
    await fillDisplayName();
    await userEvent.click(createButton);
    expect(mocks.createFeatureEnv).not.toHaveBeenCalled();

    await userEvent.click(acknowledgement);
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
    expect(createButton).toBeEnabled();
    await userEvent.click(createButton);
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    expect(acknowledgement).toBeChecked();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(mocks.created).not.toHaveBeenCalled();

    await userEvent.click(createButton);
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenNthCalledWith(2, {
      appID: 'app-id-1',
      displayName: '订单联调',
      sourceEnvID: 'prod-id',
      nsType: 'shared',
    });
    expect(screen.queryByRole('button', { name: '立即部署' })).not.toBeInTheDocument();
    expect(
      screen.queryByText('当前使用应用默认配置，未配置环境级变量。可直接部署，也可先调整配置。'),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/应用使用了组件，沿用来源环境的特性环境将无法部署。/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '调整配置' })).toBeInTheDocument();
    expect(mocks.created).toHaveBeenCalledTimes(1);
    expect(mocks.deploy).not.toHaveBeenCalled();
  });

  it('当应用使用组件但选择独立创建时，应隐藏风险确认并保留立即部署', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建' })).toBeEnabled();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenCalledWith(expect.objectContaining({ nsType: 'independent' }));
    await userEvent.click(screen.getByRole('button', { name: '立即部署' }));
    expect(mocks.deploy).toHaveBeenCalledWith(createdEnv);
  });

  it('当使用组件的应用选择联邦来源时，应禁止独立创建并要求确认，保留来源切换和折叠前的填写内容', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('checkbox', { name: '我已知晓风险，仍要沿用来源环境创建' }));
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod-gz');
    expect(screen.getByRole('radio', { name: /独立创建/ })).toBeDisabled();
    expect(screen.getByText(/来源环境是联邦集群，暂不能「独立创建」。/)).toBeInTheDocument();
    expect(screen.getByRole('checkbox')).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByText('命名空间：沿用来源环境（默认）')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByRole('checkbox')).toBeChecked();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    await userEvent.click(screen.getByRole('checkbox'));
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenCalledWith(
      expect.objectContaining({ nsType: 'shared', sourceEnvID: 'federation-id' }),
    );
    expect(screen.queryByRole('button', { name: '立即部署' })).not.toBeInTheDocument();
    expect(screen.getByText(/来源环境是联邦集群，暂不能「独立创建」。/)).toBeInTheDocument();
  });

  it('当只勾选风险确认后关闭并重开时，应提示未保存修改并清空上次确认', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByRole('button', { name: '打开创建侧栏' }));
    expect(await screen.findByRole('checkbox')).not.toBeChecked();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    expect(mocks.infoBox).toHaveBeenCalledTimes(1);
  });

  it('当组件信息仍在加载时，应阻止创建并在加载后显示风险，且不覆盖用户输入', async () => {
    let resolveDetail!: (value: AppDetailOutputObj) => void;
    mocks.getApp.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveDetail = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(mocks.createFeatureEnv).not.toHaveBeenCalled();
    resolveDetail(appWithComponents);
    expect(await screen.findByRole('checkbox')).not.toBeChecked();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
  });

  it('当来源列表迟到时，应保留已输入名称、风险确认、折叠状态及未保存标记', async () => {
    let resolveList!: () => void;
    mocks.sourceListWait = new Promise(resolve => {
      resolveList = resolve;
    });
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm({ waitForSource: false });
    await fillDisplayName();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));

    resolveList();
    await flushPromises();
    expect(screen.getByRole('combobox', { name: '来源环境' })).toHaveValue('prod');
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByRole('checkbox')).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    expect(mocks.infoBox).toHaveBeenCalledTimes(1);
  });

  it.each([
    { source: 'prod', selectedLabel: /独立创建/, isFederation: false },
    { source: 'prod-gz', selectedLabel: /沿用来源环境/, isFederation: true },
  ])(
    '当来源列表迟到且默认来源为$source时，应保留填写并按来源约束处理独立选择',
    async ({ source, selectedLabel, isFederation }) => {
      let resolveList!: () => void;
      mocks.sourceListWait = new Promise(resolve => {
        resolveList = resolve;
      });
      mocks.currentEnv.name = source;
      mocks.getApp.mockResolvedValue(appWithComponents);
      await openCreateForm({ waitForSource: false });
      await fillDisplayName();
      await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));

      resolveList();
      await flushPromises();
      expect(screen.getByRole('combobox', { name: '来源环境' })).toHaveValue(source);
      expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
      expect(screen.getByRole('radio', { name: selectedLabel })).toBeChecked();
      expect(screen.getByRole('radio', { name: /独立创建/ })).toHaveProperty('disabled', isFederation);
    },
  );

  it('当局部组件查询失败时，应保留全局应用详情及环境名单并禁止创建', async () => {
    const existingDetail = mocks.appContext.appDetail;
    mocks.getApp.mockRejectedValueOnce(new MockedApiError());
    await openCreateForm();

    expect(await screen.findByText('获取应用组件信息失败，请重试。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    expect(mocks.fetchAppDetail).not.toHaveBeenCalled();
    expect(mocks.appContext.appDetail).toBe(existingDetail);
    expect(mocks.appContext.appDetail?.visibleEnvNames).toEqual(['prod']);
  });

  it.each(['请求失败', '空响应', '错误应用'])('当组件信息%s时，应阻止创建并支持保留输入重试', async failure => {
    if (failure === '请求失败') mocks.getApp.mockRejectedValueOnce(new MockedApiError());
    if (failure === '空响应') mocks.getApp.mockResolvedValueOnce(null);
    if (failure === '错误应用') mocks.getApp.mockResolvedValueOnce({ id: 'another-app' });
    await openCreateForm();
    expect(await screen.findByText('获取应用组件信息失败，请重试。')).toBeInTheDocument();
    await fillDisplayName();
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(mocks.createFeatureEnv).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    await waitFor(() => expect(screen.getByRole('button', { name: '创建' })).toBeEnabled());
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(mocks.getApp).toHaveBeenCalledTimes(2);
  });

  it('当组件信息查询失败但选择独立创建时，应允许创建', async () => {
    mocks.getApp.mockRejectedValueOnce(new MockedApiError());
    await openCreateForm();
    expect(await screen.findByText('获取应用组件信息失败，请重试。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    expect(screen.getByRole('button', { name: '创建' })).toBeEnabled();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenCalledWith(expect.objectContaining({ nsType: 'independent' }));
  });

  it('当打开组件配置并返回后组件清单变化时，应刷新清单和确认状态，保留表单及当前应用路由', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await fillDisplayName();
    await userEvent.selectOptions(screen.getByRole('combobox', { name: '来源环境' }), 'prod-gz');
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '组件配置' }));
    expect(mocks.resolve).toHaveBeenCalledWith({
      name: 'detail',
      params: { space: 'ws-1', name: 'app-a', type: 'trpc', menuName: 'module' },
    });
    expect(window.open).toHaveBeenCalledWith('/component-config', '_blank');
    expect(screen.getByRole('radio', { name: /沿用来源环境/ })).toBeChecked();
    mocks.getApp.mockResolvedValue({
      id: 'app-id-1',
      appModelSpec: { components: [{ name: 'kafka', type: 'kafka' }] },
    });
    window.dispatchEvent(new Event('focus'));
    await waitFor(() => expect(mocks.getApp).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.getByRole('checkbox')).not.toBeChecked());
    expect(screen.getByText(/应用使用了组件（kafka）/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(screen.getByRole('combobox', { name: '来源环境' })).toHaveValue('prod-gz');
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
  });

  it('当沿用来源环境且高级设置已折叠时，组件清单变化应重新展开并要求重新确认', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '组件配置' }));
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');

    mocks.getApp.mockResolvedValue({
      id: 'app-id-1',
      appModelSpec: { components: [{ name: 'kafka', type: 'kafka' }] },
    });
    window.dispatchEvent(new Event('focus'));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'true'),
    );
    expect(screen.getByRole('checkbox')).not.toBeChecked();
    expect(screen.getByText(/应用使用了组件（kafka）/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled();
  });

  it('当已选择独立创建并折叠后组件清单变化时，应保持折叠并允许创建', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('button', { name: '组件配置' }));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    await userEvent.click(screen.getByRole('button', { name: '高级设置' }));
    mocks.getApp.mockResolvedValue({
      id: 'app-id-1',
      appModelSpec: { components: [{ name: 'kafka', type: 'kafka' }] },
    });
    window.dispatchEvent(new Event('focus'));
    await waitFor(() => expect(mocks.getApp).toHaveBeenCalledTimes(2));
    await flushPromises();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByText('命名空间：独立创建')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建' })).toBeEnabled();
  });

  it('当多次从同一组件配置标签页返回时，应持续刷新组件并在全部移除后恢复普通创建', async () => {
    mocks.getApp.mockResolvedValue(appWithComponents);
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '组件配置' }));
    window.dispatchEvent(new Event('focus'));
    await waitFor(() => expect(mocks.getApp).toHaveBeenCalledTimes(2));
    await flushPromises();
    expect(screen.getByRole('checkbox')).toBeChecked();

    // 用户再次切回同一个配置标签页编辑，不会重新点击创建侧栏里的配置链接。
    mocks.getApp.mockResolvedValue({ id: 'app-id-1', appModelSpec: { components: [] } });
    window.dispatchEvent(new Event('focus'));
    await waitFor(() => expect(mocks.getApp).toHaveBeenCalledTimes(3));
    await waitFor(() => expect(screen.queryByRole('checkbox')).not.toBeInTheDocument());
    expect(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名')).toHaveValue('订单联调');
    expect(screen.getByRole('button', { name: '创建' })).toBeEnabled();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '立即部署' })).toBeInTheDocument();
  });

  it('当关闭重开后上次组件请求才返回时，不应覆盖本次组件结果', async () => {
    let resolveOldDetail!: (value: AppDetailOutputObj) => void;
    mocks.getApp.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveOldDetail = resolve;
        }),
    );
    await openCreateForm();
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    await userEvent.click(screen.getByRole('button', { name: '打开创建侧栏' }));
    await waitFor(() => expect(mocks.getApp).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.getByRole('button', { name: '创建' })).toBeEnabled());
    resolveOldDetail(appWithComponents);
    await flushPromises();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByRole('button', { name: '创建' })).toBeEnabled();
  });

  it('当切换应用后旧应用的组件请求才返回时，不应把旧组件风险带到当前应用', async () => {
    let resolveOldDetail!: (value: AppDetailOutputObj) => void;
    mocks.getApp.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveOldDetail = resolve;
        }),
    );
    await openCreateForm();
    mocks.getApp.mockResolvedValue({ id: 'app-id-2', appModelSpec: { components: [] } });
    mocks.appContext.appID = 'app-id-2';
    await waitFor(() => expect(mocks.getApp).toHaveBeenLastCalledWith({ appID: 'app-id-2' }));
    await flushPromises();
    resolveOldDetail(appWithComponents);
    await flushPromises();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '高级设置' })).toHaveAttribute('aria-expanded', 'false');
    expect(mocks.createFeatureEnv).not.toHaveBeenCalled();
  });

  it('当创建请求期间用户切换模式时，成功页应依据实际提交的共享风险隐藏立即部署', async () => {
    let resolveCreate!: (value: EnvOutput) => void;
    mocks.getApp.mockResolvedValue(appWithComponents);
    mocks.createFeatureEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        }),
    );
    await openCreateForm();
    await fillDisplayName();
    await userEvent.click(screen.getByRole('checkbox'));
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.createFeatureEnv).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByRole('radio', { name: /独立创建/ }));
    resolveCreate(createdEnv);
    expect(await screen.findByText('环境创建成功')).toBeInTheDocument();
    expect(mocks.createFeatureEnv).toHaveBeenCalledWith(expect.objectContaining({ nsType: 'shared' }));
    expect(screen.queryByRole('button', { name: '立即部署' })).not.toBeInTheDocument();
    expect(mocks.deploy).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '组件配置' }));
    expect(window.open).toHaveBeenCalledWith('/component-config', '_blank');
  });
});
