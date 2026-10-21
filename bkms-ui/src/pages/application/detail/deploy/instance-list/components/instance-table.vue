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
        :pagination="displayPagination"
        :row-config="{
          keyField: 'id',
          isHover: true,
          isCurrent: true,
        }"
        :settings="settings"
        :show-overflow="false"
        :show-settings="true"
        :sort-config="{ remote: true }"
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
          :width="64"
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
          class-name="instance-name-col"
          field="id"
          :label="$t('实例')"
          min-width="120"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <div class="instance-name">
              {{ row.id || '--' }}
              <Copy
                v-if="row.id"
                class="instance-name-copy"
                :title="$t('复制')"
                @click.stop="copyText(row.id)"
              />
            </div>
          </template>
        </TableColumn>

        <!-- 状态列：实例运行状态、北极星注册状态 -->
        <TableColumn
          class-name="instance-status-col"
          field="status"
          :label="$t('状态')"
          min-width="200"
        >
          <template #header>
            <InstanceGroupFilter
              v-if="showFilter"
              :groups="statusFilterGroups"
              :label="$t('状态')"
              @change="handleFilterChange"
            />
            <span v-else>{{ $t('状态') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <div class="multi-line-cell">
              <span class="cell-label">{{ $t('实例') }}</span>
              <StatusIcon
                class="instance-status"
                emphasized
                :message="row.status !== 'Running' ? row.message : ''"
                :size="12"
                :status="row.status"
              />
              <span class="cell-label">{{ $t('北极星') }}</span>
              <span>
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
                  <div class="polaris-status">
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
              </span>
            </div>
          </template>
        </TableColumn>

        <!-- 就绪状态 -->
        <TableColumn
          field="isHealthy"
          :label="$t('就绪')"
          min-width="68"
        >
          <template #header>
            <InstanceGroupFilter
              v-if="showFilter"
              :groups="readinessFilterGroups"
              :label="$t('就绪')"
              :tooltip="$t('实例是否可以接收流量')"
              @change="handleFilterChange"
            />
            <span
              v-else
              v-bk-tooltips="$t('实例是否可以接收流量')"
              class="border-b border-dashed border-[#979ba5]"
            >
              {{ $t('就绪') }}
            </span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <span :class="row.isHealthy ? 'text-[#2DCB56]' : 'text-[#FF9C01]'">
              {{ row.isHealthy ? $t('是') : $t('否') }}
            </span>
          </template>
        </TableColumn>

        <!-- 镜像 Tag -->
        <TableColumn
          field="image"
          :label="$t('镜像 Tag')"
          min-width="120"
        >
          <template #header>
            <InstanceGroupFilter
              v-if="showFilter"
              :groups="imageFilterGroups"
              :label="$t('镜像 Tag')"
              @change="handleFilterChange"
            />
            <span v-else>{{ $t('镜像 Tag') }}</span>
          </template>
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <MiddleEllipsis
              v-bk-tooltips="row.image"
              :text="row.image?.split(':')?.pop() || '--'"
            />
          </template>
        </TableColumn>

        <TableColumn
          class-name="instance-publish-col"
          field="latestPublish"
          :label="$t('热更新')"
          min-width="120"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <Popover
              v-if="row.latestPublish"
              placement="top"
              :popover-delay="[80, 80]"
              theme="light"
              :width="560"
            >
              <span class="publish-status">
                <StatusDotIcon
                  v-if="row.latestPublish.status === 'success' || row.latestPublish.status === 'failed'"
                  :icon="row.latestPublish.status === 'success' ? 'normal' : 'abnormal'"
                  :size="12"
                />
                <span class="publish-status-text">{{ getPublishStatusLabel(row.latestPublish.status) }}</span>
              </span>
              <template #content>
                <div class="w-full min-w-0 px-[4px]">
                  <div class="mb-[12px] flex items-center text-[14px] whitespace-nowrap">
                    <span class="font-bold text-[#313238] shrink-0">{{ $t('热更新') }}</span>
                    <span
                      class="text-[#979BA5] text-[12px] ml-[10px] truncate"
                      :title="row.id"
                      >{{ row.id }}</span
                    >
                  </div>
                  <div class="text-[12px] text-[#4D4F56] mb-[12px]">
                    <i class="bkms-icon bkms-icon-circle-info text-[14px] mr-[4px]"></i>
                    {{ $t('展示该实例最近一次通过 bkms-cli 的热更新概要') }}
                  </div>
                  <div class="py-[4px]">
                    <DetailItem
                      :label="$t('二进制')"
                      :label-width="70"
                    >
                      <div
                        v-if="row.latestPublish.binaryName"
                        class="group inline-flex items-center gap-[4px]"
                      >
                        <span>{{ row.latestPublish.binaryName }}</span>
                        <Copy
                          class="shrink-0 cursor-pointer text-[#3A84FF] opacity-0 transition group-hover:opacity-100"
                          :title="$t('复制')"
                          @click.stop="copyText(row.latestPublish.binaryName)"
                        />
                      </div>
                      <span v-else>--</span>
                    </DetailItem>
                    <DetailItem
                      :label="$t('摘要')"
                      :label-width="70"
                    >
                      <Popover
                        v-if="row.latestPublish.md5"
                        disable-teleport
                        ext-cls="publish-nested-popover"
                        placement="top"
                        theme="dark"
                      >
                        <div class="group inline-flex items-center gap-[4px]">
                          <span class="cursor-pointer border-b border-dashed border-[#979ba5]">
                            {{ row.latestPublish.md5.slice(0, 8) }}
                          </span>
                          <Copy
                            class="shrink-0 cursor-pointer text-[#3A84FF] opacity-0 transition group-hover:opacity-100"
                            :title="$t('复制')"
                            @click.stop="copyText(row.latestPublish.md5)"
                          />
                        </div>
                        <template #content>
                          <div class="break-all leading-[20px]">{{ row.latestPublish.md5 }}</div>
                        </template>
                      </Popover>
                      <span v-else>--</span>
                    </DetailItem>
                    <DetailItem
                      :label="$t('操作人')"
                      :label-width="70"
                    >
                      {{ row.latestPublish.operator || '--' }}
                    </DetailItem>
                    <DetailItem
                      :label="$t('更新时间')"
                      :label-width="70"
                    >
                      {{ row.latestPublish.updatedAt ? formatTimeByTimezone(row.latestPublish.updatedAt) : '--' }}
                    </DetailItem>
                    <DetailItem
                      :label="$t('状态')"
                      :label-width="70"
                    >
                      <span class="inline-flex items-center">
                        <StatusDotIcon
                          v-if="row.latestPublish.status === 'success' || row.latestPublish.status === 'failed'"
                          :icon="row.latestPublish.status === 'success' ? 'normal' : 'abnormal'"
                          :size="12"
                        />
                        {{ getPublishResultLabel(row.latestPublish.status) }}
                      </span>
                    </DetailItem>
                    <DetailItem
                      v-if="row.latestPublish.status === 'failed' && row.latestPublish.message"
                      class="fail-reason-item"
                      :label="$t('失败原因')"
                      :label-width="70"
                    >
                      <Popover
                        disable-teleport
                        ext-cls="publish-nested-popover"
                        placement="top"
                        theme="dark"
                        :width="400"
                      >
                        <div class="group flex min-w-0 items-center gap-[4px]">
                          <span class="block min-w-0 truncate">{{ row.latestPublish.message }}</span>
                          <Copy
                            class="shrink-0 cursor-pointer text-[#3A84FF] opacity-0 transition group-hover:opacity-100"
                            :title="$t('复制')"
                            @click.stop="copyText(row.latestPublish.message)"
                          />
                        </div>
                        <template #content>
                          <div class="break-all leading-[20px]">{{ row.latestPublish.message }}</div>
                        </template>
                      </Popover>
                    </DetailItem>
                  </div>
                </div>
              </template>
            </Popover>
            <span v-else>--</span>
          </template>
        </TableColumn>

        <!-- 资源规格列：主容器 CPU / 内存规格，悬浮展示完整的 Requests/Limits -->
        <TableColumn
          field="resources"
          :label="$t('资源规格')"
          min-width="100"
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

        <!-- 网络列：Pod IP 与所在节点 IP -->
        <TableColumn
          field="ip"
          :label="$t('网络')"
          min-width="150"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            <!-- tooltip 覆盖整个网络单元格，仅在 Pod IP 或 Node IP 发生省略时启用。 -->
            <div
              v-bk-tooltips="{
                content: getNetworkTooltip(row),
                delay: 100,
                disabled: !isNetworkOverflow(row),
              }"
              class="multi-line-cell"
              @mouseenter="handleNetworkMouseEnter($event, row)"
              @mouseleave="handleNetworkMouseLeave"
            >
              <span class="cell-label">Pod IP</span>
              <HoverCopy
                class="network-hover-copy"
                :copy-value="row.ip || ''"
                :disable-overflow-tooltip="true"
                :text="row.ip || '--'"
              />
              <span class="cell-label">Node IP</span>
              <HoverCopy
                class="network-hover-copy"
                :copy-value="row.nodeIP || ''"
                :disable-overflow-tooltip="true"
                :text="row.nodeIP || '--'"
              />
            </div>
          </template>
        </TableColumn>

        <!-- 重启次数、运行时长分列：数值短，合并后反而看不清 -->
        <TableColumn
          field="restartCount"
          label="Restart"
          min-width="104"
          show-overflow="tooltip"
          sortable
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.restartCount ?? '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="age"
          label="Age"
          min-width="80"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: AppInstanceOutputObj }">
            {{ row.age || '--' }}
          </template>
        </TableColumn>

        <!-- 右侧固定，页面变窄出现横向滚动时操作列仍留在视口内 -->
        <TableColumn
          class-name="instance-actions-col"
          fixed="right"
          header-class-name="instance-actions-col"
          :label="$t('操作')"
          :width="108"
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
                @click.stop="
                  emit('row-action', { action: 'log' as InstanceRowAction, envName: props.envName, instance: row })
                "
              >
                {{ $t('日志') }}
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
                v-if="!isEnglish"
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
              <!-- 不共用 is-show：固定列会再渲染一份单元格，共用状态会同时弹出两个菜单 -->
              <Dropdown
                placement="bottom-end"
                :popover-options="{ clickContentAutoHide: true }"
                trigger="click"
              >
                <i class="row-more-trigger bkms-icon bkms-icon-more-fill"></i>
                <template #content>
                  <Dropdown.DropdownMenu>
                    <Dropdown.DropdownItem
                      v-bk-tooltips="{
                        content: isFederation
                          ? $t('联邦集群不支持灰度操作')
                          : $t('仅支持实例状态为 Running、Pending 的实例'),
                        disabled: canRowGray(row),
                        placement: 'left',
                      }"
                      :class="{ 'is-disabled': !canRowGray(row) }"
                      @click="canRowGray(row) && emitRowAction('gray', row)"
                    >
                      {{ $t('灰度') }}
                    </Dropdown.DropdownItem>
                    <Dropdown.DropdownItem @click="emitRowAction('monitor', row)">
                      {{ $t('监控') }}
                    </Dropdown.DropdownItem>
                    <Dropdown.DropdownItem
                      v-if="isEnglish"
                      v-bk-tooltips="{
                        content: $t('未获取到北极星信息，无法调整权重'),
                        disabled: canAdjustInstanceWeight(row),
                        placement: 'left',
                      }"
                      :class="{ 'is-disabled': !canAdjustInstanceWeight(row) }"
                      @click="canAdjustInstanceWeight(row) && emitRowAction('weight', row)"
                    >
                      {{ $t('调整权重') }}
                    </Dropdown.DropdownItem>
                  </Dropdown.DropdownMenu>
                </template>
              </Dropdown>
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
  import { AngleDownLine, Copy, RightShape } from 'bkui-vue/lib/icon';
  import { AppInstanceOutputObj } from '~/@types/v1/instance';
  import { copyText, formatTimeByTimezone } from '~/common/util';
  import HoverCopy from '~/components/hover-copy.vue';
  import MiddleEllipsis from '~/components/middle-ellipsis.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import StatusIcon from '~/components/status-icon.vue';
  import TableException from '~/components/table-exception.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useGPAConfigPolling } from '~/composables/use-gpa-config-polling';
  import { useLanguage } from '~/composables/use-language';
  import { useResourceSpecDisplay } from '~/composables/use-resource-spec-display';
  import useTableCheckbox from '~/composables/use-table-checkbox';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useTableSettings } from '~/composables/use-table-settings';
  import AutoScaleTag from '~/pages/application/detail/components/auto-scale-tag.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import {
    type RestartSortOrder,
    paginateInstances,
    resolveInstanceSourceMode,
    sortInstancesByRestart,
  } from '../composables/instance-watch-utils';
  import { useInstanceListWatch } from '../composables/use-instance-list-watch';
  import {
    canAdjustInstanceWeight,
    canInstanceGrayDeploy,
    canLogin,
    canViewLog,
    isPolarisHealthy,
  } from '../instance-utils';
  import InstanceGroupFilter, { type FilterGroup } from './instance-group-filter.vue';

  import type {
    InstanceDataLoadedPayload,
    InstanceListSourceMode,
    InstanceRowAction,
    InstanceRowActionPayload,
    InstanceSelectionChangePayload,
    InstanceTableMode,
  } from '../types';
  import type { FilterItem } from '~/components/custom-filter.vue';

  interface Props {
    /** 外部传入数据（单环境模式），提供后跳过内部 loadInstances */
    data?: AppInstanceOutputObj[];
    envDisplayName?: string;
    envKind?: string;
    envName: string;
    envType?: string;
    /** 列筛选项数据 */
    filterOptions?: Record<string, FilterItem[]>;
    /** 内部数据源模式；多环境父组件按联邦/非联邦明确传入。 */
    instanceSourceMode?: InstanceListSourceMode;
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
  const { isEnglish } = useLanguage();

  const { getResourceText, getResourceTips } = useResourceSpecDisplay();

  // 列设置：资源规格、热更新列默认不勾选，用户可在表格右上角列设置中主动开启。
  // 列勾选与行高（size）偏好均持久化，刷新后恢复。
  // 所有环境共用列设置，任一环境的配置调整都会同步到其他环境表格。
  // 列合并后字段含义已变，换新 id 避免旧的隐藏/显示偏好错配到合并列上。
  const tableSettingsId = 'instance-table-v2';
  const { settings, handleSettingChange } = useTableSettings(tableSettingsId, {
    defaultChecked: ['id', 'status', 'isHealthy', 'image', 'ip', 'restartCount', 'age'],
    disabled: ['id'],
  });

  const statusFilterGroups = computed<FilterGroup[]>(() => [
    { field: 'status', label: window.i18n.t('实例状态'), options: props.filterOptions.status ?? [] },
    { field: 'polarisStatus', label: window.i18n.t('北极星状态'), options: props.filterOptions.polarisStatus ?? [] },
  ]);
  const readinessFilterGroups = computed<FilterGroup[]>(() => [
    { field: 'isHealthy', label: window.i18n.t('就绪'), options: props.filterOptions.isHealthy ?? [] },
  ]);
  function canRowGray(row: AppInstanceOutputObj) {
    return !props.isFederation && canInstanceGrayDeploy(row);
  }

  function emitRowAction(action: InstanceRowAction, instance: AppInstanceOutputObj) {
    emit('row-action', { action, envName: props.envName, instance });
  }

  const imageFilterGroups = computed<FilterGroup[]>(() => [
    { field: 'image', label: window.i18n.t('镜像 Tag'), options: props.filterOptions.image ?? [] },
  ]);

  type NetworkRowKey = string;
  const hoveredNetworkRowKey = ref<NetworkRowKey | null>(null);
  let hoveredNetworkCell: HTMLElement | null = null;
  let networkResizeObserver: ResizeObserver | undefined;

  function getNetworkRowKey(row: AppInstanceOutputObj): NetworkRowKey {
    return row.id ?? `${row.ip ?? ''}|${row.nodeIP ?? ''}`;
  }

  function getNetworkTooltip(row: AppInstanceOutputObj) {
    return `Pod IP: ${row.ip || '--'}\nNode IP: ${row.nodeIP || '--'}`;
  }

  function getPublishResultLabel(status?: string) {
    if (status === 'success') return window.i18n.t('成功');
    if (status === 'failed') return window.i18n.t('失败');
    return status || '--';
  }

  function getPublishStatusLabel(status?: string) {
    if (status === 'success') return window.i18n.t('已更新');
    if (status === 'failed') return window.i18n.t('更新失败');
    return status || '--';
  }

  function handleNetworkMouseEnter(event: MouseEvent, row: AppInstanceOutputObj) {
    const cell = event.currentTarget;
    if (!(cell instanceof HTMLElement)) return;

    networkResizeObserver?.disconnect();
    hoveredNetworkCell = cell;
    updateHoveredNetworkCell(cell, row);

    if (typeof ResizeObserver !== 'undefined') {
      networkResizeObserver = new ResizeObserver(() => {
        if (hoveredNetworkCell === cell) updateHoveredNetworkCell(cell, row, true);
      });
      networkResizeObserver.observe(cell);
    }
  }

  function handleNetworkMouseLeave(event: MouseEvent) {
    // 溢出状态变化时会补发合成 mouseleave，仅让 tooltip 隐藏，不能停止当前单元格的尺寸监听。
    if (!event.isTrusted) return;

    networkResizeObserver?.disconnect();
    networkResizeObserver = undefined;
    hoveredNetworkCell = null;
    hoveredNetworkRowKey.value = null;
  }

  function isNetworkCellOverflow(cell: HTMLElement) {
    return Array.from(cell.querySelectorAll<HTMLElement>('.network-hover-copy .text-ov')).some(
      valueCell =>
        valueCell.scrollWidth > valueCell.clientWidth + 1 || valueCell.scrollHeight > valueCell.clientHeight + 1,
    );
  }

  function isNetworkOverflow(row: AppInstanceOutputObj) {
    return hoveredNetworkRowKey.value === getNetworkRowKey(row);
  }

  // 调整列宽时重新测量当前悬停单元格，并同步 tooltip 的显示/隐藏状态。
  function updateHoveredNetworkCell(cell: HTMLElement, row: AppInstanceOutputObj, refreshTooltip = false) {
    const nextKey = isNetworkCellOverflow(cell) ? getNetworkRowKey(row) : null;
    const changed = hoveredNetworkRowKey.value !== nextKey;
    hoveredNetworkRowKey.value = nextKey;

    // 指令只在 mouseenter 时触发，状态变化发生在悬停期间时需要补发事件让 tooltip 重新处理。
    if (refreshTooltip && changed) {
      cell.dispatchEvent(new MouseEvent(nextKey ? 'mouseenter' : 'mouseleave'));
    }
  }

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
  /** 多环境内部数据源：优先用父组件明确传入的模式，否则按联邦标识回退；外部数据模式不会启用。 */
  const instanceSourceMode = computed<InstanceListSourceMode>(
    () => props.instanceSourceMode ?? resolveInstanceSourceMode(Boolean(props.isFederation)),
  );

  const {
    clear: clearWatchedInstances,
    instances: watchedInstances,
    lastError: watchError,
    refresh: refreshWatch,
  } = useInstanceListWatch({
    enabled: () => !isExternalData.value && !isCollapsed.value,
    getMode: () => instanceSourceMode.value,
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

  // 选择提示条改变 prepend 高度后，同步重算左右固定列的位置。
  watch(
    () => props.mode === 'single' && hasSelection.value,
    () => {
      tableRef.value?.getVxeTableInstance?.()?.recalculate();
    },
    { flush: 'post' },
  );

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
  :deep(.publish-nested-popover.bk-popover.bk-pop2-content .bk-pop2-arrow) {
    background-color: #26323d !important;
  }

  .fail-reason-item :deep(> div:last-child) {
    min-width: 0;
  }

  .row-more-trigger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    font-size: 16px;
    color: #979ba5;
    cursor: pointer;
    border-radius: 2px;

    &:hover {
      color: #3a84ff;
      background-color: #f0f1f5;
    }
  }

  .is-disabled {
    color: #c4c6cc;
    cursor: not-allowed;
  }

  .instance-name {
    padding: 8px 0;
    line-height: 20px;
    word-break: break-all;
    white-space: normal;
  }

  .instance-name-copy {
    display: none !important;
    margin-left: 4px;
    vertical-align: -3px;
    color: #3a84ff;
    cursor: pointer;
  }

  .instance-name:hover .instance-name-copy {
    display: inline-flex !important;
  }

  :deep(.network-hover-copy > div) {
    border-bottom: none !important;
  }

  .multi-line-cell {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    gap: 2px 8px;
    align-items: center;
    padding: 8px 0;
    line-height: 20px;

    .cell-label {
      color: #979ba5;
      text-align: right;
    }

    > :nth-child(even) {
      min-width: 0;
      overflow: hidden;
    }

    > .instance-status {
      min-width: max-content;
      overflow: visible;
    }

    .polaris-status {
      display: inline-flex;
      align-items: center;
      white-space: nowrap;
      cursor: pointer;
    }
  }

  .publish-status {
    display: inline-flex;
    align-items: center;
    white-space: nowrap;
    cursor: default;
  }

  .publish-status-text {
    line-height: 16px;
    border-bottom: 1px dashed #979ba5;
  }

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
    /* 列宽不够时在单元格内省略，避免把表格撑出横向滚动条 */
    :deep([class~='vxe-body--column'] .vxe-cell) {
      overflow: hidden;
    }

    :deep([class~='vxe-header--column'][class~='col--sortable'] .vxe-cell) {
      white-space: nowrap;
    }

    :deep(.instance-name-col .vxe-cell) {
      overflow: visible;
      white-space: normal;
    }

    :deep(.instance-status-col .vxe-cell) {
      overflow: visible;
    }

    :deep(.instance-actions-col .vxe-cell) {
      padding-right: 8px !important;
      padding-left: 8px !important;
    }

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
    :deep([class~='vxe-table--body-prepend-wrapper']) {
      border: none;
    }
  }
</style>
