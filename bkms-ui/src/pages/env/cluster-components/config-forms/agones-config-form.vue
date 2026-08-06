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
  >
    <div class="flex items-start gap-[16px]">
      <Form.FormItem
        class="flex-1"
        :label="$t('命名空间')"
        property="namespace"
        required
      >
        <Input
          v-model="innerFormData.namespace"
          disabled
          placeholder="bcs-system"
        />
      </Form.FormItem>

      <Form.FormItem
        class="flex-1"
        :label="$t('Chart 版本')"
        property="chartVersion"
        required
      >
        <Select
          v-model="innerFormData.chartVersion"
          filterable
          :list="chartVersionOptions"
          :placeholder="$t('请选择Agones版本')"
        />
      </Form.FormItem>
    </div>

    <div class="flex items-start gap-[16px]">
      <Form.FormItem
        class="flex-1"
        :label="`Controller ${$t('副本数')}`"
        property="controllerReplicas"
        required
      >
        <Input
          v-model.number="innerFormData.controllerReplicas"
          type="number"
        />
      </Form.FormItem>

      <Form.FormItem
        class="flex-1"
        :label="`Allocator ${$t('副本数')}`"
        property="allocatorReplicas"
        required
      >
        <Input
          v-model.number="innerFormData.allocatorReplicas"
          type="number"
        />
      </Form.FormItem>
    </div>

    <Form.FormItem
      :label="$t('镜像仓库地址')"
      property="imageRepository"
      required
    >
      <Input
        v-model.trim="innerFormData.imageRepository"
        :placeholder="$t('示例：gcr.io/agones-images')"
      />
      <p class="text-[#979BA5] text-[12px] mt-[4px]">{{ $t('如需使用私有镜像仓库，请填写完整地址') }}</p>
    </Form.FormItem>
  </Form>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Form, Input, Select } from 'bkui-vue';
  import { ClusterAddonInfoOutput } from '~/@types/v1/cluster-addon';
  import { parseYamlValues } from '~/common/util';

  interface Emits {
    (e: 'update:formData', value: FormDataOutput): void;
  }

  /** 对外输出的接口数据结构 */
  interface FormDataOutput {
    chartVersion: string;
    namespace: string;
    values: {
      agones: {
        allocator: { replicas: number };
        controller: { replicas: number };
        image: { registry: string | undefined };
      };
    };
  }

  interface InnerFormDataModel {
    allocatorReplicas: null | number;
    chartVersion: string;
    controllerReplicas: null | number;
    imageRepository: string;
    namespace: string;
  }

  interface Props {
    addonInfo?: ClusterAddonInfoOutput | null;
    formData?: FormDataOutput | null;
    isUpdate?: boolean;
  }

  const INITIAL_FORM_DATA: InnerFormDataModel = {
    chartVersion: '',
    namespace: 'bcs-system',
    controllerReplicas: null,
    allocatorReplicas: null,
    imageRepository: 'mirrors.tencent.com/bkms',
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

  const formRef = ref();

  const innerFormData = reactive<InnerFormDataModel>({ ...INITIAL_FORM_DATA });

  /** Chart 版本选项 */
  const chartVersionOptions = computed(() => {
    const versions = props.addonInfo?.chartInfo?.availableVersions || [];
    return versions.map(v => ({ label: v, value: v }));
  });

  /** 监听 addonInfo / isUpdate 变化，统一回填表单 */
  watch(
    [() => props.addonInfo, () => props.isUpdate],
    ([addonInfo, isUpdate]) => {
      if (!addonInfo) {
        applyFormData(INITIAL_FORM_DATA);
        return;
      }
      applyFormData(isUpdate ? buildUpdateFormData(addonInfo) : buildInstallFormData(addonInfo));
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

  /** 安装模式：构建表单回填数据 */
  function buildInstallFormData(_addonInfo: ClusterAddonInfoOutput): InnerFormDataModel {
    return {
      ...INITIAL_FORM_DATA,
      chartVersion: _addonInfo.chartInfo?.defaultChartVersion || '',
      namespace: 'bcs-system',
    };
  }

  /** 更新模式：构建表单回填数据 */
  function buildUpdateFormData(addonInfo: ClusterAddonInfoOutput): InnerFormDataModel {
    const currentValues = parseYamlValues(addonInfo.installInfo?.currentValues) as Record<
      string,
      Record<string, Record<string, unknown>>
    >;
    return {
      ...INITIAL_FORM_DATA,
      chartVersion: addonInfo.installInfo?.currentChartVersion || '',
      namespace: 'bcs-system',
      controllerReplicas: currentValues.agones?.controller?.replicas
        ? Number(currentValues.agones.controller.replicas)
        : null,
      allocatorReplicas: currentValues.agones?.allocator?.replicas
        ? Number(currentValues.agones.allocator.replicas)
        : null,
      imageRepository: String(currentValues.agones?.image?.registry ?? ''),
    };
  }

  /** 向外同步表单数据（转换为接口数据结构） */
  function emitFormData() {
    emit('update:formData', {
      chartVersion: innerFormData.chartVersion,
      namespace: innerFormData.namespace,
      values: {
        agones: {
          image: {
            registry: innerFormData.imageRepository || undefined,
          },
          controller: {
            replicas: Number(innerFormData.controllerReplicas),
          },
          allocator: {
            replicas: Number(innerFormData.allocatorReplicas),
          },
        },
      },
    });
  }

  /** 校验表单 */
  async function validate(): Promise<void> {
    await formRef.value?.validate?.();
  }
</script>
