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

import { BATCH_PAUSE_MODE } from './enums/deploy';

/**
 * 暂停方式文案的展示形态：
 * - detail：策略列表 / 明细 Popover 的一行（「永久暂停 · 手动继续」）
 * - card：批次卡片第二行（「永久暂停，手动继续」，与设计稿的逗号分隔一致）
 * - step：灰度进度步骤条的副文本（「永久暂停 · 手动继续」/「不停 · 自动继续」）
 *
 * 三种形态的差异仅来自设计稿的标点与「自动继续」后缀，映射关系集中在此，
 * 避免「同一个 pauseMode 在不同页面各写一份 if」后只改一处导致文案漂移。
 */
export type PauseTextVariant = 'card' | 'detail' | 'step';

type TranslateFn = (key: string, params?: (number | string)[]) => string;

/** duration 模式未配置时长时的兜底秒数，与策略表单默认值一致 */
const DEFAULT_PAUSE_SECONDS = 30;

interface PauseTextCopy {
  /** 定时暂停后自动继续；`{0}` 为秒数 */
  duration: string;
  /** 不暂停 */
  none: string;
  /** 永久暂停，等待人工继续 */
  permanent: string;
}

const PAUSE_TEXT_COPY: Record<PauseTextVariant, PauseTextCopy> = {
  card: {
    permanent: '永久暂停，手动继续',
    duration: '暂停 {0} 秒后自动继续',
    none: '不停',
  },
  detail: {
    permanent: '永久暂停 · 手动继续',
    duration: '暂停 {0} 秒后自动继续',
    none: '不停',
  },
  step: {
    permanent: '永久暂停 · 手动继续',
    duration: '暂停 {0} 秒 · 自动继续',
    none: '不停 · 自动继续',
  },
};

/** 暂停方式文案描述：key 为 i18n 模板，params 依序插值（供组件层自行翻译） */
export interface PauseTextDescriptor {
  key: string;
  params: (number | string)[];
}

/**
 * 把 i18n 模板的 `{n}` 占位符按参数补齐。
 * dev 下 vue-i18n 运行时编译不可用时会原样返回模板串，因此渲染前需要再补一次；
 * 译文已插值时不会命中，属幂等操作。
 */
export function interpolate(text: string, params: (number | string)[] = []): string {
  return params.reduce<string>((out, param, index) => out.split(`{${index}}`).join(String(param)), text);
}

/** 直接将暂停方式渲染为字符串（无需 VNode 的场景，如卡片 / Popover 行） */
export function resolvePauseText(
  pauseMode: string | undefined,
  pauseSeconds: number | undefined,
  t: TranslateFn,
  variant: PauseTextVariant,
): string {
  const { key, params } = resolvePauseTextDescriptor(pauseMode, pauseSeconds, variant);
  return translateMessage(t, key, params);
}

/**
 * 暂停方式文案（策略配置与运行态进度共用）。
 *
 * @param pauseMode 批次暂停方式；非法值降级为「不停」
 * @param pauseSeconds 仅 duration 模式生效，缺省 / 非法值按 30 秒兜底
 * @param variant 展示形态，见 PauseTextVariant
 */
export function resolvePauseTextDescriptor(
  pauseMode: string | undefined,
  pauseSeconds: number | undefined,
  variant: PauseTextVariant,
): PauseTextDescriptor {
  const copy = PAUSE_TEXT_COPY[variant];
  if (pauseMode === BATCH_PAUSE_MODE.PERMANENT) return { key: copy.permanent, params: [] };
  if (pauseMode === BATCH_PAUSE_MODE.DURATION) {
    const seconds = Number(pauseSeconds) > 0 ? Number(pauseSeconds) : DEFAULT_PAUSE_SECONDS;
    return { key: copy.duration, params: [seconds] };
  }
  return { key: copy.none, params: [] };
}

/** 带参数翻译：key 缺失时也不会把 `{0}` 原样展示给用户 */
export function translateMessage(t: TranslateFn, key: string, params: (number | string)[] = []): string {
  return interpolate(t(key, params), params);
}
