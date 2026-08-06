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
  <Skeleton :loading="skeletonLoading">
    <template #loading>
      <div class="mb-[12px] flex justify-end">
        <Layout.shape :width="348" />
      </div>
      <Layout.table />
    </template>
    <div class="mb-[12px] flex justify-end">
      <Input
        v-model.trim="searchValue"
        class="max-w-[348px]"
        clearable
        :placeholder="$t('搜索 Chart 版本/镜像 Tag/操作人')"
        type="search"
      />
    </div>
    <Table
      :data="deployHistoryList"
      :pagination="pagination"
      @page-limit-change="pageSizeChange"
      @page-value-change="pageChange"
    >
      <template #empty>
        <TableException
          :type="curExceptionType"
          @clear="handleClearFilters"
          @refresh="handleGetData"
        >
        </TableException>
      </template>
      <TableColumn
        :label="$t('Chart 版本')"
        min-width="150"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: any }">
          {{ row.chartVersion || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('镜像 Tag')"
        min-width="100"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: any }">
          {{ row.imageTag || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('状态')"
        min-width="100"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: any }">
          <StatusIcon
            :message="row.message"
            :pending="row.status === 'loading'"
            :status="row.status"
            :status-color-map="statusColorMap"
            :status-text-map="statusTextMap"
            type="result"
          />
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('操作人')"
        min-width="100"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: any }">
          {{ row.operator || '--' }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('部署时间')"
        min-width="100"
        show-overflow="tooltip"
      >
        <template #default="{ row }: { row: any }">
          {{ filterTimeFormat(row.createdAt || '') }}
        </template>
      </TableColumn>
      <TableColumn
        :label="$t('操作')"
        min-width="100"
        show-overflow="tooltip"
      >
        <template #default="{ row, rowIndex }: { row: any; rowIndex?: number }">
          <div class="flex items-center gap-[10px]">
            <span v-bk-tooltips="{ content: $t('仅支持查看近期部署历史的 Values'), disabled: row.values }">
              <Button
                :disabled="!row.values"
                text
                theme="primary"
                @click.stop="handleShowValues(row)"
              >
                {{ $t('查看 Values') }}
              </Button>
            </span>
            <span
              v-bk-tooltips="{
                content: $t('不可回滚到当前版本'),
                disabled: !isLatestVersion(rowIndex) || !row.values,
              }"
            >
              <Button
                :disabled="!row.values || isLatestVersion(rowIndex)"
                text
                theme="primary"
                @click.stop="showPreviewRollback(row)"
              >
                {{ $t('回滚到此版本') }}
              </Button>
            </span>
          </div>
        </template>
      </TableColumn>
    </Table>
  </Skeleton>
  <Sideslider
    v-model:is-show="isShowValues"
    quick-close
    render-directive="show"
    :title="$t('查看 Values')"
    :width="700"
    @hidden="values = ''"
  >
    <template #default>
      <!-- 代码编辑器 -->
      <MsEditor
        class="!h-[calc(100vh-58px)]"
        :model-value="values"
        readonly
      />
    </template>
  </Sideslider>
  <!-- 回滚预览 -->
  <PreviewRollback
    v-model:is-show="previewRollbackConfig.isShow"
    :app-name="appDetailStore.app"
    :deploy-i-d="previewRollbackConfig.historyID"
    :env-name="envName"
    :traffic-lane-name="laneName"
    @success="handleRollbackSuccess"
  >
  </PreviewRollback>
</template>
<script setup lang="ts">
  import { onBeforeUnmount, onMounted, ref, watch, watchEffect } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, Input, Sideslider } from 'bkui-vue';
  import { HelmDeployRecordOutputObj } from '~/@types/v1/deploy';
  import { filterTimeFormat } from '~/common/util';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useInterval from '~/composables/use-interval';
  import usePageConf from '~/composables/use-page';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useAppDetail } from '~/stores/app-detail';

  import PreviewRollback from './preview-rollback.vue';
  import { useHelmDeploy } from './use-helm-deploy';

  interface IProps {
    envName: string;
    initialLoading?: boolean;
    laneName: string;
    skipInitialFetch?: boolean;
  }
  const props = withDefaults(defineProps<IProps>(), {
    initialLoading: false,
    skipInitialFetch: false,
  });

  const emit = defineEmits<{
    searchChange: [searchValue: string];
  }>();

  const appDetailStore = useAppDetail();

  const searchValue = ref('');
  const debouncedSearch = useDebounce(searchValue, 300);

  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  const { deployHistoryList, count, statusColorMap, statusTextMap, handleListDeployHistories } = useHelmDeploy();

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    deployHistoryList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: handleGetData,
      onPageSizeChange: handleGetData,
    },
    count,
  );

  const isShowValues = ref<boolean>(false);
  const values = ref<string>('');
  function handleShowValues(row: HelmDeployRecordOutputObj) {
    values.value = row?.values ?? '';
    isShowValues.value = true;
  }
  const skeletonLoading = ref(false);
  async function handleGetData() {
    try {
      await handleListDeployHistories({
        page: pagination.value.current,
        pageSize: pagination.value.limit,
        envName: props.envName,
        trafficLaneName: props.laneName,
        keyword: debouncedSearch.value,
      });
      clearErrorType();
    } catch (err) {
      console.error(err);
      setTypeToError();
    } finally {
      skeletonLoading.value = false;
    }
  }
  const { start, stop } = useInterval(handleGetData, 5000); // 轮询

  const previewRollbackConfig = ref({
    isShow: false,
    historyID: '',
  });
  // 回滚成功处理
  function handleRollbackSuccess() {
    previewRollbackConfig.value.isShow = false;
    handleGetData();
  }

  // 判断是否为最新版本（第一页的第一条）
  function isLatestVersion(rowIndex?: number): boolean {
    return pagination.value.current === 1 && rowIndex === 0;
  }

  // 显示预览回滚弹窗
  function showPreviewRollback(row: HelmDeployRecordOutputObj) {
    previewRollbackConfig.value.isShow = true;
    previewRollbackConfig.value.historyID = row?.id || '';
  }

  onMounted(async () => {
    // 如果父组件标记跳过初始化获取，则使用父组件的加载状态
    if (props.skipInitialFetch) {
      skeletonLoading.value = props.initialLoading;
    } else {
      // 否则正常初始化加载
      skeletonLoading.value = true;
      await handleGetData();
      skeletonLoading.value = false;
    }
    start();
  });

  // 监听父组件传递的加载状态变化
  watchEffect(() => {
    if (props.skipInitialFetch) {
      skeletonLoading.value = props.initialLoading;
    }
  });

  // 清除筛选并搜索
  function handleClearFilters() {
    searchValue.value = '';
    handleGetData();
  }

  watch(debouncedSearch, async newValue => {
    emit('searchChange', newValue);
    handleResetPage();
    handleGetData();
  });

  onBeforeUnmount(() => {
    deployHistoryList.value = [];
    stop();
  });
</script>
<style lang="postcss" scoped>
  :deep(.bk-modal-content) {
    height: calc(100vh - 52px) !important;
  }
</style>
