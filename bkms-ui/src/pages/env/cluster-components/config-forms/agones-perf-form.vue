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
    form-type="vertical"
    :model="innerFormData"
    :rules="formRules"
  >
    <Form.FormItem
      :label="$t('开启优化')"
      property="enableOptimization"
    >
      <Switcher
        v-model="innerFormData.enableOptimization"
        theme="primary"
      />
    </Form.FormItem>

    <template v-if="innerFormData.enableOptimization">
      <div class="flex items-start gap-[16px]">
        <Form.FormItem
          class="flex-1 relative"
          :label="$t('Workers 数量')"
          property="workersCount"
          required
        >
          <Input
            v-model.number="innerFormData.workersCount"
            :min="0"
            type="number"
          />
        </Form.FormItem>

        <Form.FormItem
          class="flex-1"
          :label="$t('API 并发数')"
          property="apiConcurrency"
          required
        >
          <Input
            v-model.number="innerFormData.apiConcurrency"
            :min="0"
            type="number"
          />
        </Form.FormItem>
      </div>
    </template>
  </Form>
</template>

<script setup lang="ts">
  import { reactive, ref, watch } from 'vue';

  import { Form, Input, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ClusterAddonInfoOutput } from '~/@types/v1/cluster-addon';
  import { parseYamlValues } from '~/common/util';

  interface Emits {
    (e: 'update:formData', value: FormDataOutput): void;
  }

  /** 对外输出的接口数据结构 */
  interface FormDataOutput {
    enableOptimization: boolean;
    values: Record<string, unknown>;
  }

  interface InnerFormDataModel {
    apiConcurrency: null | number;
    enableOptimization: boolean;
    workersCount: null | number;
  }

  interface Props {
    addonInfo?: ClusterAddonInfoOutput | null;
    formData?: FormDataOutput | null;
    isUpdate?: boolean;
  }

  const INITIAL_FORM_DATA: InnerFormDataModel = {
    enableOptimization: false,
    workersCount: null,
    apiConcurrency: null,
  };

  const props = withDefaults(defineProps<Props>(), {
    addonInfo: null,
    formData: null,
    isUpdate: false,
  });

  const emit = defineEmits<Emits>();

  defineExpose({
    validate,
  });

  const { t } = useI18n();
  const formRef = ref();

  const innerFormData = reactive<InnerFormDataModel>({ ...INITIAL_FORM_DATA });

  // 自定义校验规则：正确处理 number 类型（0 是合法值）
  const formRules = {
    workersCount: [
      {
        required: true,
        message: () => t('Workers 数量不能为空'),
        trigger: 'change',
        validator: (value: unknown) => value !== null && value !== undefined && value !== '',
      },
    ],
    apiConcurrency: [
      {
        required: true,
        message: () => t('API 并发数不能为空'),
        trigger: 'change',
        validator: (value: unknown) => value !== null && value !== undefined && value !== '',
      },
    ],
  };

  /** 监听 addonInfo / isUpdate 变化，统一回填表单 */
  watch(
    [() => props.addonInfo, () => props.isUpdate],
    ([addonInfo, isUpdate]) => {
      if (!addonInfo) {
        applyFormData(INITIAL_FORM_DATA);
        return;
      }
      applyFormData(isUpdate ? buildUpdateFormData(addonInfo) : INITIAL_FORM_DATA);
    },
    { immediate: true },
  );

  // 监听内部表单变化，向外同步
  watch(innerFormData, () => {
    emitFormData();
  });

  /** 统一写入表单数据 */
  function applyFormData(values: InnerFormDataModel) {
    Object.assign(innerFormData, values);
    emitFormData();
  }

  /** 更新模式：构建表单回填数据 */
  function buildUpdateFormData(addonInfo: ClusterAddonInfoOutput): InnerFormDataModel {
    const currentValues = parseYamlValues(addonInfo.installInfo?.currentValues) as Record<
      string,
      Record<string, Record<string, unknown>>
    >;
    const controller = currentValues.agones?.controller ?? {};
    const numWorkers = controller.numWorkers;
    const apiServerQPS = controller.apiServerQPS;

    // 如果存在性能参数值，则默认开启优化
    const hasPerfValues = numWorkers !== undefined || apiServerQPS !== undefined;
    return {
      ...INITIAL_FORM_DATA,
      enableOptimization: hasPerfValues,
      workersCount: numWorkers != null ? Number(numWorkers) : null,
      apiConcurrency: apiServerQPS != null ? Number(apiServerQPS) : null,
    };
  }

  /** 向外同步表单数据（转换为接口数据结构） */
  function emitFormData() {
    if (!innerFormData.enableOptimization) {
      emit('update:formData', {
        enableOptimization: false,
        values: {},
      });
      return;
    }
    emit('update:formData', {
      enableOptimization: true,
      values: {
        agones: {
          controller: {
            numWorkers: Number(innerFormData.workersCount),
            apiServerQPS: Number(innerFormData.apiConcurrency),
          },
        },
      },
    });
  }

  /** 校验表单 */
  async function validate(): Promise<void> {
    // 未开启优化时跳过校验
    if (!innerFormData.enableOptimization) return;
    await formRef.value?.validate?.();
  }
</script>
