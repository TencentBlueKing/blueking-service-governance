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
  fetchRankedMonitorInstances,
  getCachedMetric,
  getMonitorInstanceKey,
  getMonitorInstanceLabel,
  invalidateMonitorChartQueries,
  registerMonitorChartApi,
} from '~/pages/application/detail/deploy/monitor-chart-bridge';

import type {
  GetInstanceTimeSeriesRequest,
  InstanceRankOutput,
  MetricTimeSeries,
  RankInstancesRequest,
  TimeSeriesItem,
} from '~/@types/v1/bkintegrations-bkmonitor';

const { requestMetric, requestRanking } = vi.hoisted(() => ({
  requestMetric: vi.fn<(params: GetInstanceTimeSeriesRequest) => Promise<Record<string, MetricTimeSeries>>>(),
  requestRanking: vi.fn<(params: RankInstancesRequest) => Promise<InstanceRankOutput[]>>(),
}));
vi.mock('~/api/modules/v1', () => ({
  BkintegrationsBkmonitorService: { getInstanceTimeSeries: requestMetric, rankInstances: requestRanking },
}));
vi.mock('~/modules/i18n', () => ({
  i18n: { global: { t: (key: string) => key } },
}));

const baseQuery: Omit<MonitorMetricQuery, 'instances'> = {
  appID: 'demo-app',
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
    envNames: ['prod'],
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
  requestRanking.mockReset();
});

describe('monitor instance ranking', () => {
  const { appID, startTime, endTime, interval } = baseQuery;
  const rankingQuery = { appID, startTime, endTime, interval, envNames: ['prod'] };

  it('requests ranking once with second timestamps and preserves the single environment API order', async () => {
    requestRanking.mockResolvedValue([
      { instance: 'pod-a', value: 1 },
      { instance: 'pod-b', value: 2 },
    ]);
    expect(await fetchRankedMonitorInstances(rankingQuery)).toEqual([selected('pod-a'), selected('pod-b')]);
    expect(requestRanking).toHaveBeenCalledExactlyOnceWith({
      appID,
      envName: 'prod',
      startTime,
      endTime,
      interval,
      limit: 10,
      rankType: 'top',
      method: 'sum',
    });
    expect(requestMetric).not.toHaveBeenCalled();
  });

  it.each(['top', 'bottom'] as const)('ranks merged environment results by CPU value for %s', async rankType => {
    requestRanking.mockImplementation(async ({ envName }) =>
      envName === 'prod'
        ? [
            { instance: 'same-pod', value: 5 },
            { instance: 'prod-b', value: 1 },
          ]
        : [
            { instance: 'test-a', value: 3 },
            { instance: 'same-pod', value: 8 },
          ],
    );
    const instances = await fetchRankedMonitorInstances({
      ...rankingQuery,
      envNames: ['prod', 'test', 'prod'],
      limit: 2,
      rankType,
      method: 'max',
    });
    expect(instances).toEqual(
      rankType === 'top'
        ? [selected('same-pod', 'test'), selected('same-pod')]
        : [selected('prod-b'), selected('test-a', 'test')],
    );
    expect(requestRanking.mock.calls.map(([params]) => params)).toEqual(
      ['prod', 'test'].map(envName => ({
        appID,
        envName,
        startTime,
        endTime,
        interval,
        limit: 2,
        rankType,
        method: 'max',
      })),
    );
  });

  it('keeps successful environments when a ranking request fails and returns no selection for empty rankings', async () => {
    requestRanking
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce([{ instance: 'pod-a', value: 2 }]);
    expect(await fetchRankedMonitorInstances({ ...rankingQuery, envNames: ['prod', 'test'] })).toEqual([
      selected('pod-a', 'test'),
    ]);
    requestRanking.mockResolvedValueOnce([]);
    expect(await fetchRankedMonitorInstances(rankingQuery)).toEqual([]);
    expect(requestMetric).not.toHaveBeenCalled();
  });
});

