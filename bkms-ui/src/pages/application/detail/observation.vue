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
    <FlexRow
      class="mb-[16px] flex-shrink-0 bg-[#EAEBF0] px-[12px] py-[8px] shadow-[0_2px_4px_0_#0000001a]"
      rclass="flex items-center gap-[8px]"
    >
      <template #left>
        <div class="flex">
          <EnvSelectPanel
            v-model="curEnv"
            v-model:item="trpcDeployStore.curEnvItem"
            class="mr-[16px]"
            init-first-env-when-empty
          />
        </div>
      </template>
      <template #right>
        <i
          class="bkms-icon bkms-icon-full-screen text-[16px] bg-[#fafbfd] cursor-pointer p-[4px] rounded-[4px] hover:text-[#3A84FF]"
          @click="handleFullScreen"
        ></i>
        <a
          v-if="standaloneMonitorUrl"
          class="bkms-icon bkms-icon-jump-link text-[16px] bg-[#fafbfd] cursor-pointer p-[4px] rounded-[4px] hover:text-[#3A84FF]"
          :href="standaloneMonitorUrl"
          rel="noopener noreferrer"
          target="_blank"
        ></a>
      </template>
    </FlexRow>
    <div
      ref="iframeContainerRef"
      class="min-h-0 flex-1 bg-white"
    >
      <MonitorIframe
        v-if="isIframeMounted"
        ref="monitorIframeRef"
        :url="iframeUrl"
        @route-change="handleRouteChange"
      />
      <Exception
        v-else-if="apmConfigMissing"
        class="large-exception"
        scene="part"
        type="empty"
      >
        <template #type>
          <img src="/empty.svg" />
        </template>
        <template #description>
          <div class="text-[#313238] text-[24px]">{{ $t('尚未开启观测功能') }}</div>
          <div class="text-[#4D4F56] text-[14px] leading-[22px] mt-[16px]">
            {{
              $t(
                appDetailStore.appDetail?.framework === 'blank'
                  ? '应用尚未接入可观测'
                  : '应用框架配置文件中未配置可观测',
              )
            }}
          </div>
        </template>
        <Button
          theme="primary"
          @click="handleViewApmGuide"
        >
          {{ $t('查看配置指引') }}
        </Button>
      </Exception>
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
          <div class="text-[#4D4F56] text-[14px] leading-[22px]">{{ $t('当前环境未创建 APM 实例') }}</div>
        </template>
        <Button
          theme="primary"
          @click="handleGoCreateApm"
        >
          {{ $t('去创建') }}
        </Button>
      </Exception>
    </div>
  </div>
