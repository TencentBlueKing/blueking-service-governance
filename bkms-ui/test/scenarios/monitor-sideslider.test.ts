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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import MonitorSideslider from '~/pages/application/detail/deploy/monitor-sideslider.vue';

import type { MonitorInstanceGroup } from '~/pages/application/detail/deploy/instance-list/types';
import type {
  fetchMonitorMetric,
  MonitorInstance,
  MonitorMetricQuery,
  MonitorMetricTimeSeries,
} from '~/pages/application/detail/deploy/monitor-chart-bridge';

// UI 桩组件共同保留侧栏的查询与选择事件链路。
/* eslint-disable vue/one-component-per-file */

const mocks = vi.hoisted(() => ({
  chartQuery: vi.fn(),
  fetchMonitorMetric: vi.fn(),
  getInstanceTimeSeries: vi.fn(),
  lifecycleRequests: false,
  listAppInstances: vi.fn(),
  message: vi.fn(),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values: unknown[] = []) => key.replace(/\{(\d+)\}/g, (_, index) => String(values[index])),
  }),
}));
vi.mock('~/api/modules/v1', () => ({
  InstanceService: { listAppInstances: mocks.listAppInstances },
  BkintegrationsBkmonitorService: { getInstanceTimeSeries: mocks.getInstanceTimeSeries },
}));
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/stores/deploy-env', () => ({ useDeployEnvStore: () => ({ currentEnv: 'prod' }) }));
vi.mock('~/composables/use-interval', () => ({ default: () => ({ start: vi.fn(), stop: vi.fn() }) }));
vi.mock('~/pages/application/detail/deploy/monitor-chart-bridge', () => ({
  fetchMonitorMetric: mocks.fetchMonitorMetric,
  getMonitorInstanceKey: ({ envName, instance }: MonitorInstance) => JSON.stringify([envName, instance]),
  getMonitorInstanceLabel: ({ envName, instance }: MonitorInstance, multiEnv = false) =>
    multiEnv ? `${envName}/${instance}` : instance,
  isPercentUnit: (unit?: string) => unit === 'percent',
  setChartTimeRange: vi.fn(),
}));
vi.mock('@blueking/date-picker', () => ({
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
  const { defineComponent, inject, ref, watch } = await import('vue');
  const { default: date } = await import('dayjs');
  return {
    __esModule: true,
    default: defineComponent({
      name: 'ChartCardStub',
      props: {
        all: Boolean,
        instances: { type: Array, default: () => [] },
        limit: { type: Number, default: undefined },
        limitType: { type: String, default: undefined },
        method: { type: String, default: undefined },
        interval: { type: Number, default: undefined },
        metricKey: { type: String, required: true },
        queryId: { type: Number, required: true },
        queryEnabled: Boolean,
        envName: { type: String, default: undefined },
        envNames: { type: Array, default: undefined },
        compareOffsets: { type: Array, default: () => [] },
      },
      emits: ['metric-data', 'plot-width-change'],
      setup(props, { emit }) {
        const timeRange = inject('timeRange', ref<string[]>([]));
        watch(
          () => ({ queryId: props.queryId, queryEnabled: props.queryEnabled }),
          async ({ queryId, queryEnabled }) => {
            if (!mocks.lifecycleRequests || !queryEnabled) return;
            const query = {
              all: props.all,
              appID: 'demo-app',
              envName: props.envName,
              envNames: props.envNames,
              instances: props.instances,
              interval: props.interval,
              limit: props.limit,
              limitType: props.limitType,
              method: props.method,
              metricKey: props.metricKey,
              startTime: date(timeRange.value[0]).unix(),
              endTime: date(timeRange.value[1]).unix(),
            };
            mocks.chartQuery({ ...query, queryId });
            const data = await mocks.fetchMonitorMetric(query);
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
    emits: ['update:modelValue', 'blur'],
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
    Checkbox: { render: () => null },
    Divider: slot,
    Input: input,
    Message: mocks.message,
    Progress: slot,
    Radio: { Group: slot, Button: slot },
    Select: Object.assign(select, { Option: option }),
    Sideslider: slot,
    Switcher: slot,
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

async function enableLifecycleRequests() {
  const actualBridge = await vi.importActual<{ fetchMonitorMetric: typeof fetchMonitorMetric }>(
    '~/pages/application/detail/deploy/monitor-chart-bridge',
  );
  mocks.fetchMonitorMetric.mockImplementation(actualBridge.fetchMonitorMetric);
  mocks.lifecycleRequests = true;
  mocks.getInstanceTimeSeries.mockImplementation((query: { instances?: string[]; metricKey: string }) =>
    Promise.resolve({ [query.metricKey]: metric(query.instances ?? ['pod-1', 'pod-2']) }),
  );
}

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
      directives: { 'bk-tooltips': {} },
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

beforeEach(() => {
  vi.clearAllMocks();
  mocks.lifecycleRequests = false;
  mocks.fetchMonitorMetric.mockImplementation((query: MonitorMetricQuery) =>
    Promise.resolve(metric(query.instances || [])),
  );
});

describe('实例监控侧栏', () => {
  it('一次打开在图表宽度就绪后仅请求一轮八图三列', async () => {
    vi.setSystemTime(new Date('2026-11-04T12:00:00+08:00'));
    await enableLifecycleRequests();
    const { wrapper, chart } = await mountSlider(undefined, undefined, false);
    try {
      await wrapper.setProps({ isShow: true });
      await flushPromises();
      expect(chart().props('queryEnabled')).toBe(false);
      expect(mocks.getInstanceTimeSeries).not.toHaveBeenCalled();
      chart().vm.$emit('plot-width-change', 450);
      await flushPromises();
      expect(chart().props('queryEnabled')).toBe(true);
      expect(chart().props('interval')).toBe(384);
      for (const card of wrapper.findAllComponents({ name: 'ChartCardStub' })) {
        card.vm.$emit('plot-width-change', 450);
      }
      chart().vm.$emit('plot-width-change', 0);
      await flushPromises();
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      expect(new Set(mocks.chartQuery.mock.calls.map(([query]) => query.queryId)).size).toBe(1);
    } finally {
      wrapper.unmount();
      vi.useRealTimers();
    }
  });

  it('手动固定汇聚周期后图表宽度就绪不额外请求', async () => {
    await enableLifecycleRequests();
    const { wrapper, selects, chart } = await mountSlider();
    try {
      selects()[2].vm.$emit('update:modelValue', 60);
      await flushPromises();
      const beforeWidth = mocks.getInstanceTimeSeries.mock.calls.length;
      chart().vm.$emit('plot-width-change', 500);
      await flushPromises();
      expect(beforeWidth).toBe(22);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(beforeWidth);
      expect(chart().props('interval')).toBe(60);
    } finally {
      wrapper.unmount();
    }
  });

  it('auto 尺寸变化仅在实际步长改变时重拉一轮', async () => {
    await enableLifecycleRequests();
    const { wrapper, chart } = await mountSlider();
    try {
      expect(chart().props('interval')).toBe(384);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      chart().vm.$emit('plot-width-change', 451);
      await flushPromises();
      expect(chart().props('interval')).toBe(384);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      chart().vm.$emit('plot-width-change', 500);
      await flushPromises();
      expect(chart().props('interval')).toBe(346);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(22);
      expect(new Set(mocks.chartQuery.mock.calls.map(([query]) => query.queryId)).size).toBe(2);
    } finally {
      wrapper.unmount();
    }
  });

  it('关闭后再次打开等待新绘图区宽度再请求', async () => {
    await enableLifecycleRequests();
    const { wrapper, chart } = await mountSlider();
    try {
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      await wrapper.setProps({ isShow: false });
      await wrapper.setProps({ isShow: true });
      await flushPromises();
      expect(chart().props('queryEnabled')).toBe(false);
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      chart().vm.$emit('plot-width-change', 450);
      await flushPromises();
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(22);
    } finally {
      wrapper.unmount();
    }
  });

  it('手动刷新在尺寸和查询条件相同时仍正常重拉一轮', async () => {
    await enableLifecycleRequests();
    const { wrapper } = await mountSlider();
    try {
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(11);
      await wrapper.get('button').trigger('click');
      await new Promise(resolve => setTimeout(resolve, 350));
      await flushPromises();
      expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(22);
      expect(new Set(mocks.chartQuery.mock.calls.map(([query]) => query.queryId)).size).toBe(2);
    } finally {
      wrapper.unmount();
    }
  });

  it('初始打开复用 2000 个实例快照，不额外取实例列表且未搜索不渲染候选', async () => {
    const { wrapper, selects, chart } = await mountSlider();
    expect(mocks.listAppInstances).not.toHaveBeenCalled();
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(0);
    expect(wrapper.get('[data-testid="instance-selector"]').attributes('data-placeholder')).toBe('全部实例（2000）');
    expect(selects()[0].props('modelValue')).toEqual([]);
    expect(chart().props()).toMatchObject({ all: true, instances: [], limit: 10, limitType: 'top', method: 'sum' });
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('pod-');
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(100);
    expect(wrapper.text()).toContain('仅展示前 100 个结果，请缩小搜索范围');
    wrapper.unmount();
  });

  it('搜索时保留隐藏选择，超过 50 个时阻止新增，清空后恢复全部模式与 Limit', async () => {
    const { wrapper, selects, chart } = await mountSlider({ prod: ['pod-1999'] });
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
    await nextTick();
    expect(chart().props('instances')).toHaveLength(50);
    expect(chart().props('limit')).toBeUndefined();
    expect(selects()[1].props('disabled')).toBe(true);
    selects()[0].vm.$emit('clear');
    await nextTick();
    expect(chart().props()).toMatchObject({ all: true, instances: [], limit: 10, limitType: 'top' });
    expect(selects()[1].props('disabled')).toBe(false);
    wrapper.unmount();
  });

  it('全部模式汇总表按 CPU 当前期顺序补查其余三列，不向补查传排名', async () => {
    mocks.fetchMonitorMetric.mockImplementation((query: MonitorMetricQuery) =>
      Promise.resolve(
        metric(query.instances || [], query.metricKey === 'memory_limit_usage' ? 'percent' : 'cores', 0.5),
      ),
    );
    const { wrapper, emitMetric, rows } = await mountSlider();
    await emitMetric('memory_usage', metric(['pod-other'], 'bytes', 999));
    await emitMetric('cpu_limit_usage', metric(['pod-b', 'pod-a'], 'percent', 0.2));
    expect(mocks.fetchMonitorMetric).toHaveBeenCalledTimes(3);
    expect(mocks.fetchMonitorMetric.mock.calls.map(([query]) => query.metricKey)).toEqual([
      'memory_limit_usage',
      'cpu_usage',
      'memory_usage',
    ]);
    for (const [query] of mocks.fetchMonitorMetric.mock.calls) {
      expect(query.instances).toEqual(instances(['pod-b', 'pod-a']));
      expect(query).toMatchObject({ method: 'sum' });
      expect(query).not.toHaveProperty('instanceEnvMap');
      expect(query).not.toHaveProperty('all');
      expect(query).not.toHaveProperty('limit');
      expect(query).not.toHaveProperty('limitType');
    }
    expect(rows()).toEqual([
      { name: 'pod-b', cpuLimitUsageRate: 20, memLimitUsageRate: 50, cpuUsage: 0.5, memUsage: 0.5 },
      { name: 'pod-a', cpuLimitUsageRate: 20, memLimitUsageRate: 50, cpuUsage: 0.5, memUsage: 0.5 },
    ]);
    wrapper.unmount();
  });

  it('手选 30 个时复用图表结果，保留所有行且不额外补查', async () => {
    const names = Array.from({ length: 30 }, (_, index) => `pod-${index}`);
    const { wrapper, chart, emitMetric, rows } = await mountSlider({ prod: names });
    expect(chart().props()).toMatchObject({ all: false, instances: instances(names) });
    expect(chart().props('limit')).toBeUndefined();
    for (const key of ['cpu_limit_usage', 'memory_limit_usage', 'cpu_usage', 'memory_usage']) {
      await emitMetric(key, metric([...names].reverse(), key.includes('limit') ? 'percent' : 'cores', 0.25));
    }
    expect(rows()).toHaveLength(30);
    expect(rows().map((row: { name: string }) => row.name)).toEqual(names);
    expect(rows()[29]).toMatchObject({ cpuLimitUsageRate: 25, memLimitUsageRate: 25, cpuUsage: 0.25, memUsage: 0.25 });
    expect(mocks.fetchMonitorMetric).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('多环境同名实例可独立勾选，移除一个标签不影响另一个环境', async () => {
    const { wrapper, selects, chart } = await mountSlider(undefined, [
      { envName: 'dev', instances: ['web-0'] },
      { envName: 'prod', instances: ['web-0'] },
    ]);
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('web-0');
    const candidates = wrapper.findAll('[data-testid="instance-candidate"]');
    expect(candidates.map(candidate => candidate.attributes('value'))).toEqual([
      instanceKey('web-0', 'dev'),
      instanceKey('web-0', 'prod'),
    ]);
    await candidates[0].setValue(true);
    selects()[0].vm.$emit('toggle', false);
    await nextTick();
    expect(chart().props('instances')).toEqual(instances(['web-0'], 'dev'));
    await candidates[1].setValue(true);
    selects()[0].vm.$emit('toggle', false);
    await nextTick();
    expect(chart().props('instances')).toEqual([
      { envName: 'dev', instance: 'web-0' },
      { envName: 'prod', instance: 'web-0' },
    ]);
    selects()[0].vm.$emit('tag-remove', instanceKey('web-0', 'dev'));
    await nextTick();
    expect(selects()[0].props('modelValue')).toEqual([instanceKey('web-0', 'prod')]);
    expect(chart().props('instances')).toEqual(instances(['web-0'], 'prod'));
    wrapper.unmount();
  });

  it('多环境同名 CPU 排名实例分别补查三列并保留两行环境归属', async () => {
    const actualBridge = await vi.importActual<{ fetchMonitorMetric: typeof fetchMonitorMetric }>(
      '~/pages/application/detail/deploy/monitor-chart-bridge',
    );
    mocks.fetchMonitorMetric.mockImplementation(actualBridge.fetchMonitorMetric);
    mocks.getInstanceTimeSeries.mockImplementation(
      (query: { envName: string; instances: string[]; metricKey: string }) => {
        const value = query.envName === 'dev' ? 1 : 2;
        const result = metric(
          instances(query.instances, query.envName),
          query.metricKey === 'memory_limit_usage' ? 'percent' : 'cores',
          query.metricKey === 'memory_limit_usage' ? value / 4 : value,
        );
        return Promise.resolve({ [query.metricKey]: result });
      },
    );
    const { wrapper, emitMetric, rows } = await mountSlider(undefined, [
      { envName: 'dev', instances: ['web-0'] },
      { envName: 'prod', instances: ['web-0'] },
    ]);
    await emitMetric('cpu_limit_usage', {
      unit: 'percent',
      series: [
        { instance: 'web-0', envName: 'dev', stat: { last: [0, 0.2] } },
        { instance: 'web-0', envName: 'prod', stat: { last: [0, 0.4] } },
      ],
    });
    expect(mocks.fetchMonitorMetric).toHaveBeenCalledTimes(3);
    for (const [query] of mocks.fetchMonitorMetric.mock.calls) {
      expect(query.instances).toEqual([
        { envName: 'dev', instance: 'web-0' },
        { envName: 'prod', instance: 'web-0' },
      ]);
    }
    expect(mocks.getInstanceTimeSeries).toHaveBeenCalledTimes(6);
    for (const metricKey of ['memory_limit_usage', 'cpu_usage', 'memory_usage']) {
      const queries = mocks.getInstanceTimeSeries.mock.calls
        .map(([query]) => query)
        .filter(query => query.metricKey === metricKey);
      expect(queries.map(query => query.envName)).toEqual(['dev', 'prod']);
      for (const query of queries) {
        expect(query.instances).toEqual(['web-0']);
        expect(query).not.toHaveProperty('all');
        expect(query).not.toHaveProperty('limit');
        expect(query).not.toHaveProperty('limitType');
      }
    }
    expect(rows()).toEqual([
      { name: 'dev/web-0', cpuLimitUsageRate: 20, memLimitUsageRate: 25, cpuUsage: 1, memUsage: 1 },
      { name: 'prod/web-0', cpuLimitUsageRate: 40, memLimitUsageRate: 50, cpuUsage: 2, memUsage: 2 },
    ]);
    wrapper.unmount();
  });

  it('旧筛选的慢补查和图表回传不会覆盖新筛选的汇总表', async () => {
    const pending: Array<() => void> = [];
    mocks.fetchMonitorMetric.mockImplementation((query: MonitorMetricQuery) =>
      query.method === 'sum'
        ? new Promise(resolve => pending.push(() => resolve(metric(query.instances || [], 'cores', 99))))
        : Promise.resolve(metric(query.instances || [], 'cores', 2)),
    );
    const { wrapper, chart, selects, emitMetric, rows } = await mountSlider();
    const oldQueryId = chart().props('queryId');
    await emitMetric('cpu_limit_usage', metric(['pod-old'], 'percent', 0.9));
    selects()[3].vm.$emit('update:modelValue', 'max');
    await nextTick();
    await emitMetric('cpu_limit_usage', metric(['pod-new'], 'percent', 0.2));
    for (const resolve of pending) resolve();
    await flushPromises();
    await emitMetric('cpu_limit_usage', metric(['pod-stale'], 'percent', 0.8), oldQueryId);
    expect(rows()).toEqual([
      { name: 'pod-new', cpuLimitUsageRate: 20, memLimitUsageRate: 2, cpuUsage: 2, memUsage: 2 },
    ]);
    expect(mocks.fetchMonitorMetric).toHaveBeenCalledTimes(6);
    wrapper.unmount();
  });

  it('手动刷新保留 Limit 排名和汇聚方法，只更新查询版本', async () => {
    const { wrapper, chart, selects } = await mountSlider();
    selects()[1].vm.$emit('update:modelValue', 'bottom');
    selects()[3].vm.$emit('update:modelValue', 'avg');
    await wrapper.get('input[type="number"]').setValue('20');
    const previousQueryId = chart().props('queryId');
    await wrapper.get('button').trigger('click');
    await new Promise(resolve => setTimeout(resolve, 350));
    await nextTick();
    expect(chart().props()).toMatchObject({ all: true, limit: 20, limitType: 'bottom', method: 'avg' });
    expect(chart().props('queryId')).toBeGreaterThan(previousQueryId);
    expect(mocks.listAppInstances).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('实例快照更新仅改变搜索候选，不改变已提交查询或丢失已选实例', async () => {
    const { wrapper, chart, selects } = await mountSlider({ prod: ['pod-0'] });
    const committed = { ...chart().props() };
    await wrapper.setProps({
      instanceGroups: [
        {
          envName: 'prod',
          instances: Array.from({ length: 2000 }, (_, index) => (index ? `pod-${index}` : 'pod-new-0')),
        },
      ],
    });
    await wrapper.get('input[placeholder="输入实例名称搜索"]').setValue('pod-new-0');
    await flushPromises();
    expect(wrapper.findAll('[data-testid="instance-candidate"]')).toHaveLength(1);
    expect(wrapper.get('[data-testid="instance-candidate"]').attributes('value')).toBe(instanceKey('pod-new-0'));
    expect(selects()[0].props('modelValue')).toEqual([instanceKey('pod-0')]);
    for (const key of ['queryId', 'instances', 'envNames']) {
      expect(chart().props(key)).toBe(committed[key]);
    }
    expect(mocks.fetchMonitorMetric).not.toHaveBeenCalled();
    await wrapper.get('button').trigger('click');
    await new Promise(resolve => setTimeout(resolve, 350));
    await nextTick();
    expect(chart().props('queryId')).toBeGreaterThan(committed.queryId);
    expect(chart().props('instances')).toEqual(instances(['pod-0']));
    wrapper.unmount();
  });
});
