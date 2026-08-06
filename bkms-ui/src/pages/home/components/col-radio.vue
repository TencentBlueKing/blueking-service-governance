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
  <Radio.Group
    v-model="modelValue"
    class="text-[12px] flex flex-col"
    :with-validate="false"
    @change="handleChange"
  >
    <template
      v-for="item in data"
      :key="item.value"
    >
      <Radio
        class="flex items-center !ml-0 leading-[32px]"
        :label="item.value"
      >
        <span class="mr-[8px] text-[#4D4F56]">{{ item.name }}</span>
        <i
          v-if="!item.isHideIcon"
          class="bkms-icon bkms-icon-warning-circle text-[14px] text-[#979BA5] mr-[3px]"
        ></i>
        <span
          v-if="item?.tips"
          class="text-[#979BA5]"
        >
          {{ item.tips }}
        </span>
      </Radio>
      <slot
        v-if="item.value === modelValue"
        name="current"
      >
      </slot>
    </template>
  </Radio.Group>
</template>

<script lang="ts" setup>
  import { Radio } from 'bkui-vue';
  interface IProps {
    data: {
      isHideIcon?: boolean;
      name: string;
      tips?: string;
      value: string;
    }[];
  }
  const modelValue = defineModel<string>('modelValue');
  defineProps<IProps>();
  const emit = defineEmits(['change']);
  function handleChange(val: string) {
    emit('change', val);
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-radio-label) {
    font-size: 12px;
  }
</style>
