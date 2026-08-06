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
    :class="[
      'flex-1 flex flex-col w-full px-[24px] py-[18px] overflow-y-hidden',
      componentType === 'AllData' && viewMode === 'card' ? '' : 'gap-[16px]',
    ]"
  >
    <Skeleton
      :loading="isLoading"
      theme="gray"
    >
      <template #loading>
        <FlexRow>
          <template #left>
            <Layout.shape
              class="mr-[10px]"
              :width="400"
            />
            <Layout.shape :width="400" />
          </template>
          <template #right>
            <Layout.shape :width="110" />
            <Layout.shape
              class="ml-[8px]"
              :width="80"
            />
          </template>
        </FlexRow>
        <div class="grid gap-[16px] auto-rows-[max-content] cols-[repeat(auto-fill,_minmax(380px,_1fr))] mt-[16px]">
          <Layout.shape
            v-for="index in 8"
            :key="index"
            height="100%"
            type="rect"
            width="100%"
          >
            <template #content>
              <Layout.paragraph />
            </template>
          </Layout.shape>
        </div>
      </template>
      <FlexRow average>
        <template #left>
          <div class="flex items-center gap-[8px]">
            <Button
              theme="primary"
              @click="handleCreateComponent('component')"
            >
              <Plus
                height="24"
                width="24"
              />
              {{ t('新建组件') }}
            </Button>
            <Radio.Group
              v-model="componentType"
              type="capsule"
            >
              <Radio.Button
                v-for="(value, key) in ProcessedData"
                :key="key"
                :label="key"
              >
                <span>{{ value.label }}</span>
                <span
                  :class="[
                    'h-[16px] leading-[16px] ml-[4px] px-[6px] rounded-[8px]',
                    componentType === key ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-[#fff]',
                  ]"
                >
                  {{ value.value.length }}
                </span>
              </Radio.Button>
            </Radio.Group>
          </div>
        </template>
        <template #right>
          <div class="flex items-center justify-end gap-[8px] ml-[8px]">
            <SearchSelect
              v-model="searchValue"
              class="w-[520px] bg-[#fff] relative z-[100]"
              :data="componentSearchData"
              :placeholder="
                createPlaceholder({
                  type: 'searchSelect',
                  labels: ['ID', '使用范围'],
                })
              "
              unique-select
              value-behavior="need-key"
            >
            </SearchSelect>
            <Radio.Group
              v-model="viewMode"
              class="square-radio-button"
              type="capsule"
            >
              <Radio.Button
                v-bk-tooltips="$t('卡片模式')"
                label="card"
              >
                <i class="bkms-icon text-[14px] bkms-icon-shitu-lianglan"></i>
              </Radio.Button>
              <Radio.Button
                v-bk-tooltips="$t('列表模式')"
                label="list"
              >
                <i class="bkms-icon text-[14px] bkms-icon-shitu-liebiao"></i>
              </Radio.Button>
            </Radio.Group>
          </div>
        </template>
      </FlexRow>
      <div
        ref="marketplaceTableContentRef"
        class="flex-1 min-h-0 overflow-y-auto"
      >
        <!-- 卡片模式 -->
        <SpaceComCardGrid
          v-if="viewMode === 'card'"
          :data="ProcessedData"
          :type="componentType"
          @delete="handleDeleteComponent"
          @edit="handleEditComponent"
        >
          <template #empty>
            <TableException
              v-if="ProcessedData[componentType].value.length === 0"
              class="mt-[40px] custom-empty"
              :type="curExceptionType"
              @clear="searchValue = []"
              @refresh="fetchComponentList"
            >
              <template #type>
                <img
                  :height="200"
                  src="/empty.svg"
                  :width="440"
                />
              </template>
            </TableException>
          </template>
        </SpaceComCardGrid>
        <!-- 列表模式 -->
        <Table
          v-else-if="viewMode === 'list'"
          ref="ComponentTableRef"
          auto-resize
          :data="ProcessedData[componentType].value"
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
            </template>
          </TableColumn>
          <TableColumn
            field="scopeType"
            filter-multiple
            :filters="filterOptions.scopeType"
            :label="t('使用范围')"
            width="150"
          >
            <template #default="{ row }">
              <CustomTag :theme="row.scopeType === 'global' ? 'success' : 'default'">
                {{ row.scopeType === 'global' ? $t('所有空间可见') : t('仅本空间可见') }}
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
            field="appCompInstanceCount"
            :label="t('实例数量')"
            width="100"
          >
            <template #default="{ row }">
              {{ row.appCompInstanceCount != null ? row.appCompInstanceCount : '--' }}
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
              <div
                v-if="row.managedByWorkspaceIDs?.some((item: string) => item === spaceStore.currentSpace)"
                class="flex items-center gap-[16px]"
              >
                <Button
                  text
                  theme="primary"
                  @click.stop="handleEditComponent(row)"
                >
                  {{ t('编辑') }}
                </Button>
                <!-- 删除 -->
                <div
                  v-bk-tooltips="{
                    content: t('组件已被应用使用，不支持删除'),
                    disabled: !((row.appCompInstanceCount ?? 0) > 0),
                  }"
                >
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
                      :disabled="(row.appCompInstanceCount ?? 0) > 0"
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
                </div>
              </div>
              <span v-else>--</span>
            </template>
          </TableColumn>
        </Table>
      </div>
    </Skeleton>
    <ComponentManagement
      ref="componentManagementRef"
      @refresh="fetchComponentList"
    >
    </ComponentManagement>
    <!-- 组件详情 -->
    <!-- <ComponentDetail
      ref="ComponentDetailRef"
      :allowed-range="activeTab === 'space' ? spaceStore.currentSpace : ''"
      :row="currentRow"
      @refresh="fetchComponentList"
    >
    </ComponentDetail> -->
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onMounted, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Message, PopConfirm, Radio, SearchSelect } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useElementHeight } from '~/composables/use-element-height';
  import { useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useSpaceStore } from '~/stores/space';

  import ComponentManagement from '../../marketplace/component-management.vue';
  import SpaceComCardGrid from './space-com-card-grid.vue';

  import type { ComponentDefOutputObj } from '~/@types/v1/component-defs';
  import type { ListComponentDefsRequest } from '~/@types/v1/component-defs';

  /** 不支持分类的参数 */
  type UnSupportCategoryParams = Omit<ListComponentDefsRequest, never>;
  // 引入国际化
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const spaceStore = useSpaceStore();
  const { formatDateString } = useTime();

  const viewMode = ref<'card' | 'list'>('card');

  const isLoading = ref(false);
  const componentList = ref<ComponentDefOutputObj[]>([]);
  // 分页数据
  const pagination = ref({ count: 0, limit: 20, current: 1 });

  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
  });

  // ref
  const ComponentTableRef = ref();
  const componentManagementRef = ref();
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
      name: t('使用范围'),
      id: 'scopeType',
      multiple: true,
      placeholder: t('请选择使用范围'),
      children: [
        { name: t('所有空间可见'), id: 'global' },
        { name: t('仅本空间可见'), id: 'workspace' },
      ],
      field: 'scopeType',
    },
  ]);

  const { filterChangeEvent, searchValue, tableDataMatchSearch, filterOptions } = useTableSearchSelect(
    componentList,
    componentSearchData,
    {
      tableRef: ComponentTableRef,
    },
  );
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  // 分类处理后的组件
  const ProcessedData = computed(() => {
    // 本空间组件
    const PersonalSpaceData = [];
    // 内置组件
    const BuiltinData = [];
    // 共享组件
    const ShareData = [];
    for (let i = 0; i < tableDataMatchSearch.value.length; i++) {
      const dataItem = tableDataMatchSearch.value[i];
      if (dataItem.managedByWorkspaceIDs?.some(item => item === spaceStore.currentSpace)) {
        PersonalSpaceData.push(dataItem);
      } else if (dataItem.isBuiltin) {
        BuiltinData.push(dataItem);
      } else if (dataItem.managedByWorkspaceIDs?.some(item => item != spaceStore.currentSpace)) {
        ShareData.push(dataItem);
      }
    }
    return {
      AllData: {
        label: t('全部'),
        value: tableDataMatchSearch.value,
      },
      PersonalSpaceData: {
        label: t('本空间'),
        value: PersonalSpaceData,
      },
      BuiltinData: {
        label: t('内置'),
        value: BuiltinData,
      },
      ShareData: {
        label: t('其他空间共享'),
        value: ShareData,
      },
    };
  });

  export type ProcessedDataKey = keyof ProcessedDataType;
  export type ProcessedDataType = typeof ProcessedData.value;

  const componentType = ref<ProcessedDataKey>('AllData');

  // 获取组件列表
  async function fetchComponentList(showLoading = true) {
    try {
      if (showLoading) {
        isLoading.value = true;
      }
      // 获取组件列表接口替换为： ApiServerService.ListComponentDefs
      // 空间组件
      componentList.value = await ApiServerService.ListComponentDefs({
        scopeWorkspaceID: spaceStore.currentSpace,
      } as UnSupportCategoryParams).catch(() => []);

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

  function handleCreateComponent(type: 'component' | 'version') {
    nextTick(() => {
      componentManagementRef.value?.open?.(type);
    });
  }

  // 删除组件
  async function handleDeleteComponent(row: ComponentDefOutputObj) {
    if ((row.appCompInstanceCount ?? 0) > 0) return;
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
      componentManagementRef.value?.open?.('edit', row);
    });
  }

  watch(searchValue, () => {
    pagination.value.current = 1;
  });

  watch(componentType, () => {
    pagination.value.current = 1;
    pagination.value.count = ProcessedData.value[componentType.value].value.length;
  });

  // 操作记录列表变化时，更新总数
  watch(tableDataMatchSearch, newValue => {
    pagination.value.count = newValue.length;
  });

  onMounted(() => {
    fetchComponentList();
  });
</script>

<style lang="postcss" scoped>
  :deep(.custom-empty) {
    .bk-exception-img {
      width: 440px;
      height: 200px;
    }
    .bk-exception-title {
      color: #4d4f56;
    }
  }
  .square-radio-button {
    :deep(.bk-radio-button-label) {
      padding: 0 5px;
    }
  }
</style>
