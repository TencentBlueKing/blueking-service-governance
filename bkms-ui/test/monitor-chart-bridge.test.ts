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

import { createApp } from 'vue';

import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  type MonitorChartApiResult,
  type MonitorInstance,
  type MonitorMetricQuery,
  fetchMonitorMetric,
  getCachedMetric,
  getMonitorInstanceKey,
  getMonitorInstanceLabel,
  registerMonitorChartApi,
} from '~/pages/application/detail/deploy/monitor-chart-bridge';

import type {
  GetInstanceTimeSeriesRequest,
  MetricTimeSeries,
  TimeSeriesItem,
} from '~/@types/v1/bkintegrations-bkmonitor';

const { requestMetric } = vi.hoisted(() => ({
  requestMetric: vi.fn<(params: GetInstanceTimeSeriesRequest) => Promise<Record<string, MetricTimeSeries>>>(),
}));
vi.mock('~/api/modules/v1', () => ({
  BkintegrationsBkmonitorService: { getInstanceTimeSeries: requestMetric },
}));
vi.mock('~/modules/i18n', () => ({
  i18n: { global: { t: (key: string) => key } },
}));

const baseQuery: MonitorMetricQuery = {
  appID: 'demo-app',
  envName: 'prod',
  metricKey: 'cpu_usage',
  startTime: 1781366400,
  endTime: 1781370000,
  interval: 60,
};

function chartApi() {
  const app = createApp({});
  registerMonitorChartApi(app);
  const api: {
    bkmonitor: { getInstanceTimeSeries: (params: Record<string, unknown>) => Promise<MonitorChartApiResult> };
  } = app.config.globalProperties.$api;
  return api.bkmonitor.getInstanceTimeSeries;
}

function chartParams(queryId: number, extra: Record<string, unknown> = {}) {
  return {
    ...baseQuery,
    start_time: baseQuery.startTime,
    end_time: baseQuery.endTime,
    queryId,
    ...extra,
  };
}

function response(items: TimeSeriesItem[]): Record<string, MetricTimeSeries> {
  return { cpu_usage: { unit: 'cores', series: items } };
}

function selected(instance: string, envName = 'prod'): MonitorInstance {
  return { envName, instance };
}

function series(instance: string, avg = 1): TimeSeriesItem {
  return { instance, dataPoints: [[avg, baseQuery.startTime * 1000]], stat: { avg: [0, avg], last: [0, avg] } };
}

beforeEach(() => {
  requestMetric.mockReset();
});

describe('monitor chart query modes', () => {
  it('queries all instances with ranking defaults and second timestamps', async () => {
    requestMetric.mockResolvedValue(response([series('pod-a')]));
    await fetchMonitorMetric({ ...baseQuery, all: true });
    expect(requestMetric).toHaveBeenCalledWith({
      ...baseQuery,
      all: true,
      limit: 10,
      limitType: 'top',
      method: 'sum',
    });
    expect(requestMetric.mock.calls[0][0]).not.toHaveProperty('instances');
  });

  it.each([1, 30, 50])('shows all %i explicitly selected instances without ranking fields', async count => {
    const instances = Array.from({ length: count }, (_, index) => `pod-${index}`);
    requestMetric.mockResolvedValue(response(instances.map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: instances.map(instance => selected(instance)),
      all: false,
      limit: 10,
      limitType: 'bottom',
      method: 'max',
    });
    expect(result?.series).toHaveLength(count);
    expect(requestMetric).toHaveBeenCalledWith({ ...baseQuery, instances, method: 'max' });
    const params = requestMetric.mock.calls[0][0];
    expect(params).not.toHaveProperty('all');
    expect(params).not.toHaveProperty('limit');
    expect(params).not.toHaveProperty('limitType');
  });

  it.each(['top', 'bottom'] as const)('ranks the merged environment results for %s', async limitType => {
    requestMetric.mockImplementation(async ({ envName }) =>
      response(
        envName === 'prod' ? [series('prod-a', 5), series('prod-b', 1)] : [series('test-a', 3), series('test-b', 8)],
      ),
    );
    const result = await fetchMonitorMetric({
      ...baseQuery,
      envNames: ['prod', 'test'],
      all: true,
      limit: 2,
      limitType,
    });
    expect(result?.series?.map(item => [item.instance, item.envName])).toEqual(
      limitType === 'top'
        ? [
            ['test-b', 'test'],
            ['prod-a', 'prod'],
          ]
        : [
            ['prod-b', 'prod'],
            ['test-a', 'test'],
          ],
    );
  });

  it('groups explicit selections by environment without truncating the combined result', async () => {
    requestMetric.mockImplementation(async ({ instances }) => response((instances ?? []).map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: [selected('prod-a'), selected('test-a', 'test'), selected('test-b', 'test')],
      limit: 1,
    });
    expect(result?.series).toHaveLength(3);
    expect(requestMetric.mock.calls.map(([params]) => [params.envName, params.instances])).toEqual([
      ['prod', ['prod-a']],
      ['test', ['test-a', 'test-b']],
    ]);
  });

  it('does not turn an empty explicit selection into an all-instances request', async () => {
    expect(await fetchMonitorMetric({ ...baseQuery, instances: [] })).toBeUndefined();
    expect(requestMetric).not.toHaveBeenCalled();
  });

  it('queries same-name instances in their own environments and deduplicates only within each environment', async () => {
    requestMetric.mockImplementation(async ({ instances }) => response((instances ?? []).map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: [selected('same-pod'), selected('same-pod', 'test'), selected('same-pod')],
      limit: 1,
      limitType: 'bottom',
    });
    expect(result?.series?.map(item => [item.envName, item.instance])).toEqual([
      ['prod', 'same-pod'],
      ['test', 'same-pod'],
    ]);
    expect(requestMetric.mock.calls.map(([params]) => [params.envName, params.instances])).toEqual([
      ['prod', ['same-pod']],
      ['test', ['same-pod']],
    ]);
    for (const [params] of requestMetric.mock.calls) {
      expect(params).not.toHaveProperty('all');
      expect(params).not.toHaveProperty('limit');
      expect(params).not.toHaveProperty('limitType');
    }
  });
});

