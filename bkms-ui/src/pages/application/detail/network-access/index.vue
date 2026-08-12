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
  <div class="flex flex-col h-full overflow-hidden">
    <!-- 自定义 Header 区域 -->
    <TabHeader
      v-model:active-tab="activeTab"
      :tabs="tabList"
      :title="$t('网络访问')"
    />

    <!-- 内容区域 -->
    <div class="px-[24px] py-[20px] flex-1 overflow-auto">
      <component :is="currentTabComponent" />
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { Component } from 'vue';
  import { computed } from 'vue';

  import { useI18n } from 'vue-i18n';
  import TabHeader from '~/components/tab-header.vue';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';

  import Service from './service/service.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();

  interface TabConfig extends TabItem {
    component: Component;
  }

  // Tab 名称常量（模板与校验同源）
  const TAB_NAMES = {
    service: 'service',
  } as const;

  const tabList: TabConfig[] = [{ label: `${t('服务')} (Service)`, name: TAB_NAMES.service, component: Service }];

  // Tab 与 URL query（activeTab）双向同步锚定
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: Object.values(TAB_NAMES),
        default: TAB_NAMES.service,
      },
    },
  });
  const activeTab = fields.activeTab;

  // 根据 activeTab 获取当前要显示的组件
  const currentTabComponent = computed(() => {
    const currentTab = tabList.find(tab => tab.name === activeTab.value);
    return currentTab?.component || tabList[0]?.component;
  });
</script>
