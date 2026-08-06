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

import type { App } from 'vue';

import { BkintegrationsBkmonitorService } from '~/api/modules/v1';

import type { MetricTimeSeries, TimeSeriesItem } from '~/@types/v1/bkintegrations-bkmonitor';

export interface MonitorChartApiResult {
  metrics: [];
  query_config?: undefined;
  series: SeriesItemLike[];
}

/** useEcharts 注入的入参：{ start_time, end_time, ...target.data, ...params } */
interface MonitorChartApiParams {
  [key: string]: unknown;
  appID?: string;
  end_time?: number;
  envName?: string;
  /** 实例 ID → 环境名称 映射（多环境模式），桥接层据此按环境分组并行请求 */
  instanceEnvMap?: Record<string, string>;
  instances?: string[];
  interval?: number;
  metricKey?: string;
  start_time?: number;
}

/** 包内 SeriesItem 的最小结构（datapoints 为 [值, 时间戳]） */
interface SeriesItemLike {
  alias: string;
  datapoints: [number, number][];
  stack?: string;
  target: string;
  type: string;
  unit?: string;
}

/** 当前时间范围（秒级 Unix），由 monitor-sideslider 注入 */
let currentStartTime = 0;
let currentEndTime = 0;

/** 各 metricKey 最新时序缓存，供 chart-card-item 在 duration-change 时回传给父组件表格 */
const metricCache = new Map<string, MetricTimeSeries>();

/** 进行中的请求缓存，避免 ExploreChart 与 useEcharts 同参并发时重复拉取监控数据 */
const inFlightRequests = new Map<string, Promise<MonitorChartApiResult>>();

// ─── 导出函数（按字母序） ───────────────────────────────────────

export function getCachedMetric(metricKey: string): MetricTimeSeries | undefined {
  return metricCache.get(metricKey);
}

/** 判断指标单位是否为百分比（如 "percent" / "%"），此类数据需 ×100 还原为百分值 */
export function isPercentUnit(unit?: string): boolean {
  if (!unit) return false;
  const u = unit.toLowerCase();
  return u === 'percent' || u === '%' || u.includes('percent') || u.includes('%');
}

/** 在应用入口注册全局 $api，供 ExploreChart 内部 useEcharts 调用 */
export function registerMonitorChartApi(app: App): void {
  (app.config.globalProperties as Record<string, unknown>).$api = {
    bkmonitor: {
      getInstanceTimeSeries,
    },
  };
}

export function setChartTimeRange(start: number, end: number): void {
  currentStartTime = start;
  currentEndTime = end;
}

// ─── 内部函数（按字母序） ───────────────────────────────────────

