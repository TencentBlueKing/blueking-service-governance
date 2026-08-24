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
  <div class="flex h-full min-h-0 flex-col">
    <Skeleton
      :full-height="false"
      :loading="isLoading"
      theme="gray"
    >
      <!-- 首次请求使用与真实页面结构一致的骨架，刷新时保留统一的加载反馈。 -->
      <template #loading>
        <div class="mb-[16px] flex shrink-0 items-center justify-between">
          <Layout.shape :width="360" />
          <Layout.shape :width="110" />
        </div>
        <div class="mb-[16px] grid grid-cols-4 gap-[16px]">
          <Layout.shape
            v-for="index in 4"
            :key="index"
            :height="72"
            width="100%"
          />
        </div>
        <div class="rounded-[2px] bg-white p-[16px]">
          <div class="mb-[16px] flex justify-end"><Layout.shape :width="560" /></div>
          <Layout.table />
        </div>
      </template>

      <!-- 顶部操作栏：环境类型是总览表格的一级筛选，部署按钮复用部署管理现有侧栏。 -->
      <div
        class="mb-[16px] flex shrink-0 items-center justify-between gap-[16px] bg-[#EAEBF0] px-[12px] py-[8px] shadow-[0_2px_4px_0_#0000001a]"
      >
        <Radio.Group
          v-model="globalTypeFilter"
          class="shrink-0"
          type="capsule"
        >
          <Radio.Button
            v-for="option in envTypeOptions"
            :key="option.value"
            :label="option.value"
          >
            <div class="flex items-center justify-center">
              {{ option.label }}
              <span
                :class="[
                  'ml-[4px] h-[16px] rounded-[8px] px-[6px] leading-[16px]',
                  option.value === globalTypeFilter ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-white',
                ]"
              >
                {{ option.count }}
              </span>
            </div>
          </Radio.Button>
        </Radio.Group>
        <div class="flex items-center gap-[8px]">
          <Button
            v-bk-tooltips="$t('刷新')"
            class="!min-w-[32px] !px-0 bg-[#fff]"
            @click="handleRefresh"
          >
            <i
              :class="[
                'bkms-icon bkms-icon-refresh text-[16px]',
                isLoading ? 'animate-spin [animation-duration:1.5s]' : '',
              ]"
            ></i>
          </Button>
          <Button
            v-if="canAddDeploy"
            class="bg-[#fff]"
            @click="emit('create-feature-env')"
          >
            <Plus class="text-[18px]" />
            {{ $t('新建环境') }}
          </Button>
          <Button
            v-if="canAddDeploy"
            theme="primary"
            @click="emit('deploy', deployTargets)"
          >
            {{ $t('部署') }}
          </Button>
        </div>
      </div>

      <!-- 指标卡统计当前环境类型的数据；所有卡片始终可点击，数量为 0 时也能查看对应空结果。 -->
      <div class="mb-[16px] grid shrink-0 grid-cols-4 gap-[16px]">
        <button
          v-for="stat in stats"
          :key="stat.key"
          :class="[
            'flex min-w-0 cursor-pointer items-center justify-between gap-[12px] rounded-[2px] bg-white px-[16px] py-[12px] text-left shadow-[0_2px_4px_0_#1919290d] hover:shadow-[0_2px_8px_0_#1919291f]',
            activeStat === stat.key ? '!bg-[#E1ECFF] outline outline-[1px] outline-[#3A84FF]' : '',
          ]"
          type="button"
          @click="handleStatClick(stat.key)"
        >
          <div class="flex min-w-0 flex-col gap-[4px]">
            <span class="text-[12px] text-[#979BA5]">{{ stat.label }}</span>
            <div class="flex items-baseline gap-[8px]">
              <span
                class="text-[24px] font-bold leading-[28px]"
                :style="{ color: stat.color }"
              >
                {{ stat.value }}
              </span>
              <span
                v-if="stat.desc"
                class="truncate text-[12px] text-[#979BA5]"
              >
                {{ stat.desc }}
              </span>
            </div>
          </div>
          <span
            class="flex h-[40px] w-[40px] shrink-0 items-center justify-center rounded-full"
            :style="{ backgroundColor: stat.iconBg, color: stat.iconColor }"
          >
            <StatIcon :name="stat.key" />
          </span>
        </button>
      </div>

      <!-- 表格容器占满剩余高度，仅表格内容滚动，顶部筛选与分页保持可见。 -->
      <div class="flex min-h-0 flex-1 flex-col rounded-[2px] bg-white p-[16px] shadow-[0_2px_4px_0_#1919290d]">
        <div class="mb-[16px] flex shrink-0 justify-end">
          <SearchSelect
            v-model="searchValue"
            class="relative z-[100] w-[560px] shrink-0 bg-white"
            :data="searchData"
            :placeholder="createPlaceholder({ type: 'searchSelect', labels: ['环境名称', '部署状态', '资源规格'] })"
            unique-select
            value-behavior="need-key"
          />
        </div>
        <div
          ref="tableContentRef"
          class="min-h-0 flex-1 overflow-hidden"
        >
          <Table
            ref="tableRef"
            v-bkloading="{ loading: isLoading }"
            :data="visibleRows"
            :filter-config="{ remote: true }"
            :max-height="tableHeight"
            :pagination="pagination"
            row-class-name="cursor-pointer"
            :row-config="{ isHover: true }"
            :row-height="48"
            :show-overflow="false"
            :sort-config="sortConfig"
            @filter-change="handleFilterChange"
            @page-limit-change="handlePageLimitChange"
            @page-value-change="handlePageValueChange"
            @row-click="handleRowClick"
          >
            <!-- 区分接口异常、筛选无结果和接口成功但无数据三种空态。 -->
            <template #empty>
              <TableException
                :type="isError ? 'error' : hasFilter ? 'search' : 'empty'"
                @clear="clearFilters"
                @refresh="load"
              />
            </template>
            <TableColumn
              field="displayName"
              fixed="left"
              :label="$t('环境')"
              :min-width="220"
            >
              <template #default="{ row }: { row: DeployOverviewRow }">
                <div class="flex items-center gap-[4px]">
                  <span class="truncate text-[#3A84FF] hover:text-[#699DF4]">
                    {{ row.displayName }}
                  </span>
                  <Tag
                    v-if="row.isFeature"
                    class="shrink-0 bg-[#E2F5F7] text-[#3A9EAA]"
                    size="small"
                  >
                    {{ $t('特性') }}
                  </Tag>
                </div>
                <div class="text-[#979BA5]">{{ row.name }}</div>
              </template>
            </TableColumn>
            <TableColumn
              field="type"
              :label="$t('环境类型')"
              min-width="110"
            >
              <template #default="{ row }: { row: DeployOverviewRow }">
                <Tag :class="envTypeTagClassMap[row.type]">{{ envTypeMap[row.type]?.name || '--' }}</Tag>
              </template>
            </TableColumn>
            <TableColumn
              field="deployStatus"
              filter-multiple
              :filters="filterOptions.deployStatus"
              min-width="130"
            >
              <template #header>
                <CustomFilter
                  field="deployStatus"
                  :filters="filterOptions.deployStatus || []"
                  :label="$t('部署状态')"
                  :table-ref="tableRef"
                />
              </template>
              <template #default="{ row }: { row: DeployOverviewRow }">
                <div class="flex w-fit items-center">
                  <StatusDotIcon
                    :icon="getStatusInfo(row.deployStatus).icon"
                    :size="12"
                  />
                  <span class="text-[#4D4F56]">{{ getStatusInfo(row.deployStatus).text || '--' }}</span>
                </div>
              </template>
            </TableColumn>
            <TableColumn
              :label="$t('实例数（运行/期望/异常）')"
              min-width="200"
            >
              <template #default="{ row }: { row: DeployOverviewRow }">
                <div class="flex items-center">
                  <span class="text-[#313238]">{{ row.runningCount ?? '--' }}</span>
                  <span class="mx-[4px] text-[#DCDEE5]">/</span>
                  <span class="text-[#313238]">{{ row.desiredCount ?? '--' }}</span>
                  <span class="mx-[4px] text-[#DCDEE5]">/</span>
                  <span :class="row.abnormalCount ? 'text-[#EA3636]' : 'text-[#979BA5]'">
                    {{ row.abnormalCount ?? '--' }}
                  </span>
                  <!-- 未配置不显示；已关闭显示中性标签；已启用复用 GPA 状态标签。 -->
                  <template v-if="row.autoScale.enabled">
                    <Popover
                      v-if="row.autoScale.tips.length && !row.autoScale.abnormal"
                      placement="top"
                      :popover-delay="[100, 0]"
                    >
                      <AutoScaleTag
                        class="ml-[8px]"
                        enabled
                        size="small"
                        :status="row.autoScale.status"
                      />
                      <template #content>
                        <div
                          v-for="tip in row.autoScale.tips"
                          :key="tip"
                          class="whitespace-nowrap leading-[20px]"
                        >
                          {{ tip }}
                        </div>
                      </template>
                    </Popover>
                    <AutoScaleTag
                      v-else
                      class="ml-[8px]"
                      enabled
                      size="small"
                      :status="row.autoScale.status"
                    />
                  </template>
                </div>
              </template>
            </TableColumn>
            <TableColumn
              :label="$t('资源规格')"
              min-width="120"
            >
              <template #default="{ row }: { row: DeployOverviewRow }">
                <Popover
                  v-if="getResourceText(row)"
                  placement="top"
                >
                  <span class="cursor-default border-b border-dashed border-[#979BA5]">{{ getResourceText(row) }}</span>
                  <template #content>
                    <div
                      v-for="line in getResourceTips(row)"
                      :key="line"
                      class="whitespace-nowrap leading-[20px]"
                    >
                      {{ line }}
                    </div>
                  </template>
                </Popover>
                <span v-else>--</span>
              </template>
            </TableColumn>
            <TableColumn
              field="deployedAt"
              :label="$t('最近部署')"
              min-width="160"
              sortable
            >
              <template #default="{ row }: { row: DeployOverviewRow }">
                <span
                  v-bk-tooltips="{
                    content: formatDeployedAt(row.deployedAt).tooltip,
                    disabled: !formatDeployedAt(row.deployedAt).tooltip,
                  }"
                >
                  {{ formatDeployedAt(row.deployedAt).text }}
                </span>
              </template>
            </TableColumn>
          </Table>
        </div>
      </div>
    </Skeleton>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, toRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Popover, Radio, SearchSelect, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import CustomFilter from '~/components/custom-filter.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import TableException from '~/components/table-exception.vue';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useElementHeight } from '~/composables/use-element-height';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import { formatRelativeTimeWithTooltip } from '~/composables/use-time';
  import AutoScaleTag from '~/pages/application/detail/components/auto-scale-tag.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import StatIcon from './stat-icon.vue';
  import { type DeployOverviewDeployTarget, type DeployOverviewRow, useDeployOverview } from './use-deploy-overview';

  import type { EnvOutput } from '~/@types/v1/env';

  const props = defineProps<{ envList: EnvOutput[] }>();
  const emit = defineEmits<{
    'create-feature-env': [];
    deploy: [targets: DeployOverviewDeployTarget[]];
    'update:deploy-targets': [targets: DeployOverviewDeployTarget[]];
    'view-instances': [envName: string];
  }>();

  const appDetailStore = useAppDetail();
  const { createPlaceholder } = useSearchPlaceholder();
  const tableRef = ref();
  const tableContentRef = ref<HTMLElement | null>(null);
  const canAddDeploy = computed(() => isAppModelAppType(appDetailStore.appType));

  // 接口适配和表格状态集中在 composable，当前组件只维护 DOM 引用与页面事件。
  const {
    activeStat,
    clearFilters,
    deployTargets,
    envTypeOptions,
    filterOptions,
    getResourceText,
    getResourceTips,
    getStatusInfo,
    globalTypeFilter,
    handleFilterChange,
    handlePageLimitChange,
    handlePageValueChange,
    handleStatClick,
    hasFilter,
    isError,
    isLoading,
    load,
    pagination,
    searchData,
    searchValue,
    sortConfig,
    stats,
    visibleRows,
  } = useDeployOverview(toRef(props, 'envList'));

  const { height: tableHeight } = useElementHeight(tableContentRef, { watchSource: isLoading });

  /** 将最近部署时间转换为相对时间，并保留完整时间作为 tooltip。 */
  function formatDeployedAt(deployedAt: string) {
    return formatRelativeTimeWithTooltip(deployedAt, { alwaysRelative: true });
  }

  /** 刷新过程中忽略重复点击，避免并发请求总览接口。 */
  function handleRefresh() {
    if (isLoading.value) return;
    load();
  }

  /** 点击总览表格任意数据单元格时进入对应环境的实例列表。 */
  function handleRowClick(_event: Event, row: DeployOverviewRow) {
    emit('view-instances', row.name);
  }

  // 部署、移除部署等父级操作完成后，通过暴露的 load 主动刷新总览。
  defineExpose({ load });

  // 应用或应用类型变化时重新请求；composable 内部会丢弃上一应用的迟到响应。
  watch([() => appDetailStore.appID, () => appDetailStore.appType], load, { immediate: true });

  // 环境列表可能晚于总览接口返回，持续把最新部署目标同步给已打开的新增部署侧栏。
  watch(deployTargets, targets => emit('update:deploy-targets', targets));
</script>
