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

import { i18n } from '~/modules/i18n';
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
  /** 时间对比偏移量列表（秒），空数组 / 不传 = 关闭 */
  compareOffsets?: number[];
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

/** TimeSeriesItem 内部扩展：携带颜色元信息供图例同步使用 */
interface TimeSeriesItemWithColor extends TimeSeriesItem {
  _color?: string;
}

/** 当前时间范围（秒级 Unix），由 monitor-sideslider 注入 */
let currentStartTime = 0;
let currentEndTime = 0;

/** 各 metricKey 最新时序缓存，供 chart-card-item 在 duration-change 时回传给父组件表格 */
const metricCache = new Map<string, MetricTimeSeries>();

/** 进行中的请求缓存，避免 ExploreChart 与 useEcharts 同参并发时重复拉取监控数据 */
const inFlightRequests = new Map<string, Promise<MonitorChartApiResult>>();

const COMPARE_INSTANCE_PREFIX = '__CMP__';

const COMPARE_PALETTE: string[] = [
  '#EA3636', '#FF9C01', '#3A84FF', '#2DCB56',
  '#A855F7', '#EC4899', '#14B8A6', '#6366F1',
];

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

/** 按偏移量值返回色板颜色 */
export function getColorByOffset(offset: number): string {
  const known: Record<number, number> = {
    60: 0,          // 1 分钟  → 蓝
    300: 1,         // 5 分钟  → 红
    3600: 2,        // 1 小时  → 绿
    86400: 3,       // 1 天    → 黄
    604800: 4,      // 7 天    → 浅蓝
    2592000: 5,     // 30 天   → 橙
    31536000: 6,    // 1 年    → 紫
  };
  if (offset in known) return COMPARE_PALETTE[known[offset]];
  const str = String(offset);
  let hash = 5381;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) + hash + str.charCodeAt(i)) | 0;
  }
  return COMPARE_PALETTE[Math.abs(hash) % COMPARE_PALETTE.length];
}

// ─── 内部函数（按字母序） ───────────────────────────────────────

/** 拉取单段指标时序（不缓存），供基础查询和时间对比共用 */
async function fetchMetricSegment(
  appID: string, envName: string, instances: string[], metricKey: string,
  interval: number | undefined, startTime: number, endTime: number,
): Promise<MetricTimeSeries | undefined> {
  const data = await BkintegrationsBkmonitorService.getInstanceTimeSeries({
    appID, envName, instances, metricKey, startTime, endTime, interval,
  }).catch(() => undefined);
  if (!data || typeof data !== 'object') return undefined;
  return (data[metricKey] ?? Object.values(data)[0]) as MetricTimeSeries | undefined;
}

/** 将对比时段数据包装为独立时序段：偏移时间戳、添加 __CMP__ 前缀、注入颜色 */
function transformCompareSegment(
  compare: MetricTimeSeries | undefined, offsetSeconds: number, timeLabel: string,
): MetricTimeSeries {
  const result: MetricTimeSeries = { series: [] };
  if (!compare?.series?.length) return result;
  const color = getColorByOffset(offsetSeconds);
  for (const s of compare.series) {
    const newInstance = s.instance
      ? `${COMPARE_INSTANCE_PREFIX}${timeLabel}-${s.instance}`
      : s.instance;
    result.series!.push({
      ...s,
      instance: newInstance,
      dataPoints: (s.dataPoints ?? []).map(
        p => [p?.[0], (p?.[1] ?? 0) - offsetSeconds * 1000] as number[],
      ),
      _color: color,
    } as TimeSeriesItemWithColor);
  }
  return result;
}

/** 全局 $api.bkmonitor.getInstanceTimeSeries：被 ExploreChart 内部 useEcharts 调用 */
async function getInstanceTimeSeries(resultParams: MonitorChartApiParams): Promise<MonitorChartApiResult> {
  const { appID, compareOffsets, envName, instanceEnvMap, instances, metricKey, interval, start_time, end_time } = resultParams;
  // 规范化对比偏移：过滤非数字 / <=0 / 重复
  const normalizedCompareOffsets = Array.isArray(compareOffsets)
    ? Array.from(new Set(compareOffsets.filter((v): v is number => typeof v === 'number' && v > 0)))
    : [];
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
          compareOffsets: normalizedCompareOffsets,
          metricKey,
          interval,
          startTime,
          endTime,
        })
      : JSON.stringify({ appID, compareOffsets: normalizedCompareOffsets, envName, instances, metricKey, interval, startTime, endTime });
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

    // 单环境模式
    if (!envName) return { series: [], metrics: [] };
    const currentMetric = await fetchMetricSegment(appID, envName, instances, metricKey, interval, startTime, endTime);
    if (!currentMetric) return { series: [], metrics: [] };

    // 时间对比：在基础时序上叠加过去时段数据
    if (normalizedCompareOffsets.length > 0) {
      const compareResults = await Promise.all(
        normalizedCompareOffsets.map(async offset => {
          const metric = await fetchMetricSegment(
            appID, envName, instances, metricKey, interval,
            startTime - offset, endTime - offset,
          );
          return transformCompareSegment(metric, offset, formatOffsetLabel(offset));
        }),
      );
      const merged: MetricTimeSeries = { ...currentMetric, series: [...(currentMetric.series ?? [])] };
      for (const cmp of compareResults) {
        if (cmp.series?.length) merged.series!.push(...cmp.series);
      }
      metricCache.set(metricKey, merged);
      return { series: toSeriesItems(merged, metricKey), metrics: [] };
    }

    metricCache.set(metricKey, currentMetric);
    return { series: toSeriesItems(currentMetric, metricKey), metrics: [] };
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
  const displayUnit = resolveDisplayUnit(metricKey, metric.unit ?? '');
  const percentScale = isPercentUnit(displayUnit) ? 100 : 1;
  return metric.series.map((s: TimeSeriesItem) => {
    const instance = s.instance ?? '';
    const hasCmpPrefix = instance.startsWith(COMPARE_INSTANCE_PREFIX);
    const displayName = hasCmpPrefix ? instance.slice(COMPARE_INSTANCE_PREFIX.length) : instance;
    const ext = s as TimeSeriesItemWithColor;
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
    if (ext._color) (obj as unknown as Record<string, unknown>).color = ext._color;
    return obj;
  });
}
