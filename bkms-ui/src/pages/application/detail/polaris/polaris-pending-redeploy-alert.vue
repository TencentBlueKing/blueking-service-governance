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
  <div
    v-if="pendingRedeployRows.length"
    class="mb-[16px] overflow-hidden border border-[#F9D090] bg-[#FDF4E8] hover:bg-[#FDEED8]"
  >
    <div
      class="flex h-[32px] cursor-pointer items-center px-[10px] text-[12px] text-[#63656E]"
      @click="isExpanded = !isExpanded"
    >
      <i class="bkms-icon bkms-icon-warning-circle mt-[-1px] mr-[8px] text-[14px] text-[#F59500]"></i>
      <i18n-t keypath="共有 {0} 个环境的北极星更新，需要重新部署后方可生效">
        <span class="mx-[3px] font-bold">{{ pendingRedeployEnvCount }}</span>
      </i18n-t>
      <Button
        class="ml-[8px]"
        :loading="refreshing"
        text
        theme="primary"
        @click.stop="handleRefresh"
        @mousedown.stop
      >
        <i class="bkms-icon bkms-icon-refresh mr-[2px]"></i>
        {{ $t('刷新') }}
      </Button>
      <AngleRight
        :class="['ml-auto text-[20px] text-[#979BA5] transition-transform duration-200', isExpanded ? 'rotate-90' : '']"
      />
    </div>
    <div
      v-if="isExpanded"
      class="px-[32px] pb-[12px]"
    >
      <Table
        class="polaris-pending-redeploy-table bg-[#fff]"
        :data="pendingRedeployRows"
        :merge-cells="pendingRedeployMergeCells"
        :row-class-name="getPendingRedeployRowClassName"
        :row-config="{
          isHover: true,
          isCurrent: false,
        }"
      >
        <TableColumn
          field="envName"
          :label="$t('待部署环境')"
          min-width="240"
        >
          <template #default="{ row }">
            <div class="flex items-center">
              <span
                v-bk-tooltips="row.envDisplayName"
                class="truncate"
              >
                {{ row.envDisplayName }}
              </span>
              <Tag
                v-if="row.envType && envTypeMap[row.envType]"
                :class="['ml-[8px] shrink-0', envTypeTagClassMap[row.envType]]"
                size="small"
              >
                {{ envTypeMap[row.envType].name }}
              </Tag>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="polarisName"
          :label="$t('北极星服务名')"
          min-width="220"
        >
          <template #default="{ row }">
            {{ row.polarisName || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="changes"
          :label="$t('修改内容')"
          min-width="360"
        >
          <template #default="{ row }">
            <div
              class="pending-redeploy-change-cell flex flex-col justify-center text-[12px] leading-[20px] text-[#63656E]"
            >
              <template v-if="row.changes.length">
                <div
                  v-for="change in row.changes"
                  :key="change.key"
                >
                  <span>{{ change.label }}：</span>
                  <template v-if="change.oldValue === undefined">
                    <span class="font-bold text-[#313238]">{{ formatRedeployValue(change.newValue) }}</span>
                  </template>
                  <template v-else>
                    <span>{{ formatRedeployValue(change.oldValue) }}</span>
                    <span class="mx-[8px]">→</span>
                    <span class="font-bold text-[#313238]">{{ formatRedeployValue(change.newValue) }}</span>
                  </template>
                </div>
              </template>
              <!-- 删除状态 -->
              <span v-else-if="isPendingDeleteStatus(row.status)">
                {{ $t('解除关联（重新部署后实例将从北极星摘除）') }}
              </span>
              <span v-else>{{ $t('北极星配置待重新部署生效') }}</span>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作')"
          width="100"
        >
          <template #default="{ row }">
            <Button
              :disabled="isEnvDeploying(row.envName)"
              text
              theme="primary"
              @click="emit('go-deploy', row.envName)"
            >
              {{ isEnvDeploying(row.envName) ? $t('部署中') : $t('去部署') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Tag } from 'bkui-vue';
  import { AngleRight } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import { formatPolarisRedeployValue, getPolarisRedeployChanges } from './redeploy-utils';

  import type { PolarisRedeployChange } from './redeploy-utils';
  import type { EnvOutput } from '~/@types/v1/env';

  interface PendingRedeployRow {
    changes: PolarisRedeployChange[];
    envDisplayName: string;
    envName: string;
    envType?: string;
    polarisName?: string;
    status: string;
  }

  type PolarisEnvStateWithStatus = NonNullable<PolarisConfigOutputObj['envStates']>[string] & {
    status?: string;
  };

  interface TableMergeCell {
    col: number;
    colspan: number;
    row: number;
    rowspan: number;
  }

  const props = defineProps<{
    configs: PolarisConfigOutputObj[];
    deployingEnvNames: string[];
    envList: EnvOutput[];
    refreshing: boolean;
  }>();

  const emit = defineEmits<{
    (e: 'go-deploy', envName: string): void;
    (e: 'refresh'): void;
  }>();

  const isExpanded = ref(true);
  const deployingEnvNameSet = computed(() => new Set(props.deployingEnvNames));

  /** 顶部待部署提示表格数据：基于完整配置列表，避免被搜索筛选隐藏。 */
  const pendingRedeployRows = computed<PendingRedeployRow[]>(() => {
    const groupedRows = new Map<string, PendingRedeployRow[]>();
    props.configs.forEach(config => {
      Object.entries(config.envStates || {}).forEach(([envName, state]) => {
        const status = getPolarisEnvStatus((state as PolarisEnvStateWithStatus).status);
        if (!isPendingRedeployStatus(status)) return;

        const envInfo = getEnvInfo(envName);
        const rows = groupedRows.get(envName) || [];
        rows.push({
          changes: getPolarisRedeployChanges(config, envName),
          envDisplayName: envInfo.displayName,
          envName,
          envType: envInfo.envType,
          polarisName: config.polarisName,
          status,
        });
        groupedRows.set(envName, rows);
      });
    });

    return Array.from(groupedRows.values()).flat();
  });

  /** 顶部提示数量按环境去重统计，而不是按北极星配置数量统计。 */
  const pendingRedeployEnvCount = computed(() => new Set(pendingRedeployRows.value.map(row => row.envName)).size);

  /** 同一个环境下多条北极星服务记录时，合并环境列和操作列。 */
  const pendingRedeployMergeCells = computed<TableMergeCell[]>(() => {
    const mergeCells: TableMergeCell[] = [];
    let rowIndex = 0;
    while (rowIndex < pendingRedeployRows.value.length) {
      const envName = pendingRedeployRows.value[rowIndex].envName;
      let rowspan = 1;
      while (pendingRedeployRows.value[rowIndex + rowspan]?.envName === envName) {
        rowspan += 1;
      }
      if (rowspan > 1) {
        mergeCells.push({ row: rowIndex, col: 0, rowspan, colspan: 1 }, { row: rowIndex, col: 3, rowspan, colspan: 1 });
      }
      rowIndex += rowspan;
    }
    return mergeCells;
  });

  /** 统一复用北极星待部署变更值格式化逻辑，供顶部提示表格渲染。 */
  function formatRedeployValue(value?: number | string) {
    return formatPolarisRedeployValue(value);
  }

  /** 根据环境名补齐环境展示名和类型，接口无环境详情时回退展示原环境名。 */
  function getEnvInfo(envName: string) {
    const env = props.envList.find(item => item.name === envName);
    return {
      displayName: env?.displayName || envName,
      envType: env?.type,
    };
  }

  /** 根据修改内容行数返回表格行样式，控制一行 42px、两行 56px 的高度。 */
  function getPendingRedeployRowClassName({ row }: { row: PendingRedeployRow }) {
    return row.changes.length > 1 ? 'pending-redeploy-row-two-line' : 'pending-redeploy-row-one-line';
  }

  /** 兼容 envStates.status 的大小写和空格差异。 */
  function getPolarisEnvStatus(status?: string) {
    return status?.trim().toLowerCase() || '';
  }

  /** 手动刷新时阻止重复请求，并保持告警面板当前的展开状态。 */
  function handleRefresh() {
    if (props.refreshing) return;
    emit('refresh');
  }

  /** 同一环境存在任意部署中记录时，禁用该环境的重复部署入口。 */
  function isEnvDeploying(envName: string) {
    return deployingEnvNameSet.value.has(envName);
  }

  /** pendingDelete 表示 scope 已置空，重新部署后会从北极星摘除实例。 */
  function isPendingDeleteStatus(status?: string) {
    return getPolarisEnvStatus(status) === 'pendingdelete';
  }

  /** 非空状态且不是 deployed，就认为需要重新部署后生效。 */
  function isPendingRedeployStatus(status?: string) {
    const normalizedStatus = getPolarisEnvStatus(status);
    return !!normalizedStatus && normalizedStatus !== 'deployed';
  }
</script>

<style lang="postcss">
  .polaris-pending-redeploy-table {
    .vxe-body--row.pending-redeploy-row-one-line {
      height: 42px !important;
    }

    .vxe-body--row.pending-redeploy-row-two-line {
      height: 56px !important;
    }

    .vxe-body--column .vxe-cell {
      max-height: none !important;
    }

    .pending-redeploy-row-one-line .pending-redeploy-change-cell {
      min-height: 42px;
    }

    .pending-redeploy-row-two-line .pending-redeploy-change-cell {
      min-height: 56px;
    }
  }
</style>
