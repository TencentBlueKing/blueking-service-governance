<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <div class="flex flex-col p-[16px] bg-white shadow-[0_2px_4px_0_#1919290d]">
    <div class="flex items-center justify-between mb-[16px]">
      <Radio.Group
        class="shrink-0"
        :model-value="activeState"
        type="capsule"
        @change="handleStateChange"
      >
        <Radio.Button
          v-for="item in statusOptions"
          :key="item.value"
          :label="item.value"
        >
          <span class="inline-flex gap-[7px] items-center">
            <span
              class="w-[8px] h-[8px] rounded-full shadow-[0_0_0_3px_#f0f1f5]"
              :style="{ backgroundColor: item.color }"
            />
            <span>{{ item.label }}</span>
            <span
              class="min-w-[22px] px-[8px] leading-[20px] text-[#63656e] text-center bg-white rounded-[10px]"
              :class="activeState === item.value ? '!text-[#3a84ff] !bg-[#E1ECFF]' : ''"
            >
              {{ formatCount(statusCounts[item.key]) }}
            </span>
          </span>
        </Radio.Button>
      </Radio.Group>

      <SearchSelect
        v-model="searchSelectValue"
        class="w-[420px] bg-white relative z-[100]"
        :data="workspaceSearchData"
        :placeholder="t('搜索空间 ID、空间名称')"
        unique-select
        value-behavior="need-key"
      />
    </div>

    <Table
      v-bkloading="{ loading: isLoading }"
      auto-resize
      class="flex-1 w-full min-h-0"
      :data="workspaceList"
      :pagination="pagination"
      :row-config="{
        isHover: true,
        isCurrent: true,
      }"
      :row-height="56"
      :sort-config="sortConfig"
      sync-resize
      @page-limit-change="handlePageLimitChange"
      @page-value-change="handlePageValueChange"
      @sort-change="handleSortChange"
    >
      <template #empty>
        <TableException
          :type="curExceptionType"
          @clear="handleClearFilters"
          @refresh="fetchWorkspaceList"
        />
      </template>

      <TableColumn
        field="displayName"
        :label="t('空间名称（ID）')"
        :min-width="210"
        show-overflow="tooltip"
      >
        <template #default="{ row, rowIndex }: { row: WorkspaceWithStatsOutput; rowIndex?: number }">
          <div class="flex items-center min-w-0">
            <span
              class="inline-flex shrink-0 items-center justify-center w-[40px] h-[40px] text-[16px] font-semibold text-white rounded-[4px]"
              :style="{ backgroundColor: getIconBackground(rowIndex) }"
            >
              {{ getWorkspaceInitial(row) }}
            </span>
            <span class="flex flex-col min-w-0 ml-[10px] leading-[20px]">
              <span
                class="truncate text-[#3A84FF] cursor-pointer hover:text-[#1768ef]"
                @click="handleGotoWorkspaceDetail(row)"
              >
                {{ row.displayName || '--' }}
              </span>
              <span class="truncate text-[#979ba5]">
                {{ row.id || '--' }}
              </span>
            </span>
          </div>
        </template>
      </TableColumn>

      <TableColumn
        field="description"
        :label="t('空间描述')"
        :min-width="180"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.description || '--' }}
        </template>
      </TableColumn>

      <TableColumn
        field="appCount"
        :label="t('应用数量')"
        :width="90"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.appCount ?? 0 }}
        </template>
      </TableColumn>

      <TableColumn
        field="envCount"
        :label="t('环境数量')"
        :width="90"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.envCount ?? 0 }}
        </template>
      </TableColumn>

      <TableColumn
        field="state"
        :label="t('状态')"
        :width="120"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          <span class="inline-flex gap-[10px] items-center">
            <span
              class="w-[8px] h-[8px] rounded-full shadow-[0_0_0_3px_#f0f1f5]"
              :style="{ backgroundColor: getStateColor(row.state) }"
            />
            {{ getStateLabel(row.state) }}
          </span>
        </template>
      </TableColumn>

      <TableColumn
        field="creator"
        :label="t('创建人')"
        :min-width="130"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.creator || '--' }}
        </template>
      </TableColumn>

      <TableColumn
        field="updater"
        :label="t('最近更新人')"
        :min-width="130"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.updater || '--' }}
        </template>
      </TableColumn>

      <TableColumn
        field="updatedAt"
        :label="t('最近更新时间')"
        :min-width="180"
        show-overflow="tooltip"
        sortable
      >
        <template #default="{ row }: { row: WorkspaceWithStatsOutput }">
          {{ row.updatedAt ? formatDateString(row.updatedAt) : '--' }}
        </template>
      </TableColumn>
    </Table>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Radio, SearchSelect } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { PlatmgtService } from '~/api/modules/v1';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';

  import type { ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { ListPlatWorkspacesRequest, WorkspaceStatsOutput, WorkspaceWithStatsOutput } from '~/@types/v1/platmgt';

  type SortOrder = '' | 'asc' | 'desc';
  type StatusCountKey = 'all' | 'disabled' | 'ready';
  type WorkspaceStateOption = 'all' | 'Disabled' | 'Ready';

  const ICON_COLORS = ['#3A84FF', '#2CAF5E', '#F59500', '#6228FF', '#EF6317', '#EA3636'];
  const STATUS_COLORS: Record<string, string> = {
    Ready: '#2CAF5E',
    Processing: '#F59500',
    Disabled: '#C4C6CC',
  };

  const { t } = useI18n();
  const router = useRouter();
  const { formatDateString } = useTime();

  const workspaceList = ref<WorkspaceWithStatsOutput[]>([]);
  const isLoading = ref(false);
  const activeState = ref<WorkspaceStateOption>('Ready');
  const stateFilter = computed(() => (activeState.value === 'all' ? '' : activeState.value));
  const searchSelectValue = ref<ISearchValue[]>([]);
  const searchValue = computed(() => getSearchKeyword(searchSelectValue.value));
  const debouncedSearchValue = useDebounce(searchValue, 300);
  const sortBy = ref('');
  const sortOrder = ref<SortOrder>('');
  const requestSequence = ref(0);

  const pagination = ref({
    count: 0,
    current: 1,
    limit: 10,
    remote: true,
    showTotalCount: true,
  });
  const sortConfig = {
    multiple: false,
    trigger: 'cell',
  };

  const statusCounts = reactive<Record<StatusCountKey, null | number>>({
    all: null,
    ready: null,
    disabled: null,
  });
  const workspaceSearchData = ref<ISearchItem[]>([
    {
      name: t('空间 ID'),
      id: 'id',
      placeholder: t('空间 ID'),
    },
    {
      name: t('空间名称'),
      id: 'displayName',
      placeholder: t('空间名称'),
    },
  ]);
  const statusOptions = computed(() => [
    { key: 'all' as const, label: t('全部'), value: 'all' as const, color: '#3A84FF' },
    { key: 'ready' as const, label: t('启用中'), value: 'Ready' as const, color: STATUS_COLORS.Ready },
    { key: 'disabled' as const, label: t('已停用'), value: 'Disabled' as const, color: STATUS_COLORS.Disabled },
  ]);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: [searchValue, stateFilter],
  });

  async function fetchWorkspaceList() {
    const currentSequence = ++requestSequence.value;
    isLoading.value = true;

    try {
      const keyword = debouncedSearchValue.value.trim();
      const params: ListPlatWorkspacesRequest = {
        ...(keyword ? { keyword } : {}),
        ...(stateFilter.value ? { state: stateFilter.value } : {}),
        ...(sortBy.value && sortOrder.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}),
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      };
      const response = await PlatmgtService.listPlatWorkspaces(params);
      if (currentSequence !== requestSequence.value) return;

      workspaceList.value = response.results || [];
      const count = Number(response.count) || 0;
      pagination.value.count = count;
      updateStatusCounts(response.statistics);
      clearErrorType();
    } catch {
      if (currentSequence !== requestSequence.value) return;

      workspaceList.value = [];
      pagination.value.count = 0;
      setTypeToError();
    } finally {
      if (currentSequence === requestSequence.value) {
        isLoading.value = false;
      }
    }
  }

  function formatCount(count: null | number) {
    return count === null ? '--' : count;
  }

  function getIconBackground(rowIndex = 0) {
    const absoluteIndex = (pagination.value.current - 1) * pagination.value.limit + rowIndex;
    return ICON_COLORS[absoluteIndex % ICON_COLORS.length];
  }

  function getSearchKeyword(value: ISearchValue[]) {
    return value[0]?.values?.[0]?.id || '';
  }

  function getStateColor(state?: string) {
    return STATUS_COLORS[state || ''] || '#C4C6CC';
  }

  function getStateLabel(state?: string) {
    const stateLabels: Record<string, string> = {
      Ready: t('启用中'),
      Processing: t('处理中'),
      Disabled: t('已停用'),
    };
    return stateLabels[state || ''] || state || '--';
  }

  function getWorkspaceInitial(row: WorkspaceWithStatsOutput) {
    const name = (row.displayName || row.id || '').trim();
    return Array.from(name)[0] || '--';
  }

  function handleClearFilters() {
    const hasSearchValue = Boolean(searchValue.value);
    searchSelectValue.value = [];
    activeState.value = 'Ready';
    pagination.value.current = 1;
    if (!hasSearchValue) fetchWorkspaceList();
  }

  function handleGotoWorkspaceDetail(row: WorkspaceWithStatsOutput) {
    if (!row.id) return;
    router.push({
      name: 'platformWorkspaceDetail',
      params: {
        workspaceID: row.id,
      },
    });
  }

  function handlePageLimitChange(limit: number) {
    pagination.value.limit = limit;
    pagination.value.current = 1;
    fetchWorkspaceList();
  }

  function handlePageValueChange(current: number) {
    pagination.value.current = current;
    fetchWorkspaceList();
  }

  function handleSortChange({ field, order }: { field: string; order: null | SortOrder }) {
    sortBy.value = order ? field : '';
    sortOrder.value = order || '';
    pagination.value.current = 1;
    fetchWorkspaceList();
  }

  function handleStateChange(state: boolean | number | string) {
    if (state !== 'all' && state !== 'Ready' && state !== 'Disabled') return;
    if (activeState.value === state) return;
    activeState.value = state;
    pagination.value.current = 1;
    fetchWorkspaceList();
  }

  function toCount(value?: string) {
    const count = Number(value);
    return Number.isFinite(count) ? count : null;
  }

  function updateStatusCounts(stats?: WorkspaceStatsOutput) {
    statusCounts.all = toCount(stats?.totalCount);
    statusCounts.ready = toCount(stats?.readyCount);
    statusCounts.disabled = toCount(stats?.disabledCount);
  }

  watch(debouncedSearchValue, () => {
    pagination.value.current = 1;
    fetchWorkspaceList();
  });

  onMounted(() => {
    fetchWorkspaceList();
  });
</script>
