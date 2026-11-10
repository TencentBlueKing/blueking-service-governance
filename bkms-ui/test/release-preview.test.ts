/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) that applies to the current version
 * of the project delivered to anyone in the future.
 */

import { describe, expect, it } from 'vitest';
import {
  buildBatchSquares,
  buildReleasePreview,
  buildSquaresByRatio,
  buildStrategyHint,
  formatOptionTarget,
  isFullTarget,
  normalizeTargetText,
  parseBatchTarget,
  resolveBatchUpdatedCount,
  shouldAppendAutoFullBatch,
} from '~/pages/application/detail/deploy/release-preview';

import type { BatchOutputObj } from '~/@types/v1/release-strategies';

/**
 * 覆盖部署侧栏发布预览的批次换算纯函数：
 *   1. 百分比 target 按实例总数向上取整换算；
 *   2. 正整数 target 直接取数且不越界；
 *   3. 暂停次数统计与时间线行的 pause 结构（permanent / duration / none）。
 */

describe('resolveBatchUpdatedCount', () => {
  it('百分比按实例总数向上取整换算（20 × 25% = 5）', () => {
    expect(resolveBatchUpdatedCount('25%', 20)).toBe(5);
  });

  it('百分比向上取整（20 × 33% = 6.6 → 7）', () => {
    expect(resolveBatchUpdatedCount('33%', 20)).toBe(7);
  });

  it('全量百分比等于实例总数', () => {
    expect(resolveBatchUpdatedCount('100%', 20)).toBe(20);
  });

  it('正整数 target 直接取数', () => {
    expect(resolveBatchUpdatedCount('1', 20)).toBe(1);
  });

  it('正整数 target 超过实例总数时收敛为实例总数', () => {
    expect(resolveBatchUpdatedCount('50', 20)).toBe(20);
  });

  it('target 为空或非法时兜底返回实例总数', () => {
    expect(resolveBatchUpdatedCount('', 20)).toBe(20);
    expect(resolveBatchUpdatedCount(undefined, 20)).toBe(20);
    expect(resolveBatchUpdatedCount('abc', 20)).toBe(20);
  });

  it('实例总数非法时兜底为 1，且换算不越界', () => {
    expect(resolveBatchUpdatedCount('100%', 0)).toBe(1);
    expect(resolveBatchUpdatedCount('100%', -5)).toBe(1);
  });
});

