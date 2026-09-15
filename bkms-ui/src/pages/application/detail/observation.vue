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
    <FlexRow class="mb-[16px] flex-shrink-0 bg-[#EAEBF0] px-[12px] py-[8px] shadow-[0_2px_4px_0_#0000001a]">
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
      </template>
    </FlexRow>
    <div
      ref="iframeContainerRef"
      class="min-h-0 flex-1 bg-white"
    >
      <MonitorIframe
        v-if="currentApm && !apmConfigMissing"
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
            {{ $t('应用框架配置文件中未配置可观测') }}
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
  import {
    type MonitorIframeObservabilityType,
    type MonitorIframeSetParamsPayload,
    useMonitorIframe,
  } from '~/composables/use-monitor-iframe';
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
  // iframe URL：初始化（observabilityQuery 就绪）后构建，传给 monitor-iframe 渲染
  const iframeUrl = ref('');

  const serviceName = ref('');
  const apmConfigMissing = ref(false);
  const isInitializingEnvFromUrl = ref(false);

  const currentApm = ref<GetEnvApmOutput | null>(null);

  // iframe v-if 挂载状态（currentApm && !apmConfigMissing）与其上一状态记录：
  // 用于区分「iframe 冷启动（首次进入 / v-if 卸载后重建，需以新 URL 整体重建）」与「持续挂载热更新（URL 锁定 + postMessage 下发参数）」
  const isIframeMounted = computed(() => !!currentApm.value && !apmConfigMissing.value);
  let prevIframeMounted = false;

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

  // 优先当前环境绑定的 APM name，没有则使用环境名
  const apmAppName = computed(() => {
    return currentApm.value?.name || curEnv.value;
  });

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

  // 观测参数：初始为静态占位（异步依赖未就绪），异步依赖（currentApm/serviceName）就绪后由主 watch 的 computeObservabilityQuery() 更新为正确值
  // 后续参数变更通过 monitorIframeRef.sendSetParams 下发，不重载 iframe
  const observabilityQuery = ref<MonitorIframeObservabilityType>({
    ...DEFAULT_APM_CONFIG,
    dashboardId: 'service-default-overview',
    'filter-app_name': '',
  });

  // APM 配置指引文档链接
  const apmGuideUrl = computed(() => {
    if (appDetailStore.appType === 'taf') {
      return `${import.meta.env.BK_DOC_URL}${DOC_LINKS.APM_GUIDE_TAF}`;
    }
    const language = appDetailStore.appDetail?.appModelSpec?.trpcSpec?.language;
    const docPath = language === 'cpp' ? DOC_LINKS.APM_GUIDE_TRPC_CPP : DOC_LINKS.APM_GUIDE_TRPC_GO;
    return `${import.meta.env.BK_DOC_URL}${docPath}`;
  });

  /**
   * 计算观测参数：其余字段由 URL 恢复；filter 恒用当前环境派生值（apmAppName / serviceName）
   * 注意：改为手动调用后，调用时机决定了 iframe src 的更新时机（避免 route-change 回写导致 iframe 频繁重载）
   */
  function computeObservabilityQuery() {
    const {
      'filter-app_name': _queryAppName,
      'filter-service_name': _queryServiceName,
      ...restApmQuery
    } = parsedApmQuery.value;
    const result = {
      ...DEFAULT_APM_CONFIG,
      dashboardId: 'service-default-overview',
      ...restApmQuery,
      'filter-app_name': apmAppName.value,
      'filter-service_name': serviceName.value,
    };
    return result;
  }

  /** iframe 内部状态变化：同步完整观测参数到 URL apmQuery 字段（供刷新/分享恢复） */
  function handleRouteChange(payload: { hash: string; href: string; query: Record<string, unknown> }) {
    const { query } = payload;
    if (!query || Object.keys(query).length === 0) return;
    fields.apmQuery.value = JSON.stringify(query);
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
  const fetchApmData = async (): Promise<null | { apmName: string; seq: number; service: string }> => {
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
    const serviceNameValue = service?.serviceName ?? '';
    serviceName.value = serviceNameValue;
    // 返回 seq：主 watch 需在 fetch 返回后的每个 await 恢复点复查，覆盖 fetchApmData 内部的过期拦截到副作用执行间的空窗
    return { seq, apmName: apm?.name || env.name, service: serviceNameValue };
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
      const isStale = () => result.seq !== fetchSeq;
      if (isStale()) return;
      // 2. 派生数据就绪后重新计算观测参数（filter 恒用派生值）
      observabilityQuery.value = computeObservabilityQuery();
      const merged: ApmQueryParams = {
        ...parsedApmQuery.value,
        'filter-app_name': result.apmName,
        // 无条件覆盖：service 为空也显式写空串，清掉 URL 残留的旧环境服务名（避免刷新/分享串环境）
        'filter-service_name': result.service,
      };
      // 3. 快照写入 URL（env + apmQuery，供刷新/分享恢复；显式携带当前 env 避免覆盖并发写入）
      await router.replace({
        query: {
          ...route.query,
          env: curEnv.value,
          apmQuery: JSON.stringify(merged),
        },
      });
      if (isStale()) return;
      // 4. 判断 iframe 是否需冷启动（首次进入，或经历 v-if 卸载重建——如切到无 APM 环境后再切回）：
      //    冷启动时 iframe 处于加载期，sendSetParams 会被静默丢弃（isIframeReady=false），必须以新 URL 整体重建
      const needRebuildIframe = !prevIframeMounted && isIframeMounted.value;
      prevIframeMounted = isIframeMounted.value;
      // 5. 重建/初始化走 buildIframeUrl（iframe 首载即带正确参数）；已初始化且持续挂载时 URL 锁定，参数变更仅走 postMessage（二者二选一）
      if (iframeUrl.value && !needRebuildIframe) {
        // service 为空也显式下发空串：平台语义「空串=无过滤/清除」，防止 iframe 内部残留上一环境的服务筛选（与 URL 快照/URL 构建路径一致）
        const payload: MonitorIframeSetParamsPayload = {
          'filter-app_name': result.apmName,
          'filter-service_name': result.service,
        };
        monitorIframeRef.value?.sendSetParams(payload);
      } else {
        const url = await buildIframeUrl(observabilityQuery.value);
        if (!isStale()) {
          iframeUrl.value = url;
        }
      }
    },
    { immediate: true },
  );
</script>
