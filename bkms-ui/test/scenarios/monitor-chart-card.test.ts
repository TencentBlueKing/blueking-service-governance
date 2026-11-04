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

import type { Ref } from 'vue';

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ChartCardItem from '~/pages/application/detail/deploy/chart-card-item.vue';

import type * as MonitorChartBridge from '~/pages/application/detail/deploy/monitor-chart-bridge';
import type { MonitorMetricTimeSeries } from '~/pages/application/detail/deploy/monitor-chart-bridge';

/* eslint-disable vue/one-component-per-file */
const mocks = vi.hoisted(() => ({
  getCachedMetric: vi.fn(),
  selectLegend: vi.fn(),
  legends: [] as { alias: string; color: string; name: string; show: boolean }[],
}));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'demo-app' }) }));
vi.mock('~/api/modules/v1', () => ({ BkintegrationsBkmonitorService: {} }));
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('echarts/core', () => ({ connect: vi.fn() }));
vi.mock('~/pages/application/detail/deploy/monitor-chart-bridge', async importOriginal => ({
  ...(await importOriginal<typeof MonitorChartBridge>()),
  getCachedMetric: mocks.getCachedMetric,
}));
vi.mock('@blueking/monitor-vue3-components', async () => {
  const { defineComponent, h, ref } = await import('vue');
  return {
    ExploreChart: defineComponent({
      name: 'ExploreChartStub',
      setup(_, { expose }) {
        expose({ legendData: ref(mocks.legends), handleSelectLegend: mocks.selectLegend });
        return () => h('div');
      },
    }),
  };
});
vi.mock('@blueking/table', async () => {
  const { defineComponent, h, inject, provide, toRef } = await import('vue');
  return {
    Table: defineComponent({
      name: 'LegendTableStub',
      props: { data: { type: Array, default: () => [] } },
      setup(props, { slots }) {
        provide('legend-test-rows', toRef(props, 'data'));
        return () => h('div', slots.default?.());
      },
    }),
    TableColumn: defineComponent({
      name: 'LegendTableColumnStub',
      props: { field: { type: String, default: '' } },
      setup(props, { slots }) {
        const rows = inject<Ref<Record<string, unknown>[]>>('legend-test-rows');
        return () =>
          h(
            'div',
            rows?.value.map(row =>
              h(
                'div',
                { 'data-testid': props.field === 'name' ? 'legend-name-cell' : undefined },
                slots.default?.({ row }) ?? String(row[props.field] ?? ''),
              ),
            ),
          );
      },
    }),
  };
});

function mountCard() {
  return mount(ChartCardItem, {
    props: {
      all: true,
      instances: [],
      envName: 'prod',
      envNames: ['prod', 'test'],
      metricKey: 'cpu_usage',
      queryId: 1,
    },
    global: { mocks: { $t: (key: string) => key } },
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.legends = [];
});

describe('实例监控图例的环境归属', () => {
  it('同名实例分别显示自身统计，点击图例使用各自环境和名称', async () => {
    mocks.legends = [
      { alias: 'prod/same-pod', color: '#ff0000', name: 'prod/same-pod', show: true },
      { alias: 'test/same-pod', color: '#0000ff', name: 'test/same-pod', show: true },
    ];
    const metric: MonitorMetricTimeSeries = {
      unit: 'cores',
      series: [
        { envName: 'prod', instance: 'same-pod', stat: { min: [0, 1], max: [0, 2], avg: [0, 3] } },
        { envName: 'test', instance: 'same-pod', stat: { min: [0, 4], max: [0, 5], avg: [0, 6] } },
      ],
    };
    mocks.getCachedMetric.mockReturnValue(metric);
    const wrapper = mountCard();
    const table = wrapper.getComponent({ name: 'LegendTableStub' });
    await wrapper.vm.$nextTick();
    expect(table.props('data')).toEqual([
      expect.objectContaining({ alias: 'prod/same-pod', min: '1.00', max: '2.00', avg: '3.00' }),
      expect.objectContaining({ alias: 'test/same-pod', min: '4.00', max: '5.00', avg: '6.00' }),
    ]);
    const names = wrapper.findAll('[data-testid="legend-name-cell"] .cursor-pointer');
    await names[0].trigger('click');
    await names[1].trigger('click');
    expect(mocks.selectLegend.mock.calls).toEqual([
      [{ actionType: 'click', item: { name: 'prod/same-pod' } }],
      [{ actionType: 'click', item: { name: 'test/same-pod' } }],
    ]);
    wrapper.unmount();
  });

  it('全局排名只返回一个环境时仍按查询环境范围匹配图例', async () => {
    mocks.legends = [{ alias: 'prod/same-pod', color: '#ff0000', name: 'prod/same-pod', show: true }];
    const metric: MonitorMetricTimeSeries = {
      unit: 'cores',
      series: [{ envName: 'prod', instance: 'same-pod', stat: { min: [0, 1], max: [0, 2], avg: [0, 3] } }],
    };
    mocks.getCachedMetric.mockReturnValue(metric);
    const wrapper = mountCard();
    await wrapper.vm.$nextTick();
    expect(wrapper.getComponent({ name: 'LegendTableStub' }).props('data')).toEqual([
      expect.objectContaining({ name: 'prod/same-pod', min: '1.00', max: '2.00', avg: '3.00' }),
    ]);
    wrapper.unmount();
  });
});
