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
  <Skeleton :loading="!hasLoaded">
    <template #loading>
      <TopologySkeleton />
    </template>
    <div class="flex flex-col h-full custom-resource-topology">
      <FlexRow
        v-if="nodes.length > 0"
        class="h-[48px] py-[8px] px-[16px] shrink-0 shadow-[0_2px_4px_0_#0000001a]"
      >
        <template #left>
          <!-- <Tab
            v-model:active="activeTab"
            :tabs="tabs"
          /> -->
        </template>
        <template #right>
          <div class="flex items-center gap-[8px]">
            <TopologySearch
              class="min-w-[560px]"
              :nodes="searchableNodes"
              @locate="handleLocateNode"
              @update:selected-ids="handleSearchSelect"
            />
            <!-- <Button class="!min-w-[32px] px-[8px]">
              <i class="bkms-icon bkms-icon-setting-line text-[16px]"></i>
            </Button> -->
          </div>
        </template>
      </FlexRow>
      <!-- 无资源空状态 -->
      <Exception
        v-if="hasLoaded && nodes.length === 0"
        class="large-exception py-[18px] px-[24px]"
        scene="part"
        type="empty"
      >
        <template #type>
          <img src="/empty.svg" />
        </template>
        <template #description>
          <div class="text-[20px] text-[#313238] mb-[16px]">{{ $t('暂无资源') }}</div>
          <div class="text-[14px] text-[#4D4F56]">{{ $t('该环境下无任何资源') }}</div>
          <slot name="empty-deploy"></slot>
        </template>
      </Exception>
      <ResizeLayout
        v-else
        :border="false"
        class="flex-1 mt-[2px] min-h-0 custom-resize-layout"
        collapsible
        :initial-divide="260"
      >
        <template #aside>
          <TopologyStatistics
            v-model:visible-node-ids="visibleNodeIds"
            :nodes="nodes"
          />
        </template>
        <template #main>
          <ResourceTopology
            v-if="activeTab === 'topo'"
            class="h-full"
            :edges="edges"
            :focused-node-id="focusedNodeId"
            :nodes="nodes"
            :selected-node-ids="selectedNodeIds"
            :show-only-node-ids="visibleNodeIds"
            @menu-click="handleShowResourceDetail"
            @node-click="handleNodeDblClick"
          />
          <ResourceList
            v-else-if="activeTab === 'list'"
            :nodes="nodes"
          />
        </template>
      </ResizeLayout>
      <!-- 资源详情侧边栏 -->
      <NodeDetailSidebar
        v-model:is-show="showDetail"
        :active-tab="sidebarTab"
        :app-id="appDetail.appID"
        :env-name="props.envName"
        :node-data="sidebarNodeData"
      />
    </div>
  </Skeleton>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref, shallowRef, watch } from 'vue';

  import { useIntervalFn } from '@vueuse/core';
  import { Exception, ResizeLayout } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ResourceTopologyDataOutputObj } from '~/@types/v1/topology';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import NodeDetailSidebar from './node-detail-sidebar.vue';
  import ResourceList from './resource-list.vue';
  import ResourceTopology from './resource-topology.vue';
  import TopologySearch, { type SearchNode } from './topology-search.vue';
  import TopologySkeleton from './topology-skeleton.vue';
  import TopologyStatistics from './topology-statistics.vue';
  import { TopoNodeData } from './types';
  // import { generateMockTopologyData } from './utils/mock-data';

  const props = defineProps<{
    envName: string;
  }>();

  type TopologyData = Omit<ResourceTopologyDataOutputObj, 'nodes'> & {
    nodes: SearchNode[];
  };

  const { t } = useI18n();
  const appDetail = useAppDetail();
  // 轮询获取资源拓扑数据(30s)
  useIntervalFn(handleGetResourceTopology, 30000);

  const resourceData = shallowRef<TopologyData>();
  const hasLoaded = ref(false);
  const nodes = computed(() => resourceData.value?.nodes ?? []);
  const edges = computed(() => resourceData.value?.edges ?? []);

  const _tabs = [
    {
      label: t('拓扑模式'),
      name: 'topo',
      icon: 'bkms-icon bkms-icon-jiegou-shuxing',
    },
    {
      label: t('列表模式'),
      name: 'list',
      icon: 'bkms-icon bkms-icon-shitu-liebiao-2',
      disabled: true,
    },
  ];
  const activeTab = ref<(typeof _tabs)[number]['name']>('topo');

  const visibleNodeIds = ref<string[]>([]);

  /** 搜索组件可用的节点（受左侧过滤影响） */
  const searchableNodes = computed(() => {
    if (visibleNodeIds.value.length === 0) return nodes.value;
    const idSet = new Set(visibleNodeIds.value);
    return nodes.value.filter(n => idSet.has(n.id));
  });

  // 搜索节点
  const selectedNodeIds = ref<string[]>([]);
  const focusedNodeId = ref('');

  // 获取资源拓扑数据
  async function handleGetResourceTopology() {
    if (!appDetail.appID || !props.envName) return;

    resourceData.value = await ApiServerService.GetResourceTopology({
      appID: appDetail.appID,
      envName: props.envName,
      trafficLaneName: '',
    });
    // 使用 mock 数据（开发调试用）
    // resourceData.value = generateMockTopologyData(5000, {
    //   appID: appDetail.appID,
    //   envName: props.envName,
    //   namespace: props.envName,
    // }) as TopologyData;
    hasLoaded.value = true;
  }

  function handleLocateNode(nodeId: string) {
    focusedNodeId.value = nodeId;
  }

  function handleSearchSelect(ids: string[]) {
    selectedNodeIds.value = ids;
    if (ids.length === 0) focusedNodeId.value = '';
  }

  /** 显示资源详情 */
  const showDetail = ref(false);
  const sidebarNodeData = ref<null | TopoNodeData>(null);
  const sidebarTab = ref<string>('overview');

  /** 单击节点 → 打开侧栏（默认概览 Tab） */
  function handleNodeDblClick(data: TopoNodeData) {
    sidebarNodeData.value = data;
    sidebarTab.value = 'overview';
    showDetail.value = true;
  }

  /** 右键菜单点击 → 打开侧栏到对应 Tab */
  function handleShowResourceDetail(action: string, data: TopoNodeData) {
    // 右键菜单 action 映射到 tab name（YAML → yaml，其余 id 与 tab name 一致）
    const tabMap: Record<string, string> = { YAML: 'yaml' };
    const tab = tabMap[action] || action;
    sidebarNodeData.value = data;
    sidebarTab.value = tab;
    showDetail.value = true;
  }

  watch([() => props.envName, () => appDetail.appID], () => {
    handleGetResourceTopology();
  });

  onMounted(async () => {
    await handleGetResourceTopology();
  });
</script>

<style lang="postcss" scoped>
  :deep(.custom-resize-layout) {
    .bk-resize-layout-aside {
      border-right: 0;
      box-shadow: 0 2px 4px 0 #0000000f;
      position: relative;
      z-index: 2;
      &::after {
        width: 0;
      }
    }
  }
</style>
