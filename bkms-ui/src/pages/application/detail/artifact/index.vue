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
    <!-- Helm-like 应用：显示 TabHeader -->
    <template v-if="isHelmLike">
      <!-- 自定义 Header 区域 -->
      <TabHeader
        v-model:active-tab="activeTab"
        :tabs="tabList"
        :title="$t('制品管理')"
      />

      <!-- 内容区域 -->
      <div class="flex-1 overflow-auto p-[16px]">
        <div class="overflow-hidden rounded-[6px] bg-[#fff] p-[16px] shadow-[0_2px_4px_0_#1919290d]">
          <component :is="currentTabComponent" />
        </div>
      </div>
    </template>

    <!-- 非 Helm-like 应用：直接显示容器镜像（自带 header） -->
    <template v-else>
      <!-- 默认 Header 区域 -->
      <TabHeader :title="$t('制品管理')" />

      <div class="flex-1 overflow-auto p-[16px]">
        <div class="overflow-hidden rounded-[6px] bg-[#fff] p-[16px] shadow-[0_2px_4px_0_#1919290d]">
          <ContainerImage />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type { Component } from 'vue';
  import { computed } from 'vue';

  import { useI18n } from 'vue-i18n';
  import TabHeader from '~/components/tab-header.vue';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import { useAppDetail } from '~/stores/app-detail';

  import ContainerImage from './container-image.vue';
  import HelmChart from './helm-chart.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();

  const appDetailStore = useAppDetail();

  /** 是否为 Helm-like 应用（显示 TabHeader）；非 helm-like 无 Tab，activeTab 固定为默认值 */
  const isHelmLike = computed(() => isHelmLikeAppType(appDetailStore.appType));

  interface TabConfig extends TabItem {
    component: Component;
  }

  const tabList: TabConfig[] = [
    { label: t('容器镜像'), name: 'container-image', component: ContainerImage },
    { label: t('Helm Chart'), name: 'helm-chart', component: HelmChart },
  ];

  // Tab 与 URL query（activeTab）双向同步锚定；非 helm-like 应用无 Tab，activeTab 不参与 URL 同步
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: tabList.map(tab => tab.name),
        default: tabList[0]?.name || 'container-image',
        override: valueFromQuery => (isHelmLike.value ? (valueFromQuery ?? tabList[0]?.name) : ''),
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
