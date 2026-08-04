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
  <main class="flex-1 flex min-h-0">
    <Skeleton
      :loading="isLoading"
      theme="gray"
    >
      <template #loading>
        <div class="flex w-full h-full">
          <div class="w-[280px]">
            <Layout.shape
              height="100%"
              width="100%"
            />
          </div>
          <div class="flex-1 px-[24px] py-[14px]">
            <Layout.shape :width="200" />
            <Layout.shape
              class="mt-[16px]"
              width="100%"
            />
            <FlexRow class="my-[16px]">
              <template #left>
                <Layout.shape :width="110" />
              </template>
              <template #right>
                <Layout.shape :width="300" />
              </template>
            </FlexRow>
            <Layout.table />
          </div>
        </div>
      </template>
      <ResizeLayout
        :border="false"
        class="flex-1 min-w-0 min-h-0"
        collapsible
        :initial-divide="280"
        :max="400"
        :min="280"
      >
        <template #aside>
          <!-- 左侧树形菜单 -->
          <div class="flex-1 bg-[#F5F7FA] flex flex-col">
            <div class="px-[16px] py-[12px]">
              <Radio.Group
                v-model="viewMode"
                class="w-full flex mb-[14px]"
                type="capsule"
              >
                <Radio.Button
                  class="flex-1"
                  label="component"
                >
                  <i class="bkms-icon bkms-icon-single-column text-[14px]"></i>
                  <span class="text-[14px] pl-[4px]">{{ $t('组件视角') }}</span>
                </Radio.Button>
                <Radio.Button
                  class="flex-1"
                  label="environment"
                >
                  <i class="bkms-icon bkms-icon-single-column text-[14px]"></i>
                  <span class="text-[14px] pl-[4px]">{{ $t('环境视角') }}</span>
                </Radio.Button>
              </Radio.Group>
              <Input
                v-model.trim="treeSearchValue"
                clearable
                :placeholder="viewMode === 'component' ? $t('搜索组件名称') : $t('搜索环境名称')"
                type="search"
              />
            </div>
            <!-- 搜索为空显示 -->
            <Exception
              v-show="filteredTreeData.length === 0"
              :type="treeSearchValue ? 'search' : 'empty'"
              @clear="handleClearFilters"
            >
            </Exception>
            <div
              v-show="filteredTreeData.length !== 0"
              :class="[
                'border-b-[1px] border-b-[#EAEBF0]',
                'flex items-center justify-between px-[16px] h-[36px] cursor-pointer',
                selectedTreeNode === 'all' ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'text-[#4D4F56]',
                { 'hover:bg-[#f0f1f5]': selectedTreeNode !== 'all' },
              ]"
              @click="handleTreeNodeClick('all')"
            >
              <div class="flex items-center flex-1">
                <i class="bkms-icon bkms-icon-quanbu-xuanzhong"></i>
                <span class="text-[12px] truncate ml-[4px]">{{ $t('全部') }}</span>
              </div>
              <span
                :class="[
                  'text-[12px] text-[#979BA5] px-[8px] rounded-[2px]',
                  selectedTreeNode === 'all' ? 'bg-[#3A84FF] text-[#fff]' : 'text-[#4D4F56] bg-[#F0F1F5]',
                ]"
                >{{ componentList.length }}</span
              >
            </div>
            <div class="flex-1 overflow-auto mt-[8px] mb-[16px]">
              <div
                v-for="item in filteredTreeData"
                :key="item.id"
                :class="[
                  'flex items-center justify-between min-h-[36px] px-[16px] py-[4px] cursor-pointer',
                  selectedTreeNode === item.id ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'text-[#4D4F56]',
                  { 'hover:bg-[#f0f1f5]': selectedTreeNode !== item.id },
                ]"
                @click="handleTreeNodeClick(item.id)"
              >
                <div class="flex flex-col flex-1 min-w-0 gap-[4px]">
                  <div class="flex items-center">
                    <span class="text-[12px] truncate">{{ item.displayName || item.name }}</span>
                    <Tag
                      v-if="item.type && viewMode === 'environment'"
                      :class="['ml-[8px] pointer-events-none', envTypeTagClassMap[item.type]]"
                      size="small"
                    >
                      {{ envTypeMap[item.type].name }}
                    </Tag>
                  </div>
                  <span
                    v-if="item.type && viewMode === 'environment'"
                    class="block text-[#979BA5]"
                  >
                    {{ item.name }}
                  </span>
                </div>
                <span
                  :class="[
                    'text-[12px] text-[#979BA5] px-[8px] rounded-[2px] shrink-0',
                    selectedTreeNode === item.id ? 'bg-[#3A84FF] text-[#fff]' : 'text-[#4D4F56] bg-[#F0F1F5]',
                  ]"
                  >{{ item.count }}</span
                >
              </div>
            </div>
          </div>
        </template>
        <template #main>
          <!-- 右侧内容区 -->
          <div class="flex-1 flex flex-col overflow-hidden px-[24px] py-[14px] bg-[#fff]">
            <!-- 标题和提示 -->
            <div class="flex items-center mb-[12px]">
              <h3 class="text-[16px] font-bold">
                {{ currentTreeNode?.displayName || currentTreeNode?.name || $t('全部') }} ( {{ count }} )
              </h3>
              <template v-if="currentTreeNode?.type">
                <span class="mx-[8px] text-[#979BA5]">( {{ currentTreeNode.name }} )</span>
                <Tag
                  :class="envTypeTagClassMap[currentTreeNode.type]"
                  size="small"
                >
                  {{ envTypeMap[currentTreeNode.type].name }}
                </Tag>
              </template>
            </div>
            <Alert
              class="mb-[12px]"
              theme="info"
              >{{ $t('在空间上配置的组件需要应用主动引用才能生效') }}</Alert
            >

            <!-- 操作栏 -->
            <FlexRow class="mb-[12px]">
              <template #left>
                <div class="flex items-center gap-[8px]">
                  <Button
                    theme="primary"
                    @click="handleAddComponent"
                  >
                    <Plus
                      :height="24"
                      :width="24"
                    />
                    {{ $t('添加组件实例') }}
                  </Button>
                  <!-- <Button
                :disabled="!selections.length"
                @click="handleBatchDelete"
              >
                {{ $t('批量删除') }}
              </Button> -->
                </div>
              </template>
              <template #right>
                <SearchSelect
                  v-model="searchValue"
                  class="w-[520px] bg-[#fff] relative z-[100]"
                  :data="searchSelectData"
                  :placeholder="
                    createPlaceholder({
                      type: 'searchSelect',
                      labels:
                        viewMode === 'component'
                          ? ['实例名称', '组件', '可用环境', '已使用的应用']
                          : ['实例名称', '组件', '已使用的应用'],
                    })
                  "
                  unique-select
                  value-behavior="need-key"
                >
                </SearchSelect>
              </template>
            </FlexRow>

            <!-- 表格 -->
            <div
              ref="tableRef"
              class="flex-1 min-h-[0px]"
            >
              <Table
                ref="tableComponentRef"
                :data="tableDataMatchSearch"
                :max-height="tableHeight"
                :pagination="pagination"
                :row-config="{
                  keyField: 'name',
                  isHover: true,
                  isCurrent: true,
                }"
                :show-overflow="false"
                :virtual-y-config="{ enabled: true, gt: 0 }"
                @checkbox-all="handleCheckboxAll"
                @checkbox-change="handleCheckboxChange"
                @filter-change="filterChangeEvent"
                @page-limit-change="pageSizeChange"
                @page-value-change="pageChange"
              >
                <template #empty>
                  <TableException
                    :type="curExceptionType"
                    @clear="searchValue = []"
                    @refresh="getComponentList"
                  >
                  </TableException>
                </template>

                <!-- 复选框列 -->
                <!-- <TableColumn
                      fixed="left"
                      type="checkbox"
                      width="50"
                    /> -->

                <TableColumn
                  field="name"
                  fixed="left"
                  :label="$t('实例名称')"
                  min-width="150"
                >
                  <template #default="{ row }">
                    {{ row.name || '--' }}
                  </template>
                </TableColumn>

                <TableColumn
                  field="type"
                  filter-multiple
                  :filters="filterOptions.type"
                  :label="$t('组件')"
                  min-width="150"
                >
                  <template #header>
                    <CustomFilter
                      field="type"
                      :filters="filterOptions.type"
                      :label="$t('组件')"
                      :table-ref="tableComponentRef"
                    />
                  </template>
                  <template #default="{ row }">
                    {{ row.type || '--' }}
                  </template>
                </TableColumn>

                <TableColumn
                  field="properties"
                  :label="$t('配置内容')"
                  min-width="250"
                >
                  <template #default="{ row }">
                    <div
                      v-if="Object.keys(row.properties || {}).length"
                      v-bk-tooltips="{
                        content: propertiesToText(row.properties),
                        disabled: Object.keys(row.properties || {}).length < 4,
                        placement: 'left',
                        delay: 500,
                      }"
                      class="flex flex-col gap-[2px]"
                    >
                      <span
                        v-for="key in Object.keys(row.properties || {}).slice(0, 3)"
                        :key="key"
                        class="text-[12px] text-[#4D4F56] truncate"
                      >
                        {{ `${key}: ${formatPropertyValue(row.properties[key])}` }}
                      </span>
                      <span
                        v-if="Object.keys(row.properties || {}).length > 3"
                        class="text-[12px] text-[#979BA5]"
                      >
                        ...
                      </span>
                    </div>
                    <span v-else>--</span>
                  </template>
                </TableColumn>

                <TableColumn
                  v-if="viewMode === 'component'"
                  field="scopeEnvNames"
                  :filters="filterOptions.scopeEnvNames"
                  :label="$t('可用环境')"
                  min-width="180"
                >
                  <template #header>
                    <CustomFilter
                      field="scopeEnvNames"
                      :filters="filterOptions.scopeEnvNames"
                      :label="$t('可用环境')"
                      :table-ref="tableComponentRef"
                    />
                  </template>
                  <template #default="{ row }">
                    <Tag
                      v-if="row.scopeType === 'global'"
                      class="pointer-events-none"
                      theme="danger"
                      >{{ $t('所有环境') }}</Tag
                    >
                    <div
                      v-else-if="row.scopeEnvNames?.length"
                      class="flex items-center gap-[4px] flex-wrap"
                    >
                      <Tag
                        v-for="(env, index) in row.scopeEnvNames.slice(0, 2)"
                        :key="index"
                        class="pointer-events-none"
                      >
                        {{ env }}
                      </Tag>
                      <Popover theme="light">
                        <Tag v-if="row.scopeEnvNames.length > 2"> +{{ row.scopeEnvNames.length - 2 }} </Tag>
                        <template #content>
                          <div class="max-w-[350px] flex flex-wrap gap-[4px]">
                            <Tag
                              v-for="(env, index) in row.scopeEnvNames.slice(2)"
                              :key="index"
                            >
                              {{ env }}
                            </Tag>
                          </div>
                        </template>
                      </Popover>
                    </div>
                    <span v-else>--</span>
                  </template>
                </TableColumn>

                <TableColumn
                  field="refAppIDs"
                  :filters="filterOptions.refAppIDs"
                  :label="$t('已使用的应用')"
                  min-width="200"
                >
                  <template #header>
                    <CustomFilter
                      field="refAppIDs"
                      :filters="filterOptions.refAppIDs"
                      :label="$t('已使用的应用')"
                      :table-ref="tableComponentRef"
                    />
                  </template>
                  <template #default="{ row }">
                    <div
                      v-if="row.refAppIDs?.length"
                      class="flex items-center gap-[4px] flex-wrap"
                    >
                      <Tag
                        v-for="(env, index) in row.refAppIDs.slice(0, 2)"
                        :key="index"
                        class="pointer-events-none"
                      >
                        {{ env }}
                      </Tag>
                      <Popover theme="light">
                        <Tag v-if="row.refAppIDs.length > 2"> +{{ row.refAppIDs.length - 2 }} </Tag>
                        <template #content>
                          <div class="max-w-[350px] flex flex-wrap gap-[4px]">
                            <Tag
                              v-for="(env, index) in row.refAppIDs.slice(2)"
                              :key="index"
                            >
                              {{ env }}
                            </Tag>
                          </div>
                        </template>
                      </Popover>
                    </div>
                    <span v-else>--</span>
                  </template>
                </TableColumn>

                <!-- 操作列 -->
                <TableColumn
                  fixed="right"
                  :label="$t('操作')"
                  width="160"
                >
                  <template #default="{ row }">
                    <div class="flex items-center gap-[12px]">
                      <Button
                        text
                        theme="primary"
                        @click="handleEdit(row)"
                      >
                        {{ $t('编辑') }}
                      </Button>
                      <Button
                        text
                        theme="primary"
                        @click="handleClone(row)"
                      >
                        {{ $t('克隆') }}
                      </Button>
                      <Button
                        v-bk-tooltips="{
                          content: $t('组件已被应用使用，不支持删除'),
                          disabled: row.refAppIDs.length === 0,
                        }"
                        :disabled="row.refAppIDs.length > 0"
                        text
                        theme="primary"
                        @click="handleDelete(row)"
                      >
                        {{ $t('删除') }}
                      </Button>
                    </div>
                  </template>
                </TableColumn>
              </Table>
            </div>
          </div>
        </template>
      </ResizeLayout>
    </Skeleton>
  </main>
  <!-- 组件选择器 -->
  <ComponentsSideSlider
    v-model="isShow"
    :btn-loading="updating"
    :data="curComponent"
    module-type="space"
    :name-config="{
      disabled: !!curRow?.refAppIDs?.length && isEdit,
      message: $t('组件已被引用，不支持修改名称'),
    }"
    :title="isEdit ? t('编辑组件') : t('添加组件')"
    @close="handleCloseSlider"
    @submit="handleSubmit"
  />
