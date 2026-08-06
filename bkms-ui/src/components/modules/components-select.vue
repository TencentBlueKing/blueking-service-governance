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
  <Select
    :disabled="disabled"
    filterable
    :loading="loading"
    :model-value="modelValue"
    :placeholder="$t('请选择组件')"
    :with-validate="false"
    @change="handleChangeComponent"
  >
    <Select.Option
      v-for="item in filteredComponentList"
      :key="item.name"
      :name="item.name"
      :value="item.name"
    >
    </Select.Option>
  </Select>
</template>
<script lang="ts" setup>
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Select } from 'bkui-vue';
  import { RenderManagerService } from '~/api/modules/rendermanager';

  import type { ComponentType } from '~/@types/api';
  import type { ComponentOutputObj } from '~/@types/v1/app';

  const props = defineProps<{
    disabled?: boolean; // 是否禁用选择器
    modelValue: string;
    type?: Array<ComponentType> | ComponentType; // 是否只展示 Deploy 组件
  }>();

  const emits = defineEmits<{
    (e: 'update:modelValue', val: string): void;
    (e: 'change' | 'init', com: ComponentOutputObj): void;
  }>();

  const loading = ref(false);
  const componentList = ref<ComponentOutputObj[]>([]);
  const componentType = computed(() => {
    if (!props.type || !props.type?.length) return;

    return Array.isArray(props.type) ? props.type : [props.type];
  });
  const filteredComponentList = computed(() =>
    componentList.value.filter(
      item => !componentType.value || componentType.value?.includes(item.type as ComponentType),
    ),
  );

  function handleChangeComponent(val: string) {
    emits('update:modelValue', val);
  }

  async function handleGetComponentList() {
    loading.value = true;
    componentList.value = (await RenderManagerService.ListComponent().catch(() => [])) as ComponentOutputObj[];
    if (props.modelValue) {
      const com = componentList.value.find(item => item.name === props.modelValue);
      if (com) {
        emits('init', com);
      }
    }
    loading.value = false;
  }

  watch(
    () => props.modelValue,
    val => {
      if (!val) return;
      // 更新组件信息
      const com = componentList.value.find(item => item.name === val);
      if (com) {
        emits('change', com);
      }
    },
    { immediate: true },
  );

  onBeforeMount(() => {
    handleGetComponentList();
  });
</script>
