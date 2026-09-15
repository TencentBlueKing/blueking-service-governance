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
  <div
    v-if="currentEnv"
    class="flex flex-col gap-[16px] h-full min-h-0"
  >
    <div class="bg-[#fff] shadow-[0_2px_4px_0_#1919290d] rounded-[2px] overflow-hidden">
      <ApmInstance
        :current-apm="currentApm"
        :data="currentEnv"
        @update:current-apm="getEnvApm"
      />
    </div>
    <MonitorIframe
      v-if="currentApm && iframeUrl"
      class="flex-1 min-h-0 bg-[#fff] shadow-[0_2px_4px_0_#1919290d] rounded-[2px] overflow-hidden"
      :url="iframeUrl"
      @route-change="handleRouteChange"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { GetEnvApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1';
  import { type ApmQueryParams, DEFAULT_APM_CONFIG } from '~/common/const';
  import MonitorIframe from '~/components/monitor-iframe.vue';
  import { useMonitorIframe } from '~/composables/use-monitor-iframe';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import { useEnvDetailStore } from '~/stores/env-detail';

  import ApmInstance from './apm-instance.vue';

  const envDetailStore = useEnvDetailStore();
  const currentEnv = computed(() => envDetailStore.currentEnv);
  const currentApm = ref<GetEnvApmOutput | null>(null);
  const iframeUrl = ref('');
  let requestSequence = 0;
  let buildUrlSeq = 0;
  let disposed = false;
  onBeforeUnmount(() => {
    disposed = true;
    ++requestSequence;
    ++buildUrlSeq;
  });
  const { buildIframeUrl } = useMonitorIframe('application', { apm_submenu: 0, needMenu: false });

  const { fields } = useUrlQuerySync({
    apmQuery: {
      queryKey: 'apmQuery',
      data: {
        default: '',
      },
    },
  });

  async function getEnvApm() {
    const envID = currentEnv.value?.id;
    const sequence = ++requestSequence;
    ++buildUrlSeq;
    iframeUrl.value = '';
    if (!envID) {
      currentApm.value = null;
      return;
    }

    currentApm.value = null;
    const apm = await BkintegrationsBkmonitorService.getEnvApm({ envID }, { interceptorErr: false }).catch(() => null);
    if (!disposed && sequence === requestSequence && currentEnv.value?.id === envID) {
      currentApm.value = apm;
    }
  }

  const apmAppName = computed(() => currentApm.value?.name);

  const parsedApmQuery = computed<ApmQueryParams>(() => {
    const raw = fields.apmQuery.value;
    if (!raw) return {};
    try {
      const parsed = JSON.parse(raw);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as ApmQueryParams) : {};
    } catch {
      return {};
    }
  });

  const observabilityQuery = computed(() => {
    const { 'filter-app_name': _queryAppName, ...restApmQuery } = parsedApmQuery.value;
    return {
      ...DEFAULT_APM_CONFIG,
      dashboardId: 'overview',
      ...restApmQuery,
      'filter-app_name': apmAppName.value ?? '',
    };
  });

  function handleRouteChange(payload: { hash: string; href: string; query: Record<string, unknown> }) {
    const { query } = payload;
    if (disposed || !query || Object.keys(query).length === 0) return;
    fields.apmQuery.value = JSON.stringify(query);
  }

  watch(
    () => currentEnv.value?.id,
    () => {
      getEnvApm();
    },
    { immediate: true },
  );

  watch(apmAppName, (appName, prevAppName) => {
    if (!appName) return;
    if (fields.apmQuery.value) {
      fields.apmQuery.value = JSON.stringify({ ...parsedApmQuery.value, 'filter-app_name': appName });
    }
    if (prevAppName && prevAppName !== appName) {
      iframeUrl.value = '';
    }
  });

  watch(
    () => observabilityQuery.value['filter-app_name'],
    appName => {
      if (iframeUrl.value || !appName) return;
      const seq = ++buildUrlSeq;
      buildIframeUrl(observabilityQuery.value)
        .then(url => {
          if (!disposed && seq === buildUrlSeq) {
            iframeUrl.value = url;
          }
        })
        .catch(() => {});
    },
    { immediate: true },
  );
</script>
