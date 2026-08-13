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
  <div class="flex items-center justify-between gap-[12px] mb-[16px]">
    <div :id="DEPLOY_ENV_SELECT_SLOT_ID"></div>
    <SearchSelect
      v-model="searchValue"
      class="min-w-[560px] bg-[#fff] relative z-[100]"
      :data="searchData"
      :placeholder="
        createPlaceholder({
          type: 'searchSelect',
          labels: ['镜像 Tag', '操作人'],
        })
      "
      unique-select
      value-behavior="need-key"
    />
  </div>
  <Skeleton
    :full-height="false"
    :loading="isLoading"
  >
    <template #loading>
      <Layout.table />
    </template>
    <div class="bg-[#FFFFFF]">
      <!-- 表格 -->
      <Table
        :data="tableData"
        :pagination="pagination"
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <TableException
            :type="tableData.length === 0 && searchValue.length === 0 ? 'empty' : 'search'"
            @clear="handleClearSearch"
          />
        </template>
        <TableColumn
          :label="$t('镜像 Tag')"
          min-width="150"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            {{ row.imageTag || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('实例数')"
          :width="100"
        >
          <template #default="{ row }">
            {{ row.replicas ?? '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('部署状态')"
          min-width="120"
        >
          <template #default="{ row }">
            <StatusIcon
              class="line-height-[22px]"
              :message="row.status !== APP_DEPLOY_STATUS.DEPLOYED ? row.message : ''"
              :pending="row.status === APP_DEPLOY_STATUS.DEPLOYING || row.status === APP_DEPLOY_STATUS.UNINSTALLING"
              :size="12"
              :status="row.status"
              :status-color-map="deployStatusColorMap"
              :status-text-map="deployStatusTextMap"
            />
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作人')"
          :min-width="120"
        >
          <template #default="{ row }">
            {{ row.operator || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作时间')"
          min-width="150"
        >
          <template #default="{ row }">
            {{ row.createdAt ? formatDateString(row.createdAt) : '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作')"
          width="180"
        >
          <template #default="{ row }">
            <Button
              text
              theme="primary"
              @click.stop="handleShowResourceSnapshots(row)"
            >
              {{ $t('查看详情') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </div>
  </Skeleton>

  <ResourceSnapshotSideslider
    v-model:is-show="showResourceSnapshots"
    :deploy-id="selectedDeployID"
  />
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, SearchSelect } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppModelDeployRecordOutputObj, PaginatedAppModelDeployRecordsOutputObjs } from '~/@types/v1/deploy';
  import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import StatusIcon from '~/components/status-icon.vue';
  import TableException from '~/components/table-exception.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { DEPLOY_ENV_SELECT_SLOT_ID } from './constants';
  import ResourceSnapshotSideslider from './resource-snapshot-sideslider.vue';
  import { type DeployableAppType, useDeployAPIs } from './use-deploy';

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString } = useTime();
  const { appStatusTextMap: deployStatusTextMap, appStatusColorMap: deployStatusColorMap } = useDeployStatusMap();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  const searchValue = ref<Array<{ id: string; name: string; values?: Array<{ id: string; name: string }> }>>([]);
  const tableData = ref<AppModelDeployRecordOutputObj[]>([]);
  const isLoading = ref(false);
  const count = ref(0);
  const showResourceSnapshots = ref(false);
  const selectedDeployID = ref('');

  // 搜索配置
  const searchData = [
    {
      name: t('镜像 Tag'),
      id: 'imageTag',
    },
    {
      name: t('操作人'),
      id: 'operator',
    },
  ];

  // 从 searchValue 中提取搜索关键字
  const searchKeyword = computed(() => {
    return searchValue.value
      .map(item => {
        if (item.values && item.values.length > 0) {
          return item.values.map(v => v.id).join(',');
        }
        return item.name;
      })
      .join(' ');
  });

  // 获取部署历史列表
  async function fetchDeployHistory() {
    if (!appDetailStore.appID || !trpcDeployStore.curEnvItem?.name) return;

    isLoading.value = true;
    try {
      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      const res = await deployAPIs.listDeployRecords({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem.name,
        keyword: searchKeyword.value,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      });
      tableData.value = (res as unknown as PaginatedAppModelDeployRecordsOutputObjs).results || [];
      count.value = Number((res as unknown as PaginatedAppModelDeployRecordsOutputObjs).count) || 0;
    } catch {
      tableData.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    tableData,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: fetchDeployHistory,
      onPageSizeChange: fetchDeployHistory,
    },
    count,
  );

  // 清除搜索
  function handleClearSearch() {
    searchValue.value = [];
  }

  // 查看历史详情
  function handleShowResourceSnapshots(row: AppModelDeployRecordOutputObj) {
    selectedDeployID.value = row?.id ?? '';
    showResourceSnapshots.value = true;
  }

  // 监听搜索关键字变化
  watch(
    searchValue,
    () => {
      handleResetPage();
      fetchDeployHistory();
    },
    { immediate: true, deep: true },
  );

  // 监听环境变化
  watch(
    () => trpcDeployStore.curEnvItem?.name,
    () => {
      handleResetPage();
      fetchDeployHistory();
    },
  );
</script>
