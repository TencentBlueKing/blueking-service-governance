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

import type { BatchOutputObj } from '~/@types/v1/release-strategies';

/** 单个批次 target 的解析结果（策略预览 / 部署预览共用的唯一事实源） */
export interface ParsedBatchTarget {
  /** 是否为全量（百分比 ≥ 100）；注意空 target 不在此判定，见 isFullTarget */
  isFull: boolean;
  /** trim 后的原始文本；空 target 时为空串（表单回填需要区分「待填写」） */
  text: string;
  /** 计量单位 */
  unit: 'count' | 'percent';
  /** 解析出的数值；百分比非法（NaN）或正整数缺失 / 非 / 0 以下时为 undefined */
  value?: number;
}

/** 进度方块着色：prev=此前批次已更新，new=本批新增，pending=尚未更新 */
export type ProgressSquareState = 'new' | 'pending' | 'prev';

/** 进度方块着色行的最小结构（占比模型，供策略预览与部署预览共用） */
export interface RatioProgress {
  /** 本批累计占比（0 ~ 1） */
  accumulatedRatio: number;
  /** 本批之前累计占比（0 ~ 1） */
  prevRatio: number;
}

/** 批次之后的暂停信息（仅 duration / permanent 两种模式） */
export interface ReleasePreviewPause {
  mode: 'duration' | 'permanent';
  /** 定时暂停时长（秒），仅 duration 模式有意义 */
  seconds?: number;
}

export interface ReleasePreviewResult {
  rows: ReleasePreviewRow[];
  summary: ReleasePreviewSummary;
}

/** 发布预览时间线中的一行（一个批次） */
export interface ReleasePreviewRow {
  /** 第 N 批，从 1 开始 */
  batchIndex: number;
  /** 是否为系统补齐的全量批（末批非 100% 时后端自动补齐剩余实例） */
  isAutoFull: boolean;
  /** 本批是否为全量（target 为 100%） */
  isFull: boolean;
  /** target 是否为百分比 */
  isPercent: boolean;
  /** 本批完成后的暂停信息；pauseMode 为 none 时不展示 */
  pause?: ReleasePreviewPause;
  /** 本批之前累计已更新的实例数（用于进度方块分段着色） */
  prevCount: number;
  /** 原始 target 文本（如 "1" / "25%"），用于展示百分比 */
  targetText: string;
  /** 本批"累计更新到"的实例数（百分比按实例总数向上取整换算） */
  updatedCount: number;
}

/** 发布预览摘要栏四项指标 */
export interface ReleasePreviewSummary {
  /** 批次数 */
  batchCount: number;
  /** 暂停次数（pauseMode 非 none 的批次数） */
  pauseCount: number;
  /** 实例总数 */
  totalReplicas: number;
  /** 流量批次（当前后端批次 target 不支持权重，固定为 0） */
  trafficBatchCount: 0;
}

/** 策略说明文案的结构化数据（由组件结合 i18n 拼装为完整句子） */
export interface StrategyHintParts {
  /** 用户配置的批次数；区分「单批全量」与「两批收尾全量」（如 25% → 100%） */
  batchCount: number;
  firstIsPercent: boolean;
  /** 首批 target 原文（如 "1" / "25%"） */
  firstTarget: string;
  /** 末批是否 100%（全量） */
  lastIsFull: boolean;
  /**
   * 中间批次 target 原文列表，用于「确认后累计到 …」。
   * 末批非全量时它同样属于中间段：系统会在其后再补一个全量批，收尾统一为「最后自动全量」。
   */
  middleTargets: string[];
}

/** target 展示文案的 i18n 词条（全量 / 个） */
export interface TargetTexts {
  full: string;
  unit: string;
}

/**
 * 为单个批次生成实例进度方块着色序列。
 * 实例数超过 maxSquares 时按比例分组合并（每个方块代表一段实例，取段首实例着色）；
 * 实例数少于格数时 1:1 精确到实例（与策略预览 buildSquaresByRatio 的固定格数着色有意不同）。
 */
export function buildBatchSquares(
  row: Pick<ReleasePreviewRow, 'prevCount' | 'updatedCount'>,
  replicas: number,
  maxSquares = 20,
): ProgressSquareState[] {
  const total = Math.max(1, Math.floor(Number(replicas) || 0));
  const squareCount = Math.min(total, Math.max(1, Math.floor(maxSquares) || 1));
  return Array.from({ length: squareCount }, (_, index) => {
    const instance = Math.floor((index * total) / squareCount) + 1;
    if (instance <= row.prevCount) return 'prev';
    if (instance <= row.updatedCount) return 'new';
    return 'pending';
  });
}

/**
 * 根据策略批次列表与实例总数构建发布预览数据。
 * 复杂度 O(batches)，纯函数无副作用，便于单测与复用。
 * 末批非全量时追加「系统补齐的全量批」，与策略预览及后端实际行为对齐。
 */
