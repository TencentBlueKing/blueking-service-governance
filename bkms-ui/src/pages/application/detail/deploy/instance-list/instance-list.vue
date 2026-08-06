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
    :loading="isLoading"
  >
    <template #loading>
      <div class="flex items-center justify-between mb-[12px]">
        <div class="flex items-center gap-[8px]">
          <Layout.shape />
          <Layout.shape />
          <Layout.shape />
        </div>
        <Layout.shape :width="348" />
      </div>
      <Layout.table />
    </template>
    <div class="flex items-center justify-between mb-[12px]">
      <InstanceBatchToolbar
        :can-gray-deploy="canGrayDeploy"
        :disable-admin-command="isCrossPageSelection"
        :disable-delete="isCrossPageSelection"
        :is-all-instances-selected="isAllInstancesSelected"
        :selected-count="selectedCount"
        show-remove-deploy-shortcut
        @admin-command="instanceActions.openAdminCommand"
        @delete="instanceActions.openDelete"
        @gray="handleShowGrayUpgrade()"
        @monitor="instanceActions.openMonitor()"
        @remove-deploy="emit('remove-deploy')"
      />
      <SearchSelect
        v-model="searchValue"
        class="min-w-[348px] bg-[#fff] relative z-[100]"
        :data="searchData"
        :placeholder="
          createPlaceholder({
            type: 'searchSelect',
            labels: ['实例', '镜像 Tag', '实例状态', '健康状态', '北极星状态'],
          })
        "
        unique-select
        value-behavior="need-key"
      >
      </SearchSelect>
    </div>
    <InstanceTable
      ref="instanceTableRef"
      :data="tableDataMatchSearch"
      :enable-max-height="false"
      :env-name="trpcDeployStore.curEnvItem?.name || ''"
      :filter-options="filterOptions"
      mode="single"
      show-filter
      :total-count="total"
      @filter-change="handleFilterChange"
      @page-change="handlePageChange"
      @page-size-change="handlePageSizeChange"
      @row-action="handleRowAction"
    >
      <template #empty>
        <TableException
          :type="curExceptionType"
          @clear="handleClearFilters"
          @refresh="handleListReleaseInstances"
        >
        </TableException>
      </template>
    </InstanceTable>
  </Skeleton>

  <InstanceActionsHost ref="actionsHostRef" />
