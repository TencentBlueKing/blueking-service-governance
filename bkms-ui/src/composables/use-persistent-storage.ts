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

/**
 * 统一持久化存储 Hooks
 */

import { computed, reactive, ref, watch } from 'vue';

import { useLocalStorage } from '@vueuse/core';
import { useRoute } from 'vue-router';

/**
 * 分页状态接口
 * 定义表格分页所需的基本属性
 */
export interface IPaginationState {
  /** 数据总数 */
  count: number;
  /** 当前页码 */
  current: number;
  /** 每页条数 */
  limit: number;
}

/**
 * 存储配置选项接口
 * 用于控制数据的存储方式和范围
 */
export interface IStorageOptions {
  /** 是否全局存储（所有页面共享），默认 false */
  global?: boolean;
  /** 自定义存储 key，默认使用当前路由路径 */
  storageKey?: string;
}

/**
 *
 * 使用示例：
 * // 导航展开状态
 * const { useNavigation } = usePersistentStorage();
 * const nav = useNavigation(true, true);
 *
 * // 表格分页
 * const { usePagination } = usePersistentStorage();
 * const pagination = usePagination({ defaultLimit: 20 });
 *
 * // 自定义存储
 * const { useStorage } = usePersistentStorage();
 * const customState = useStorage('my-config', defaultValue, { global: true });
 */
export function usePersistentStorage() {
  const route = useRoute();

  /**
   * 通用存储方法
   * 基于 VueUse 的 useLocalStorage，提供响应式的 localStorage 存储
   *
   * @param key 存储的配置项名称（会作为 storage key 的一部分）
   * @param defaultValue 默认值（首次访问时使用）
   * @param options 配置选项
   * @param options.global 是否全局存储（true: 所有页面共享，false: 页面独立）
   * @param options.storageKey 自定义存储key（不指定则使用当前路由路径）
   * @returns 响应式的存储引用
   */
  const useStorage = <T>(key: string, defaultValue: T, options: IStorageOptions = {}) => {
    const { global = false, storageKey } = options;

    // 生成完整的存储key，格式: global_${key} 或 ${routePath}_${key}
    const fullStorageKey = global ? `global_${key}` : `${storageKey || route.path}_${key}`;
    return useLocalStorage(fullStorageKey, defaultValue);
  };

  /**
   * 导航展开/收起状态持久化
   * 用于保存侧边栏导航的展开状态，默认全局存储（所有页面共享）
   * @param defaultOpen 默认展开配置，{ value: 默认是否展开, force: 是否跳过持久化 }
   * @param global 是否全局存储，默认 true（所有页面共享同一个导航状态）
   * @returns isOpen 响应式的展开状态
   * @returns toggle 切换展开/收起状态的方法
   * @returns setOpen 直接设置展开状态的方法
   */
  const useNavigation = (defaultOpen: { force?: boolean; value?: boolean } = { value: true }, global = true) => {
    const defaultValue = defaultOpen.value ?? true;
    const isOpen = defaultOpen.force ? ref(defaultValue) : useStorage('navigation-open', defaultValue, { global });

    return {
      /** 响应式的展开状态 */
      isOpen,
      /** 切换展开/收起状态 */
      toggle: () => {
        isOpen.value = !isOpen.value;
      },
      /** 设置展开状态 */
      setOpen: (open: boolean) => {
        isOpen.value = open;
      },
    };
  };

  /**
   * 表格分页状态持久化
   * 用于保存表格的分页参数（页码、每页条数、总数），默认页面级存储
   *
   * @param options.defaultCurrent 默认当前页码，默认值 1
   * @param options.defaultLimit 默认每页条数，默认值 20
   * @param options.global 是否全局存储，默认 false（页面独立）
   * @param options.storageKey 自定义存储key，默认使用当前路由路径
   * @returns 包含分页状态和操作方法的对象
   */
  const usePagination = (
    options: {
      defaultCurrent?: number;
      defaultLimit?: number;
      global?: boolean;
      storageKey?: string;
    } = {},
  ) => {
    const { defaultCurrent = 1, defaultLimit = 20, global = false, storageKey } = options;

    // 存储分页状态到 localStorage
    const storedPagination = useStorage<Partial<IPaginationState>>(
      'table-pagination',
      {
        count: 0,
        current: defaultCurrent,
        limit: defaultLimit,
      },
      { global, storageKey },
    );

    // 创建响应式分页状态
    const pagination = reactive<IPaginationState>({
      count: storedPagination.value.count || 0,
      current: storedPagination.value.current || defaultCurrent,
      limit: storedPagination.value.limit || defaultLimit,
    });

    // 监听分页状态变化，自动同步到 localStorage
    watch(
      () => ({ count: pagination.count, current: pagination.current, limit: pagination.limit }),
      newValue => {
        storedPagination.value = newValue;
      },
      { deep: true },
    );

    /**
     * 更新当前页码
     * @param current 新的页码值
     */
    const updateCurrent = (current: number) => {
      if (current !== pagination.current) {
        pagination.current = current;
      }
    };

    /**
     * 更新每页条数
     * @param limit 新的每页条数
     */
    const updateLimit = (limit: number) => {
      if (limit !== pagination.limit) {
        pagination.limit = limit;
        pagination.current = 1; // 重置到第一页
      }
    };

    /**
     * 更新数据总数
     * @param count 新的数据总数
     */
    const updateCount = (count: number) => {
      pagination.count = count;
    };

    /**
     * 重置到第一页
     */
    const resetToFirstPage = () => {
      updateCurrent(1);
    };

    const tablePagination = computed(() => ({
      count: pagination.count,
      current: pagination.current,
      limit: pagination.limit,
      showTotalCount: true,
    }));

    return {
      /** 响应式分页状态对象 */
      pagination,
      /** 重置到第一页 */
      resetToFirstPage,
      /** 用于 Table 组件的分页配置 */
      tablePagination,
      /** 更新数据总数 */
      updateCount,
      /** 更新当前页码 */
      updateCurrent,
      /** 更新每页条数（会自动重置到第一页） */
      updateLimit,
    };
  };

  return {
    /** 导航状态持久化 */
    useNavigation,
    /** 表格分页状态持久化 */
    usePagination,
    /** 通用存储方法，用于其他自定义配置 */
    useStorage,
  };
}

export default usePersistentStorage;
