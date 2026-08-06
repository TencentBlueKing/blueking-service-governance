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
  <Sideslider
    v-model:is-show="isShow"
    quick-close
    render-directive="if"
    :width="960"
    @closed="handleClosed"
  >
    <template #header>
      <div class="flex items-center justify-between w-full pr-[16px]">
        <div class="flex items-center gap-[8px] min-w-0">
          <!-- 资源图标 -->
          <div
            class="flex items-center justify-center size-[32px] rounded-[4px] shrink-0"
            :style="{ background: statusConfig.iconBgColor, color: statusConfig.color }"
          >
            <component
              :is="iconComponent"
              v-if="iconComponent"
              class="size-[24px] svg-icon"
            />
            <span
              v-else
              class="text-[16px] font-600 text-[#fff]"
            >
              {{ nodeData?.kind?.charAt(0) || '?' }}
            </span>
          </div>
          <div class="flex flex-col">
            <span
              class="text-[12px] text-[#000000] fw-[500]"
              :title="nodeData?.name"
            >
              {{ nodeData?.name }}
            </span>
            <div class="flex items-center gap-[16px] text-[12px] text-[#8E9BB3]">
              <span> {{ $t('类型') }}：{{ nodeData?.kind }} </span>
              <div class="flex items-center gap-[4px]">
                <span>{{ $t('状态') }}：</span>
                <TopologyStatusIcon
                  :size="12"
                  :type="nodeData?.nodeStatus || 'unknown'"
                />
                <span>
                  {{ nodeData?.status }}
                </span>
              </div>
              <span v-if="detail?.createdAt"> {{ $t('创建时间') }}：{{ formatTime(detail.createdAt) }} </span>
            </div>
          </div>
        </div>
        <!-- WebConsole 按钮（仅Pod） -->
        <Button
          v-if="isPod"
          v-bk-tooltips="{
            content: $t('实例当前未处于运行状态，无法登录'),
            disabled: canLogin,
          }"
          class="shrink-0 ml-[12px]"
          :disabled="!canLogin"
          size="small"
          @click="handleOpenWebConsole"
        >
          <div class="flex items-center gap-[4px]">
            <i class="bkms-icon bkms-icon-yuanma text-[16px]"> </i>
            <span> WebConsole </span>
          </div>
        </Button>
      </div>
    </template>
    <template #default>
      <div class="h-full flex flex-col">
        <Tab
          v-model:active="activeTab"
          class="h-full"
          :label-height="42"
          type="unborder-card"
        >
          <Tab.TabPanel
            name="overview"
            render-directive="if"
          >
            <template #label>{{ $t('概览') }}</template>
            <DetailOverview
              :app-id="appId"
              :env-name="envName"
              :loading="detailLoading"
              :node-data="mergedNodeData"
            />
          </Tab.TabPanel>
          <Tab.TabPanel
            name="events"
            render-directive="if"
          >
            <template #label>
              <span>{{ $t('事件') }}</span>
            </template>
            <DetailEvents
              :app-id="appId"
              :env-name="envName"
              :node-id="nodeData?.id || ''"
            />
          </Tab.TabPanel>
          <Tab.TabPanel
            v-if="isPod"
            :disabled="!canViewLog"
            name="log"
            render-directive="if"
          >
            <template #label>
              <span
                v-bk-tooltips="{
                  content: $t('实例尚未创建成功或宿主机异常，暂无法获取日志'),
                  disabled: canViewLog,
                }"
                >{{ $t('日志') }}</span
              >
            </template>
            <DetailLog
              :app-id="appId"
              :env-name="envName"
              :node-name="nodeData?.name || ''"
            />
          </Tab.TabPanel>
          <Tab.TabPanel
            name="yaml"
            render-directive="if"
          >
            <template #label>YAML</template>
            <DetailYaml
              :app-id="appId"
              :env-name="envName"
              :node-id="nodeData?.id || ''"
            />
          </Tab.TabPanel>
        </Tab>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Sideslider, Tab } from 'bkui-vue';
  import { TopologyNodeDetail } from '~/@types/topology';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { formatTimeByTimezone } from '~/common/util';
  import { useTabManager } from '~/composables/use-tab-manager';

  import { KIND_ICON_MAP, LOG_ALLOWED_STATUSES, normalizeStatus, STATUS_CONFIG } from './constants';
  import DetailEvents from './detail-events.vue';
  import DetailLog from './detail-log.vue';
  import DetailOverview from './detail-overview.vue';
  import DetailYaml from './detail-yaml.vue';
  import TopologyStatusIcon from './topology-status-icon.vue';

  import type { TopoNodeData } from './types';

  const props = defineProps<{
    activeTab?: string;
    appId: string;
    envName: string;
    nodeData: null | TopoNodeData;
  }>();

  const { openTab, isTabOpen } = useTabManager();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const activeTab = ref<string>(props.activeTab || 'overview');
  const detail = ref<null | TopologyNodeDetail>(null);
  const detailLoading = ref(false);

  // 合并节点数据和详情数据
  const mergedNodeData = computed<null | (TopologyNodeDetail & TopoNodeData)>(() => {
    if (!props.nodeData) return null;
    if (!detail.value) return null;
    return {
      ...props.nodeData,
      ...detail.value,
    };
  });

  const isPod = computed(() => props.nodeData?.kind === 'Pod');
  const iconComponent = computed(() => KIND_ICON_MAP[props.nodeData?.kind ?? ''] ?? null);
  const statusConfig = computed(() => {
    const status = props.nodeData?.nodeStatus ?? normalizeStatus(props.nodeData?.status ?? 'unknown');
    return STATUS_CONFIG[status];
  });
  // 登录：仅 Running 状态可登录
  const canLogin = computed(() => props.nodeData?.status === 'Running');
  // 日志：Running、CrashLoopBackOff、Error、Completed、Succeeded 状态可查看日志
  const canViewLog = computed(() => LOG_ALLOWED_STATUSES.some(k => k === props.nodeData?.status?.toLowerCase()));

  /** 获取节点详情 */
  async function fetchNodeDetail() {
    if (!props.nodeData?.id || !props.appId || !props.envName) return;
    detailLoading.value = true;
    try {
      detail.value = await ApiServerService.GetTopologyNodeDetail({
        appID: props.appId,
        envName: props.envName,
        trafficLaneName: '',
        nodeID: props.nodeData.id,
      });
    } catch (_) {
      detail.value = null;
    } finally {
      detailLoading.value = false;
    }
  }

  function formatTime(time: string) {
    return formatTimeByTimezone(time);
  }

  function handleClosed() {
    detail.value = null;
    activeTab.value = 'overview';
  }

  /** 打开 WebConsole（参考 instance-list.vue） */
  async function handleOpenWebConsole() {
    if (!props.nodeData || !isPod.value) return;
    const podName = props.nodeData.name;
    if (!podName) return;
    const instanceKey = `${props.nodeData.extras?.ip || ''}-${podName}`;
    if (isTabOpen(instanceKey)) {
      await openTab(instanceKey);
      return;
    }
    try {
      const res = await ApiServerService.CreateAppInstanceWebConsole({
        appID: props.appId,
        envName: props.envName,
        instanceID: podName,
      });
      if (!res?.url) return;
      await openTab(res.url, instanceKey);
    } catch (error) {
      console.error(error);
    }
  }

  /** nodeData 变化时自动获取详情并同步 activeTab */
  watch(
    () => props.nodeData,
    newData => {
      if (newData) {
        activeTab.value = props.activeTab || 'overview';
        detail.value = null;
        fetchNodeDetail();
      }
    },
  );

  /** activeTab 变化时同步 */
  watch(
    () => props.activeTab,
    newTab => {
      if (newTab) {
        activeTab.value = newTab;
      }
    },
  );
</script>

<style lang="postcss" scoped>
  .svg-icon :deep(svg) {
    width: 100%;
    height: 100%;
  }

  :deep(.bk-tab-header) {
    padding: 0 24px;
    background-color: #fff;
  }
  :deep(.bk-modal-content) {
    overflow: hidden !important;
    scrollbar-gutter: auto !important;
  }
  :deep(.bk-tab-header-nav .bk-tab-header-item) {
    padding: 0 !important;
    margin-right: 24px !important;
    font-size: 14px;
  }
  :deep(.bk-sideslider-content) {
    height: calc(100vh - 53px);
  }
  :deep(.bk-tab-content) {
    padding: 0 !important;
    height: calc(100% - 42px);
  }
</style>
