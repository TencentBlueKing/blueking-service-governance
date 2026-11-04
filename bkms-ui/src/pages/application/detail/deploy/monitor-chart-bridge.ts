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
import { i18n } from '~/modules/i18n';

import type { MetricTimeSeries, TimeSeriesItem } from '~/@types/v1/bkintegrations-bkmonitor';

export type MonitorAggregationMethod = 'avg' | 'count' | 'max' | 'min' | 'sum';

export interface MonitorChartApiResult {
  metrics: [];
  query_config?: undefined;
  series: SeriesItemLike[];
}
/** 实例名称只在所属环境内唯一。 */
export interface MonitorInstance {
  envName: string;
  instance: string;
}

export type MonitorLimitType = 'bottom' | 'top';

export interface MonitorMetricQuery {
  all?: boolean;
  appID: string;
  endTime: number;
  envName?: string;
  envNames?: string[];
  instances?: MonitorInstance[];
  interval?: number;
  limit?: number;
  limitType?: MonitorLimitType;
  method?: MonitorAggregationMethod;
  metricKey: string;
  startTime: number;
}

export interface MonitorMetricTimeSeries extends MetricTimeSeries {
  series?: MonitorTimeSeriesItem[];
}

/** 保留环境归属用于合并、补查和匹配；请求按环境拆分，实例名称单独进入请求体。 */
export interface MonitorTimeSeriesItem extends TimeSeriesItem {
  _color?: string;
  envName?: string;
}

/** useEcharts 注入的入参：{ start_time, end_time, ...target.data, ...params } */
interface MonitorChartApiParams {
  [key: string]: unknown;
  all?: boolean;
  appID?: string;
  /** 时间对比偏移量列表（秒），空数组 / 不传 = 关闭 */
  compareOffsets?: number[];
  end_time?: number;
  envName?: string;
  envNames?: string[];
  instances?: MonitorInstance[];
  interval?: number;
  limit?: number;
  limitType?: MonitorLimitType;
  method?: MonitorAggregationMethod;
  metricKey?: string;
  queryId?: number;
  start_time?: number;
}

/** 包内 SeriesItem 的最小结构（datapoints 为 [值, 时间戳]） */
interface SeriesItemLike {
  alias: string;
  color?: string;
  datapoints: [number, number][];
  name?: string;
  stack?: string;
  target: string;
  type: string;
  unit?: string;
}

/** 当前时间范围（秒级 Unix），由 monitor-sideslider 注入 */
let currentStartTime = 0;
let currentEndTime = 0;

/** 每个指标仅保留最新查询；当前期与图表对比数据分开，避免汇总表混入对比实例。 */
const metricCache = new Map<
  string,
  {
    chart: MonitorMetricTimeSeries;
    current: MonitorMetricTimeSeries;
    queryId?: number;
  }
>();
const activeQueries = new Map<string, string>();

/** 进行中的请求缓存，避免 ExploreChart 与 useEcharts 同参并发时重复拉取监控数据 */
const inFlightRequests = new Map<string, Promise<MonitorChartApiResult>>();

const COMPARE_INSTANCE_PREFIX = '__CMP__';

const COMPARE_PALETTE: string[] = [
  '#EA3636',
  '#FF9C01',
  '#3A84FF',
  '#2DCB56',
  '#A855F7',
  '#EC4899',
  '#14B8A6',
  '#6366F1',
];

// ─── 导出函数（按字母序） ───────────────────────────────────────

/** 按环境查询一段指标，不写图表缓存；汇总表补查也使用此入口。 */
export async function fetchMonitorMetric(query: MonitorMetricQuery): Promise<MonitorMetricTimeSeries | undefined> {
  const { appID, metricKey, startTime, endTime, interval } = query;
  const envToInstances = new Map<string, string[]>();
  if (query.all) {
    for (const env of query.envNames?.length ? query.envNames : [query.envName]) {
      if (env) envToInstances.set(env, []);
    }
  } else {
    // 同名 Pod 可以属于不同环境，必须先按环境分组，再向 SDK 传原始实例名。
    for (const { envName, instance } of query.instances ?? []) {
      if (!envName || !instance) continue;
      const names = envToInstances.get(envName) ?? [];
      if (!names.includes(instance)) names.push(instance);
      envToInstances.set(envName, names);
    }
  }
  if (!appID || !metricKey || !envToInstances.size) return undefined;

  const results = await Promise.all(
    [...envToInstances].map(async ([envName, instances]) => {
      // 显式实例查询不合并排名默认值，避免手选与补查结果被 top10 截断。
      const scope = query.all
        ? { all: true, limit: query.limit ?? 10, limitType: query.limitType ?? 'top' }
        : { instances };
      const data = await BkintegrationsBkmonitorService.getInstanceTimeSeries({
        appID,
        envName,
        metricKey,
        startTime,
        endTime,
        interval,
        method: query.method ?? 'sum',
        ...scope,
      }).catch(() => undefined);
      const metric = data?.[metricKey];
      return metric ? { ...metric, series: (metric.series ?? []).map(series => ({ ...series, envName })) } : undefined;
    }),
  );
  const first = results.find(result => result !== undefined);
  if (!first) return undefined;
  let series = results.flatMap(result => result?.series ?? []);
  if (query.all && envToInstances.size > 1) {
    const direction = query.limitType === 'bottom' ? 1 : -1;
    series = series
      .sort((a, b) => direction * ((a.stat?.avg?.[1] ?? 0) - (b.stat?.avg?.[1] ?? 0)))
      .slice(0, query.limit ?? 10);
  }
  return { ...first, series };
}

