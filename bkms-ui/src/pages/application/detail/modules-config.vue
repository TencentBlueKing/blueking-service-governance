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
    :loading="loading"
    theme="gray"
  >
    <template #loading>
      <div class="h-full px-[24px] py-[20px]">
        <Layout.shape width="100%" />
        <FlexRow
          average
          class="my-[16px]"
        >
          <template #left>
            <Layout.shape :width="110" />
          </template>
        </FlexRow>
        <Layout.table />
      </div>
    </template>
    <div class="h-full px-[24px] py-[20px]">
      <Alert
        class="mb-[12px]"
        theme="info"
        >{{ $t('通过组件可以灵活修改应用的 Kubernetes 资源配置，如调整参数、添加网络策略等。') }}</Alert
      >
      <FlexRow>
        <template #left>
          <Button
            theme="primary"
            @click="handleShowSlider"
          >
            <Plus
              class="mr-[4px]"
              :height="24"
              :width="24"
            />
            {{ $t('添加组件实例') }}
          </Button>
        </template>
        <template #right>
          <SearchSelect
            v-model="searchValue"
            class="w-[520px] bg-[#fff] relative z-[100]"
            :data="searchSelectData"
            :placeholder="
              createPlaceholder({
                type: 'searchSelect',
                labels: ['实例名称', '组件', '可用环境', '来源'],
              })
            "
            unique-select
            value-behavior="need-key"
          >
          </SearchSelect>
        </template>
      </FlexRow>
      <div
        ref="componentTableContentRef"
        class="!h-[calc(100%-48px-33px-16px)] flex flex-col gap-[16px] mt-[16px]"
      >
        <Table
          ref="componentTableRef"
          :data="tableDataMatchSearch"
          :max-height="componentTableContentHeight"
          :pagination="pagination"
          :row-config="{
            isHover: true,
            isCurrent: true,
          }"
          :show-overflow="false"
          @checkbox-all="handleCheckboxAll"
          @checkbox-change="handleCheckboxChange"
          @filter-change="filterChangeEvent"
          @page-limit-change="pageSizeChange"
          @page-value-change="pageChange"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="handleClearFilters"
              @refresh="handleFetchList"
            >
            </TableException>
          </template>

          <!-- <TableColumn
            fixed="left"
            type="checkbox"
            width="50"
          /> -->

          <TableColumn
            field="name"
            fixed="left"
            label="实例名称"
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
                :table-ref="componentTableRef"
              />
            </template>
            <template #default="{ row }">
              {{ row.type || '--' }}
            </template>
          </TableColumn>

          <TableColumn
            field="properties"
            label="配置内容"
            min-width="250"
          >
            <template #default="{ row }">
              <div
                v-if="Object.keys(row.properties || {})?.length"
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
                  v-if="Object.keys(row.properties || {})?.length > 3"
                  class="text-[12px] text-[#979BA5]"
                >
                  ...
                </span>
              </div>
              <span v-else>--</span>
            </template>
          </TableColumn>

          <TableColumn
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
                :table-ref="componentTableRef"
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
                  v-for="(env, index) in row.scopeEnvNames?.slice(0, 2)"
                  :key="index"
                  class="pointer-events-none"
                >
                  {{ env }}
                </Tag>
                <Popover theme="light">
                  <Tag v-if="row.scopeEnvNames?.length > 2"> +{{ row.scopeEnvNames?.length - 2 }} </Tag>
                  <template #content>
                    <Tag
                      v-for="(env, index) in row.scopeEnvNames?.slice(2)"
                      :key="index"
                      class="mr-[4px]"
                    >
                      {{ env }}
                    </Tag>
                  </template>
                </Popover>
              </div>
              <span v-else>--</span>
            </template>
          </TableColumn>

          <TableColumn
            field="source"
            filter-multiple
            :filters="filterOptions.source"
            :label="$t('来源')"
            min-width="150"
          >
            <template #default="{ row }">
              {{ row.refWorkspaceCompName ? $t('引用空间组件实例') : $t('自定义') }}
            </template>
          </TableColumn>

          <!-- 操作列 -->
          <TableColumn
            fixed="right"
            :label="$t('操作')"
            min-width="120"
          >
            <template #default="{ row }">
              <div class="flex items-center gap-[12px]">
                <Button
                  v-bk-tooltips="{
                    content: $t('引用创建的实例不支持编辑'),
                    disabled: !row.refWorkspaceCompName,
                  }"
                  :disabled="!!row.refWorkspaceCompName"
                  text
                  theme="primary"
                  @click="handleEdit(row)"
                >
                  {{ $t('编辑') }}
                </Button>
                <Button
                  v-bk-tooltips="{
                    content: $t('引用创建的实例不支持克隆'),
                    disabled: !row.refWorkspaceCompName,
                  }"
                  :disabled="!!row.refWorkspaceCompName"
                  text
                  theme="primary"
                  @click="handleClone(row)"
                >
                  {{ $t('克隆') }}
                </Button>
                <Button
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
  </Skeleton>
  <ComponentsSideSlider
    v-model="isShow"
    :btn-loading="updating"
    :data="curComponent"
    module-type="app"
    :title="isClone ? t('添加组件实例') : ''"
    @close="handleCloseSlider"
    @submit="handleSubmitComponent"
  />
