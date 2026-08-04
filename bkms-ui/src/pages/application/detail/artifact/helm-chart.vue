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
  <div class="w-full">
    <Skeleton
      :full-height="false"
      :loading="isLoading"
      theme="gray"
    >
      <template #loading>
        <FlexRow class="flex">
          <template #left>
            <Layout.shape
              :height="32"
              :width="120"
            />
            <Layout.shape
              class="ml-[8px]"
              :height="32"
              :width="120"
            />
          </template>
          <template #right>
            <Layout.shape
              :height="32"
              :width="400"
            />
          </template>
        </FlexRow>
        <Layout.table class="mt-[16px]" />
      </template>
      <FlexRow class="flex">
        <template #left>
          <Button
            v-bk-tooltips="{
              content: $t('应用的 HelmChart 来源非代码仓库，暂不支持该功能'),
              disabled: !isNotGitRepo,
            }"
            class="mr-[8px]"
            :disabled="isNotGitRepo"
            theme="primary"
            @click="handleCreateVersion"
          >
            <i class="bkms-icon bkms-icon-jiahao text-[14px] mr-[8px]"></i>
            {{ $t('新建版本') }}
          </Button>
          <Button
            v-bk-tooltips="{
              content: $t('应用的 HelmChart 来源非代码仓库，暂不支持该功能'),
              disabled: !isNotGitRepo,
            }"
            :disabled="isNotGitRepo"
            @click="handleViewBuildRecords"
          >
            <i class="bkms-icon bkms-icon-record text-[14px] mr-[8px]"></i>
            {{ $t('版本构建记录') }}
          </Button>
        </template>
        <template #right>
          <div class="flex">
            <Input
              v-model.trim="searchKeyword"
              class="w-[400px]"
              clearable
              :placeholder="createPlaceholder({ type: 'input', labels: ['版本号'] })"
              type="search"
              @clear="handleClearSearch"
              @enter="handleSearch"
            />
          </div>
        </template>
      </FlexRow>
      <Alert
        v-show="buildStatusInfo.status === 'running'"
        class="my-[16px]"
        theme="info"
      >
        <template #title>
          <i18n-t
            class="text-[12px]"
            keypath="{0} 版本正在构建中，{1}"
          >
            <span>{{ buildStatusInfo.version }}</span>
            <Button
              class="!underline"
              text
              theme="primary"
              @click="handleViewBuildRecords"
            >
              {{ $t('查看构建记录') }}
            </Button>
          </i18n-t>
        </template>
      </Alert>
      <Alert
        v-show="buildStatusInfo.status === 'failed'"
        class="my-[16px]"
        theme="danger"
      >
        <template #title>
          <div class="flex items-center gap-[4px]">
            {{ buildFailedTitle }}
            <Button
              class="!underline"
              text
              theme="primary"
              @click="handleViewBuildRecords"
            >
              {{ $t('查看构建记录') }}
            </Button>
          </div>
        </template>
      </Alert>
      <Table
        v-bkloading="{ loading: isLoading }"
        auto-resize
        class="mt-[16px] w-full helm-chart-table"
        :data="chartList"
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
            @refresh="fetchChartList"
          />
        </template>
        <TableColumn
          field="chartVersion"
          fixed="left"
          :label="$t('版本号')"
          show-overflow="tooltip"
          :width="160"
        >
          <template #default="{ row }">
            <Button
              v-if="row.chartVersion"
              text
              theme="primary"
              @click="handleViewVersionDetail(row)"
            >
              {{ row.chartVersion }}
            </Button>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <TableColumn
          field="createdAt"
          :label="$t('创建时间')"
          show-overflow="tooltip"
          :width="180"
        >
          <template #default="{ row }">
            {{ row.createdAt ? formatDateString(row.createdAt) : '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="digest"
          :label="$t('摘要')"
          :min-width="100"
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.digest"
              :text="formatDigest(row.digest)"
              :tooltip="row.digest"
            />
          </template>
        </TableColumn>
        <TableColumn
          field="deployedEnvs"
          :label="$t('已部署环境')"
          :min-width="200"
        >
          <template #default="{ row }">
            <div
              v-if="row.deployedEnvs?.length"
              class="flex flex-wrap gap-[4px]"
            >
              <Tag
                v-for="item in getDeployedEnvTags(row.deployedEnvs)"
                :key="item.key"
                v-bk-tooltips="{ content: getEnvTooltipContent(item.envNames) }"
                :class="envTypeTagClassMap[item.key]"
              >
                {{ item.label }}（{{ item.count }}）
              </Tag>
            </div>
            <span v-else>--</span>
          </template>
        </TableColumn>
      </Table>
      <CreateHelmChartVersionDialog
        v-model="showCreateDialog"
        :app-id="appDetailStore.appID"
        @success="handleCreateSuccess"
      />
      <HelmChartBuildRecords
        ref="buildRecordsRef"
        v-model:is-show="isShowBuildRecords"
        @build-status-change="handleBuildStatusChange"
      />
      <HelmChartVersionDetail
        v-model:is-show="isShowVersionDetail"
        :app-id="appDetailStore.appID"
        :chart-version="currentChartVersion"
      />
    </Skeleton>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Input, Message, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppHelmChartOutputObj } from '~/@types/v1/helm-charts';
  import { HelmChartsService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';

  import CreateHelmChartVersionDialog from './components/create-helm-chart-version-dialog.vue';
  import HelmChartBuildRecords from './components/helm-chart-build-records.vue';
  import HelmChartVersionDetail from './components/helm-chart-version-detail.vue';

  import type { BuildStatus } from './components/helm-chart-build-records.vue';

  const DEPLOYED_ENV_TYPES = ['development', 'test', 'staging', 'production'] as const;

  type DeployedEnvTag = {
    count: number;
    envNames: string[];
    key: (typeof DEPLOYED_ENV_TYPES)[number];
    label: string;
  };
  type DeployedEnvType = (typeof DEPLOYED_ENV_TYPES)[number];

  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString } = useTime();
  const appDetailStore = useAppDetail();
  const { t } = useI18n();

  const isNotGitRepo = computed(() => appDetailStore?.appDetail?.helmSpec?.helmSource?.repoType !== 'GitRepo');

  const showCreateDialog = ref(false);
  const isShowBuildRecords = ref(false);
  const isShowVersionDetail = ref(false);
  const currentChartVersion = ref('');
  const buildStatusInfo = ref<{ status: BuildStatus; version: string }>({ status: '' as BuildStatus, version: '' });
  const buildFailedTitle = ref('');
  const runningNotified = ref(false);
  const buildRecordsRef = ref<InstanceType<typeof HelmChartBuildRecords>>();

  const chartList = ref<AppHelmChartOutputObj[]>([]);
  const count = ref(0);
  const { pagination, pageChange, pageSizeChange } = usePageConf(
    chartList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: () => fetchChartList(),
      onPageSizeChange: () => fetchChartList(),
    },
    count,
  );

  const searchKeyword = ref('');
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchKeyword,
  });

  const isLoading = ref(false);

  /** 获取 Helm Chart 列表 */
  async function fetchChartList() {
    if (!appDetailStore.appID) return;
    isLoading.value = true;
    try {
      const res = await HelmChartsService.listAppHelmCharts({
        appID: appDetailStore.appID,
        keyword: searchKeyword.value,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      });

      chartList.value = res.results || [];
      count.value = Number(res?.count) || 0;
      clearErrorType();
    } catch {
      setTypeToError();
      chartList.value = [];
      count.value = 0;
    } finally {
      isLoading.value = false;
    }
  }

  /** 格式化摘要，截取冒号后前8位 */
  function formatDigest(digest?: string) {
    if (!digest) return '';
    return digest.includes(':') ? digest.split(':')[1].slice(0, 8) : digest.slice(0, 8);
  }

  function getDeployedEnvTags(deployedEnvs: AppHelmChartOutputObj['deployedEnvs']): DeployedEnvTag[] {
    if (!deployedEnvs?.length) return [];

    const typeEnvNamesMap: Record<DeployedEnvType, string[]> = {
      development: [],
      staging: [],
      production: [],
      test: [],
    };

    deployedEnvs.forEach(({ envName, envType }) => {
      if (!envType || !DEPLOYED_ENV_TYPES.includes(envType as DeployedEnvType)) return;
      typeEnvNamesMap[envType as DeployedEnvType].push(envName ?? '');
    });

    return DEPLOYED_ENV_TYPES.filter(envType => typeEnvNamesMap[envType].length > 0).map(envType => ({
      count: typeEnvNamesMap[envType].length,
      envNames: typeEnvNamesMap[envType],
      key: envType,
      label: envTypeMap[envType]?.name || envType,
    }));
  }

  function getEnvTooltipContent(envNames: string[]) {
    return envNames.join('、');
  }

  /** 构建状态变化回调 */
  function handleBuildStatusChange(status: BuildStatus, num: string) {
    buildStatusInfo.value.status = status;
    if (status === 'running' && !runningNotified.value) {
      runningNotified.value = true;
      Message({
        message: t('构建 {0}：流水线构建任务已触发，正在执行中', [`#${num}`]),
        theme: 'primary',
      });
    } else if (status === 'success' && runningNotified.value) {
      runningNotified.value = false;
      fetchChartList();
      Message({
        message: t('构建 {0}：流水线构建{1}', [`#${num}`, t('成功')]),
        theme: 'success',
      });
    } else if (status === 'failed' && runningNotified.value) {
      runningNotified.value = false;
      fetchChartList();
      buildFailedTitle.value = t('构建 {0}：流水线构建{1}', [`#${num}`, t('失败')]);
    }
  }

  function handleClearFilters() {
    searchKeyword.value = '';
    fetchChartList();
  }

  function handleClearSearch() {
    searchKeyword.value = '';
    if (pagination.value.current === 1) {
      fetchChartList();
    } else {
      pageChange(1);
    }
  }

  /** 新建版本成功回调 */
  function handleCreateSuccess(version: string, buildID: string) {
    buildStatusInfo.value = { version, status: 'running' };
    showCreateDialog.value = false;
    isShowBuildRecords.value = true;
    // 等侧栏渲染后触发轮询等待该 buildID 出现
    nextTick(() => {
      buildRecordsRef.value?.triggerPolling(buildID);
    });
  }

  /** 新建版本按钮点击事件 */
  function handleCreateVersion() {
    showCreateDialog.value = true;
  }

  function handleSearch() {
    if (pagination.value.current === 1) {
      fetchChartList();
    } else {
      pageChange(1);
    }
  }

  /** 版本构建记录按钮点击事件 */
  function handleViewBuildRecords() {
    isShowBuildRecords.value = true;
  }

  /** 查看版本详情按钮点击事件 */
  function handleViewVersionDetail(row: AppHelmChartOutputObj) {
    currentChartVersion.value = row?.chartVersion ?? '';
    isShowVersionDetail.value = true;
  }

  watch(
    () => appDetailStore.appID,
    async () => {
      await fetchChartList();
    },
    {
      immediate: true,
    },
  );
</script>

<style lang="postcss" scoped>
  :deep(.helm-chart-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }
    .vxe-cell--sort {
      height: 20px;
      padding: 0 3px;
    }
  }
</style>