/** 时间对比偏移量格式化为可读标签 */
export function formatOffsetLabel(seconds: number): string {
  const t = i18n.global.t.bind(i18n.global);
  if (seconds < 60) return t('{0} 秒前', [seconds]);
  if (seconds < 3600) return t('{0} 分钟前', [Math.floor(seconds / 60)]);
  if (seconds < 86400) return t('{0} 小时前', [Math.floor(seconds / 3600)]);
  if (seconds < 7 * 86400) {
    const days = Math.floor(seconds / 86400);
    return days === 1 ? t('昨天') : t('{0} 天前', [days]);
  }
  if (seconds < 30 * 86400) {
    const weeks = Math.floor(seconds / (7 * 86400));
    return weeks === 1 ? t('上周') : t('{0} 周前', [weeks]);
  }
  if (seconds < 365 * 86400) {
    const months = Math.floor(seconds / (30 * 86400));
    return months === 1 ? t('一月前') : t('{0} 月前', [months]);
  }
  const years = Math.floor(seconds / (365 * 86400));
  return years === 1 ? t('一年前') : t('{0} 年前', [years]);
}

export function getCachedMetric(
  metricKey: string,
  queryId?: number,
  currentOnly = false,
): MonitorMetricTimeSeries | undefined {
  const cached = metricCache.get(metricKey);
  if (!cached || cached.queryId !== queryId) return undefined;
  return currentOnly ? cached.current : cached.chart;
}

/** 按偏移量值返回色板颜色 */
export function getColorByOffset(offset: number): string {
  const known: Record<number, number> = {
    60: 0, // 1 分钟  → 蓝
    300: 1, // 5 分钟  → 红
    3600: 2, // 1 小时  → 绿
    86400: 3, // 1 天    → 黄
    604800: 4, // 7 天    → 浅蓝
    2592000: 5, // 30 天   → 橙
    31536000: 6, // 1 年    → 紫
  };
  if (offset in known) return COMPARE_PALETTE[known[offset]];
  const str = String(offset);
  let hash = 5381;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) + hash + str.charCodeAt(i)) | 0;
  }
  return COMPARE_PALETTE[Math.abs(hash) % COMPARE_PALETTE.length];
}

/** 选择器与汇总表共用组合键；数组编码避免仅按名称匹配或拼接分隔符产生冲突。 */
export function getMonitorInstanceKey(instance: { envName?: string; instance?: string }): string {
  return JSON.stringify([instance.envName || '', instance.instance || '']);
}

