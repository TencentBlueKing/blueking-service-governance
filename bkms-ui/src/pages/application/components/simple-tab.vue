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
  <div class="flex items-center h-[42px] border-b-[1px] border-[#DCDEE5]">
    <slot name="tab-prefix"></slot>
    <div
      v-for="item in tabItems"
      :key="item.id"
      :class="[
        'flex items-center justify-center px-[20px] h-full cursor-pointer text-[14px] text-[#4D4F56]',
        activeID === item.id ? 'border-b-[1px] border-[#3A84FF] !text-[#3A84FF]' : '',
      ]"
      @click="handleChangeActive(item.id)"
    >
      {{ item.name }}
    </div>
  </div>
  <slot></slot>
</template>
<script lang="ts" setup>
  import { computed, provide, reactive, ref, watch } from 'vue';

  import type SimpleTabPanel from './simple-tab-panel.vue';

  export interface IProvide {
    activeID: ITabItem['id'];
    registry: (key: PropertyKey, data: InstanceType<typeof SimpleTabPanel>) => void;
    unregistry: (id: number | string) => void;
  }

  export interface ITabItem {
    id: number | string;
    name: string;
  }

  interface Emits {
    (e: 'update:modelValue', id: ITabItem['id']): void;
  }
  interface IProps {
    modelValue: ITabItem['id'];
  }

  const props = defineProps<IProps>();
  const emits = defineEmits<Emits>();

  const tabPanels = ref<Map<PropertyKey, InstanceType<typeof SimpleTabPanel>>>(new Map());
  const tabItems = computed(() => {
    const values = tabPanels.value.values();
    return [...values].map(panel => ({ id: panel.id, name: panel.name }));
  });

  const activeID = ref<ITabItem['id']>(props.modelValue);
  watch(
    () => props.modelValue,
    () => {
      activeID.value = props.modelValue;
    },
  );

  function handleChangeActive(id: ITabItem['id']) {
    activeID.value = id;
    emits('update:modelValue', id);
  }

  // 注册子组件信息
  function registry(key: PropertyKey, panel: InstanceType<typeof SimpleTabPanel>) {
    if (tabPanels.value.has(key)) {
      console.warn(`repeat tab id ${key.toString()}`);
      return;
    }

    tabPanels.value.set(key, panel);
  }

  function unregistry(key: PropertyKey) {
    tabPanels.value.delete(key);
  }

  provide<IProvide>('simple-tab', reactive({ registry, unregistry, activeID }));
</script>
