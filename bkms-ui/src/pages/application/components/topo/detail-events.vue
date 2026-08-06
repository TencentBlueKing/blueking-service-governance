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
  <Skeleton
    :full-height="false"
    :loading="isLoading && isFirstLoad"
  >
    <template #loading>
      <div class="p-[24px]">
        <div class="flex items-center justify-end gap-[12px] mb-[16px]">
          <Layout.shape :width="300" />
          <Layout.shape :width="360" />
        </div>
        <Layout.table />
      </div>
    </template>
    <div class="h-full p-[24px] flex flex-col gap-[16px]">
      <div class="flex justify-end items-center gap-[12px]">
        <DatePicker
          v-model="dateRange"
          class="w-[300px]"
          clearable
          :placeholder="$t('请选择时间范围')"
          type="daterange"
          @change="handleDateChange"
        />
        <SearchSelect
          v-model="searchValue"
          class="min-w-[360px] bg-[#fff] relative z-[100]"
          :data="searchData"
          :placeholder="
            createPlaceholder({
              type: 'searchSelect',
              labels: ['事件级别'],
            })
          "
          unique-select
          value-behavior="need-key"
        />
      </div>
      <div
        ref="EventTableContentRef"
        class="flex-1"
      >
        <Table
          v-bkloading="{ loading: isLoading }"
          :data="tableData"
          :max-height="eventTableContentHeight"
          :pagination="pagination"
          @page-limit-change="pageSizeChange"
          @page-value-change="pageChange"
        >
          <template #empty>
            <TableException
              :type="tableData.length === 0 && searchValue.length === 0 ? 'empty' : 'search'"
              @clear="handleClearSearch"
            />
          </template>
          <TableColumn
            :label="$t('时间')"
            :width="150"
          >
            <template #default="{ row }">
              {{ row.createdAt ? formatDateString(row.createdAt) : '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('组件')"
            show-overflow="tooltip"
            :width="120"
          >
            <template #default="{ row }">
              {{ row.componentName || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('类型')"
            show-overflow="tooltip"
            :width="120"
          >
            <template #default="{ row }">
              {{ row.type || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('事件级别')"
            :width="100"
          >
            <template #default="{ row }">
              <span :class="row.level === 'Warning' ? 'text-[#EA3636]' : 'text-[#2DCB56]'">
                {{ row.level || '--' }}
              </span>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('事件内容')"
            show-overflow="tooltip"
          >
            <template #default="{ row }">
              {{ row.content || '--' }}
            </template>
          </TableColumn>
        </Table>
      </div>
    </div>
  </Skeleton>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { DatePicker, SearchSelect } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { TopologyNodeEvent } from '~/@types/topology';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import TableException from '~/components/table-exception.vue';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTime from '~/composables/use-time';

  const props = defineProps<{
    appId: string;
    envName: string;
    nodeId: string;
  }>();

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString } = useTime();

  const searchValue = ref<Array<{ id: string; name: string; values?: Array<{ id: string; name: string }> }>>([]);
  const dateRange = ref<[] | [Date, Date]>([]);
  const tableData = ref<TopologyNodeEvent[]>([]);
  const isLoading = ref(false);
  const isFirstLoad = ref(true);
  const count = ref(0);

  const EventTableContentRef = ref<HTMLElement>();
  const { height: eventTableContentHeight } = useElementHeight(EventTableContentRef, {
    watchSource: isLoading,
    defaultHeight: 200,
  });

  /** 从 searchValue 中提取事件级别 */
  const selectedLevel = computed(() => {
    const levelItem = searchValue.value.find(item => item.id === 'level');
    return levelItem?.values?.[0]?.id ?? '';
  });

  const searchData = [
    {
      name: t('事件级别'),
      id: 'level',
      children: [
        { id: 'Normal', name: 'Normal' },
        { id: 'Warning', name: 'Warning' },
      ],
    },
  ];

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    tableData,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: fetchEvents,
      onPageSizeChange: fetchEvents,
    },
    count,
  );

  async function fetchEvents() {
    if (!props.nodeId || !props.appId || !props.envName) return;
    isLoading.value = true;
    try {
      const [startDate, endDate] = dateRange.value;
      const hasValidDateRange = startDate && endDate;

      const res = await ApiServerService.ListTopologyNodeEvents({
        appID: props.appId,
        envName: props.envName,
        trafficLaneName: '',
        nodeID: props.nodeId,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
        ...(selectedLevel.value && { level: selectedLevel.value }),
        ...(hasValidDateRange && {
          startedAt: Math.floor(new Date(startDate).getTime() / 1000),
          endedAt: Math.floor(new Date(endDate).getTime() / 1000),
        }),
      });
      tableData.value = res?.results ?? [];
      count.value = Number(res?.count ?? 0);
    } catch (_) {
      tableData.value = [];
      count.value = 0;
    } finally {
      isLoading.value = false;
      isFirstLoad.value = false;
    }
  }

  function handleClearSearch() {
    searchValue.value = [];
    dateRange.value = [];
  }

  function handleDateChange() {
    handleResetPage();
    fetchEvents();
  }

  watch(
    searchValue,
    () => {
      handleResetPage();
      fetchEvents();
    },
    { deep: true },
  );

  watch(
    () => props.nodeId,
    newId => {
      if (newId) {
        handleResetPage();
        fetchEvents();
      }
    },
    { immediate: true },
  );
</script>
