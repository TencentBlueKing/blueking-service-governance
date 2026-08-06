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
  <div class="tab-header-container relative z-10 bg-[#FFF] px-[24px]">
    <div
      v-if="title"
      :class="tabs.length > 0 ? 'pt-[14px] pb-[10px]' : 'h-[52px] flex items-center'"
    >
      <span class="text-[16px] font-medium text-[#313238]">{{ title }}</span>
      <slot name="title-extra"></slot>
    </div>

    <!-- Tab 区域 -->
    <Tab
      v-if="tabs.length > 0"
      v-model:active="activeTabModel"
      :label-height="labelHeight"
      type="unborder-card"
    >
      <Tab.TabPanel
        v-for="tab in tabs"
        :key="tab.name"
        :disabled="tab.disabled"
        :label="tab.label"
        :name="tab.name"
      />
    </Tab>

    <slot></slot>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Tab } from 'bkui-vue';

  /**
   * Tab 项配置
   */
  export interface TabItem {
    /** 是否禁用 */
    disabled?: boolean;
    /** Tab 显示文本 */
    label: string;
    /** Tab 唯一标识 */
    name: string;
  }

  const props = withDefaults(
    defineProps<{
      /** 当前激活的 Tab */
      activeTab?: string;
      /** Tab 标签高度 */
      labelHeight?: number;
      /** Tab 列表配置 */
      tabs?: TabItem[];
      /** 标题文本 */
      title?: string;
    }>(),
    {
      title: '',
      tabs: () => [],
      activeTab: '',
      labelHeight: 40,
    },
  );

  const emit = defineEmits<{
    /** 更新激活的 Tab */
    'update:activeTab': [value: string];
  }>();

  const activeTabModel = computed({
    get: () => props.activeTab,
    set: (value: string) => emit('update:activeTab', value),
  });
</script>

<style lang="postcss" scoped>
  .tab-header-container {
    box-shadow: 0 3px 4px 0 #0000000a;
    border-bottom: none;

    :deep(.bk-tab-header) {
      padding-left: 0 !important;
      padding-right: 0 !important;
      border: none !important;
    }

    :deep(.bk-tab-header-nav .bk-tab-header-item) {
      padding: 0 !important;
      margin-right: 32px !important;
      line-height: 36px !important;
      font-size: 14px;
    }

    :deep(.bk-tab-content) {
      display: none;
    }
  }
</style>
