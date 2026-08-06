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

import { type Ref, computed } from 'vue';

import { mapKeys } from '~/common/util';

import type { ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';

/**
 * 筛选选项接口
 */
export interface IFilters {
  /** 是否选中 */
  checked?: boolean;
  /** 显示标签 */
  label: string;
  /** 选项值 */
  value: string;
}

/**
 * 搜索筛选 Hook
 *
 * ⚠️ 重要：TableColumn 的 field 属性必须与 filterKeys 中的字段 id 完全一致
 *
 * @param searchSelectData - 搜索选择器的配置数据
 * @param searchValue - 当前选中的搜索值（响应式）
 * @param filterKeys - 需要生成 filterOptions 的字段 id 列表（使用 as const 获得类型提示）
 */
export default function useSearchFilter<T extends readonly string[]>(
  searchSelectData: Ref<ISearchItem[]>,
  searchValue: Ref<ISearchValue[]>,
  filterKeys: T,
) {
  /** 筛选选项（自动从 searchSelectData 派生并标记选中状态） */
  const filterOptions = computed(() => {
    const result = {} as Record<T[number], IFilters[]>;
    filterKeys.forEach(id => {
      result[id as T[number]] = createFilterOptions(id);
    });
    return result;
  });
  /** 创建筛选选项（从 searchSelectData 提取并转换为 filter 格式） */
  function createFilterOptions(id: string): IFilters[] {
    // 从 searchSelectData 中找到对应 id 的配置项
    const data = searchSelectData.value.find(item => item.id === id);
    // 获取当前该字段已选中的值
    const curItemSearchData = searchValue.value.find(item => item.id === id)?.values || [];

    // 如果没有 children 数据，返回空数组
    if (!data?.children || !Array.isArray(data.children)) {
      return [];
    }

    // 将 children 数据转换为 filter 选项格式，并标记选中状态
    return mapKeys(data.children, {
      label: 'name',
      value: 'id',
    }).map(item => {
      const isSelected = curItemSearchData.some(curItem => curItem.id === item.value);
      return {
        ...item,
        checked: isSelected,
      } as IFilters;
    });
  }

  /** 处理筛选条件变化（将 filter 变化同步到 searchValue） */
  function handleFilterChange({ field, values }: { field: string; values: string[] }) {
    // 从 searchSelectData 中找到对应 field 的配置项
    const searchItem = searchSelectData.value.find(item => item.id === field);
    if (!searchItem) return;

    // 查找 searchValue 中是否已存在该 id 的数据
    const existingIndex = searchValue.value.findIndex(item => item.id === searchItem.id);

    // 如果 values 为空，删除该项（用户取消了所有选择）
    if (values.length === 0) {
      if (existingIndex > -1) {
        searchValue.value.splice(existingIndex, 1);
      }
      return;
    }

    // 将 values 转换为 ISearchValue 格式
    const valuesList = values.map(value => {
      // 从 children 中找到对应的 name（用于显示）
      const child = searchItem.children?.find((item: any) => item.id === value);
      return {
        id: value,
        name: child?.name || value,
      };
    });

    // 构造新的 ISearchValue 对象
    const newSearchValue: ISearchValue = {
      id: searchItem.id,
      name: searchItem.name,
      values: valuesList,
    };

    // 如果已存在，替换；否则新增
    if (existingIndex > -1) {
      searchValue.value[existingIndex] = newSearchValue;
    } else {
      searchValue.value.push(newSearchValue);
    }
  }

  return {
    filterOptions,
    handleFilterChange,
  };
}