</template>
<script lang="ts" setup>
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { SearchSelect } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import { InstanceService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import InstanceActionsHost from './components/instance-actions-host.vue';
  import InstanceBatchToolbar from './components/instance-batch-toolbar.vue';
  import InstanceTable from './components/instance-table.vue';
  import { useInstanceListController } from './composables/use-instance-list-controller';
  import { isPolarisHealthy } from './instance-utils';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';

  const props = defineProps<{
    hasDeployRecord?: boolean;
  }>();
  const emit = defineEmits<{
    'remove-deploy': [];
  }>();

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const actionsHostRef = ref<InstanceType<typeof InstanceActionsHost> | null>(null);

  // 部署列表
  const instanceTableRef = ref<InstanceType<typeof InstanceTable> | null>(null);
  const isLoading = ref(false);
  const instanceList = ref<AppInstanceOutputObj[]>([]);
  const total = ref(0);
  const paginationCurrent = ref(1);
  const paginationLimit = ref(10);

  const isCrossPageSelection = computed(() => instanceTableRef.value?.isCrossPageSelection ?? false);

  // 从 InstanceTable 获取选中状态
  const selectedCount = computed(() => instanceTableRef.value?.selectedCount ?? 0);
  // 判断是否选中了所有实例
  const isAllInstancesSelected = computed(() => instanceTableRef.value?.isAllSelected ?? false);
  // 清除所有选中状态
  function clearSelections() {
    instanceTableRef.value?.clearSelections?.();
  }

  // 获取实际选中的实例
  function getSelectedInstances(): AppInstanceOutputObj[] {
    return instanceTableRef.value?.getSelections?.() ?? [];
  }

  const { canGrayDeploy, handleRowAction, instanceActions, startPolling, stopPolling, timer } =
    useInstanceListController({
      actionsHostRef,
      beforeRowAction: payload => {
        if (payload.action === 'gray') {
          instanceTableRef.value?.clearSelections?.();
        }
      },
      pollInterval: 5000,
      getEnvName: () => trpcDeployStore.curEnvItem?.name || '',
      getSelectedInstances,
      selectedCount,
      isAllInstancesSelected,
      clearSelections,
      refreshData: handleListReleaseInstances,
      grayEnvDisplayName: () => trpcDeployStore.curEnvItem?.displayName || '',
      commandEnvDisplayName: () => trpcDeployStore.curEnvItem?.displayName || '',
      resolveGrayInstanceIds: () => {
        const table = instanceTableRef.value;
        if (!table) return undefined;
        const isReallySelectAll = table.isCrossPageSelection && table.selectedCount === table.getTotal();
        return isReallySelectAll ? [] : undefined;
      },
    });

  // 加载实例列表数据
  async function handleListReleaseInstances() {
    try {
      isLoading.value = true;

      if (!appDetailStore.app || !trpcDeployStore.curEnvItem?.name) return;
      const res = await InstanceService.listAppInstances({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name,
        page: paginationCurrent.value,
        pageSize: paginationLimit.value,
      }).catch(() => {
        clearSelections();
        setTypeToError();
        return { count: 0, results: [] };
      });

      clearErrorType();
      total.value = Number(res.count);
      instanceList.value = (res.results || []) as AppInstanceOutputObj[];

      // 动态更新筛选项（镜像 Tag、健康状态、北极星状态）
      updateDynamicFilterChildren(instanceList.value);
    } catch (err) {
      console.error(err);
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  }

  // 分页事件处理
  function handlePageChange(current: number) {
    paginationCurrent.value = current;
    handleListReleaseInstances();
  }

  // 处理每页条数切换并重置到第一页。
  function handlePageSizeChange(limit: number) {
    paginationCurrent.value = 1;
    paginationLimit.value = limit;
    handleListReleaseInstances();
  }

  // 灰度：需要处理跨页全选的特殊逻辑
  function handleShowGrayUpgrade(row?: AppInstanceOutputObj) {
    if (row) {
      instanceTableRef.value?.clearSelections?.();
    }
    instanceActions.openGray(row);
  }

  /** 空值筛选标识 */
  const EMPTY_FILTER_ID = '__empty__';

  /** 从实例列表中动态提取筛选项，更新 searchData 中各字段的 children */
  function updateDynamicFilterChildren(instances: AppInstanceOutputObj[]) {
    const imageSet = new Set<string>();
    const healthySet = new Set<string>();
    const polarisSet = new Set<string>();

    for (const instance of instances) {
      // 镜像 Tag
      if (instance.image) {
        imageSet.add(instance.image);
      }
      // 健康状态
      healthySet.add(String(instance.isHealthy));
      // 北极星状态
      if (!instance.polarisInfos?.length) {
        polarisSet.add(EMPTY_FILTER_ID);
      } else {
        polarisSet.add(isPolarisHealthy(instance) ? 'healthy' : 'unhealthy');
      }
    }

    const healthyLabelMap: Record<string, string> = { true: 'Healthy', false: 'UnHealthy' };
    const polarisLabelMap: Record<string, string> = {
      healthy: 'Healthy',
      [EMPTY_FILTER_ID]: '--',
      unhealthy: 'UnHealthy',
    };

    // 批量更新各字段的 children
    const childrenMap: Record<string, { id: string; name: string }[]> = {
      image: Array.from(imageSet).map(image => ({
        id: image,
        name: image.split(':').pop() || image,
      })),
      isHealthy: Array.from(healthySet).map(val => ({
        id: val,
        name: healthyLabelMap[val] || val,
      })),
      polarisStatus: Array.from(polarisSet).map(val => ({
        id: val,
        name: polarisLabelMap[val] || val,
      })),
    };

    for (const item of searchData.value) {
      if (childrenMap[item.id]) {
        item.children = childrenMap[item.id];
      }
    }
  }

  // 实例状态列表
  const statusOptions = [
    { id: 'Succeeded', name: 'Succeeded' },
    { id: 'Failed', name: 'Failed' },
    { id: 'Unknown', name: 'Unknown' },
    { id: 'Pending', name: 'Pending' },
    { id: 'Running', name: 'Running' },
    { id: 'CrashLoopBackOff', name: 'CrashLoopBackOff' },
    { id: 'ImagePullBackOff', name: 'ImagePullBackOff' },
    { id: 'Terminating', name: 'Terminating' },
    { id: 'Evicted', name: 'Evicted' },
    { id: 'NotReady', name: 'NotReady' },
    { id: 'Completed', name: 'Completed' },
  ];

  const searchData = ref([
    {
      name: t('实例'),
      id: 'id',
      multiple: false,
    },
    {
      name: t('镜像 Tag'),
      id: 'image',
      multiple: true,
      children: [] as { id: string; name: string }[],
    },
    {
      name: t('实例状态'),
      id: 'status',
      multiple: true,
      children: statusOptions,
    },
    {
      name: t('健康状态'),
      id: 'isHealthy',
      multiple: true,
      children: [] as { id: string; name: string }[],
    },
    {
      name: t('北极星状态'),
      id: 'polarisStatus',
      multiple: true,
      children: [] as { id: string; name: string }[],
    },
  ]);

  /** SearchSelect 选中值 */
  const searchValue = ref<ISearchValue[]>([]);

  /** 可筛选的列字段名 */
  const filterKeys = ['image', 'status', 'isHealthy', 'polarisStatus'] as const;

  /** 使用 useSearchFilter hook 实现 TableColumn filter 与 SearchSelect 联动 */
  const { filterOptions, handleFilterChange } = useSearchFilter(searchData, searchValue, filterKeys);

  /** 前端筛选逻辑 */
  const tableDataMatchSearch = computed(() => {
    if (!searchValue.value.length) return instanceList.value;

    let filteredList = [...instanceList.value];

    for (const filter of searchValue.value) {
      if (!filter.values?.length) continue;
      const selectedValues = filter.values.map(v => v.id);

      switch (filter.id) {
        case 'id':
          // 实例名称：模糊搜索
          filteredList = filteredList.filter(item =>
            selectedValues.some(val => item.id?.toLowerCase().includes(val.toLowerCase())),
          );
          break;
        case 'image':
          // 镜像 Tag：精确匹配
          filteredList = filteredList.filter(item => selectedValues.includes(item?.image ?? ''));
          break;
        case 'status':
          // 实例状态：精确匹配
          filteredList = filteredList.filter(item => selectedValues.includes(item?.status ?? ''));
          break;
        case 'isHealthy':
          // 健康状态：布尔值转字符串匹配
          filteredList = filteredList.filter(item => selectedValues.includes(String(item.isHealthy)));
          break;
        case 'polarisStatus':
          // 北极星状态：聚合判断
          filteredList = filteredList.filter(item => {
            if (!item.polarisInfos?.length) {
              return selectedValues.includes(EMPTY_FILTER_ID);
            }
            return selectedValues.includes(isPolarisHealthy(item) ? 'healthy' : 'unhealthy');
          });
          break;
      }
    }

    return filteredList;
  });
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  // 清除所有筛选条件
  // 清空搜索组件和表格列筛选状态。
  function handleClearFilters() {
    const vxeTable = instanceTableRef.value?.getVxeTableInstance?.();
    if (vxeTable) {
      for (const key of filterKeys) {
        vxeTable.clearFilter(key);
      }
    }
    resetPagination();
    searchValue.value = [];
  }
  // 重置实例列表
  function resetInstanceList() {
    stopPolling();
    clearSelections();
    total.value = 0;
    instanceList.value = [];
    updateDynamicFilterChildren([]);
  }
  function resetPagination() {
    paginationCurrent.value = 1;
    instanceTableRef.value?.resetPage?.();
  }

  watch(searchValue, () => {
    resetPagination();
  });

  watch(
    () => appDetailStore.app,
    async () => {
      await appDetailStore.fetchAppDetail();
      await trpcDeployStore.getDeploySpec();
    },
    { immediate: true },
  );

  watch(
    [() => appDetailStore.appID, () => trpcDeployStore.curEnvItem?.name, () => props?.hasDeployRecord],
    async () => {
      // 如果当前环境没有部署记录，则清空旧数据并停止轮询，避免移除部署后继续展示上一次实例列表。
      if (!props?.hasDeployRecord) {
        resetInstanceList();
        return;
      }
      await handleListReleaseInstances();
      if (!timer.value) {
        startPolling();
      }
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    stopPolling();
  });

  defineExpose({
    handleListReleaseInstances,
  });
</script>
