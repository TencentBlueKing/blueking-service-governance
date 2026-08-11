<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- 自定义 Header 区域 -->
    <TabHeader
      v-model:active-tab="activeTab"
      :tabs="tabList"
      :title="$t('监控告警')"
    />

    <div class="flex-1 overflow-hidden px-[24px] py-[20px]">
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

  import AlertGroup from './alert-group.vue';
  import AlertRecordList from './alert-record-list.vue';
  import AlertStrategy from './alert-strategy.vue';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();

  const route = useRoute();
  const router = useRouter();

  // Tab 配置（扩展 TabItem，添加组件字段）
  interface TabConfig extends TabItem {
    component: Component;
  }

  const tabList: TabConfig[] = [
    { label: t('告警记录'), name: 'alert-record', component: AlertRecordList },
    { label: t('告警策略'), name: 'alert-strategy', component: AlertStrategy },
    { label: t('告警组'), name: 'alert-group', component: AlertGroup },
  ];

  // 从路由 query 中获取 activeTab，如果没有则使用第一项
  const activeTab = computed({
    get: () => {
      const tabFromQuery = route.query.activeTab as string;
      // 如果有有效的 tab 参数，返回它；否则返回第一项
      return tabFromQuery && tabList.some(tab => tab.name === tabFromQuery)
        ? tabFromQuery
        : tabList[0]?.name || 'alert-record';
    },
    set: (value: string) => {
      // 切换到非「告警策略」Tab 时清除 groupID 预筛选参数
      const { groupID: _gid, ...rest } = route.query;
      router.replace({
        query: {
          ...(value === 'alert-strategy' ? route.query : rest),
          activeTab: value,
        },
      });
    },
  });

  // 当前显示的组件
  const currentComponent = computed(() => {
    const tab = tabList.find(item => item.name === activeTab.value);
    return tab?.component || AlertRecordList;
  });

  // 首次进入时，如果没有 activeTab 参数，则添加默认值到 URL
  onMounted(() => {
    if (!route.query.activeTab) {
      router.replace({
        query: {
          ...route.query,
          activeTab: tabList[0]?.name || 'alert-record',
        },
      });
    }
  });
</script>
