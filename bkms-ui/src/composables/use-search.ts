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

import type { ComputedRef, Ref, ShallowRef } from 'vue';
import { computed, ref, watch } from 'vue';

import type { VxeTableDefines } from '@blueking/vxe-table';
import type { ISearchItem } from 'bkui-vue/lib/search-select/utils';

export interface IInputKey {
  field: string; // 搜索字段，支持多个字段a.b.c
  fuzzy?: boolean; // 是否模糊搜索，只有单选才生效
  id: string;
}
// 特殊配置
export interface IOption {
  ignoreCase?: boolean; // 是否区分大小写，默认 true（不区分）
  rule?: 'and' | 'or'; // 过滤逻辑
}

// 联合类型：要么都不配置，要么都配置
export type IOption2<Table> = IOption2Base | IOption2WithFilters<Table>;

// 基础配置接口
export interface IOption2Base extends IOption {
  tableRef?: never;
}

// 完整筛选功能配置接口
export interface IOption2WithFilters<Table> extends IOption {
  tableRef: Ref<Table>;
}

export interface ISearchValue {
  id: string;
  name: string;
  values: ITableFilterItem[];
}

export interface ISelectKey<T = unknown> extends ISearchItem {
  field: string; // 搜索字段，支持多个字段a.b.c
  fuzzy?: boolean; // 是否模糊搜索，只有单选才生效
  handleFilter?: (item: T, values: ISearchValue['values']) => boolean; // 自定义过滤函数
}

export interface ITableFilterItem {
  checked?: boolean;
  id: string;
  name: string;
}

export interface ITableSearchInputResult<T> {
  searchValue: Ref<string>;
  tableDataMatchSearch: ComputedRef<T[]>;
}

export interface ITableSearchSelectResult<T> {
  filterOptions: Ref<Record<string, ITypeOption['options']>>;
  searchValue: Ref<ISearchValue[]>;
  tableDataMatchSearch: ComputedRef<T[]>;
  filterChangeEvent: (params: VxeTableDefines.FilterChangeEventParams) => void;
}

export interface ITypeOption {
  id: string;
  name: string;
  options: {
    checked?: boolean;
    label: string;
    value: string;
  }[];
}

interface VxeTableHost {
  getVxeTableInstance?: () => {
    setFilter?: (field: string, filters: ITypeOption['options']) => void;
  };
}

/**
 * 搜索 推荐配合SearchInput组件一起使用
 * 表格搜索输入框 Hook
 * 提供基于关键词的表格数据过滤功能，支持多字段搜索、模糊匹配、大小写敏感等特性
 *
 * @template T - 表格数据项的类型
 *
 * @param data - 响应式的表格数据数组
 * @param keys - 响应式的搜索配置数组，定义可搜索的字段和匹配规则
 * @param options - 可选配置项
 * @param options.ignoreCase - 是否区分大小写，默认 true（不区分）
 * @param options.rule - 多条件过滤逻辑，'and' 表示所有条件都满足，'or' 表示任一条件满足
 *
 * @returns 返回搜索相关的响应式数据
 * @returns searchValue - 当前搜索关键词字符串
 * @returns tableDataMatchSearch - 根据搜索条件过滤后的表格数据
 *
 * @example
 * ```typescript
 * interface User {
 *   id: string;
 *   name: string;
 *   email: string;
 *   department: { name: string };
 * }
 *
 * const users = ref<User[]>([...]);
 * const searchKeys = ref<ISelectKey[]>([
 *   { id: 'name', field: 'name', fuzzy: true },
 *   { id: 'email', field: 'email', fuzzy: true },
 *   { id: 'dept', field: 'department.name', fuzzy: true }
 * ]);
 *
 * const { searchValue, tableDataMatchSearch } = useTableSearchInput(
 *   users,
 *   searchKeys,
 *   { ignoreCase: false }
 * );
 * ```
 */
export function useTableSearchInput<T>(
  data: Ref<T[]>,
  keys: Ref<IInputKey[]>,
  options?: IOption,
): ITableSearchInputResult<T> {
  const searchValue = ref('');
  const tableDataMatchSearch = computed(() => {
    if (!searchValue.value) return data.value;

    return data.value.filter(item =>
      keys.value?.some?.(key => {
        const tmpKey = key?.field?.split('.') || [];
        const str = tmpKey.reduce((pre: unknown, currentKey: string) => {
          // 检查 pre 是否为 null、undefined 或不是对象类型
          if (pre == null || typeof pre !== 'object') {
            return pre;
          }
          // 安全地访问对象属性
          return (pre as Record<string, unknown>)[currentKey];
        }, item as unknown);
        if (options?.ignoreCase === false) {
          // 模糊
          return key?.fuzzy ? String(str).includes(searchValue.value) : String(str) === searchValue.value;
        }
        // 模糊
        return key?.fuzzy
          ? String(str).toLowerCase().includes(searchValue.value.toLowerCase())
          : String(str).toLowerCase() === searchValue.value.toLowerCase();
      }),
    );
  });

  return {
    searchValue,
    tableDataMatchSearch,
  };
}

