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

import type { Ref } from 'vue';
import { computed, ref } from 'vue';

import { get } from 'lodash-es';

/**
 * 表格勾选（支持跨页全选）
 * @param data 当前页数据
 * @param path key路径
 * @param total 数据总数
 */
export default function useTableCheckbox<T>(data: Ref<Array<T>>, path: string, total?: Ref<number>) {
  const selections: Ref<T[]> = ref([]);
  const isCrossPageSelection = ref(false);
  const excludedIds = ref<Set<any>>(new Set());

  // 计算实际选中的数量
  const selection = computed(() =>
    isCrossPageSelection.value && total?.value
      ? Array.from({ length: total.value - excludedIds.value.size })
      : selections.value,
  );

  // 是否有选中项
  const hasSelection = computed(() =>
    isCrossPageSelection.value
      ? (total?.value ? total.value - excludedIds.value.size : 0) > 0
      : selections.value.length > 0,
  );

  // 当前页是否全选
  const isCurrentPageAllChecked = computed(() => {
    if (!data.value.length) return false;
    return isCrossPageSelection.value
      ? data.value.every(item => !excludedIds.value.has(get(item, path)))
      : data.value.every(item => selections.value.some(s => get(s, path) === get(item, path)));
  });

  // 表头 checkbox 的 indeterminate 状态
  const isIndeterminate = computed(() => {
    if (!data.value.length) return false;
    const checkedCount = data.value.filter(item => {
      const key = get(item, path);
      return isCrossPageSelection.value
        ? !excludedIds.value.has(key)
        : selections.value.some(s => get(s, path) === key);
    }).length;
    return checkedCount > 0 && checkedCount < data.value.length;
  });

  // 单选切换
  function handleCheckboxChange({ checked, row }: { checked: boolean; row: T }) {
    const key = get(row, path);
    if (isCrossPageSelection.value) {
      if (checked) {
        // 在跨页全选模式下重新勾选项，从排除列表中移除
        excludedIds.value.delete(key);
      } else {
        // 在跨页全选模式下取消勾选某一项，退出跨页全选模式，清除所有选择
        isCrossPageSelection.value = false;
        excludedIds.value.clear();
        selections.value = [];
      }
    } else {
      const index = selections.value.findIndex(item => get(item, path) === key);
      if (checked && index === -1) {
        selections.value.push(row);
      } else if (!checked && index > -1) {
        selections.value.splice(index, 1);
      }
    }
  }

  // 表头：全选/取消全选
  function handleCheckboxAll({ checked }: { checked: boolean }) {
    if (isCrossPageSelection.value) {
      data.value.forEach(item => {
        checked ? excludedIds.value.delete(get(item, path)) : excludedIds.value.add(get(item, path));
      });
    } else if (checked) {
      const existingKeys = new Set(selections.value.map(item => get(item, path)));
      data.value.forEach(item => {
        const key = get(item, path);
        if (!existingKeys.has(key)) selections.value.push(item);
      });
    } else {
      const currentPageKeys = new Set(data.value.map(item => get(item, path)));
      selections.value = selections.value.filter(item => !currentPageKeys.has(get(item, path)));
    }
  }

  // 跨页全选
  const handleSelectAllCrossPage = () => {
    isCrossPageSelection.value = true;
    excludedIds.value.clear();
    selections.value = [];
  };

  // 清除所有选择
  const handleClearSelection = () => {
    isCrossPageSelection.value = false;
    excludedIds.value.clear();
    selections.value = [];
  };

  return {
    selections,
    selection,
    hasSelection,
    isCrossPageSelection,
    excludedIds,
    isCurrentPageAllChecked,
    isIndeterminate,
    handleCheckboxChange,
    handleCheckboxAll,
    handleSelectAllCrossPage,
    handleClearSelection,
  };
}
