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
  <div
    :class="{ 'overflow-y-auto': prop.maxHeight > 0 }"
    :style="scrollStyle"
  >
    <Form
      ref="formRef"
      form-type="vertical"
      :model="formModel"
    >
      <div
        v-for="(item, index) in validValue"
        :key="`${index}`"
        class="flex items-center mb-[10px]"
      >
        <Form.FormItem
          class="!mb-0 flex-1"
          error-display-type="tooltips"
          :property="`items.${index}.${keyOfKey}`"
          :rules="getKeyRules()"
        >
          <div
            v-if="textarea"
            class="relative h-[30px]"
          >
            <Input
              v-model="item[keyOfKey]"
              autosize
              class="hover-expand-textarea"
              :disabled="disabled"
              :placeholder="keyPlaceholder"
              :resize="false"
              type="textarea"
              @blur="handleKeyBlur"
            />
          </div>
          <Input
            v-else
            v-model.trim="item[keyOfKey]"
            :disabled="disabled"
            :placeholder="keyPlaceholder"
            @blur="handleKeyBlur"
          />
        </Form.FormItem>
        <slot name="middle">
          <div
            :class="[
              'size-[30px] flex justify-center items-center text-[#979BA5] text-[12px]',
              'border-1 border-[#c4c6cc] mx-[8px] radius-[2px] shrink-0',
            ]"
          >
            =
          </div>
        </slot>
        <Form.FormItem
          class="!mb-0 flex-1"
          error-display-type="tooltips"
          :property="`items.${index}.${keyOfValue}`"
          :rules="prop.valueRules"
        >
          <div
            v-if="textarea"
            class="relative h-[30px]"
          >
            <Input
              v-model="item[keyOfValue]"
              autosize
              class="hover-expand-textarea"
              :disabled="disabled"
              :placeholder="valuePlaceholder"
              :resize="false"
              type="textarea"
            />
          </div>
          <Input
            v-else
            v-model.trim="item[keyOfValue]"
            :disabled="disabled"
            :placeholder="valuePlaceholder"
          />
        </Form.FormItem>
        <Button
          v-bk-tooltips="{
            content: $t('至少保留一个'),
            disabled: !(minRows > 0 && validValue.length <= minRows),
          }"
          :disabled="disabled || (minRows > 0 && validValue.length <= minRows)"
          text
          @click="handleDel(index)"
        >
          <Del
            :class="[
              'ml-[8px]',
              disabled || (minRows > 0 && validValue.length <= minRows)
                ? 'text-[#DCDEE5]'
                : 'text-[#979BA5] hover:text-[#4D4F56]',
            ]"
            height="14px"
            width="14px"
          ></Del>
        </Button>
      </div>
    </Form>
  </div>
  <Button
    :disabled="disabled"
    text
    theme="primary"
    @click="handleAdd"
  >
    <div class="flex items-center">
      <span class="bkms-icon bkms-icon-plus-circle-shape text-[14px]"></span>
      <span class="text-[12px] ml-[6px]">{{ $t('添加') }}</span>
    </div>
  </Button>
</template>

