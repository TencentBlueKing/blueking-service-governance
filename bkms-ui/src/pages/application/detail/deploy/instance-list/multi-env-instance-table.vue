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

<!-- 多环境实例表格 - 每个环境独立 Table 纵向排列 -->
<template>
  <Skeleton
    :full-height="false"
    :loading="isLoading"
  >
    <template #loading>
      <div class="flex items-center justify-between my-[12px]">
        <div class="flex items-center gap-[8px]">
          <Layout.shape />
          <Layout.shape />
          <Layout.shape />
        </div>
      </div>
      <Layout.table />
    </template>

    <!-- 批量操作区域 -->
    <div class="flex items-center justify-between mb-[12px]">
      <InstanceBatchToolbar
        :can-gray-deploy="canGrayDeploy"
        :disable-admin-command="hasCrossPageSelection"
        :disable-delete="hasCrossPageSelection"
        :disable-gray="hasCrossPageSelection || isSelectedEnvFederation"
        :gray-disabled-tip="isSelectedEnvFederation ? $t('联邦集群不支持灰度操作') : ''"
        :is-all-instances-selected="isSelectedEnvAllInstancesSelected"
        :selected-count="selectedCount"
        show-remove-deploy-shortcut
        @admin-command="instanceActions.openAdminCommand"
        @delete="instanceActions.openDelete"
        @gray="handleOpenGray"
        @monitor="instanceActions.openMonitor(undefined, undefined, selectedEnvs)"
        @remove-deploy="handleRemoveDeploy"
      />
    </div>

    <div
      v-if="selectedEnvs.length > 0"
      class="flex flex-col pb-[18px]"
    >
      <InstanceTable
        v-for="envName in selectedEnvs"
        :key="envName"
        :ref="el => setEnvTableRef(envName, el)"
        :data="isEnvRequestable(envName) ? undefined : []"
        :env-display-name="getEnvDisplayName(envName)"
        :env-kind="getEnvKind(envName)"
        :env-name="envName"
        :env-type="getEnvType(envName)"
        :is-federation="isEnvFederation(envName)"
        mode="multiEnv"
        :selected-env-name="selectedEnvName"
        show-env-header
        @data-loaded="handleEnvDataLoaded"
        @row-action="handleRowAction"
        @selection-change="handleEnvSelectionChange"
      />
    </div>

    <TableException
      v-else
      type="empty"
    />
  </Skeleton>

  <InstanceActionsHost ref="actionsHostRef" />
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import TableException from '~/components/table-exception.vue';
  import { isFederationEnv } from '~/composables/use-is-federation-env';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  import InstanceActionsHost from './components/instance-actions-host.vue';
  import InstanceBatchToolbar from './components/instance-batch-toolbar.vue';
  import InstanceTable from './components/instance-table.vue';
  import { useInstanceListController } from './composables/use-instance-list-controller';

  import type { InstanceDataLoadedPayload, InstanceSelectionChangePayload, InstanceTableExpose } from './types';
  import type { EnvOutputObj } from '~/@types/env';

  const props = defineProps<{
    /** 允许发起数据请求的环境名称列表 */
    requestableEnvNames: string[];
  }>();
  const emit = defineEmits<{
    'remove-deploy': [env: EnvOutputObj];
  }>();

  const appDetailStore = useAppDetail();
  const envStore = useDeployEnvStore();
  const actionsHostRef = ref<InstanceType<typeof InstanceActionsHost> | null>(null);

  /** 可请求环境名称的 Set，用于快速判断 */
  const requestableEnvNameSet = computed(() => new Set(props.requestableEnvNames || []));
  /** 当前选中的环境名称列表（来自环境存储） */
  const selectedEnvs = computed(() => envStore.selectedEnvs || []);
  const isLoading = ref(false);
  /** 环境名称到 InstanceTable 实例的映射 */
  const envTableRefs = ref<Map<string, InstanceTableExpose>>(new Map());

  // 获取环境展示名称，优先使用环境中心返回的 displayName。
  function getEnvDisplayName(envName: string): string {
    const envItem = envStore.envList?.find(e => e.name === envName);
    return envItem?.displayName || envName;
  }

  function getEnvKind(envName: string): string {
    const envItem = envStore.envList?.find(e => e.name === envName);
    return envItem?.kind || '';
  }

  // 获取环境类型，用于渲染环境标签样式。
  function getEnvType(envName: string): string {
    const envItem = envStore.envList?.find(e => e.name === envName);
    return envItem?.type || '';
  }

  // 判断环境是否是联邦集群
  function isEnvFederation(envName: string): boolean {
    return isFederationEnv(envStore.envList?.find(env => env.name === envName));
  }

  /** 判断指定环境是否属于可请求范围 */
  function isEnvRequestable(envName: string): boolean {
    return requestableEnvNameSet.value.has(envName);
  }

  // 维护每个环境表格实例的引用映射。
  function setEnvTableRef(envName: string, el: unknown) {
    if (el) {
      envTableRefs.value.set(envName, el as InstanceTableExpose);
    } else {
      envTableRefs.value.delete(envName);
    }
  }

  const envSelections = ref<Map<string, AppInstanceOutputObj[]>>(new Map());
  /** 环境名称到实例总数的映射 */
  const envTotals = ref<Map<string, number>>(new Map());

  /** 当前存在选中实例的环境名称（优先返回第一个非零选中的环境） */
  const selectedEnvName = computed<string | undefined>(() => {
    for (const [envName, tableRef] of envTableRefs.value) {
      if ((tableRef?.selectedCount ?? 0) > 0) {
        return envName;
      }
    }
    return undefined;
  });

  /** 当前选中环境是否是联邦集群 */
  const isSelectedEnvFederation = computed(() =>
    selectedEnvName.value ? isEnvFederation(selectedEnvName.value) : false,
  );

  /** 汇总所有环境表格的选中实例总数 */
  const selectedCount = computed(() => {
    let count = 0;
    for (const tableRef of envTableRefs.value.values()) {
      count += tableRef?.selectedCount ?? 0;
    }
    return count;
  });

  /** 汇总所有环境表格的实例总数 */
  const totalInstances = computed(() => {
    let sum = 0;
    for (const total of envTotals.value.values()) {
      sum += total;
    }
    return sum;
  });

  /** 是否全选所有实例 */
  const isAllInstancesSelected = computed(() => {
    return selectedCount.value === totalInstances.value && totalInstances.value > 0;
  });

  /** 是否全选当前操作环境下的所有实例 */
  const isSelectedEnvAllInstancesSelected = computed(() => {
    const tableRef = getSelectedEnvTableRef();
    const total = tableRef?.getTotal?.() ?? 0;
    return (tableRef?.selectedCount ?? 0) === total && total > 0;
  });

  /** 是否存在跨页选中（禁用批量删除/灰度等操作） */
  const hasCrossPageSelection = computed(() => {
    for (const tableRef of envTableRefs.value.values()) {
      if (tableRef?.isCrossPageSelection) {
        return true;
      }
    }
    return false;
  });

  // 汇总所有环境表格中的已选实例。
  function getAllSelections(): AppInstanceOutputObj[] {
    const all: AppInstanceOutputObj[] = [];
    for (const selections of envSelections.value.values()) {
      all.push(...selections);
    }
    return all;
  }

  function getSelectedEnvItem() {
    return envStore.envList?.find(env => env.name === selectedEnvName.value);
  }

  // 获取当前存在选中项的环境表格引用。
  function getSelectedEnvTableRef() {
    return selectedEnvName.value ? envTableRefs.value.get(selectedEnvName.value) : undefined;
  }

  // 清空所有环境表格中的勾选状态。
  function handleClearAllSelections() {
    for (const tableRef of envTableRefs.value.values()) {
      tableRef?.clearSelections?.();
    }
    envSelections.value = new Map();
  }

  // 同步单个环境表格返回的总数信息。
  function handleEnvDataLoaded(payload: InstanceDataLoadedPayload) {
    envTotals.value.set(payload.envName, payload.total);
    envTotals.value = new Map(envTotals.value);
  }

  // 同步单个环境表格返回的选中实例集合。
  function handleEnvSelectionChange(payload: InstanceSelectionChangePayload) {
    envSelections.value.set(payload.envName, payload.selections);
    envSelections.value = new Map(envSelections.value);
  }

  // 打开灰度部署弹窗
  function handleOpenGray() {
    if (isSelectedEnvFederation.value) return;
    instanceActions.openGray();
  }

  // 刷新所有未折叠环境表格的数据。
  async function handleRefreshAll() {
    const tasks: Promise<void>[] = [];
    for (const tableRef of envTableRefs.value.values()) {
      if (tableRef.isCollapsed) {
        continue;
      }
      if (tableRef.loadInstances) {
        tasks.push(tableRef.loadInstances());
      }
    }
    await Promise.all(tasks);
  }

  // 移除部署快捷操作
  function handleRemoveDeploy() {
    const envItem = getSelectedEnvItem();
    if (!envItem) return;
    emit('remove-deploy', envItem);
  }

  /** 实例列表控制器：统一管理批量操作、刷新和灰度部署等逻辑。 */
  const { canGrayDeploy, handleRowAction, instanceActions } = useInstanceListController({
    actionsHostRef,
    getEnvName: () => selectedEnvName.value || selectedEnvs.value[0] || '',
    getSelectedInstances: getAllSelections,
    selectedCount,
    isAllInstancesSelected,
    clearSelections: handleClearAllSelections,
    refreshData: handleRefreshAll,
    resolveGrayInstanceIds: () => {
      const table = getSelectedEnvTableRef();
      if (!table) return undefined;
      const isReallySelectAll = table.isCrossPageSelection && table.selectedCount === table.getTotal();
      return isReallySelectAll ? [] : undefined;
    },
  });

  /** 监听选中环境和可请求环境变化，清理已失效的汇总数据。 */
  watch(
    [() => appDetailStore.appID, selectedEnvs, () => props.requestableEnvNames],
    () => {
      if (!appDetailStore.appID || selectedEnvs.value.length === 0) {
        return;
      }

      for (const envName of envSelections.value.keys()) {
        if (!selectedEnvs.value.includes(envName) || !isEnvRequestable(envName)) {
          envSelections.value.delete(envName);
          envTotals.value.delete(envName);
        }
      }
      envSelections.value = new Map(envSelections.value);
      envTotals.value = new Map(envTotals.value);
    },
    { immediate: true, deep: true },
  );

  /** 暴露给父组件的方法 */
  defineExpose({
    handleRefreshAll,
  });
</script>