describe('monitor chart time series', () => {
  it.each([1, 30, 50])('shows all %i explicitly selected instances without ranking fields', async count => {
    const instances = Array.from({ length: count }, (_, index) => `pod-${index}`);
    requestMetric.mockResolvedValue(response(instances.map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: instances.map(instance => selected(instance)),
      method: 'max',
    });
    expect(result?.series).toHaveLength(count);
    expect(requestMetric).toHaveBeenCalledExactlyOnceWith({ ...baseQuery, envName: 'prod', instances, method: 'max' });
    expect(requestRanking).not.toHaveBeenCalled();
  });

  it('groups explicit selections by environment without truncating the combined result', async () => {
    requestMetric.mockImplementation(async ({ instances }) => response((instances ?? []).map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: [selected('prod-a'), selected('test-a', 'test'), selected('test-b', 'test')],
    });
    expect(result?.series).toHaveLength(3);
    expect(requestMetric.mock.calls.map(([params]) => [params.envName, params.instances])).toEqual([
      ['prod', ['prod-a']],
      ['test', ['test-a', 'test-b']],
    ]);
  });

  it('skips empty explicit requests', async () => {
    expect(await fetchMonitorMetric({ ...baseQuery, instances: [] })).toBeUndefined();
    expect(await fetchMonitorMetric({ ...baseQuery, instances: [selected(' ')] })).toBeUndefined();
    expect(requestMetric).not.toHaveBeenCalled();
  });

  it('queries same-name instances in their own environments and deduplicates only within each environment', async () => {
    requestMetric.mockImplementation(async ({ instances }) => response((instances ?? []).map(name => series(name))));
    const result = await fetchMonitorMetric({
      ...baseQuery,
      instances: [selected('same-pod'), selected('same-pod', 'test'), selected('same-pod')],
    });
    expect(result?.series?.map(item => [item.envName, item.instance])).toEqual([
      ['prod', 'same-pod'],
      ['test', 'same-pod'],
    ]);
    expect(requestMetric.mock.calls.map(([params]) => [params.envName, params.instances])).toEqual([
      ['prod', ['same-pod']],
      ['test', ['same-pod']],
    ]);
  });
});

