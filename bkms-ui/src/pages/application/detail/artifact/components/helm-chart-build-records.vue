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
  <Sideslider
    :is-show="isShow"
    :title="t('版本构建记录')"
    :width="960"
    @closed="handleClose"
  >
    <div class="px-[24px] py-[20px]">
      <FlexRow class="mb-[16px]">
        <template #right>
          <div class="flex items-center">
            <Input
              v-model.trim="searchValue"
              class="w-[400px]"
              clearable
              :placeholder="
                createPlaceholder({
                  labels: ['版本号', '操作人'],
                })
              "
              type="search"
            />
          </div>
        </template>
      </FlexRow>
      <Table
        auto-resize
        class="w-full build-records-table"
        :data="buildList"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        sync-resize
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="handleClearFilters"
            @refresh="fetchBuildList"
          />
        </template>
        <TableColumn
          field="buildNum"
          fixed="left"
          :label="t('构建号')"
          show-overflow="tooltip"
          :width="80"
        >
          <template #default="{ row }">
            <div
              class="cursor-pointer flex items-center"
              @click.stop="handleGotoPipeline(row)"
            >
              <span
                v-if="row.buildNum"
                :class="[getColor(row.status)]"
              >
                {{ `#${row.buildNum}` }}
              </span>
              <span v-else>--</span>
              <i
                v-if="isRunning(row.status)"
                class="bkms-icon bkms-icon-half-circle text-[12px] ml-[6px] animate-spin"
                :class="[getColor(row.status)]"
              ></i>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="chartVersion"
          :label="t('版本号')"
          show-overflow="tooltip"
          :width="80"
        >
          <template #default="{ row }">
            {{ row.chartVersion || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="sourceMaterial"
          :label="t('源材料')"
          show-overflow="tooltip"
          :width="150"
        >
          <template #default="{ row }">
            <div
              v-if="row.sourceMaterial"
              class="flex items-center"
            >
              <span class="bkms-icon bkms-icon-branchs mr-[5px] text-[14px]"></span>
              <span>{{ row.sourceMaterial.revision }}</span>
              <span
                v-if="row.sourceMaterial.commitID"
                class="bkms-icon bkms-icon-commit mx-[5px] text-[16px]"
              ></span>
              <Button
                v-if="row.sourceMaterial.commitID"
                text
                theme="primary"
                @click.stop="handleToCommit(row.sourceMaterial.commitUrl)"
              >
                {{ row.sourceMaterial.commitID.slice(0, 8) }}
              </Button>
            </div>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <TableColumn
          field="operator"
          :label="t('触发人')"
          show-overflow="tooltip"
          :width="120"
        >
          <template #default="{ row }">
            <span class="text-[12px]">{{ row?.operator || '--' }}</span>
          </template>
        </TableColumn>
        <TableColumn
          field="startedAt"
          :label="t('构建开始时间')"
          show-overflow="tooltip"
          :width="180"
        >
          <template #default="{ row }">
            {{ row.startedAt ? formatDateString(row.startedAt) : '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="endedAt"
          :label="t('构建结束时间')"
          show-overflow="tooltip"
          :width="180"
        >
          <template #default="{ row }">
            {{
              row.endedAt && !isRunning(row.status) && String(row.endedAt) !== '0001-01-01T00:00:00Z'
                ? formatDateString(row.endedAt)
                : '--'
            }}
          </template>
        </TableColumn>
        <TableColumn
          field="constructionTime"
          :label="t('构建耗时')"
          show-overflow="tooltip"
          :width="120"
        >
          <template #default="{ row }">
            {{ row?.constructionTime || '--' }}
          </template>
        </TableColumn>
      </Table>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, Input, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { HelmChartBuildRecordOutputObj } from '~/@types/v1/helm-charts';
  import { HelmChartsService } from '~/api/modules/v1';
  import { gotoPipelineDetail } from '~/common/util';
  import useInterval from '~/composables/use-interval';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  export type BuildStatus = 'failed' | 'running' | 'success';

  type BuildListType = {
    buildID: string;
    buildNum: string;
    chartVersion: string;
    constructionTime: string;
    endedAt: string;
    operator: string;
    pipelineID: string;
    sourceMaterial: null | {
      commitID: string;
      commitUrl: string;
      revision: string;
    };
    startedAt: string;
    status: string;
  };

  const emit = defineEmits<{
    (e: 'build-status-change', status: BuildStatus, num: string): void;
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();
  const { formatDateString, calculateTimeDifference } = useTime();
  const { createPlaceholder } = useSearchPlaceholder();
  const { start, stop } = useInterval(fetchBuildList, 10000);

  type statusType = 'FAILED' | 'RUNNING' | 'SUCCED' | 'UNKNOWN';
  const statusColor: Record<statusType, string> = {
    RUNNING: 'text-[#3d86ff]',
    SUCCED: 'text-[#30b061]',
    FAILED: 'text-[#ec4343]',
    UNKNOWN: 'text-[#ffb848]',
  };
  const failedStatus = ['failed', 'pollingBroken'];

  function getColor(status: string) {
    if (status === 'running') return statusColor.RUNNING;
    if (status === 'success') return statusColor.SUCCED;
    if (failedStatus.includes(status)) return statusColor.FAILED;
    return statusColor.UNKNOWN;
  }

  function isRunning(status: string) {
    return status === 'running';
  }

  const isShow = defineModel<boolean>('isShow');

  const buildList = ref<BuildListType[]>([]);
  const count = ref(0);
  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    buildList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: fetchBuildList,
      onPageSizeChange: fetchBuildList,
    },
    count,
  );

  const searchValue = ref('');
  const debounceSearch = useDebounce(searchValue, 300);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  /** 等待出现的构建 ID（模式1），清空则为模式2 */
  const pendingBuildID = ref('');

  /** 轮询策略：决定是否继续轮询，独立于数据获取 */
  function adjustPolling() {
    if (pendingBuildID.value) {
      // 模式1：等待特定 buildID 出现
      const found = buildList.value.some(item => item.buildID === pendingBuildID.value);
      if (found) {
        pendingBuildID.value = '';
        // 找到了，落入模式2判断（不 return）
      } else {
        return; // 没找到，继续轮询（useInterval 的 timerFn 自动调度下一次）
      }
    }
    // 模式2：按 running 状态决定是否继续轮询
    if (!buildList.value.some(item => isRunning(item.status))) {
      stop();
    }
  }

  /** 第一条构建状态变化时通知父组件 */
  function emitStatusChange() {
    if (!buildList.value.length) return;
    const { status, buildNum } = buildList.value[0];
    if (status === 'success') emit('build-status-change', 'success', buildNum);
    else if (failedStatus.includes(status)) emit('build-status-change', 'failed', buildNum);
    else if (status === 'running') emit('build-status-change', 'running', buildNum);
  }

  async function fetchBuildList() {
    if (!appDetailStore.appID) return;
    try {
      const res = await HelmChartsService.listHelmChartBuildRecords({
        appID: appDetailStore.appID,
        keyword: debounceSearch.value,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      });
      clearErrorType();
      count.value = Number(res?.count) || 0;
      buildList.value = (res.results || []).map(mapBuildRecord);
      emitStatusChange();
      adjustPolling();
    } catch {
      setTypeToError();
      buildList.value = [];
      count.value = 0;
    }
  }

  function handleClearFilters() {
    searchValue.value = '';
    handleResetPage();
    fetchBuildList();
  }

  function handleClose() {
    isShow.value = false;
    // 关闭侧栏但不断轮询，仍需通过轮询获取最新状态来通知父组件
  }

  function handleGotoPipeline(row: BuildListType) {
    const bkCIProjectID = spaceStore.workspaceDetail?.bkSystems?.bkCIProjectID ?? '';
    gotoPipelineDetail(bkCIProjectID, row.pipelineID, row.buildID);
  }

  function handleToCommit(url: string) {
    if (url) window.open(url, '_blank');
  }

  /** 将 API 原始数据映射为表格展示数据 */
  function mapBuildRecord(item: HelmChartBuildRecordOutputObj): BuildListType {
    return {
      buildNum: item?.num ?? '',
      chartVersion: item.chartVersion || '',
      sourceMaterial:
        item?.extras && item.extras?.BK_CI_GIT_REPO_URL
          ? {
              revision: item.params?.BKMS_REPO_REVISION || '',
              commitID: item.extras?.BK_CI_GIT_REPO_HEAD_COMMIT_ID || '',
              commitUrl: `${item.extras.BK_CI_GIT_REPO_URL.replace(/\.git$/, '')}/commit/${item.extras?.BK_CI_GIT_REPO_HEAD_COMMIT_ID}`,
            }
          : null,
      operator: item.operator || '',
      startedAt: item.startedAt ? formatDateString(item.startedAt) : '--',
      endedAt:
        item.endedAt && !isRunning(item?.status ?? '') && String(item.endedAt) !== '0001-01-01T00:00:00Z'
          ? formatDateString(item.endedAt)
          : '--',
      constructionTime: calculateTimeDifference(String(item.startedAt), String(item.endedAt)),
      status: item?.status ?? '',
      pipelineID: item?.pipelineID ?? '',
      buildID: item?.buildID ?? '',
    };
  }

  /** 暴露给父组件：触发轮询等待特定 buildID 出现 */
  function triggerPolling(buildID: string) {
    pendingBuildID.value = buildID;
    handleClearFilters();
    start();
  }

  watch(debounceSearch, () => {
    handleResetPage();
    fetchBuildList();
  });

  // 首次加载获取一次，根据结果决定是否开启轮询
  onMounted(() => fetchBuildList());

  defineExpose({ triggerPolling });
</script>

<style lang="postcss" scoped>
  :deep(.build-records-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }
  }
</style>