/**
 * 搜索 推荐配合SearchSelect组件一起使用
 * 表格搜索选择器 Hook (增强版)
 * 提供基于搜索配置的表格数据过滤功能，支持多选、模糊搜索、大小写敏感等特性
 * 与 VXE Table 的筛选功能深度集成，支持双向联动
 *
 * @template T - 表格数据项的类型
 * @template S - 搜索配置项的类型，需要包含 id、field、multiple、fuzzy 等属性
 *
 * @param data - 响应式的表格数据数组
 * @param searchData - 响应式的搜索配置数组，定义可搜索的字段和选项
 * @param options - 可选配置项
 * @param options.ignoreCase - 是否区分大小写，默认 true（不区分）
 * @param options.rule - 多条件过滤逻辑，'and' 表示所有条件都满足，'or' 表示任一条件满足
 * @param options.filters - 表格筛选器配置，用于与 VXE Table 的筛选功能联动（与 tableRef 必须同时配置）
 * @param options.tableRef - VXE Table 实例引用，用于双向联动筛选状态（与 filters 必须同时配置）
 *
 * @returns 返回搜索相关的响应式数据和方法
 * @returns searchValue - 当前搜索值数组，每个元素包含搜索字段 id、名称和选中的值列表
 * @returns tableDataMatchSearch - 根据搜索条件过滤后的表格数据
 * @returns filterChangeEvent - 表格筛选器变化事件处理函数，用于同步筛选状态到搜索值
 *
 * @example
 * ```typescript
 * interface User {
 *   id: string;
 *   name: string;
 *   status: string;
 *   department: { name: string };
 * }
 *
 * interface SearchConfig {
 *   id: string;
 *   field: string;
 *   multiple?: boolean;
 *   fuzzy?: boolean;
 *   children?: any[];
 * }
 *
 * const users = ref<User[]>([...]);
 * const searchConfig = shallowRef<SearchConfig[]>([
 *   { id: 'name', field: 'name', fuzzy: true },
 *   { id: 'status', field: 'status', multiple: true }
 * ]);
 *
 * // 基础用法（不使用表格筛选功能）
 * const { searchValue, tableDataMatchSearch } = useTableSearchSelect(
 *   users,
 *   searchConfig,
 *   { rule: 'and', ignoreCase: false }
 * );
 *
 * // 使用表格筛选功能（filters 和 tableRef 必须同时配置）
 * const filters = shallowRef<ITypeOption[]>([...]);
 * const tableRef = ref(null);
 * const { searchValue, tableDataMatchSearch, filterChangeEvent } = useTableSearchSelect(
 *   users,
 *   searchConfig,
 *   {
 *     rule: 'and',
 *     ignoreCase: false,
 *     filters,
 *     tableRef
 *   }
 * );
 * ```
 * ================== 注意 ==================
 * 1、使用 CustomFilter 组件时，CustomFilter 和 TableColumn 需要配置相同的 field、filters
 *
 */