describe('monitor chart comparison and cache', () => {
  it('reuses the resolved instances for comparison even if a current curve is missing', async () => {
    requestMetric.mockResolvedValueOnce(response([series('current-top')])).mockResolvedValueOnce(
      response([
        {
          ...series('previous-only'),
          dataPoints: [[2, (baseQuery.startTime - 3600) * 1000]],
        },
      ]),
    );
    const instances = [selected('current-top'), selected('previous-only')];
    const result = await chartApi()(
      chartParams(101, {
        instances,
        compareOffsets: [3600],
        start_time: baseQuery.startTime + 60,
        end_time: baseQuery.endTime + 60,
      }),
    );
    const query = { ...baseQuery, envName: 'prod', instances: ['current-top', 'previous-only'], method: 'sum' };
    expect(requestMetric.mock.calls[0][0]).toEqual(query);
    expect(requestMetric.mock.calls[1][0]).toEqual({
      ...query,
      startTime: baseQuery.startTime - 3600,
      endTime: baseQuery.endTime - 3600,
    });
    expect(result.series[1].datapoints).toEqual([[2, baseQuery.startTime * 1000]]);
    expect(getCachedMetric('cpu_usage', 101, true)?.series).toHaveLength(1);
    expect(getCachedMetric('cpu_usage', 101)?.series).toHaveLength(2);
    expect(requestRanking).not.toHaveBeenCalled();
  });

  it('skips all requests for an empty instance list and skips comparison across environments', async () => {
    const api = chartApi();
    requestMetric.mockResolvedValue(response([]));
    await api(chartParams(102, { instances: [], compareOffsets: [3600] }));
    expect(requestMetric).not.toHaveBeenCalled();
    requestMetric.mockResolvedValue(response([series('pod')]));
    await api(
      chartParams(103, {
        instances: [selected('pod'), selected('pod', 'test')],
        envNames: ['prod', 'test'],
        compareOffsets: [3600],
      }),
    );
    expect(requestMetric).toHaveBeenCalledTimes(2);
  });

  it('deduplicates simultaneous identical requests but isolates method, interval and instance changes', async () => {
    const api = chartApi();
    const resolvePending: Array<(value: Record<string, MetricTimeSeries>) => void> = [];
    requestMetric.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePending.push(resolve);
        }),
    );
    const instances = [selected('pod')];
    const first = api(chartParams(104, { instances }));
    const duplicate = api(chartParams(104, { instances }));
    expect(requestMetric).toHaveBeenCalledTimes(1);
    const newMethod = api(chartParams(104, { instances, method: 'avg' }));
    const newInterval = api(chartParams(104, { instances, interval: 120 }));
    const newInstances = api(chartParams(104, { instances: [selected('pod-b')] }));
    expect(requestMetric).toHaveBeenCalledTimes(4);
    resolvePending.forEach(resolve => resolve(response([series('pod')])));
    await Promise.all([first, duplicate, newMethod, newInterval, newInstances]);
  });

  it('ignores a late old response before it can request comparison or replace the latest cache', async () => {
    const api = chartApi();
    let resolveOld!: (value: Record<string, MetricTimeSeries>) => void;
    requestMetric
      .mockImplementationOnce(
        () =>
          new Promise(resolve => {
            resolveOld = resolve;
          }),
      )
      .mockResolvedValueOnce(response([series('new-pod')]));
    const old = api(chartParams(105, { instances: [selected('old-pod')], compareOffsets: [3600] }));
    await api(chartParams(106, { instances: [selected('new-pod')] }));
    resolveOld(response([series('old-pod')]));
    expect((await old).series).toEqual([]);
    expect(getCachedMetric('cpu_usage', 105)).toBeUndefined();
    expect(getCachedMetric('cpu_usage', 106, true)?.series?.map(item => item.instance)).toEqual(['new-pod']);
    expect(requestMetric).toHaveBeenCalledTimes(2);
  });

  it('suppresses old comparison requests while the sidebar waits for the next ranking', async () => {
    let resolveCurrent!: (value: Record<string, MetricTimeSeries>) => void;
    requestMetric.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveCurrent = resolve;
        }),
    );
    const old = chartApi()(chartParams(109, { instances: [selected('pod')], compareOffsets: [3600] }));
    invalidateMonitorChartQueries();
    resolveCurrent(response([series('pod')]));
    expect((await old).series).toEqual([]);
    expect(requestMetric).toHaveBeenCalledTimes(1);
    expect(getCachedMetric('cpu_usage', 109)).toBeUndefined();
  });

  it('keeps multi-environment curve labels and statistic lookups distinct for same-name instances', async () => {
    requestMetric.mockImplementation(async ({ envName }) => response([series('same-pod', envName === 'prod' ? 5 : 2)]));
    const result = await chartApi()(
      chartParams(107, {
        instances: [selected('same-pod'), selected('same-pod', 'test')],
        envNames: ['prod', 'test'],
      }),
    );
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
    const result = await chartApi()(
      chartParams(108, {
        instances: [selected('same-pod')],
        envNames: ['prod', 'test'],
      }),
    );
    expect(result.series.map(item => item.target)).toEqual(['prod/same-pod']);
    expect(getMonitorInstanceLabel({ envName: 'prod', instance: '__CMP__昨天-same-pod' })).toBe('昨天-same-pod');
    expect(getMonitorInstanceLabel({ envName: 'prod', instance: '__CMP__昨天-same-pod' }, true)).toBe(
      'prod/昨天-same-pod',
    );
  });
});
