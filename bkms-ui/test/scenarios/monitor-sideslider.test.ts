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

import { nextTick } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import dayjs from 'dayjs';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchMonitorMetric } from '~/pages/application/detail/deploy/monitor-chart-bridge';
import MonitorSideslider from '~/pages/application/detail/deploy/monitor-sideslider.vue';

import type * as BkDatePicker from '@blueking/date-picker';
import type { MonitorInstanceGroup } from '~/pages/application/detail/deploy/instance-list/types';
import type { MonitorInstance, MonitorMetricTimeSeries } from '~/pages/application/detail/deploy/monitor-chart-bridge';

// UI 桩组件共同保留侧栏的查询与选择事件链路。
/* eslint-disable vue/one-component-per-file */

const mocks = vi.hoisted(() => ({
  chartQuery: vi.fn(),
  getInstanceTimeSeries: vi.fn(),
  listAppInstances: vi.fn(),
  rankInstances: vi.fn(),
  message: vi.fn(),
  registerPoll: vi.fn<(callback: () => Promise<void>) => void>(),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values: unknown[] = []) => key.replace(/\{(\d+)\}/g, (_, index) => String(values[index])),
  }),
}));
vi.mock('~/api/modules/v1', () => ({
  InstanceService: { listAppInstances: mocks.listAppInstances },
  BkintegrationsBkmonitorService: {
    getInstanceTimeSeries: mocks.getInstanceTimeSeries,
    rankInstances: mocks.rankInstances,
  },
}));
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/stores/deploy-env', () => ({ useDeployEnvStore: () => ({ currentEnv: 'prod' }) }));
vi.mock('~/composables/use-interval', () => ({
  default: (callback: () => Promise<void>) => {
    mocks.registerPoll(callback);
    return { start: vi.fn(), stop: vi.fn() };
  },
}));
vi.mock('@blueking/date-picker', async importOriginal => ({
  ...(await importOriginal<typeof BkDatePicker>()),
  default: {
    name: 'DatePickerStub',
    emits: ['update:model-value'],
    render: () => null,
  },
}));
vi.mock('~/components/interval-selector.vue', () => ({
  default: {
    name: 'IntervalSelectorStub',
    emits: ['refresh'],
    template: '<button @click="$emit(\'refresh\')">刷新</button>',
  },
}));
vi.mock('~/components/toggle-card.vue', () => ({ default: { template: '<div><slot /></div>' } }));
vi.mock('~/pages/application/detail/deploy/chart-card-item.vue', async () => {
  const { defineComponent, watch } = await import('vue');
  return {
    __esModule: true,
    default: defineComponent({
      name: 'ChartCardStub',
      props: {
        instances: { type: Array as () => MonitorInstance[], default: () => [] },
        method: { type: String as () => 'avg' | 'count' | 'max' | 'min' | 'sum', default: undefined },
        interval: { type: Number, default: undefined },
        metricKey: { type: String, required: true },
        queryId: { type: Number, required: true },
        queryEnabled: Boolean,
        envNames: { type: Array, default: undefined },
        compareOffsets: { type: Array, default: () => [] },
        startTime: { type: Number, required: true },
        endTime: { type: Number, required: true },
      },
      emits: ['metric-data', 'plot-width-change'],
      setup(props, { emit }) {
        watch(
          [() => props.queryId, () => props.queryEnabled],
          async ([queryId, queryEnabled]) => {
            if (!queryEnabled || !props.instances.length) return;
            const query = {
              appID: 'demo-app',
              instances: props.instances,
              interval: props.interval,
              method: props.method,
              metricKey: props.metricKey,
              startTime: props.startTime,
              endTime: props.endTime,
            };
            mocks.chartQuery({ ...query, queryId });
            const data = await fetchMonitorMetric(query);
            emit('metric-data', { data, metricKey: props.metricKey, queryId });
          },
          { immediate: true },
        );
        return () => null;
      },
    }),
  };
});
vi.mock('@blueking/table', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Table: defineComponent({
      name: 'SummaryTableStub',
      props: { data: { type: Array, default: () => [] } },
      setup(props) {
        return () =>
          h(
            'table',
            { 'data-testid': 'summary' },
            props.data.map(row => h('tr', JSON.stringify(row))),
          );
      },
    }),
    TableColumn: { render: () => null },
  };
});
vi.mock('bkui-vue', async () => {
  const { defineComponent, h, ref } = await import('vue');
  const slot = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.header?.(), slots.default?.()]),
  });
  const select = defineComponent({
    name: 'SelectStub',
    props: {
      modelValue: { type: [String, Number, Array, Boolean], default: undefined },
      customContent: Boolean,
      multiple: Boolean,
      placeholder: { type: String, default: '' },
      disabled: Boolean,
    },
    emits: ['update:modelValue', 'clear', 'tag-remove', 'toggle'],
    setup(props, { emit, expose, slots }) {
      expose({ setSelected: () => undefined });
      return () =>
        props.customContent
          ? h('div', { 'data-testid': 'instance-selector', 'data-placeholder': props.placeholder }, slots.default?.())
          : h(
              'select',
              {
                disabled: props.disabled,
                value: props.modelValue,
                onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
              },
              slots.default?.(),
            );
    },
  });
  const option = defineComponent({
    props: { id: { type: [String, Number], default: '' }, name: { type: String, default: '' } },
    setup: props => () => h('option', { value: props.id }, props.name),
  });
  const input = defineComponent({
    name: 'InputStub',
    props: {
      modelValue: { type: [String, Number], default: '' },
      placeholder: { type: String, default: '' },
      type: { type: String, default: 'text' },
      disabled: Boolean,
    },
    emits: ['update:modelValue', 'blur', 'change', 'enter'],
    setup:
      (props, { emit }) =>
      () =>
        h('input', {
          value: props.modelValue,
          placeholder: props.placeholder,
          type: props.type || 'text',
          disabled: props.disabled,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
          onBlur: () => emit('blur'),
          onChange: (event: Event) => {
            const { value } = event.target as HTMLInputElement;
            emit('update:modelValue', value);
            emit('change', value);
          },
          onKeydown: (event: KeyboardEvent) => {
            if (event.key === 'Enter') emit('enter', props.modelValue);
          },
        }),
  });
  interface TreeNode {
    children?: TreeNode[];
    id: string;
    name: string;
  }
  const tree = defineComponent({
    name: 'TreeStub',
    inheritAttrs: false,
    props: { data: { type: Array, default: () => [] } },
    emits: ['node-checked'],
    setup(props, { emit, expose }) {
      const checked = ref(new Set<string>());
      const leaves = () => (props.data as TreeNode[]).flatMap(node => node.children || [node]);
      expose({
        setChecked(ids: string[], selected: boolean) {
          for (const id of ids) selected ? checked.value.add(id) : checked.value.delete(id);
        },
      });
      return () =>
        h(
          'div',
          leaves().map(node =>
            h('input', {
              type: 'checkbox',
              'data-testid': 'instance-candidate',
              value: node.id,
              checked: checked.value.has(node.id),
              onChange: (event: Event) => {
                if ((event.target as HTMLInputElement).checked) checked.value.add(node.id);
                else checked.value.delete(node.id);
                emit(
                  'node-checked',
                  leaves().filter(leaf => checked.value.has(leaf.id)),
                );
              },
            }),
          ),
        );
    },
  });
  return {
    Divider: slot,
    Input: input,
    Message: mocks.message,
    Progress: slot,
    Radio: { Group: slot, Button: slot },
    Select: Object.assign(select, { Option: option }),
    Sideslider: slot,
    Switcher: { name: 'SwitcherStub', emits: ['update:modelValue'], render: () => null },
    Tag: slot,
    Tree: tree,
  };
});

