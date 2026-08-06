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
  <Input
    v-if="valueType === 'STRING'"
    v-model.trim="localValue"
    clearable
    :disabled="disabled"
    :placeholder="placeholder"
  />
  <Input
    v-else-if="valueType === 'INT'"
    v-model.trim="localValue"
    :disabled="disabled"
    :placeholder="placeholder"
    :precision="0"
    type="number"
  />
  <Input
    v-else-if="valueType === 'TEXT'"
    v-model="localValue"
    :disabled="disabled"
    :placeholder="placeholder"
    type="textarea"
  ></Input>
  <Radio.Group
    v-else-if="valueType === 'BOOL'"
    v-model="localValue"
  >
    <Radio
      :disabled="disabled"
      :label="true"
      >true</Radio
    >
    <Radio
      :disabled="disabled"
      :label="false"
      >false</Radio
    >
  </Radio.Group>
  <Select
    v-else-if="valueType === 'SELECT'"
    v-model="localValue"
    :disabled="disabled"
    display-key="label"
    id-key="value"
    multiple-mode="tag"
  >
    <template
      v-if="selectedOption?.value"
      #tag
    >
      <span class="text-[14px] text-[#313238]">{{ selectedOption?.value }}</span>
      <span class="text-[14px] text-[#979BA5]">（{{ selectedOption?.label }}）</span>
    </template>
    <Select.Option
      v-for="(item, index) in selectOptions"
      :key="`${index}-${item}`"
      :value="item.value"
    >
      <span
        class="text-[14px]"
        :class="item.value === localValue ? 'text-[#3A84FF]' : 'text-[#313238]'"
        >{{ item.value }}</span
      >
      <span class="text-[14px] text-[#979BA5]">（{{ item.label }}）</span>
    </Select.Option>
  </Select>
  <KeyValue
    v-else-if="valueType === 'MAP'"
    v-model="mapValue"
    :disabled="disabled"
  />
</template>

<script setup lang="ts">
  import { type PropType, computed, ref, watch } from 'vue';

  import { Input, Radio, Select } from 'bkui-vue';

  export type InputType = 'BOOL' | 'INT' | 'MAP' | 'SELECT' | 'STRING' | 'TEXT';

  interface SelectOption {
    label: string;
    value: string;
  }

  const props = defineProps({
    type: {
      type: String as PropType<InputType>,
      default: '',
    },
    modelValue: {
      type: [String, Number, Boolean] as PropType<boolean | null | number | string>,
      required: true,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
    selectOptions: {
      type: Array as PropType<SelectOption[]>,
      default: () => [],
    },
    placeholder: {
      type: String,
      default: '',
    },
  });

  const emit = defineEmits<{
    (e: 'update:modelValue', value: boolean | null | number | string): void;
  }>();

  const TYPE_MAP = {
    number: 'INT',
    string: 'STRING',
    boolean: 'BOOL',
  } as const;
  const localValue = ref(props.modelValue);

  const selectedOption = computed(() => props.selectOptions.find(option => option.value === localValue.value));
  const valueType = computed<InputType>(() => {
    if (props.type) return props.type;
    return TYPE_MAP[typeof props.modelValue as keyof typeof TYPE_MAP] ?? 'STRING';
  });
  // map类型
  const mapValue = ref<{ [key: string]: string }>({});

  watch(
    () => props.modelValue,
    newValue => {
      if (props.type === 'MAP') {
        if (!newValue) {
          mapValue.value = {};
        } else {
          try {
            mapValue.value = JSON.parse(newValue as string);
          } catch (e) {
            console.warn('JSON 解析失败', e);
            mapValue.value = {};
          }
        }
      } else if (props.type === 'BOOL' && newValue === '') {
        // 组件默认值可能不存在，这时newValue为字符串
        localValue.value = null;
      } else {
        localValue.value = newValue;
      }
    },
    { immediate: true },
  );

  watch(localValue, newValue => {
    emit('update:modelValue', newValue);
  });
  watch(mapValue, newValue => {
    emit('update:modelValue', JSON.stringify(newValue));
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-select-tag-wrapper) {
    gap: unset !important;
  }
</style>
