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

import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
import { useSpaceStore } from '~/stores/space';

import { filterStrategiesByName, sortStrategies } from './utils';

import type { ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

const PAGE_SIZE = 10;

/**
 * 发布策略列表状态管理。
 * 接口一次性返回全量策略（不分页），此处完成前端过滤、排序、分页切片。
 */
export default function useReleaseStrategies() {
  const spaceStore = useSpaceStore();

  const isLoading = ref(false);
  const isError = ref(false);
  const keyword = ref('');
  const rawStrategies = ref<ReleaseStrategyOutputObj[]>([]);

  const pagination = ref({
    count: 0,
    limit: PAGE_SIZE,
    current: 1,
    // 接口一次返回全量数据，前端完成过滤/排序/切片，Table 不再自行分页
    remote: true,
  });

  /** 过滤 + 排序后的完整列表（空间策略按更新时间倒序在前，内置策略固定底部） */
  const filteredStrategies = computed(() => {
    const sorted = sortStrategies(rawStrategies.value);
    return filterStrategiesByName(sorted, keyword.value);
  });

  /** 当前页数据（前端切片） */
  const strategies = computed(() => {
    const { current, limit } = pagination.value;
    const start = (current - 1) * limit;
    return filteredStrategies.value.slice(start, start + limit);
  });

  /** 同步总数，并在删除/搜索后修正越界页码 */
  watch(
    filteredStrategies,
    list => {
      pagination.value.count = list.length;
      const maxPage = Math.max(1, Math.ceil(list.length / pagination.value.limit));
      if (pagination.value.current > maxPage) pagination.value.current = maxPage;
    },
    { immediate: true },
  );

  /** 拉取策略列表；失败保留上次数据并标记错误态 */
  async function fetchList() {
    isLoading.value = true;
    try {
      rawStrategies.value =
        (await ReleaseStrategiesService.listReleaseStrategies({ workspaceID: spaceStore.currentSpace })) ?? [];
      isError.value = false;
    } catch (error) {
      console.error(error);
      isError.value = true;
    } finally {
      isLoading.value = false;
    }
  }

  /** 关键字变化时重置到第一页（本地过滤即时生效） */
  watch(keyword, () => {
    pagination.value.current = 1;
  });

  function handlePageLimitChange(limit: number) {
    pagination.value.limit = limit;
    pagination.value.current = 1;
  }

  function handlePageValueChange(current: number) {
    pagination.value.current = current;
  }

  return {
    isLoading,
    isError,
    keyword,
    pagination,
    /** 全量策略（未经搜索过滤与分页切片），供克隆命名等需要全局信息的场景使用 */
    allStrategies: rawStrategies,
    /** 过滤 + 排序后的完整列表（未分页），供保存后定位目标行使用 */
    filteredStrategies,
    strategies,
    fetchList,
    handlePageLimitChange,
    handlePageValueChange,
  };
}
