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

<!--
  灰度发布进度视图的实例列表（独立于原实例列表）：
  反映当前实例的灰度归属（灰度新起 / 灰度前），随实例 Watch 与进度轮询实时刷新。
-->
<template>
  <div
    class="rounded-[2px] border border-[#DCDEE5] bg-white"
    data-testid="canary-instance-table"
  >
    <div class="flex items-center justify-between gap-[16px] px-[16px] py-[12px]">
      <!-- 胶囊筛选：全部 / 灰度新起 / 灰度前（与基础信息成员筛选同款 Radio.Group capsule） -->
      <Radio.Group
        v-model="activeFilter"
        type="capsule"
        @change="handleFilterChange"
      >
        <Radio.Button
          v-for="option in filterOptions"
          :key="option.id"
          :data-testid="`canary-filter-${option.id}`"
          :label="option.id"
        >
          <span class="flex items-center gap-[6px]">
            {{ option.label }}
            <!-- 计数角标：选中态蓝底蓝字，未选中白底灰字 -->
            <span
              :class="[
                'flex h-[16px] items-center justify-center rounded-full px-[6px] text-[12px] leading-none',
                activeFilter === option.id ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-white text-[#63656E]',
              ]"
            >
              {{ counts[option.id] }}
            </span>
          </span>
        </Radio.Button>
      </Radio.Group>
      <!-- 按实例名搜索 -->
      <Input
        v-model="keyword"
        class="w-[320px]"
        clearable
        :placeholder="$t('搜索实例名')"
        type="search"
      />
    </div>

    <Table
      :data="pagedInstances"
      :pagination="displayPagination"
      :row-config="{ keyField: 'id', isHover: true }"
      :show-overflow="false"
      :show-settings="false"
      @page-limit-change="handlePageSizeChange"
      @page-value-change="handlePageChange"
    >
      <template #empty>
        <TableException type="empty" />
      </template>

      <!-- 实例 -->
      <TableColumn
        field="id"
        :label="$t('实例')"
        min-width="200"
      >
        <template #default="{ row }: { row: AppInstanceOutputObj }">
          <span class="break-all leading-[20px]">{{ row.id || '--' }}</span>
        </template>
      </TableColumn>

      <!-- 流量权重：取北极星信息的静态权重 -->
      <TableColumn
        field="weight"
        :label="$t('流量权重')"
        :width="100"
      >
        <template #default="{ row }: { row: AppInstanceOutputObj }">
          {{ resolveInstanceWeight(row) }}
        </template>
      </TableColumn>

      <!-- 镜像 Tag：灰度新起实例带绿色小标签 -->
      <TableColumn
        field="image"
        :label="$t('镜像 Tag')"
        min-width="160"
      >
        <template #default="{ row }: { row: AppInstanceOutputObj }">
          <div class="flex items-center gap-[6px]">
            <span>{{ row.image?.split(':')?.pop() || '--' }}</span>
            <Tag
              v-if="resolveInstanceCanaryState(row, progress) === 'new'"
              class="!border-none !bg-[#DAF6E5] !text-[12px] !leading-[18px] !text-[#299E56]"
              size="small"
            >
              {{ $t('灰度新起') }}
            </Tag>
          </div>
        </template>
      </TableColumn>

      <!-- 操作：日志 / 监控 / 登录 / 调整权重 -->
      <TableColumn
        fixed="right"
        :label="$t('操作')"
        :width="200"
      >
        <template #default="{ row }: { row: AppInstanceOutputObj }">
          <div class="flex items-center gap-[4px]">
            <Button
              v-bk-tooltips="{
                content: $t('实例尚未创建成功或宿主机异常，暂无法获取日志'),
                disabled: canViewLog(row),
              }"
              :disabled="!canViewLog(row)"
              text
              theme="primary"
              @click.stop="emitRowAction('log', row)"
            >
              {{ $t('日志') }}
            </Button>
            <Button
              text
              theme="primary"
              @click.stop="emitRowAction('monitor', row)"
            >
              {{ $t('监控') }}
            </Button>
            <Button
              v-bk-tooltips="{
                content: $t('实例当前未处于运行状态，无法登录'),
                disabled: canLogin(row),
              }"
              :disabled="!canLogin(row)"
              text
              theme="primary"
              @click.stop="emitRowAction('login', row)"
            >
              {{ $t('登录') }}
            </Button>
            <Button
              v-bk-tooltips="{
                content: $t('未获取到北极星信息，无法调整权重'),
                disabled: canAdjustInstanceWeight(row),
              }"
              :disabled="!canAdjustInstanceWeight(row)"
              text
              theme="primary"
              @click.stop="emitRowAction('weight', row)"
            >
              {{ $t('调整权重') }}
            </Button>
          </div>
        </template>
      </TableColumn>
    </Table>

    <!-- 行操作弹窗宿主（自包含，弹窗 teleport 到 body） -->
    <InstanceActionsHost ref="actionsHostRef" />
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Input, Radio, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import TableException from '~/components/table-exception.vue';
  import useIsFederationEnv from '~/composables/use-is-federation-env';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { resolveInstanceCanaryState } from '../batch-release';
  import { resolveInstanceSourceMode } from '../composables/instance-watch-utils';
  import { useInstanceListController } from '../composables/use-instance-list-controller';
  import { useInstanceListWatch } from '../composables/use-instance-list-watch';
  import { canAdjustInstanceWeight, canLogin, canViewLog } from '../instance-utils';
  import InstanceActionsHost from './instance-actions-host.vue';

  import type { InstanceCanaryState } from '../batch-release';
  import type { InstanceListSourceMode, InstanceRowAction } from '../types';
  import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';

  const props = defineProps<{
    /** 当前环境名（实例 Watch 请求作用域） */
    envName: string;
    /** 分批发布进度快照（灰度归属判定口径：工作负载版本号为主，新版本镜像 Tag 兜底） */
    progress: BatchReleaseProgressOutputObj | null;
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();
  const actionsHostRef = ref<InstanceType<typeof InstanceActionsHost> | null>(null);

  const isFederationEnv = useIsFederationEnv(() => trpcDeployStore.curEnvItem);
  // 联邦标识由环境信息异步补齐；未确定前不把环境误判为普通集群而建立 Watch（与原实例列表同款用法）。
  const isFederationResolved = computed(() => typeof trpcDeployStore.curEnvItem?.cluster?.isFederation === 'boolean');

  const { instances: instanceList, refresh: refreshInstances } = useInstanceListWatch({
    enabled: () => isFederationResolved.value,
    getMode: (): InstanceListSourceMode => resolveInstanceSourceMode(isFederationEnv.value),
    getScope: () => ({
      appID: appDetailStore.appID,
      envName: props.envName || trpcDeployStore.curEnvItem?.name || '',
    }),
  });

  // 灰度列表无批量操作：选择态恒为空，控制器仅复用行操作弹窗链路。
  const { handleRowAction } = useInstanceListController({
    actionsHostRef,
    getEnvName: () => props.envName || trpcDeployStore.curEnvItem?.name || '',
    getSelectedInstances: () => [],
    selectedCount: computed(() => 0),
    isAllInstancesSelected: computed(() => false),
    clearSelections: () => undefined,
    refreshData: refreshInstances,
  });

  type CanaryFilterId = 'all' | InstanceCanaryState;

  function emitRowAction(action: InstanceRowAction, instance: AppInstanceOutputObj) {
    handleRowAction({ action, envName: props.envName || trpcDeployStore.curEnvItem?.name || '', instance });
  }

  /** 流量权重：取首个北极星信息的静态权重（有动态权重时优先），与原实例列表口径一致。 */
  function resolveInstanceWeight(instance: AppInstanceOutputObj) {
    const polaris = instance.polarisInfos?.[0];
    if (!polaris) return '--';
    return polaris.staticWeight || polaris.weight;
  }

  const filterOptions = computed(() => [
    { id: 'all' as const, label: t('全部') },
    { id: 'new' as const, label: t('灰度新起') },
    { id: 'before' as const, label: t('灰度前') },
  ]);

  const activeFilter = ref<CanaryFilterId>('all');

  /** 切换筛选后回到第一页，避免停留在空页。 */
  function handleFilterChange() {
    pagination.value.current = 1;
  }

  /** 胶囊筛选计数：computed O(n) 派生，随 Watch 与进度轮询实时刷新。 */
  const counts = computed(() => {
    const result: Record<CanaryFilterId, number> = { all: instanceList.value.length, before: 0, new: 0 };
    for (const instance of instanceList.value) {
      result[resolveInstanceCanaryState(instance, props.progress)] += 1;
    }
    return result;
  });

  /** 实例名模糊搜索。 */
  const keyword = ref('');

  const filteredInstances = computed(() => {
    const text = keyword.value.trim().toLowerCase();
    return instanceList.value.filter(instance => {
      if (activeFilter.value !== 'all' && resolveInstanceCanaryState(instance, props.progress) !== activeFilter.value) {
        return false;
      }
      return !text || (instance.id ?? '').toLowerCase().includes(text);
    });
  });

  // 本地分页：数据源为 Watch 全量快照，与原实例列表一致
  const pagination = ref({
    current: 1,
    limit: 10,
  });

  const displayTotal = computed(() => filteredInstances.value.length);

  const pagedInstances = computed(() => {
    const start = (pagination.value.current - 1) * pagination.value.limit;
    return filteredInstances.value.slice(start, start + pagination.value.limit);
  });

  const displayPagination = computed(() => ({
    current: pagination.value.current,
    count: displayTotal.value,
    limit: pagination.value.limit,
    limitList: [10, 20, 50],
    remote: true,
    showLimit: true,
    showTotalCount: true,
    align: 'right' as const,
  }));

  // 数据变化后收敛页码，避免停留在空页
  watch(displayTotal, total => {
    const maxPage = Math.max(1, Math.ceil(total / pagination.value.limit));
    if (pagination.value.current > maxPage) {
      pagination.value.current = maxPage;
    }
  });

  function handlePageChange(current: number) {
    pagination.value.current = current;
  }

  function handlePageSizeChange(limit: number) {
    pagination.value.current = 1;
    pagination.value.limit = limit;
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-radio-capsule) {
    .is-checked {
      box-shadow: 0 2px 4px 0 rgba(0, 0, 0, 0.1);
    }
  }
</style>
