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
  <div class="flex flex-col gap-[12px]">
    <div
      v-for="item in options"
      :key="item.value"
      v-bk-tooltips="{
        content: item.disabledTip,
        disabled: !item.disabled || !item.disabledTip,
      }"
      class="flex items-center gap-[12px] p-[16px] rounded-[2px] border transition-all duration-200"
      :class="[
        item.disabled
          ? 'border-[#DCDEE5] bg-[#FAFBFD] cursor-not-allowed opacity-60'
          : modelValue === item.value
            ? 'border-[#3A84FF] bg-[#F0F5FF] cursor-pointer'
            : 'border-[#DCDEE5] bg-white hover:border-[#3A84FF] cursor-pointer',
      ]"
      @click="handleSelect(item)"
    >
      <div
        class="mt-[2px] w-[16px] h-[16px] rounded-full border-[1px] shrink-0 flex items-center justify-center transition-all duration-200"
        :class="[modelValue === item.value ? 'border-[#3A84FF]' : 'border-[#DCDEE5]']"
      >
        <div
          v-if="modelValue === item.value"
          class="w-[8px] h-[8px] rounded-full bg-[#3A84FF]"
        />
      </div>
      <div class="flex-1 min-w-0">
        <div
          class="text-[14px] font-medium leading-[22px]"
          :class="[modelValue === item.value ? 'text-[#3A84FF]' : 'text-[#313238]']"
        >
          {{ item.label }}
        </div>
        <div
          v-if="item.description"
          class="mt-[4px] text-[12px] leading-[20px]"
          :class="[modelValue === item.value ? 'text-[#699DF4]' : 'text-[#979BA5]']"
        >
          {{ item.description }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  export interface CardRadioOption {
    description?: string;
    /** 是否禁用该项；禁用后不可点击，并显示 disabledTip 提示 */
    disabled?: boolean;
    /** 禁用时的 hover 提示文案 */
    disabledTip?: string;
    label: string;
    value: string;
  }

  interface Props {
    options: CardRadioOption[];
  }

  defineProps<Props>();
  const modelValue = defineModel<string>('modelValue');
  const emit = defineEmits<{
    (e: 'change', value: string): void;
  }>();

  function handleSelect(item: CardRadioOption) {
    if (item.disabled) return;
    if (modelValue.value !== item.value) {
      modelValue.value = item.value;
      emit('change', item.value);
    }
  }
</script>
