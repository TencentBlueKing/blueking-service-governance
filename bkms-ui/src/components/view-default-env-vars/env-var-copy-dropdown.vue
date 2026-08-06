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
  <!-- 多格式复制 -->
  <Dropdown
    placement="bottom-end"
    :popover-options="{ boundary: 'body', clickContentAutoHide: true }"
    trigger="click"
  >
    <template #default="{ popoverShow }">
      <div
        class="ml-[6px] inline-flex h-[22px] shrink-0 overflow-hidden rounded-[2px] bg-[#fff] border border-[#3A84FF] text-[#3A84FF] opacity-0 transition group-hover:opacity-100"
        :class="{ 'opacity-100': popoverShow }"
      >
        <button
          v-bk-tooltips="{
            content: primaryOption.recommended
              ? $t('复制 {0}（推荐）', [primaryOption.format(variableKey)])
              : $t('复制 {0}', [primaryOption.format(variableKey)]),
            placement: 'top',
          }"
          class="flex cursor-pointer items-center gap-[4px] border-0 bg-transparent px-[6px] text-inherit hover:bg-[#EAF3FF]"
          type="button"
          @click.stop="handleCopy(primaryOption.format(variableKey))"
          @mousedown.stop
        >
          <Copy class="text-[12px]" />
          <span>{{ $t('复制') }}</span>
        </button>
        <span class="w-px bg-[#3A84FF]"></span>
        <button
          class="flex w-[20px] cursor-pointer items-center justify-center border-0 bg-transparent text-inherit hover:bg-[#EAF3FF]"
          type="button"
        >
          <AngleDownLine
            class="text-[12px] transition-transform duration-200"
            :class="{ 'rotate-180': popoverShow }"
          />
        </button>
      </div>
    </template>
    <template #content>
      <Dropdown.DropdownMenu class="!w-[251px] !p-0">
        <Dropdown.DropdownItem
          v-for="option in options"
          :key="option.id"
          class="group !h-auto !px-[12px] !py-[10px]"
          :class="{ '!bg-[#E1ECFF]': option.id === selectedOptionId }"
          @click="handleSelect(option)"
        >
          <div class="flex min-w-0 flex-col whitespace-normal leading-[20px]">
            <div class="flex min-w-0 items-center gap-[4px]">
              <span
                class="min-w-0 truncate text-[#4D4F56] group-hover:text-[#3A84FF]"
                :class="{ '!text-[#3A84FF]': option.id === selectedOptionId }"
              >
                {{ option.format(variableKey) }}
              </span>
              <Tag
                v-if="option.recommended"
                class="shrink-0"
                size="small"
                theme="success"
              >
                {{ $t('推荐') }}
              </Tag>
            </div>
            <div class="text-[#979BA5]">{{ option.description }}</div>
          </div>
        </Dropdown.DropdownItem>
      </Dropdown.DropdownMenu>
    </template>
  </Dropdown>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Dropdown, Tag } from 'bkui-vue';
  import { AngleDownLine, Copy } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useCopy } from '~/composables/use-copy';

  interface ICopyOption {
    description: string;
    id: string;
    recommended?: boolean;
    format: (key: string) => string;
  }

  const props = defineProps<{
    options: ICopyOption[];
    selectedOptionId?: string;
    variableKey: string;
  }>();
  const emit = defineEmits<{
    'update:selectedOptionId': [id: string];
  }>();

  const primaryOption = computed(
    () => props.options.find(option => option.id === props.selectedOptionId) ?? props.options[0]!,
  );

  const { t } = useI18n();
  const { copyText } = useCopy();

  function handleCopy(value: string) {
    copyText(value, t('已复制 {0}', [value]));
  }

  function handleSelect(option: ICopyOption) {
    emit('update:selectedOptionId', option.id);
    handleCopy(option.format(props.variableKey));
  }
</script>
