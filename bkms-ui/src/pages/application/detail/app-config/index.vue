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
      :title="$t('应用配置')"
    />

    <div class="flex-1 overflow-hidden">
      <component :is="currentComponent" />
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { Component } from 'vue';
  import { computed, onMounted } from 'vue';

  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import TabHeader from '~/components/tab-header.vue';

  import DeployConfig from './deploy-config.vue';
  import EnvVariable from './env-variable.vue';
  import FrameworkConfig from './framework-config.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();

  const route = useRoute();
  const router = useRouter();

  // Tab 配置（扩展 TabItem，添加组件字段）
  interface TabConfig extends TabItem {
    component: Component;
  }

  const tabList: TabConfig[] = [
    { label: t('部署配置'), name: 'deploy-config', component: DeployConfig },
    { label: t('框架配置文件'), name: 'framework-config', component: FrameworkConfig },
    { label: t('环境变量'), name: 'env-variable', component: EnvVariable },
  ];

  // 从路由 query 中获取 activeTab，如果没有则使用第一项
  const activeTab = computed({
    get: () => {
      const tabFromQuery = route.query.activeTab as string;
      // 如果有有效的 tab 参数，返回它；否则返回第一项
      return tabFromQuery && tabList.some(tab => tab.name === tabFromQuery)
        ? tabFromQuery
        : tabList[0]?.name || 'deploy-config';
    },
    set: (value: string) => {
      router.replace({
        query: {
          ...route.query,
          activeTab: value,
        },
      });
    },
  });

  // 当前显示的组件
  const currentComponent = computed(() => {
    const tab = tabList.find(item => item.name === activeTab.value);
    return tab?.component || DeployConfig;
  });

  // 首次进入时，如果没有 activeTab 参数，则添加默认值到 URL
  onMounted(() => {
    if (!route.query.activeTab) {
      router.replace({
        query: {
          ...route.query,
          activeTab: tabList[0]?.name || 'deploy-config',
        },
      });
    }
  });
</script>