function metric(names: (MonitorInstance | string)[], unit = 'cores', value = 1): MonitorMetricTimeSeries {
  return {
    unit,
    series: names.map(instance => ({
      instance: typeof instance === 'string' ? instance : instance.instance,
      envName: typeof instance === 'string' ? 'prod' : instance.envName,
      stat: { last: [0, value] },
      dataPoints: [[value, 1000]],
    })),
  };
}

const instanceKey = (instance: string, envName = 'prod') => JSON.stringify([envName, instance]);
const instances = (names: string[], envName = 'prod'): MonitorInstance[] =>
  names.map(instance => ({ envName, instance }));

async function mountSlider(
  initialSelection?: Record<string, string[]>,
  instanceGroups?: MonitorInstanceGroup[],
  isShow = true,
) {
  const wrapper = mount(MonitorSideslider, {
    props: {
      isShow,
      envName: 'prod',
      envNames: instanceGroups?.map(group => group.envName),
      instanceGroups: instanceGroups ?? [
        { envName: 'prod', instances: Array.from({ length: 2000 }, (_, index) => `pod-${index}`) },
      ],
      initialSelection,
    },
    global: {
      components: { ToggleCard: { template: '<div><slot /></div>' } },
      directives: { 'bk-tooltips': {}, bkloading: {} },
      mocks: {
        $t: (key: string, values: unknown[] = []) => key.replace(/\{(\d+)\}/g, (_, index) => String(values[index])),
      },
    },
  });
  await flushPromises();
  const selects = () => wrapper.findAllComponents({ name: 'SelectStub' });
  const charts = () => wrapper.findAllComponents({ name: 'ChartCardStub' });
  const chart = (metricKey = 'cpu_limit_usage') => charts().find(item => item.props('metricKey') === metricKey)!;
  if (isShow) {
    charts()[0]?.vm.$emit('plot-width-change', 450);
    await flushPromises();
  }
  return {
    wrapper,
    selects,
    chart,
    rows: () => wrapper.getComponent({ name: 'SummaryTableStub' }).props('data'),
    async emitMetric(metricKey: string, data: MonitorMetricTimeSeries, queryId = chart(metricKey).props('queryId')) {
      chart(metricKey).vm.$emit('metric-data', { metricKey, data, queryId });
      await flushPromises();
    },
  };
}