</template>
<script lang="ts" setup>
  import { computed, nextTick, ref, watch } from 'vue';

  import { Button, Exception } from 'bkui-vue';
  import { useRoute, useRouter } from 'vue-router';
  import { GetEnvApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1';
  import { type ApmQueryParams, DEFAULT_APM_CONFIG, DOC_LINKS } from '~/common/const';
  import FlexRow from '~/components/flex-row.vue';
  import MonitorIframe from '~/components/monitor-iframe.vue';
  import { envDetailLocation } from '~/composables/use-env-manager';
  import { type MonitorIframeSetParamsPayload, useMonitorIframe } from '~/composables/use-monitor-iframe';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  type APMErrorResponse = {
    error?: {
      details?: Array<{
        code?: string;
      }>;
    };
  };

  const DEFAULT_SERVICE_DASHBOARD = 'service-default-overview';

  const appDetailStore = useAppDetail();
  const router = useRouter();
  const route = useRoute();

  const envStore = useDeployEnvStore();
  const curEnv = ref(envStore.currentEnv);
  const trpcDeployStore = useTrpcDeployStore();

  // env 参数与当前环境双向同步：URL 无 env 时回退 store 当前环境（从部署管理切来可继承，便于分享直达）
  const { fields } = useUrlQuerySync({
    env: {
      queryKey: 'env',
      data: { default: envStore.currentEnv || '' },
    },
    apmQuery: {
      queryKey: 'apmQuery',
      data: {
        default: '',
      },
    },
  });
  const targetEnvName = fields.env;

  const iframeContainerRef = ref<HTMLElement | null>(null);

  // monitor-iframe 组件实例（调用其暴露的 sendSetParams 下发参数）
  const monitorIframeRef = ref<InstanceType<typeof MonitorIframe>>();

  // monitor-iframe 公共能力：URL 构建（内部自取 bizId；apm_nav_list:false 隐藏 APM 导航栏面包屑，服务观测场景由页面自持导航）
  const { buildIframeUrl } = useMonitorIframe('service', { needMenu: false, apm_nav_list: false });
  // iframe src 只在冷启动时构建，后续视图变化由监控回报。
  const iframeUrl = ref('');
  const iframeRouteHash = ref('');
  const iframeTarget = ref<{ filters: MonitorIframeSetParamsPayload; pending?: { appChanged: boolean } }>();

  const apmConfigMissing = ref(false);
  const isInitializingEnvFromUrl = ref(false);

  const currentApm = ref<GetEnvApmOutput | null>(null);

  const isIframeMounted = computed(() => !!currentApm.value && !apmConfigMissing.value);

  function isApplicationView(hash: string) {
    return /^#\/apm\/application\/?(?:\?|$)/.test(hash);
  }

  const standaloneMonitorUrl = computed(() => {
    if (!isIframeMounted.value || !iframeUrl.value) return '';
    let url: URL;
    try {
      url = new URL(iframeUrl.value);
    } catch {
      return '';
    }
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return '';
    for (const key of ['needMenu', 'apm_nav_list', 'parentOrigin']) {
      url.searchParams.delete(key);
    }
    const hash = iframeRouteHash.value || url.hash;
    const target = iframeTarget.value;
    if (!target?.pending) {
      url.hash = hash;
      return url.href;
    }

    // 确认前使用目标过滤条件，只沿用最近视图中可跨环境复用的参数。
    const path = hash.split('?', 1)[0];
    const latestParams = new URLSearchParams(hash.slice(path.length + 1));
    const keepApplicationView = !target.filters['filter-service_name'] && isApplicationView(path);
    const keepPath = keepApplicationView || /^#\/apm\/service(?:\/|$)/.test(path);
    const params = new URLSearchParams({
      ...target.filters,
      sceneId: keepApplicationView ? 'apm_application' : 'apm_service',
    });
    const viewKeys = ['from', 'to', 'interval', 'timezone', 'refreshInterval'];
    if (keepPath) {
      viewKeys.push('dashboardId', 'sceneType');
    } else {
      params.set('dashboardId', DEFAULT_SERVICE_DASHBOARD);
      params.set('sceneType', DEFAULT_APM_CONFIG.sceneType);
    }
    for (const key of viewKeys) {
      const value = latestParams.get(key);
      if (value !== null) params.set(key, value);
    }
    url.hash = `${keepPath ? path : '#/apm/service'}?${params}`;
    return url.href;
  });

  // 全屏 iframe 容器
  function handleFullScreen() {
    const el = iframeContainerRef.value;
    if (!el) return;
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      el.requestFullscreen().catch(() => {});
    }
  }

  // 跳转到环境管理页面的「观测数据」Tab 去创建 APM
  function handleGoCreateApm() {
    const envId = trpcDeployStore.curEnvItem?.id;
    if (!envId) return;
    router.push(envDetailLocation(envId, 'observability'));
  }

  // 解析 URL 中的 iframe 观测参数（route-change 同步回写），异常/非对象/数组（如 JSON.parse('[1,2]')）返回空
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

  // APM 配置指引文档链接
  const apmGuideUrl = computed(() => {
    const framework = appDetailStore.appDetail?.framework;
    if (framework === 'blank') {
      return `${import.meta.env.BK_DOC_URL}${DOC_LINKS.APM_GUIDE_BLANK}`;
    }
    if (framework === 'taf') {
      return `${import.meta.env.BK_DOC_URL}${DOC_LINKS.APM_GUIDE_TAF}`;
    }
    const language = appDetailStore.appDetail?.language;
    const docPath = language === 'cpp' ? DOC_LINKS.APM_GUIDE_TRPC_CPP : DOC_LINKS.APM_GUIDE_TRPC_GO;
    return `${import.meta.env.BK_DOC_URL}${docPath}`;
  });

  /** iframe 内部状态变化：记录当前视图并同步观测参数，供新标签打开、刷新和分享。 */
  function handleRouteChange(payload: { hash: string; href: string; query: Record<string, unknown> }) {
    const { hash, query } = payload;
    if (query && Object.keys(query).length > 0) {
      fields.apmQuery.value = JSON.stringify(query);
    }
    if (typeof hash !== 'string' || !hash.startsWith('#/')) return;
    iframeRouteHash.value = hash;

    const target = iframeTarget.value;
    if (!target?.pending) return;
    const appName = query?.['filter-app_name'] ?? query?.app_name;
    const service = query?.['filter-service_name'] ?? query?.service_name ?? '';
    // 跨应用切换找不到对应服务时，监控也可能回到目标应用视图。
    if (
      appName === target.filters['filter-app_name'] &&
      (service === target.filters['filter-service_name'] ||
        (target.pending.appChanged && service === '' && isApplicationView(hash)))
    ) {
      target.pending = undefined;
    }
  }

  // 跳转到 APM 配置指引文档
  function handleViewApmGuide() {
    window.open(apmGuideUrl.value, '_blank');
  }

  function isApmConfigMissingError(err: unknown) {
    const details = (err as APMErrorResponse)?.error?.details;
    return Array.isArray(details) && details.some(detail => detail.code === 'APM_CONFIG_MISSING');
  }

  // 当前环境变化：并发拉取 APM 实例与服务名，name/id 同取自 curEnvItem，保证 filter-app_name / filter-service_name 同时就绪（避免 set-params 多次下发）
  // fetchSeq 序号：并发切换环境时丢弃过期请求结果，防止旧环境数据覆盖新状态
  let fetchSeq = 0;
  const fetchApmData = async (): Promise<null | { filters: MonitorIframeSetParamsPayload; seq: number }> => {
    const seq = ++fetchSeq;
    const env = trpcDeployStore.curEnvItem;
    if (!env?.id || !env.name || !appDetailStore.appID) return null;
    let configMissing = false;
    const [apm, service] = await Promise.all([
      BkintegrationsBkmonitorService.getEnvApm({ envID: env.id }, { interceptorErr: false }).catch(() => null),
      ApiServerService.GetApmServiceName(
        { appID: appDetailStore.appID, envName: env.name },
        { interceptorErr: false },
      ).catch((err: unknown) => {
        configMissing = isApmConfigMissingError(err);
        return null;
      }),
    ]);
    // 期间又触发了新请求（环境/应用切换），丢弃本次过期结果
    if (seq !== fetchSeq) return null;
    apmConfigMissing.value = configMissing;
    currentApm.value = apm;
    if (!isIframeMounted.value) {
      iframeUrl.value = '';
      iframeRouteHash.value = '';
      iframeTarget.value = undefined;
    }
    // 返回 seq：主 watch 需在 fetch 返回后的每个 await 恢复点复查，覆盖 fetchApmData 内部的过期拦截到副作用执行间的空窗
    return {
      filters: {
        'filter-app_name': apm?.name || env.name,
        'filter-service_name': service?.serviceName ?? '',
      },
      seq,
    };
  };

  // URL 中的 env → 初始化当前环境（首次进入时生效）；curEnv → 写回 URL（首次默认环境与用户切换环境都写入，便于分享直达）
  watch(
    targetEnvName,
    envName => {
      if (envName && envName !== curEnv.value && !isInitializingEnvFromUrl.value) {
        isInitializingEnvFromUrl.value = true;
        curEnv.value = envName;
        nextTick(() => {
          isInitializingEnvFromUrl.value = false;
        });
      }
    },
    { immediate: true },
  );

  watch(
    () => curEnv.value,
    envName => {
      if (envName && envName !== targetEnvName.value && !isInitializingEnvFromUrl.value) {
        targetEnvName.value = envName;
      }
    },
    { immediate: true },
  );

  watch(
    [() => trpcDeployStore.curEnvItem, () => appDetailStore.appID],
    async () => {
      // 1. 拉取当前环境的 APM 实例与服务名（fetchSeq 丢弃过期请求，返回 null 时直接结束）
      const result = await fetchApmData();
      if (!result) return;
      // fetchApmData 仅在「返回瞬间仍为最新」时返回结果，但其后副作用（router.replace / buildIframeUrl）仍含 await，
      // 期间可能有更新的环境切换插入，故每次 await 恢复后复查 seq，过期即整体丢弃（与 env/detail.vue buildUrlSeq 同思路）
      const { filters, seq } = result;
      const isStale = () => seq !== fetchSeq;
      if (isStale()) return;
      const merged = { ...parsedApmQuery.value, ...filters };
      // 快照写入 URL（env + apmQuery，供刷新/分享恢复；显式携带当前 env 避免覆盖并发写入）
      await router.replace({
        query: {
          ...route.query,
          env: curEnv.value,
          apmQuery: JSON.stringify(merged),
        },
      });
      if (isStale() || !isIframeMounted.value) return;
      // 冷启动构建 URL，持续挂载时通过 postMessage 更新过滤条件。
      if (iframeUrl.value) {
        const previousFilters = iframeTarget.value?.filters;
        const appChanged = filters['filter-app_name'] !== previousFilters?.['filter-app_name'];
        if (appChanged || filters['filter-service_name'] !== previousFilters?.['filter-service_name']) {
          iframeTarget.value = { filters, pending: { appChanged } };
        }
        monitorIframeRef.value?.sendSetParams(filters);
      } else {
        const url = await buildIframeUrl({
          ...DEFAULT_APM_CONFIG,
          dashboardId: DEFAULT_SERVICE_DASHBOARD,
          ...merged,
        });
        if (!isStale()) {
          iframeTarget.value = { filters };
          iframeUrl.value = url;
        }
      }
    },
    { immediate: true },
  );
</script>
