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
  <div class="h-full overflow-auto flex flex-col">
    <!-- 环境选择器 -->
    <div
      class="bg-[#EAEBF0] shadow-[0_2px_4px_0_#0000001a] px-[12px] py-[8px] mx-[24px] mt-[20px] mb-[16px] flex items-center gap-[10px] flex-shrink-0"
    >
      <EnvPerspectiveSelect
        data-testid="deploy-config-env-select"
        :env-list="envList"
        :model-value="currentEnvName"
        :modified-env-names="configuredEnvNames"
        @change="handleEnvSelectChange"
      />
      <Popover render-type="auto">
        <i class="bkms-icon bkms-icon-circle-info text-[16px] text-[#63656e]"></i>
        <template #content>
          <p>{{ $t('全局：所有环境共用，不可按环境修改。') }}</p>
          <p>{{ $t('按环境：继承默认配置，支持按环境单独覆盖。') }}</p>
          <div class="flex items-center">
            <span>{{ deployConfigTipPrefix }}</span>
            <i class="inline-block w-[3px] h-[12px] bg-[#ff9c01] mx-[5px] flex-shrink-0"></i>
            <span>{{ deployConfigTipSuffix }}</span>
          </div>
        </template>
      </Popover>
    </div>
    <div class="px-[24px] min-h-0 flex-1 h-full">
      <!-- 骨架屏（仅 loading 时显示） -->
      <Skeleton
        :full-height="false"
        :loading="isLoading"
        :once="false"
      >
        <template #loading>
          <Layout.shape
            :height="28"
            width="100%"
          />
          <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
            <Layout.formItem />
            <Layout.formItem />
          </div>
          <Layout.shape
            :height="28"
            width="100%"
          />
          <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
            <Layout.formItem />
            <Layout.formItem />
          </div>
          <Layout.shape
            :height="28"
            width="100%"
          />
          <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
            <Layout.formItem class="col-span-2" />
            <Layout.formItem />
            <Layout.formItem />
            <Layout.formItem />
            <Layout.formItem />
          </div>
          <Layout.shape
            :height="28"
            width="100%"
          />
          <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
            <Layout.formItem />
            <Layout.formItem />
          </div>
          <div class="h-[60px] bg-[#fff]"></div>
        </template>
      </Skeleton>

      <!-- 实际内容（始终渲染，loading 时隐藏） -->
      <div
        v-show="!isLoading"
        class="flex flex-col gap-[16px] pb-[20px]"
        :data-env-name="currentEnvName"
        data-testid="deploy-config-content"
      >
        <!-- 启动配置 -->
        <ProgramConfig :current-env="currentExtendedEnv" />

        <!-- 生命周期 -->
        <Lifecycle
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
        />

        <!-- 健康探针 -->
        <HealthProbe
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
          @loading-change="handleHealthProbeLoadingChange"
        />

        <!-- 资源规格 -->
        <ResourcesForm
          ref="resourcesFormRef"
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
          @loading-change="handleResourcesLoadingChange"
        />

        <!-- 更新策略 -->
        <UpdateStrategyForm
          ref="updateStrategyFormRef"
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
          @loading-change="handleStrategyLoadingChange"
        />

        <!-- 元数据配置 -->
        <MetadataConfig
          ref="metadataConfigRef"
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
          @loading-change="handleMetadataLoadingChange"
        />

        <!-- 网络接入 -->
        <NetworkAccess
          ref="networkAccessRef"
          :current-env="currentExtendedEnv"
          @env-modified-change="handleEnvModifiedChange"
        />

        <!-- 开发模式由技术栈能力控制，默认配置不提供环境级开关。 -->
        <div v-if="devModeSupported && !currentExtendedEnv?.isDefault">
          <DevModeForm
            :current-env="currentExtendedEnv"
            @env-modified-change="handleEnvModifiedChange"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { Popover } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { EnvOutput } from '~/@types/v1/env';
  import { AppSpecService, EnvService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import { isRuntimeModSupported } from '~/composables/use-app-runtime';
  import { DEFAULT_ENV_NAME, useAppConfigEnv } from '~/pages/application/detail/app-config/use-app-config-env';
  import { useAppDetail } from '~/stores/app-detail';

  import DevModeForm from './components/dev-mode-form.vue';
  import HealthProbe from './components/health-probe.vue';
  import Lifecycle from './components/lifecycle.vue';
  import MetadataConfig from './components/metadata-config.vue';
  import NetworkAccess from './components/network-access.vue';
  import ProgramConfig from './components/program-config.vue';
  import ResourcesForm from './components/resources-form.vue';
  import UpdateStrategyForm from './components/update-strategy-form.vue';
  import EnvPerspectiveSelect from './env-perspective-select.vue';

  import type { ExtendedEnv } from './components/types';

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const devModeSupported = computed(() => isRuntimeModSupported(appDetailStore.runtime, 'devMode'));
  const { getEnvStorageKey, getStoredEnvName, saveEnvNameToStorage } = useAppConfigEnv();
  let initializationID = 0;

  function isCurrentScope(storageKey: string, requestID: number) {
    return storageKey === getEnvStorageKey() && requestID === initializationID;
  }

  // 将带 {0} 占位符的 tip 文本拆分为前后两部分，中间插入黄线图标
  const deployConfigTipText = computed(() => t('带 {0} 标记的字段已脱离继承，不再跟随默认配置变更。', ['\x00']));
  const deployConfigTipPrefix = computed(() => deployConfigTipText.value.split('\x00')[0] || '');
  const deployConfigTipSuffix = computed(() => deployConfigTipText.value.split('\x00')[1] || '');

  const resourcesFormRef = ref<InstanceType<typeof ResourcesForm>>();
  const updateStrategyFormRef = ref<InstanceType<typeof UpdateStrategyForm>>();
  // MetadataConfig 暴露 handleEnvChange，和资源规格/更新策略一起参与环境切换确认。
  const metadataConfigRef = ref<InstanceType<typeof MetadataConfig>>();
  const networkAccessRef = ref<InstanceType<typeof NetworkAccess>>();

  // 各组件的 loading 状态
  const healthProbeLoading = ref(true);
  const resourcesLoading = ref(true);
  const strategyLoading = ref(true);
  const metadataLoading = ref(true);
  const envSwitchingLoading = ref(false);

  // AppSpec section 首次加载完成前保持骨架屏，避免不同卡片混合显示新旧环境数据。
  const isLoading = computed(
    () =>
      envSwitchingLoading.value ||
      healthProbeLoading.value ||
      resourcesLoading.value ||
      strategyLoading.value ||
      metadataLoading.value,
  );

  /** 健康探针 loading 状态变化回调 */
  function handleHealthProbeLoadingChange(val: boolean) {
    healthProbeLoading.value = val;
  }

  /** 元数据配置 loading 状态变化回调 */
  function handleMetadataLoadingChange(val: boolean) {
    metadataLoading.value = val;
  }

  /** 资源规格 loading 状态变化回调 */
  function handleResourcesLoadingChange(val: boolean) {
    resourcesLoading.value = val;
  }

  /** 更新策略 loading 状态变化回调 */
  function handleStrategyLoadingChange(val: boolean) {
    strategyLoading.value = val;
  }

  // 当前选中的环境名称（'__default__' 表示默认配置，其他为具体环境名）
  const currentEnvName = ref<string>(DEFAULT_ENV_NAME);

  // 环境列表（用于查找环境详情）
  const envList = ref<EnvOutput[]>([]);

  // 已配置部署规格的环境名称列表
  const configuredEnvNames = ref<string[]>([]);

  function clearRouteEnvName() {
    if (!('envName' in route.query)) return;
    const { envName: _envName, ...query } = route.query;
    router.replace({ query });
  }

  /** 获取稳定的环境展示对象，不依赖部署规格概览，避免保存后刷新概览时重建当前环境对象。 */
  function getExtendedEnv(envName: string): ExtendedEnv | null {
    const isDefault = envName === '' || envName === DEFAULT_ENV_NAME;

    if (isDefault) {
      return {
        name: '',
        displayName: t('默认配置'),
        type: '',
        isDefault: true,
      };
    }

    const env = envList.value.find(e => e.name === envName);
    if (!env) return null;

    return {
      name: env.name!,
      displayName: env.displayName || env.name,
      type: env.type,
      isDefault: false,
    };
  }

  /** 获取配置加载使用的环境对象，通过 isModified 决定是否需要请求环境级配置。 */
  function getLoadExtendedEnv(envName: string): ExtendedEnv | null {
    const env = getExtendedEnv(envName);
    if (!env) return null;

    return {
      ...env,
      isModified: !env.isDefault && configuredEnvNames.value.includes(env.name),
    };
  }

  function getRouteEnvName() {
    const envName = route.query.envName;
    return Array.isArray(envName) ? envName[0] || '' : envName || '';
  }

  // 转换为 ExtendedEnv（提供给各表单组件）
  const currentExtendedEnv = computed<ExtendedEnv | null>(() => getExtendedEnv(currentEnvName.value));

  // 获取部署规格概览（哪些环境已配置）
  async function fetchDeploySpecOverview() {
    const appID = appDetailStore.appID;
    if (!appID) return;
    const storageKey = getEnvStorageKey();
    const requestID = initializationID;
    try {
      const result = await AppSpecService.getAppSpecOverview({
        appID,
      });
      if (!isCurrentScope(storageKey, requestID)) return;
      configuredEnvNames.value = result?.configuredEnvs || [];
    } catch {
      if (isCurrentScope(storageKey, requestID)) configuredEnvNames.value = [];
    }
  }

  // 获取环境列表
  async function getEnvList() {
    const appID = appDetailStore.appID;
    if (!appID) {
      envList.value = [];
      return false;
    }
    const storageKey = getEnvStorageKey();
    const requestID = initializationID;
    try {
      const result = await EnvService.listAppEnvs({ appID });
      if (!isCurrentScope(storageKey, requestID)) return false;
      envList.value = result;
      return true;
    } catch {
      if (isCurrentScope(storageKey, requestID)) envList.value = [];
      return false;
    }
  }

  /** 通知参与 AppSpec 配置的组件切换环境，并返回各组件的确认结果。 */
  function handleAppSpecSectionsEnvChange(env: ExtendedEnv) {
    return Promise.all([
      resourcesFormRef.value?.handleEnvChange(env),
      updateStrategyFormRef.value?.handleEnvChange(env),
      metadataConfigRef.value?.handleEnvChange(env),
      networkAccessRef.value?.handleEnvChange(env),
    ]);
  }

  // 环境修改状态变更回调（保存/恢复默认后刷新概览）
  async function handleEnvModifiedChange() {
    const storageKey = getEnvStorageKey();
    const requestID = initializationID;
    await fetchDeploySpecOverview();
    if (!isCurrentScope(storageKey, requestID)) return;
    if (!currentExtendedEnv.value?.isDefault) {
      saveEnvNameToStorage(currentEnvName.value);
    }
  }

  // 环境选择变化
  async function handleEnvSelectChange(envName: string) {
    if (envSwitchingLoading.value || networkAccessRef.value?.isSaving()) return;

    const storageKey = getEnvStorageKey();
    const requestID = initializationID;
    envSwitchingLoading.value = true;

    try {
      // 构造扩展环境对象
      const extendedEnv = getLoadExtendedEnv(envName);
      if (!extendedEnv) return;

      // 通知所有参与 AppSpec 配置的组件先确认脏数据；任一组件取消则保持当前环境不变。
      const [resourcesConfirmed, strategyConfirmed, metadataConfirmed, networkAccessConfirmed] =
        await handleAppSpecSectionsEnvChange(extendedEnv);
      if (!isCurrentScope(storageKey, requestID)) return;

      // 如果任一组件取消，不更新当前环境
      const isEnvChangeCanceled =
        resourcesConfirmed === false ||
        strategyConfirmed === false ||
        metadataConfirmed === false ||
        networkAccessConfirmed === false;
      if (isEnvChangeCanceled) {
        if (currentExtendedEnv.value) {
          await networkAccessRef.value?.handleEnvChange(currentExtendedEnv.value);
        }
        return;
      }

      currentEnvName.value = envName;
      saveEnvNameToStorage(envName);

      await nextTick();
    } finally {
      if (isCurrentScope(storageKey, requestID)) envSwitchingLoading.value = false;
    }
  }

  async function initDeployConfig() {
    const requestID = ++initializationID;
    const storageKey = getEnvStorageKey();
    const isCurrent = () => isCurrentScope(storageKey, requestID);
    resetDeployConfig();
    envSwitchingLoading.value = true;
    try {
      const [envListLoaded] = await Promise.all([getEnvList(), fetchDeploySpecOverview()]);
      if (!isCurrent()) return;
      await initEnvFromRouteOrStorage(envListLoaded, isCurrent);
    } finally {
      if (isCurrent()) envSwitchingLoading.value = false;
    }
  }

  /** 初始化指定环境及各配置组件。 */
  async function initEnv(envName: string, shouldSaveToStorage: boolean, isCurrent: () => boolean) {
    const extendedEnv = getLoadExtendedEnv(envName);
    if (!extendedEnv) return false;

    currentEnvName.value = envName;
    if (shouldSaveToStorage) {
      saveEnvNameToStorage(envName);
    }

    await nextTick();
    if (!isCurrent()) return false;
    await handleAppSpecSectionsEnvChange(extendedEnv);
    return isCurrent();
  }

  async function initEnvFromRouteOrStorage(envListLoaded: boolean, isCurrent: () => boolean) {
    const routeEnvName = getRouteEnvName();
    const storedEnvName = getStoredEnvName();

    try {
      if (routeEnvName && (await initEnv(routeEnvName, true, isCurrent))) {
        return;
      }
      if (!isCurrent()) return;

      if (!storedEnvName) {
        await initEnv(DEFAULT_ENV_NAME, false, isCurrent);
        return;
      }

      if (!(await initEnv(storedEnvName, true, isCurrent)) && isCurrent()) {
        // 只有成功取得完整环境列表才能确认环境已失效，接口失败时保留跨页记忆。
        if (envListLoaded) saveEnvNameToStorage(DEFAULT_ENV_NAME);
        await initEnv(DEFAULT_ENV_NAME, false, isCurrent);
      }
    } finally {
      if (routeEnvName && isCurrent()) {
        clearRouteEnvName();
      }
    }
  }

  function resetDeployConfig() {
    envList.value = [];
    configuredEnvNames.value = [];
    currentEnvName.value = DEFAULT_ENV_NAME;
    envSwitchingLoading.value = false;
  }

  watch(
    getEnvStorageKey,
    async () => {
      if (appDetailStore.appID) {
        await initDeployConfig();
      } else {
        initializationID += 1;
        resetDeployConfig();
      }
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    initializationID += 1;
  });
</script>
