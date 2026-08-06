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
  <div class="flex flex-col h-full min-w-0 w-full">
    <!-- 头部 -->
    <div
      class="flex-shrink-0 h-[40px] pl-[16px] pr-[8px] rounded-t-sm border-b-[1px] border-[#DCDEE5] bg-[#F5F7FA] flex items-center justify-between"
    >
      <span class="text-[12px] text-[#313238]">
        {{ $t('可引用变量') }}
        <span class="text-[#979BA5]">({{ variableList.length }})</span>
      </span>
      <Button
        class="p-[4px]"
        text
        @click="emit('close')"
      >
        <CloseLine
          fill="#979BA5"
          height="12"
          width="12"
        />
      </Button>
    </div>
    <div class="p-[12px_16px_0]">
      <Radio.Group
        v-model="activeTab"
        class="flex flex-wrap w-full"
        type="capsule"
      >
        <Radio.Button
          v-for="tab in tabs"
          :key="tab.value"
          class="flex-1"
          :label="tab.value"
        >
          <span class="text-[12px]">{{ tab.label }}</span>
          <span
            :class="[
              'h-[16px] leading-[16px] ml-[4px] px-[6px] rounded-[8px]',
              activeTab === tab.value ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-[#fff]',
            ]"
          >
            {{ tab.count }}
          </span>
        </Radio.Button>
      </Radio.Group>
    </div>
    <!-- 搜索 -->
    <div class="px-[16px]">
      <Alert
        class="my-[16px]"
        closable
        theme="info"
        :title="$t('在组件配置中，可通过 {0} 引用环境变量。', ['{{ .key }}'])"
      />

      <Input
        v-model="searchValue"
        clearable
        :placeholder="$t('搜索变量名称')"
        size="small"
      />
    </div>
    <!-- 变量列表 -->
    <div class="p-[12px_16px] overflow-auto flex-1">
      <Table
        :data="filteredVariables"
        :row-config="{ isHover: true }"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="() => (searchValue = '')"
          />
        </template>
        <TableColumn
          field="name"
          label="Key"
          min-width="150"
        >
          <template #default="{ row }">
            <div class="text-[12px] flex items-center gap-[8px]">
              <HoverCopy
                :copy-value="defaultCopyFormat(row.name)"
                :text="row.name"
                :tooltip="row.description || undefined"
              >
                <Tag
                  v-if="row.source === 'system'"
                  size="small"
                >
                  {{ t('系统变量') }}
                </Tag>
              </HoverCopy>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, inject, Ref, ref, shallowRef } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Input, Radio, Tag } from 'bkui-vue';
  import { CloseLine } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import HoverCopy from '~/components/hover-copy.vue';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import useTableEmpty from '~/composables/use-table-empty';

  import { BUILTIN_VARS_SYMBOL } from './params-table/constants';

  import type { BuiltinVarOutputObj } from '~/@types/v1/component-defs';

  interface IProps {
    /** 输入变量列表，来自组件输入模板的参数 */
    inputVariableNames?: { description: string; name: string }[];
  }

  interface IVariable {
    description?: string;
    name: string;
    source: 'input' | 'system';
    value: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    inputVariableNames: () => [],
  });

  const { t } = useI18n();
  const emit = defineEmits<{
    (e: 'close'): void;
  }>();

  const builtinVars = inject<Ref<BuiltinVarOutputObj[]>>(BUILTIN_VARS_SYMBOL);

  const activeTab = ref<string>('all');

  const tabs = computed(() => [
    { label: t('全部'), value: 'all', count: variableList.value.length },
    { label: t('输入变量'), value: 'input', count: props.inputVariableNames.length },
    { label: t('系统变量'), value: 'system', count: builtinVars?.value?.length || 0 },
  ]);

  const variableList = computed<IVariable[]>(() => {
    const inputVars: IVariable[] = props.inputVariableNames.map(item => ({
      name: item.name,
      value: '',
      source: 'input' as const,
      description: item.description ?? '',
    }));

    const systemVars = (builtinVars?.value || []).map(v => ({
      name: v.key || '',
      value: '',
      source: 'system' as const,
      description: v.description || '',
    }));

    return [...inputVars, ...systemVars];
  });

  // 按 tab 过滤
  const tabFilteredList = computed(() => {
    if (activeTab.value === 'all') return variableList.value;
    return variableList.value.filter(v => v.source === activeTab.value);
  });

  // 搜索
  const searchKeys = shallowRef<IInputKey[]>([{ field: 'name', id: 'name', fuzzy: true }]);
  const { searchValue, tableDataMatchSearch } = useTableSearchInput(tabFilteredList, searchKeys);
  const { curExceptionType } = useTableEmpty({ filters: searchValue });

  const filteredVariables = tableDataMatchSearch;

  function defaultCopyFormat(key: string) {
    return `{{ .${key} }}`;
  }
</script>
