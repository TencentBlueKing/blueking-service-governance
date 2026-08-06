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

import { computed, ref, watch } from 'vue';
import type { WatchSource } from 'vue';

import { debounce } from 'lodash-es';

interface FilterOptions {
  filters: WatchSource<unknown> | WatchSource<unknown>[];
  ignoreKeys?: string[];
}
type TableEmptyType = 'empty' | 'error' | 'search';
/**
 * 表格空状态管理 Hook
 *
 * 用于自动判断表格的空状态类型（空数据/搜索无结果/错误）
 *
 * @example
 * const { curExceptionType, setTypeToError, clearErrorType } = useTableEmpty({
 *   filters: searchValue,           // 监听的筛选条件
 *   ignoreKeys: ['dateRange']       // 可选：忽略某些字段的监听
 * });
 *
 * // curExceptionType 会自动返回: 'empty' | 'search' | 'error'
 */
export default function useTableEmpty(opts: FilterOptions) {
  const isSearch = ref(false);
  const isError = ref(false);

  const curExceptionType = computed((): TableEmptyType => {
    if (isError.value) return 'error';
    else if (isSearch.value) return 'search';
    return 'empty';
  });

  /**
   * 设置当前表格状态为错误状态
   * 通常在接口请求失败时调用
   */
  function setTypeToError() {
    isError.value = true;
  }

  /**
   * 清除错误状态
   * 通常在重新请求数据前调用
   */
  function clearErrorType() {
    isError.value = false;
  }

  function deepFindFilter(curFilter: unknown[]): boolean {
    for (const item of curFilter) {
      if (item === null || item === undefined) {
        continue;
      }
      if (typeof item !== 'object' || item instanceof Date) {
        // 如果是基本类型或 Date 类型，检查是否非空
        if (!isValueEmpty(item)) {
          return true;
        }
      } else if (Array.isArray(item)) {
        // 如果是数组，递归检查
        if (deepFindFilter(item)) {
          return true;
        }
      } else {
        // 如果是对象，检查其值
        if (deepFindFilter(Object.values(item as Record<string, unknown>))) {
          return true;
        }
      }
    }
    return false;
  }

  function isValueEmpty(value: unknown) {
    return value === '' || value === null || value === undefined;
  }

  // 使用防抖优化，避免频繁触发导致的性能问题
  const updateSearchState = debounce((val: unknown) => {
    let result = false;
    if (Array.isArray(val)) {
      result = deepFindFilter(val);
    } else if (val !== null && typeof val === 'object') {
      const values = Object.entries(val)
        .filter(([key]) => !opts?.ignoreKeys?.includes(key))
        .map(([, value]) => value);
      result = deepFindFilter(values);
    } else {
      result = !isValueEmpty(val);
    }
    isSearch.value = result;
  }, 300);

  watch(
    opts.filters as WatchSource<unknown>,
    val => {
      updateSearchState(val);
    },
    { deep: true, immediate: true },
  );

  return {
    setTypeToError,
    clearErrorType,
    curExceptionType,
  };
}
