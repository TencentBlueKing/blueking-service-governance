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
  <div class="bg-[url(/home-bg.png)] bg-repeat-x flex justify-center">
    <div class="min-w-[1366px] max-w-[1600px] w-full mt-[24px]">
      <div class="flex items-start justify-end info-panel">
        <div class="flex items-center h-[80px] w-[400px]">
          <div class="flex-1 h-full rounded-tl-lg rounded-lg relative flex flex-col">
            <div class="relative flex-1 flex items-center justify-around text-[#fff]">
              <div
                v-for="item in enabledSpaceStatisticsList"
                :key="item.key"
                class="flex flex-col items-center"
              >
                <span class="text-[14px] text-align-center">{{ item.label }}</span>
                <span class="font-black text-[24px] text-align-center">{{ item.value }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <Skeleton
        class="p-[24px] shadow-md"
        :loading="skeletonLoading"
      >
        <template #loading>
          <FlexRow>
            <template #left>
              <Layout.shape :width="110" />
            </template>
            <template #right>
              <div class="flex">
                <Layout.shape
                  class="mr-[10px]"
                  :width="176"
                />
                <Layout.shape :width="400" />
              </div>
            </template>
          </FlexRow>
          <Layout.table class="mt-[10px]" />
        </template>
        <div class="rounded-lg p-[24px] bg-[#fff] shadow-md">
          <FlexRow>
            <template #left>
              <Button
                theme="primary"
                @click="isShowCreateSpace = true"
              >
                {{ $t('新建空间') }}
              </Button>
            </template>
            <template #right>
              <div class="flex">
                <Tab
                  v-model:active="spaceStore.statusTab"
                  class="mr-[10px]"
                  :tabs="tabList"
                >
                </Tab>
                <SearchSelect
                  v-model="searchValue"
                  class="w-[520px] bg-[#fff] relative z-[100]"
                  :data="spaceSearchData"
                  :placeholder="
                    createPlaceholder({
                      type: 'searchSelect',
                      labels: ['空间 ID', '空间名称'],
                    })
                  "
                  unique-select
                  value-behavior="need-key"
                >
                </SearchSelect>
              </div>
            </template>
          </FlexRow>
          <Table
            class="mt-[16px] w-full"
            :data="curPageData"
            :max-height="maxHeight"
            :pagination="pagination"
            :row-config="{
              isHover: true,
              isCurrent: true,
            }"
            :row-height="56"
            :sort-config="sortConfig"
          >
            <template #empty>
              <TableException
                :type="curExceptionType"
                @clear="handleClearFilters"
                @refresh="getWorkspaceList"
              >
              </TableException>
            </template>
            <TableColumn
              field="name"
              :label="$t('空间名称(ID)')"
              show-overflow="tooltip"
              :width="200"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                <div class="leading-[20px]">
                  <Button
                    v-bk-tooltips="{
                      disabled: row?.state === spaceStore.spaceState.Ready,
                      content: $t('停用的空间无法访问'),
                    }"
                    :disabled="row?.state !== spaceStore.spaceState.Ready"
                    text
                    theme="primary"
                    @click="handleChangeSpace(row)"
                  >
                    {{ row.displayName }}
                  </Button>
                  <div class="text-[#9798A5]">{{ row.id || '--' }}</div>
                </div>
              </template>
            </TableColumn>
            <TableColumn
              field="description"
              :label="$t('空间描述')"
              :min-width="130"
              show-overflow="tooltip"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                {{ row.description || '--' }}
              </template>
            </TableColumn>
            <TableColumn
              field="app"
              :label="$t('应用')"
              show-overflow="tooltip"
              width="60"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                <template v-if="row.id && workspaceStaticData?.[row.id]?.appCount">
                  <Button
                    :disabled="row?.state !== spaceStore.spaceState.Ready"
                    text
                    theme="primary"
                    @click="handleChangeSpace(row)"
                  >
                    {{ workspaceStaticData?.[row.id].appCount }}
                  </Button>
                </template>
                <span v-else>--</span>
              </template>
            </TableColumn>
            <TableColumn
              field="env"
              :label="$t('环境')"
              show-overflow="tooltip"
              width="60"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                <template v-if="row.id && workspaceStaticData?.[row.id]?.envCount">
                  <Button
                    :disabled="row?.state !== spaceStore.spaceState.Ready"
                    text
                    theme="primary"
                    @click="handleChangeSpace(row, 'env')"
                  >
                    {{ workspaceStaticData?.[row.id].envCount }}
                  </Button>
                </template>
                <span v-else>--</span>
              </template>
            </TableColumn>
            <TableColumn
              field="state"
              :label="$t('状态')"
              show-overflow="tooltip"
              width="150"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                <div class="flex">
                  <PopConfirm
                    :confirm-text="row?.state === spaceStore.spaceState.Ready ? $t('停用') : $t('启用')"
                    :content="
                      row?.state === spaceStore.spaceState.Ready
                        ? $t('停用后将无法使用，请谨慎操作！')
                        : $t('启用后将恢复该空间的所有功能')
                    "
                    placement="top"
                    :title="
                      row?.state === spaceStore.spaceState.Ready ? $t('确定停用该空间？') : $t('确定启用该空间？')
                    "
                    trigger="click"
                    width="280"
                    @confirm="() => handleChangeStatus(row)"
                  >
                    <Switcher
                      :disabled="false"
                      theme="primary"
                      :value="row?.state === spaceStore.spaceState.Ready"
                    />
                    <span class="ml-[10px]">
                      {{ row?.state === spaceStore.spaceState.Ready ? $t('启用中') : $t('已停用') }}
                    </span>
                  </PopConfirm>
                </div>
              </template>
            </TableColumn>
            <TableColumn
              field="creator"
              :label="$t('创建者')"
              show-overflow="tooltip"
              width="110"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">{{ row?.creator || '--' }}</template>
            </TableColumn>
            <TableColumn
              field="updater"
              :label="$t('更新者')"
              show-overflow="tooltip"
              width="110"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">{{ row?.updater || '--' }}</template>
            </TableColumn>
            <TableColumn
              field="updatedAt"
              :label="$t('更新时间')"
              show-overflow="tooltip"
              sortable
              :width="200"
            >
              <template #default="{ row }: { row: WorkspaceInfoOutputObj }">
                {{ filterTimeFormat(row?.updatedAt || '') || '--' }}
              </template>
            </TableColumn>
          </Table>
        </div>
      </Skeleton>
    </div>

    <!-- 新增/编辑团队空间 -->
    <TeamSpace
      v-model:is-show="isShowCreateSpace"
      @confirm="getWorkspaceList"
    />
  </div>
