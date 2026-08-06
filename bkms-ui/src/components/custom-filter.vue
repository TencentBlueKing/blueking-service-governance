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
  <Popover
    ref="popoverRef"
    :arrow="false"
    placement="bottom"
    theme="light custom-filter-popover"
    trigger="click"
  >
    <slot>
      <span>
        <span class="bkms-custom-filter-title">{{ label }}</span>
        <i
          class="cursor-pointer hover:text-[#606266] relative text-[#c0c4cc] top-[1px] left-[4px] vxe-filter--btn vxe-table-icon-funnel"
          :class="appliedFilters.length > 0 ? '!text-[#3a84ff]' : ''"
        ></i>
      </span>
    </slot>
    <template #content>
      <div class="flex flex-col h-full w-[150px]">
        <div class="p-[8px] pb-0">
          <Input
            v-model.trim="keyword"
            behavior="simplicity"
            :placeholder="$t('请输入关键字')"
          >
            <template #prefix>
              <Search class="custom-filter-search-icon text-[16px] mt-[2px]" />
            </template>
          </Input>
        </div>
        <div
          class="flex-1 overflow-hidden"
          :style="{ maxHeight: `${maxHeight}px` }"
          @click.stop
        >
          <Checkbox.Group
            v-model="activeFilters"
            class="flex flex-col w-full h-full"
          >
            <RecycleScroller
              :buffer="10"
              class="w-full"
              :item-size="36"
              :items="filteredData"
              key-field="value"
              :style="{ maxHeight: `${maxHeight}px` }"
            >
              <template #default="{ item }">
                <Checkbox
                  :key="getFilterItem(item).value"
                  :checked="getFilterItem(item).checked"
                  class="!ml-0 w-full text-[12px] text-[#63656e]"
                  :indeterminate="getFilterItem(item).value === 'all' ? isIndeterminate : false"
                  :label="getFilterItem(item).value"
                >
                  <span :title="String(getFilterItem(item).title ?? getFilterItem(item).label)">
                    {{ getFilterItem(item).label }}
                  </span>
                </Checkbox>
              </template>
            </RecycleScroller>
          </Checkbox.Group>
        </div>
        <div class="flex items-center justify-end h-[42px] border-t px-[8px] border-[#dcdee5]">
          <Button
            class="mr-[8px]"
            :disabled="activeFilters.length === 0"
            size="small"
            theme="primary"
            @click="handleConfirm"
          >
            {{ $t('确定') }}
          </Button>
          <Button
            size="small"
            @click="handleReset"
          >
            {{ $t('重置') }}
          </Button>
        </div>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  // 自定义筛选组件 - 支持虚拟滚动的多选筛选器
  // 使用示例：
  // <CustomFilter
  //   field="status"
  //   label="状态"
  //   :filters="[{ label: '启用', value: 1, checked: true }]" 与Blueking/Table的filters参数一致
  //   :table-ref="tableRef" 需传入TableRef触发filter-change事件
  //   :show-all="true"
  // />
  // [注意事项]
  // 1. TableColumn需传入field参数，filters参数，否则无法触发filter-change事件 [可参考北极星列表的使用]
  // 2. 该组件的本质是接管了原Table的filter功能，并在main.css对原filter icon做了display: none处理(基于.bkms-custom-filter-title查找的元素)
  import { computed, ref, watch } from 'vue';

  import { Button, Checkbox, Input, Popover } from 'bkui-vue';
  import { Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { RecycleScroller } from 'vue-virtual-scroller';

  import 'vue-virtual-scroller/dist/vue-virtual-scroller.css';

  export interface FilterItem {
    checked?: boolean;
    label: number | string;
    title?: string;
    value: number | string;
  }

  interface IProps {
    field: string;
    filters: FilterItem[];
    label: string;
    maxHeight?: number;
    showAll?: boolean;
    tableRef: TableRef;
  }

  interface TableRef {
    getVxeTableInstance?: () => {
      setFilter?: (field: string, filters: FilterItem[], checked?: boolean) => void;
    };
  }
  const props = withDefaults(defineProps<IProps>(), {
    maxHeight: 300,
    showAll: true,
  });
  const { t } = useI18n();
  // 当前选中项（未应用）
  const activeFilters = ref<(number | string)[]>([]);
  // 已应用的筛选项（用于图标高亮）
  const appliedFilters = ref<(number | string)[]>([]);
  // 搜索关键词
  const keyword = ref('');
  const popoverRef = ref<InstanceType<typeof Popover>>();

  function getFilterItem(item: unknown): FilterItem {
    return item as FilterItem;
  }

  // 半选状态：选中了部分项（非全部非空）
  const isIndeterminate = computed(() => {
    if (!props.showAll) return false;
    const allItems = props.filters || [];
    const selectedCount = allItems.filter(item => activeFilters.value.includes(item.value)).length;
    return selectedCount > 0 && selectedCount < allItems.length;
  });

  // 过滤数据：支持关键词搜索，动态判断"全部"是否选中
  const filteredData = computed(() => {
    const allOption: FilterItem = { label: t('全部'), value: 'all', title: t('全选/取消'), checked: false };
    let data = props?.filters || [];
    if (data.length === 0) {
      return [];
    }

    if (keyword.value) {
      const lowerKeyword = keyword.value.toLowerCase();
      data = props.filters.filter(item => String(item.label).toLowerCase().includes(lowerKeyword));
    }
    const isAllChecked = data.every(item => item.checked);
    // 如果 showAll 为 true 且有数据，在数据前面添加"全部"选项
    return props.showAll && data.length > 0
      ? [
          {
            ...allOption,
            checked: isAllChecked,
          },
          ...data,
        ]
      : data;
  });

  // 确认筛选：应用选中项到表格
  function handleConfirm() {
    const hasAll = activeFilters.value.includes('all');

    const updatedFilters = props.filters.map(item => ({
      ...item,
      checked: hasAll || activeFilters.value.includes(item.value),
    }));

    syncFilterData();
    props.tableRef.getVxeTableInstance?.()?.setFilter?.(props.field, updatedFilters, true);
    popoverRef.value?.hide();
  }

  // 重置筛选：清空所有选项
  function handleReset() {
    activeFilters.value = [];
    keyword.value = '';
    const updatedFilters = props.filters.map(item => ({
      ...item,
      checked: false,
    }));
    syncFilterData();
    props.tableRef.getVxeTableInstance?.()?.setFilter?.(props.field, updatedFilters, true);
    popoverRef.value?.hide();
  }

  // 初始化：根据 filters 的 checked 属性设置选中项
  function initSelectedItems() {
    const checkedItems = props.filters?.filter(item => item.checked)?.map(item => item.value) || [];

    // 所有项都选中时，自动添加 'all'
    if (props.showAll && checkedItems.length === props.filters?.length && props.filters?.length > 0) {
      activeFilters.value = ['all', ...checkedItems];
    } else {
      activeFilters.value = checkedItems;
    }
    syncFilterData();
  }

  // 同步数据：将当前选中项同步到已应用项
  function syncFilterData() {
    appliedFilters.value = [...activeFilters.value];
  }

  // 标志位：防止内部修改 activeFilters 时触发级联 watcher
  let isInternalUpdate = false;

  // 监听选中项变化，自动处理"全选"逻辑（参考 bk-select show-select-all 逻辑）
  watch(
    activeFilters,
    (newVal, oldVal) => {
      if (!props.showAll || isInternalUpdate) {
        isInternalUpdate = false;
        return;
      }

      const hasAll = newVal.includes('all');
      const hadAll = oldVal?.includes('all');
      const allItems = [...new Set((props.filters || []).filter(item => item.value != null).map(item => item.value))];
      const allItemsSelected = allItems.length > 0 && allItems.every(item => newVal.includes(item));

      // 场景1：选中"全部" → 自动选中所有项
      if (hasAll && !hadAll) {
        isInternalUpdate = true;
        activeFilters.value = ['all', ...allItems];
      }
      // 场景2："全部"仍选中但并非所有项都选中 → 取消某个项导致，移除"全部"保留其余
      else if (hasAll && !allItemsSelected) {
        isInternalUpdate = true;
        activeFilters.value = newVal.filter(item => item !== 'all');
      }
      // 场景3：取消"全部" → 清空所有选项
      else if (!hasAll && hadAll) {
        isInternalUpdate = true;
        activeFilters.value = [];
      }
      // 场景4：手动选中所有项 → 自动勾选"全部"
      else if (!hasAll && allItemsSelected) {
        isInternalUpdate = true;
        activeFilters.value = ['all', ...allItems];
      }
    },
    { deep: true },
  );

  // 监听 filters 变化，重新初始化选中项
  // immediate: true 确保挂载时若 filters 已有 checked 项能正确初始化（如外部预设筛选条件的场景）
  watch(
    () => props.filters,
    () => {
      initSelectedItems();
    },
    { deep: true, immediate: true },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bk-input.is-simplicity) {
    &.is-focused {
      .custom-filter-search-icon {
        color: #3a84ff;
      }
    }
    &:hover:not(.is-disabled) {
      background-color: unset;
    }
    .bk-input--text {
      background-color: unset;
      &:hover,
      &:focus {
        background-color: unset;
      }
    }
    .bk-input--suffix-icon {
      background-color: unset !important;
      &:hover {
        background-color: unset !important;
      }
    }
  }
  :deep(.bk-checkbox) {
    height: 36px;
    padding: 8px 10px 8px 12px;
    display: inline-block;
    &:hover {
      background-color: #f5f7fa;
      .bk-checkbox-input {
        border-color: #3a84ff;
        box-shadow: 0 0 0 1px #699df4;
      }
    }
    align-self: start;
    .bk-checkbox-input {
      width: 14px;
      height: 14px;
      border-radius: 3px;
      border-color: #dcdee5;
      box-shadow: 0 0 0 1px #dcdee5;
    }
    .bk-checkbox-label {
      width: calc(100% - 22px);
      white-space: nowrap;
      text-overflow: ellipsis;
      overflow: hidden;
    }
  }
  :deep(.bk-checkbox.is-checked) {
    .bk-checkbox-label {
      color: #3a84ff;
    }
  }
</style>

<style>
  .custom-filter-popover {
    padding: 0 !important;
  }
</style>
