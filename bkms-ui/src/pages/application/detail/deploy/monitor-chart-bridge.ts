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

export interface MonitorMetricQuery extends MonitorQueryTime {
  instances: MonitorInstance[];
  metricKey: string;
}

export interface MonitorMetricTimeSeries extends MetricTimeSeries {
  series?: MonitorTimeSeriesItem[];
}

export type MonitorRankType = 'bottom' | 'top';

/** 请求按环境拆分，结果保留环境归属，用于图例和汇总表匹配。 */
export interface MonitorTimeSeriesItem extends TimeSeriesItem {
  _color?: string;
  envName?: string;
}

/** 以 target.data 中的时间快照为准，与本轮排名请求保持一致。 */
interface MonitorChartApiParams {
  [key: string]: unknown;
  appID?: string;
  /** 时间对比偏移量列表（秒），空数组 / 不传 = 关闭 */
  compareOffsets?: number[];
  endTime?: number;
  envNames?: string[];
  instances?: MonitorInstance[];
  interval?: number;
  method?: MonitorAggregationMethod;
  metricKey?: string;
  queryId?: number;
  startTime?: number;
}

interface MonitorQueryTime {
  appID: string;
  endTime: number;
  interval?: number;
  method?: MonitorAggregationMethod;
  startTime: number;
}

interface MonitorRankingQuery extends MonitorQueryTime {
  envNames: string[];
  limit?: number;
  rankType?: MonitorRankType;
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

/** 按环境查指标时序，只传已解析的实例名单。 */
export async function fetchMonitorMetric(query: MonitorMetricQuery): Promise<MonitorMetricTimeSeries | undefined> {
  const { appID, metricKey, startTime, endTime, interval } = query;
  const envToInstances = new Map<string, string[]>();
  // 同名 Pod 可以属于不同环境，按环境分组后再传原始实例名。
  for (const { envName, instance } of query.instances) {
    const name = instance.trim();
    if (!envName || !name) continue;
    const names = envToInstances.get(envName) ?? [];
    if (!names.includes(name)) names.push(name);
    envToInstances.set(envName, names);
  }
  if (!appID || !metricKey || !envToInstances.size) return undefined;

  const results = await Promise.all(
    [...envToInstances].map(async ([envName, instances]) => {
      const data = await BkintegrationsBkmonitorService.getInstanceTimeSeries({
        appID,
        envName,
        instances,
        metricKey,
        startTime,
        endTime,
        interval,
        method: query.method ?? 'sum',
      }).catch(() => undefined);
      const metric = data?.[metricKey];
      return metric ? { ...metric, series: (metric.series ?? []).map(series => ({ ...series, envName })) } : undefined;
    }),
  );
  const first = results.find(result => result !== undefined);
  if (!first) return undefined;
  return { ...first, series: results.flatMap(result => result?.series ?? []) };
}

/** 每个环境只查一次 CPU 排名，所有图表共用合并后的实例名单。 */
export async function fetchRankedMonitorInstances(query: MonitorRankingQuery): Promise<MonitorInstance[]> {
  const { appID, startTime, endTime, interval, limit = 10, rankType = 'top', method = 'sum' } = query;
  if (!appID) return [];
  const envNames = [...new Set(query.envNames.filter(Boolean))];
  const results = await Promise.all(
    envNames.map(async envName => {
      const ranked = await BkintegrationsBkmonitorService.rankInstances({
        appID,
        envName,
        startTime,
        endTime,
        interval,
        limit,
        rankType,
        method,
      }).catch(() => undefined);
      return (ranked ?? []).flatMap(item => {
        const instance = item.instance?.trim();
        return instance ? [{ envName, instance, value: item.value ?? 0 }] : [];
      });
    }),
  );

  // 单环境以接口顺序为准，多环境按 CPU 平均用量重排后截取全局 N。
  const ranked = results.flat();
  if (envNames.length > 1) {
    const direction = rankType === 'bottom' ? 1 : -1;
    ranked.sort((a, b) => direction * (a.value - b.value));
  }
  return ranked.slice(0, limit).map(({ envName, instance }) => ({ envName, instance }));
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

/** 等待新排名或关闭侧栏时，旧的时序响应不应再触发对比请求。 */
export function invalidateMonitorChartQueries(): void {
  activeQueries.clear();
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

// ─── 内部函数（按字母序） ───────────────────────────────────────

/** 全局 $api.bkmonitor.getInstanceTimeSeries：被 ExploreChart 内部 useEcharts 调用。 */
async function getInstanceTimeSeries(params: MonitorChartApiParams): Promise<MonitorChartApiResult> {
  const { appID, metricKey, queryId, instances, startTime, endTime, interval, method = 'sum' } = params;
  if (!appID || !metricKey || !instances?.length || startTime === undefined || endTime === undefined) {
    return { series: [], metrics: [] };
  }
  const query: MonitorMetricQuery = {
    appID,
    metricKey,
    startTime,
    endTime,
    instances,
    interval,
    method,
  };
  const showEnvName = (params.envNames?.length ?? 0) > 1;
  const compareOffsets = showEnvName
    ? []
    : Array.from(new Set((params.compareOffsets ?? []).filter(offset => typeof offset === 'number' && offset > 0)));
  const dedupKey = JSON.stringify({ ...query, queryId, compareOffsets, showEnvName });
  activeQueries.set(metricKey, dedupKey);
  const pending = inFlightRequests.get(dedupKey);
  if (pending) return pending;
  metricCache.delete(metricKey);

  const request = (async (): Promise<MonitorChartApiResult> => {
    const current = (await fetchMonitorMetric(query)) ?? { series: [] };
    if (activeQueries.get(metricKey) !== dedupKey) return { series: [], metrics: [] };
    const chart: MonitorMetricTimeSeries = { ...current, series: [...(current.series ?? [])] };
    if (compareOffsets.length) {
      // 对比时段复用当前查询的名单，避免某个指标缺少曲线时缩小对比范围。
      const compareResults = await Promise.all(
        compareOffsets.map(async offset => {
          const compare = await fetchMonitorMetric({
            ...query,
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
    return { series: toSeriesItems(chart, metricKey, showEnvName), metrics: [] };
  })().finally(() => inFlightRequests.delete(dedupKey));
  inFlightRequests.set(dedupKey, request);
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
