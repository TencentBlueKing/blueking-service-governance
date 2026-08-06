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
        :disable-gray="hasCrossPageSelection"
        :is-all-instances-selected="isSelectedEnvAllInstancesSelected"
        :selected-count="selectedCount"
        show-remove-deploy-shortcut
        @admin-command="instanceActions.openAdminCommand"
        @delete="instanceActions.openDelete"
        @gray="instanceActions.openGray()"
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
        mode="multiEnv"
        :selected-env-name="selectedEnvName"
        show-env-header
        :total-count="isEnvRequestable(envName) ? undefined : 0"
        @collapse-change="handleCollapseChange"
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
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import TableException from '~/components/table-exception.vue';
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
  /** 已折叠的环境名称集合 */
  const collapsedEnvNames = ref<Set<string>>(new Set());

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

  /** 判断指定环境是否属于可请求范围 */
  function isEnvRequestable(envName: string): boolean {
    return requestableEnvNameSet.value.has(envName);
  }

  /** 判断两个环境名称数组是否完全一致（顺序和长度都相等） */
  function isSameEnvNames(a: string[] = [], b: string[] = []) {
    return a.length === b.length && a.every((envName, index) => envName === b[index]);
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

  /** 是否存在未折叠且有数据的激活环境 */
  const hasActiveSelectedEnv = computed(() =>
    selectedEnvs.value.some(envName => !collapsedEnvNames.value.has(envName)),
  );

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

  /** 处理环境表格折叠/展开变化：折叠时停止轮询，展开时加载数据并启动轮询 */
  async function handleCollapseChange(payload: { envName: string; isCollapsed: boolean }) {
    const nextCollapsedEnvNames = new Set(collapsedEnvNames.value);
    if (payload.isCollapsed) {
      nextCollapsedEnvNames.add(payload.envName);
    } else {
      nextCollapsedEnvNames.delete(payload.envName);
    }
    collapsedEnvNames.value = nextCollapsedEnvNames;

    if (payload.isCollapsed) {
      if (!hasActiveSelectedEnv.value) {
        stopPolling();
      }
      return;
    }

    await envTableRefs.value.get(payload.envName)?.loadInstances?.();
    if (!timer.value) {
      startPolling();
    }
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

  /** 实例列表控制器：统一管理批量操作、轮询、灰度部署等逻辑 */
  const { canGrayDeploy, handleRowAction, instanceActions, startPolling, stopPolling, timer } =
    useInstanceListController({
      actionsHostRef,
      pollInterval: 10000,
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

  /** 监听 appID、选中环境、可请求环境变化：清理无效数据并控制轮询启停 */
  watch(
    [() => appDetailStore.appID, selectedEnvs, () => props.requestableEnvNames],
    async () => {
      if (!appDetailStore.appID || selectedEnvs.value.length === 0) {
        stopPolling();
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
      collapsedEnvNames.value = new Set(
        [...collapsedEnvNames.value].filter(envName => selectedEnvs.value.includes(envName)),
      );

      if (hasActiveSelectedEnv.value && !timer.value) {
        startPolling();
      } else if (!hasActiveSelectedEnv.value) {
        stopPolling();
      }
    },
    { immediate: true, deep: true },
  );

  /** 监听可请求环境列表变化：环境就绪后刷新数据并启动轮询 */
  watch(
    () => props.requestableEnvNames,
    async (requestableEnvNames, oldRequestableEnvNames = []) => {
      if (!appDetailStore.appID || selectedEnvs.value.length === 0) {
        return;
      }
      if (isSameEnvNames(requestableEnvNames, oldRequestableEnvNames)) {
        return;
      }
      await nextTick();
      await handleRefreshAll();
      if (hasActiveSelectedEnv.value && !timer.value) {
        startPolling();
      }
    },
    { deep: true },
  );

  /** 组件卸载前停止轮询 */
  onBeforeUnmount(() => {
    stopPolling();
  });

  /** 暴露给父组件的方法 */
  defineExpose({
    handleRefreshAll,
  });
</script>
