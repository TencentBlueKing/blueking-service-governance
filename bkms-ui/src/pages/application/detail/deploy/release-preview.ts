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

/** 进度方块着色：prev=此前批次已更新，new=本批新增，pending=尚未更新 */
export type ProgressSquareState = 'new' | 'pending' | 'prev';

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
  firstIsPercent: boolean;
  /** 首批 target 原文（如 "1" / "25%"） */
  firstTarget: string;
  /** 末批是否 100%（全量） */
  lastIsFull: boolean;
  /** 末批 target 原文 */
  lastTarget: string;
  /** 中间批次 target 原文列表 */
  middleTargets: string[];
}

/** target 展示文案的 i18n 词条（全量 / 个） */
export interface TargetTexts {
  full: string;
  unit: string;
}

/**
 * 为单个批次生成实例进度方块着色序列。
 * 实例数超过 maxSquares 时按比例分组合并（每个方块代表一段实例，取段首实例着色）。
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
 */
export function buildReleasePreview(batches: BatchOutputObj[] | undefined, replicas: number): ReleasePreviewResult {
  const list = batches ?? [];
  const total = Math.max(1, Math.floor(Number(replicas) || 0));

  let prevCount = 0;
  const rows: ReleasePreviewRow[] = list.map((batch, index) => {
    const pauseMode = batch.pauseMode ?? 'none';
    const targetText = (batch.target ?? '').trim() || '100%';
    const isPercent = targetText.endsWith('%');
    const row: ReleasePreviewRow = {
      batchIndex: index + 1,
      prevCount,
      updatedCount: resolveBatchUpdatedCount(targetText, total),
      targetText,
      isPercent,
      isFull: isPercent && Number.parseFloat(targetText) >= 100,
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

/** 解析策略批次为目标说明结构；空批次返回 null */
export function buildStrategyHint(batches: BatchOutputObj[] | undefined): null | StrategyHintParts {
  const targets = (batches ?? []).map(batch => (batch.target ?? '').trim() || '100%');
  if (targets.length === 0) return null;
  const lastTarget = targets[targets.length - 1] ?? '';
  return {
    firstTarget: targets[0] ?? '',
    firstIsPercent: (targets[0] ?? '').endsWith('%'),
    middleTargets: targets.slice(1, -1),
    lastTarget,
    lastIsFull: lastTarget.endsWith('%') && Number.parseFloat(lastTarget) >= 100,
  };
}

/** 将单个批次 target 格式化为选项摘要 token：100% → 全量、正整数 → "1个"、百分比 → "25%" */
export function formatOptionTarget(target: string | undefined, texts: TargetTexts): string {
  const trimmed = (target ?? '').trim() || '100%';
  if (trimmed.endsWith('%')) {
    const pct = Number.parseFloat(trimmed.slice(0, -1));
    return Number.isNaN(pct) || pct >= 100 ? texts.full : trimmed;
  }
  return `${trimmed}${texts.unit}`;
}

/**
 * 将单个批次 target（正整数或百分比）换算为"累计更新到"的实例数。
 * - 百分比：按实例总数向上取整（如 20 × 25% = 5，20 × 33% = 6.6 → 7），不超过实例总数；
 * - 正整数：直接取该值，超过实例总数时收敛为实例总数；
 * - 非法值：兜底返回实例总数（视同全量）。
 */
export function resolveBatchUpdatedCount(target: string | undefined, replicas: number): number {
  const total = Math.max(1, Math.floor(Number(replicas) || 0));
  const trimmed = (target ?? '').trim();
  if (!trimmed) return total;

  if (trimmed.endsWith('%')) {
    const pct = Number.parseFloat(trimmed.slice(0, -1));
    if (Number.isNaN(pct) || pct <= 0) return total;
    return Math.min(Math.ceil((total * pct) / 100), total);
  }

  const count = Number.parseInt(trimmed, 10);
  if (Number.isNaN(count) || count <= 0) return total;
  return Math.min(count, total);
}