describe('monitor chart comparison and cache', () => {
  it('locks comparison to current returned instances and aligns past points to the current time axis', async () => {
    requestMetric.mockResolvedValueOnce(response([series('current-top')])).mockResolvedValueOnce(
      response([
        {
          ...series('current-top'),
          dataPoints: [[2, (baseQuery.startTime - 3600) * 1000]],
        },
      ]),
    );
    const result = await chartApi()(chartParams(101, { all: true, compareOffsets: [3600] }));
    expect(requestMetric.mock.calls[1][0]).toEqual({
      ...baseQuery,
      instances: ['current-top'],
      method: 'sum',
      startTime: baseQuery.startTime - 3600,
      endTime: baseQuery.endTime - 3600,
    });
    expect(result.series[1].datapoints).toEqual([[2, baseQuery.startTime * 1000]]);
    expect(getCachedMetric('cpu_usage', 101, true)?.series).toHaveLength(1);
    expect(getCachedMetric('cpu_usage', 101)?.series).toHaveLength(2);
  });

  it('skips comparison requests for empty current results and for multiple environments', async () => {
    const api = chartApi();
    requestMetric.mockResolvedValue(response([]));
    await api(chartParams(102, { all: true, compareOffsets: [3600] }));
    expect(requestMetric).toHaveBeenCalledTimes(1);
    requestMetric.mockClear();
    requestMetric.mockResolvedValue(response([series('pod')]));
    await api(chartParams(103, { all: true, envNames: ['prod', 'test'], compareOffsets: [3600] }));
    expect(requestMetric).toHaveBeenCalledTimes(2);
  });

  it('deduplicates simultaneous identical requests but isolates method and ranking changes', async () => {
    const api = chartApi();
    let resolvePending!: (value: Record<string, MetricTimeSeries>) => void;
    requestMetric.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePending = resolve;
        }),
    );
    const first = api(chartParams(104, { all: true }));
    const duplicate = api(chartParams(104, { all: true }));
    expect(requestMetric).toHaveBeenCalledTimes(1);
    resolvePending(response([series('pod')]));
    await Promise.all([first, duplicate]);
    requestMetric.mockResolvedValue(response([series('pod')]));
    await api(chartParams(104, { all: true, method: 'avg' }));
    await api(chartParams(104, { all: true, method: 'avg', limit: 5, limitType: 'bottom' }));
    expect(requestMetric).toHaveBeenCalledTimes(3);
  });

  it('does not let a late old response replace the newest query or let table lookups contaminate it', async () => {
    const api = chartApi();
    let resolveOld!: (value: Record<string, MetricTimeSeries>) => void;
    requestMetric
      .mockImplementationOnce(
        () =>
          new Promise(resolve => {
            resolveOld = resolve;
          }),
      )
      .mockResolvedValueOnce(response([series('new-pod')]))
      .mockResolvedValueOnce(response([series('table-only')]));
    const old = api(chartParams(105, { all: true }));
    await api(chartParams(106, { instances: [selected('new-pod')] }));
    await fetchMonitorMetric({ ...baseQuery, instances: [selected('table-only')] });
    resolveOld(response([series('old-pod')]));
    expect((await old).series).toEqual([]);
    expect(getCachedMetric('cpu_usage', 105)).toBeUndefined();
    expect(getCachedMetric('cpu_usage', 106, true)?.series?.map(item => item.instance)).toEqual(['new-pod']);
  });

  it('keeps multi-environment curve labels and statistic lookups distinct for same-name instances', async () => {
    requestMetric.mockImplementation(async ({ envName }) => response([series('same-pod', envName === 'prod' ? 5 : 2)]));
    const result = await chartApi()(chartParams(107, { all: true, envNames: ['prod', 'test'] }));
    expect(result.series.map(item => [item.target, item.name, item.alias])).toEqual([
      ['prod/same-pod', 'prod/same-pod', 'prod/same-pod'],
      ['test/same-pod', 'test/same-pod', 'test/same-pod'],
    ]);
    const current = getCachedMetric('cpu_usage', 107, true)?.series ?? [];
    const statisticsByLabel = new Map(current.map(item => [getMonitorInstanceLabel(item, true), item.stat]));
    expect(result.series.map(item => statisticsByLabel.get(item.target)?.avg?.[1])).toEqual([5, 2]);
    expect(new Set(current.map(getMonitorInstanceKey)).size).toBe(2);
  });

  it('uses the queried environment range for labels even when global ranking returns one environment', async () => {
    requestMetric.mockImplementation(async ({ envName }) => response([series('same-pod', envName === 'prod' ? 5 : 2)]));
    const result = await chartApi()(chartParams(108, { all: true, envNames: ['prod', 'test'], limit: 1 }));
    expect(result.series.map(item => item.target)).toEqual(['prod/same-pod']);
    expect(getMonitorInstanceLabel({ envName: 'prod', instance: '__CMP__昨天-same-pod' })).toBe('昨天-same-pod');
    expect(getMonitorInstanceLabel({ envName: 'prod', instance: '__CMP__昨天-same-pod' }, true)).toBe(
      'prod/昨天-same-pod',
    );
  });
});