</template>

<script setup lang="ts">
  import { computed, h, onBeforeMount, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, InfoBox, Input, Message, Popover, Radio, ResizeLayout, SearchSelect, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { type ComponentOutputObj as ComponentOutput } from '~/@types/v1/app';
  import { type EnvOutput } from '~/@types/v1/env';
  import { type WorkspaceComponentOutputObj } from '~/@types/v1/workspace-components';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import FlexRow from '~/components/flex-row.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import Exception from '~/components/table-exception.vue';
  import { useElementHeight } from '~/composables/use-element-height';
  import useEnvManager, { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import usePageConf from '~/composables/use-page';
  import { ISelectKey, ITableFilterItem, useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableCheckbox from '~/composables/use-table-checkbox';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useSpaceStore } from '~/stores/space';

  import type { ComponentData, ScopedType } from '~/components/modules/components-sideSlider.vue';

  // 环境节点数据结构
  interface EnvTreeNode {
    count: number;
    displayName?: string; // 环境展示名称
    id: string;
    name: string;
    type?: string; // 环境类型
  }

  type SpaceComponent = ComponentOutput & { scoped: { envs: string[]; type: ScopedType } };

  const { t } = useI18n();

  const spaceStore = useSpaceStore();
  const { envList, handleGetEnvList } = useEnvManager();
  const { createPlaceholder } = useSearchPlaceholder();

  // 状态管理
  const isLoading = ref(true);
  const tableComponentRef = ref();
  const viewMode = ref<'component' | 'environment'>('component');

  // 树形菜单相关
  const treeSearchValue = ref('');
  const selectedTreeNode = ref('all');

  // 从 componentList 派生 treeData (组件视角)
  const componentTreeData = computed<EnvTreeNode[]>(() => {
    const typeCountMap = new Map<string, number>();

    componentList.value.forEach(item => {
      if (item.type) {
        typeCountMap.set(item.type, (typeCountMap.get(item.type) || 0) + 1);
      }
    });

    return Array.from(typeCountMap.entries()).map(([type, count]) => ({
      id: type,
      name: type,
      count,
    }));
  });

  // 从 componentList 派生环境树数据 (环境视角)
  const environmentTreeData = computed<EnvTreeNode[]>(() => {
    const envCountMap = new Map<string, number>();

    componentList.value.forEach(item => {
      if (item.scopeEnvNames && item.scopeEnvNames.length > 0) {
        // 特定环境的组件
        item.scopeEnvNames.forEach(envName => {
          envCountMap.set(envName, (envCountMap.get(envName) || 0) + 1);
        });
      }
    });

    // 创建环境名称到环境信息的映射
    const envMap = new Map<string, EnvOutput>();
    envList.value.forEach(env => {
      if (env.name) envMap.set(env.name, env);
    });

    // 添加具体环境节点，整合环境的 displayName 和 type
    const envNodes = Array.from(envCountMap.entries()).map(([envName, count]) => {
      const envInfo = envMap.get(envName);
      return {
        id: envName,
        name: envName,
        displayName: envInfo?.displayName,
        type: envInfo?.type,
        count,
      };
    });

    return envNodes;
  });

  // 根据视角模式选择对应的树数据
  const treeData = computed<EnvTreeNode[]>(() => {
    return viewMode.value === 'component' ? componentTreeData.value : environmentTreeData.value;
  });

  // 过滤后的树形数据
  const filteredTreeData = computed(() => {
    const searchValue = treeSearchValue.value.trim();
    if (!searchValue) return treeData.value;

    const lowerSearchValue = searchValue.toLowerCase();
    return treeData.value.filter(item => {
      const displayName = item.displayName?.toLowerCase() ?? '';
      const name = item.name.toLowerCase();
      return displayName.includes(lowerSearchValue) || name.includes(lowerSearchValue);
    });
  });

  // 当前选中节点
  const currentTreeNode = computed<EnvTreeNode | undefined>(() =>
    treeData.value.find(item => item.id === selectedTreeNode.value),
  );

  // 根据选中的树节点过滤后的组件列表
  const filteredComponentList = computed(() => {
    if (selectedTreeNode.value === 'all') {
      return componentList.value;
    }

    if (viewMode.value === 'component') {
      // 组件视角：按 type 过滤
      return componentList.value.filter(item => item.type === selectedTreeNode.value);
    } else {
      // 环境视角：按 scopeEnvNames 过滤
      if (selectedTreeNode.value === '__global__') {
        // 全局环境
        return componentList.value.filter(item => item.scopeType === 'global');
      } else {
        // 特定环境
        return componentList.value.filter(
          item => item.scopeEnvNames && item.scopeEnvNames.includes(selectedTreeNode.value),
        );
      }
    }
  });

  // 处理树节点点击
  function handleTreeNodeClick(nodeId: string) {
    selectedTreeNode.value = nodeId;
  }

  // 监听视角切换，重置选中节点
  watch(viewMode, () => {
    selectedTreeNode.value = 'all';
    treeSearchValue.value = '';
  });

  // 表格数据
  const componentList = ref<WorkspaceComponentOutputObj[]>([]);
  async function getComponentList() {
    isLoading.value = true;
    try {
      const list = await ApiServerService.ListWorkspaceComponents({
        workspaceID: spaceStore.currentSpace,
      });

      // 按创建时间降序排序，新创建的显示在最前方
      componentList.value = list.sort((a, b) => {
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

  // 表格高度
  const tableRef = ref<HTMLElement>();
  const { height: tableHeight } = useElementHeight(tableRef, {
    watchSource: isLoading,
    defaultHeight: 400,
  });

  // 搜索和分页
  const componentTypeOptions = computed<ITableFilterItem[]>(() => {
    // 使用 Set 优化去重性能
    const typeSet = new Set<string>();
    componentList.value.forEach(item => {
      if (item.type) {
        typeSet.add(item.type);
      }
    });
    return Array.from(typeSet).map(item => ({ name: item, id: item }));
  });
  const evnOptions = computed<ITableFilterItem[]>(() => {
    const envSet = new Set<string>();

    for (const component of componentList.value) {
      if (component.scopeType === 'global') {
        envSet.add(t('所有环境'));
      } else if (component.scopeEnvNames) {
        component.scopeEnvNames.forEach(env => envSet.add(env));
      }
    }

    return Array.from(envSet).map(item => ({ name: item, id: item }));
  });
  const appIDsOptions = computed<ITableFilterItem[]>(() => {
    const appIDsSet = new Set<string>();
    componentList.value.forEach(item => {
      if (item.refAppIDs && item.refAppIDs.length > 0) {
        item.refAppIDs.forEach(appID => appIDsSet.add(appID));
      }
    });
    return Array.from(appIDsSet).map(item => ({ name: item, id: item }));
  });

  const searchSelectData = computed<ISelectKey<WorkspaceComponentOutputObj>[]>(() => {
    const allSearchOptions: ISelectKey<WorkspaceComponentOutputObj>[] = [
      {
        name: t('实例名称'),
        id: 'name',
        multiple: false,
        placeholder: t('实例名称'),
        field: 'name',
        fuzzy: true,
      },
      {
        name: t('组件'),
        id: 'type',
        multiple: true,
        placeholder: t('组件'),
        field: 'type',
        children: componentTypeOptions.value,
      },
      {
        name: t('已使用的应用'),
        id: 'refAppIDs',
        multiple: true,
        placeholder: t('已使用的应用'),
        field: 'refAppIDs',
        children: appIDsOptions.value,
        handleFilter: (item, values) => values.some(v => item.refAppIDs?.includes(v.id) ?? false),
      },
    ];

    // 只有在 component 视图模式下才添加"可用环境"筛选
    if (viewMode.value === 'component') {
      allSearchOptions.push({
        name: t('可用环境'),
        id: 'scopeEnvNames',
        multiple: true,
        placeholder: t('可用环境'),
        field: 'scopeEnvNames',
        children: evnOptions.value,
        handleFilter: (item, values) => {
          // 检查是否选中了"所有环境"且组件是全局的
          if (values.some(v => v.id === t('所有环境')) && item.scopeType === 'global') {
            return true;
          }
          // 检查组件的环境是否有任意一个在选中的值中
          return values.some(v => item.scopeEnvNames?.includes(v.id) ?? false);
        },
      });
    }

    return allSearchOptions;
  });

  const { filterChangeEvent, searchValue, tableDataMatchSearch, filterOptions } = useTableSearchSelect(
    filteredComponentList,
    searchSelectData,
    {
      // filters: filterOptions,
      tableRef: tableComponentRef,
    },
  );
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });
  const count = computed(() => tableDataMatchSearch.value.length);
  const { pagination, pageChange, pageSizeChange } = usePageConf(
    tableDataMatchSearch,
    {
      current: 1,
      limit: 10,
      remote: false,
    },
    count,
  );

  // 复选框相关
  const { handleCheckboxAll, handleCheckboxChange } = useTableCheckbox(componentList, 'name');

  // 辅助函数
  function formatPropertyValue(value: object | string): string {
    if (typeof value === 'object') {
      return JSON.stringify(value);
    }
    return String(value);
  }

  // 添加组件相关
  const isShow = ref(false);
  const updating = ref(false);
  const curComponent = shallowRef<SpaceComponent>();
  const curRow = ref<WorkspaceComponentOutputObj>();
  /**
   * 添加组件操作
   */
  async function executeAddComponent({
    data,
    scoped,
  }: {
    data: ComponentData;
    scoped?: { envs: string[]; type: ScopedType };
  }) {
    const result = await ApiServerService.CreateWorkspaceComponent({
      workspaceID: spaceStore.currentSpace,
      compName: data.name,
      type: data.type ?? '',
      version: 'v1.0.0',
      properties: data.properties,
      scopeType: scoped!.type,
      scopeEnvNames: scoped!.envs,
    })
      .then(() => true)
      .catch(() => false)
      .finally(() => (updating.value = false));
    return result;
  }
  /**
   * 编辑组件操作
   * @param param
   */
  async function executeEditComponent({
    data,
    scoped,
  }: {
    data: ComponentData;
    scoped?: { envs: string[]; type: ScopedType };
  }) {
    const result = await ApiServerService.PatchWorkspaceComponent({
      workspaceID: spaceStore.currentSpace,
      compName: curComponent.value!.name ?? '',
      properties: data.properties,
      scopeType: scoped!.type,
      scopeEnvNames: scoped!.envs,
      name: data.name,
    })
      .then(() => true)
      .catch(() => false)
      .finally(() => (updating.value = false));
    return result;
  }

  function handleAddComponent() {
    isShow.value = true;
  }

  // 批量删除
  // function handleBatchDelete() {
  //   if (!selections.value.length) return;

  //   InfoBox({
  //     title: t('确定删除选中的组件？'),
  //     content: t('共选中 {0} 个组件实例', [selections.value.length]),
  //     cancelText: t('取消'),
  //     confirmButtonTheme: 'danger',
  //     async onConfirm() {
  //       // TODO: 实现批量删除逻辑
  //       const promises = selections.value.map(row =>
  //         ApiServerService.DeleteWorkspaceComponent({
  //           workspaceID: spaceStore.currentSpace,
  //           compName: row.name,
  //         }),
  //       );

  //       const result = await Promise.all(promises)
  //         .then(() => true)
  //         .catch(() => false);

  //       if (!result) return;
  //       Message({
  //         message: t('操作成功'),
  //         theme: 'success',
  //       });
  //       selections.value = [];
  //       tableComponentRef.value?.getVxeTableInstance()?.clearCheckboxRow();
  //       await getComponentList();
  //     },
  //   });
  // }

  const isEdit = ref<boolean>(false);

  // 清空左侧搜索
  function handleClearFilters() {
    treeSearchValue.value = '';
  }

  /**
   * 克隆组件
   * @param row 组件数据
   */
  function handleClone(row: WorkspaceComponentOutputObj) {
    curRow.value = row;
    const cloneSuffix = '-copy';
    curComponent.value = {
      name: row.name + cloneSuffix,
      type: row.type,
      version: row.version,
      properties: row.properties,
      scoped: {
        envs: row.scopeEnvNames,
        type: row.scopeType as ScopedType,
      },
    } as SpaceComponent;
    isShow.value = true;
  }

  function handleCloseSlider() {
    isShow.value = false;
    isEdit.value = false;
    curComponent.value = undefined;
    curRow.value = undefined;
  }

  function handleDelete(row: WorkspaceComponentOutputObj) {
    if (row.refAppIDs?.length) return;
    InfoBox({
      title: t('确定删除组件？'),
      cancelText: t('取消'),
      confirmButtonTheme: 'danger',
      async onConfirm() {
        const result = await ApiServerService.DeleteWorkspaceComponent({
          workspaceID: spaceStore.currentSpace,
          compName: row.name ?? '',
        })
          .then(() => true)
          .catch(() => false);
        if (!result) return;
        Message({
          message: t('操作成功'),
          theme: 'success',
        });
        await getComponentList();
      },
    });
  }

  /**
   * 编辑组件
   * @param row
   */
  function handleEdit(row: WorkspaceComponentOutputObj) {
    isEdit.value = true;
    curRow.value = row;
    curComponent.value = {
      name: row.name,
      type: row.type,
      version: row.version,
      properties: row.properties,
      scoped: {
        envs: row.scopeEnvNames,
        type: row.scopeType as ScopedType,
      },
    } as SpaceComponent;
    isShow.value = true;
  }

  /**
   * 添加组件
   * @param param
   */
  async function handleSubmit({
    data,
    scoped,
  }: {
    data: ComponentData;
    scoped?: { envs: string[]; type: ScopedType };
  }) {
    updating.value = true;
    let result = false;
    if (isEdit.value) {
      // 编辑组件
      result = await executeEditComponent({
        data,
        scoped,
      });
    } else {
      // 添加组件
      result = await executeAddComponent({
        data,
        scoped,
      });
    }
    if (result) {
      Message({
        message: t('操作成功'),
        theme: 'success',
      });
      getComponentList();
      isShow.value = false;
    }
  }

  /**
   * 渲染配置内容
   */
  function propertiesToText(properties: object) {
    if (!properties || Object.keys(properties).length === 0) {
      return h('span', '--');
    }

    const entries = Object.entries(properties);

    return h('div', { class: 'flex flex-col gap-[2px] max-h-[300px] overflow-y-auto' }, [
      ...entries.map(([key, value]) =>
        h('span', { class: 'text-[12px] truncate' }, `${key}: ${formatPropertyValue(value)}`),
      ),
    ]);
  }

  onBeforeMount(async () => {
    await handleGetEnvList();
    await getComponentList();
  });
</script>
<style lang="postcss" scoped>
  :deep(.bk-resize-layout > .bk-resize-layout-aside .bk-resize-layout-aside-content) {
    display: flex;
  }
  :deep(.bk-resize-layout > .bk-resize-layout-main) {
    display: flex;
    min-height: 0;
    min-width: 0;
  }
  :deep(.bk-resize-layout > .bk-resize-layout-aside) {
    transition: none !important;
  }
</style>
