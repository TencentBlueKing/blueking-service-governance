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

<!-- 合并列的分组筛选：一个表头内按字段分组勾选，确认后逐字段抛出 filter-change -->
<template>
  <Popover
    ref="popoverRef"
    :arrow="false"
    placement="bottom-start"
    theme="light custom-filter-popover"
    trigger="click"
    @after-show="initSelected"
  >
    <span class="cursor-pointer">
      <span
        v-bk-tooltips="{ content: tooltip, disabled: !tooltip }"
        :class="{ 'border-b border-dashed border-[#979ba5]': tooltip }"
      >
        {{ label }}
      </span>
      <i
        class="hover:text-[#606266] relative text-[#c0c4cc] top-[1px] left-[4px] vxe-filter--btn vxe-table-icon-funnel"
        :class="hasApplied ? '!text-[#3a84ff]' : ''"
      ></i>
    </span>
    <template #content>
      <div>
        <div
          class="flex py-[4px]"
          @click.stop
        >
          <div
            v-for="group in visibleGroups"
            :key="group.field"
            class="group-section w-[160px]"
          >
            <div class="px-[12px] h-[28px] leading-[28px] text-[12px] text-[#979ba5]">{{ group.label }}</div>
            <Checkbox.Group
              v-model="selected[group.field]"
              class="flex flex-col w-full max-h-[288px] overflow-y-auto"
            >
              <Checkbox
                v-for="option in group.options"
                :key="option.value"
                class="!ml-0 w-full text-[12px] text-[#63656e]"
                :label="option.value"
              >
                {{ option.label }}
              </Checkbox>
            </Checkbox.Group>
          </div>
        </div>
        <div class="flex items-center justify-end h-[42px] border-t px-[8px] border-[#dcdee5]">
          <Button
            class="mr-[8px]"
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
  import { computed, ref } from 'vue';

  import { Button, Checkbox, Popover } from 'bkui-vue';

  import type { FilterItem } from '~/components/custom-filter.vue';

  export interface FilterGroup {
    field: string;
    label: string;
    options: FilterItem[];
  }

  const props = defineProps<{
    groups: FilterGroup[];
    label: string;
    tooltip?: string;
  }>();

  const emit = defineEmits<{
    (e: 'change', payload: { field: string; values: string[] }): void;
  }>();

  const popoverRef = ref<InstanceType<typeof Popover>>();

  function createEmptySelected() {
    return Object.fromEntries(props.groups.map(group => [group.field, []]));
  }

  const selected = ref<Record<string, (number | string)[]>>(createEmptySelected());

  const visibleGroups = computed(() => props.groups.filter(group => group.options.length > 0));
  const hasApplied = computed(() => props.groups.some(group => group.options.some(option => option.checked)));

  function emitAll(getValues: (group: FilterGroup) => (number | string)[]) {
    for (const group of props.groups) {
      emit('change', { field: group.field, values: getValues(group).map(String) });
    }
    popoverRef.value?.hide();
  }

  function handleConfirm() {
    emitAll(group => selected.value[group.field] ?? []);
  }

  function handleReset() {
    selected.value = createEmptySelected();
    emitAll(() => []);
  }

  function initSelected() {
    selected.value = Object.fromEntries(
      props.groups.map(group => [
        group.field,
        group.options.filter(option => option.checked).map(option => option.value),
      ]),
    );
  }
</script>

<style lang="postcss" scoped>
  .group-section + .group-section {
    border-left: 1px solid #f0f1f5;
  }

  :deep(.bk-checkbox) {
    height: 32px;
    padding: 0 12px;

    &:hover {
      background-color: #f5f7fa;
    }
  }

  :deep(.bk-checkbox.is-checked .bk-checkbox-label) {
    color: #3a84ff;
  }
</style>
