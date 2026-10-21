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

import { computed, defineComponent, h, ref } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type AppInstanceOutputObj } from '~/@types/v1/instance';

/* 保留真实表单校验，其他 bkui-vue 组件使用轻量桩。 */
/* eslint-disable vue/one-component-per-file */

const mocks = vi.hoisted(() => ({
  openWeight: vi.fn(),
  runActionSuccess: vi.fn(),
  updateWeight: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/composables/use-tab-manager', () => ({
  useTabManager: () => ({ isTabOpen: vi.fn(), openTab: vi.fn() }),
}));
vi.mock('~/api/modules/v1', () => ({ InstanceService: { updateAppInstancePolaris: mocks.updateWeight } }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/pages/application/detail/deploy/instance-list/composables/use-instance-action-context', () => ({
  provideInstanceActionContext: vi.fn(),
  runActionSuccess: mocks.runActionSuccess,
  useInstanceActionContext: () => ({}),
}));
vi.mock('~/components/divider-header.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: { title: { type: String, default: '' } },
      setup(props, { slots }) {
        return () => h('div', [props.title, slots.default?.()]);
      },
    }),
  };
});
vi.mock('bkui-vue', async () => {
  const { defineComponent, h } = await import('vue');
  const { default: bkForm } = await import('bkui-vue/lib/form');
  const stubSlot = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  });
  const stubButton = defineComponent({
    props: { disabled: Boolean },
    setup(props, { attrs, slots }) {
      return () => h('button', { ...attrs, disabled: props.disabled }, slots.default?.());
    },
  });
  const stubDialog = defineComponent({
    props: { isShow: Boolean },
    setup(props, { slots }) {
      return () =>
        props.isShow ? h('div', { role: 'dialog' }, [slots.header?.(), slots.default?.(), slots.footer?.()]) : null;
    },
  });
  const stubInput = defineComponent({
    props: { modelValue: { type: [String, Number], default: '' } },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('input', {
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        });
    },
  });
  const stubDropdown = defineComponent({
    setup(_, { slots }) {
      return () => h('div', [slots.default?.(), slots.content?.()]);
    },
  });
  const stubDropdownItem = defineComponent({
    setup(_, { attrs, slots }) {
      return () => h('div', { ...attrs, role: 'menuitem' }, slots.default?.());
    },
  });
  return {
    Alert: stubSlot,
    Button: stubButton,
    Dialog: stubDialog,
    Dropdown: Object.assign(stubDropdown, { DropdownItem: stubDropdownItem, DropdownMenu: stubSlot }),
    Form: bkForm,
    Input: stubInput,
    Popover: stubSlot,
  };
});

import WeightDialog from '~/pages/application/detail/deploy/instance-list/components/actions/weight-dialog.vue';
import InstanceBatchToolbar from '~/pages/application/detail/deploy/instance-list/components/instance-batch-toolbar.vue';
import { useInstanceActions } from '~/pages/application/detail/deploy/instance-list/composables/use-instance-actions';

const translate = (key: string, values: Array<number | string> = []) =>
  key.replace(/\{(\d+)\}/g, (_, index: string) => String(values[Number(index)]));
const global = { directives: { 'bk-tooltips': {} }, mocks: { $t: translate } };

