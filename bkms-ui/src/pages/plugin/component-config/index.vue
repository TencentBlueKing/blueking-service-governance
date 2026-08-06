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
  <main class="flex flex-col w-full relative h-full bg-[#fff]">
    <span class="text-[16px] py-[14px] ml-[24px]">{{ $t('组件配置') }}</span>
    <Tab
      v-model:active="activeTab"
      class="min-h-[0px] flex-1"
      type="unborder-card"
    >
      <Tab.TabPanel
        name="instance"
        render-directive="if"
      >
        <template #label>
          <span class="text-[14px]">{{ $t('组件实例') }}</span>
        </template>
        <ComponentInstances />
      </Tab.TabPanel>
      <Tab.TabPanel
        name="component"
        render-directive="if"
      >
        <template #label>
          <span class="text-[14px]">{{ $t('组件') }}</span>
        </template>
        <SpaceComponent />
      </Tab.TabPanel>
    </Tab>
  </main>
</template>

<script setup lang="ts">
  import { ref } from 'vue';

  import { Tab } from 'bkui-vue';

  import ComponentInstances from './component-list.vue';
  import SpaceComponent from './space-component.vue';

  // 状态管理
  const activeTab = ref<'component' | 'instance'>('instance');
</script>
<style lang="postcss" scoped>
  :deep(.bk-tab-header) {
    box-shadow: 0 3px 4px 0 #0000000a;
    border-bottom: none;
    z-index: 1;
    padding-left: 24px !important;
    padding-right: 24px !important;
  }
  :deep(.bk-tab-content) {
    display: flex;
    min-height: 0px;
    min-width: 0px;
    overflow: auto;
    background-color: #f5f7fa;
    padding: 0 !important;

    .bk-tab-panel {
      flex: 1;
      min-height: 0px;
      min-width: 0px;
      display: flex;
      flex-direction: column;
    }
  }
  :deep(.bk-tab-header-nav .bk-tab-header-item) {
    padding: 0 !important;
    margin-right: 32px !important;
    line-height: 36px !important;
  }
</style>