describe('buildReleasePreview', () => {
  const batches: BatchOutputObj[] = [
    { target: '1', pauseMode: 'none' },
    { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
    { target: '50%', pauseMode: 'permanent' },
    { target: '100%', pauseMode: 'none' },
  ];

  it('摘要栏统计：批次数、暂停次数、实例总数、流量批次固定 0', () => {
    const { summary } = buildReleasePreview(batches, 20);
    expect(summary).toEqual({
      batchCount: 4,
      pauseCount: 2,
      totalReplicas: 20,
      trafficBatchCount: 0,
    });
  });

  it('时间线行按批次顺序生成，百分比与正整数 target 均正确换算', () => {
    const { rows } = buildReleasePreview(batches, 20);
    expect(rows.map(row => row.batchIndex)).toEqual([1, 2, 3, 4]);
    expect(rows.map(row => row.updatedCount)).toEqual([1, 5, 10, 20]);
  });

  it('duration 暂停携带秒数，permanent 暂停不携带秒数', () => {
    const { rows } = buildReleasePreview(batches, 20);
    expect(rows[1].pause).toEqual({ mode: 'duration', seconds: 600 });
    expect(rows[2].pause).toEqual({ mode: 'permanent', seconds: undefined });
  });

  it('pauseMode 为 none 的批次不生成暂停信息', () => {
    const { rows } = buildReleasePreview(batches, 20);
    expect(rows[0].pause).toBeUndefined();
    expect(rows[3].pause).toBeUndefined();
  });

  it('pauseMode 缺失时按 none 处理（不统计、不生成暂停）', () => {
    const { summary, rows } = buildReleasePreview([{ target: '100%' }], 20);
    expect(summary.pauseCount).toBe(0);
    expect(rows[0].pause).toBeUndefined();
  });

  it('批次列表为空或缺失时返回空时间线与零统计', () => {
    const empty = buildReleasePreview([], 20);
    expect(empty.summary.batchCount).toBe(0);
    expect(empty.rows).toEqual([]);

    const missing = buildReleasePreview(undefined, 20);
    expect(missing.summary.batchCount).toBe(0);
    expect(missing.rows).toEqual([]);
  });

  it('实例总数非法时兜底为 1', () => {
    const { summary } = buildReleasePreview(batches, 0);
    expect(summary.totalReplicas).toBe(1);
  });

  it('时间线行携带 target 元信息与前一批次累计（用于标题与进度方块着色）', () => {
    const { rows } = buildReleasePreview(batches, 20);
    expect(rows[0]).toMatchObject({ prevCount: 0, targetText: '1', isPercent: false, isFull: false });
    expect(rows[1]).toMatchObject({ prevCount: 1, targetText: '25%', isPercent: true, isFull: false });
    expect(rows[2]).toMatchObject({ prevCount: 5, targetText: '50%', isPercent: true, isFull: false });
    expect(rows[3]).toMatchObject({ prevCount: 10, targetText: '100%', isPercent: true, isFull: true });
    // 末批已是全量，无需系统补齐
    expect(rows.every(row => !row.isAutoFull)).toBeTruthy();
  });
});

describe('末批全量补齐语义（策略预览 / 部署预览共用）', () => {
  it('isFullTarget：100% 及以上与空值视为全量，其余视为部分', () => {
    expect(isFullTarget('100%')).toBe(true);
    expect(isFullTarget('120%')).toBe(true);
    expect(isFullTarget(undefined)).toBe(true);
    expect(isFullTarget('')).toBe(true);
    expect(isFullTarget('50%')).toBe(false);
    // 正整数 target 表示实例个数，永远不是全量
    expect(isFullTarget('3')).toBe(false);
  });

  it('shouldAppendAutoFullBatch：仅当末批非全量时补齐', () => {
    expect(shouldAppendAutoFullBatch(['1', '50%'])).toBe(true);
    expect(shouldAppendAutoFullBatch(['1', '100%'])).toBe(false);
    expect(shouldAppendAutoFullBatch([])).toBe(false);
  });

  it('末批非全量时追加「系统补齐的全量批」，与策略预览一致', () => {
    const { rows, summary } = buildReleasePreview(
      [
        { target: '1', pauseMode: 'none' },
        { target: '50%', pauseMode: 'permanent' },
      ],
      20,
    );
    expect(rows.map(row => row.batchIndex)).toEqual([1, 2, 3]);
    // 补齐批：接在末批之后、累计到实例总数、带全量标记、无暂停
    expect(rows[2]).toMatchObject({
      isAutoFull: true,
      isFull: true,
      isPercent: true,
      prevCount: 10,
      targetText: '100%',
      updatedCount: 20,
    });
    expect(rows[2].pause).toBeUndefined();
    // 用户配置的批次不被标记为补齐
    expect(rows.filter(row => row.isAutoFull)).toHaveLength(1);
    // 摘要批次数按用户配置的策略批次统计，与策略预览摘要口径一致
    expect(summary.batchCount).toBe(2);
  });
});

describe('buildBatchSquares', () => {
  it('实例数不超过上限时按 1:1 生成方块并分段着色', () => {
    // 累计 1/20：1 个本批新增，19 个未更新
    expect(buildBatchSquares({ prevCount: 0, updatedCount: 1 }, 20)).toEqual([
      'new',
      ...Array.from({ length: 19 }, () => 'pending'),
    ]);
    // 累计 5/20（前批已 1）：1 个此前批次、4 个本批新增、15 个未更新
    expect(buildBatchSquares({ prevCount: 1, updatedCount: 5 }, 20)).toEqual([
      'prev',
      'new',
      'new',
      'new',
      'new',
      ...Array.from({ length: 15 }, () => 'pending'),
    ]);
  });

  it('实例数超过上限时按比例合并方块', () => {
    // 100 实例、上限 20 方块：每方块代表 5 实例，累计 50 时前 10 个方块为此前批次
    const squares = buildBatchSquares({ prevCount: 50, updatedCount: 50 }, 100);
    expect(squares).toHaveLength(20);
    expect(squares.filter(state => state === 'prev')).toHaveLength(10);
    expect(squares.filter(state => state === 'pending')).toHaveLength(10);
  });

  it('实例数非法时按 1 个实例兜底', () => {
    expect(buildBatchSquares({ prevCount: 0, updatedCount: 1 }, 0)).toEqual(['new']);
  });
});

describe('buildStrategyHint', () => {
  it('解析批次序列为首/中/末段结构', () => {
    const hint = buildStrategyHint([
      { target: '1', pauseMode: 'none' },
      { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
      { target: '50%', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
    ]);
    expect(hint).toEqual({
      batchCount: 4,
      firstTarget: '1',
      firstIsPercent: false,
      middleTargets: ['25%', '50%'],
      lastIsFull: true,
    });
  });

  it('末批非全量时归入累计段，收尾由系统补齐的全量批承担', () => {
    // 「先 1 个实例，确认后累计到 25%、50%，最后自动全量」
    expect(
      buildStrategyHint([
        { target: '1', pauseMode: 'none' },
        { target: '25%', pauseMode: 'none' },
        { target: '50%', pauseMode: 'none' },
      ]),
    ).toEqual({
      batchCount: 3,
      firstTarget: '1',
      firstIsPercent: false,
      middleTargets: ['25%', '50%'],
      lastIsFull: false,
    });
  });

  it('两批策略（25% → 100%）不应被误判为单批全量：batchCount 保留批次数供组件区分', () => {
    const hint = buildStrategyHint([
      { target: '25%', pauseMode: 'none' },
      { target: '100%', pauseMode: 'none' },
    ]);
    expect(hint).toEqual({
      batchCount: 2,
      firstTarget: '25%',
      firstIsPercent: true,
      middleTargets: [],
      lastIsFull: true,
    });
  });

  it('空批次返回 null，缺失 target 按全量兜底', () => {
    expect(buildStrategyHint([])).toBeNull();
    expect(buildStrategyHint(undefined)).toBeNull();
    expect(buildStrategyHint([{ pauseMode: 'none' }])).toEqual({
      batchCount: 1,
      firstTarget: '100%',
      firstIsPercent: true,
      middleTargets: [],
      lastIsFull: true,
    });
  });
});

describe('formatOptionTarget', () => {
  const texts = { full: '全量', unit: '个' };

  it('全量百分比展示为全量词条', () => {
    expect(formatOptionTarget('100%', texts)).toBe('全量');
    expect(formatOptionTarget('120%', texts)).toBe('全量');
  });

  it('部分百分比保留原文，正整数追加单位', () => {
    expect(formatOptionTarget('25%', texts)).toBe('25%');
    expect(formatOptionTarget('1', texts)).toBe('1个');
  });

  it('缺失或非法 target 按全量兜底', () => {
    expect(formatOptionTarget('', texts)).toBe('全量');
    expect(formatOptionTarget(undefined, texts)).toBe('全量');
  });
});

describe('parseBatchTarget（策略预览 / 部署预览共用的 target 解析）', () => {
  it('百分比解析为数值与单位，≥100 判为全量', () => {
    expect(parseBatchTarget('25%')).toEqual({ isFull: false, text: '25%', unit: 'percent', value: 25 });
    expect(parseBatchTarget(' 100% ')).toEqual({ isFull: true, text: '100%', unit: 'percent', value: 100 });
    expect(parseBatchTarget('120%').isFull).toBe(true);
  });

  it('正整数解析为个数，永远不是全量', () => {
    expect(parseBatchTarget('1')).toEqual({ isFull: false, text: '1', unit: 'count', value: 1 });
    expect(parseBatchTarget('100').isFull).toBe(false);
  });

  it('空串保留为空文本（表单回填需区分待填写），由 isFullTarget / normalizeTargetText 决定展示语义', () => {
    expect(parseBatchTarget('')).toEqual({ isFull: false, text: '', unit: 'count', value: undefined });
    expect(parseBatchTarget(undefined).text).toBe('');
    expect(normalizeTargetText('')).toBe('100%');
    expect(isFullTarget('')).toBe(true);
  });

  it('非法数值收敛为 undefined，不误判全量', () => {
    expect(parseBatchTarget('abc').value).toBeUndefined();
    expect(parseBatchTarget('abc').isFull).toBe(false);
    expect(parseBatchTarget('abc%').value).toBeUndefined();
    expect(parseBatchTarget('0%').value).toBe(0);
  });
});

describe('buildSquaresByRatio（策略预览固定格数着色）', () => {
  it('按格中心落点着色，与原 buildPreviewSquares 行为一致', () => {
    // prev 0.25 / accumulated 0.7：5 格此前批次、9 格本批新增、6 格未更新
    const squares = buildSquaresByRatio({ accumulatedRatio: 0.7, prevRatio: 0.25 });
    expect(squares).toHaveLength(20);
    expect(squares.filter(state => state === 'prev')).toHaveLength(5);
    expect(squares.filter(state => state === 'new')).toHaveLength(9);
    expect(squares.filter(state => state === 'pending')).toHaveLength(6);
  });
});
