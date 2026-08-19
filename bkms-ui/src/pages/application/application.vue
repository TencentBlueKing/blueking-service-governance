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
  <main class="flex flex-col h-full w-full relative overflow-hidden bg-[#F5F7FA] px-[24px] py-[16px]">
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
        <Layout.table :rows="10" />
      </template>
      <!-- 操作和搜索 -->
      <FlexRow
        average
        class="mb-[16px]"
      >
        <template #left>
          <div class="flex items-center gap-[8px]">
            <Button
              theme="primary"
              @click="goCreateApplication"
            >
              <Plus
                :height="24"
                :width="24"
              />
              {{ $t('创建应用') }}
            </Button>
          </div>
        </template>
        <template #right>
          <div class="flex items-center gap-[12px] justify-end">
            <SearchSelect
              v-model="searchValue"
              class="w-[520px] bg-[#fff] relative z-[100]"
              :data="appSearchData"
              :placeholder="
                createPlaceholder({
                  type: 'searchSelect',
                  labels: ['应用名称', '应用类型', '语言', '已部署环境'],
                })
              "
              unique-select
              value-behavior="need-key"
            >
            </SearchSelect>
            <!-- 排序 -->
            <div class="flex items-center">
              <Select
                v-model="curSortOption"
                class="w-[136px] bg-[#fff]"
                :clearable="false"
              >
                <Select.Option
                  v-for="(item, index) in sortOptions"
                  :id="item.value"
                  :key="index"
                  :name="item.label"
                />
              </Select>
              <Button
                :key="`${curSortOption}-${sortOrder}`"
                v-bk-tooltips="sortTip"
                class="!min-w-[32px] !px-[8px] !ml-[-1px] text-[#3A84FF]"
                @click="toggleSortOrder"
              >
                <i :class="['bkms-icon', sortOrder === 'asc' ? 'bkms-icon-shengxu' : 'bkms-icon-jiangxu']" />
              </Button>
            </div>
            <Radio.Group
              v-model="appTableMode"
              type="capsule"
            >
              <Radio.Button label="list">
                <i class="bkms-icon bkms-icon-shitu-liebiao text-[14px]"></i>
                {{ $t('列表模式') }}
              </Radio.Button>
              <Radio.Button label="global">
                <i class="bkms-icon bkms-icon-shitu-lianglan text-[14px]"></i>
                {{ $t('全局视图') }}
              </Radio.Button>
            </Radio.Group>
          </div>
        </template>
      </FlexRow>
      <!-- 表格数据 -->
      <div
        v-if="appTableMode === 'list'"
        ref="appTableContentRef"
        class="flex-1"
      >
        <Table
          ref="tableRef"
          :data="sortedTableData"
          :filter-config="{ remote: true }"
          :max-height="appTableHeight"
          :pagination="tablePagination"
          :row-class-name="getRowActiveClass"
          :row-config="{
            keyField: 'name',
            isHover: true,
            isCurrent: true,
          }"
          @cell-click="({ row }) => handleShowAppDetail(row)"
          @filter-change="handleFilterChange"
          @page-limit-change="
            limit => {
              updateLimit(limit);
            }
          "
          @page-value-change="updateCurrent"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="handleClearFilters"
              @refresh="handleGetAppList"
            >
            </TableException>
          </template>
          <TableColumn
            field="name"
            :label="$t('应用名称')"
          >
            <template #default="{ row }: { row: AppInfoOutputObj }">
              <Button
                text
                theme="primary"
                @click="handleShowAppDetail(row)"
                >{{ row.name || '--' }}</Button
              >
            </template>
          </TableColumn>
          <TableColumn
            field="kind"
            :filters="kindFilterOptions"
            show-overflow="tooltip"
          >
            <template #header>
              <CustomFilter
                field="kind"
                :filters="kindFilterOptions"
                :label="$t('应用类型')"
                :table-ref="tableRef"
              />
            </template>
            <template #default="{ row }: { row: AppInfoOutputObj }">
              <TypeIcon
                class="w-[100px]"
                classes="min-w-[40px] inline-block"
                :show-label="false"
                :type="row?.type"
              />
            </template>
          </TableColumn>
          <TableColumn
            field="language"
            :filters="languageFilterOptions"
            show-overflow="tooltip"
          >
            <template #header>
              <CustomFilter
                field="language"
                :filters="languageFilterOptions"
                :label="$t('语言')"
                :table-ref="tableRef"
              />
            </template>
            <template #default="{ row }: { row: AppInfoOutputObj }">
              {{ row?.language || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            field="deployedEnvs"
            :filters="deployedEnvsFilterOptions"
            min-width="200"
          >
            <template #header>
              <CustomFilter
                field="deployedEnvs"
                :filters="deployedEnvsFilterOptions"
                :label="$t('已部署环境')"
                :table-ref="tableRef"
              />
            </template>
            <template #default="{ row }: { row: AppInfoOutputObj }">
              <MoreTag
                v-if="row?.deployedEnvs?.length"
                :data="row.deployedEnvs"
              >
                <template #default="{ item }: { item: AppDeployedEnvOutputObj }">
                  <Popover
                    :popover-delay="[100, 0]"
                    theme="popover-dark-translucent"
                    trigger="hover"
                  >
                    <Tag>
                      <div class="flex items-center">
                        <ColorIcon
                          v-if="item.deployStatus"
                          class="mr-[4px]"
                          :icon="getDeployStatusInfo(row.type as AppType, item.deployStatus).icon"
                          :size="12"
                        />
                        {{ item.displayName || item.name }}
                      </div>
                    </Tag>
                    <template #content>
                      <div class="flex items-center gap-[6px]">
                        <ColorIcon
                          :icon="getDeployStatusInfo(row.type as AppType, item.deployStatus || '').icon"
                          :size="14"
                        />
                        <span>{{ item.displayName || item.name }}</span>
                        <span class="text-[#979BA5]">
                          （{{ getDeployStatusInfo(row.type as AppType, item.deployStatus || '').text }}）
                        </span>
                      </div>
                    </template>
                  </Popover>
                </template>
              </MoreTag>
              <span v-else>--</span>
            </template>
          </TableColumn>
          <TableColumn
            field="creator"
            :label="$t('创建者')"
            show-overflow="tooltip"
          >
            <template #default="{ row }: { row: AppInfoOutputObj }">
              {{ row?.creator || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            fixed="right"
            :label="$t('操作')"
            :width="150"
          >
            <template #default="{ row }: { row: AppInfoOutputObj }">
              <Button
                class="mr-[16px]"
                text
                theme="primary"
                @click.stop="handleGoBuild(row)"
              >
                {{ $t('构建') }}
              </Button>
              <Button
                text
                theme="primary"
                @click.stop="handleGoDeploy(row)"
                >{{ $t('部署') }}</Button
              >
            </template>
          </TableColumn>
        </Table>
      </div>
      <!-- 全局视图 -->
      <GlobalView
        v-else
        :apps="sortedTableData"
        :search-value="searchValue"
        :space="space"
        @clear="handleClearFilters"
      />
    </Skeleton>
  </main>
</template>
<script lang="ts" setup>
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Popover, Radio, SearchSelect, Select, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { sortByDate } from '~/common/util';
  import ColorIcon from '~/components/color-icon.vue';
  import CustomFilter from '~/components/custom-filter.vue';
  import MoreTag from '~/components/more-tag.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePersistentStorage from '~/composables/use-persistent-storage';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import TypeIcon from './components/type-icon.vue';
  import GlobalView from './global-view.vue';

  import type { ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { AppDeployedEnvOutputObj, AppInfoOutputObj } from '~/@types/v1/app';
  import type { AppType } from '~/composables/app-type';

  type AppSortField = 'createdAt' | 'lastOperatedAt' | 'name';

  interface IProps {
    active?: string; // 当前详情行的名称
    envName?: string; // 从环境管理页面跳转时传入的环境名称（用于自动筛选）
    space: string; // 空间
  }
  const props = defineProps<IProps>();

  const router = useRouter();
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const spaceStore = useSpaceStore();
  const appDetailStore = useAppDetail();
  const { getDeployStatusInfo } = useDeployStatusMap();

  const isLoading = ref(false);
  const appList = ref<AppInfoOutputObj[]>([]);

  const curSortOption = ref<AppSortField>('lastOperatedAt');
  const sortOrder = ref<'asc' | 'desc'>('asc');
  const sortOptions: { label: string; value: AppSortField }[] = [
    { label: t('应用名称'), value: 'name' },
    { label: t('操作时间'), value: 'lastOperatedAt' },
    { label: t('创建时间'), value: 'createdAt' },
  ];
  const appNameCollator = new Intl.Collator('zh-CN', { sensitivity: 'base' });
  const compareAppID = (left: AppInfoOutputObj, right: AppInfoOutputObj) =>
    appNameCollator.compare(left.id || '', right.id || '');
  const sortTip = computed(() => {
    if (curSortOption.value === 'name') return sortOrder.value === 'asc' ? 'A-Z' : 'Z-A';
    return sortOrder.value === 'asc' ? t('最新') : t('最早');
  });

  // 使用持久化分页hooks - 自动使用当前路由路径作为存储key
  const { usePagination } = usePersistentStorage();
  const { updateCurrent, updateLimit, updateCount, resetToFirstPage, tablePagination } = usePagination({
    defaultLimit: 20,
  });

  // ref
  const appTableContentRef = ref<HTMLElement>();

  // 表格查看模式
  const appTableMode = ref<'global' | 'list'>('list');

  // 使用 hooks 获取表格容器高度
  const { height: appTableHeight } = useElementHeight(appTableContentRef, {
    watchSource: isLoading,
    defaultHeight: 600,
  });

  // 搜索选择器配置
  const appSearchData = ref<ISearchItem[]>([
    {
      name: t('应用名称'),
      id: 'name',
    },
    {
      name: t('应用类型'),
      id: 'kind',
      multiple: true,
      children: [], // 动态填充
    },
    {
      name: t('语言'),
      id: 'language',
      multiple: true,
      children: [], // 动态填充
    },
    {
      name: t('已部署环境'),
      id: 'deployedEnvs',
      multiple: true,
      children: [], // 动态填充
    },
  ]);

  /** SearchSelect 选中值 */
  const searchValue = ref<ISearchValue[]>([]);

  /** 使用 useSearchFilter hook 实现 TableColumn filter 与 SearchSelect 联动 */
  const { filterOptions, handleFilterChange } = useSearchFilter(appSearchData, searchValue, [
    'kind',
    'language',
    'deployedEnvs',
  ] as const);

  /** 各列筛选配置 */
  const kindFilterOptions = computed(() => filterOptions.value.kind);
  const languageFilterOptions = computed(() => filterOptions.value.language);
  const deployedEnvsFilterOptions = computed(() => filterOptions.value.deployedEnvs);

  const tableRef = ref();

  /** 前端筛选逻辑 */
  const tableDataMatchSearch = computed(() => {
    if (!searchValue.value.length) return appList.value;

    let filteredList = [...appList.value];

    for (const filter of searchValue.value) {
      if (!filter.values?.length) continue;
      const selectedValues = filter.values.map(v => v.id);

      switch (filter.id) {
        case 'name':
          // 应用名称：模糊搜索
          filteredList = filteredList.filter(item =>
            selectedValues.some(val => item.name?.toLowerCase().includes(val.toLowerCase())),
          );
          break;
        case 'kind':
          filteredList = filteredList.filter(
            item =>
              (selectedValues.includes(EMPTY_FILTER_ID) && !item.type) || selectedValues.includes(item.type || ''),
          );
          break;
        case 'language':
          filteredList = filteredList.filter(
            item =>
              (selectedValues.includes(EMPTY_FILTER_ID) && !item.language) ||
              selectedValues.includes(item.language || ''),
          );
          break;
        case 'deployedEnvs':
          // 数组字段：只要有交集就匹配；空值匹配 __empty__
          filteredList = filteredList.filter(
            item =>
              (selectedValues.includes(EMPTY_FILTER_ID) && !item.deployedEnvs?.length) ||
              item.deployedEnvs?.some(env => selectedValues.includes(env.name || '')),
          );
          break;
      }
    }

    return filteredList;
  });

  /** 按用户选择的字段排序；空值始终置底，并以应用 ID 作为稳定的二次排序键 */
  const sortedTableData = computed(() => {
    const field = curSortOption.value;
    const data = [...tableDataMatchSearch.value];

    if (field !== 'name') {
      // 时间字段的升序交互表示最近时间优先，与时间戳的自然排序方向相反
      const dateSortOrder = sortOrder.value === 'asc' ? 'desc' : 'asc';
      return sortByDate(data, item => item[field], dateSortOrder, compareAppID);
    }

    const direction = sortOrder.value === 'asc' ? 1 : -1;
    return data.sort((left, right) => {
      if (!left.name || !right.name) {
        if (!left.name && right.name) return 1;
        if (left.name && !right.name) return -1;
      } else {
        const result = appNameCollator.compare(left.name, right.name);
        if (result !== 0) return result * direction;
      }

      return compareAppID(left, right);
    });
  });

  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  function getRowActiveClass({ row }: { row: AppInfoOutputObj }) {
    return router.currentRoute.value.query?.active === row.name ? 'row--current' : '';
  }

  // 创建应用
  function goCreateApplication() {
    router.push({
      name: 'createApplication',
      params: {
        space: props.space,
      },
    });
  }

  // 清除筛选
  function handleClearFilters() {
    // 清除 VXE Table 筛选状态
    const vxeTable = tableRef.value?.getVxeTableInstance?.();
    if (vxeTable) {
      vxeTable.clearFilter('kind');
      vxeTable.clearFilter('language');
      vxeTable.clearFilter('deployedEnvs');
    }
    // 清除 SearchSelect 选中值
    searchValue.value = [];
  }

  // 获取应用列表
  async function handleGetAppList() {
    if (!spaceStore.currentSpace) return;
    isLoading.value = true;
    appList.value = await ApiServerService.ListApps({
      workspaceID: spaceStore.currentSpace,
    })
      .then(data => {
        clearErrorType();
        // 动态填充筛选项 children
        handleInitFilterOptions(data);
        return data;
      })
      .catch(() => {
        setTypeToError();
        return [];
      });
    updateCount(appList.value.length);
    isLoading.value = false;
  }

  // 构建管理
  function handleGoBuild(row: AppInfoOutputObj) {
    router.push({
      name: 'detail',
      params: {
        name: row.name,
        menuName: 'build',
        type: row?.type || '',
      },
    });
  }

  // 部署管理
  function handleGoDeploy(row: AppInfoOutputObj) {
    router.push({
      name: 'detail',
      params: {
        name: row.name,
        menuName: 'deployment',
        type: row?.type || '',
      },
    });
  }

  /** 空值筛选标识 */
  const EMPTY_FILTER_ID = '__empty__';

  /** 从列表数据中提取去重值，填充筛选项 children（含空值「未设置」选项） */
  function handleInitFilterOptions(data: AppInfoOutputObj[]) {
    const kindSet = new Map<string, string>();
    const languageSet = new Map<string, string>();
    const envSet = new Map<string, string>();

    let hasEmptyKind = false;
    let hasEmptyLanguage = false;
    let hasEmptyEnvs = false;

    for (const item of data) {
      if (item.type) {
        kindSet.set(item.type, item.type);
      } else {
        hasEmptyKind = true;
      }
      if (item.language) {
        languageSet.set(item.language, item.language);
      } else {
        hasEmptyLanguage = true;
      }
      if (item.deployedEnvs?.length) {
        item.deployedEnvs.forEach(env => {
          if (env.name) envSet.set(env.name, env.displayName || env.name);
        });
      } else {
        hasEmptyEnvs = true;
      }
    }

    const emptyOption = { id: EMPTY_FILTER_ID, name: t('未设置') };
    const toChildren = (map: Map<string, string>, hasEmpty: boolean) => {
      const children = Array.from(map, ([id, name]) => ({ id, name }));
      if (hasEmpty) children.push(emptyOption);
      return children;
    };

    const findItem = (id: string) => appSearchData.value.find(item => item.id === id);
    const kindItem = findItem('kind');
    const languageItem = findItem('language');
    const envItem = findItem('deployedEnvs');

    if (kindItem) kindItem.children = toChildren(kindSet, hasEmptyKind);
    if (languageItem) languageItem.children = toChildren(languageSet, hasEmptyLanguage);
    if (envItem) envItem.children = toChildren(envSet, hasEmptyEnvs);
  }
  // 应用详情
  function handleShowAppDetail(row: AppInfoOutputObj) {
    appDetailStore.updateAppID(row.id || '');
    router.push({
      name: 'detail',
      params: {
        name: row.name,
        menuName: 'info',
        type: row?.type || '',
      },
    });
  }

  function toggleSortOrder() {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc';
  }

  watch(searchValue, () => {
    resetToFirstPage();
  });

  watch([curSortOption, sortOrder], () => {
    resetToFirstPage();
  });

  // 应用列表变化时，更新总数
  watch(tableDataMatchSearch, newValue => {
    updateCount(newValue.length);
  });

  // space变化时，重新请求表格数据
  watch(
    () => props.space,
    async (newSpace, oldSpace) => {
      if (newSpace === oldSpace) return;
      await handleGetAppList();
      resetToFirstPage();
    },
  );

  onBeforeMount(async () => {
    // 初始化应用信息
    appDetailStore.updateAppName(props?.active || '');
    await handleGetAppList();

    // 从环境管理页面跳转时，自动按已部署环境筛选
    if (props.envName) {
      handleFilterChange({ field: 'deployedEnvs', values: [props.envName!] });
    }
  });
</script>