/** 全局 $api.bkmonitor.getInstanceTimeSeries：被 ExploreChart 内部 useEcharts 调用 */
async function getInstanceTimeSeries(resultParams: MonitorChartApiParams): Promise<MonitorChartApiResult> {
  const { appID, envName, instanceEnvMap, instances, metricKey, interval, start_time, end_time } = resultParams;
  if (!appID || !metricKey || !instances?.length) {
    return { series: [], metrics: [] };
  }
  // 优先使用包内 useEcharts 从 timeRange 解析出的时间戳（确保与图表 X 轴一致），
  // 回退到 setChartTimeRange 注入的模块级变量
  const startTime =
    (typeof start_time === 'number' ? start_time : currentStartTime) ||
    Math.floor(Date.now() / 1000) - 2 * 24 * 60 * 60;
  const endTime = (typeof end_time === 'number' ? end_time : currentEndTime) || Math.floor(Date.now() / 1000);

  // 判断是否为多环境模式：instanceEnvMap 按实例映射到多个不同环境
  const envToInstances = new Map<string, string[]>();
  if (instanceEnvMap) {
    for (const inst of instances) {
      const env = instanceEnvMap[inst];
      if (env) {
        const existing = envToInstances.get(env);
        if (existing) {
          existing.push(inst);
        } else {
          envToInstances.set(env, [inst]);
        }
      }
    }
  }
  const isMultiEnv = envToInstances.size > 1;

  // 同参并发去重：ExploreChart 与同组件内并行调用的 useEcharts 共享同一次请求
  const dedupKey =
    isMultiEnv && instanceEnvMap
      ? JSON.stringify({
          appID,
          __multi: [...instances].sort().map(i => `${i}@${instanceEnvMap[i] ?? envName}`),
          metricKey,
          interval,
          startTime,
          endTime,
        })
      : JSON.stringify({ appID, envName, instances, metricKey, interval, startTime, endTime });
  const pending = inFlightRequests.get(dedupKey);
  if (pending) return pending;

  const request = (async (): Promise<MonitorChartApiResult> => {
    // 多环境模式：并行请求各环境，合并 series
    if (isMultiEnv) {
      const reqs = [...envToInstances.entries()].map(([env, envInsts]) =>
        BkintegrationsBkmonitorService.getInstanceTimeSeries({
          appID,
          envName: env,
          instances: envInsts,
          metricKey,
          startTime,
          endTime,
          interval,
        }).catch(() => undefined),
      );
      const results = await Promise.all(reqs);

      const mergedSeries: TimeSeriesItem[] = [];
      let mergedUnit: string | undefined;

      for (const data of results) {
        if (!data) continue;
        const metric = typeof data === 'object' ? (data[metricKey] ?? Object.values(data)[0]) : undefined;
        if (metric) {
          if (metric.series) mergedSeries.push(...metric.series);
          if (!mergedUnit) mergedUnit = metric.unit;
        }
      }

      const mergedMetric: MetricTimeSeries = { series: mergedSeries, unit: mergedUnit };
      metricCache.set(metricKey, mergedMetric);
      return { series: toSeriesItems(mergedMetric, metricKey), metrics: [] };
    }

    // 单环境模式：保持原有逻辑
    if (!envName) {
      return { series: [], metrics: [] };
    }
    const data = await BkintegrationsBkmonitorService.getInstanceTimeSeries({
      appID,
      envName,
      instances,
      metricKey,
      startTime,
      endTime,
      interval,
    }).catch(() => undefined);
    const metric = data && typeof data === 'object' ? (data[metricKey] ?? Object.values(data)[0]) : undefined;
    if (metric) metricCache.set(metricKey, metric);
    return { series: toSeriesItems(metric, metricKey), metrics: [] };
  })();
  inFlightRequests.set(dedupKey, request);
  request.finally(() => inFlightRequests.delete(dedupKey));
  return request;
}

/**
 * 根据原始 unit 与 metricKey 推导包所需的展示单位。
 * 注意：包内 getValueFormat 的 ID 为王驼/id 而非显示名，例如字节速率 ID 是 "Bps"（"bytes/sec" 只是显示名）。
 * - 网络类指标统一设为 "Bps"，包内置 decimalSIPrefix("B/s") 自动缩放为 KB/s / MB/s / GB/s
 * - 百分比保留 "percent"（toPercent 仅追加 "%"，缩放已在桥接层完成）
 * - 其他指标保留原始 unit
 */
function resolveDisplayUnit(metricKey: string, rawUnit: string): string {
  const isNetwork = /^network_(receive|transmit|in|out|read|write)/.test(metricKey);
  if (isNetwork) return 'Bps';
  return rawUnit;
}

/** 将自建接口的 MetricTimeSeries 转换为包所需的 SeriesItem[]（datapoints 已是 [值, 时间戳] 格式） */
function toSeriesItems(metric: MetricTimeSeries | undefined, metricKey: string): SeriesItemLike[] {
  if (!metric?.series?.length) return [];
  // 百分比单位（如 CPU 利用率）原始值为小数（0.36 表示 36%），需 ×100 还原
  const displayUnit = resolveDisplayUnit(metricKey, metric.unit ?? '');
  const percentScale = isPercentUnit(displayUnit) ? 100 : 1;
  return metric.series.map((s: TimeSeriesItem) => ({
    target: s.instance ?? '',
    alias: s.instance ?? '',
    unit: displayUnit,
    type: 'line',
    // stack: 'Total', // 不堆叠
    // API 返回的 dataPoints 实际为 [value, timestamp_ms]，与包内 createSeries 的 point[0]=值/point[1]=时间戳 契约一致，无需翻转
    datapoints: (s.dataPoints ?? []).map(p => [((p?.[0] ?? 0) * percentScale) as number, p?.[1]] as [number, number]),
  }));
}