export function buildReleasePreview(batches: BatchOutputObj[] | undefined, replicas: number): ReleasePreviewResult {
  const list = batches ?? [];
  const total = Math.max(1, Math.floor(Number(replicas) || 0));

  let prevCount = 0;
  const rows: ReleasePreviewRow[] = list.map((batch, index) => {
    const pauseMode = batch.pauseMode ?? 'none';
    const parsed = parseBatchTarget(batch.target);
    const targetText = parsed.text || '100%';
    const row: ReleasePreviewRow = {
      batchIndex: index + 1,
      prevCount,
      updatedCount: resolveBatchUpdatedCount(targetText, total),
      targetText,
      isPercent: parsed.unit === 'percent',
      isFull: isFullTarget(batch.target),
      isAutoFull: false,
    };
    prevCount = row.updatedCount;
    if (pauseMode === 'permanent' || pauseMode === 'duration') {
      row.pause = {
        mode: pauseMode,
        seconds: pauseMode === 'duration' ? Number(batch.pauseSeconds) || 0 : undefined,
      };
    }
    return row;
  });

  if (shouldAppendAutoFullBatch(list.map(batch => batch.target))) {
    rows.push({
      batchIndex: list.length + 1,
      prevCount,
      updatedCount: total,
      targetText: '100%',
      isPercent: true,
      isFull: true,
      isAutoFull: true,
    });
  }

  return {
    summary: {
      batchCount: list.length,
      pauseCount: list.filter(batch => (batch.pauseMode ?? 'none') !== 'none').length,
      totalReplicas: total,
      trafficBatchCount: 0,
    },
    rows,
  };
}

/**
 * 按累计占比生成固定格数的进度方块着色序列（策略预览使用）。
 * 第 i 格代表区间 [i / count, (i + 1) / count]，按格中心落点决定颜色。
 */
export function buildSquaresByRatio(row: RatioProgress, count = 20): ProgressSquareState[] {
  return Array.from({ length: Math.max(1, Math.floor(count) || 1) }, (_, index) => {
    const position = (index + 0.5) / Math.max(1, Math.floor(count) || 1);
    if (position <= row.prevRatio) return 'prev';
    if (position <= row.accumulatedRatio) return 'new';
    return 'pending';
  });
}

/** 解析策略批次为目标说明结构；空批次返回 null */
export function buildStrategyHint(batches: BatchOutputObj[] | undefined): null | StrategyHintParts {
  const targets = (batches ?? []).map(batch => normalizeTargetText(batch.target));
  if (targets.length === 0) return null;
  const lastTarget = targets[targets.length - 1] ?? '';
  const lastIsFull = isFullTarget(lastTarget);
  return {
    batchCount: targets.length,
    firstTarget: targets[0] ?? '',
    firstIsPercent: parseBatchTarget(targets[0]).unit === 'percent',
    // 末批全量时它即收尾的全量批；否则它仍属于累计段，全量批由系统补齐
    middleTargets: targets.slice(1, lastIsFull ? -1 : undefined),
    lastIsFull,
  };
}

/** 将单个批次 target 格式化为选项摘要 token：100% → 全量、正整数 → "1个"、百分比 → "25%" */
export function formatOptionTarget(target: string | undefined, texts: TargetTexts): string {
  const parsed = parseBatchTarget(target);
  // 空 target 按接口契约视为全量
  if (!parsed.text) return texts.full;
  if (parsed.unit === 'percent') {
    return parsed.value === undefined || parsed.value >= 100 ? texts.full : parsed.text;
  }
  return `${parsed.text}${texts.unit}`;
}

/** target 是否为全量（100% 及以上）；空 target 在接口契约里等同 100%，同样视为全量。 */
export function isFullTarget(target: string | undefined): boolean {
  const parsed = parseBatchTarget(target);
  return !parsed.text || parsed.isFull;
}

/** 展示用规范化文本：接口契约中空 target 等同 100% */
export function normalizeTargetText(target: string | undefined): string {
  return parseBatchTarget(target).text || '100%';
}

/**
 * 解析接口 / 表单拼装的批次 target（"1" / "25%"）。
 * 策略预览与部署预览的全量判断、实例换算、选项摘要都从该函数取数，
 * 避免 endsWith('%') / parseFloat / ≥100 等细节在两侧各自演化产生分歧。
 */
export function parseBatchTarget(target: string | undefined): ParsedBatchTarget {
  const text = (target ?? '').trim();
  if (text.endsWith('%')) {
    const pct = Number.parseFloat(text.slice(0, -1));
    const value = Number.isFinite(pct) ? pct : undefined;
    return { isFull: value !== undefined && value >= 100, text, unit: 'percent', value };
  }
  const count = Number.parseInt(text, 10);
  const value = Number.isFinite(count) && count > 0 ? count : undefined;
  return { isFull: false, text, unit: 'count', value };
}

/**
 * 将单个批次 target（正整数或百分比）换算为"累计更新到"的实例数。
 * - 百分比：按实例总数向上取整（如 20 × 25% = 5，20 × 33% = 6.6 → 7），不超过实例总数；
 * - 正整数：直接取该值，超过实例总数时收敛为实例总数；
 * - 非法值：兜底返回实例总数（视同全量）。
 */
export function resolveBatchUpdatedCount(target: string | undefined, replicas: number): number {
  const total = Math.max(1, Math.floor(Number(replicas) || 0));
  const parsed = parseBatchTarget(target);
  if (!parsed.text || parsed.value === undefined || parsed.value <= 0) return total;
  if (parsed.unit === 'percent') return Math.min(Math.ceil((total * parsed.value) / 100), total);
  return Math.min(parsed.value, total);
}

/**
 * 末批未达全量时，后端会在所有批次步骤完成后进入 UpdatingPrimary 阶段，把剩余实例补齐到全量
 * （见 bkms-server/pkg/deploy/batchrelease/status.go），即系统自动追加一个全量批。
 * 策略预览与部署预览/提示必须共用该判断，避免同一策略在两侧给出不同的收尾语义。
 */
export function shouldAppendAutoFullBatch(targets: Array<string | undefined>): boolean {
  if (!targets.length) return false;
  return !isFullTarget(targets[targets.length - 1]);
}
