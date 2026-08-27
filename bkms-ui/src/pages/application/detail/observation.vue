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
        :base-query-params="{
          needMenu: false,
        }"
        :observability-query="{
          'filter-app_name': apmAppName,
          'filter-service_name': serviceName,
          method: 'AVG',
          interval: 'auto',
          dashboardId: 'service-default-overview',
          from: 'now-1h',
          to: 'now',
          timezone: 'Asia/Shanghai',
          refreshInterval: -1,
          sceneType: 'overview',
          queryString: '',
          preciseFilter: false,
          isGroupByLimit: false,
        }"
        type="service"
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
  import { DOC_LINKS } from '~/common/const';
  import FlexRow from '~/components/flex-row.vue';
  import MonitorIframe from '~/components/monitor-iframe.vue';
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
  const route = useRoute();
  const router = useRouter();

  const envStore = useDeployEnvStore();
  const curEnv = ref(envStore.currentEnv);
  const trpcDeployStore = useTrpcDeployStore();

  // env 参数与当前环境双向同步：URL 无 env 时回退 store 当前环境（从部署管理切来可继承，便于分享直达）
  const { fields } = useUrlQuerySync({
    env: {
      queryKey: 'env',
      data: { default: envStore.currentEnv || '' },
    },
  });
  const targetEnvName = fields.env;

  const iframeContainerRef = ref<HTMLElement | null>(null);

  const serviceName = ref('');
  const apmConfigMissing = ref(false);
  const isInitializingEnvFromUrl = ref(false);

  const currentApm = ref<GetEnvApmOutput | null>(null);

  // 获取当前环境关联的 APM 实例
  async function getEnvApm() {
    const envID = trpcDeployStore.curEnvItem?.id;
    if (!envID) {
      currentApm.value = null;
      return;
    }
    currentApm.value = await BkintegrationsBkmonitorService.getEnvApm({ envID }, { interceptorErr: false }).catch(
      () => null,
    );
  }

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
    router.push({
      name: 'env',
      params: { space: route.params.space },
      query: {
        active: curEnv.value,
        activeTab: 'observability',
      },
    });
  }

  // 优先当前环境绑定的 APM name，没有则使用环境名
  const apmAppName = computed(() => {
    return currentApm.value?.name || curEnv.value;
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

  // 跳转到 APM 配置指引文档
  function handleViewApmGuide() {
    window.open(apmGuideUrl.value, '_blank');
  }

  function isApmConfigMissingError(err: unknown) {
    const details = (err as APMErrorResponse)?.error?.details;
    return Array.isArray(details) && details.some(detail => detail.code === 'APM_CONFIG_MISSING');
  }

  // 获取 APM 服务名称
  const fetchApmServiceName = async () => {
    if (!curEnv.value || !appDetailStore.appID) return;
    serviceName.value = '';
    apmConfigMissing.value = false;
    try {
      const result = await ApiServerService.GetApmServiceName(
        {
          appID: appDetailStore.appID,
          envName: curEnv.value,
        },
        { interceptorErr: false },
      );
      serviceName.value = result?.serviceName ?? '';
    } catch (err: unknown) {
      apmConfigMissing.value = isApmConfigMissingError(err);
    }
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
    [curEnv, () => appDetailStore.appID],
    () => {
      fetchApmServiceName();
    },
    { immediate: true },
  );

  watch(
    () => trpcDeployStore.curEnvItem,
    newVal => {
      if (newVal?.id) {
        getEnvApm();
      }
    },
    { immediate: true },
  );
</script>