<script setup lang="ts">
  import type { PropType } from 'vue';
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Form, Input } from 'bkui-vue';
  import { Del } from 'bkui-vue/lib/icon';
  import { isPlainObject } from 'lodash-es';

  // 表单校验规则类型定义
  export interface FormRule {
    message?: string;
    pattern?: RegExp;
    required?: boolean;
    trigger?: string;
    validator?: (value: unknown) => boolean | Promise<boolean>;
  }

  const prop = defineProps({
    textarea: {
      type: Boolean,
      default: false,
    },
    keyPlaceholder: {
      type: String,
      default: 'key',
    },
    valuePlaceholder: {
      type: String,
      default: 'value',
    },
    keyOfKey: {
      type: String,
      default: 'key',
    },
    keyOfValue: {
      type: String,
      default: 'value',
    },
    minRows: {
      type: Number,
      default: 0, // 最少保留行数，0 表示不限制
    },
    disabled: {
      type: Boolean,
      default: false,
    },
    showEqual: {
      type: Boolean,
      default: true,
    },
    maxHeight: {
      type: Number,
      default: 0,
    },
    // key 校验规则
    keyRules: {
      type: Array as PropType<FormRule[]>,
      default: () => [],
    },
    // key 唯一性校验规则；传入后会校验所有非空 key 是否重名
    keyUniqueRule: {
      type: Object as PropType<FormRule | null>,
      default: null,
    },
    // value 校验规则
    valueRules: {
      type: Array as PropType<FormRule[]>,
      default: () => [],
    },
  });

  // 使用 defineModel 定义双向绑定
  const modelValue = defineModel<Record<string, string> | Record<string, string>[]>({
    type: [Array, Object] as PropType<Record<string, string> | Record<string, string>[]>,
    default: () => [],
  });

  const emits = defineEmits(['init:modelValue']);

  /** v-model绑定的值是否emit过首次 */
  let isInitialModelValue = false;
  const formRef = ref<InstanceType<typeof Form>>();

  // 防止死循环的标志
  let isInternalUpdate = false;

  const value = ref<Record<string, string>[]>([]);

  // 表单模型,用于 Form 组件校验
  const formModel = reactive({
    items: value,
  });

  const scrollStyle = computed(() =>
    prop.maxHeight > 0
      ? {
          maxHeight: `${prop.maxHeight}px`,
          paddingRight: '20px',
        }
      : undefined,
  );

  // 过滤掉 undefined 元素，确保模板渲染不会出错
  const validValue = computed(() => value.value.filter(item => item !== undefined && item !== null));

  // 清除校验
  function clearValidate() {
    formRef.value?.clearValidate();
  }

  /** 合并 key 基础规则和可选的唯一性规则 */
  function getKeyRules(): FormRule[] {
    if (!prop.keyUniqueRule) return prop.keyRules;

    return [
      ...prop.keyRules,
      {
        ...prop.keyUniqueRule,
        validator: (value: unknown) => {
          if (!value) return true;
          const key = String(value).trim();
          return validValue.value.filter(item => item[prop.keyOfKey]?.trim() === key).length <= 1;
        },
      },
    ];
  }

  function handleAdd() {
    value.value.push({ [prop.keyOfKey]: '', [prop.keyOfValue]: '' });
  }
  function handleDel(index: number) {
    // 如果设置了最少行数限制，且当前行数已经是最少行数，则不允许删除
    if (prop.minRows > 0 && value.value.length <= prop.minRows) {
      return;
    }
    value.value.splice(index, 1);
  }

  /** 唯一性规则依赖全部 key，任一 key 失焦后同步刷新所有 key 字段的校验状态 */
  async function handleKeyBlur() {
    if (!prop.keyUniqueRule) return;

    const keyFields = value.value.map((_, index) => `items.${index}.${prop.keyOfKey}`);
    await formRef.value?.validate(keyFields).catch(() => {});
  }

  // 暴露校验方法
  async function validate() {
    try {
      await formRef.value?.validate();
      return true;
    } catch {
      return false;
    }
  }

  // 监听 modelValue 变化，同步到内部 value
  watch(
    modelValue,
    val => {
      // 如果是内部触发的更新，跳过
      if (isInternalUpdate) {
        return;
      }

      if (Array.isArray(val)) {
        value.value = val.length > 0 ? val : [];
      } else {
        // 如果是 modelValue 是对象，转换数据结构
        value.value = Object.keys(val).map(key => ({ [prop.keyOfKey]: key, [prop.keyOfValue]: val[key] }));
      }

      // 如果设置了最少行数，且当前行数小于最少行数，自动添加空行
      if (prop.minRows > 0 && value.value.length < prop.minRows) {
        const needRows = prop.minRows - value.value.length;
        for (let i = 0; i < needRows; i++) {
          value.value.push({ [prop.keyOfKey]: '', [prop.keyOfValue]: '' });
        }
      }
    },
    { immediate: true, deep: true },
  );

  // 监听内部 value 变化，同步到 modelValue
  watch(
    value,
    val => {
      // 标记为内部更新，防止触发上面的 watch
      isInternalUpdate = true;

      // 如果modelValue是对象，需要转换为对象传回去
      if (isPlainObject(modelValue.value)) {
        const result: { [key in string]: string } = {};
        val
          .filter(v => v && v[prop.keyOfKey])
          .forEach(item => {
            result[item[prop.keyOfKey]] = item[prop.keyOfValue];
          });
        modelValue.value = result;
        if (!isInitialModelValue) {
          emits('init:modelValue', result);
          isInitialModelValue = true;
        }
      } else {
        const result = val.filter(item => item && item[prop.keyOfKey] && item[prop.keyOfValue]);
        modelValue.value = result;
        if (!isInitialModelValue) {
          emits('init:modelValue', result);
          isInitialModelValue = true;
        }
      }

      // 在下一个 tick 重置标志
      Promise.resolve().then(() => {
        isInternalUpdate = false;
      });
    },
    { deep: true, immediate: true },
  );

  defineExpose({
    validate,
    clearValidate,
  });
</script>
<style lang="postcss" scoped>
  .hover-expand-textarea {
    position: absolute;
    transition: all 0.2s ease;
    height: 30px;
    min-height: 30px;
    top: 0;
    left: 0;
    &.is-focused {
      height: auto;
      max-height: 80px;
      z-index: 9;
    }
  }

  :deep(.bk-form-item) {
    margin-bottom: 0;
  }
</style>