const rank = (names: string[]) => names.map((instance, index) => ({ instance, value: names.length - index }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.setSystemTime(new Date(2026, 10, 5, 12));
  mocks.rankInstances.mockResolvedValue(rank(['pod-1', 'pod-2']));
  mocks.getInstanceTimeSeries.mockImplementation((query: { instances: string[]; metricKey: string }) =>
    Promise.resolve({ [query.metricKey]: metric(query.instances) }),
  );
});

afterEach(() => vi.useRealTimers());

describe('实例监控侧栏', () => {
  it('宽度就绪后先排名一次，名单返回后仅查询一轮八图', async () => {
    const { promise, resolve } = deferred<ReturnType<typeof rank>>();
    mocks.rankInstances.mockReturnValueOnce(promise);
    const { wrapper, chart } = await mountSlider(undefined, undefined, false);
    try {
      await wrapper.setProps({ isShow: true });
      await flushPromises();
      expect(mocks.rankInstances).not.toHaveBeenCalled();
      expect(chart().props('queryEnabled')).toBe(false);
      chart().vm.$emit('plot-width-change', 450);
      await flushPromises();
      expect(mocks.rankInstances).toHaveBeenCalledExactlyOnceWith({
        appID: 'demo-app',
        envName: 'prod',
        startTime: dayjs().startOf('day').unix(),
        endTime: dayjs().unix(),
        interval: 96,
        limit: 10,
        rankType: 'top',
        method: 'sum',
      });
      expect(mocks.getInstanceTimeSeries).not.toHaveBeenCalled();
      await mocks.registerPoll.mock.lastCall?.[0]();
      expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
      resolve(rank(['pod-7', 'pod-1']));
      for (const card of wrapper.findAllComponents({ name: 'ChartCardStub' })) {
        card.vm.$emit('plot-width-change', 450);
      }
      chart().vm.$emit('plot-width-change', 0);
      await flushPromises();
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
      expect(new Set(mocks.chartQuery.mock.calls.map(([query]) => query.queryId)).size).toBe(1);
      for (const [query] of mocks.getInstanceTimeSeries.mock.calls) {
        expect(query).toEqual({
          appID: 'demo-app',
          envName: 'prod',
          instances: ['pod-7', 'pod-1'],
          metricKey: expect.any(String),
          startTime: dayjs().startOf('day').unix(),
          endTime: dayjs().unix(),
          interval: 96,
          method: 'sum',
        });
      }
    } finally {
      wrapper.unmount();
    }
  });

  it('手动固定汇聚周期后图表宽度变化不额外请求', async () => {
    const { wrapper, selects, chart } = await mountSlider();
    selects()[2].vm.$emit('update:modelValue', 60);
    await flushPromises();
    chart().vm.$emit('plot-width-change', 500);
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    expect(chart().props('interval')).toBe(60);
    wrapper.unmount();
  });

  it('Limit 键入期间不查询，回车和步进各提交一轮，重开后恢复默认值', async () => {
    const { wrapper, chart } = await mountSlider();
    const limitInput = wrapper.get<HTMLInputElement>('input[type="number"]');

    for (const value of ['2', '25']) {
      limitInput.element.value = value;
      await limitInput.trigger('input');
      await flushPromises();
      expect(chart().props('queryEnabled')).toBe(true);
      expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
    }

    await limitInput.trigger('keydown', { key: 'Enter' });
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 25 }));
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);

    await limitInput.trigger('change');
    await limitInput.trigger('blur');
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);

    const limitComponent = wrapper
      .findAllComponents({ name: 'InputStub' })
      .find(input => input.props('type') === 'number')!;
    limitComponent.vm.$emit('update:modelValue', 26);
    limitComponent.vm.$emit('change', 26);
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(3);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 26 }));
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(24);

    await wrapper.setProps({ isShow: false });
    await wrapper.setProps({ isShow: true });
    chart().vm.$emit('plot-width-change', 450);
    await flushPromises();
    expect(wrapper.get<HTMLInputElement>('input[type="number"]').element.value).toBe('10');
    expect(mocks.rankInstances).toHaveBeenCalledTimes(4);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 10 }));
    wrapper.unmount();
  });

  it('auto 尺寸变化仅在实际步长改变时重拉一轮', async () => {
    const { wrapper, chart } = await mountSlider();
    chart().vm.$emit('plot-width-change', 451);
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
    chart().vm.$emit('plot-width-change', 500);
    await flushPromises();
    expect(chart().props('interval')).toBe(87);
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    wrapper.unmount();
  });

  it('当跨天重新打开侧栏时，应等宽度就绪后查询新一天的数据', async () => {
    const { wrapper, chart } = await mountSlider();
    await wrapper.setProps({ isShow: false });
    vi.setSystemTime(new Date(2026, 10, 6, 1));
    await wrapper.setProps({ isShow: true });
    await flushPromises();
    expect(chart().props('queryEnabled')).toBe(false);
    expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
    chart().vm.$emit('plot-width-change', 450);
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(
      expect.objectContaining({ startTime: dayjs().startOf('day').unix(), endTime: dayjs().unix() }),
    );
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    wrapper.unmount();
  });

  it('当用户手动刷新时，应更新相对时间和步长并只请求一轮', async () => {
    vi.useRealTimers();
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout'] });
    vi.setSystemTime(new Date(2026, 10, 5, 12));
    const { wrapper, chart, selects } = await mountSlider();
    const limitInput = wrapper.get<HTMLInputElement>('input[type="number"]');
    limitInput.element.value = '20';
    await limitInput.trigger('input');
    selects()[1].vm.$emit('update:modelValue', 'bottom');
    selects()[3].vm.$emit('update:modelValue', 'avg');
    await limitInput.trigger('blur');
    await flushPromises();
    const previousQueryId = chart().props('queryId');
    mocks.rankInstances.mockResolvedValueOnce(rank(['pod-refreshed']));
    vi.setSystemTime(new Date(2026, 10, 5, 12, 1));
    await wrapper.get('button').trigger('click');
    await vi.advanceTimersByTimeAsync(300);
    await flushPromises();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(3);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(
      expect.objectContaining({
        startTime: dayjs().startOf('day').unix(),
        endTime: dayjs().unix(),
        interval: 97,
        limit: 20,
        rankType: 'bottom',
        method: 'avg',
      }),
    );
    expect(chart().props('instances')).toEqual(instances(['pod-refreshed']));
    expect(chart().props('queryId')).toBeGreaterThan(previousQueryId);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(24);
    wrapper.unmount();
  });

  it.each([
    ['相对', true],
    ['绝对', false],
  ])('当自动刷新使用%s范围时，应查询对应时间并只请求一轮', async (_, relative) => {
    const { wrapper } = await mountSlider();
    const end = dayjs();
    const start = end.subtract(24, 'hour');
    const dateRange = relative ? ['now-24h', 'now'] : [start.valueOf(), end.valueOf()];
    wrapper
      .getComponent({ name: 'DatePickerStub' })
      .vm.$emit('update:model-value', dateRange, [{ dayjs: start }, { dayjs: end }]);
    await flushPromises();

    vi.setSystemTime(end.add(1, 'minute').toDate());
    await mocks.registerPoll.mock.lastCall?.[0]();
    await flushPromises();
    const time = { startTime: start.unix() + (relative ? 60 : 0), endTime: end.unix() + (relative ? 60 : 0) };
    expect(mocks.rankInstances).toHaveBeenCalledTimes(3);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(expect.objectContaining(time));
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(24);
    for (const [query] of mocks.getInstanceTimeSeries.mock.calls.slice(-8)) {
      expect(query).toMatchObject(time);
    }
    wrapper.unmount();
  });

  it('关闭手选监控并取消列表勾选后，再次打开使用排名结果', async () => {
    const { wrapper, selects, chart } = await mountSlider({ prod: ['pod-0'] });
    expect(mocks.rankInstances).not.toHaveBeenCalled();
    expect(chart().props('instances')).toEqual(instances(['pod-0']));
    expect(selects()[1].props('disabled')).toBe(true);

    await wrapper.setProps({ isShow: false });
    await wrapper.setProps({ initialSelection: undefined, isShow: true });
    chart().vm.$emit('plot-width-change', 450);
    await flushPromises();

    expect(selects()[0].props('modelValue')).toEqual([]);
    expect(selects()[1].props('disabled')).toBe(false);
    expect(mocks.rankInstances).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ limit: 10, rankType: 'top' }),
    );
    expect(chart().props('instances')).toEqual(instances(['pod-1', 'pod-2']));
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    for (const [query] of mocks.getInstanceTimeSeries.mock.calls.slice(-8)) {
      expect(query.instances).toEqual(['pod-1', 'pod-2']);
    }
    wrapper.unmount();
  });

  it('初始打开复用 2000 个实例快照，未搜索不渲染候选', async () => {
    const { wrapper, selects } = await mountSlider();
    expect(mocks.listAppInstances).not.toHaveBeenCalled();
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(0);
    expect(wrapper.get('[data-testid="instance-selector"]').attributes('data-placeholder')).toBe('全部实例（2000）');
    expect(selects()[0].props('modelValue')).toEqual([]);
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('pod-');
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(100);
    expect(wrapper.text()).toContain('仅展示前 100 个结果，请缩小搜索范围');
    wrapper.unmount();
  });

  it('保留隐藏选择，限制 50 个手选实例，清空后恢复排名', async () => {
    const { wrapper, selects, chart } = await mountSlider({ prod: ['pod-1999'] });
    expect(mocks.rankInstances).not.toHaveBeenCalled();
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
    expect(chart().props('instances')).toEqual(instances(['pod-1999']));
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('pod-');
    const tree = wrapper.getComponent({ name: 'TreeStub' });
    const nodes = Array.from({ length: 49 }, (_, index) => ({ id: instanceKey(`pod-${index}`), name: `pod-${index}` }));
    tree.vm.$emit('node-checked', nodes);
    await nextTick();
    tree.vm.$emit('node-checked', [...nodes, { id: instanceKey('pod-49'), name: 'pod-49' }]);
    await flushPromises();
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'warning', message: '最多选择 50 个实例' });
    expect(selects()[0].props('modelValue')).toHaveLength(50);
    expect(selects()[0].props('modelValue')).toContain(instanceKey('pod-1999'));
    selects()[0].vm.$emit('toggle', false);
    await flushPromises();
    expect(chart().props('instances')).toHaveLength(50);
    expect(selects()[1].props('disabled')).toBe(true);
    expect(mocks.rankInstances).not.toHaveBeenCalled();
    selects()[0].vm.$emit('clear');
    selects()[0].vm.$emit('toggle', false);
    await flushPromises();
    expect(chart().props('instances')).toEqual(instances(['pod-1', 'pod-2']));
    expect(selects()[1].props('disabled')).toBe(false);
    expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
    wrapper.unmount();
  });

  it('汇总表遵守排名顺序，复用图表的最新值并保留缺少指标的实例', async () => {
    mocks.rankInstances.mockResolvedValueOnce(rank(['pod-b', 'pod-a', 'pod-empty']));
    mocks.getInstanceTimeSeries.mockImplementation((query: { metricKey: string }) =>
      Promise.resolve({
        [query.metricKey]: metric(['pod-a', 'pod-b'], query.metricKey.includes('limit') ? 'percent' : 'cores', 0.25),
      }),
    );
    const { wrapper, rows } = await mountSlider();
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
    expect(rows()).toEqual([
      { name: 'pod-b', cpuLimitUsageRate: 25, memLimitUsageRate: 25, cpuUsage: 0.25, memUsage: 0.25 },
      { name: 'pod-a', cpuLimitUsageRate: 25, memLimitUsageRate: 25, cpuUsage: 0.25, memUsage: 0.25 },
      {
        name: 'pod-empty',
        cpuLimitUsageRate: undefined,
        memLimitUsageRate: undefined,
        cpuUsage: undefined,
        memUsage: undefined,
      },
    ]);
    wrapper.unmount();
  });

  it('排名为空时不请求任何时序', async () => {
    mocks.rankInstances.mockResolvedValueOnce([]);
    const { wrapper, chart, rows } = await mountSlider();
    expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
    expect(mocks.getInstanceTimeSeries).not.toHaveBeenCalled();
    expect(chart().props('instances')).toEqual([]);
    expect(rows()).toEqual([]);
    wrapper.unmount();
  });

  it('多环境合并排名后按环境查询，保留同名实例及全局顺序', async () => {
    mocks.rankInstances
      .mockResolvedValueOnce([{ instance: 'web-0', value: 1 }])
      .mockResolvedValueOnce([{ instance: 'web-0', value: 2 }]);
    mocks.getInstanceTimeSeries.mockImplementation(
      (query: { envName: string; instances: string[]; metricKey: string }) =>
        Promise.resolve({ [query.metricKey]: metric(query.instances, 'cores', query.envName === 'dev' ? 1 : 2) }),
    );
    const { wrapper, chart, rows } = await mountSlider(undefined, [
      { envName: 'dev', instances: ['web-0'] },
      { envName: 'prod', instances: ['web-0'] },
    ]);
    expect(mocks.rankInstances.mock.calls.map(([query]) => query.envName)).toEqual(['dev', 'prod']);
    expect(chart().props('instances')).toEqual([...instances(['web-0']), ...instances(['web-0'], 'dev')]);
    expect(rows().map((row: { cpuUsage: number; name: string }) => [row.name, row.cpuUsage])).toEqual([
      ['prod/web-0', 2],
      ['dev/web-0', 1],
    ]);
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    wrapper.unmount();
  });

  it('时间对比沿用当前名单，调整时间范围则用新参数重查排名和时序', async () => {
    const { wrapper, chart, selects } = await mountSlider();
    const initialInstances = chart().props('instances');
    wrapper.getComponent({ name: 'SwitcherStub' }).vm.$emit('update:modelValue', true);
    await nextTick();
    selects()[4].vm.$emit('update:modelValue', [3600]);
    await flushPromises();
    expect(chart().props('compareOffsets')).toEqual([3600]);
    expect(chart().props('instances')).toBe(initialInstances);
    expect(mocks.rankInstances).toHaveBeenCalledTimes(1);
    const from = dayjs().subtract(3, 'hour');
    const to = dayjs().subtract(1, 'hour');
    wrapper
      .getComponent({ name: 'DatePickerStub' })
      .vm.$emit('update:model-value', undefined, [{ dayjs: from }, { dayjs: to }]);
    await flushPromises();
    const time = { startTime: from.unix(), endTime: to.unix(), interval: 60 };
    expect(mocks.rankInstances).toHaveBeenCalledTimes(2);
    expect(mocks.rankInstances).toHaveBeenLastCalledWith(expect.objectContaining(time));
    expect(mocks.getInstanceTimeSeries).toHaveBeenLastCalledWith(expect.objectContaining(time));
    wrapper.unmount();
  });

  it('过期排名不会发起时序，旧图表结果也不会覆盖新汇总', async () => {
    const { promise, resolve } = deferred<ReturnType<typeof rank>>();
    const { wrapper, chart, selects, rows, emitMetric } = await mountSlider();
    const oldQueryId = chart().props('queryId');
    mocks.rankInstances.mockReturnValueOnce(promise).mockResolvedValueOnce(rank(['pod-new']));
    selects()[3].vm.$emit('update:modelValue', 'max');
    await flushPromises();
    selects()[3].vm.$emit('update:modelValue', 'avg');
    await flushPromises();
    resolve(rank(['pod-stale']));
    await flushPromises();
    await emitMetric('cpu_usage', metric(['pod-new'], 'cores', 99), oldQueryId);
    expect(chart().props('instances')).toEqual(instances(['pod-new']));
    expect(rows()[0]).toMatchObject({ name: 'pod-new', cpuUsage: 1 });
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(16);
    wrapper.unmount();
  });

  it('实例快照更新只改变搜索候选，保留已提交的手选实例', async () => {
    const { wrapper, chart, selects } = await mountSlider({ prod: ['pod-0'] });
    const committed = chart().props('queryId');
    await wrapper.setProps({
      instanceGroups: [{ envName: 'prod', instances: ['pod-new-0'] }],
    });
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('pod-new-0');
    await flushPromises();
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(1);
    expect(wrapper.get('[data-testid="instance-candidate"]').attributes('value')).toBe(instanceKey('pod-new-0'));
    expect(selects()[0].props('modelValue')).toEqual([instanceKey('pod-0')]);
    expect(chart().props('queryId')).toBe(committed);
    expect(chart().props('instances')).toEqual(instances(['pod-0']));
    expect(mocks.rankInstances).not.toHaveBeenCalled();
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(8);
    wrapper.unmount();
  });
});
