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
    >
      <template #title-extra>
        <Button
          v-if="activeTab === 'framework-config'"
          class="float-right"
          text
          theme="primary"
          @click="mountPreviewVisible = true"
        >
          <i class="bkms-icon bkms-icon-yanjing-kejian mr-[4px] text-[14px]"></i>
          {{ $t('挂载预览') }}
        </Button>
      </template>
    </TabHeader>

    <div class="min-h-0 flex-1 overflow-hidden">
      <component :is="currentComponent" />
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { Component } from 'vue';
  import { computed, provide, ref } from 'vue';

  import { Button } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import TabHeader from '~/components/tab-header.vue';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import ConfigFiles from '~/pages/application/detail/app-config/config-files.vue';
  import DeployConfig from '~/pages/application/detail/app-config/deploy-config.vue';
  import EnvVariable from '~/pages/application/detail/app-config/env-variable.vue';
  import { MOUNT_PREVIEW_VISIBLE_KEY } from '~/pages/application/detail/app-config/use-config-file-defs';

  import type { TabItem } from '~/components/tab-header.vue';

  const { t } = useI18n();
  const mountPreviewVisible = ref(false);
  provide(MOUNT_PREVIEW_VISIBLE_KEY, mountPreviewVisible);

  // Tab 配置（扩展 TabItem，添加组件字段）
  interface TabConfig extends TabItem {
    component: Component;
  }

  const tabList: TabConfig[] = [
    { label: t('部署配置'), name: 'deploy-config', component: DeployConfig },
    { label: t('配置文件'), name: 'framework-config', component: ConfigFiles },
    { label: t('环境变量'), name: 'env-variable', component: EnvVariable },
  ];

  // Tab 与 URL query（activeTab）双向同步锚定
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: tabList.map(tab => tab.name),
        default: tabList[0]?.name || 'deploy-config',
      },
    },
  });
  const activeTab = fields.activeTab;

  // 当前显示的组件
  const currentComponent = computed(() => {
    const tab = tabList.find(item => item.name === activeTab.value);
    return tab?.component || DeployConfig;
  });
</script>
