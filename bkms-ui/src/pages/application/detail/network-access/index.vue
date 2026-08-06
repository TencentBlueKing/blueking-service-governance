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
  import { computed, onBeforeUnmount, watch } from 'vue';

  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import TabHeader from '~/components/tab-header.vue';

  import Service from './service/service.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();

  const route = useRoute();
  const router = useRouter();

  interface TabConfig extends TabItem {
    component: Component;
  }

  const tabList: TabConfig[] = [{ label: `${t('服务')} (Service)`, name: 'service', component: Service }];

  // 初始化 activeTab query 参数（在组件创建时同步执行）
  if (!route.query.activeTab) {
    router.replace({
      query: {
        ...route.query,
        activeTab: tabList[0]?.name || 'service',
      },
    });
  }

  // 从路由 query 中获取 activeTab，如果没有则使用第一项
  const activeTab = computed({
    get: () => {
      const tabFromQuery = route.query.activeTab as string;
      // 如果有有效的 tab 参数，返回它；否则返回第一项
      return tabFromQuery && tabList.some(tab => tab.name === tabFromQuery)
        ? tabFromQuery
        : tabList[0]?.name || 'service';
    },
    set: (value: string) => {
      router.push({
        query: {
          ...route.query,
          activeTab: value,
        },
      });
    },
  });

  // 根据 activeTab 获取当前要显示的组件
  const currentTabComponent = computed(() => {
    const currentTab = tabList.find(tab => tab.name === activeTab.value);
    return currentTab?.component || tabList[0]?.component;
  });

  // 监听路由变化，确保 activeTab 有效
  watch(
    () => route.query.activeTab,
    newTab => {
      // 如果 query 参数不是有效的 tab，则重置为第一项
      if (newTab && !tabList.some(tab => tab.name === newTab)) {
        router.replace({
          query: {
            ...route.query,
            activeTab: tabList[0]?.name || 'service',
          },
        });
      }
    },
  );

  // 组件卸载前清除 activeTab query 参数
  onBeforeUnmount(() => {
    // 如果父组件已经清理过了（切换应用/菜单），这里就不再处理
    if (route.query.activeTab) {
      const { activeTab, ...restQuery } = route.query;
      // 使用 nextTick 延迟执行，让父组件的路由跳转先完成
      setTimeout(() => {
        if (router.currentRoute.value.query.activeTab) {
          router.replace({
            query: restQuery,
          });
        }
      }, 0);
    }
  });
</script>
