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
  <div class="network-header-container bg-[#FFF] px-[24px]">
    <div class="pt-[14px] pb-[10px]">
      <span class="text-[16px] font-medium text-[#313238]">{{ title }}</span>
    </div>
    <Tab
      v-model:active="activeTabModel"
      :label-height="40"
      type="unborder-card"
    >
      <Tab.TabPanel
        v-for="tab in tabs"
        :key="tab.name"
        :label="tab.label"
        :name="tab.name"
      />
    </Tab>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Tab } from 'bkui-vue';

  interface TabItem {
    label: string;
    name: string;
  }

  const props = withDefaults(
    defineProps<{
      activeTab?: string;
      tabs?: TabItem[];
      title?: string;
    }>(),
    {
      title: '',
      tabs: () => [],
      activeTab: '',
    },
  );

  const emit = defineEmits<{
    'update:activeTab': [value: string];
  }>();

  const activeTabModel = computed({
    get: () => props.activeTab,
    set: (value: string) => emit('update:activeTab', value),
  });
</script>

<style lang="postcss" scoped>
  .network-header-container {
    box-shadow: 0 3px 4px 0 #0000000a;
    :deep(.bk-tab-header) {
      border: none !important;
    }
    :deep(.bk-tab-content) {
      display: none;
    }
  }
</style>
