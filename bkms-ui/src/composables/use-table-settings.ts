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

import { computed, toValue } from 'vue';
import type { ComputedRef, Ref } from 'vue';

import { useUiStore } from '~/stores/ui';

import type { ISettings } from '@blueking/table';

/** 表格列设置的个性化差异：相对默认勾选列，hidden=默认勾选但被取消，shown=默认未勾选但被勾选；size=用户选择的行高。 */
interface TableSettingDiff {
  hidden: string[];
  shown: string[];
  size?: ISettings['size'];
}

interface UseTableSettingsOptions {
  /** 默认勾选列（表格列设置的基线） */
  defaultChecked: string[];
  /** 不可隐藏列：静态传入，强制并入 checked，不落盘 */
  disabled?: string[];
}

/** 偏好命名空间前缀，与其他 UI 偏好在 ui store 中区分。 */
const TABLE_PREFERENCE_PREFIX = 'table:';

/**
 * 表格列设置 hook：读写指定表格的列勾选状态，仅持久化与默认勾选的差异到 ui store。
 *
 * @param tableId 表格唯一标识，由调用方保证唯一且稳定：持久化按 tableId 隔离，
 *   同一表格使用相同 id 才能命中同一份偏好；不同表格（含 v-for 多实例）须传不同 id 避免串数据。
 *   支持响应式（如随 envName 变化切换到独立列设置）。
 *
 * 注意：只持久化差异（hidden/shown），size 不存不回写，保持表格默认行高密度。
 */
export function useTableSettings(
  tableId: ComputedRef<string> | Ref<string> | string,
  options: UseTableSettingsOptions,
) {
  const uiStore = useUiStore();
  const { defaultChecked, disabled = [] } = options;

  // 当前个性化差异；无记录或缓存被污染（非 { hidden, shown } 形状）时视为零差异。
  const diff = computed<TableSettingDiff>(() => {
    const stored = uiStore.preferences[getPreferenceKey(toValue(tableId))];
    const storedDiff = stored as TableSettingDiff | undefined;
    if (storedDiff && Array.isArray(storedDiff.hidden) && Array.isArray(storedDiff.shown)) {
      return storedDiff;
    }
    return { hidden: [], shown: [] };
  });

  // 生效的列设置：默认勾选扣除 hidden、补上 shown，并强制并入不可隐藏列。
  // size：'small' 是组件库默认行高，等价于未个性化，不回写；仅回写 medium/mini，脏数据被忽略。
  const settings = computed(() => {
    const checked = [
      ...new Set([
        ...defaultChecked.filter(field => !diff.value.hidden.includes(field)),
        ...diff.value.shown,
        ...disabled,
      ]),
    ];
    const size = diff.value.size;
    const validSize = size === 'medium' || size === 'mini' ? size : undefined;

    return { checked, disabled, ...(validSize ? { size: validSize } : {}) };
  });

  /** 同步列设置勾选状态与行高：计算相对默认勾选的差异后写入 store，无任何差异时自动清除该表记录。 */
  function handleSettingChange(data: ISettings) {
    const current = diff.value;
    const next: TableSettingDiff = { hidden: current.hidden, shown: current.shown, size: current.size };
    // checked 通常随 setting-change 携带；缺失时只同步行高，不改变列勾选差异。
    if (data.checked) {
      const nextChecked = data.checked;
      next.hidden = defaultChecked.filter(field => !nextChecked.includes(field) && !disabled.includes(field));
      next.shown = nextChecked.filter(field => !defaultChecked.includes(field) && !disabled.includes(field));
    }
    // 'small' 是组件库默认行高，面板关闭时恒会 emit，视为未个性化不写入；
    // 否则用户只打开一次面板（未碰行高）也会被持久化，且破坏"无差异自动清除"。
    if (data.size && data.size !== 'small') {
      next.size = data.size;
    }
    const hasDiff = next.hidden.length > 0 || next.shown.length > 0 || next.size !== undefined;
    uiStore.setPreference(getPreferenceKey(toValue(tableId)), hasDiff ? next : null);
  }

  return { settings, handleSettingChange };
}

function getPreferenceKey(tableId: string) {
  return `${TABLE_PREFERENCE_PREFIX}${tableId}`;
}
