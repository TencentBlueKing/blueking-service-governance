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

import { resolvePauseText, translateMessage } from '~/common/batch-release';
import { type BatchPauseMode, BATCH_PAUSE_MODE } from '~/common/enums/deploy';
import {
  buildSquaresByRatio,
  formatOptionTarget,
  parseBatchTarget,
  shouldAppendAutoFullBatch,
} from '~/pages/application/detail/deploy/release-preview';

import type { BatchInput, BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';
import type { ProgressSquareState } from '~/pages/application/detail/deploy/release-preview';
import type { TargetTexts } from '~/pages/application/detail/deploy/release-preview';

/** 详情面板的带参错误文案：key 为 i18n 模板，params 依序插值 */
export interface BatchErrorDetail {
  key: string;
  params: (number | string)[];
}

/** 批次完成后的暂停方式：none 不暂停 / duration 定时暂停 / permanent 永久暂停（枚举见 ~/common/enums/deploy） */
export type PauseMode = BatchPauseMode;

/** 进度方块单格状态：已更新 / 本批新增 / 待更新（与部署预览共用同一类型） */
export type PreviewSquareState = ProgressSquareState;

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
 * 校验错误的 i18n 描述：key 为固定模板，params 依序插值。
 * 不能把常量拼进 key（如 `不能超过 ${NAME_MAX_LENGTH} 个字符`）：
 * 那样 key 会随常量变成一条新词条，非中文环境直接回退成 key 原文，且会在 locale 里留下死词条。
 */
export interface StrategyFormError {
  key: string;
  params: (number | string)[];
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

/** 表单校验结果：错误值为 i18n 模板 + 插值参数（项目约定中文 key 直译） */
export interface StrategyFormValidation {
  /** 批次行级错误的详情文案（额外带上被比较批次的累计值），缺省时回落到 batchErrors */
  batchErrorDetails: Record<string, BatchErrorDetail>;
  /**
   * 批次行级错误（简短文案），key 为批次的稳定标识。
   * 批次卡片第二行直接展示，宽度有限，只带批次序号不带累计值。
   */
  batchErrors: Record<string, BatchErrorDetail>;
  /** 表单级错误：批次列表为空 */
  batchesEmptyError?: StrategyFormError;
  /**
   * 批次暂停时长错误（仅 duration 模式），key 为批次的稳定标识。
   * 详情面板据此把文案内联到「暂停」输入下方，而不是挂在「目标资源」行上；
   * 卡片仍从 batchErrors 取合并后的错误展示。
   */
  batchPauseErrors: Record<string, BatchErrorDetail>;
  descriptionError?: StrategyFormError;
  nameError?: StrategyFormError;
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
 * 实现统一收敛在 ~/common/batch-release.translateMessage，与运行态进度共用一套插值逻辑。
 */
export function translate(t: TranslateFn, key: string, params: Array<number | string>): string {
  return translateMessage(t, key, params);
}

export const NAME_MAX_LENGTH = 32;
export const DESCRIPTION_MAX_LENGTH = 100;
export const PAUSE_SECONDS_MIN = 30;
export const PAUSE_SECONDS_MAX = 86_400;
/** 目标资源单位为百分比时的上限 */
export const PERCENT_TARGET_MAX = 100;

/**
 * 后端「策略名称冲突」错误文案匹配（bkms-server ErrStrategyNameConflict：
 * "release strategy name already exists" / 内置策略占用同链路文案）。
 * 只精确匹配已存在类表述，供提交失败时决定是否内联到「策略名称」项；
 * 不能宽松匹配 name，否则 envName invalid 等其他含 name 的报错会被误归类。
 */
export const STRATEGY_NAME_CONFLICT_PATTERN = /name already exists|名称[^\n]*已存在|已存在同名/;
/** 切到「定时暂停」时的默认时长，与设计稿一致 */
export const DEFAULT_PAUSE_SECONDS = 30;

const PAUSE_MODES: PauseMode[] = Object.values(BATCH_PAUSE_MODE);

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
 * 着色规则实现在 release-preview 的 buildSquaresByRatio，与部署预览共用。
 */
export function buildPreviewSquares(
  row: Pick<StrategyPreviewRow, 'accumulatedRatio' | 'prevRatio'>,
  count = PREVIEW_SQUARE_COUNT,
): PreviewSquareState[] {
  return buildSquaresByRatio(row, count);
}

/**
 * 已校验的表单批次 → 预览弹窗时间线行。
 *
 * 只展示用户配置的批次：末批未配置到 100% 时系统仍会在其完成后自动更新剩余实例
 * （后端 UpdatingPrimary 阶段），但预览不再补齐全量批，改为弹窗底部提示
 * （见 shouldShowRemainingAutoUpdateTip）。
 */
export function buildStrategyPreviewRows(batches: StrategyBatchForm[], t: TranslateFn): StrategyPreviewRow[] {
  const ratios = resolvePreviewRatios(batches);
  let prevRatio = 0;

  return batches.map((batch, index) => {
    const accumulatedRatio = ratios[index] ?? 1;
    const row: StrategyPreviewRow = {
      key: batch.key,
      label: translate(t, '第 {0} 批', [index + 1]),
      targetText: formatTargetValue(batch),
      countValue: formatBatchTargetText(batch, t),
      countLabel: index === 0 ? t('首批') : t('累计'),
      prevRatio,
      accumulatedRatio,
      isFull: isFullBatch(batch),
      hasPause: batch.pauseMode !== BATCH_PAUSE_MODE.NONE,
      // 预览时间线的暂停标签与列表页同形态（「·」分隔）
      pauseText: resolvePauseText(batch.pauseMode, batch.pauseSeconds, t, 'detail'),
      pauseTheme: batch.pauseMode === BATCH_PAUSE_MODE.PERMANENT ? 'warn' : 'brand',
    };
    prevRatio = accumulatedRatio;
    return row;
  });
}

/**
 * 预览摘要：批次数 / 暂停次数 / 首批数量 / 流量批次（本期无流量批次，固定 0）。
 * 单位文案随语言切换（count → `个` 在英文下为空串），因此由调用方传入翻译函数。
 */
export function buildStrategyPreviewSummary(batches: StrategyBatchForm[], t: TranslateFn): StrategyPreviewSummary {
  const firstBatch = batches[0];
  const firstValue = firstBatch?.targetValue;
  const hasFirst = firstValue !== undefined && firstValue !== null && !Number.isNaN(Number(firstValue));
  return {
    batchCount: batches.length,
    pauseCount: batches.filter(batch => batch.pauseMode !== BATCH_PAUSE_MODE.NONE).length,
    firstCountValue: hasFirst ? String(Number(firstValue)) : '0',
    firstCountUnit: firstBatch?.targetUnit === 'percent' ? '%' : t('个'),
    trafficBatchCount: 0,
  };
}

/**
 * 计算批次分段条各段宽度占比（0 ~ 1）。
 * 每段按本批新增占比绘制：将各批累计 target 经 parseBatchTarget 解析为数值
 * （百分比去 % 并收敛到 0 ~ 100、正整数原样），取相邻差值为段宽，差值过小的批次兜底 0.5 保证可见。
 * 注意：仅作可视化示意，混用「正整数实例数」与「百分比」两种量纲时差值不具精确占比语义。
 */
export function calcBatchSegmentRatios(batches: BatchOutputObj[] | undefined): number[] {
  const values = (batches ?? []).map(batch => {
    const parsed = parseBatchTarget(batch.target);
    if (parsed.unit === 'percent') {
      return parsed.value === undefined ? 100 : Math.min(Math.max(parsed.value, 0), 100);
    }
    return parsed.value ?? 0;
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
    pauseMode: BATCH_PAUSE_MODE.PERMANENT,
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
  const parsed = parseBatchTarget(batch.target);
  // 仅「显式配置的百分比 ≥ 100」追加（全量）后缀；空 target 保持走全量 token 兜底
  const isFull = parsed.unit === 'percent' && parsed.isFull;
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
 * 文案映射统一收敛在 ~/common/batch-release，卡片与列表页只是形态（标点）不同。
 */
export function formatBatchPauseText(
  batch: Pick<StrategyBatchForm, 'pauseMode' | 'pauseSeconds'>,
  t: TranslateFn,
): string {
  return resolvePauseText(batch.pauseMode, batch.pauseSeconds, t, 'card');
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
  return resolvePauseText(batch.pauseMode, batch.pauseSeconds, t, 'detail');
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

/** 是否为全量：累计目标达到 100%（判定与部署预览的 isFullTarget 口径一致，均为百分比 ≥ 100） */
export function isFullBatch(batch: Pick<StrategyBatchForm, 'targetUnit' | 'targetValue'>): boolean {
  return parseBatchTarget(formatTargetValue(batch)).isFull;
}

/** 归一化暂停方式：接口返回的非法值降级为 none */
export function normalizePauseMode(mode: string | undefined): PauseMode {
  return PAUSE_MODES.includes(mode as PauseMode) ? (mode as PauseMode) : BATCH_PAUSE_MODE.NONE;
}

/**
 * 解析接口 target（"1" / "25%"）为「数值 + 单位」。
 * 与部署侧共用 parseBatchTarget；差异仅在空 target 语义：
 * 表单回填时空值表示「待填写」（count / undefined），展示与全量判断才按契约兜底 100%。
 */
export function parseTarget(target: string | undefined): { unit: TargetUnit; value?: number } {
  const parsed = parseBatchTarget(target);
  return { unit: parsed.unit, value: parsed.text ? parsed.value : undefined };
}

/**
 * 是否需要展示「剩余实例自动更新」提示：末批未配置到 100% 时，
 * 系统会在最后一批完成后自动更新剩余实例（与部署侧发布预览共用同一判断）。
 */
export function shouldShowRemainingAutoUpdateTip(batches: StrategyBatchForm[]): boolean {
  return shouldAppendAutoFullBatch(batches.map(formatTargetValue));
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
      pauseSeconds: pauseMode === BATCH_PAUSE_MODE.DURATION ? batch.pauseSeconds : undefined,
    };
  });
}

/** 表单批次行 → 接口提交体（目标拼回 "1" / "25%"；duration 之外的 pauseSeconds 置空） */
export function toBatchInputs(batches: StrategyBatchForm[]): BatchInput[] {
  return batches.map(batch => ({
    target: formatTargetValue(batch),
    pauseMode: batch.pauseMode,
    ...(batch.pauseMode === BATCH_PAUSE_MODE.DURATION ? { pauseSeconds: Number(batch.pauseSeconds) } : {}),
  }));
}

/**
 * 校验定时暂停秒数：PAUSE_SECONDS_MIN ~ PAUSE_SECONDS_MAX，合法返回 null。
 * 文案用固定 key + params 插值，不能把常量拼进 key：常量改动后 key 与 locale 对不上，
 * 非中文环境会直接回退成 key 原文。
 */
export function validatePauseSeconds(seconds: number | undefined): BatchErrorDetail | null {
  const value = Number(seconds);
  if (!Number.isInteger(value) || value < PAUSE_SECONDS_MIN || value > PAUSE_SECONDS_MAX) {
    return { key: '暂停时长需为 {0} ~ {1} 秒之间的整数', params: [PAUSE_SECONDS_MIN, PAUSE_SECONDS_MAX] };
  }
  return null;
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
  const batchErrors: Record<string, BatchErrorDetail> = {};
  const batchErrorDetails: Record<string, BatchErrorDetail> = {};
  const batchPauseErrors: Record<string, BatchErrorDetail> = {};

  const nameError = validateStrategyName(form.name, options);

  const descriptionError: StrategyFormError | undefined =
    form.description.length > DESCRIPTION_MAX_LENGTH
      ? { key: '策略说明不能超过 {0} 个字符', params: [DESCRIPTION_MAX_LENGTH] }
      : undefined;
  const batchesEmptyError: StrategyFormError | undefined =
    form.batches.length === 0 ? { key: '请至少配置一个批次', params: [] } : undefined;

  form.batches.forEach(batch => {
    const targetError = validateTargetValue(batch.targetValue, batch.targetUnit);
    if (targetError) {
      batchErrors[batch.key] = { key: targetError, params: [] };
      return;
    }
    if (batch.pauseMode === 'duration') {
      const pauseError = validatePauseSeconds(batch.pauseSeconds);
      // 暂停时长错误只进 batchPauseErrors：卡片宽度有限，长文案会挤压左侧批次列表，
      // 文案统一由详情面板的 FormItem 在输入下方内联展示。
      if (pauseError) batchPauseErrors[batch.key] = pauseError;
    }
  });

  // 同单位累计值需严格递增：按单位分别记录「上一次出现的同单位值与批次序号」再比较。
  // 只比较相邻同单位对会漏判混合单位配置（如 [5 个, 50%, 3 个]）；
  // 跨单位比较跳过——「个」与「%」是两种量纲，无可比性，绝对关系由后端 target 校验兜底。
  const lastSameUnit: Partial<Record<TargetUnit, { index: number; text: string; value: number }>> = {};
  form.batches.forEach((current, index) => {
    if (batchErrors[current.key]) return;
    const value = Number(current.targetValue);
    const last = lastSameUnit[current.targetUnit];
    if (last && value <= last.value) {
      // 被比较的批次不一定是「上一批」（混合单位时会跨批比较），文案用批次序号指明，避免误导。
      // 卡片受宽度限制只带序号；详情面板再补上被比较批次的累计值，与设计稿一致。
      const batchLabel = String(last.index + 1);
      batchErrors[current.key] = { key: '累计更新数需大于第 {0} 批', params: [batchLabel] };
      batchErrorDetails[current.key] = {
        key: '累计更新数需大于第 {0} 批（{1}）',
        params: [batchLabel, last.text],
      };
      return;
    }
    if (Number.isFinite(value)) {
      lastSameUnit[current.targetUnit] = { index, text: formatBatchTargetText(current, key => key), value };
    }
  });

  return {
    valid:
      !nameError &&
      !descriptionError &&
      !batchesEmptyError &&
      Object.keys(batchErrors).length === 0 &&
      Object.keys(batchPauseErrors).length === 0,
    nameError: nameError ?? undefined,
    descriptionError,
    batchesEmptyError,
    batchErrors,
    batchErrorDetails,
    batchPauseErrors,
  };
}

/**
 * 校验策略名称：非空 → 长度 → 与当前空间已有策略重名（合法返回 null）。
 *
 * 重名以前端列表数据为准做同步提示（名称 blur / 提交 / 预览时即时反馈），
 * 后端唯一性校验作为保底：列表可能滞后（他人刚创建），此时由接口错误内联兜底。
 * 与自身原名称相同不算重复（编辑态未改名）；名称按 trim 后精确比较，大小写不同视为不同名称。
 */
export function validateStrategyName(
  name: string,
  options: StrategyFormValidateOptions = {},
): null | StrategyFormError {
  const trimmed = (name ?? '').trim();
  if (!trimmed) return { key: '策略名称不能为空', params: [] };
  if (trimmed.length > NAME_MAX_LENGTH) {
    return { key: '策略名称不能超过 {0} 个字符', params: [NAME_MAX_LENGTH] };
  }

  const selfName = (options.selfName ?? '').trim();
  const duplicated = (options.existingNames ?? []).some(existing => {
    const target = (existing ?? '').trim();
    if (!target || target === selfName) return false;
    return target === trimmed;
  });
  return duplicated ? { key: '策略名称在当前空间已存在', params: [] } : null;
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