export function useTableSearchSelect<T, Table>(
  data: Ref<T[]>,
  searchData: ShallowRef<ISelectKey<T>[]>,
  options?: IOption2<Table>,
): ITableSearchSelectResult<T> {
  // 搜索值状态，存储用户选择的搜索条件
  const searchValue = ref<ISearchValue[]>([]);
  /** 筛选选项（自动从 searchData 派生并标记选中状态） */
  const filterOptions = computed(() => {
    const result: Record<string, ITypeOption['options']> = {};

    for (const item of searchData.value) {
      if (item.children?.length) {
        result[item.field] = createFilterOptions(item.field);
      }
    }

    return result;
  });

  /** 创建筛选选项（从 searchData 提取并转换为 filter 格式） */
  function createFilterOptions(field: string): ITypeOption['options'] {
    // 从 searchData 中找到对应 id 的配置项
    const data = searchData.value.find(item => item.field === field);

    // 如果没有 children 数据，返回空数组
    if (!data?.children?.length) {
      return [];
    }

    // 获取当前该字段已选中的值，转为 Set 提升查找性能
    const selectedValues = new Set(searchValue.value.find(item => item.id === field)?.values.map(v => v.id) || []);

    // 将 children 数据转换为 filter 选项格式，并标记选中状态
    return data.children.map(child => ({
      value: child.id,
      label: child.name,
      checked: selectedValues.has(child.id),
    }));
  }

  // 计算属性：根据搜索条件过滤表格数据
  const tableDataMatchSearch = computed(() => {
    // 如果没有搜索条件，返回原始数据
    if (!searchValue.value.length) return data.value;

    // 根据配置的过滤规则（and/or）过滤数据
    return data.value.filter(item =>
      searchData.value?.[options?.rule === 'or' ? 'some' : 'every']?.(key => {
        // 查找当前字段对应的搜索值
        const searchValueOfKey = searchValue.value.find(item => item.id === key.id);

        // 如果没有找到对应的搜索值
        if (!searchValueOfKey) return options?.rule !== 'or';

        // 如果配置了自定义过滤函数
        if (key?.handleFilter) {
          return key.handleFilter(item, searchValueOfKey.values);
        }

        // 解析字段路径（支持嵌套字段如 'user.profile.name'）
        const tmpKey = key?.field?.split('.') || [];
        const str = tmpKey.reduce((pre: unknown, currentKey: string) => {
          // 检查 pre 是否为 null、undefined 或不是对象类型
          if (pre == null || typeof pre !== 'object') {
            return pre;
          }
          // 安全地访问对象属性
          return (pre as Record<string, unknown>)[currentKey];
        }, item);

        // 处理多选情况：multiple 为 true 或搜索值数组长度大于 1
        if (key.multiple || searchValueOfKey.values.length > 1) {
          const ids = searchValueOfKey.values.map(item => item.id);
          return ids.includes(String(str));
        }

        // 处理单选情况
        const idStr = searchValueOfKey.values.map(item => item.id).join('');

        // 根据大小写敏感配置进行匹配
        if (options?.ignoreCase === false) {
          // 区分大小写的匹配
          return key?.fuzzy ? String(str).includes(idStr) : String(str) === idStr;
        }

        // 不区分大小写的匹配
        return key?.fuzzy
          ? String(str).toLowerCase().includes(idStr.toLowerCase())
          : String(str).toLowerCase() === idStr.toLowerCase();
      }),
    );
  });

  /**
   * 处理表格筛选器变化事件
   * 当用户在表格列头使用筛选功能时，同步更新搜索值状态
   *
   * @param params - VXE Table 筛选变化事件参数
   * @param params.filterList - 当前激活的筛选条件列表
   */
  function filterChangeEvent({ filterList }: VxeTableDefines.FilterChangeEventParams) {
    // 检查必要的依赖是否存在
    if (!searchData?.value) return;
    if (!Object.keys(filterOptions.value).length) return;

    // 预处理筛选选项配置，构建 field -> { value -> label } 的映射关系
    // 用于快速查找筛选值对应的显示标签
    const fieldOptionsMap = new Map<string, Map<string, string>>();

    for (const option in filterOptions.value) {
      const valueLabelMap = new Map<string, string>();
      filterOptions.value[option].forEach(opt => {
        valueLabelMap.set(opt.value, opt.label);
      });
      fieldOptionsMap.set(option, valueLabelMap);
    }

    // 预处理筛选列表，构建 field -> values 的映射关系
    // 用于快速查找每个字段的筛选值
    const filterListMap = new Map<string, string[]>();
    filterList.forEach(item => {
      filterListMap.set(item.field, item.values);
    });

    // 处理具有子选项的搜索配置项
    searchData.value
      .filter(item => item.children) // 只处理有 children 属性的配置项
      .forEach(data => {
        // 获取当前字段的筛选值
        const values = filterListMap.get(data.field) || [];
        const valueLabelMap = fieldOptionsMap.get(data.field);

        // 生成类型列表，将筛选值转换为搜索值格式
        const typeList = values.map(type => ({
          id: type,
          name: valueLabelMap?.get(type) || type, // 优先使用配置的标签，否则使用原值
        }));

        // 查找当前搜索值中是否已存在该字段的搜索条件
        const typeSearchIndex = searchValue.value.findIndex(item => item.id === data.id);

        if (typeList.length > 0) {
          // 如果有筛选值
          if (typeSearchIndex > -1) {
            // 更新已存在的搜索条件
            searchValue.value[typeSearchIndex].values = typeList;
          } else {
            // 添加新的搜索条件
            searchValue.value.push({
              id: data.id,
              name: data.name,
              values: typeList,
            });
          }
        } else if (typeSearchIndex > -1) {
          // 如果没有筛选值且之前存在搜索条件，则移除该搜索条件
          searchValue.value.splice(typeSearchIndex, 1);
        }
      });
  }

  /**
   * 监听搜索值变化，同步更新表格筛选器状态
   * 实现 SearchSelect 组件与 VXE Table 筛选功能的双向联动
   */
  watch(searchValue, newValue => {
    // 检查表格实例和筛选配置是否存在
    if (!options?.tableRef?.value || !Object.keys(filterOptions.value).length) return;
    const table = (options.tableRef.value as VxeTableHost).getVxeTableInstance?.();
    if (!table) return;

    // 预处理搜索值，构建 id -> values 的映射关系
    // 用于快速查找每个搜索字段对应的选中值
    const searchMap = new Map<string, ISearchValue['values']>();
    newValue.forEach(item => {
      searchMap.set(item.id, item.values);
    });

    for (const option in filterOptions.value) {
      // 调用 VXE Table 的 setFilter 方法更新筛选器
      table.setFilter?.(option, [...filterOptions.value[option]]);
    }
  });

  return {
    searchValue,
    tableDataMatchSearch,
    filterOptions,
    filterChangeEvent,
  };
}
