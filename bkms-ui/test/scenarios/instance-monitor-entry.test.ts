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

import { computed, defineComponent, h, nextTick, ref } from 'vue';

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InstanceActionsHost from '~/pages/application/detail/deploy/instance-list/components/instance-actions-host.vue';
import { useInstanceActions } from '~/pages/application/detail/deploy/instance-list/composables/use-instance-actions';

import type { MonitorInstanceGroup } from '~/pages/application/detail/deploy/instance-list/types';

const mocks = vi.hoisted(() => ({ message: vi.fn() }));

vi.mock('bkui-vue', () => ({ InfoBox: vi.fn(), Message: mocks.message }));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1', () => ({ InstanceService: {} }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/composables/use-tab-manager', () => ({
  useTabManager: () => ({ isTabOpen: vi.fn(), openTab: vi.fn() }),
}));
vi.mock('~/pages/application/detail/deploy/instance-list/instance-utils', () => ({
  canAdjustInstanceWeight: () => false,
  renderDeleteInfoBoxContent: vi.fn(),
}));
vi.mock('~/pages/application/detail/deploy/monitor-sideslider.vue', () => ({
  default: {
    name: 'MonitorSidesliderStub',
    props: ['isShow', 'initialSelection', 'envName', 'envNames', 'instanceGroups'],
    render: () => null,
  },
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
vi.mock('~/pages/application/detail/deploy/instance-list/components/actions/management-command-sideslider.vue', () => ({
  default: { render: () => null },
}));

function mountMonitor(count = 0, monitorSelection?: Record<string, string[]>) {
  const groups = ref<MonitorInstanceGroup[]>([{ envName: 'prod', instances: ['pod-a', 'pod-b'] }]);
  const selected = Array.from({ length: count }, (_, index) => ({ id: `pod-${index}` }));
  const harness = defineComponent({
    setup() {
      const host = ref<InstanceType<typeof InstanceActionsHost> | null>(null);
      const actions = useInstanceActions(
        {
          clearSelections: vi.fn(),
          getEnvName: () => 'prod',
          getMonitorSelection: monitorSelection ? () => monitorSelection : undefined,
          getSelectedInstances: () => selected,
          isAllInstancesSelected: computed(() => false),
          refreshData: vi.fn(),
          selectedCount: computed(() => selected.length),
        },
        host,
      );
      return () =>
        h('div', [
          h('button', { onClick: () => actions.openMonitor() }, '顶部监控'),
          h(
            'button',
            {
              onClick: () =>
                actions.handleRowAction({
                  action: 'monitor',
                  envName: 'prod',
                  instance: { id: 'pod-removed' },
                }),
            },
            '行监控',
          ),
          h(InstanceActionsHost, { ref: host, monitorInstanceGroups: groups.value }),
        ]);
    },
  });
  const wrapper = mount(harness);
  return { groups, selected, wrapper, slider: wrapper.getComponent({ name: 'MonitorSidesliderStub' }) };
}

beforeEach(() => vi.clearAllMocks());

describe('实例监控入口', () => {
  it('顶部继承 50 个已选实例，不截断选择', async () => {
    const { selected, slider, wrapper } = mountMonitor(50);
    await wrapper.get('button').trigger('click');
    expect(slider.props('isShow')).toBe(true);
    expect(slider.props('initialSelection')).toEqual({ prod: selected.map(instance => instance.id) });
    expect(mocks.message).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('超过 50 个已选实例时提示并阻止打开，行监控仍聚焦该实例', async () => {
    const { slider, wrapper } = mountMonitor(51);
    await wrapper.get('button').trigger('click');
    expect(slider.props('isShow')).toBe(false);
    expect(mocks.message).toHaveBeenCalledWith({
      theme: 'warning',
      message: '一次最多选择 50 个实例，请减少勾选数量，或清空选择后查看全部实例',
    });
    await wrapper.findAll('button')[1].trigger('click');
    expect(slider.props('isShow')).toBe(true);
    expect(slider.props('initialSelection')).toEqual({ prod: ['pod-removed'] });
    wrapper.unmount();
  });

  it('无勾选时传全部模式，并响应式传递同数量替换后的实例快照', async () => {
    const { groups, slider, wrapper } = mountMonitor();
    await wrapper.get('button').trigger('click');
    expect(slider.props('initialSelection')).toBeUndefined();
    expect(slider.props('instanceGroups')).toEqual(groups.value);
    groups.value = [{ envName: 'prod', instances: ['pod-c', 'pod-d'] }];
    await nextTick();
    expect(slider.props('instanceGroups')).toEqual([{ envName: 'prod', instances: ['pod-c', 'pod-d'] }]);
    wrapper.unmount();
  });

  it('多环境顶部继承各环境的同名实例，不合并环境归属', async () => {
    const selection = { prod: ['shared-pod'], test: ['shared-pod'] };
    const { slider, wrapper } = mountMonitor(0, selection);
    await wrapper.get('button').trigger('click');
    expect(slider.props('isShow')).toBe(true);
    expect(slider.props('initialSelection')).toEqual(selection);
    expect(mocks.message).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('多环境合计超过 50 个时阻止打开，行入口仍仅选择所在环境实例', async () => {
    const selection = {
      prod: Array.from({ length: 25 }, (_, index) => `pod-${index}`),
      test: Array.from({ length: 26 }, (_, index) => `pod-${index}`),
    };
    const { slider, wrapper } = mountMonitor(0, selection);
    await wrapper.get('button').trigger('click');
    expect(slider.props('isShow')).toBe(false);
    expect(mocks.message).toHaveBeenCalledWith({
      theme: 'warning',
      message: '一次最多选择 50 个实例，请减少勾选数量，或清空选择后查看全部实例',
    });
    await wrapper.findAll('button')[1].trigger('click');
    expect(slider.props('isShow')).toBe(true);
    expect(slider.props('initialSelection')).toEqual({ prod: ['pod-removed'] });
    wrapper.unmount();
  });
});
