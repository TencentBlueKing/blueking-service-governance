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
  <div class="flex flex-col gap-[16px] h-full">
    <Tab
      v-model:active="activeTab"
      class="relative z-0"
      type="card-grid"
      @change="handleActiveTabChange"
    >
      <Tab.TabPanel
        :disabled="isLoading"
        :label="t('空间组件')"
        :name="TAB_NAMES.space"
      >
        <template #label>
          <span class="text-[14px] text-[#4D4F56]">
            {{ t('空间组件') }}
          </span>
        </template>
      </Tab.TabPanel>
      <Tab.TabPanel
        :disabled="isLoading"
        :label="t('市场组件')"
        :name="TAB_NAMES.marketplace"
      >
        <template #label>
          <span class="text-[14px] text-[#4D4F56]"> {{ t('市场组件') }}</span>
        </template>
      </Tab.TabPanel>
      <!-- tab背景 -->
      <div class="absolute w-[calc(100%+48px)] h-[54px] top-[-14px] left-[-24px] bg-[#EAEBF0] z-[-1]"></div>
    </Tab>
    <Skeleton :loading="isLoading">
      <template #loading>
        <FlexRow>
          <template #left>
            <Layout.shape
              class="mr-[10px]"
              :width="110"
            />
          </template>
          <template #right>
            <Layout.shape :width="400" />
          </template>
        </FlexRow>
        <Layout.table class="mt-[16px]" />
      </template>
      <FlexRow average>
        <template #left>
          <div class="flex items-center gap-[8px]">
            <Button
              v-if="activeTab === TAB_NAMES.space"
              theme="primary"
              @click="handleCreateComponent('component')"
            >
              <Plus
                height="24"
                width="24"
              />
              {{ t('新建组件') }}
            </Button>
          </div>
        </template>
        <template #right>
          <div class="flex items-center justify-end">
            <SearchSelect
              v-model="searchValue"
              class="w-[520px] bg-[#fff] relative z-[100]"
              :data="componentSearchData"
              :placeholder="
                createPlaceholder({
                  type: 'searchSelect',
                  labels: ['component.label.ID', 'component.label.isPublic', 'component.label.version2'],
                })
              "
              unique-select
              value-behavior="need-key"
            >
            </SearchSelect>
          </div>
        </template>
      </FlexRow>
      <div
        ref="marketplaceTableContentRef"
        class="flex-1 overflow-auto"
      >
        <Table
          ref="ComponentTableRef"
          auto-resize
          :data="tableDataMatchSearch"
          :filter-config="{ remote: true }"
          :max-height="marketplaceTableContentHeight"
          :pagination="pagination"
          :row-config="{
            isHover: true,
            isCurrent: true,
          }"
          :sort-config="sortConfig"
          sync-resize
          @filter-change="filterChangeEvent"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="searchValue = []"
              @refresh="fetchComponentList"
            >
            </TableException>
          </template>
          <TableColumn
            field="name"
            label="ID"
            min-width="150"
          >
            <template #default="{ row }">
              {{ row.name || '--' }}
              <Tag
                v-if="row.isBuiltin"
                class="ml-[10px]"
                theme="success"
              >
                {{ t('官方') }}
              </Tag>
            </template>
          </TableColumn>
          <TableColumn
            v-if="activeTab === TAB_NAMES.space"
            field="public"
            filter-multiple
            :filters="isPublicOptions"
            :label="t('是否公开')"
            width="150"
          >
            <template #default="{ row }">
              <CustomTag :theme="row.public ? 'success' : 'default'">
                {{ row.public ? t('公开') : t('不公开') }}
              </CustomTag>
            </template>
          </TableColumn>
          <TableColumn
            field="definition.description"
            :label="t('描述')"
            min-width="200"
          >
            <template #default="{ row }">
              {{ row.description || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            field="updatedBy"
            :label="t('更新人')"
            width="150"
          >
            <template #default="{ row }">
              {{ row.updater || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            field="updatedTime"
            :label="t('更新时间')"
            min-width="150"
            sortable
          >
            <template #default="{ row }">
              {{ row.updatedAt ? formatDateString(row.updatedAt) : '--' }}
            </template>
          </TableColumn>
          <TableColumn
            fixed="right"
            :label="t('操作')"
            min-width="150"
          >
            <template #default="{ row }">
              <div class="flex items-center gap-[16px]">
                <Button
                  text
                  theme="primary"
                  @click.stop="handleEditComponent(row)"
                >
                  {{ t('编辑') }}
                </Button>
                <template v-if="activeTab === TAB_NAMES.space">
                  <!-- 删除 -->
                  <PopConfirm
                    :confirm-config="{
                      theme: 'danger',
                    }"
                    :confirm-text="t('删除')"
                    placement="bottom"
                    :title="t('确认删除该组件？')"
                    trigger="click"
                    @confirm="handleDeleteComponent(row)"
                  >
                    <Button
                      text
                      theme="primary"
                      @click.stop
                    >
                      {{ t('删除') }}
                    </Button>
                    <template #content>
                      <div class="flex flex-col mb-[16px]">
                        <span>{{ t('组件名称: {name}', { name: row.displayName || row.name }) }}</span>
                        <span>{{ t('删除后，将不可恢复，请谨慎操作！') }}</span>
                      </div>
                    </template>
                  </PopConfirm>
                </template>
              </div>
            </template>
          </TableColumn>
        </Table>
      </div>
    </Skeleton>
    <ComponentManagement
      ref="CreateComponentRef"
      @refresh="fetchComponentList"
    >
    </ComponentManagement>
    <!-- 组件详情 -->
    <!-- <ComponentDetail
      ref="ComponentDetailRef"
      :allowed-range="activeTab === TAB_NAMES.space ? spaceStore.currentSpace : ''"
      :row="currentRow"
      @refresh="fetchComponentList"
    >
    </ComponentDetail> -->
  </div>
</template>

<script lang="ts" setup>
  import { nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Message, PopConfirm, SearchSelect, Tab, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useElementHeight } from '~/composables/use-element-height';
  import useInterval from '~/composables/use-interval';
  import { useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useUrlActiveTab } from '~/composables/use-url-active-tab';
  import { useSpaceStore } from '~/stores/space';

  import ComponentManagement from './component-management.vue';

  import type { ComponentDefOutputObj } from '~/@types/v1/component-defs';

  // Tab 名称常量（模板与校验同源）
  const TAB_NAMES = {
    marketplace: 'marketplace',
    space: 'space',
  } as const;
  type IActive = (typeof TAB_NAMES)[keyof typeof TAB_NAMES];
  // 引入国际化
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const spaceStore = useSpaceStore();
  const { formatDateString } = useTime();

  // 过滤
  // 是否公开列表
  const isPublicOptions = shallowRef([
    { label: t('公开'), value: 'true' },
    { label: t('不公开'), value: 'false' },
  ]);

  // Tab 与 URL query（active）双向同步锚定
  const { fields } = useUrlActiveTab({
    activeTab: {
      queryKey: 'active',
      tabValues: Object.values(TAB_NAMES),
      defaultTab: TAB_NAMES.space,
    },
  });
  const activeTab = fields.activeTab;
  const isLoading = ref(false);
  const componentList = ref<ComponentDefOutputObj[]>([]);
  // 分页数据
  const pagination = ref({ count: 0, limit: 20, current: 1 });
  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
  });

  // ref
  /** vxe-table 实例引用，用于筛选联动 */
  interface ITableRef {
    getVxeTableInstance?: () => { clearFilter?: () => void; setFilter?: (...args: unknown[]) => void };
  }
  const ComponentTableRef = ref<ITableRef>();
  const CreateComponentRef = ref<InstanceType<typeof ComponentManagement>>();
  const marketplaceTableContentRef = ref<HTMLElement>();

  // 使用 hooks 获取表格容器高度
  const { height: marketplaceTableContentHeight } = useElementHeight(marketplaceTableContentRef, {
    watchSource: isLoading,
    defaultHeight: 600,
  });

  const componentSearchData = shallowRef([
    {
      name: 'ID',
      id: 'ID',
      multiple: false,
      placeholder: t('请输入ID'),
      field: 'name',
      fuzzy: true,
    },
    {
      name: t('是否公开'),
      id: t('是否公开'),
      multiple: true,
      placeholder: t('请选择是否公开'),
      children: isPublicOptions.value.map(item => ({ name: item.label, id: item.value })),
      field: 'public',
    },
    {
      name: t('版本信息'),
      id: t('版本'),
      multiple: false,
      placeholder: t('请输入版本'),
      field: 'version',
      fuzzy: true,
    },
  ]);
  const { filterChangeEvent, searchValue, tableDataMatchSearch } = useTableSearchSelect(
    componentList,
    componentSearchData,
    {
      tableRef: ComponentTableRef,
    },
  );
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });
  const { start, stop } = useInterval(() => fetchComponentList(false), 10000); // 轮询

  // 获取组件列表
  async function fetchComponentList(showLoading = true) {
    try {
      if (showLoading) {
        isLoading.value = true;
      }
      // 获取组件列表接口替换为： ApiServerService.ListComponentDefs
      if (activeTab.value === TAB_NAMES.space) {
        // 空间组件
        const list = await ApiServerService.ListComponentDefs({}).catch(() => []);
        componentList.value = list.filter(item => item.scopeWorkspaceIDs?.includes(spaceStore.currentSpace));
      } else {
        // 市场组件
        componentList.value = await ApiServerService.ListComponentDefs({
          scopeWorkspaceID: spaceStore.currentSpace,
        }).catch(() => []);
      }

      // 按创建时间降序排序，新创建的显示在最前方
      componentList.value.sort((a, b) => {
        const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
        const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
        return timeB - timeA;
      });
      clearErrorType();
    } catch (error) {
      console.error(error);
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  }

  // 切换 Tab（activeTab setter 会同步 URL query）
  function handleActiveTabChange(active: IActive) {
    activeTab.value = active;
    fetchComponentList();
  }

  function handleCreateComponent(type: 'component' | 'edit') {
    nextTick(() => {
      CreateComponentRef.value?.open?.(type);
    });
  }

  // 删除组件
  async function handleDeleteComponent(row: ComponentDefOutputObj) {
    const result = await ApiServerService.DeleteComponentDef({
      compDefName: row.name ?? '',
    })
      .then(() => true)
      .catch(() => false);
    if (result) {
      Message({
        theme: 'success',
        message: t('删除成功'),
      });
      fetchComponentList();
    }
  }

  // 编辑组件
  function handleEditComponent(row: ComponentDefOutputObj) {
    nextTick(() => {
      CreateComponentRef.value?.open?.('edit', row);
    });
  }

  watch(searchValue, () => {
    pagination.value.current = 1;
  });

  // 操作记录列表变化时，更新总数
  watch(tableDataMatchSearch, newValue => {
    pagination.value.count = newValue.length;
  });

  onMounted(() => {
    fetchComponentList();
    start();
  });

  onBeforeUnmount(() => {
    stop();
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-tab-content) {
    padding: 0 !important;
  }

  :deep(.bk-tab-header--active) {
    span {
      color: #3a84ff;
    }
  }

  :deep(.bk-tab-header-item:not(.bk-tab-header--active)) {
    background-color: #dcdee5;
  }
</style>
