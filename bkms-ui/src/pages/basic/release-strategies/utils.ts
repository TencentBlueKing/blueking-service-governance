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

import { formatOptionTarget } from '~/pages/application/detail/deploy/release-preview';

import type { BatchInput, BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';
import type { TargetTexts } from '~/pages/application/detail/deploy/release-preview';

/** 详情面板的带参错误文案：key 为 i18n 模板，params 依序插值 */
export interface BatchErrorDetail {
  key: string;
  params: string[];
}

/** 批次完成后的暂停方式：none 不暂停 / duration 定时暂停 / permanent 永久暂停 */
export type PauseMode = 'duration' | 'none' | 'permanent';

/** 进度方块单格状态：已更新 / 本批新增 / 待更新 */
export type PreviewSquareState = 'new' | 'pending' | 'prev';

/**
 * 批次行表单模型。
 * 目标资源在设计稿中拆为「数字 + 单位」，提交时再拼回 "1" / "25%" 的接口契约。
 */
export interface StrategyBatchForm {
  /**
   * 稳定标识：新增/删除批次后选中态与错误提示不会因下标漂移而错位
   */
  key: string;
  pauseMode: PauseMode;
  /** 仅 duration 模式有意义，30 ~ 86400 秒 */
  pauseSeconds?: number;
  /** 累计值：正整数；单位为 percent 时范围 1 ~ 100 */
  targetUnit: TargetUnit;
  targetValue?: number;
}

/** 策略抽屉表单模型 */
export interface StrategyFormData {
  batches: StrategyBatchForm[];
  description: string;
  name: string;
}

/**
 * 名称唯一性校验的上下文。
 * 前端按当前空间已加载的策略列表做同步校验（输入即反馈），后端仍会保底校验同名。
 */
export interface StrategyFormValidateOptions {
  /** 当前空间已有策略名称（含平台内置策略） */
  existingNames?: string[];
  /** 编辑态被编辑策略的原名称：与自身同名不算重复 */
  selfName?: string;
}

/** 表单校验结果：错误值为 i18n key（项目约定中文 key 直译） */
export interface StrategyFormValidation {
  /** 批次行级错误的详情文案（如带上上一批累计值），缺省时回落到 batchErrors */
  batchErrorDetails: Record<string, BatchErrorDetail>;
  /**
   * 批次行级错误（简短文案），key 为批次的稳定标识。
   * 批次卡片第二行直接展示，宽度有限不携带参数。
   */
  batchErrors: Record<string, string>;
  /** 表单级错误：批次列表为空 */
  batchesEmptyError?: string;
  descriptionError?: string;
  nameError?: string;
  valid: boolean;
}

/** 预览弹窗中的一行（仅在表单校验通过后构建，不存在「待填写」态） */
export interface StrategyPreviewRow {
  /** 本批累计占比（0 ~ 1），用于进度方块着色 */
  accumulatedRatio: number;
  /** 目标值右侧的计量说明：首批 / 累计（设计稿「1 个 首批」「25% 累计」） */
  countLabel: string;
  /** 目标值主文案（「1 个」/「25%」） */
  countValue: string;
  /** 本批之后是否需要暂停（none 时不展示暂停标签） */
  hasPause: boolean;
  /** 本批累计目标是否已达 100%（展示「全量」标签） */
  isFull: boolean;
  key: string;
  /** 第 N 批 */
  label: string;
  /** 暂停方式文案 */
  pauseText: string;
  /** 暂停标签配色：permanent 橙色告警 / duration 蓝色提示 */
  pauseTheme: 'brand' | 'warn';
  /** 本批之前累计占比（0 ~ 1），与 accumulatedRatio 之间为「本批新增」区段 */
  prevRatio: number;
  /** 原始 target 文本（如 "1" / "25%"），无目标时为空串 */
  targetText: string;
}

/** 预览弹窗摘要栏指标 */
export interface StrategyPreviewSummary {
  batchCount: number;
  /** 首批目标值文案与单位（count → 个，percent → %） */
  firstCountUnit: string;
  firstCountValue: string;
  pauseCount: number;
  /** 流量批次：当前策略仅支持更新批次，固定为 0 */
  trafficBatchCount: number;
}

/** 进度方块格数，与设计稿一致 */
export const PREVIEW_SQUARE_COUNT = 20;

/** 累计更新目标的计量单位：count 个（实例数） / percent 百分比 */
export type TargetUnit = 'count' | 'percent';

/** 轻量翻译函数结构类型（对齐项目 use-tag-config.ts 等约定，兼容 vue-i18n 数组/命名插值） */
export type TranslateFn = (key: string, params?: Record<string, unknown> | unknown[]) => string;

/**
 * 带参数翻译：部分运行环境（dev 下 vue-i18n 运行时编译不可用）会原样返回含 `{0}` 的模板串，
 * 这里对残留占位符再做一次替换；译文已插值时不会命中，属幂等操作。
 */
export function translate(t: TranslateFn, key: string, params: Array<number | string>): string {
  const text = t(key, params);
  return params.reduce<string>((out, param, index) => out.split(`{${index}}`).join(String(param)), text);
}

export const NAME_MAX_LENGTH = 32;
export const DESCRIPTION_MAX_LENGTH = 100;
export const PAUSE_SECONDS_MIN = 30;
export const PAUSE_SECONDS_MAX = 86_400;
/** 切到「定时暂停」时的默认时长，与设计稿一致 */
export const DEFAULT_PAUSE_SECONDS = 30;

const PAUSE_MODES: PauseMode[] = ['none', 'duration', 'permanent'];

/** 明细 Popover 的批次行（模板直接渲染，避免在模板里做下标运算） */
export interface BatchDetailRow {
  /** 累计数文案 */
  accumulated: string;
  /** 行 key（批次下标） */
  key: string;
  /** 第 N 批 */
  label: string;
  /** 暂停方式文案 */
  pauseMode: string;
}

let batchKeySeed = 0;

/** 构建明细 Popover 的批次行数据 */
export function buildBatchDetailRows(
  batches: BatchOutputObj[] | undefined,
  texts: TargetTexts,
  t: TranslateFn,
): BatchDetailRow[] {
  return (batches ?? []).map((batch, index) => ({
    key: String(index),
    label: translate(t, '第 {0} 批', [index + 1]),
    accumulated: formatAccumulatedTarget(batch, index, texts, t),
    pauseMode: formatPauseMode(batch, t),
  }));
}

/**
 * 进度方块着色：第 i 格代表区间 [i / count, (i + 1) / count]，
 * 按格中心落在「已更新 / 本批新增 / 待更新」哪一段决定颜色。
 */
export function buildPreviewSquares(
  row: Pick<StrategyPreviewRow, 'accumulatedRatio' | 'prevRatio'>,
  count = PREVIEW_SQUARE_COUNT,
): PreviewSquareState[] {
  return Array.from({ length: count }, (_, index) => {
    const position = (index + 0.5) / count;
    if (position <= row.prevRatio) return 'prev';
    if (position <= row.accumulatedRatio) return 'new';
    return 'pending';
  });
}

/**
 * 已校验的表单批次 → 预览弹窗时间线行。
 *
 * 全量由系统补齐：用户把末批配置到 100% 时即视为全量批；
 * 未配置到 100%（如配置 25% / 70%）时，系统会在末批确认后自动补一个全量批，
 * 与「最后自动全量」的语义对齐，因此此处追加一行系统补齐的全量批。
 */
export function buildStrategyPreviewRows(batches: StrategyBatchForm[], t: TranslateFn): StrategyPreviewRow[] {
  const ratios = resolvePreviewRatios(batches);
  let prevRatio = 0;

  const rows = batches.map((batch, index) => {
    const targetText = formatTargetValue(batch);
    const accumulatedRatio = ratios[index] ?? 1;
    const row: StrategyPreviewRow = {
      key: batch.key,
      label: translate(t, '第 {0} 批', [index + 1]),
      targetText,
      countValue: formatBatchTargetText(batch, t),
      countLabel: index === 0 ? t('首批') : t('累计'),
      prevRatio,
      accumulatedRatio,
      isFull: isFullBatch(batch),
      hasPause: batch.pauseMode !== 'none',
      pauseText: formatBatchPauseText(batch, t),
      pauseTheme: batch.pauseMode === 'permanent' ? 'warn' : 'brand',
    };
    prevRatio = accumulatedRatio;
    return row;
  });

  // 末批已全量则无需补齐；否则系统在末批确认后自动补齐剩余实例
  const lastBatch = batches[batches.length - 1];
  if (lastBatch && !isFullBatch(lastBatch)) {
    rows.push({
      key: `${lastBatch.key}-auto-full`,
      label: translate(t, '第 {0} 批', [batches.length + 1]),
      targetText: '100%',
      countValue: '100%',
      countLabel: t('累计'),
      prevRatio,
      accumulatedRatio: 1,
      isFull: true,
      hasPause: false,
      pauseText: '',
      pauseTheme: 'brand',
    });
  }

  return rows;
}

/** 预览摘要：批次数 / 暂停次数 / 首批数量 / 流量批次（本期无流量批次，固定 0） */
export function buildStrategyPreviewSummary(batches: StrategyBatchForm[]): StrategyPreviewSummary {
  const firstBatch = batches[0];
  const firstValue = firstBatch?.targetValue;
  const hasFirst = firstValue !== undefined && firstValue !== null && !Number.isNaN(Number(firstValue));
  return {
    batchCount: batches.length,
    pauseCount: batches.filter(batch => batch.pauseMode !== 'none').length,
    firstCountValue: hasFirst ? String(Number(firstValue)) : '0',
    firstCountUnit: firstBatch?.targetUnit === 'percent' ? '%' : '个',
    trafficBatchCount: 0,
  };
}

/**
 * 计算批次分段条各段宽度占比（0 ~ 1）。
 * 每段按本批新增占比绘制：将各批累计 target 解析为数值（百分比去 %、正整数原样），
 * 取相邻差值为段宽，差值过小的批次兜底 0.5 保证可见。
 * 注意：仅作可视化示意，混用「正整数实例数」与「百分比」两种量纲时差值不具精确占比语义。
 */
export function calcBatchSegmentRatios(batches: BatchOutputObj[] | undefined): number[] {
  const values = (batches ?? []).map(batch => {
    const target = (batch.target ?? '').trim() || '100%';
    if (target.endsWith('%')) {
      const pct = Number.parseFloat(target.slice(0, -1));
      return Number.isNaN(pct) ? 100 : Math.min(Math.max(pct, 0), 100);
    }
    const count = Number.parseInt(target, 10);
    return Number.isNaN(count) || count < 0 ? 0 : count;
  });
  if (values.length === 0) return [];

  let prev = 0;
  const widths = values.map(value => {
    const width = Math.max(value - prev, 0.5);
    prev = value;
    return width;
  });
  const total = widths.reduce((sum, width) => sum + width, 0) || 1;
  return widths.map(width => width / total);
}

/** 生成批次稳定标识（自增序号即可满足单次表单内唯一） */
export function createBatchKey(): string {
  batchKeySeed += 1;
  return `batch-${batchKeySeed}`;
}

/**
 * 新增批次：累计值留空由用户填写（不预填默认值，避免默认值被误提交），默认永久暂停等人工确认。
 * 单位仍默认「个」：首次填写按实例数理解更直观，需要百分比时再切换。
 */
export function createEmptyBatch(): StrategyBatchForm {
  return {
    key: createBatchKey(),
    targetUnit: 'count',
    targetValue: undefined,
    pauseMode: 'permanent',
  };
}

/** 按名称模糊过滤（不区分大小写） */
export function filterStrategiesByName(list: ReleaseStrategyOutputObj[], keyword: string): ReleaseStrategyOutputObj[] {
  const trimmed = keyword.trim().toLowerCase();
  if (!trimmed) return list;
  return list.filter(strategy => (strategy.name ?? '').toLowerCase().includes(trimmed));
}

/**
 * 批次明细「累计数」列文案：首批直接展示 token，后续批次加「累计」前缀。
 * 100% 展示为「100%（全量）」而非「全量」，与设计稿明细表一致。
 */
export function formatAccumulatedTarget(
  batch: Pick<BatchOutputObj, 'target'>,
  index: number,
  texts: TargetTexts,
  t: TranslateFn,
): string {
  const target = (batch.target ?? '').trim();
  const isFull = target.endsWith('%') && Number.parseFloat(target) >= 100;
  const token = isFull ? t('{0}（全量）', ['100%']) : formatOptionTarget(batch.target, texts);
  return index === 0 ? token : t('累计 {0}', [token]);
}

/** 批次卡片第一行：`第 N 批 · 累计 1 个` / `第 N 批 · 累计 25%` / 未填时 `第 N 批 · 待填写` */
export function formatBatchCardTitle(
  batch: Pick<StrategyBatchForm, 'targetUnit' | 'targetValue'>,
  index: number,
  t: TranslateFn,
): string {
  const label = translate(t, '第 {0} 批', [index + 1]);
  const target = formatBatchTargetText(batch, t);
  if (!target) return `${label} · ${t('待填写')}`;
  return `${label} · ${translate(t, '累计 {0}', [target])}`;
}

/**
 * 生成批次摘要文案（批次列展示）：如 "(1个) · (25%) · (全量)"。
 * 空批次返回空串。
 */
export function formatBatchesSummary(batches: BatchOutputObj[] | undefined, texts: TargetTexts): string {
  const list = batches ?? [];
  if (list.length === 0) return '';
  return list.map(batch => `(${formatOptionTarget(batch.target, texts)})`).join(' · ');
}

/**
 * 批次卡片第二行的暂停方式文案：不停 / 暂停 600 秒后自动继续 / 永久暂停，手动继续。
 * 永久暂停的文案与列表页 `formatPauseMode`（用 · 分隔）略有差异，与设计稿保持一致。
 */
export function formatBatchPauseText(
  batch: Pick<StrategyBatchForm, 'pauseMode' | 'pauseSeconds'>,
  t: TranslateFn,
): string {
  if (batch.pauseMode === 'permanent') return t('永久暂停，手动继续');
  if (batch.pauseMode === 'duration') {
    return translate(t, '暂停 {0} 秒后自动继续', [Number(batch.pauseSeconds) || DEFAULT_PAUSE_SECONDS]);
  }
  return t('不停');
}

/** 表单批次的累计值文案：个数带单位「个」，百分比如 25% */
export function formatBatchTargetText(
  batch: Pick<StrategyBatchForm, 'targetUnit' | 'targetValue'>,
  t: TranslateFn,
): string {
  const target = formatTargetValue(batch);
  if (!target) return '';
  return batch.targetUnit === 'percent' ? target : translate(t, '{0} 个', [Number(batch.targetValue)]);
}

/** 暂停方式展示文案：permanent → 永久暂停 · 手动继续 / duration → 暂停 N 秒后自动继续 / none → 不停 */
export function formatPauseMode(batch: Pick<BatchOutputObj, 'pauseMode' | 'pauseSeconds'>, t: TranslateFn): string {
  const mode = batch.pauseMode ?? 'none';
  if (mode === 'permanent') return t('永久暂停 · 手动继续');
  if (mode === 'duration') return translate(t, '暂停 {0} 秒后自动继续', [batch.pauseSeconds ?? 0]);
  return t('不停');
}

/** 表单批次 → 接口 target 字符串（"1" / "25%"），未填返回空串 */
export function formatTargetValue(batch: Pick<StrategyBatchForm, 'targetUnit' | 'targetValue'>): string {
  const value = batch.targetValue;
  if (value === undefined || value === null || Number.isNaN(Number(value))) return '';
  return batch.targetUnit === 'percent' ? `${Number(value)}%` : String(Number(value));
}

/**
 * 克隆策略的默认名称：`原名_克隆01`。
 * 与当前空间已有策略重名时序号递增（_克隆02、_克隆03…）；
 * 名称受接口长度限制（32 字符），超长时截断原名并保留后缀，便于识别为克隆体。
 */
export function generateCloneName(sourceName: string | undefined, existingNames: string[] = []): string {
  const base = (sourceName ?? '').trim();
  const usedNames = new Set(existingNames.map(name => (name ?? '').trim()));

  let index = 1;
  let name = '';
  do {
    const suffix = buildCloneNameSuffix(index);
    name = `${base.slice(0, Math.max(NAME_MAX_LENGTH - suffix.length, 0))}${suffix}`;
    index += 1;
  } while (usedNames.has(name) && index <= 999);

  return name;
}

/** 是否为全量：累计目标达到 100%（全量批由系统在末批之后隐含补齐，用户配置到 100% 属等价场景） */
export function isFullBatch(batch: Pick<StrategyBatchForm, 'targetUnit' | 'targetValue'>): boolean {
  return batch.targetUnit === 'percent' && Number(batch.targetValue) === 100;
}

/** 归一化暂停方式：接口返回的非法值降级为 none */
export function normalizePauseMode(mode: string | undefined): PauseMode {
  return PAUSE_MODES.includes(mode as PauseMode) ? (mode as PauseMode) : 'none';
}

/** 解析接口 target（"1" / "25%"）为「数值 + 单位」 */
export function parseTarget(target: string | undefined): { unit: TargetUnit; value?: number } {
  const trimmed = (target ?? '').trim();
  if (trimmed.endsWith('%')) {
    const pct = Number.parseFloat(trimmed.slice(0, -1));
    return { unit: 'percent', value: Number.isFinite(pct) ? pct : undefined };
  }
  if (!trimmed) return { unit: 'count', value: undefined };
  const count = Number.parseInt(trimmed, 10);
  return { unit: 'count', value: Number.isFinite(count) ? count : undefined };
}

/**
 * 列表排序：空间策略在前（updatedAt 倒序，新建策略自然排到首行），
 * 平台内置策略固定底部（保持接口返回顺序）。
 */
export function sortStrategies(list: ReleaseStrategyOutputObj[]): ReleaseStrategyOutputObj[] {
  const spaceStrategies = list.filter(strategy => !strategy.isBuiltin);
  const builtinStrategies = list.filter(strategy => strategy.isBuiltin);
  spaceStrategies.sort((a, b) => (b.updatedAt ?? '').localeCompare(a.updatedAt ?? ''));
  return [...spaceStrategies, ...builtinStrategies];
}

/** 接口批次 → 表单批次行（字段兜底 + 生成稳定标识 + 目标拆为数值与单位） */
export function toBatchForms(batches: BatchOutputObj[] | undefined): StrategyBatchForm[] {
  return (batches ?? []).map(batch => {
    const pauseMode = normalizePauseMode(batch.pauseMode);
    const target = parseTarget(batch.target);
    return {
      key: createBatchKey(),
      targetUnit: target.unit,
      targetValue: target.value,
      pauseMode,
      pauseSeconds: pauseMode === 'duration' ? batch.pauseSeconds : undefined,
    };
  });
}

/** 表单批次行 → 接口提交体（目标拼回 "1" / "25%"；duration 之外的 pauseSeconds 置空） */
export function toBatchInputs(batches: StrategyBatchForm[]): BatchInput[] {
  return batches.map(batch => ({
    target: formatTargetValue(batch),
    pauseMode: batch.pauseMode,
    ...(batch.pauseMode === 'duration' ? { pauseSeconds: Number(batch.pauseSeconds) } : {}),
  }));
}

/** 校验定时暂停秒数：30 ~ 86400，合法返回空串 */
export function validatePauseSeconds(seconds: number | undefined): string {
  const value = Number(seconds);
  if (!Number.isInteger(value) || value < PAUSE_SECONDS_MIN || value > PAUSE_SECONDS_MAX) {
    return `暂停时长需为 ${PAUSE_SECONDS_MIN} ~ ${PAUSE_SECONDS_MAX} 秒之间的整数`;
  }
  return '';
}

/**
 * 校验策略表单，返回字段级错误（值为 i18n key）。
 * 批次错误按批次 key 返回，供抽屉内联展示；纯函数无副作用，便于单测。
 *
 * 跨批次规则：相邻两批单位相同时累计值必须严格递增（「个」与「%」量纲不同，跳过比较）。
 * 全量批不由用户配置：末批确认后由系统隐含补齐「剩余全部实例」，因此不校验末批是否 100%。
 */
export function validateStrategyForm(
  form: StrategyFormData,
  options: StrategyFormValidateOptions = {},
): StrategyFormValidation {
  const batchErrors: Record<string, string> = {};
  const batchErrorDetails: Record<string, BatchErrorDetail> = {};

  const nameError = validateStrategyName(form.name, options);

  const descriptionError =
    form.description.length > DESCRIPTION_MAX_LENGTH ? `策略说明不能超过 ${DESCRIPTION_MAX_LENGTH} 个字符` : '';
  const batchesEmptyError = form.batches.length === 0 ? '请至少配置一个批次' : '';

  form.batches.forEach(batch => {
    const targetError = validateTargetValue(batch.targetValue, batch.targetUnit);
    if (targetError) {
      batchErrors[batch.key] = targetError;
      return;
    }
    if (batch.pauseMode === 'duration') {
      const pauseError = validatePauseSeconds(batch.pauseSeconds);
      if (pauseError) batchErrors[batch.key] = pauseError;
    }
  });

  // 同单位累计值需严格递增
  for (let index = 1; index < form.batches.length; index += 1) {
    const prev = form.batches[index - 1] as StrategyBatchForm;
    const current = form.batches[index] as StrategyBatchForm;
    if (batchErrors[prev.key] || batchErrors[current.key]) continue;
    if (prev.targetUnit !== current.targetUnit) continue;
    if (Number(current.targetValue) <= Number(prev.targetValue)) {
      // 卡片文案受宽度限制不带参数；详情面板补上「上一批累计值」，与设计稿一致
      batchErrors[current.key] = '累计更新数需大于上一批';
      batchErrorDetails[current.key] = {
        key: '累计更新数需大于上一批（{0}）',
        params: [formatBatchTargetText(prev, key => key)],
      };
    }
  }

  return {
    valid: !nameError && !descriptionError && !batchesEmptyError && Object.keys(batchErrors).length === 0,
    nameError: nameError || undefined,
    descriptionError: descriptionError || undefined,
    batchesEmptyError: batchesEmptyError || undefined,
    batchErrors,
    batchErrorDetails,
  };
}

/**
 * 校验策略名称：非空 → 长度 → 与当前空间已有策略重名（合法返回空串）。
 *
 * 重名以前端列表数据为准做同步提示（名称 blur / 提交 / 预览时即时反馈），
 * 后端唯一性校验作为保底：列表可能滞后（他人刚创建），此时由接口错误内联兜底。
 * 与自身原名称相同不算重复（编辑态未改名）；名称按 trim 后精确比较，大小写不同视为不同名称。
 */
export function validateStrategyName(name: string, options: StrategyFormValidateOptions = {}): string {
  const trimmed = (name ?? '').trim();
  if (!trimmed) return '策略名称不能为空';
  if (trimmed.length > NAME_MAX_LENGTH) return `策略名称不能超过 ${NAME_MAX_LENGTH} 个字符`;

  const selfName = (options.selfName ?? '').trim();
  const duplicated = (options.existingNames ?? []).some(existing => {
    const target = (existing ?? '').trim();
    if (!target || target === selfName) return false;
    return target === trimmed;
  });
  return duplicated ? '策略名称在当前空间已存在' : '';
}

/** 校验单个批次的累计值：个为 >0 正整数，百分比为 1% ~ 100% 整数，合法返回空串 */
export function validateTargetValue(value: number | string | undefined, unit: TargetUnit): string {
  const raw = typeof value === 'string' ? value.trim() : value;
  if (raw === undefined || raw === null || raw === '') return '请输入累计更新副本数';

  const num = Number(raw);
  if (!Number.isFinite(num) || !Number.isInteger(num) || num <= 0) {
    return unit === 'percent' ? '百分比需为 1% ~ 100% 的整数' : '请输入大于 0 的正整数';
  }
  if (unit === 'percent' && num > 100) return '百分比需为 1% ~ 100% 的整数';
  return '';
}

/** 克隆名称序号后缀：_克隆01、_克隆02 ……（两位数，超出后自然递增为三位） */
function buildCloneNameSuffix(index: number): string {
  return `_克隆${String(index).padStart(2, '0')}`;
}

/**
 * 每批累计占比（0 ~ 1）。
 *
 * 预览不要求填写实例总数，因此只按配置值示意：百分比直接映射（25% → 0.25），
 * 实例数同口径除以 100（如 3 个 → 0.03），与「方格只表示比例」的说明一致。
 */
function resolvePreviewRatios(batches: StrategyBatchForm[]): number[] {
  return batches.map(batch => Math.min(Math.max(Number(batch.targetValue) || 0, 0), 100) / 100);
}
