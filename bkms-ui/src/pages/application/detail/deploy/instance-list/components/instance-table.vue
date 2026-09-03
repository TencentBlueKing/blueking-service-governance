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

<!-- 实例表格：multiEnv 带环境头；支持跨页全选 / 列筛选 -->
<template>
  <div class="instance-table env-instance-table">
    <!-- 环境 Header（仅多环境） -->
    <div
      v-if="showEnvHeader && mode === 'multiEnv'"
      :class="[
        'env-header flex items-center gap-[8px] h-[42px] px-[16px] bg-[#f5f7fa] cursor-pointer',
        { collapsed: isCollapsed },
      ]"
      @click="toggleCollapse"
    >
      <RightShape
        :class="['text-[#979ba5] text-[14px] transition-transform duration-200', { 'rotate-90': !isCollapsed }]"
      />
      <span class="font-medium text-[#313238]">{{ envDisplayName }}</span>
      <Tag
        v-if="isFeatureEnv"
        class="bg-[#E2F5F7] text-[#3A9EAA]"
      >
        {{ $t('特性') }}
      </Tag>
      <Tag
        v-if="envType && envTypeMap[envType]"
        :class="envTypeTagClassMap[envType]"
      >
        {{ envTypeMap[envType]?.name || '' }}
      </Tag>
      <!-- 多环境下展示当前环境的自动扩缩容状态 -->
      <AutoScaleTag
        :enabled="isAutoScaleEnabled"
        :status="autoScaleStatus"
      />
    </div>

    <!-- Table 主体：折叠时卸载以节省渲染与刷新 -->
    <div
      v-if="!isCollapsed"
      class="env-table-body"
    >
      <Table
        ref="tableRef"
        :data="instanceList"
        :filter-config="filterConfig"
        :max-height="maxHeight"
        :pagination="displayPagination"
        :row-config="{
          keyField: 'id',
          isHover: true,
          isCurrent: true,
        }"
        :settings="settings"
        :show-settings="true"
        :sort-config="{ remote: true }"
        @filter-change="handleFilterChange"
        @page-limit-change="handlePageSizeChange"
        @page-value-change="handlePageChange"
        @setting-change="handleSettingChange"
        @sort-change="handleSortChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @refresh="handleRefresh"
          >
          </TableException>
        </template>

        <!-- prepend：跨页选择信息条 -->
        <template #prepend>
          <div
            v-if="mode === 'single' && hasSelection"
            class="flex items-center justify-center h-[30px] bg-[#ebecf0] text-[12px]"
          >
            <template v-if="isCrossPageSelection">
              <i18n-t keypath="已跨页全选 “{0}” 条，">
                <span class="font-bold mx-1">{{ displayTotal - excludedIds.size }}</span>
              </i18n-t>
              <Button
                text
                theme="primary"
                @click="handleClearSelection"
              >
                {{ $t('取消选择') }}
              </Button>
            </template>
            <template v-else>
              <i18n-t keypath="已选择 “{0}” 条，">
                <span class="font-bold mx-1">{{ selection.length }}</span>
              </i18n-t>
              <Button
                v-if="displayTotal > paginationInternal.limit"
                text
                theme="primary"
                @click="handleSelectAllCrossPage"
              >
                {{ $t('选择所有页共 {0} 条', [displayTotal]) }}
              </Button>
              <Button
                v-else
                text
                theme="primary"
                @click="handleClearSelection"
                >{{ $t('取消选择') }}</Button
              >
            </template>
          </div>
        </template>

        <!-- 复选框列：跨页全选 -->
        <TableColumn
          fixed="left"
          :width="80"
        >
          <template #header>
            <div class="flex items-center justify-start">
              <Checkbox
                v-bk-tooltips="{
                  content: $t('不支持跨环境操作'),
                  disabled: !isSelectAllDisabled,
                }"
                :disabled="isSelectAllDisabled || instanceList.length === 0"
                :indeterminate="isIndeterminate"
                :model-value="isCurrentPageAllChecked"
                @change="handleHeaderCheckboxClick"
              />
              <Dropdown
                v-if="mode !== 'multiEnv'"
                placement="bottom-start"
                trigger="click"
                @show-change="(val: boolean) => (isDropdownOpen = val)"
              >
                <AngleDownLine
                  :class="[
                    'ml-[8px] mt-[4px] text-[#979ba5] text-[12px] cursor-pointer transition-transform duration-200',
                    { 'rotate-180': isDropdownOpen },
                  ]"
                />
                <template #content>
                  <Dropdown.DropdownMenu>
                    <Dropdown.DropdownItem @click="handleSelectCurrentPage">
                      {{ $t('本页全选') }}
                    </Dropdown.DropdownItem>
                    <Dropdown.DropdownItem
                      :disabled="displayTotal <= paginationInternal.limit"
                      @click="handleSelectAllCrossPage"
                    >
                      {{ $t('跨页全选') }}
                    </Dropdown.DropdownItem>
                  </Dropdown.DropdownMenu>
                </template>
              </Dropdown>
            </div>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <Checkbox
              v-bk-tooltips="{
                content: $t('不支持跨环境操作'),
                disabled: !isCheckboxDisabled,
              }"
              :disabled="isCheckboxDisabled"
              :model-value="
                isCrossPageSelection ? !excludedIds.has(row.id) : selections.some(item => item.id === row.id)
              "
              @change="(checked: boolean) => handleCheckboxChange({ checked, row })"
            />
          </template>
        </TableColumn>

        <!-- 实例列 -->
        <TableColumn
          field="id"
          :label="$t('实例')"
          min-width="150"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.id }}
          </template>
        </TableColumn>

        <!-- 镜像 Tag 列（条件筛选头） -->
        <TableColumn
          field="image"
          :filters="showFilter ? filterOptions.image : undefined"
          :label="$t('镜像 Tag')"
          min-width="100"
          :show-overflow="false"
        >
          <template #header>
            <CustomFilter
              v-if="showFilter"
              :field="'image'"
              :filters="filterOptions.image || []"
              :label="$t('镜像 Tag')"
              :table-ref="tableRef"
            />
            <span v-else>{{ $t('镜像 Tag') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <span
              v-bk-tooltips="row.image"
              class="inline-block max-w-full truncate align-middle"
            >
              {{ row.image?.split(':')?.pop() }}
            </span>
          </template>
        </TableColumn>

        <!-- Pod IP 列 -->
        <TableColumn
          field="ip"
          label="Pod IP"
          min-width="100"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.ip || '--' }}
          </template>
        </TableColumn>

        <!-- Node IP 列 -->
        <TableColumn
          field="nodeIP"
          label="Node IP"
          min-width="100"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.nodeIP || '--' }}
          </template>
        </TableColumn>

        <!-- 实例状态列（条件筛选头） -->
        <TableColumn
          field="status"
          :filters="showFilter ? filterOptions.status : undefined"
          :label="$t('实例状态')"
          min-width="100"
          show-overflow="tooltip"
        >
          <template #header>
            <CustomFilter
              v-if="showFilter"
              :field="'status'"
              :filters="filterOptions.status || []"
              :label="$t('实例状态')"
              :table-ref="tableRef"
            />
            <span v-else>{{ $t('实例状态') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <StatusIcon
              emphasized
              :message="row.status !== 'Running' ? row.message : ''"
              :size="12"
              :status="row.status"
            />
          </template>
        </TableColumn>

        <!-- 健康状态列（条件筛选头） -->
        <TableColumn
          field="isHealthy"
          :filters="showFilter ? filterOptions.isHealthy : undefined"
          :label="$t('健康状态')"
          min-width="100"
          show-overflow="tooltip"
        >
          <template #header>
            <CustomFilter
              v-if="showFilter"
              :field="'isHealthy'"
              :filters="filterOptions.isHealthy || []"
              :label="$t('健康状态')"
              :table-ref="tableRef"
            />
            <span v-else>{{ $t('健康状态') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <div class="flex items-center">
              <StatusDotIcon
                :icon="row.isHealthy ? 'normal' : 'abnormal'"
                :size="12"
              />
              {{ row.isHealthy ? 'Healthy' : 'UnHealthy' }}
            </div>
          </template>
        </TableColumn>

        <!-- 北极星状态列（条件筛选头） -->
        <TableColumn
          field="polarisStatus"
          :filters="showFilter ? filterOptions.polarisStatus : undefined"
          :label="$t('北极星状态')"
          min-width="120"
        >
          <template #header>
            <CustomFilter
              v-if="showFilter"
              :field="'polarisStatus'"
              :filters="filterOptions.polarisStatus || []"
              :label="$t('北极星状态')"
              :table-ref="tableRef"
            />
            <span v-else>{{ $t('北极星状态') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <Popover
              v-if="row.polarisInfos?.length"
              hide-ignore-reference
              :offset="{
                mainAxis: 0,
                crossAxis: -10,
              }"
              :popover-delay="[100, 0]"
              theme="light"
              :width="770"
            >
              <div class="inline-flex cursor-pointer items-center">
                <StatusDotIcon
                  :icon="isPolarisHealthy(row) ? 'normal' : 'abnormal'"
                  :size="12"
                />
                <span class="!border-b-[1px] !border-[#AFAFAF] !border-dashed">
                  {{ isPolarisHealthy(row) ? 'Healthy' : 'UnHealthy' }}
                </span>
              </div>
              <template #content>
                <div class="px-[4px] min-w-[500px]">
                  <div class="mb-[12px] flex items-center text-[14px] whitespace-nowrap">
                    <span class="font-bold text-[#313238] shrink-0">{{ $t('北极星健康状态') }}</span>
                    <span
                      class="text-[#979BA5] ml-[12px] truncate"
                      :title="row.id"
                    >
                      {{ row.id }}
                    </span>
                  </div>
                  <div class="text-[12px] text-[#4D4F56] mb-[12px]">
                    <i class="bkms-icon bkms-icon-circle-info text-[14px] mr-[4px]"></i>
                    {{ $t('数据有 15s 左右延迟') }}
                  </div>
                  <Table
                    :data="row.polarisInfos"
                    :max-height="280"
                  >
                    <TableColumn
                      :label="$t('健康状态')"
                      min-width="80"
                    >
                      <template #default="{ row: polarisRow }">
                        <div class="flex items-center">
                          <StatusDotIcon
                            :icon="polarisRow.isHealthy ? 'normal' : 'abnormal'"
                            :size="12"
                          />
                          {{ polarisRow.isHealthy ? 'Healthy' : 'UnHealthy' }}
                        </div>
                      </template>
                    </TableColumn>
                    <TableColumn
                      label="ServiceName"
                      min-width="120"
                      show-overflow="tooltip"
                    >
                      <template #default="{ row: polarisRow }">
                        <HoverCopy
                          :copy-value="polarisRow.serviceName"
                          :text="polarisRow.serviceName"
                        />
                      </template>
                    </TableColumn>
                    <TableColumn
                      label="IP Port"
                      min-width="160"
                      show-overflow="tooltip"
                    >
                      <template #default="{ row: polarisRow }">
                        <HoverCopy
                          :copy-value="`${polarisRow.ip}:${polarisRow.port}`"
                          :text="`${polarisRow.ip}:${polarisRow.port}`"
                        />
                      </template>
                    </TableColumn>
                    <TableColumn
                      v-if="hasDynamicWeight(row.polarisInfos)"
                      :label="$t('动态权重')"
                      min-width="60"
                    >
                      <template #default="{ row: polarisRow }">
                        {{ polarisRow.weight }}
                      </template>
                    </TableColumn>
                    <TableColumn
                      :label="$t('流量权重')"
                      min-width="60"
                    >
                      <template #default="{ row: polarisRow }">
                        {{
                          hasDynamicWeight(row.polarisInfos)
                            ? polarisRow.staticWeight || polarisRow.weight
                            : polarisRow.weight
                        }}
                      </template>
                    </TableColumn>
                  </Table>
                </div>
              </template>
            </Popover>
            <span v-else>--</span>
          </template>
        </TableColumn>

        <!-- Restart 列 -->
        <TableColumn
          field="restartCount"
          label="Restart"
          min-width="100"
          show-overflow="tooltip"
          sortable
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.restartCount ?? '--' }}
          </template>
        </TableColumn>

        <!-- Age 列 -->
        <TableColumn
          field="age"
          label="Age"
          min-width="100"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.age || '--' }}
          </template>
        </TableColumn>

        <!-- 资源规格列：主容器 CPU / 内存规格，悬浮展示完整的 Requests/Limits -->
        <TableColumn
          field="resources"
          :label="$t('资源规格')"
          min-width="120"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <Popover
              v-if="getResourceText(row.resources)"
              placement="top"
            >
              <span class="cursor-default border-b border-dashed border-[#979BA5]">{{
                getResourceText(row.resources)
              }}</span>
              <template #content>
                <div
                  v-for="line in getResourceTips(row.resources)"
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

        <!-- 操作列 -->
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          min-width="240"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <div class="flex items-center gap-[10px]">
              <Button
                v-bk-tooltips="{
                  content: isFederation ? $t('联邦集群不支持灰度操作') : $t('仅支持实例状态为 Running、Pending 的实例'),
                  disabled: !isFederation && canInstanceGrayDeploy(row),
                }"
                :disabled="isFederation || !canInstanceGrayDeploy(row)"
                text
                theme="primary"
                @click.stop="
                  emit('row-action', { action: 'gray' as InstanceRowAction, envName: props.envName, instance: row })
                "
              >
                {{ $t('灰度') }}
              </Button>
              <Button
                v-bk-tooltips="{
                  content: $t('实例尚未创建成功或宿主机异常，暂无法获取日志'),
                  disabled: canViewLog(row),
                }"
                :disabled="!canViewLog(row)"
                text
                theme="primary"
                @click.stop="
                  emit('row-action', { action: 'log' as InstanceRowAction, envName: props.envName, instance: row })
                "
              >
                {{ $t('日志') }}
              </Button>
              <Button
                text
                theme="primary"
                @click.stop="
                  emit('row-action', { action: 'monitor' as InstanceRowAction, envName: props.envName, instance: row })
                "
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
                @click.stop="
                  emit('row-action', { action: 'login' as InstanceRowAction, envName: props.envName, instance: row })
                "
              >
                {{ $t('登录') }}
              </Button>
              <Button
                v-bk-tooltips="{
                  content: $t('未获取到北极星信息，无法调整权重'),
                  disabled: (row.polarisInfos?.length ?? 0) > 0,
                }"
                :disabled="!((row.polarisInfos?.length ?? 0) > 0)"
                text
                theme="primary"
                @click.stop="
                  emit('row-action', { action: 'weight' as InstanceRowAction, envName: props.envName, instance: row })
                "
              >
                {{ $t('调整权重') }}
              </Button>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Checkbox, Dropdown, Popover, Tag } from 'bkui-vue';
  import { AngleDownLine, RightShape } from 'bkui-vue/lib/icon';
  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import CustomFilter from '~/components/custom-filter.vue';
  import HoverCopy from '~/components/hover-copy.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import StatusIcon from '~/components/status-icon.vue';
  import TableException from '~/components/table-exception.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useGPAConfigPolling } from '~/composables/use-gpa-config-polling';
  import { useResourceSpecDisplay } from '~/composables/use-resource-spec-display';
  import useTableCheckbox from '~/composables/use-table-checkbox';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useTableSettings } from '~/composables/use-table-settings';
  import AutoScaleTag from '~/pages/application/detail/components/auto-scale-tag.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import {
    type RestartSortOrder,
    paginateInstances,
    sortInstancesByRestart,
  } from '../composables/instance-watch-utils';
  import { useInstanceListWatch } from '../composables/use-instance-list-watch';
  import { canInstanceGrayDeploy, canLogin, canViewLog, isPolarisHealthy } from '../instance-utils';

  import type {
    InstanceDataLoadedPayload,
    InstanceRowAction,
    InstanceRowActionPayload,
    InstanceSelectionChangePayload,
    InstanceTableMode,
  } from '../types';
  import type { FilterItem } from '~/components/custom-filter.vue';

  interface Props {
    /** 外部传入数据（单环境模式），提供后跳过内部 loadInstances */
    data?: AppInstanceOutputObj[];
    /** 是否限制表格最大高度 */
    enableMaxHeight?: boolean;
    envDisplayName?: string;
    envKind?: string;
    envName: string;
    envType?: string;
    /** 列筛选项数据 */
    filterOptions?: Record<string, FilterItem[]>;
    isFederation?: boolean;
    mode?: InstanceTableMode;
    selectedEnvName?: string;
    showEnvHeader?: boolean;
    /** 是否显示列筛选头 */
    showFilter?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    envDisplayName: '',
    envKind: '',
    envType: '',
    isFederation: false,
    mode: 'multiEnv',
    showEnvHeader: true,
    data: undefined,
    showFilter: false,
    filterOptions: () => ({}),
    enableMaxHeight: true,
  });
  const emit = defineEmits<{
    (e: 'selection-change', payload: InstanceSelectionChangePayload): void;
    (e: 'row-action', payload: InstanceRowActionPayload): void;
    (e: 'data-loaded', payload: InstanceDataLoadedPayload): void;
    (e: 'collapse-change', payload: { envName: string; isCollapsed: boolean }): void;
    (e: 'filter-change', payload: { field: string; values: string[] }): void;
    (e: 'page-change', current: number): void;
    (e: 'page-size-change', limit: number): void;
  }>();

  const appDetailStore = useAppDetail();
  const tableRef = ref();

  const { getResourceText, getResourceTips } = useResourceSpecDisplay();

  // 列设置：资源规格等新增列默认不勾选，用户可在表格右上角列设置中开启。
  // 列勾选与行高（size）偏好均持久化，刷新后恢复。
  // 多环境模式下按环境名区分列设置，v-for 渲染的多个表格互不共享。
  const tableSettingsId = computed(() => `instance-table-${props.envName || 'default'}`);
  const { settings, handleSettingChange } = useTableSettings(tableSettingsId, {
    defaultChecked: ['id', 'image', 'ip', 'nodeIP', 'status', 'isHealthy', 'polarisStatus', 'restartCount', 'age'],
    disabled: ['id'],
  });

  // 特性环境
  const isFeatureEnv = computed(() => props.envKind === 'feature');

  function hasDynamicWeight(polarisInfos: AppInstanceOutputObj['polarisInfos']) {
    return polarisInfos?.some(
      info => info.staticWeight !== undefined && info.staticWeight !== '' && info.staticWeight !== info.weight,
    );
  }

  // 折叠状态
  const isCollapsed = ref(false);
  const isDropdownOpen = ref(false);

  // 切换多环境表格的折叠状态。
  function toggleCollapse() {
    isCollapsed.value = !isCollapsed.value;
    emit('collapse-change', {
      envName: props.envName,
      isCollapsed: isCollapsed.value,
    });
  }

  // 是否使用外部数据
  const isExternalData = computed(() => props.data !== undefined);

  const {
    clear: clearWatchedInstances,
    instances: watchedInstances,
    lastError: watchError,
    refresh: refreshWatch,
  } = useInstanceListWatch({
    enabled: () => !isExternalData.value && !isCollapsed.value,
    getScope: () => ({
      appID: appDetailStore.appID,
      envName: props.envName,
    }),
  });

  watch(isExternalData, external => {
    if (external) clearWatchedInstances();
  });

  /** 单环境使用父组件传入的全量筛选结果，多环境使用当前环境的 Watch 快照。 */
  const allInstances = computed<AppInstanceOutputObj[]>(() =>
    isExternalData.value ? (props.data ?? []) : watchedInstances.value,
  );

  // 分页与 Restart 本地排序
  const paginationInternal = ref({
    current: 1,
    limit: 10,
  });
  const restartSortOrder = ref<RestartSortOrder>(null);

  const sortedInstances = computed(() => sortInstancesByRestart(allInstances.value, restartSortOrder.value));

  const displayTotal = computed(() => allInstances.value.length);
  const instanceList = computed(() => {
    return paginateInstances(sortedInstances.value, paginationInternal.value.current, paginationInternal.value.limit);
  });

  const {
    enabled: isAutoScaleEnabled,
    status: autoScaleStatus,
    updatePolling: updateAutoScalePolling,
  } = useGPAConfigPolling({
    active: () => props.mode === 'multiEnv',
    appID: () => appDetailStore.appID,
    envName: () => props.envName,
  });

  // Table 最大高度
  const maxHeight = computed(() => {
    if (!props.enableMaxHeight) return undefined;
    if (instanceList.value.length <= 5) return undefined;
    return 320;
  });

  const displayPagination = computed(() => ({
    current: paginationInternal.value.current,
    count: displayTotal.value,
    limit: paginationInternal.value.limit,
    remote: true,
    limitList: [10, 20, 50, 100],
    showLimit: true,
    showTotalCount: true,
    align: 'right' as const,
  }));

  const filterConfig = computed(() => (props.showFilter ? { remote: true } : undefined));

  // 空状态处理
  const emptyFilters = ref([]);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: emptyFilters,
  });

  watch(watchError, error => {
    if (error && allInstances.value.length === 0) {
      setTypeToError();
    } else if (!error) {
      clearErrorType();
    }
  });

  watch(
    displayTotal,
    total => {
      const maxPage = Math.max(1, Math.ceil(total / paginationInternal.value.limit));
      if (paginationInternal.value.current > maxPage) {
        paginationInternal.value.current = maxPage;
      }
      emit('data-loaded', {
        envName: props.envName,
        total,
        instances: allInstances.value,
      });
    },
    { immediate: true },
  );

  // 筛选事件
  // 将表格列筛选变化透传给父组件。
  function handleFilterChange(event: { field: string; values: string[] }) {
    emit('filter-change', event);
  }

  // 处理本地分页变化。
  function handlePageChange(current: number) {
    paginationInternal.value.current = current;
    emit('page-change', current);
  }

  // 处理分页大小变化并重置到第一页。
  function handlePageSizeChange(limit: number) {
    paginationInternal.value.current = 1;
    paginationInternal.value.limit = limit;
    emit('page-size-change', limit);
  }

  // 触发表格刷新，内部数据模式下重新请求实例列表。
  function handleRefresh() {
    if (!isExternalData.value) {
      loadInstances();
    }
  }

  // Restart 在全量集合上排序后再分页。
  function handleSortChange(event: { field?: string; order?: null | string }) {
    restartSortOrder.value =
      event.field === 'restartCount' && (event.order === 'asc' || event.order === 'desc') ? event.order : null;
    resetPage();
  }

  // 重新执行全量 List + Watch（多环境模式使用）。
  async function loadInstances() {
    if (isExternalData.value) return;
    await refreshWatch();
  }

  function resetPage(current = 1) {
    paginationInternal.value.current = current;
  }

  // ---- 跨页全选逻辑 ----

  const totalRef = computed(() => displayTotal.value);
  const {
    selections,
    selection,
    hasSelection,
    isCrossPageSelection,
    excludedIds,
    isCurrentPageAllChecked,
    isIndeterminate,
    handleCheckboxChange,
    handleCheckboxAll,
    handleSelectAllCrossPage,
    handleClearSelection,
  } = useTableCheckbox(instanceList, 'id', totalRef);

  // Watch 删除实例时清理失效选择，MODIFIED 时把已选对象同步为最新投影。
  watch(
    allInstances,
    instances => {
      const instanceMap = new Map(instances.map(instance => [instance.id, instance]));
      selections.value = selections.value
        .map(selection => instanceMap.get(selection.id))
        .filter((selection): selection is AppInstanceOutputObj => Boolean(selection));
      excludedIds.value = new Set([...excludedIds.value].filter(instanceID => instanceMap.has(instanceID)));
    },
    { deep: true },
  );

  // 跨环境禁用逻辑
  const isCheckboxDisabled = computed(() => {
    if (!props.selectedEnvName) return false;
    return props.envName !== props.selectedEnvName;
  });
  const isSelectAllDisabled = computed(() => isCheckboxDisabled.value);

  const selectedCount = computed(() =>
    isCrossPageSelection.value ? displayTotal.value - excludedIds.value.size : selections.value.length,
  );
  const isAllSelected = computed(() => selectedCount.value === displayTotal.value && displayTotal.value > 0);

  // 表头 Checkbox 点击
  // 统一处理表头复选框的全选与取消全选。
  function handleHeaderCheckboxClick() {
    if (isCurrentPageAllChecked.value) {
      handleCheckboxAll({ checked: false });
    } else {
      handleSelectCurrentPage();
    }
  }

  // 本页全选
  function handleSelectCurrentPage() {
    if (isCrossPageSelection.value) {
      handleClearSelection();
    }
    handleCheckboxAll({ checked: true });
  }

  // 选中变化时通知父组件
  watch(
    [selections, isCrossPageSelection, () => excludedIds.value.size],
    () => {
      const effectiveSelections = isCrossPageSelection.value
        ? allInstances.value.filter(item => !excludedIds.value.has(item.id))
        : selections.value;
      emit('selection-change', {
        envName: props.envName,
        selections: effectiveSelections,
      });
    },
    { deep: true },
  );

  watch(isAutoScaleEnabled, enabled => updateAutoScalePolling(enabled), { immediate: true });

  defineExpose({
    clearSelections: handleClearSelection,
    getSelections: () =>
      isCrossPageSelection.value
        ? allInstances.value.filter(item => !excludedIds.value.has(item.id))
        : selections.value,
    selectedCount,
    isAllSelected,
    isCrossPageSelection,
    getTotal: () => displayTotal.value,
    isCollapsed,
    loadInstances,
    resetPage,
    getVxeTableInstance: () => tableRef.value?.getVxeTableInstance?.(),
  });
</script>

<style lang="postcss" scoped>
  .env-instance-table:first-child {
    .env-header {
      border-top: 1px solid #e8eaec;
    }
  }
  .env-header {
    border: 1px solid #e8eaec;
    border-top: none;
    border-bottom: none;

    &.collapsed {
      border-bottom: 1px solid #e8eaec;
    }

    &:hover {
      background-color: #eaebf0;
    }
  }

  .env-table-body {
    :deep(.bk-table) {
      border: none;
      border-radius: 0;
    }
    :deep(.bk-table-head) {
      th {
        background-color: #fafbfd;
      }
    }
    :deep(.bk-vxe-table-pagination-wrapper) {
      padding: 5px 16px;
    }
    :deep(.vxe-table--body-prepend-wrapper) {
      border: none;
    }
  }
</style>
