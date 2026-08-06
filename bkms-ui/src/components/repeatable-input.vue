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
  <Form
    ref="formRef"
    :model="formModel"
  >
    <Form.FormItem
      v-for="(_, index) in modelValue"
      :key="index"
      error-display-type="tooltips"
      :icon-offset="300"
      :label="''"
      label-width="0"
      :property="'items.' + index"
      :required="required"
      :rules="itemRules"
    >
      <div class="flex w-full items-center">
        <Input
          :model-value="modelValue[index]"
          :placeholder="placeholder"
          @update:model-value="handleItemChange(index, $event)"
        />
        <Del
          class="ml-[8px] cursor-pointer text-[14px] text-[#979BA5] hover:text-[#4D4F56]"
          @click="handleDel(index)"
        ></Del>
      </div>
    </Form.FormItem>
    <Button
      text
      theme="primary"
      @click="handleAdd"
    >
      <div class="flex items-center">
        <span class="bkms-icon bkms-icon-plus-circle-shape text-[14px]"></span>
        <span class="ml-[6px] text-[12px]">{{ addText || $t('添加') }}</span>
      </div>
    </Button>
  </Form>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Form, Input } from 'bkui-vue';
  import { Del } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';

  const props = defineProps<{
    addText?: string;
    placeholder?: string;
    /** 是否必填，默认 false */
    required?: boolean;
    /** 是否在输入时移除首尾空格 */
    trimOnInput?: boolean;
  }>();

  const { t } = useI18n();
  const modelValue = defineModel<string[]>({ default: () => [] });

  const formRef = ref<InstanceType<typeof Form> | null>(null);

  /** Form 绑定的 model（将数组包装为对象，property 路径用 items.0 / items.1 ...） */
  const formModel = computed(() => ({ items: modelValue.value }));

  /** 每行的校验规则 */
  const itemRules = computed(() =>
    props.required
      ? [
          {
            message: t('必填项'),
            trigger: 'blur',
            validator: (val: string) => val.trim() !== '',
          },
        ]
      : [],
  );

  /** 清除校验状态 */
  function clearValidate() {
    formRef.value?.clearValidate?.();
  }

  /** 添加一行 */
  function handleAdd() {
    modelValue.value = [...modelValue.value, ''];
  }

  /** 删除一行 */
  function handleDel(index: number) {
    const next = [...modelValue.value];
    next.splice(index, 1);
    modelValue.value = next;
  }

  /** 单行内容变更 */
  function handleItemChange(index: number, value: string) {
    const next = [...modelValue.value];
    next[index] = props.trimOnInput ? value.trim() : value;
    modelValue.value = next;
  }

  /** 校验方法，供父组件调用 */
  async function validate() {
    // 没有行时，如果 required 则视为不通过
    if (props.required && modelValue.value.length === 0) return false;
    try {
      return await formRef.value?.validate();
    } catch {
      return false;
    }
  }

  defineExpose({
    clearValidate,
    validate,
    add: handleAdd,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-item) {
    margin-bottom: 16px;
    .bk-form-error-tips {
      right: 28px;
    }
  }
</style>
