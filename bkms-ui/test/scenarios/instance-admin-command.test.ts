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

import { computed, defineComponent, h, reactive, ref } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import InstanceActionsHost from '~/pages/application/detail/deploy/instance-list/components/instance-actions-host.vue';
import ManagementCommand from '~/pages/application/detail/deploy/instance-list/components/management-command.vue';
import { useInstanceActions } from '~/pages/application/detail/deploy/instance-list/composables/use-instance-actions';

import type { PublicView } from '~/@types/v1/app';
import type { AppInstanceOutputObj } from '~/@types/v1/instance';

/* UI 组件使用轻量桩，保留操作控制器、侧栏和管理命令的真实传参链路。 */
/* eslint-disable vue/one-component-per-file */

const originalScrollIntoView = Element.prototype.scrollIntoView;

const mocks = vi.hoisted(() => ({
  app: { appID: 'demo-app', runtime: null as null | PublicView },
  executeTafAdminCmd: vi.fn(),
  executeTrpcAdminCmd: vi.fn(),
  legacyEnv: { curEnvItem: undefined as undefined | { name: string } },
  listTrpcAdminCmds: vi.fn(),
}));

vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => mocks.app }));
vi.mock('~/stores/trpc-deploy', () => ({ useTrpcDeployStore: () => mocks.legacyEnv }));
vi.mock('~/api/modules/v1', () => ({
  InstanceService: {
    executeTafAdminCmd: mocks.executeTafAdminCmd,
    executeTrpcAdminCmd: mocks.executeTrpcAdminCmd,
    listTrpcAdminCmds: mocks.listTrpcAdminCmds,
  },
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('monaco-editor', () => ({}));
vi.mock('~/composables/use-leave-confirm', () => ({ default: () => ({ confirmBox: vi.fn() }) }));
vi.mock('~/composables/use-tab-manager', () => ({
  useTabManager: () => ({ isTabOpen: vi.fn(), openTab: vi.fn() }),
}));
vi.mock('~/components/toggle-card.vue', () => ({
  default: { template: '<div><slot /></div>' },
}));
vi.mock('~/pages/application/detail/deploy/monitor-sideslider.vue', () => ({
  default: { render: () => null },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/actions/weight-dialog.vue', () => ({
  default: { render: () => null },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/actions/gray-dialog.vue', () => ({
  default: { render: () => null },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/actions/instance-log-sideslider.vue', () => ({
  default: { render: () => null },
}));
vi.mock('@blueking/table', () => ({
  Table: { render: () => null },
  TableColumn: { render: () => null },
}));
vi.mock('bkui-vue', async () => {
  const { defineComponent, h } = await import('vue');
  const slot = defineComponent({
    setup(_, { slots }) {
      return () => h('div', [slots.title?.(), slots.default?.()]);
    },
  });
  const button = defineComponent({
    props: { disabled: Boolean },
    setup(props, { slots }) {
      return () => h('button', { disabled: props.disabled }, slots.default?.());
    },
  });
  const input = defineComponent({
    props: { modelValue: { type: String, default: '' } },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('input', {
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        });
    },
  });
  const select = defineComponent({
    props: { modelValue: { type: String, default: '' } },
    emits: ['update:modelValue'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'select',
          {
            value: props.modelValue,
            onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
          },
          slots.default?.(),
        );
    },
  });
  const option = defineComponent({
    props: { id: { type: String, default: '' }, name: { type: String, default: '' } },
    setup(props) {
      return () => h('option', { value: props.id }, props.name);
    },
  });
  const sideslider = defineComponent({
    props: { isShow: Boolean },
    setup(props, { slots }) {
      return () => (props.isShow ? h('section', { role: 'dialog' }, [slots.header?.(), slots.default?.()]) : null);
    },
  });
  return {
    Alert: slot,
    Button: button,
    Divider: slot,
    Exception: slot,
    InfoBox: vi.fn(),
    Input: input,
    Message: vi.fn(),
    OverflowTitle: slot,
    Radio: Object.assign({}, { Button: slot, Group: slot }),
    ResizeLayout: slot,
    Select: Object.assign(select, { Option: option }),
    Sideslider: sideslider,
    Tab: Object.assign(slot, { TabPanel: slot }),
    Tag: slot,
  };
});

function mountActions(envName = 'env-b', instanceIds = ['pod-b']) {
  const selectedEnv = ref(envName);
  const selected = ref<AppInstanceOutputObj[]>(instanceIds.map(id => ({ id })));
  const harness = defineComponent({
    setup() {
      const host = ref<InstanceType<typeof InstanceActionsHost> | null>(null);
      const actions = useInstanceActions(
        {
          clearSelections: vi.fn(),
          commandEnvDisplayName: () => '环境 B',
          getEnvName: () => selectedEnv.value,
          getSelectedInstances: () => selected.value,
          isAllInstancesSelected: computed(() => false),
          refreshData: vi.fn(),
          selectedCount: computed(() => selected.value.length),
        },
        host,
      );
      return () =>
        h('div', [
          h('button', { onClick: actions.openAdminCommand }, '打开管理命令'),
          h(InstanceActionsHost, { ref: host }),
        ]);
    },
  });
  const wrapper = mount(harness, {
    global: {
      directives: { 'bk-tooltips': {} },
      mocks: { $t: (key: string) => key },
      stubs: { IconButton: true, MsEditor: true },
    },
  });
  return { selectedEnv, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.app = reactive({
    appID: 'demo-app',
    runtime: { framework: 'trpc', mods: { adminCommand: { supported: true } } },
  });
  mocks.legacyEnv.curEnvItem = { name: 'env-a' };
  mocks.listTrpcAdminCmds.mockResolvedValue({ results: ['/health'] });
  const response = { status: 200, json: async () => ({ data: { results: [] } }) };
  mocks.executeTrpcAdminCmd.mockResolvedValue(response);
  mocks.executeTafAdminCmd.mockResolvedValue(response);
  Element.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
  Element.prototype.scrollIntoView = originalScrollIntoView;
});

describe('管理命令的实例环境', () => {
  it.each([
    { envName: 'env-b', legacyEnv: { name: 'env-a' } },
    { envName: 'env-single', legacyEnv: undefined },
  ])('tRPC 查询和执行使用打开侧栏时的 $envName', async ({ envName, legacyEnv }) => {
    mocks.legacyEnv.curEnvItem = legacyEnv;
    const { selectedEnv, wrapper } = mountActions(envName);
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(mocks.listTrpcAdminCmds).toHaveBeenCalledWith(
      {
        appID: 'demo-app',
        envName,
        instanceIDs: ['pod-b'],
      },
      { interceptorErr: false },
    );
    expect(wrapper.get('[role="dialog"]').text()).toContain('环境 B');

    // 侧栏打开后，列表选择发生变化也不能改变这批实例的目标环境。
    selectedEnv.value = 'env-c';
    mocks.legacyEnv.curEnvItem = { name: 'env-c' };
    await wrapper.findAll('select')[1].setValue('/health');
    const execute = wrapper.findAll('button').find(button => button.text() === '执行');
    expect(execute).toBeDefined();
    await execute!.trigger('click');
    await flushPromises();
    expect(mocks.executeTrpcAdminCmd).toHaveBeenCalledWith(
      {
        appID: 'demo-app',
        envName,
        instanceIDs: ['pod-b'],
        url: '/health',
        method: 'GET',
        params: {},
      },
      { originalResponse: true, needStatus: true, interceptorErr: false },
    );
    expect(mocks.executeTafAdminCmd).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('直接进入多环境时，TAF 执行使用所选实例环境，不依赖旧单环境状态', async () => {
    mocks.app.runtime = { framework: 'taf', mods: { adminCommand: { supported: true } } };
    mocks.legacyEnv.curEnvItem = undefined;
    const { wrapper } = mountActions();
    await wrapper.get('button').trigger('click');
    await wrapper.get('input').setValue('loadconfig');
    await wrapper
      .findAll('button')
      .find(button => button.text() === '执行')!
      .trigger('click');
    await flushPromises();
    expect(mocks.executeTafAdminCmd).toHaveBeenCalledWith(
      {
        appID: 'demo-app',
        envName: 'env-b',
        instanceIDs: ['pod-b'],
        command: 'loadconfig',
      },
      { originalResponse: true, needStatus: true, interceptorErr: false },
    );
    expect(mocks.listTrpcAdminCmds).not.toHaveBeenCalled();
    expect(mocks.executeTrpcAdminCmd).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it.each([
    null,
    { framework: 'trpc', mods: { adminCommand: { supported: false } } },
    { framework: 'taf', mods: { adminCommand: { supported: false } } },
  ])('能力未加载或不支持管理命令时不打开侧栏', async runtime => {
    mocks.app.runtime = runtime;
    const { wrapper } = mountActions();
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false);
    expect(mocks.listTrpcAdminCmds).not.toHaveBeenCalled();
    expect(mocks.executeTrpcAdminCmd).not.toHaveBeenCalled();
    expect(mocks.executeTafAdminCmd).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('直接挂载不支持管理命令的 tRPC 表单时不查询命令', async () => {
    mocks.app.runtime = { framework: 'trpc', mods: { adminCommand: { supported: false } } };
    const wrapper = mount(ManagementCommand, {
      props: { data: ['pod-b'], envName: 'env-b' },
      global: {
        directives: { 'bk-tooltips': {} },
        mocks: { $t: (key: string) => key },
        stubs: { IconButton: true, MsEditor: true },
      },
    });
    await flushPromises();
    expect(mocks.listTrpcAdminCmds).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it.each(['trpc', 'taf'])('侧栏打开后 %s 能力失效时不再执行命令', async framework => {
    mocks.app.runtime = { framework, mods: { adminCommand: { supported: true } } };
    const { wrapper } = mountActions();
    await wrapper.get('button').trigger('click');
    await flushPromises();
    if (framework === 'trpc') {
      await wrapper.findAll('select')[1].setValue('/health');
    } else {
      await wrapper.get('input').setValue('loadconfig');
    }
    mocks.app.runtime = { framework, mods: { adminCommand: { supported: false } } };
    await wrapper
      .findAll('button')
      .find(button => button.text() === '执行')!
      .trigger('click');
    await flushPromises();
    expect(mocks.executeTrpcAdminCmd).not.toHaveBeenCalled();
    expect(mocks.executeTafAdminCmd).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it.each([
    { envName: '', instanceIds: ['pod-b'] },
    { envName: 'env-b', instanceIds: [] },
  ])('没有环境或实例时不打开侧栏', async ({ envName, instanceIds }) => {
    const { wrapper } = mountActions(envName, instanceIds);
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false);
    expect(mocks.listTrpcAdminCmds).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});