/** 单环境保留实例名，多环境同时显示环境，确保图例名称及联动标识唯一。 */
export function getMonitorInstanceLabel(
  instance: { envName?: string; instance?: string },
  showEnvName = false,
): string {
  const name = instance.instance ?? '';
  const label = name.startsWith(COMPARE_INSTANCE_PREFIX) ? name.slice(COMPARE_INSTANCE_PREFIX.length) : name;
  return showEnvName && instance.envName ? `${instance.envName}/${label}` : label;
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

/** 全局 $api.bkmonitor.getInstanceTimeSeries：被 ExploreChart 内部 useEcharts 调用。 */
async function getInstanceTimeSeries(params: MonitorChartApiParams): Promise<MonitorChartApiResult> {
  const { appID, metricKey, queryId } = params;
  if (!appID || !metricKey || (!params.all && !params.instances?.length)) {
    return { series: [], metrics: [] };
  }
  const startTime =
    (typeof params.start_time === 'number' ? params.start_time : currentStartTime) ||
    Math.floor(Date.now() / 1000) - 2 * 24 * 60 * 60;
  const endTime =
    (typeof params.end_time === 'number' ? params.end_time : currentEndTime) || Math.floor(Date.now() / 1000);
  const query: MonitorMetricQuery = {
    appID,
    metricKey,
    startTime,
    endTime,
    envName: params.envName,
    envNames: params.envNames,
    instances: params.instances,
    interval: params.interval,
    all: params.all,
    limit: params.limit,
    limitType: params.limitType,
    method: params.method,
  };
  const envNames = params.envNames?.length ? params.envNames : [params.envName].filter(Boolean);
  const compareOffsets =
    envNames.length > 1
      ? []
      : Array.from(new Set((params.compareOffsets ?? []).filter(offset => typeof offset === 'number' && offset > 0)));
  const dedupKey = JSON.stringify({ ...query, queryId, compareOffsets });
  activeQueries.set(metricKey, dedupKey);
  const pending = inFlightRequests.get(dedupKey);
  if (pending) return pending;
  metricCache.delete(metricKey);

  const request = (async (): Promise<MonitorChartApiResult> => {
    const current = (await fetchMonitorMetric(query)) ?? { series: [] };
    const chart: MonitorMetricTimeSeries = { ...current, series: [...(current.series ?? [])] };
    const instances = (current.series ?? []).flatMap(series =>
      series.instance ? [{ envName: series.envName || params.envName || '', instance: series.instance }] : [],
    );
    if (instances.length && compareOffsets.length) {
      const compareResults = await Promise.all(
        compareOffsets.map(async offset => {
          const compare = await fetchMonitorMetric({
            appID,
            metricKey,
            instances,
            envName: current.series?.[0]?.envName || params.envName,
            method: params.method,
            interval: params.interval,
            startTime: startTime - offset,
            endTime: endTime - offset,
          });
          return transformCompareSegment(compare, offset, formatOffsetLabel(offset));
        }),
      );
      chart.series!.push(...compareResults.flatMap(result => result.series ?? []));
    }
    // 新筛选或刷新已经开始时，旧响应既不更新缓存，也不交给图表绘制。
    if (activeQueries.get(metricKey) !== dedupKey) return { series: [], metrics: [] };
    metricCache.set(metricKey, { current, chart, queryId });
    return { series: toSeriesItems(chart, metricKey, envNames.length > 1), metrics: [] };
  })();
  inFlightRequests.set(dedupKey, request);
  void request.finally(() => inFlightRequests.delete(dedupKey));
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
function toSeriesItems(
  metric: MonitorMetricTimeSeries | undefined,
  metricKey: string,
  showEnvName: boolean,
): SeriesItemLike[] {
  if (!metric?.series?.length) return [];
  const displayUnit = resolveDisplayUnit(metricKey, metric.unit ?? '');
  const percentScale = isPercentUnit(displayUnit) ? 100 : 1;
  return metric.series.map(s => {
    const instance = s.instance ?? '';
    const hasCmpPrefix = instance.startsWith(COMPARE_INSTANCE_PREFIX);
    const displayName = getMonitorInstanceLabel(s, showEnvName);
    // 全部展示字段统一使用 displayName（去前缀），避免 __CMP__ 透传到 UI；
    // 对比项的识别改为通过 isCompare 显式字段，下游不再依赖字符串前缀判断。
    const obj: SeriesItemLike = {
      target: displayName,
      alias: displayName,
      name: displayName,
      unit: displayUnit,
      type: 'line',
      datapoints: (s.dataPoints ?? []).map(p => [((p?.[0] ?? 0) * percentScale) as number, p?.[1]] as [number, number]),
    };
    if (hasCmpPrefix) (obj as unknown as Record<string, unknown>).isCompare = true;
    if (s._color) (obj as unknown as Record<string, unknown>).color = s._color;
    return obj;
  });
}

/** 将对比时段移回当前时间轴，并标记对比曲线与颜色。 */
function transformCompareSegment(
  compare: MonitorMetricTimeSeries | undefined,
  offsetSeconds: number,
  timeLabel: string,
): MonitorMetricTimeSeries {
  return {
    ...compare,
    series: (compare?.series ?? []).map(series => ({
      ...series,
      instance: `${COMPARE_INSTANCE_PREFIX}${timeLabel}-${series.instance ?? ''}`,
      dataPoints: (series.dataPoints ?? []).map(point => [point?.[0], (point?.[1] ?? 0) + offsetSeconds * 1000]),
      _color: getColorByOffset(offsetSeconds),
    })),
  };
}