</template>
<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, PopConfirm, SearchSelect, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { WorkspaceService } from '~/api/modules/v1';
  import { filterTimeFormat } from '~/common/util';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useDynamicsHeight from '~/composables/use-table-height';
  import { useSpaceStore } from '~/stores/space';

  import TeamSpace from './team-space.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type {
    UserStatisticsOutputObj,
    UserWorkspaceStatisticsOutputObj,
    WorkspaceInfoOutputObj,
  } from '~/@types/v1/workspace';
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const { maxHeight } = useDynamicsHeight(160, ['.bk-navigation-header', '.info-panel']);

  const skeletonLoading = ref(false);
  // 分页数据
  const pagination = ref({ count: 0, limit: 20, current: 1 });
  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
  });

  // 空间搜索配置
  const spaceSearchData = ref([
    {
      name: t('空间 ID'),
      id: t('空间 ID'),
      placeholder: t('空间 ID'),
    },
    {
      name: t('空间名称'),
      id: t('空间名称'),
      placeholder: t('空间名称'),
    },
  ]);
  const searchValue = ref<ISearchValue[]>([]);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });
  // 空间搜索
  const searchKey = ref('');
  const debounceSearch = useDebounce(searchKey, 300);

  const tabList = ref([
    { name: '', label: t('全部') },
    { name: spaceStore.spaceState.Ready, label: t('启用中') },
    { name: spaceStore.spaceState.Disabled, label: t('已停用') },
  ]);

  // 是否显示新建空间
  const isShowCreateSpace = ref(false);

  // 空间筛选
  const curPageData = computed<WorkspaceInfoOutputObj[]>(() =>
    spaceStore.list.filter(item => !spaceStore.statusTab || item.state === spaceStore.statusTab),
  );

  // 启用中空间的统计数据
  const enabledSpaceStatistics = computed(() => {
    const enabledSpaces = spaceStore.list.filter(item => item.state === spaceStore.spaceState.Ready);

    // 计算启用中空间的应用、环境总数
    let totalAppCount = 0;
    let totalEnvCount = 0;

    enabledSpaces.forEach(space => {
      if (!space.id) return;
      const spaceData = workspaceStaticData.value?.[space.id];
      if (spaceData) {
        totalAppCount += Number(spaceData.appCount) || 0;
        totalEnvCount += Number(spaceData.envCount) || 0;
      }
    });

    return {
      spaceCount: enabledSpaces.length,
      appCount: totalAppCount,
      envCount: totalEnvCount,
    };
  });

  const enabledSpaceStatisticsList = computed(() => [
    {
      key: 'space',
      label: t('空间'),
      value: enabledSpaceStatistics.value.spaceCount,
    },
    {
      key: 'app',
      label: t('应用'),
      value: enabledSpaceStatistics.value.appCount || '--',
    },
    {
      key: 'env',
      label: t('环境'),
      value: enabledSpaceStatistics.value.envCount || '--',
    },
  ]);

  // 获取空间列表
  async function getWorkspaceList() {
    try {
      await spaceStore.handleGetWorkspaceList({
        keyword: debounceSearch.value,
      });
      clearErrorType();
      // 根据当前 tab 状态设置分页总数
      pagination.value.count = curPageData.value.length;
    } catch (err) {
      console.error(err);
      setTypeToError();
    }
  }

  // 切换空间
  function handleChangeSpace(row: WorkspaceInfoOutputObj, routeName = 'app') {
    // 更新当前空间缓存
    spaceStore.updateCurrentSpace(row.id ?? '');
    // 跳转空间列表
    router.push({
      name: routeName,
      params: {
        space: row.id ?? '',
      },
    });
  }

  // 改变空间状态
  async function handleChangeStatus(row: WorkspaceInfoOutputObj) {
    try {
      const currentState = row.state;
      const targetState =
        currentState === spaceStore.spaceState.Ready ? spaceStore.spaceState.Disabled : spaceStore.spaceState.Ready;
      await WorkspaceService.setWorkspaceState({
        workspaceID: row.id ?? '',
        state: targetState,
      });
      await getWorkspaceList();
    } catch (error) {
      console.error('Failed to change workspace status:', error);
    }
  }
  // 清除筛选并搜索
  async function handleClearFilters() {
    searchValue.value = [];
    getWorkspaceList();
  }

  // 空间数据统计
  const staticData = ref<UserStatisticsOutputObj>({
    appCount: '',
    envCount: '',
    workspaceCount: '',
    workspaceStatistics: [],
  });
  const workspaceStaticData = computed(() =>
    staticData.value.workspaceStatistics?.reduce<Record<string, UserWorkspaceStatisticsOutputObj>>((pre, item) => {
      if (!item.workspaceID) return pre;
      pre[item.workspaceID] = {
        ...item,
        appCount: item.appCount,
        envCount: item.envCount,
      };
      return pre;
    }, {}),
  );

  async function handleGetUserStatic() {
    staticData.value = await WorkspaceService.getUserStatistics().catch(error => {
      console.error('Failed to get user statistics:', error);
      return {
        appCount: '',
        envCount: '',
        workspaceCount: '',
        workspaceStatistics: [],
      };
    });
  }

  // 搜索（防抖）
  watch(debounceSearch, async () => {
    await getWorkspaceList();
    pagination.value.current = 1;
  });

  // SearchSelect 值变化时同步到 searchKey
  watch(searchValue, newValue => {
    if (newValue && newValue.length > 0 && newValue[0].values && newValue[0].values.length > 0) {
      searchKey.value = newValue[0].values[0].id || '';
    } else {
      searchKey.value = '';
    }
  });

  // 监听 tab 切换，更新分页数据
  watch(
    () => spaceStore.statusTab,
    () => {
      pagination.value.count = curPageData.value.length;
      pagination.value.current = 1;
    },
  );

  onMounted(async () => {
    spaceStore.handleChangeStatusTab('Ready');
    skeletonLoading.value = true;
    await Promise.all([
      // 获取空间列表
      getWorkspaceList(),
      handleGetUserStatic(),
    ]);
    skeletonLoading.value = false;
  });
</script>
