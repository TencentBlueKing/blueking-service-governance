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
  <div class="flex h-full min-h-0 flex-col">
    <div
      class="flex h-[52px] flex-shrink-0 items-center justify-between bg-white px-[24px] shadow-[0_3px_4px_0_#0000000a]"
    >
      <div class="text-[16px] text-[#313238]">{{ $t('仪表盘') }}</div>
      <Button
        text
        theme="primary"
        @click="configVisible = true"
      >
        <i class="bkms-icon bkms-icon-setting-line mr-[4px] text-[14px]"></i>
        <span class="text-[14px]">{{ $t('配置仪表盘') }}</span>
      </Button>
    </div>

    <div
      v-bkloading="{ loading }"
      class="flex min-h-0 flex-1 flex-col"
    >
      <!-- 未配置任何仪表盘时的空态引导 -->
      <Exception
        v-if="!dashboards.length"
        class="large-exception"
        scene="part"
        type="empty"
      >
        <template #type>
          <img src="/empty.svg" />
        </template>
        <template #description>
          <div class="text-[24px] text-[#313238]">{{ $t('暂未配置仪表盘') }}</div>
          <DashboardSetupGuide class="mt-[16px] inline-block text-left" />
        </template>
        <Button
          theme="primary"
          @click="configVisible = true"
        >
          {{ $t('配置仪表盘') }}
        </Button>
      </Exception>

      <!-- 已配置：顶部切换仪表盘，下方 iframe 嵌入展示 -->
      <div
        v-else
        class="flex min-h-0 flex-1 flex-col px-[24px] py-[20px]"
      >
        <FlexRow class="mb-[16px] flex-shrink-0 bg-[#EAEBF0] px-[12px] py-[8px] shadow-[0_2px_4px_0_#0000001a]">
          <template #left>
            <Select
              ref="dashboardSelectRef"
              v-model="selectedDashboardUid"
              class="w-[280px] bg-white"
              :clearable="false"
              filterable
              :placeholder="$t('请选择仪表盘')"
            >
              <Select.Option
                v-for="item in dashboards"
                :key="item.uid"
                :label="item.title || item.uid"
                :value="item.uid"
              />
              <template #extension>
                <Button
                  class="w-full"
                  text
                  theme="primary"
                  @click="handleCreateDashboardFromSelect"
                >
                  <Plus
                    class="mr-[4px]"
                    :height="18"
                    :width="18"
                  />
                  {{ $t('添加仪表盘') }}
                </Button>
              </template>
            </Select>
          </template>
          <template #right>
            <i
              class="bkms-icon bkms-icon-full-screen cursor-pointer rounded-[4px] bg-[#fafbfd] p-[4px] text-[16px] hover:text-[#3A84FF]"
              @click="handleFullScreen"
            ></i>
          </template>
        </FlexRow>
        <div
          ref="iframeContainerRef"
          class="min-h-0 flex-1 bg-white"
        >
          <GrafanaDashboardIframe
            v-if="selectedDashboard?.url"
            :url="selectedDashboard.url"
          />
          <!-- 绑定了仪表盘但缺少访问地址时的兜底空态 -->
          <Exception
            v-else
            class="large-exception"
            scene="part"
            type="empty"
          >
            <template #type>
              <img src="/empty.svg" />
            </template>
            <template #description>
              <div class="text-[24px] text-[#313238]">{{ $t('仪表盘地址缺失') }}</div>
              <div class="mt-[16px] text-[14px] leading-[22px] text-[#4D4F56]">
                {{ $t('当前仪表盘缺少监控平台访问地址，请重新配置') }}
              </div>
            </template>
            <Button
              theme="primary"
              @click="configVisible = true"
            >
              {{ $t('配置仪表盘') }}
            </Button>
          </Exception>
        </div>
      </div>
    </div>

    <DashboardConfig
      ref="dashboardConfigRef"
      v-model:is-show="configVisible"
      @updated="handleDashboardsUpdated"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Exception, Select } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useRoute, useRouter } from 'vue-router';
  import FlexRow from '~/components/flex-row.vue';
  import GrafanaDashboardIframe from '~/components/grafana-dashboard-iframe.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import DashboardConfig from './dashboard-config.vue';
  import DashboardSetupGuide from './dashboard-setup-guide.vue';
  import { useAppDashboards } from './use-app-dashboards';

  /** 选中仪表盘的 uid 同步到 URL，便于分享与刷新后恢复 */
  const DASHBOARD_UID_QUERY_KEY = 'dashboardUid';

  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const { dashboards, fetchDashboards, loading } = useAppDashboards();

  const configVisible = ref(false);
  const dashboardConfigRef = ref<null | { openCreateDialog: () => void }>(null);
  const dashboardSelectRef = ref<null | { hidePopover?: () => void }>(null);
  const iframeContainerRef = ref<HTMLElement | null>(null);
  const selectedDashboardUid = ref('');

  const selectedDashboard = computed(() => dashboards.value.find(item => item.uid === selectedDashboardUid.value));

  function getDashboardUidQuery() {
    const value = route.query[DASHBOARD_UID_QUERY_KEY];
    return typeof value === 'string' ? value : '';
  }

  function handleCreateDashboardFromSelect() {
    dashboardSelectRef.value?.hidePopover?.();
    dashboardConfigRef.value?.openCreateDialog();
  }

  /** 配置侧变更后的回调：有指定 uid 则直接选中，否则保留原选中项、失效时回退到第一个 */
  function handleDashboardsUpdated(uid?: string) {
    if (uid) {
      selectedDashboardUid.value = uid;
      syncDashboardQuery(uid);
      return;
    }

    const firstUid = dashboards.value[0]?.uid || '';
    selectedDashboardUid.value = dashboards.value.some(item => item.uid === selectedDashboardUid.value)
      ? selectedDashboardUid.value
      : firstUid;
    syncDashboardQuery(selectedDashboardUid.value);
  }

  /** 让 iframe 容器进入/退出全屏 */
  function handleFullScreen() {
    const el = iframeContainerRef.value;
    if (!el) return;
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      el.requestFullscreen().catch(() => {});
    }
  }

  /** 将选中 uid 写入地址栏，值不变时跳过以避免重复导航 */
  function syncDashboardQuery(value: string) {
    const current = getDashboardUidQuery();
    if (value === current) return;

    const { [DASHBOARD_UID_QUERY_KEY]: _dashboardUid, ...restQuery } = route.query;
    router.replace({
      query: value ? { ...restQuery, [DASHBOARD_UID_QUERY_KEY]: value } : restQuery,
    });
  }

  // 切换应用时重新拉取绑定列表
  watch(
    () => appDetailStore.appID,
    appID => {
      fetchDashboards(appID);
    },
    { immediate: true },
  );

  // 列表或地址栏变化时校正选中项：优先地址栏，其次保留当前，最后回退到第一个
  watch([dashboards, () => route.query[DASHBOARD_UID_QUERY_KEY]], () => {
    const uidFromQuery = getDashboardUidQuery();
    if (uidFromQuery && dashboards.value.some(item => item.uid === uidFromQuery)) {
      selectedDashboardUid.value = uidFromQuery;
      return;
    }
    if (dashboards.value.some(item => item.uid === selectedDashboardUid.value)) {
      return;
    }
    selectedDashboardUid.value = dashboards.value[0]?.uid || '';
  });

  // 用户手动切换下拉时同步到地址栏
  watch(selectedDashboardUid, value => {
    syncDashboardQuery(value);
  });
</script>
