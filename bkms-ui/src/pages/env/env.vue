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
  <Skeleton
    :loading="isLoading"
    theme="gray"
  >
    <template #loading>
      <FlexRow class="w-full mb-[16px]">
        <template #left>
          <Layout.shape :width="110" />
        </template>
        <template #right>
          <Layout.shape :width="400" />
        </template>
      </FlexRow>
      <Layout.table class="mt-[10px]" />
    </template>
    <FlexRow
      average
      class="mb-[16px]"
    >
      <template #left>
        <div class="flex items-center gap-[8px]">
          <Button
            theme="primary"
            @click="isShowCreateEnv = true"
          >
            <Plus
              :height="24"
              :width="24"
            />
            {{ $t('新建环境') }}
          </Button>
          <!-- TODO: 公共环境变量 -->
          <Button
            outline
            theme="primary"
            @click="isShowPublicEnvVars = true"
          >
            <i class="bkms-icon bkms-icon-variable mr-[6px] text-[14px]"></i>
            {{ $t('公共环境变量') }}
          </Button>
        </div>
      </template>
      <template #right>
        <div class="flex justify-end gap-[8px]">
          <div class="flex items-center justify-end">
            <SearchSelect
              v-model="searchValue"
              class="w-[520px] bg-[#fff] relative z-[10]"
              :data="envSearchData"
              :placeholder="
                createPlaceholder({
                  type: 'searchSelect',
                  labels: ['环境名称', '环境ID', '环境分类'],
                })
              "
              unique-select
              value-behavior="need-key"
            >
            </SearchSelect>
          </div>
          <Select
            v-model="curEnvOption"
            :clearable="false"
          >
            <template #prefix>
              <div
                class="flex items-center px-[8px] cursor-pointer"
                @click.stop="toggleSortOrder"
              >
                <i :class="['bkms-icon', sortOrder === 'asc' ? 'bkms-icon-jiangxu' : 'bkms-icon-shengxu']" />
              </div>
            </template>
            <Select.Option
              v-for="(item, index) in envOptions"
              :id="item.value"
              :key="index"
              :name="item.label"
            />
          </Select>
        </div>
      </template>
    </FlexRow>
    <div
      ref="envTableContentRef"
      class="!h-[calc(100%-48px)]"
    >
      <Table
        ref="EnvTableRef"
        class="env-table"
        :data="tableDataMatchSearch"
        :filter-config="{ remote: true }"
        :max-height="envTableContentHeight"
        :pagination="pagination"
        :row-class-name="getRowActiveClass"
        :row-config="{
          keyField: 'name',
          isHover: true,
          isCurrent: true,
        }"
        :row-height="56"
        :sort-config="sortConfig"
        @filter-change="filterChangeEvent"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="searchValue = []"
            @refresh="handleGetEnvList"
          >
          </TableException>
        </template>
        <TableColumn
          field="name"
          :label="$t('环境名称/ID')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            <div class="leading-[20px]">
              <Button
                text
                theme="primary"
                @click="handleShowEnvDetail(row)"
                >{{ row.displayName }}</Button
              >
              <div class="text-[#979BA5]">{{ row.name }}</div>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="type"
          filter-multiple
          :filters="filterOptions.type"
          :label="$t('环境分类')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            <div :class="['w-[50px] text-[#fff] rounded-[2px] text-center env-normal', `env-${row.type}`]">
              {{ envTypeMap(row.type) || '--' }}
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="defaultNamespace"
          :label="$t('集群资源')"
          :min-width="100"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: EnvOutput }">
            <div
              v-if="row?.status === 'Ready'"
              class="flex items-center gap-[8px]"
            >
              <div class="bg-[#CBF0DA] rounded-[50%] flex items-center justify-center w-[14px] h-[14px]">
                <Done
                  :height="12"
                  text="#2CAF5E"
                  :width="12"
                />
              </div>
              {{ $t('已配置') }}
            </div>
            <div
              v-else
              class="flex items-center gap-[8px]"
            >
              <i class="bkms-icon bkms-icon-time-circle-fill text-[#C4C6CC] text-[14px]"></i>
              {{ $t('未配置') }}
            </div>
          </template>
        </TableColumn>
        <TableColumn
          align="right"
          field="appIDs"
          :label="$t('应用')"
          show-overflow="tooltip"
          :width="60"
        >
          <template #default="{ row }: { row: EnvOutput }">
            <Button
              v-if="row?.appIDs?.length"
              text
              theme="primary"
              @click.stop="handleGoAppByEnv(row)"
            >
              {{ row.appIDs.length }}
            </Button>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 空列占位-->
        <TableColumn
          label=""
          show-overflow="tooltip"
          :width="50"
        >
        </TableColumn>
        <TableColumn
          :fixed="'right'"
          :label="$t('操作')"
          :width="150"
        >
          <template #default="{ row }: { row: EnvOutput }">
            <Button
              class="mr-[16px]"
              text
              theme="primary"
              @click.stop="handleShowEnvDetail(row)"
            >
              {{ $t('编辑') }}
            </Button>
            <Button
              text
              theme="primary"
              @click.stop="handleDeleteEnv(row)"
              >{{ $t('删除') }}</Button
            >
          </template>
        </TableColumn>
      </Table>
    </div>
  </Skeleton>
  <EnvDetail
    v-if="curRow"
    ref="envDetailRef"
    :data="curRow"
    @delete="handleDeleteEnv"
    @update="handleUpdate"
  />
  <!-- 无法删除环境（有部署应用） -->
  <DeployedAppsWarning
    v-model:is-show="showDeployedWarning"
    :deployed-apps="deployedApps"
    :dialog-title="$t('无法删除环境')"
    :tips-text="$t('该环境已部署应用，请先卸载后再删除环境')"
  />
  <!-- 确认删除环境（无部署应用） -->
  <DeleteEnvDialog
    v-model:is-show="isShowDeleteEnvDialog"
    :env-display-name="deleteEnvRow?.displayName || ''"
    :env-name="deleteEnvRow?.name || ''"
    @confirm="confirmDeleteEnv"
  />
  <CreateEnv
    v-model:is-show="isShowCreateEnv"
    @confirm="handleGetEnvList"
  />
  <!-- 公共环境变量 -->
  <PublicEnvVarsSideslider
    v-model:visible="isShowPublicEnvVars"
    :space="space"
  ></PublicEnvVarsSideslider>
