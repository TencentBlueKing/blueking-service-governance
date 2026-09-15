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
        row-class-name="cursor-pointer"
        :row-config="{
          keyField: 'name',
          isHover: true,
          isCurrent: true,
        }"
        :row-height="56"
        :sort-config="sortConfig"
        @filter-change="filterChangeEvent"
        @page-limit-change="
          pagination.limit = $event;
          pagination.current = 1;
        "
        @page-value-change="pagination.current = $event"
        @row-click="handleRowClick"
        @scroll="handleScroll"
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
                @click.stop="handleShowEnvDetail(row)"
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
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #header>
            <span class="inline-flex items-center gap-[4px] mr-[4px]">
              {{ $t('环境分类') }}
              <EnvCategoryDescription />
            </span>
          </template>
          <template #default="{ row }">
            <span
              class="env-normal inline-flex items-center justify-center min-w-[40px] px-[8px] rounded-[2px]"
              :class="envTypeTagClassMap[row.type]"
            >
              {{ envTypeMap(row.type) || '--' }}
            </span>
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
  <DeleteEnvAction
    ref="deleteEnvActionRef"
    @deleted="handleGetEnvList"
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
  import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, Message, SearchSelect, Select } from 'bkui-vue';
  import { Done, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
  import { EnvOutput } from '~/@types/v1/env';
  import { EnvService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useElementHeight } from '~/composables/use-element-height';
  import { envDetailLocation, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useEnvListStateStore } from '~/stores/env-list-state';
  import { useSpaceStore } from '~/stores/space';

  import DeleteEnvAction from './components/delete-env-action.vue';
  import EnvCategoryDescription from './components/env-category-description.vue';
  import CreateEnv from './create-env.vue';
  import PublicEnvVarsSideslider from './public-env-vars/public-env-vars-sideslider.vue';

  import type { VxeTableDefines } from '@blueking/vxe-table';

  interface IProps {
    space: string;
  }

  defineProps<IProps>();

  const router = useRouter();
  const route = useRoute();
  const spaceStore = useSpaceStore();
  const { createPlaceholder } = useSearchPlaceholder();

  const deleteEnvActionRef = ref<InstanceType<typeof DeleteEnvAction>>();
  const listStateStore = useEnvListStateStore();
  const listSpace = String(route.params.space);
  const savedState = listStateStore.get(listSpace);
  let restoring = true;
  let disposed = false;
  let scrollTop = savedState?.scrollTop || 0;
  let scrollLeft = savedState?.scrollLeft || 0;

  const isLoading = ref(false);
  const envList = ref<EnvOutput[]>([]);
  let listLoaded = false;
  const pagination = ref({ count: 0, limit: savedState?.limit || 20, current: savedState?.current || 1 });
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
  const EnvTableRef = ref<{
    getVxeTableInstance: () => { scrollTo: (left: number, top: number) => Promise<unknown> };
  }>();

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
          name: t('预发布'),
          id: 'staging',
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
  const envTypeMap = (type: 'development' | 'production' | 'staging' | 'test') => {
    switch (type) {
      case 'development':
        return t('开发');
      case 'test':
        return t('测试');
      case 'staging':
        return t('预发布');
      case 'production':
        return t('生产');
      default:
        return t('未知');
    }
  };

  // 排序配置
  const curEnvOption = ref(savedState?.sortField || 'type');
  const sortOrder = ref<'asc' | 'desc'>(savedState?.sortOrder || 'asc');
  const envOptions = ref([
    { label: t('环境分类'), value: 'type' },
    { label: t('环境 ID'), value: 'name' },
    { label: t('环境名称'), value: 'displayName' },
  ]);
  const ENV_TYPE_ORDER: Record<string, number> = { development: 1, test: 2, staging: 3, production: 4 };

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
        listLoaded = true;
        return data;
      })
      .catch(() => {
        setTypeToError();
        return [];
      });
    pagination.value.count = tableDataMatchSearch.value.length;
    if (!restoring) clampPage();
    isLoading.value = false;
    if (!disposed) resolveLegacyDetail();
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

  // 点击环境行时切换当前环境；操作列按钮通过 .stop 保持各自的行为。
  function handleRowClick(_event: Event, row: EnvOutput) {
    handleShowEnvDetail(row);
  }

  function handleShowEnvDetail(row: EnvOutput) {
    if (row.id) router.push(envDetailLocation(row.id));
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

  function clampPage() {
    pagination.value.current = Math.max(
      1,
      Math.min(pagination.value.current, Math.ceil(tableDataMatchSearch.value.length / pagination.value.limit)),
    );
  }

  function handleDeleteEnv(row: EnvOutput) {
    deleteEnvActionRef.value?.show(row);
  }

  function handleScroll(event: VxeTableDefines.ScrollEventParams) {
    if (restoring) return;
    scrollTop = event.scrollTop;
    scrollLeft = event.scrollLeft;
  }

  function saveListState() {
    if (restoring) return;
    listStateStore.save(listSpace, {
      search: searchValue.value,
      sortField: curEnvOption.value,
      sortOrder: sortOrder.value,
      current: pagination.value.current,
      limit: pagination.value.limit,
      scrollTop,
      scrollLeft,
    });
  }
  onBeforeRouteLeave(saveListState);
  onBeforeUnmount(() => {
    saveListState();
    disposed = true;
  });

  watch(searchValue, () => {
    if (!restoring) pagination.value.current = 1;
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

  // Skeleton 动画结束后表格才挂载；在真实表格可用时恢复滚动。
  watch(
    [EnvTableRef, isLoading],
    async ([table, loading]) => {
      if (!table || loading || !restoring) return;
      await nextTick();
      if (disposed) return;
      await table.getVxeTableInstance().scrollTo(scrollLeft, scrollTop);
      restoring = false;
    },
    { flush: 'post' },
  );

  async function resolveLegacyDetail() {
    if (!listLoaded || isLoading.value || disposed) return;
    const { active, activeTab, ...restQuery } = route.query;
    if (typeof active !== 'string' || !active || curExceptionType.value === 'error') return;
    const env = envList.value.find(item => item.name === active);
    if (env?.id) {
      await router.replace(envDetailLocation(env.id, typeof activeTab === 'string' ? activeTab : undefined, restQuery));
    } else {
      Message({ message: t('环境不存在或已被删除'), theme: 'warning' });
      await router.replace({ query: restQuery });
    }
  }
  watch(() => [route.query.active, route.query.activeTab], resolveLegacyDetail);

  onMounted(async () => {
    if (savedState) searchValue.value = savedState.search;
    await handleGetEnvList();
    if (disposed) return;
    clampPage();
  });
</script>
<style lang="postcss" scoped>
  :deep(.env-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }

    .row--current .env-normal.env-tag-development {
      background-color: #cddffe !important;
    }
  }
  .env-normal {
    color: #63656e;
    background-color: #f0f1f5;
  }
</style>