function mountWeightDialog(options: {
  clearSelectionOnSuccess?: boolean;
  initialWeight?: null | number;
  instanceIds: string[];
}) {
  return mount(WeightDialog, {
    props: {
      envName: 'prod',
      instanceName: options.instanceIds[0] ?? '',
      visible: false,
      ...options,
    },
    global,
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.runActionSuccess.mockResolvedValue(undefined);
  mocks.updateWeight.mockResolvedValue(undefined);
});

describe('实例批量调整权重', () => {
  it('所选实例必须全部具有北极星信息，跨页结果传入全部实例', async () => {
    const selected = ref<AppInstanceOutputObj[]>([]);
    const harness = defineComponent({
      setup() {
        const actions = useInstanceActions(
          {
            clearSelections: vi.fn(),
            getEnvName: () => 'prod',
            getSelectedInstances: () => selected.value,
            isAllInstancesSelected: computed(() => false),
            refreshData: vi.fn(),
            selectedCount: computed(() => selected.value.length),
          },
          ref({ openWeight: mocks.openWeight }) as never,
        );
        return () => h('button', { onClick: actions.openSelectedWeight }, '调整所选');
      },
    });
    const wrapper = mount(harness);
    await wrapper.find('button').trigger('click');
    expect(mocks.openWeight).not.toHaveBeenCalled();

    selected.value = [
      { id: 'pod-1', polarisInfos: [{}] },
      { id: 'pod-2', polarisInfos: [] },
    ] as AppInstanceOutputObj[];
    await wrapper.find('button').trigger('click');
    expect(mocks.openWeight).not.toHaveBeenCalled();

    selected.value = Array.from({ length: 12 }, (_, index) => ({
      id: `pod-${index + 1}`,
      polarisInfos: [{}],
    })) as AppInstanceOutputObj[];
    await wrapper.find('button').trigger('click');
    expect(mocks.openWeight).toHaveBeenCalledWith({
      clearSelectionOnSuccess: true,
      envName: 'prod',
      instances: selected.value,
    });
  });

  it('无选中或含不可调整实例时禁用入口，灰度位于更多菜单', async () => {
    const wrapper = mount(InstanceBatchToolbar, {
      props: {
        canAdjustWeight: false,
        canGrayDeploy: false,
        isAllInstancesSelected: false,
        selectedCount: 0,
      },
      global,
    });
    const weightButton = wrapper.findAll('button').find(button => button.text() === '调整权重');
    const grayItem = wrapper.find('[role="menuitem"]');
    expect(weightButton?.attributes('disabled')).toBeDefined();
    expect(grayItem.attributes('aria-disabled')).toBe('true');

    await wrapper.setProps({ selectedCount: 2 });
    expect(weightButton?.attributes('disabled')).toBeDefined();
    expect(wrapper.findAll('button').some(button => button.text() === '灰度')).toBe(false);
    expect(grayItem.text()).toBe('灰度');
    expect(grayItem.attributes('aria-disabled')).toBe('true');
    await grayItem.trigger('click');
    expect(wrapper.emitted('gray')).toBeUndefined();

    await wrapper.setProps({ canGrayDeploy: true });
    expect(grayItem.attributes('aria-disabled')).toBe('false');
    await grayItem.trigger('click');
    expect(wrapper.emitted('gray')).toHaveLength(1);

    await wrapper.setProps({ canAdjustWeight: true });
    expect(weightButton?.attributes('disabled')).toBeUndefined();
    await weightButton?.trigger('click');
    expect(wrapper.emitted('weight')).toHaveLength(1);
  });

  it('单个实例显示名称与当前权重，并提交该实例 ID', async () => {
    const wrapper = mountWeightDialog({ initialWeight: 42, instanceIds: ['pod-1'] });
    await wrapper.setProps({ visible: true });
    expect(wrapper.text()).toContain('pod-1');
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('42');

    await wrapper.get('[role="dialog"] button').trigger('click');
    await flushPromises();
    expect(mocks.updateWeight).toHaveBeenCalledWith({
      appID: 'demo-app',
      envName: 'prod',
      trafficLaneName: '',
      instanceIDs: ['pod-1'],
      weight: 42,
    });
    expect(mocks.runActionSuccess).toHaveBeenCalledWith({}, { clearSelection: false });
  });

  it('多个实例显示数量，权重初始为空，输入后一次提交全部 ID', async () => {
    const wrapper = mountWeightDialog({
      clearSelectionOnSuccess: true,
      initialWeight: null,
      instanceIds: ['pod-1', 'pod-2', 'pod-3'],
    });
    await wrapper.setProps({ visible: true });
    expect(wrapper.text()).toContain('已选 3 个实例');
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('');

    await wrapper.get('[role="dialog"] button').trigger('click');
    await flushPromises();
    expect(mocks.updateWeight).not.toHaveBeenCalled();

    await wrapper.find('input').setValue('10');
    await wrapper.get('[role="dialog"] button').trigger('click');
    await flushPromises();
    expect(mocks.updateWeight).toHaveBeenCalledWith(
      expect.objectContaining({ instanceIDs: ['pod-1', 'pod-2', 'pod-3'], weight: 10 }),
    );
    expect(mocks.runActionSuccess).toHaveBeenCalledWith({}, { clearSelection: true });
  });

  it.each(['', '-1', '10001', '1.5'])('非法权重 "%s" 不提交', async weight => {
    const wrapper = mountWeightDialog({ initialWeight: 10, instanceIds: ['pod-1', 'pod-2'] });
    await wrapper.setProps({ visible: true });
    await wrapper.find('input').setValue(weight);
    await wrapper.get('[role="dialog"] button').trigger('click');
    await flushPromises();
    expect(mocks.updateWeight).not.toHaveBeenCalled();
    expect(wrapper.emitted('update:visible')).toBeUndefined();
    expect(mocks.runActionSuccess).not.toHaveBeenCalled();
  });

  it('校验未完成前连点两次只提交一次', async () => {
    const wrapper = mountWeightDialog({ initialWeight: 10, instanceIds: ['pod-1', 'pod-2'] });
    await wrapper.setProps({ visible: true });
    const button = wrapper.get('[role="dialog"] button');

    // 不等待第一次点击的校验结果，模拟校验期间再次点击
    void button.trigger('click');
    void button.trigger('click');
    await flushPromises();
    await flushPromises();

    expect(mocks.updateWeight).toHaveBeenCalledOnce();
    expect(mocks.runActionSuccess).toHaveBeenCalledOnce();
    expect(wrapper.emitted('update:visible')).toHaveLength(1);
  });

  it('接口失败时保留弹窗和输入值', async () => {
    const wrapper = mountWeightDialog({ initialWeight: 10, instanceIds: ['pod-1', 'pod-2'] });
    await wrapper.setProps({ visible: true });
    mocks.updateWeight.mockRejectedValueOnce(new Error('request failed'));
    await wrapper.find('input').setValue('25');
    await wrapper.get('[role="dialog"] button').trigger('click');
    await flushPromises();
    expect(mocks.updateWeight).toHaveBeenCalledOnce();
    expect(mocks.updateWeight).toHaveBeenCalledWith(
      expect.objectContaining({ instanceIDs: ['pod-1', 'pod-2'], weight: 25 }),
    );
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true);
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('25');
    expect(wrapper.emitted('update:visible')).toBeUndefined();
    expect(mocks.runActionSuccess).not.toHaveBeenCalled();
  });
});