</template>
<script lang="ts" setup>
  import { computed, nextTick, onBeforeMount, ref, shallowRef, watch } from 'vue';
  import { provide } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, Message, SearchSelect, Select } from 'bkui-vue';
  import { Done, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { EnvAppDeployStatusOutput, EnvOutput } from '~/@types/v1/env';
  import { EnvService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useElementHeight } from '~/composables/use-element-height';
  import { useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useSpaceStore } from '~/stores/space';

  import DeployedAppsWarning from './components/project-selector/deployed-apps-warning.vue';
  import CreateEnv from './create-env.vue';
  import DeleteEnvDialog from './delete-env-dialog.vue';
  import EnvDetail from './detail.vue';
  import PublicEnvVarsSideslider from './public-env-vars/public-env-vars-sideslider.vue';

  import type { VxeTableDefines } from '@blueking/vxe-table';

  interface IProps {
    space: string;
  }

  defineProps<IProps>();

  // 通过 provide 向子组件注入当前环境名称
  provide(
    'envName',
    computed(() => curRow.value?.name || ''),
  );

  const router = useRouter();
  const spaceStore = useSpaceStore();
  const { createPlaceholder } = useSearchPlaceholder();

  const envDetailRef = ref<InstanceType<typeof EnvDetail>>();

  const isLoading = ref(false);
  const envList = ref<EnvOutput[]>([]);
  const pagination = ref({ count: 0, limit: 20, current: 1 });
  const curRow = ref<EnvOutput>();
  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
    sortMethod({ data, sortList }: { data: EnvOutput[]; sortList: VxeTableDefines.SortCheckedParams[] }) {
      const { field, order } = sortList[0];
      if (field === 'deploymentApplication') {
        data.sort((a, b) =>
          order === 'desc'
            ? (a.appIDs?.length ?? 0) - (b.appIDs?.length ?? 0)
            : (b.appIDs?.length ?? 0) - (a.appIDs?.length ?? 0),
        );
      }
      return data;
    },
  });

  // 公共环境变量
  const isShowPublicEnvVars = ref(false);

  // 搜索(防抖)
  const searchKey = ref<string>('');
  const debounceSearch = useDebounce(searchKey, 300);

  // ref
  const envTableContentRef = ref<HTMLElement>();
  const EnvTableRef = ref<HTMLElement>();

  // 使用 useElementHeight hook 获取表格容器高度
  const { height: envTableContentHeight } = useElementHeight(envTableContentRef, {
    watchSource: isLoading,
    defaultHeight: 600,
  });

  // 引入国际化
  const { t } = useI18n();
  // 环境列表
  const envSearchData = shallowRef([
    {
      name: t('环境名称'),
      id: 'displayName',
      multiple: false,
      placeholder: t('环境名称'),
      field: 'displayName',
      fuzzy: true,
    },
    {
      name: t('环境ID'),
      id: 'name',
      multiple: false,
      placeholder: t('环境ID'),
      field: 'name',
      fuzzy: true,
    },
    {
      name: t('环境分类'),
      id: 'type',
      multiple: true,
      placeholder: t('环境分类'),
      field: 'type',
      children: [
        {
          name: t('开发'),
          id: 'development',
        },
        {
          name: t('测试'),
          id: 'test',
        },
        {
          name: t('生产'),
          id: 'production',
        },
        {
          name: t('未知'),
          id: 'unknown',
        },
      ],
    },
  ]);

  const {
    filterChangeEvent,
    searchValue,
    tableDataMatchSearch: rawTableDataMatchSearch,
    filterOptions,
  } = useTableSearchSelect(envList, envSearchData, {
    tableRef: EnvTableRef,
  });

  // 应用排序后的表格数据
  const tableDataMatchSearch = computed(() => sortEnvList(rawTableDataMatchSearch.value));
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });
  // 映射环境分类
  const envTypeMap = (type: 'development' | 'production' | 'test') => {
    switch (type) {
      case 'development':
        return t('开发');
      case 'test':
        return t('测试');
      case 'production':
        return t('生产');
      default:
        return t('未知');
    }
  };

  // 排序配置
  const curEnvOption = ref('type');
  const sortOrder = ref<'asc' | 'desc'>('asc');
  const envOptions = ref([
    { label: t('环境分类'), value: 'type' },
    { label: t('环境 ID'), value: 'name' },
    { label: t('环境名称'), value: 'displayName' },
  ]);
  const ENV_TYPE_ORDER: Record<string, number> = { development: 1, test: 2, production: 3 };

  function getRowActiveClass({ row }: { row: EnvOutput }) {
    return router.currentRoute.value.query?.active === row.name ? 'row--current' : '';
  }

  // 获取环境列表
  async function handleGetEnvList() {
    if (!spaceStore.currentSpace) return;
    isLoading.value = true;
    envList.value = await EnvService.listEnvs(
      {
        workspaceID: spaceStore.currentSpace,
      },
      { validateCode: false },
    )
      .then(data => {
        clearErrorType();
        return data;
      })
      .catch(() => {
        setTypeToError();
        return [];
      });
    pagination.value.count = envList.value.length;
    isLoading.value = false;
  }

  // 跳转到应用管理页面，按当前环境筛选
  function handleGoAppByEnv(row: EnvOutput) {
    router.push({
      name: 'app',
      params: {
        space: spaceStore.currentSpace,
        envName: row.name,
      },
    });
  }

  // env详情
  function handleShowEnvDetail(row: EnvOutput) {
    curRow.value = row;
    // 更新query参数
    router.replace({
      query: {
        active: row.name,
      },
    });
    nextTick(() => {
      envDetailRef.value?.show();
    });
  }

  // 排序
  function sortEnvList(list: EnvOutput[]) {
    const field = curEnvOption.value as keyof EnvOutput;
    const multiplier = sortOrder.value === 'asc' ? 1 : -1;

    return [...list].sort((a, b) => {
      let result = 0;
      if (field === 'type' && a.type && b.type) {
        result = (ENV_TYPE_ORDER[a.type] || 999) - (ENV_TYPE_ORDER[b.type] || 999);
      } else {
        result = String(a[field] || '').localeCompare(String(b[field] || ''), 'zh-CN');
      }
      return result * multiplier;
    });
  }

  function toggleSortOrder() {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc';
  }

  // 是否展示新建环境
  const isShowCreateEnv = ref(false);

  // 删除环境
  const isShowDeleteEnvDialog = ref(false);
  const showDeployedWarning = ref(false);
  const deployedApps = ref<EnvAppDeployStatusOutput[]>([]);
  const deleteEnvRow = ref<EnvOutput>();

  // 确认删除环境
  async function confirmDeleteEnv() {
    if (!deleteEnvRow.value) return;
    const result = await EnvService.deleteEnv({
      envID: deleteEnvRow.value?.id || '',
    })
      .then(() => true)
      .catch(() => false);
    if (result) {
      isShowDeleteEnvDialog.value = false;
      Message({
        message: t('删除成功'),
        theme: 'success',
      });
      await handleGetEnvList();
    }
  }
  async function handleDeleteEnv(row: EnvOutput) {
    deleteEnvRow.value = row;
    // 获取环境详情，检查是否有部署应用
    let envDetail;
    try {
      envDetail = await EnvService.getEnv({ envID: row?.id || '' });
    } catch (error) {
      console.error('获取环境详情失败:', error);
      Message({
        message: t('获取环境详情失败，无法验证部署状态'),
        theme: 'error',
      });
      return;
    }
    if (envDetail?.appDeployStatuses?.length) {
      deployedApps.value = envDetail.appDeployStatuses;
      showDeployedWarning.value = true;
      return;
    }
    isShowDeleteEnvDialog.value = true;
  }

  // 更新环境详情后，重新获取列表
  async function handleUpdate() {
    await handleGetEnvList();
  }

  watch(searchValue, () => {
    pagination.value.current = 1;
  });

  // 搜索（防抖）
  watch(debounceSearch, async () => {
    await handleGetEnvList();
    pagination.value.current = 1;
  });

  // 环境列表变化时，更新总数
  watch(tableDataMatchSearch, newValue => {
    pagination.value.count = newValue?.length || 0;
  });

  onBeforeMount(async () => {
    await handleGetEnvList();
    curRow.value = envList.value.find(item => item.name === router.currentRoute.value.query?.active);
    if (router.currentRoute.value.query?.active && curRow.value) {
      nextTick(() => {
        envDetailRef.value?.show();
      });
    }
  });
</script>
<style lang="postcss" scoped>
  :deep(.env-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }
  }
  .env-normal {
    color: #63656e;
    background-color: #f0f1f5;
    border: 1px solid #979ba54d;
  }
  .env-production {
    color: #299e56;
    background-color: #daf6e5;
    border: 1px solid #a1e3ba;
  }
  .env-test {
    color: #e38b02;
    background-color: #fdeed8;
    border: 1px solid #f9d090;
  }
  .env-development {
    color: #1768ef;
    background-color: #e1ecff;
    border: 1px solid #a3c5fd;
  }
</style>