</template>
<script setup lang="ts">
  import { computed, h, onBeforeMount, ref, shallowRef } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, InfoBox, Message, Popover, SearchSelect, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ComponentOutputObj } from '~/@types/v1/app';
  import { AppComponentsService } from '~/api/modules/v1';
  import { ComponentData } from '~/components/modules/components-sideSlider.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePageConf from '~/composables/use-page';
  import { ISelectKey, ITableFilterItem, useTableSearchSelect } from '~/composables/use-search';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableCheckbox from '~/composables/use-table-checkbox';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useAppDetail } from '~/stores/app-detail';

  import type { ComponentType } from '~/@types/api';

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { createPlaceholder } = useSearchPlaceholder();
  // 表格数据
  const componentList = computed<ComponentOutputObj[]>(() => {
    return (appDetail.value?.appModelSpec?.components || []).map(item => ({
      ...item,
      source: item.refWorkspaceCompName ? 'reference' : 'custom',
    })) as unknown as ComponentOutputObj[];
  });
  const curComponent = shallowRef<ComponentOutputObj>();

  // 复选框相关
  const { selections, handleCheckboxAll, handleCheckboxChange } = useTableCheckbox(componentList, 'name');

  const isShow = ref<boolean>(false);

  const loading = computed(() => appDetailStore.loading);
  const appDetail = computed(() => appDetailStore.appDetail);

  const isClone = ref<boolean>(false);
  const editIndex = ref<number>(-1); // 当前更新索引
  const componentType = ref<ComponentType | undefined>('Deploy');

  const componentTableContentRef = ref<HTMLElement>();
  const componentTableRef = ref();
  const componentTypeOptions = computed<ITableFilterItem[]>(() => {
    const typeSet = new Set<string>();
    for (const component of componentList.value) {
      if (component.type) typeSet.add(component.type);
    }
    return Array.from(typeSet).map(item => ({ name: item, id: item })) as ITableFilterItem[];
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

    return Array.from(envSet).map(item => ({ name: item, id: item })) as ITableFilterItem[];
  });
  const searchSelectData = computed<ISelectKey<ComponentOutputObj>[]>(() => [
    {
      name: t('实例名称'),
      id: t('实例名称'),
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
    },
    {
      name: t('来源'),
      id: 'source',
      multiple: false,
      placeholder: t('来源'),
      field: 'source',
      children: [
        { name: t('引用空间组件实例'), id: 'reference' },
        { name: t('自定义'), id: 'custom' },
      ],
    },
  ]);
  const { filterChangeEvent, searchValue, tableDataMatchSearch, filterOptions } = useTableSearchSelect(
    componentList,
    searchSelectData,
    {
      tableRef: componentTableRef,
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

  // 使用 useElementHeight hook 获取表格容器高度
  const { height: componentTableContentHeight } = useElementHeight(componentTableContentRef, {
    watchSource: loading,
    defaultHeight: 600,
  });

  /**
   * 关闭 Slider
   */
  function handleCloseSlider() {
    isClone.value = false;
    curComponent.value = undefined;
  }

  function handleShowSlider() {
    componentType.value = undefined;
    editIndex.value = -1; // 重置索引
    isShow.value = true;
  }

  const updating = ref<boolean>(false);
  function formatPropertyValue(value: object | string): string {
    if (typeof value === 'object') {
      return JSON.stringify(value);
    }
    return String(value);
  }

  // 清除筛选并刷新
  function handleClearFilters() {
    searchValue.value = [];
    handleFetchList();
  }

  /**
   * 克隆组件
   * @param row 组件数据
   */
  function handleClone(row: ComponentOutputObj) {
    if (row.refWorkspaceCompName) return;
    const cloneSuffix = '-copy';
    curComponent.value = {
      name: row.name + cloneSuffix,
      type: row.type,
      version: row.version,
      properties: row.properties,
      refWorkspaceCompName: row.refWorkspaceCompName,
    } as ComponentOutputObj;
    isClone.value = true; // 克隆
    isShow.value = true;
  }
  function handleDelete(row: ComponentOutputObj) {
    selections.value = [row];
    InfoBox({
      title:
        selections.value.length === 1
          ? t('确定删除该实例？')
          : t('确定批量删除 {0} 个实例？', [selections.value.length]),
      content: renderInfoBoxContent(),
      headerAlign: selections.value.length === 1 ? 'left' : 'center',
      contentAlign: 'left',
      cancelText: t('取消'),
      confirmButtonTheme: 'danger',
      async onConfirm() {
        const result = await AppComponentsService.deleteAppComponent({
          appID: appDetailStore.appID,
          compName: row?.name || '',
        })
          .then(() => true)
          .catch(() => false);
        if (!result) return;
        Message({
          message: t('操作成功'),
          theme: 'success',
        });
        handleFetchList();
      },
    });
  }

  /**
   * 编辑组件
   * @param row
   */
  function handleEdit(row: ComponentOutputObj) {
    curComponent.value = {
      name: row.name,
      type: row.type,
      version: row.version,
      properties: row.properties,
      refWorkspaceCompName: row.refWorkspaceCompName,
    } as ComponentOutputObj;
    isShow.value = true;
  }
  // 获取数据
  async function handleFetchList() {
    try {
      await appDetailStore.fetchAppDetail();
      clearErrorType();
    } catch (err) {
      console.error(err);
      setTypeToError();
    }
  }
  async function handleSubmitComponent(res: { data: ComponentData; isUseInstance: boolean }) {
    if (!appDetail.value?.name) return;
    const { data, isUseInstance } = res;

    updating.value = true;
    let result = false;
    if (curComponent.value && !isClone.value) {
      result = await AppComponentsService.patchAppComponent({
        appID: appDetail.value?.id || '',
        compName: curComponent.value?.name || '',
        properties: data.properties,
        name: data.name,
      })
        .then(() => true)
        .catch(() => false);
    } else {
      result = await AppComponentsService.createAppComponent({
        appID: appDetail.value?.id || '',
        type: data.type,
        version: data.version,
        properties: data.properties,
        refWorkspaceCompName: isUseInstance ? data.name : '',
        compName: data.name,
      })
        .then(() => true)
        .catch(() => false);
    }
    updating.value = false;

    // 关闭弹窗
    if (result) {
      isShow.value = false;
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });
      handleFetchList();
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

  function renderInfoBoxContent() {
    if (selections.value.length === 1) {
      return h('div', [
        h('div', [t('实例名称: {0}', [`${selections.value[0]?.name}`])]),
        h('div', { class: 'mt-[10px]' }, [t('删除后，将不可恢复，请谨慎操作！')]),
      ]);
    }
    const instances = selections.value.map(item => h('div', { class: ['px-[16px] leading-[32px]'] }, [`${item.name}`]));
    return h('div', [
      h('div', { class: 'bg-[#F5F7FA] py-[12px] px-[16px]' }, [t('删除后，将不可恢复，请谨慎操作！')]),
      h(
        'div',
        {
          class:
            'bg-[#F5F7FA] leading-[32px] px-[16px] mt-[16px] border-1 border-[#EAEBF0] rounded-[2px] border-b-transparent',
        },
        [h('span', [t('已选择以下 {0} 个实例', [selections.value.length])])],
      ),
      h('div', { class: 'max-h-[160px] overflow-auto border-1 border-t-transparent border-[#EAEBF0] rounded-[2px]' }, [
        ...instances,
      ]),
    ]);
  }

  onBeforeMount(() => {
    handleFetchList();
  });
</script>

<style lang="less" scoped>
  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
    border-radius: 2px;
  }

  @media (max-width: 1440px) {
    .grid-container {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (min-width: 1440px) and (max-width: 1600px) {
    .grid-container {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
  }

  @media (min-width: 1600px) {
    .grid-container {
      grid-template-columns: repeat(5, minmax(0, 1fr));
    }
  }
</style>
