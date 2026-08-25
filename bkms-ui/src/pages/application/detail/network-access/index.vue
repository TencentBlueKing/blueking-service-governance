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
  import { useRoute } from 'vue-router';
  import TabHeader from '~/components/tab-header.vue';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';

  import HostPort from './host-port/host-port.vue';
  import Service from './service/service.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();
  const route = useRoute();

  interface TabConfig extends TabItem {
    component: Component;
  }

  // Tab 名称常量（模板与校验同源）
  const TAB_NAMES = {
    hostPort: 'hostPort',
    service: 'service',
  } as const;

  /**
   * 与侧栏导航同源：按路由 type 决定 Tab。
   * 同菜单跨应用切换时 store.appType 可能短暂滞后，用路由避免 Helm 仍显示 HostPort。
   */
  const routeAppType = computed(() => {
    const typeParam = route.params.type;
    return (Array.isArray(typeParam) ? typeParam[0] : typeParam) || '';
  });

  // Helm：仅 Service；tRPC / TAF：仅 HostPort 端口映射
  const tabList = computed<TabConfig[]>(() =>
    isHelmLikeAppType(routeAppType.value)
      ? [{ label: `${t('服务')} (Service)`, name: TAB_NAMES.service, component: Service }]
      : [{ label: t('HostPort 端口映射'), name: TAB_NAMES.hostPort, component: HostPort }],
  );

  // 当前类型下唯一合法 Tab（切换应用时忽略 URL 残留的 activeTab）
  const onlyTabName = computed(() => tabList.value[0]?.name || TAB_NAMES.service);

  // Tab 与 URL query（activeTab）双向同步；default / override 均跟当前类型唯一 Tab 同源
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: Object.values(TAB_NAMES),
        default: onlyTabName.value,
        override: () => onlyTabName.value,
      },
    },
  });
  const activeTab = fields.activeTab;

  // 根据 activeTab 获取当前要显示的组件
  const currentTabComponent = computed(() => {
    const currentTab = tabList.value.find(tab => tab.name === activeTab.value);
    return currentTab?.component || tabList.value[0]?.component;
  });
</script>
