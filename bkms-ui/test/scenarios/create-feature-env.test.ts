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
 * 创建失败重试 / 请求期间防重 / 联邦默认来源 / 来源选择器与侧栏数据不一致 → V = 10。
 * stub/mock：API、应用上下文；来源选择器模拟列表加载与选择事件，侧栏仅省略过渡动画，
 * tooltip 指令以 title 暴露契约。Form、Input、Radio 和未保存修改检测使用真实实现。
 */
import { defineComponent, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CreateFeatureEnv from '~/pages/application/detail/deploy/create-feature-env.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { MockedApiError } from '../helpers/mocked-api-error';

import type { EnvOutput } from '~/@types/v1/env';

const mocks = vi.hoisted(() => ({
  listAppEnvs: vi.fn(),
  createFeatureEnv: vi.fn(),
  infoBox: vi.fn(),
  created: vi.fn(),
  deploy: vi.fn(),
  currentEnv: { name: 'prod' },
  sourceEnvs: [] as EnvOutput[],
}));

vi.mock('vue-i18n', async () => {
  const { i18nMockFactory, i18nGlobalMocks } = await import('../helpers/mock-i18n');
  return { ...(await i18nMockFactory()), useI18n: () => ({ t: i18nGlobalMocks.$t }) };
});
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({ route: { params: { space: 'ws-1', name: 'app-a', type: 'trpc' } } });
});
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  InfoBox: mocks.infoBox,
}));
vi.mock('~/api/modules/v1', () => ({
  EnvService: { listAppEnvs: mocks.listAppEnvs, createFeatureEnv: mocks.createFeatureEnv },
}));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'app-id-1', app: 'app-a' }) }));
vi.mock('~/stores/trpc-deploy', () => ({ useTrpcDeployStore: () => ({ curEnvItem: mocks.currentEnv }) }));
vi.mock('~/components/env-select-panel/env-select-panel.vue', async () => {
  const { defineComponent, onMounted } = await import('vue');
  return {
    default: defineComponent({
      props: { modelValue: { type: String, default: '' } },
      emits: ['update:modelValue', 'update:item', 'update:envList'],
      setup(_, { emit }) {
        onMounted(() => emit('update:envList', mocks.sourceEnvs));
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

const SidesliderStub = defineComponent({
  props: { isShow: Boolean },
  emits: ['closed'],
  watch: {
    isShow(value: boolean) {
      if (!value) this.$emit('closed');
    },
  },
  template: '<div v-if="isShow"><slot name="header" /><slot /></div>',
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.currentEnv.name = 'prod';
  mocks.sourceEnvs = [standardEnv, federationEnv];
  mocks.listAppEnvs.mockResolvedValue(mocks.sourceEnvs);
  mocks.createFeatureEnv.mockResolvedValue(createdEnv);
  mocks.infoBox.mockImplementation((options: { onConfirm: () => void }) => options.onConfirm());
});

async function fillDisplayName() {
  await userEvent.type(screen.getByPlaceholderText('请输入 1-32 字符的环境展示名'), '订单联调');
}

async function openCreateForm() {
  const harness = defineComponent({
    components: { CreateFeatureEnv },
    setup() {
      return { isShow: ref(false), created: mocks.created, deploy: mocks.deploy };
    },
    template: `<button @click="isShow = true">打开创建侧栏</button>
      <CreateFeatureEnv v-model:is-show="isShow" @created="created" @deploy="deploy" />`,
  });
  render(harness, {
    global: {
      mocks: i18nGlobalMocks,
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
  await waitFor(() => expect(screen.getByRole('combobox', { name: '来源环境' })).toHaveValue(mocks.currentEnv.name));
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
});
