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
import { describe, expect, it } from 'vitest';
import {
  buildBatchDetailRows,
  buildPreviewSquares,
  buildStrategyPreviewRows,
  buildStrategyPreviewSummary,
  calcBatchSegmentRatios,
  DEFAULT_PAUSE_SECONDS,
  filterStrategiesByName,
  formatAccumulatedTarget,
  formatBatchCardTitle,
  formatBatchesSummary,
  formatBatchPauseText,
  formatPauseMode,
  formatTargetValue,
  generateCloneName,
  isFullBatch,
  NAME_MAX_LENGTH,
  parseTarget,
  sortStrategies,
  toBatchForms,
  toBatchInputs,
  validatePauseSeconds,
  validateStrategyForm,
  validateStrategyName,
  validateTargetValue,
} from '~/pages/basic/release-strategies/utils';

import type { BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';
import type { StrategyBatchForm, StrategyFormData, TranslateFn } from '~/pages/basic/release-strategies/utils';

const texts = { full: '全量', unit: '个' };
const t: TranslateFn = (key, params) => {
  let out = key;
  if (Array.isArray(params)) params.forEach((p, i) => (out = out.replace(`{${i}}`, String(p))));
  return out;
};

/** 构造表单批次，减少用例噪音 */
function batch(partial: Partial<StrategyBatchForm> & { key: string }): StrategyBatchForm {
  return {
    targetUnit: 'count',
    targetValue: 1,
    pauseMode: 'none',
    ...partial,
  };
}

describe('发布策略纯函数：目标资源校验', () => {
  it('个为大于 0 的正整数时应通过校验', () => {
    expect(validateTargetValue(1, 'count')).toBe('');
    expect(validateTargetValue(100, 'count')).toBe('');
    expect(validateTargetValue('10', 'count')).toBe('');
  });

  it('百分比为 1% ~ 100% 的整数时应通过校验', () => {
    expect(validateTargetValue(1, 'percent')).toBe('');
    expect(validateTargetValue(25, 'percent')).toBe('');
    expect(validateTargetValue(100, 'percent')).toBe('');
  });

  it('空值、0、超范围百分比、非整数应被拦截', () => {
    expect(validateTargetValue(undefined, 'count')).toBe('请输入累计更新副本数');
    expect(validateTargetValue('', 'count')).toBe('请输入累计更新副本数');
    expect(validateTargetValue(0, 'count')).toContain('大于 0');
    expect(validateTargetValue(1.5, 'count')).toContain('大于 0');
    expect(validateTargetValue(0, 'percent')).toContain('1% ~ 100%');
    expect(validateTargetValue(101, 'percent')).toContain('1% ~ 100%');
    expect(validateTargetValue(1.5, 'percent')).toContain('1% ~ 100%');
  });
});

describe('发布策略纯函数：暂停秒数校验', () => {
  it('30 ~ 86400 之间的整数应通过', () => {
    expect(validatePauseSeconds(30)).toBe('');
    expect(validatePauseSeconds(86_400)).toBe('');
  });

  it('越界或非整数应被拦截', () => {
    expect(validatePauseSeconds(29)).toContain('30 ~ 86400');
    expect(validatePauseSeconds(86_401)).toContain('30 ~ 86400');
    expect(validatePauseSeconds(undefined)).toContain('30 ~ 86400');
    expect(validatePauseSeconds(1.5)).toContain('30 ~ 86400');
  });

  it('切到定时暂停时的默认时长应为 30 秒且通过校验', () => {
    expect(DEFAULT_PAUSE_SECONDS).toBe(30);
    expect(validatePauseSeconds(DEFAULT_PAUSE_SECONDS)).toBe('');
  });
});

describe('发布策略纯函数：批次累计值互转', () => {
  it('接口 target 应解析为数值 + 单位', () => {
    expect(parseTarget('1')).toEqual({ unit: 'count', value: 1 });
    expect(parseTarget('25%')).toEqual({ unit: 'percent', value: 25 });
    expect(parseTarget(' 100% ')).toEqual({ unit: 'percent', value: 100 });
    expect(parseTarget('')).toEqual({ unit: 'count', value: undefined });
    expect(parseTarget(undefined)).toEqual({ unit: 'count', value: undefined });
  });

  it('表单批次应拼回接口 target 字符串', () => {
    expect(formatTargetValue(batch({ key: 'a', targetUnit: 'count', targetValue: 3 }))).toBe('3');
    expect(formatTargetValue(batch({ key: 'b', targetUnit: 'percent', targetValue: 25 }))).toBe('25%');
    expect(formatTargetValue(batch({ key: 'c', targetValue: undefined }))).toBe('');
  });

  it('累计到 100% 视为全量', () => {
    expect(isFullBatch(batch({ key: 'a', targetUnit: 'percent', targetValue: 100 }))).toBe(true);
    expect(isFullBatch(batch({ key: 'b', targetUnit: 'percent', targetValue: 99 }))).toBe(false);
    expect(isFullBatch(batch({ key: 'c', targetUnit: 'count', targetValue: 100 }))).toBe(false);
  });
});

describe('发布策略纯函数：表单整体校验', () => {
  const validForm: StrategyFormData = {
    name: '谨慎发布',
    description: '灰度放量',
    batches: [
      batch({ key: 'first', targetUnit: 'count', targetValue: 1 }),
      batch({ key: 'last', targetUnit: 'percent', targetValue: 100 }),
    ],
  };

  it('合法表单应通过校验', () => {
    expect(validateStrategyForm(validForm).valid).toBe(true);
  });

  it('名称空 / 超过 32 字应给出名称错误', () => {
    expect(validateStrategyForm({ ...validForm, name: '  ' }).nameError).toBe('策略名称不能为空');
    expect(validateStrategyForm({ ...validForm, name: 'x'.repeat(33) }).nameError).toBe('策略名称不能超过 32 个字符');
    expect(validateStrategyForm({ ...validForm, name: 'x'.repeat(32) }).nameError).toBeUndefined();
  });

  it('名称与当前空间已有策略重名时应拦截', () => {
    const options = { existingNames: ['生产谨慎发布', '分三批灰度'] };
    expect(validateStrategyForm({ ...validForm, name: '生产谨慎发布' }, options).valid).toBe(false);
    expect(validateStrategyForm({ ...validForm, name: '生产谨慎发布' }, options).nameError).toBe(
      '策略名称在当前空间已存在',
    );
    // 前后空格由 trim 归一
    expect(validateStrategyForm({ ...validForm, name: '  生产谨慎发布 ' }, options).nameError).toBe(
      '策略名称在当前空间已存在',
    );
    // 未命中重名时通过
    expect(validateStrategyForm(validForm, options).nameError).toBeUndefined();
    // 空名称仍优先给出非空提示，不叠加重名提示
    expect(validateStrategyName('', options)).toBe('策略名称不能为空');
  });

  it('编辑态与自身原名称相同不算重名，改成其他策略名仍拦截', () => {
    const options = { existingNames: ['生产谨慎发布', '分三批灰度'], selfName: '生产谨慎发布' };
    expect(validateStrategyName('生产谨慎发布', options)).toBe('');
    expect(validateStrategyName('分三批灰度', options)).toBe('策略名称在当前空间已存在');
    expect(validateStrategyForm({ ...validForm, name: '生产谨慎发布' }, options).valid).toBe(true);
  });

  it('名称比较按 trim 后精确匹配：大小写不同视为不同名称（由后端保底校验）', () => {
    expect(validateStrategyName('Alpha', { existingNames: ['alpha'] })).toBe('');
    expect(validateStrategyName('alpha', { existingNames: ['alpha'] })).toBe('策略名称在当前空间已存在');
    expect(validateStrategyName('谨慎发布', { existingNames: [''] })).toBe('');
  });

  it('说明超过 100 字应给出说明错误', () => {
    expect(validateStrategyForm({ ...validForm, description: 'x'.repeat(101) }).descriptionError).toBe(
      '策略说明不能超过 100 个字符',
    );
    expect(validateStrategyForm({ ...validForm, description: 'x'.repeat(100) }).descriptionError).toBeUndefined();
  });

  it('批次为空应给出提示', () => {
    const result = validateStrategyForm({ ...validForm, batches: [] });
    expect(result.valid).toBe(false);
    expect(result.batchesEmptyError).toBe('请至少配置一个批次');
    expect(Object.keys(result.batchErrors)).toHaveLength(0);
  });

  it('批次累计值非法时应按批次 key 定位', () => {
    const result = validateStrategyForm({
      ...validForm,
      batches: [
        batch({ key: 'bad', targetValue: undefined }),
        batch({ key: 'full', targetUnit: 'percent', targetValue: 100 }),
      ],
    });
    expect(result.valid).toBe(false);
    expect(result.batchErrors.bad).toBe('请输入累计更新副本数');
  });

  it('定时暂停未填合法秒数应定位到对应批次', () => {
    const result = validateStrategyForm({
      ...validForm,
      batches: [
        batch({ key: 'paused', targetUnit: 'count', targetValue: 1, pauseMode: 'duration', pauseSeconds: 10 }),
        batch({ key: 'full', targetUnit: 'percent', targetValue: 100 }),
      ],
    });
    expect(result.valid).toBe(false);
    expect(result.batchErrors.paused).toContain('30 ~ 86400');
  });

  it('末批不是 100% 时应通过：全量由系统在末批之后隐含补齐', () => {
    const result = validateStrategyForm({
      ...validForm,
      batches: [
        batch({ key: 'first', targetValue: 1 }),
        batch({ key: 'last', targetUnit: 'percent', targetValue: 50 }),
      ],
    });
    expect(result.valid).toBe(true);
    expect(result.batchErrors.last).toBeUndefined();
  });

  it('仅配置一个批次时应通过', () => {
    const result = validateStrategyForm({
      ...validForm,
      batches: [batch({ key: 'only', targetUnit: 'count', targetValue: 1 })],
    });
    expect(result.valid).toBe(true);
  });

  it('同单位累计值必须严格递增，跨单位跳过比较', () => {
    const decreasing = validateStrategyForm({
      ...validForm,
      batches: [
        batch({ key: 'a', targetUnit: 'count', targetValue: 10 }),
        batch({ key: 'b', targetUnit: 'count', targetValue: 5 }),
        batch({ key: 'c', targetUnit: 'percent', targetValue: 100 }),
      ],
    });
    // 卡片用简短文案，详情面板补上上一批累计值
    expect(decreasing.batchErrors.b).toBe('累计更新数需大于上一批');
    expect(decreasing.batchErrorDetails.b).toEqual({ key: '累计更新数需大于上一批（{0}）', params: ['10 个'] });
    expect(decreasing.batchErrorDetails.a).toBeUndefined();

    const mixedUnits = validateStrategyForm({
      ...validForm,
      batches: [
        batch({ key: 'a', targetUnit: 'count', targetValue: 10 }),
        batch({ key: 'b', targetUnit: 'percent', targetValue: 25 }),
        batch({ key: 'c', targetUnit: 'percent', targetValue: 100 }),
      ],
    });
    expect(mixedUnits.valid).toBe(true);
  });
});

describe('发布策略纯函数：批次卡片文案', () => {
  it('卡片标题应按「第 N 批 · 累计 X」拼接，未填写时展示待填写', () => {
    expect(formatBatchCardTitle(batch({ key: 'a', targetUnit: 'count', targetValue: 1 }), 0, t)).toBe(
      '第 1 批 · 累计 1 个',
    );
    expect(formatBatchCardTitle(batch({ key: 'b', targetUnit: 'percent', targetValue: 25 }), 1, t)).toBe(
      '第 2 批 · 累计 25%',
    );
    expect(formatBatchCardTitle(batch({ key: 'c', targetValue: undefined }), 2, t)).toBe('第 3 批 · 待填写');
  });

  it('暂停方式应区分不停 / 定时 / 永久', () => {
    expect(formatBatchPauseText(batch({ key: 'a', pauseMode: 'none' }), t)).toBe('不停');
    expect(formatBatchPauseText(batch({ key: 'b', pauseMode: 'duration', pauseSeconds: 600 }), t)).toBe(
      '暂停 600 秒后自动继续',
    );
    expect(formatBatchPauseText(batch({ key: 'c', pauseMode: 'permanent' }), t)).toBe('永久暂停，手动继续');
  });
});

describe('发布策略纯函数：预览数据', () => {
  const batches = [
    batch({ key: 'a', targetUnit: 'count', targetValue: 1 }),
    batch({ key: 'b', targetUnit: 'percent', targetValue: 25, pauseMode: 'duration', pauseSeconds: 600 }),
    batch({ key: 'c', targetUnit: 'percent', targetValue: 100, pauseMode: 'permanent' }),
  ];

  it('预览行应给出累计占比与批次标签', () => {
    const rows = buildStrategyPreviewRows(batches, t);
    expect(rows.map(row => row.label)).toEqual(['第 1 批', '第 2 批', '第 3 批']);
    expect(rows.map(row => row.targetText)).toEqual(['1', '25%', '100%']);
    // 不填实例总数时只按配置值示意：1 / 25% / 100%
    expect(rows.map(row => row.accumulatedRatio)).toEqual([0.01, 0.25, 1]);
    expect(rows.map(row => row.prevRatio)).toEqual([0, 0.01, 0.25]);
  });

  it('全为百分比时应直接按百分比换算占比', () => {
    const rows = buildStrategyPreviewRows(
      [
        batch({ key: 'a', targetUnit: 'percent', targetValue: 10 }),
        batch({ key: 'b', targetUnit: 'percent', targetValue: 25 }),
        batch({ key: 'c', targetUnit: 'percent', targetValue: 70 }),
      ],
      t,
    );
    // 末批 70% 未到全量，系统补齐第 4 批（100%）
    expect(rows.map(row => row.accumulatedRatio)).toEqual([0.1, 0.25, 0.7, 1]);
    expect(rows.map(row => row.countValue)).toEqual(['10%', '25%', '70%', '100%']);
  });

  it('预览行应标记全量批与暂停信息', () => {
    const rows = buildStrategyPreviewRows(batches, t);
    expect(rows.map(row => row.isFull)).toEqual([false, false, true]);
    expect(rows.map(row => row.hasPause)).toEqual([false, true, true]);
    expect(rows[1].pauseText).toBe('暂停 600 秒后自动继续');
    // 暂停标签配色：永久暂停告警橙、定时暂停主蓝
    expect(rows.map(row => row.pauseTheme)).toEqual(['brand', 'brand', 'warn']);
  });

  it('目标值应区分首批与累计文案，且不依赖实例总数', () => {
    const rows = buildStrategyPreviewRows(batches, t);
    expect(rows.map(row => row.countLabel)).toEqual(['首批', '累计', '累计']);
    expect(rows.map(row => row.countValue)).toEqual(['1 个', '25%', '100%']);
  });

  it('末批未配置全量时应由系统补齐一个全量批', () => {
    const rows = buildStrategyPreviewRows(
      [
        batch({ key: 'a', targetUnit: 'count', targetValue: 1 }),
        batch({ key: 'b', targetUnit: 'percent', targetValue: 50 }),
      ],
      t,
    );
    expect(rows).toHaveLength(3);
    expect(rows[2]).toMatchObject({
      label: '第 3 批',
      isFull: true,
      accumulatedRatio: 1,
      prevRatio: 0.5,
      hasPause: false,
    });
  });

  it('进度方块应按累计占比着色', () => {
    const squares = buildPreviewSquares({ prevRatio: 0.25, accumulatedRatio: 0.7 });
    expect(squares).toHaveLength(20);
    expect(squares.filter(square => square === 'prev')).toHaveLength(5);
    expect(squares.filter(square => square === 'new')).toHaveLength(9);
    expect(squares.filter(square => square === 'pending')).toHaveLength(6);
  });

  it('摘要应统计批次数、暂停次数、首批数量与流量批次', () => {
    expect(buildStrategyPreviewSummary(batches)).toEqual({
      batchCount: 3,
      pauseCount: 2,
      firstCountValue: '1',
      firstCountUnit: '个',
      trafficBatchCount: 0,
    });
    // 首批为百分比时单位取 %
    const percentFirst = buildStrategyPreviewSummary([batch({ key: 'a', targetUnit: 'percent', targetValue: 10 })]);
    expect(percentFirst).toMatchObject({ firstCountValue: '10', firstCountUnit: '%' });
  });
});

describe('发布策略纯函数：批次展示', () => {
  const batches: BatchOutputObj[] = [
    { target: '1', pauseMode: 'none' },
    { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
    { target: '50%', pauseMode: 'permanent' },
    { target: '100%', pauseMode: 'none' },
  ];

  it('摘要文案应按 (token) · (token) 拼接，100% 显示为全量', () => {
    expect(formatBatchesSummary(batches, texts)).toBe('(1个) · (25%) · (50%) · (全量)');
  });

  it('空批次摘要应为空串', () => {
    expect(formatBatchesSummary([], texts)).toBe('');
    expect(formatBatchesSummary(undefined, texts)).toBe('');
  });

  it('分段条比例应累加为 1 且顺序递增', () => {
    const ratios = calcBatchSegmentRatios(batches);
    expect(ratios).toHaveLength(4);
    const total = ratios.reduce((sum, r) => sum + r, 0);
    expect(total).toBeCloseTo(1);
    // 末批（全量）新增占比最大
    expect(ratios[3]).toBeGreaterThan(ratios[0]);
  });

  it('暂停方式文案：permanent / duration / none', () => {
    expect(formatPauseMode({ pauseMode: 'permanent' }, t)).toBe('永久暂停 · 手动继续');
    expect(formatPauseMode({ pauseMode: 'duration', pauseSeconds: 600 }, t)).toBe('暂停 600 秒后自动继续');
    expect(formatPauseMode({ pauseMode: 'none' }, t)).toBe('不停');
  });

  it('累计数文案：首批不带「累计」前缀，后续批次带前缀，100% 展示为「100%（全量）」', () => {
    expect(formatAccumulatedTarget({ target: '1' }, 0, texts, t)).toBe('1个');
    expect(formatAccumulatedTarget({ target: '25%' }, 1, texts, t)).toBe('累计 25%');
    expect(formatAccumulatedTarget({ target: '100%' }, 0, texts, t)).toBe('100%（全量）');
    expect(formatAccumulatedTarget({ target: '100%' }, 3, texts, t)).toBe('累计 100%（全量）');
  });

  it('明细行数据应按批次顺序给出「第 N 批 / 累计数 / 暂停方式」', () => {
    const rows = buildBatchDetailRows(batches, texts, t);
    expect(rows.map(row => row.label)).toEqual(['第 1 批', '第 2 批', '第 3 批', '第 4 批']);
    expect(rows.map(row => row.accumulated)).toEqual(['1个', '累计 25%', '累计 50%', '累计 100%（全量）']);
    expect(rows.map(row => row.pauseMode)).toEqual(['不停', '暂停 600 秒后自动继续', '永久暂停 · 手动继续', '不停']);
    expect(rows.map(row => row.key)).toEqual(['0', '1', '2', '3']);
  });

  it('明细行数据对空批次返回空数组', () => {
    expect(buildBatchDetailRows(undefined, texts, t)).toEqual([]);
  });
});

describe('发布策略纯函数：列表排序与过滤', () => {
  const list: ReleaseStrategyOutputObj[] = [
    { id: 'a', name: 'Alpha', isBuiltin: false, updatedAt: '2026-01-01T00:00:00Z' },
    { id: 'builtin', name: '平台默认', isBuiltin: true, updatedAt: '2026-01-03T00:00:00Z' },
    { id: 'b', name: 'Beta', isBuiltin: false, updatedAt: '2026-01-02T00:00:00Z' },
  ];

  it('空间策略按更新时间倒序在前，内置策略固定底部', () => {
    const sorted = sortStrategies(list);
    expect(sorted.map(s => s.id)).toEqual(['b', 'a', 'builtin']);
  });

  it('按名称模糊过滤（不区分大小写）', () => {
    expect(filterStrategiesByName(list, 'alpha').map(s => s.id)).toEqual(['a']);
    expect(filterStrategiesByName(list, '  ')).toHaveLength(3);
    expect(filterStrategiesByName(list, '不存在')).toHaveLength(0);
  });
});

describe('发布策略纯函数：克隆名称生成', () => {
  it('无重名时默认为「原名_克隆01」', () => {
    expect(generateCloneName('生产谨慎发布', [])).toBe('生产谨慎发布_克隆01');
    expect(generateCloneName('生产谨慎发布', ['生产谨慎发布'])).toBe('生产谨慎发布_克隆01');
  });

  it('与已有策略重名时序号递增', () => {
    expect(generateCloneName('生产谨慎发布', ['生产谨慎发布', '生产谨慎发布_克隆01'])).toBe('生产谨慎发布_克隆02');
    expect(
      generateCloneName('生产谨慎发布', ['生产谨慎发布_克隆01', '生产谨慎发布_克隆02', '生产谨慎发布_克隆03']),
    ).toBe('生产谨慎发布_克隆04');
  });

  it('原名过长时截断，保留克隆后缀以满足长度限制', () => {
    const longName = '策'.repeat(NAME_MAX_LENGTH);
    const name = generateCloneName(longName, []);
    expect(name).toHaveLength(NAME_MAX_LENGTH);
    expect(name.endsWith('_克隆01')).toBe(true);
  });
});

describe('发布策略纯函数：批次互转', () => {
  const output: BatchOutputObj[] = [
    { target: '1', pauseMode: 'none', pauseSeconds: 0 },
    { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
  ];

  it('接口批次转表单批次时拆为数值 + 单位，非 duration 的 pauseSeconds 应被忽略', () => {
    const forms = toBatchForms(output);
    expect(forms[0]).toEqual({
      key: expect.any(String),
      targetUnit: 'count',
      targetValue: 1,
      pauseMode: 'none',
      pauseSeconds: undefined,
    });
    expect(forms[1]).toEqual({
      key: expect.any(String),
      targetUnit: 'percent',
      targetValue: 25,
      pauseMode: 'duration',
      pauseSeconds: 600,
    });
    expect(forms[0].key).not.toBe(forms[1].key);
  });

  it('表单批次转接口批次时，仅 duration 携带 pauseSeconds', () => {
    const inputs = toBatchInputs([
      batch({ key: 'a', targetUnit: 'count', targetValue: 1, pauseSeconds: 999 }),
      batch({ key: 'b', targetUnit: 'percent', targetValue: 25, pauseMode: 'duration', pauseSeconds: 600 }),
    ]);
    expect(inputs[0]).toEqual({ target: '1', pauseMode: 'none' });
    expect(inputs[1]).toEqual({ target: '25%', pauseMode: 'duration', pauseSeconds: 600 });
  });
});
