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
  <div class="flex items-center justify-between gap-[12px] mb-[16px]">
    <div :id="DEPLOY_ENV_SELECT_SLOT_ID"></div>
    <div class="flex items-center gap-[12px]">
      <DatePicker
        v-model="dateRange"
        class="w-[300px]"
        clearable
        :placeholder="$t('请选择操作时间范围')"
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
  </div>
  <Skeleton
    :full-height="false"
    :loading="isLoading"
  >
    <template #loading>
      <Layout.table />
    </template>
    <Table
      v-bkloading="{ loading: isLoading }"
      :data="tableData"
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
        :width="180"
      >
        <template #default="{ row }">
          {{ row.componentName || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('类型')"
        show-overflow="tooltip"
        :width="150"
      >
        <template #default="{ row }">
          {{ row.type || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('资源类型')"
        :width="120"
      >
        <template #default="{ row }">
          {{ row.resourceKind || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('资源名称')"
        show-overflow="tooltip"
        :width="150"
      >
        <template #default="{ row }">
          {{ row.resourcesName || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('事件级别')"
        :width="100"
      >
        <template #default="{ row }">
          <span :class="eventLevelClass(row.level)">
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
  </Skeleton>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { DatePicker, SearchSelect } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { EventEntryOutputObj } from '~/@types/v1/instance';
  import { InstanceService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import TableException from '~/components/table-exception.vue';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { DEPLOY_ENV_SELECT_SLOT_ID } from './constants';

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString } = useTime();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  const searchValue = ref<Array<{ id: string; name: string; values?: Array<{ id: string; name: string }> }>>([]);
  const dateRange = ref<[] | [Date, Date]>([]);
  const tableData = ref<EventEntryOutputObj[]>([]);
  const isLoading = ref(false);
  const count = ref(0);

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

  // 从 searchValue 中提取事件级别
  const selectedLevel = computed(() => {
    const levelItem = searchValue.value.find(item => item.id === 'level');
    if (levelItem?.values && levelItem.values.length > 0) {
      return levelItem.values[0].id;
    }
    return '';
  });

  // 事件级别样式
  function eventLevelClass(level: string) {
    return level === 'Warning' ? 'text-[#EA3636]' : 'text-[#2DCB56]';
  }

  // 获取事件列表
  async function fetchEvents() {
    if (!appDetailStore.appID || !trpcDeployStore.curEnvItem?.name) return;

    isLoading.value = true;
    try {
      const [startDate, endDate] = dateRange.value;
      const hasValidDateRange = startDate && endDate;

      const res = await InstanceService.listEvents({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem.name,
        trafficLaneName: '',
        page: pagination.value.current,
        pageSize: pagination.value.limit,
        ...(selectedLevel.value && { level: selectedLevel.value }),
        ...(hasValidDateRange && {
          startedAt: Math.floor(new Date(startDate).getTime() / 1000),
          endedAt: Math.floor(new Date(endDate).getTime() / 1000),
        }),
      });

      tableData.value = res.results ?? [];
      count.value = Number(res.count ?? 0);
    } catch {
      tableData.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  // 清除搜索
  function handleClearSearch() {
    searchValue.value = [];
    dateRange.value = [];
  }

  // 日期变化处理
  function handleDateChange() {
    handleResetPage();
    fetchEvents();
  }

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

  watch(
    searchValue,
    () => {
      handleResetPage();
      fetchEvents();
    },
    { immediate: true, deep: true },
  );

  // 监听环境变化
  watch(
    () => trpcDeployStore.curEnvItem?.name,
    () => {
      handleResetPage();
      fetchEvents();
    },
  );
</script>
