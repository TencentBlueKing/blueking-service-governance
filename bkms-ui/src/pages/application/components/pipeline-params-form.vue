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
    :key="key"
    ref="formRef"
    class="grid grid-cols-2 gap-x-[40px]"
    form-type="vertical"
    :model="form"
  >
    <Form.FormItem
      v-for="item in params"
      :key="item.id"
      :description="item?.description"
      :label="item?.id && item?.name ? `${item.id}(${item.name})` : item?.id || item?.name"
      :property="item.id"
      :required="item.required"
    >
      <DefaultValueInput
        v-model:value="form[item.id]"
        :default-value="item.defaultValue"
        :input-config="{
          placeholder: item.placeholder,
          readonly: item.readOnly,
        }"
      />
    </Form.FormItem>
  </Form>
</template>

<script lang="ts" setup>
  import { onMounted, ref, watch } from 'vue';

  import { Form } from 'bkui-vue';
  import DefaultValueInput from '~/components/default-value-input.vue';

  import type { BkCIPipelineVariableOutput } from '~/@types/v1/bkintegrations-bkci';

  interface IParam extends Omit<BkCIPipelineVariableOutput, 'id'> {
    id: string;
    placeholder?: string;
    value?: number | string;
  }
  interface IProps {
    params: IParam[];
  }
  const props = defineProps<IProps>();

  const formRef = ref();
  const form = ref<Record<string, number | string>>({});
  const key = ref(0);

  const initForm = () => {
    // 重置表单，避免保留之前流水线的参数
    form.value = {};
    const newForm: Record<string, number | string> = {};
    for (const item of props.params) {
      const key = item.id;
      const defaultValue = item?.defaultValue || '';
      newForm[key] = (item.value as number | string) || defaultValue;
    }
    form.value = newForm;
    updateKey();
  };

  const save = async () => {
    const result = {
      valid: true,
      data: {},
    };
    result.valid = await formRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);

    if (!result.valid) return result;

    result.data = form.value;
    return result;
  };

  const setData = (data: Record<string, number | string>) => {
    // 只设置传入的参数，不保留旧的表单数据
    const newForm: Record<string, number | string> = {};
    for (const item of props.params) {
      const key = item.id;
      const defaultValue = item?.defaultValue || '';
      newForm[key] = data[key] !== undefined ? data[key] : defaultValue;
    }
    form.value = newForm;
    updateKey();
  };

  /** 更新key */
  const updateKey = () => {
    key.value = Date.now();
  };

  onMounted(() => {
    initForm();
  });

  // 监听 params 变化，重新初始化表单
  watch(
    () => props.params,
    () => {
      initForm();
    },
    { deep: true },
  );

  defineExpose({
    save,
    setData,
  });
</script>
<style lang="postcss" scoped>
  :deep(.bk-form-item.is-required .bk-form-label:after) {
    position: relative;
    display: inline-block;
  }
</style>
