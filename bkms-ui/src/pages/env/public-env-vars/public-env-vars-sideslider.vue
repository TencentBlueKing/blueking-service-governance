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
  <Sideslider
    v-model:is-show="visible"
    quick-close
    render-directive="if"
    :title="$t('公共环境变量')"
    :width="1200"
    @hidden="handleHidden"
  >
    <div class="flex flex-col px-6 pt-[18px] pb-6 h-full gap-[16px]">
      <!-- 操作栏：新增按钮 + Tab 筛选 + 搜索 -->
      <div class="flex items-center justify-between flex-wrap">
        <div class="flex items-center gap-[12px]">
          <Button
            theme="primary"
            @click="handleCreate"
          >
            <Plus
              :height="24"
              :width="24"
            />
            {{ $t('新增环境变量') }}
          </Button>
          <Radio.Group
            v-model="activeTab"
            type="capsule"
          >
            <Radio.Button
              v-for="tab in tabs"
              :key="tab.key"
              :label="tab.key"
            >
              <div class="flex items-center justify-center">
                {{ tab.label }}
                <span
                  :class="[
                    'h-[16px] leading-[16px] ml-[4px] px-[6px] rounded-[8px]',
                    tab.key === activeTab ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-[#fff]',
                  ]"
                  >{{ tab.count }}</span
                >
              </div>
            </Radio.Button>
          </Radio.Group>
        </div>
        <SearchSelect
          v-model="searchValue"
          class="w-[360px]"
          :data="searchSelectData"
          :placeholder="
            createPlaceholder({
              type: 'searchSelect',
              labels: ['Key', 'Value', '描述', '生效环境类型'],
            })
          "
          unique-select
          value-behavior="need-key"
        />
      </div>

      <!-- 提示信息 -->
      <Alert theme="info"> {{ $t('可添加公共环境变量，对全部环境或某类环境生效。') }} </Alert>

      <Table
        class="public-env-var-table"
        :data="filteredList"
        :sort-config="{ remote: false, trigger: 'cell' }"
        @filter-change="handleFilterChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="handleClearFilters"
            @refresh="fetchList"
          />
        </template>

        <TableColumn
          field="key"
          label="Key"
          :min-width="220"
          show-overflow-tooltip
          sortable
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.key"
              :text="row.key"
            />
          </template>
        </TableColumn>

        <TableColumn
          field="isSensitive"
          :label="$t('是否敏感')"
          :width="110"
        >
          <template #header>
            <span
              v-bk-tooltips="$t('敏感环境变量的值将在页面上以脱敏形式展示，只有应用进程内能够获取到这些变量的明文值。')"
              class="border-b border-dashed border-[#979ba5]"
            >
              {{ $t('是否敏感') }}
            </span>
          </template>
          <template #default="{ row }">
            <Tag
              :class="row.isSensitive ? 'text-[#2CAF5E] bg-[#DAF6E5]' : 'text-[#979BA5] bg-[#F0F1F5]'"
              :theme="row.isSensitive ? 'success' : 'default'"
            >
              {{ row.isSensitive ? $t('是') : $t('否') }}
            </Tag>
          </template>
        </TableColumn>

        <TableColumn
          field="value"
          label="Value"
          :min-width="260"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <SensitiveValuePlaceholder v-if="row.isSensitive" />
            <HoverCopy
              v-else
              :class="{ 'text-[#DCDEE5]': !row.value }"
              :copy-value="row.value"
              :text="row.value"
            />
          </template>
        </TableColumn>

        <!-- 生效环境类型 -->
        <TableColumn
          field="scopeType"
          filter-multiple
          :filters="filterOptions.scopeType"
          :label="$t('生效环境类型')"
          :width="160"
        >
          <template #default="{ row }">
            <Tag :class="getScopeDisplay(row.scopeType, row.scopeValue).tagClass">
              {{ getScopeDisplay(row.scopeType, row.scopeValue).label }}
            </Tag>
          </template>
        </TableColumn>

        <!-- 描述 -->
        <TableColumn
          field="description"
          :label="$t('描述')"
          :min-width="180"
          show-overflow-tooltip
        >
          <template #default="{ row }">{{ row.description || '--' }}</template>
        </TableColumn>

        <!-- 操作 -->
        <TableColumn
          field="operation"
          fixed="right"
          :label="$t('操作')"
          :width="140"
        >
          <template #default="{ row }">
            <template v-if="row.isBuiltin">
              <span class="text-[12px] text-[#979ba5]">{{ $t('内置环境变量，不能操作') }}</span>
            </template>
            <template v-else>
              <Button
                text
                theme="primary"
                @click="handleEdit(row)"
                >{{ $t('编辑') }}</Button
              >
              <Button
                class="ml-[16px]"
                text
                theme="primary"
                @click="handleDelete(row)"
                >{{ $t('删除') }}</Button
              >
            </template>
          </template>
        </TableColumn>
      </Table>
    </div>

    <!-- 新建/编辑环境变量弹窗 -->
    <EnvVarFormDialog
      v-model:is-show="isShowFormDialog"
      :edit-data="editingVar"
      :workspace-id="space"
      @success="fetchList"
    />

    <!-- 删除环境变量弹窗 -->
    <DeleteEnvVarDialog
      v-model:is-show="isShowDeleteDialog"
      :env-var-data="deletingVar"
      :workspace-id="space"
      @deleted="fetchList"
    />
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Radio, SearchSelect, Sideslider, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ScopedEnvVarOutputObj } from '~/@types/v1/envvars';
  import { EnvvarsService } from '~/api/modules/v1';
  import { sortByDate } from '~/common/util';
  import SensitiveValuePlaceholder from '~/components/editable-variable-table/sensitive-value-placeholder.vue';
  import HoverCopy from '~/components/hover-copy.vue';
  import TableException from '~/components/table-exception.vue';
  import { getScopeDisplay } from '~/composables/use-scope-display';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';

  import DeleteEnvVarDialog from './delete-env-var-dialog.vue';
  import EnvVarFormDialog from './env-var-form-dialog.vue';

  import type { ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';

  interface Emits {
    (e: 'update:visible', value: boolean): void;
  }

  interface Props {
    space: string;
    visible: boolean;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();

  const visible = computed({
    get: () => props.visible,
    set: (val: boolean) => emit('update:visible', val),
  });

  const list = ref<ScopedEnvVarOutputObj[]>([]);
  const searchValue = ref<ISearchValue[]>([]);
  const activeTab = ref('all');

  // 弹窗控制
  const isShowFormDialog = ref(false);
  const isShowDeleteDialog = ref(false);
  const editingVar = ref<null | ScopedEnvVarOutputObj>(null);
  const deletingVar = ref<null | ScopedEnvVarOutputObj>(null);

  /** 获取某条数据对应的 Tab key（用于计数和筛选） */
  function getTabKey(item: ScopedEnvVarOutputObj): string {
    return item.scopeType === 'workspace' ? 'workspace' : (item.scopeValue as string);
  }

  /** SearchSelect 搜索条件配置 */
  const searchSelectData = ref<ISearchItem[]>([
    {
      name: 'Key',
      id: 'key',
      multiple: true,
      children: [], // 动态填充
    },
    {
      name: 'Value',
      id: 'value',
      multiple: true,
      children: [], // 动态填充
    },
    {
      name: t('描述'),
      id: 'description',
    },
    {
      name: t('生效环境类型'),
      id: 'scopeType',
      multiple: true,
      children: [
        { name: t('所有'), id: 'workspace' },
        { name: t('开发'), id: 'development' },
        { name: t('测试'), id: 'test' },
        { name: t('预发布'), id: 'staging' },
        { name: t('生产'), id: 'production' },
      ],
    },
  ]);

  const { filterOptions, handleFilterChange } = useSearchFilter(searchSelectData, searchValue, ['scopeType'] as const);

  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  /** Tab 标签页配置（带计数） */
  const tabs = computed(() => {
    const counts: Record<string, number> = { all: 0, workspace: 0, development: 0, test: 0, staging: 0, production: 0 };
    for (const item of list.value) {
      counts.all++;
      const key = getTabKey(item);
      if (key in counts) counts[key]++;
    }
    return [
      { key: 'all', label: t('全部'), count: counts.all },
      { key: 'workspace', label: t('所有'), count: counts.workspace },
      { key: 'development', label: t('开发'), count: counts.development },
      { key: 'test', label: t('测试'), count: counts.test },
      { key: 'staging', label: t('预发布'), count: counts.staging },
      { key: 'production', label: t('生产'), count: counts.production },
    ];
  });

  /** 搜索 + Tab 筛选后的列表 */
  const filteredList = computed(() => {
    let result = list.value;

    // Tab 筛选（优先级高于 SearchSelect）
    if (activeTab.value !== 'all') {
      result = result.filter(item => getTabKey(item) === activeTab.value);
    }

    if (!searchValue.value.length) return result;

    // 遍历所有搜索条件
    for (const filter of searchValue.value) {
      if (!filter.values?.length) continue;
      const selectedValues = filter.values.map(v => v.id);

      switch (filter.id) {
        case 'key':
          // Key：模糊匹配
          result = result.filter(
            item =>
              (selectedValues.includes('__empty__') && !item.key) ||
              selectedValues.some(val => item.key?.toLowerCase().includes(val.toLowerCase())),
          );
          break;
        case 'value':
          // Value：模糊匹配
          result = result.filter(
            item =>
              (selectedValues.includes('__empty__') && !item.value) ||
              selectedValues.some(val => item.value?.toLowerCase().includes(val.toLowerCase())),
          );
          break;
        case 'description':
          // 描述：模糊匹配
          result = result.filter(item =>
            selectedValues.some(val => (item.description || '').toLowerCase().includes(val.toLowerCase())),
          );
          break;
        case 'scopeType':
          // 生效环境类型：精确匹配
          result = result.filter(item => selectedValues.includes(getTabKey(item)));
          break;
      }
    }

    return result;
  });

  /** 获取列表 */
  async function fetchList() {
    if (!props.space) return;
    list.value = await EnvvarsService.listPublicScopedEnvVars({ workspaceID: props.space }, { validateCode: false })
      .then(data => {
        clearErrorType();
        sortByDate(data, item => item.createdAt);
        // 动态填充筛选项 children
        handleInitFilterOptions(data);
        return data;
      })
      .catch(() => {
        setTypeToError();
        return [];
      });
  }

  /** 空值筛选标识 */
  const EMPTY_FILTER_ID = '__empty__';

  /** 清除筛选 */
  function handleClearFilters() {
    // 清除 SearchSelect 选中值
    searchValue.value = [];
    // 重置 Tab 为全部
    activeTab.value = 'all';
  }

  /** 新增 */
  function handleCreate() {
    editingVar.value = null;
    isShowFormDialog.value = true;
  }

  /** 删除 */
  function handleDelete(row: ScopedEnvVarOutputObj) {
    deletingVar.value = row;
    isShowDeleteDialog.value = true;
  }

  /** 编辑 */
  function handleEdit(row: ScopedEnvVarOutputObj) {
    editingVar.value = row;
    isShowFormDialog.value = true;
  }

  /** 面板关闭时重置状态 */
  function handleHidden() {
    list.value = [];
    searchValue.value = [];
    activeTab.value = 'all';
    editingVar.value = null;
    deletingVar.value = null;
  }

  /** 从列表数据中提取去重值，填充筛选项 children */
  function handleInitFilterOptions(data: ScopedEnvVarOutputObj[]) {
    const keySet = new Map<string, string>();
    const valueSet = new Map<string, string>();

    let hasEmptyKey = false;
    let hasEmptyValue = false;

    for (const item of data) {
      if (item.key) {
        keySet.set(item.key, item.key);
      } else {
        hasEmptyKey = true;
      }
      if (item.value) {
        valueSet.set(item.value, item.value);
      } else {
        hasEmptyValue = true;
      }
    }

    const emptyOption = { id: EMPTY_FILTER_ID, name: t('未设置') };
    const toChildren = (map: Map<string, string>, hasEmpty: boolean) => {
      const children = Array.from(map, ([id, name]) => ({ id, name }));
      if (hasEmpty) children.push(emptyOption);
      return children;
    };

    const findItem = (id: string) => searchSelectData.value.find(item => item.id === id);
    const keyItem = findItem('key');
    const valueItem = findItem('value');

    if (keyItem) keyItem.children = toChildren(keySet, hasEmptyKey);
    if (valueItem) valueItem.children = toChildren(valueSet, hasEmptyValue);
  }

  watch(
    () => props.visible,
    val => {
      if (val && props.space) fetchList();
    },
  );
</script>

<style lang="postcss" scoped></style>

<style lang="postcss">
  .public-env-var-table .vxe-table--filter-wrapper .vxe-table--filter-body {
    min-height: 100px;
  }
</style>
