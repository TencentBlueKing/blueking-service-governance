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
    :loading="isLoading"
    theme="gray"
  >
    <template #loading>
      <div class="flex mb-[16px]">
        <Layout.shape
          class="mr-[16px]"
          :width="300"
        />
        <Layout.shape :width="400" />
      </div>
      <Layout.table />
    </template>
    <FlexRow
      average
      class="mb-[16px]"
      lclass="flex"
    >
      <template #left>
        <DatePicker
          v-model="dateRange"
          class="w-[300px] mr-[16px]"
          :placeholder="$t('请选择操作时间范围')"
          type="daterange"
          @change="handleListRecord"
        />
        <SearchSelect
          v-model="searchValue"
          class="w-[520px] bg-[#fff] relative z-[100]"
          :data="searchSelectData"
          :placeholder="
            createPlaceholder({
              type: 'searchSelect',
              labels: ['操作人', '操作类型', '操作结果'],
            })
          "
          unique-select
          value-behavior="need-key"
        >
        </SearchSelect>
      </template>
    </FlexRow>
    <Loading :loading="isLoading">
      <Table
        :data="list"
        :filter-config="{ remote: true }"
        :max-height="maxHeight"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        :row-height="40"
        :virtual-y-config="{ enabled: true, gt: 0 }"
        @filter-change="handleFilterChange"
        @page-limit-change="handlePageLimitChange"
        @page-value-change="handlePageValueChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="
              () => {
                searchValue = [];
                dateRange = ['', ''];
              }
            "
            @refresh="handleListRecord"
          />
        </template>
        <TableColumn
          field="resourceType"
          :filters="filterOptions.resourceType"
          label="资源类型"
          width="150"
        >
          <template #default="{ row }">
            <span>{{ row.resourceTypeDisplayName }}</span>
          </template>
        </TableColumn>
        <TableColumn
          field="operationType"
          :filters="filterOptions.operationType"
          :label="$t('操作类型')"
          width="120"
        >
          <template #default="{ row }">
            <span>
              {{ row.operationTypeDisplayName }}
            </span>
          </template>
        </TableColumn>
        <TableColumn
          field="attributeDisplayName"
          :label="$t('资源属性')"
          width="150"
        >
          <template #default="{ row }">
            {{ row.attributeDisplayName || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="resourceID"
          :label="$t('资源 ID')"
          min-width="150"
        >
        </TableColumn>
        <TableColumn
          field="accessType"
          :label="$t('环境')"
          width="150"
        >
          <template #default="{ row }">
            {{ envNameMapping?.[row?.group?.envName] || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="result"
          :filters="filterOptions.result"
          :label="$t('操作结果')"
          width="120"
        >
          <template #default="{ row }">
            <span>{{ getResultDisplayName(row.result) }}</span>
          </template>
        </TableColumn>
        <TableColumn
          field="username"
          :label="$t('操作人')"
          width="150"
        >
        </TableColumn>
        <TableColumn
          field="createdAt"
          :label="$t('操作时间')"
          width="200"
        >
          <template #default="{ row }">
            {{ row.createdAt ? formatTimeByTimezone(row.createdAt) : '' }}
          </template>
        </TableColumn>
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :show-overflow="false"
          :width="150"
        >
          <template #default="{ row }">
            <Button
              text
              theme="primary"
              @click.stop="handleShowLog(row)"
            >
              {{ $t('查看详情') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </Loading>
  </Skeleton>
  <!-- 操作详情 -->
  <Sideslider
    v-model:is-show="isShow"
    class="record-detail"
    :title="$t('操作详情')"
    :width="960"
  >
    <template #default>
      <!-- 代码编辑器 -->
      <MsEditor
        class="w-[100%] !h-[calc(100vh-52px)] p-[20px]"
        is-diff
        :model-value="curRow?.yamlAfter"
        :options="{
          enableSplitViewResizing: false,
          lineNumbersMinChars: 2,
        }"
        :original="curRow?.yamlBefore"
        readonly
      >
        <template #title>
          <div class="grid grid-cols-2 gap-[18px] text-[12px] text-center leading-[22px]">
            <div class="w-[52px] h-[22px] bg-[#1E3567] rounded-[2px]">
              {{ $t('操作前') }}
            </div>
            <div class="w-[52px] h-[22px] bg-[#144628] rounded-[2px] text-[#3FC362]">
              {{ $t('操作后') }}
            </div>
          </div>
        </template>
      </MsEditor>
    </template>
  </Sideslider>
</template>
<script lang="ts" setup>
  import { computed, ComputedRef, onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, DatePicker, Loading, SearchSelect, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { OperationRecordFilterOptionsOutputObj, OperationRecordOutputObj } from '~/@types/v1/operation-audit';
  import { OperationAuditService } from '~/api/modules/v1';
  import { convertToYaml, formatTimeByTimezone, mapKeys } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useEnvManager from '~/composables/use-env-manager';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useDynamicsHeight from '~/composables/use-table-height';
  import { useSpaceStore } from '~/stores/space';

  import type { DatePickerValueType } from 'bkui-vue/lib/date-picker/interface';
  import type { ICommonItem, ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';

  // 扩展类型，添加 YAML 格式数据
  interface ExtendedOperationRecord extends OperationRecordOutputObj {
    yamlAfter?: string;
    yamlBefore?: string;
  }

  // 引入国际化
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { envNameMapping, generateEnvNameMapping, handleGetEnvList } = useEnvManager();
  const spaceStore = useSpaceStore();
  const { maxHeight } = useDynamicsHeight(96, ['.header-right', '.container-header']);

  const operationFilters = ref<OperationRecordFilterOptionsOutputObj>();
  // 部署列表
  const isLoading = ref(false);
  const list = ref<OperationRecordOutputObj[]>([]);
  const pagination = ref({
    count: 0,
    limit: 10,
    current: 1,
    remote: true,
  });
  const curRow = ref<ExtendedOperationRecord>();
  // 日期范围
  const dateRange = ref<DatePickerValueType>(['', '']);
  // 操作详情
  const isShow = ref(false);
  // 搜索条件值
  const searchValue = ref<ISearchValue[]>([]);
  // 搜索条件配置
  const searchSelectData: ComputedRef<ISearchItem[]> = computed(() => [
    {
      name: t('操作人'),
      id: 'username',
    },
    {
      name: t('资源类型'),
      id: 'resourceType',
      children: mapKeys(operationFilters.value?.resourceTypes || [], {
        name: 'displayName',
        id: 'value',
      }) as unknown as ICommonItem[],
    },
    {
      name: t('操作类型'),
      id: 'operationType',
      children: mapKeys(operationFilters.value?.operationTypes || [], {
        name: 'displayName',
        id: 'value',
      }) as unknown as ICommonItem[],
    },
    {
      name: t('操作结果'),
      id: 'result',
      children: mapKeys(operationFilters.value?.operationResults || [], {
        name: 'displayName',
        id: 'value',
      }) as unknown as ICommonItem[],
    },
  ]);

  const { filterOptions, handleFilterChange } = useSearchFilter(searchSelectData, searchValue, [
    'resourceType',
    'operationType',
    'result',
  ] as const);
  // 表格空状态管理
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: [searchValue, dateRange],
  });

  /** 生成环境名称映射关系 */
  async function GenerateEnvNameMapping() {
    await handleGetEnvList();
    generateEnvNameMapping();
  }

  function getResultDisplayName(result: string): string {
    const displayName = operationFilters.value?.operationResults?.find(item => item.value === result)?.displayName;
    return displayName || '';
  }

  // 从 searchValue 中获取对应字段的值数组
  function getValuesFromSearchValue(field: string): string {
    const searchItem = searchValue.value.find(item => item.id === field);
    if (!searchItem) return '';
    return searchItem.values?.[0].id || '';
  }

  // 获取操作记录列表
  async function handleListRecord() {
    isLoading.value = true;
    try {
      const params = {
        workspaceID: spaceStore.currentSpace,
        operationType: getValuesFromSearchValue('operationType'),
        resourceType: getValuesFromSearchValue('resourceType'),
        result: getValuesFromSearchValue('result'),
        username: getValuesFromSearchValue('username'),
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      };
      if (dateRange.value instanceof Array && dateRange.value[0] && dateRange.value[1]) {
        Object.assign(params, {
          startedAt: (dateRange.value[0] as Date)?.toISOString(),
          endedAt: (dateRange.value[1] as Date)?.toISOString(),
        });
      }
      const res = await OperationAuditService.listOperationRecords(params);
      list.value = res?.results || [];
      pagination.value.count = Number(res?.count) || 0;
      clearErrorType();
    } catch (error) {
      console.error(error);
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  }

  function handlePageLimitChange(val: number) {
    pagination.value.limit = val;
    handleListRecord();
  }

  function handlePageValueChange(val: number) {
    pagination.value.current = val;
    handleListRecord();
  }

  // 查看操作详情
  function handleShowLog(row: OperationRecordOutputObj) {
    curRow.value = {
      ...row,
      yamlBefore: convertToYaml(row.data?.before || ''),
      yamlAfter: convertToYaml(row.data?.after || ''),
    };
    isShow.value = true;
  }

  async function initFilterOptions() {
    operationFilters.value = await OperationAuditService.listOperationRecordFilterOptions();
  }

  watch(searchValue, handleListRecord, { deep: true });

  onMounted(() => {
    handleListRecord();
    initFilterOptions();
    GenerateEnvNameMapping();
  });
</script>

<style lang="postcss">
  .bk-search-select-menu {
    .menu-content {
      .menu-item {
        padding: 0 8px !important;
      }
    }
  }
</style>
