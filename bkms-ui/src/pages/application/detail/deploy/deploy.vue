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
    <TabHeader
      v-model:active-tab="activeTab"
      :tabs="tabList"
      :title="$t('部署管理')"
    >
      <template #title-extra>
        <Button
          v-if="canManageFeatureEnvs"
          class="feature-env-entry float-right"
          text
          theme="primary"
          @click="isShowFeatureEnvSideslider = true"
        >
          <i18n-t keypath="应用关联的特性环境：{0} 个">
            <span class="font-bold">{{ featureEnvCount || '--' }}&nbsp;</span>
          </i18n-t>
        </Button>
      </template>
    </TabHeader>

    <!-- 与构建管理一致：内容区透出导航容器灰色底，仅 TabHeader 为白底 -->
    <div class="flex flex-1 min-h-0 flex-col overflow-hidden">
      <!-- 部署总览是跨环境视角，事件与部署历史只需要环境选择器，两者都不出现顶部栏；用 v-show 保留环境列表请求 -->
      <FlexRow
        v-show="isTopBarVisible"
        class="mx-[24px] mt-[20px] mb-[16px] shrink-0 bg-[#EAEBF0] shadow-[0_2px_4px_0_#0000001a] px-[12px] py-[8px]"
      >
        <template #left>
          <div class="flex">
            <!--
              事件、部署历史把选择器挪进页面筛选行；Teleport 是移动同一个实例，不会重建导致环境列表重复请求。
              Teleport 只在 to 变化时才重新解析目标节点，投放点又要等对应 Tab 渲染出来才存在，
              所以停留在顶部栏时把 to 指向 body 占位，靠 to 的变化触发重新解析。
            -->
            <Teleport
              :disabled="!isEnvSelectInline"
              :to="isEnvSelectInline ? DEPLOY_ENV_SELECT_SLOT_SELECTOR : 'body'"
            >
              <EnvSelectPanel
                :key="envSelectRefreshKey"
                ref="envSelectPanelRef"
                v-model="curEnv"
                v-model:model-values="curEnvs"
                class="mr-[16px]"
                init-first-env-when-empty
                :mode="envSelectMode"
                :multi-selectable="isEnvMultiSelectable"
                preserve-missing-model-value
                @update:deploy-status-list="deployStatusList = $event"
                @update:env-list="envList = $event"
                @update:item="handleEnvChange"
                @update:items="handleEnvsChange"
                @update:loading="envListLoading = $event"
                @update:mode="isMultiEnvMode = $event === 'multi'"
              />
            </Teleport>
            <!-- 部署状态 -->
            <KeyValueBadge
              v-if="isDeployStatusVisible && !isMultiEnvMode"
              class="min-w-[240px] text-[12px]"
              :key-name="$t('部署状态')"
              :key-width="64"
            >
              <ColorIcon
                class="ml-[8px] mr-[4px]"
                :icon="curDeployStatus?.icon || ''"
                :size="12"
              />
              <span class="leading-[20px] text-[#4D4F56]">{{ curDeployStatus?.text }}</span>
              <i
                v-if="curDeployStatus?.isFailed && curDeployStatus?.message"
                v-bk-tooltips="{
                  content: curDeployStatus?.message,
                  theme: 'dark',
                }"
                class="bkms-icon bkms-icon-circle-info ml-[4px] cursor-pointer text-[16px] text-[#C4C6CC]"
              ></i>
            </KeyValueBadge>
          </div>
        </template>
        <!-- 多环境不显示操作 -->
        <template
          v-if="isDeployStatusVisible && !isMultiEnvMode"
          #right
        >
          <!-- 部署/特性部署 -->
          <DeployActionButton
            :label="$t('部署')"
            :loading="precheckLoading"
            :show-feature-deploy="canFeatureDeploy"
            @deploy="handleShowFullUpdateDialog"
            @feature-deploy="handleShowFeatureDeploy"
          />
          <!-- 扩缩容 -->
          <ScaleInstance
            :effective-replicas="effectiveDeploySpec?.replicas"
            @update="fetchEffectiveDeploySpec"
          />
          <Popover
            ref="morePopoverRef"
            placement="bottom"
            theme="light"
            trigger="click"
          >
            <i
              :class="[
                'inline-block bg-[#fff] ml-[6px] size-[32px] leading-[32px]',
                'border border-[#C4C6CC] rounded-[2px] cursor-pointer',
                'bkms-icon bkms-icon-more-fill',
              ]"
            ></i>
            <template #content>
              <Button
                text
                @click="handleRemoveDeploy()"
              >
                {{ $t('移除部署') }}
              </Button>
            </template>
          </Popover>
        </template>
      </FlexRow>
      <!-- 无可用环境空状态 -->
      <Exception
        v-if="activeTab !== 'overview' && !hasAvailableEnv && !isEnvListLoading"
        class="large-exception px-[24px] py-[20px]"
        scene="part"
        type="empty"
      >
        <template #title>
          <div class="text-[#313238] text-[20px] leading-[28px]">
            {{ $t('暂无可用的环境') }}
          </div>
        </template>
        <template #description>
          <div class="text-[#4D4F56] text-[14px] leading-[22px]">
            {{ $t('环境必须先配置集群资源后才能部署应用') }}
          </div>
        </template>
        <Button
          class="mt-[8px]"
          theme="primary"
          @click="router.push({ name: 'env', params: { space: spaceStore.currentSpace } })"
        >
          {{ $t('前往配置') }}
        </Button>
      </Exception>
      <div
        v-else-if="activeTab === 'topo'"
        class="flex-1 min-h-0 px-[24px] pb-[20px]"
      >
        <ResourceTopology :env-name="curEnv" />
      </div>
      <div
        v-else
        :class="[
          'flex-1 min-h-0 px-[24px] pb-[20px]',
          isTopBarVisible ? '' : 'pt-[20px]',
          // 部署总览：操作栏固定，仅表格区域滚动
          activeTab === 'overview' ? 'flex flex-col overflow-hidden' : 'overflow-auto',
        ]"
      >
        <DeployOverview
          v-if="activeTab === 'overview'"
          ref="deployOverviewRef"
          :env-list="envList"
          @deploy="handleOverviewDeploy"
          @feature-deploy="handleShowFeatureDeploy"
          @update:deploy-targets="handleOverviewDeployTargetsUpdate"
          @view-instances="handleViewEnvInstances"
        />
        <template v-else-if="activeTab === 'instance'">
          <!-- 构建状态提示 -->
          <Alert
            v-if="isBuildAlertVisible"
            class="mb-[24px]"
            :closable="buildAlertInfo!.closable"
            :theme="buildAlertInfo!.theme"
          >
            <template #icon>
              <ColorIcon
                v-if="buildAlertInfo!.theme === 'info'"
                icon="loading"
                :size="14"
              />
              <Success
                v-else-if="buildAlertInfo!.theme === 'success'"
                fill="#65C389"
                height="14px"
                width="14px"
              />
              <Close
                v-else-if="buildAlertInfo!.theme === 'error'"
                fill="#EA3636"
                height="14px"
                width="14px"
              />
              <Warn
                v-else
                fill="#FF9C01"
                height="14px"
                width="14px"
              />
              <span class="text-[#4D4F56] text-[12px] font-bold ml-[8px]">{{ buildAlertInfo!.statusText }}</span>
            </template>
            <template #default>
              <div class="flex px-[24px] gap-[36px] text-[12px] h-[18px] leading-[18px]">
                <span>{{ $t('代码分支') }}: {{ latestDeployStatus?.branch || '--' }}</span>
                <span>{{ $t('镜像 Tag') }}: {{ latestDeployStatus?.imageTag || '--' }}</span>
                <span>{{ $t('操作人') }}: {{ latestDeployStatus?.operator || '--' }}</span>
                <Button
                  class="inline-flex items-center"
                  text
                  theme="primary"
                  @click="handleGotoPipeline"
                >
                  {{ $t('查看日志') }}
                </Button>
              </div>
            </template>
          </Alert>
          <template v-if="isMultiEnvMode || isDeployStatusVisible">
            <!-- 单环境 -->
            <InstanceList
              v-if="!isMultiEnvMode"
              :key="curEnv"
              :has-deploy-record="latestDeployStatus?.hasDeployRecord"
              @remove-deploy="handleRemoveDeploy"
            />
            <!-- 多环境 -->
            <MultiEnvInstanceTable
              v-else
              :requestable-env-names="requestableMultiEnvNames"
              @remove-deploy="handleRemoveDeploy"
            />
          </template>
          <!-- 未部署状态 -->
          <Skeleton
            v-else
            :full-height="false"
            :loading="initLoading"
          >
            <template #loading>
              <div class="flex items-center justify-between mb-[12px]">
                <div class="flex items-center gap-[8px]">
                  <Layout.shape />
                  <Layout.shape />
                </div>
                <Layout.shape :width="348" />
              </div>
              <Layout.table />
            </template>
            <Exception
              v-if="!initLoading"
              class="large-exception"
              scene="part"
              type="empty"
            >
              <template #type>
                <img src="/empty.svg" />
              </template>
              <template #description>
                <div class="text-[#4D4F56] text-[14px] leading-[22px]">{{ $t('该环境尚未部署应用') }}</div>
              </template>
              <!-- 部署/特性部署 -->
              <DeployActionButton
                class="mt-[8px]"
                :label="$t('立即部署')"
                :loading="precheckLoading"
                :show-feature-deploy="canFeatureDeploy"
                @deploy="handleShowQuicklyDeploy"
                @feature-deploy="handleShowFeatureDeploy"
              />
            </Exception>
          </Skeleton>
        </template>
        <DeployEvent v-else-if="activeTab === 'event'" />
        <DeployHistory v-else-if="activeTab === 'history'" />
      </div>
    </div>
    <!-- 立即部署 -->
    <QuicklyDeploy
      v-model:is-show="isShowQuicklyDeploy"
      :effective-replicas="effectiveDeploySpec?.replicas"
      :is-prod-env="isProdEnv"
      :target-envs="overviewDeployTargets"
      @update="handleQuickDeploySuccess"
    />
    <!-- 移除部署 -->
    <RemoveDeploy
      v-model:is-show="isShowRemoveDeploy"
      @update="handleRemoveDeploySuccess"
    />
    <!--全量更新-->
    <FullUpdate
      v-model:is-show="showFullUpdateDialog"
      :effective-replicas="effectiveDeploySpec?.replicas"
      :is-prod-env="isProdEnv"
      :latest-build-id="buildLogInfo.buildID"
      :latest-build-status="buildLogInfo.status"
      @update="handleUpdateDeploySuccess"
    />
    <!-- 特性部署 -->
    <FeatureDeploy
      v-model:is-show="isShowFeatureDeploy"
      :effective-replicas="effectiveDeploySpec?.replicas"
      @env-created="refreshFeatureEnvData"
      @update="handleFeatureDeploySuccess"
    />
    <!-- 普通部署在打开侧栏前执行环境变量预检查 -->
    <EnvVarPrecheckDialog
      v-model:is-show="isShowPrecheckDialog"
      :env-name="precheckEnvName"
      :undefined-vars="undefinedVars"
      @cancel="cancelDeploy"
      @go-modify="cancelDeploy"
      @still-deploy="continueDeploy"
    />
    <!-- 构建日志 -->
    <ViewBuildLog
      v-model:visible="showBuildLog"
      :build-info="buildLogInfo"
    />
    <!-- 应用关联的特性环境侧栏 -->
    <FeatureEnvSideslider
      v-model:is-show="isShowFeatureEnvSideslider"
      :error="featureEnvError"
      :list="featureEnvList"
      :loading="featureEnvLoading"
      @deleted="handleFeatureEnvDeleted"
      @deploy-removed="refreshFeatureEnvData"
      @refresh="fetchFeatureEnvList"
    />
  </div>
</template>
<script lang="ts" setup>
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { Alert, Button, Exception, Popover } from 'bkui-vue';
  import { Close, Success, Warn } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { AppSpecResourcesOutput } from '~/@types/v1/app-spec';
  import { LatestDeployStatus } from '~/@types/v1/deploy';
  import { EnvOutput, FeatureEnvOutput } from '~/@types/v1/env';
  import { AppSpecService, EnvService } from '~/api/modules/v1';
  import { APP_BUILD_STATUS, BUILD_INTERRUPT_STATUSES } from '~/common/enums/build';
  import { APP_DEPLOY_STATUS, DEPLOY_FAILED_STATUSES } from '~/common/enums/deploy';
  import ColorIcon from '~/components/color-icon.vue';
  import FlexRow from '~/components/flex-row.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import TabHeader from '~/components/tab-header.vue';
  import { isAppModelAppType, isHelmLikeAppType } from '~/composables/app-type';
  import { useAlertVisibility } from '~/composables/use-alert-visibility';
  import { type DeployStatusInfo, useDeployStatusMap } from '~/composables/use-deploy-status';
  import useInterval from '~/composables/use-interval';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import ViewBuildLog from '~/pages/application/detail/components/view-build-log/index.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';
  import { useSpaceStore } from '~/stores/space';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import KeyValueBadge from '../../components/key-value-badge.vue';
  import ResourceTopology from '../../components/topo/index.vue';
  import { DEPLOY_ENV_SELECT_SLOT_SELECTOR } from './constants';
  import DeployActionButton from './deploy-action-button.vue';
  import DeployEvent from './deploy-event.vue';
  import DeployHistory from './deploy-history.vue';
  import EnvVarPrecheckDialog from './env-var-precheck-dialog.vue';
  import FeatureDeploy from './feature-deploy.vue';
  import FeatureEnvSideslider from './feature-env-sideslider.vue';
  import FullUpdate from './instance-list/full-update.vue';
  import InstanceList from './instance-list/instance-list.vue';
  import MultiEnvInstanceTable from './instance-list/multi-env-instance-table.vue';
  import ScaleInstance from './instance-list/scale-instance.vue';
  import DeployOverview from './overview/deploy-overview.vue';
  import QuicklyDeploy from './quickly-deploy.vue';
  import RemoveDeploy from './remove-deploy.vue';
  import { DeployableAppType, useDeployAPIs } from './use-deploy';
  import { useEnvVarPrecheck } from './use-env-var-precheck';

  import type { DeployOverviewDeployTarget } from './overview/use-deploy-overview';
  import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
  import type { TabItem } from '~/components/tab-header.vue';
  import type {
    BuildAlertTheme,
    BuildInfo,
    BuildStatus,
  } from '~/pages/application/detail/components/view-build-log/type';

  interface DeletedFeatureEnvPayload {
    envName: string;
    sourceEnvName?: string;
  }

  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const { getAppDeployStatusInfo } = useDeployStatusMap();
  const { t } = useI18n();
  const { cancelDeploy, continueDeploy, isShowPrecheckDialog, precheck, precheckEnvName, undefinedVars } =
    useEnvVarPrecheck();

  // 环境列表（从 EnvSelect 组件 emit 获取）
  const envSelectPanelRef = ref<null | { refreshDeployStatuses?: () => Promise<void> }>(null);
  const envList = ref<EnvOutput[]>([]);
  const envSelectRefreshKey = ref(0);
  const envListLoading = ref(true);
  const deployStatusList = ref<AppDeployedEnvOutputObj[]>([]);
  const isEnvListLoading = computed(() => !appDetailStore.appID || appDetailStore.loading || envListLoading.value);

  // 构建日志侧滑：点击构建状态「查看日志」时打开
  const showBuildLog = ref(false);

  // 是否有可用环境（有环境且至少一个不是 NotReady）
  const hasAvailableEnv = computed(
    () => envList.value.length > 0 && envList.value.some(env => env.status !== 'NotReady'),
  );

  // 该规格只服务实例页的部署/扩缩容操作；总览使用聚合接口中的资源数据，不应触发此请求。
  const effectiveDeploySpec = ref<AppSpecResourcesOutput>();
  /** 获取当前单环境生效的副本规格，并丢弃环境或 Tab 切换后的过期响应。 */
  async function fetchEffectiveDeploySpec() {
    const appID = appDetailStore.appID;
    const envName = curEnv.value;
    const curEnvItemName = trpcDeployStore.curEnvItem?.name;
    if (activeTab.value === TAB_NAMES.overview) return;
    if (!appID || !envName || trpcDeployStore.curEnvItem?.status === 'NotReady') return;
    if (envName !== curEnvItemName) return;
    // 非 trpc/taf 类型应用不请求该接口
    if (isHelmLikeAppType(appDetailStore.appType)) return;
    const res = await AppSpecService.getEnvEffectiveAppSpecResources(
      {
        appID,
        envName,
      },
      { interceptorErr: false },
    ).catch(() => null);
    // 请求返回前应用/环境可能已切换或被销毁，过期结果不能覆盖新状态。
    if (
      activeTab.value === TAB_NAMES.overview ||
      appID !== appDetailStore.appID ||
      envName !== curEnv.value ||
      envName !== trpcDeployStore.curEnvItem?.name
    ) {
      return;
    }
    if (res) {
      effectiveDeploySpec.value = res;
    }
  }

  const envStore = useDeployEnvStore();
  const curEnv = computed({
    get: () => envStore.currentEnv,
    set: val => envStore.updateCurrentEnv(val),
  });

  /** 读取字符串路由参数；数组参数取第一项，缺失时返回空字符串。 */
  function getRouteParam(name: string) {
    const value = route.params[name];
    return Array.isArray(value) ? value[0] : value || '';
  }

  // 环境缓存 key
  const envSelectionScopeKey = computed(() => {
    const space = getRouteParam('space');
    const appName = getRouteParam('name');
    if (!space || !appName) return '';
    return `${space}:${appName}`;
  });

  const initialEnvSelection = computed(() =>
    envSelectionScopeKey.value ? envStore.getAppEnvSelection(envSelectionScopeKey.value) : undefined,
  );

  const routeEnvName = computed(() => {
    const envName = route.query.envName;
    return Array.isArray(envName) ? envName[0] || '' : envName || '';
  });

  /** 环境选择恢复完成后移除一次性 envName query，避免它持续覆盖用户后续选择。 */
  function clearRouteEnvName() {
    if (!('envName' in route.query)) return;
    const { envName: _envName, ...query } = route.query;
    router.replace({ query });
  }

  const curEnvs = ref<string[]>([...(initialEnvSelection.value?.selectedEnvs || [])]);
  const requestableMultiEnvNames = computed(() => {
    const deployStatusEnvNames = new Set(
      deployStatusList.value.map(item => item.name).filter((name): name is string => !!name),
    );
    return curEnvs.value.filter(envName => deployStatusEnvNames.has(envName));
  });
  const spaceStore = useSpaceStore();
  const trpcDeployStore = useTrpcDeployStore();
  const isDeployStatusVisible = computed(() => {
    if (!latestDeployStatus.value) return false;
    const { stage, status } = latestDeployStatus.value;
    if (stage === 'build') return true;
    // deploy 阶段：未卸载时展示
    return status !== APP_DEPLOY_STATUS.UNINSTALLED;
  });
  const isProdEnv = computed(() => trpcDeployStore.curEnvItem?.type === 'production');
  const canFeatureDeploy = computed(() => isAppModelAppType(appDetailStore.appType));
  const canManageFeatureEnvs = computed(() => isAppModelAppType(appDetailStore.appType));

  const initLoading = ref(true);
  const precheckLoading = ref(false);
  const isShowQuicklyDeploy = ref(false);
  const isShowFeatureEnvSideslider = ref(false);
  const isShowRemoveDeploy = ref(false);
  const isMultiEnvMode = ref(initialEnvSelection.value?.mode === 'multi');

  const TAB_NAMES = {
    overview: 'overview',
    instance: 'instance',
    topo: 'topo',
    event: 'event',
    history: 'history',
  } as const;
  const SINGLE_ENV_TABS: string[] = [TAB_NAMES.topo, TAB_NAMES.event, TAB_NAMES.history];

  // Tab 与 URL query（activeTab）双向同步锚定；多环境模式不限制 Tab 切换。
  // 环境选择只保留在页面状态中，避免旧 env query 持续覆盖用户后续选择。
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: Object.values(TAB_NAMES),
        // 缓存为多环境时首次仍进入实例列表，但不覆盖 URL 或用户后续点击的 Tab。
        default: initialEnvSelection.value?.mode === 'multi' ? TAB_NAMES.instance : TAB_NAMES.overview,
      },
    },
  });
  const activeTab = fields.activeTab;

  // 兼容旧链接：部署环境属于页面状态，不在 URL 中保留。
  watch(
    () => route.query.env,
    env => {
      if (env === undefined) return;
      const { env: _env, ...query } = route.query;
      router.replace({ query });
    },
    { immediate: true },
  );

  const tabList = computed<TabItem[]>(() => [
    { label: t('部署总览'), name: 'overview' },
    { label: t('实例列表'), name: 'instance' },
    { label: t('资源拓扑'), name: 'topo' },
    { label: t('事件'), name: 'event' },
    { label: t('部署历史'), name: 'history' },
  ]);

  /** 仅部署总览、实例列表支持多环境；其余 Tab 关闭多选入口 */
  const isEnvMultiSelectable = computed(() => !SINGLE_ENV_TABS.includes(activeTab.value));

  /** 这些 Tab 只需要环境选择器，选择器下沉到页面自己的筛选行 */
  const INLINE_ENV_SELECT_TABS = ['event', 'history'];
  const isTopBarVisible = computed(
    () => activeTab.value !== 'overview' && !INLINE_ENV_SELECT_TABS.includes(activeTab.value),
  );
  const isEnvSelectInline = ref(false);

  // 切 Tab 时投放点会随旧页面一起销毁，先把选择器收回顶部栏，等新页面挂载完再投放过去。
  watch(
    [activeTab, hasAvailableEnv, isEnvListLoading],
    async () => {
      isEnvSelectInline.value = false;
      await nextTick();
      isEnvSelectInline.value =
        (hasAvailableEnv.value || isEnvListLoading.value) && INLINE_ENV_SELECT_TABS.includes(activeTab.value);
    },
    { immediate: true },
  );

  const isRestoringEnvSelection = ref(false);
  const envSelectMode = computed(() => (isMultiEnvMode.value ? 'multi' : 'single'));

  /** 按应用恢复缓存的单/多环境选择，并让一次性定位参数拥有最高优先级。 */
  function restoreAppEnvSelection() {
    if (!envSelectionScopeKey.value) {
      nextTick(clearRouteEnvName);
      return;
    }
    isRestoringEnvSelection.value = true;
    const selection = envStore.getAppEnvSelection(envSelectionScopeKey.value);
    const targetName = routeEnvName.value;
    const selectedEnvs = targetName ? [targetName] : [...(selection?.selectedEnvs || [])];
    curEnvs.value = selectedEnvs;
    isMultiEnvMode.value = targetName ? false : selection?.mode === 'multi';
    if (!isMultiEnvMode.value) {
      envStore.updateCurrentEnv(selectedEnvs[0] || '');
    }
    envStore.updateSelectedEnvs(curEnvs.value);
    if (targetName) {
      // 先将路由指定环境写入缓存，再清理 URL；新标签页首次加载和后续手动选择均可正确恢复。
      envStore.updateAppEnvSelection(envSelectionScopeKey.value, {
        mode: 'single',
        selectedEnvs,
      });
    }
    nextTick(() => {
      isRestoringEnvSelection.value = false;
      clearRouteEnvName();
    });
  }

  // 同步环境列表到 store
  watch(
    envList,
    list => {
      envStore.updateEnvList(list);
    },
    { immediate: true },
  );

  // 同步多选环境到 store
  watch(
    [envSelectionScopeKey, routeEnvName],
    () => {
      restoreAppEnvSelection();
    },
    { immediate: true },
  );

  watch(
    [curEnvs, isMultiEnvMode],
    ([envs, isMulti]) => {
      envStore.updateSelectedEnvs(envs);
      if (isRestoringEnvSelection.value || !envSelectionScopeKey.value) return;
      envStore.updateAppEnvSelection(envSelectionScopeKey.value, {
        mode: isMulti ? 'multi' : 'single',
        selectedEnvs: envs,
      });
    },
    { deep: true },
  );

  /** 部署状态 */
  const curDeployStatus = computed(
    ():
      | (DeployStatusInfo & {
          isFailed: boolean;
          isRunning?: boolean;
          message?: string;
        })
      | null => {
      let latest = latestDeployStatus.value;
      if (!latest) return null;

      let statusInfo: DeployStatusInfo;
      let isFailed = false;
      let isRunning = false;
      if (latest.stage === 'build') {
        // 构建失败类状态映射为"部署失败"，其余（running/success）映射为"部署中"
        // success 时 stage 会从 build → deploy，但轮询可能尚未更新 stage，仍展示"部署中"
        const isBuildFailed =
          (BUILD_INTERRUPT_STATUSES as readonly string[]).includes(latest.status!) ||
          latest.status === APP_BUILD_STATUS.FAILED;
        // 构建失败或中断都是部署失败
        statusInfo = isBuildFailed
          ? getAppDeployStatusInfo(APP_DEPLOY_STATUS.FAILED)
          : getAppDeployStatusInfo(APP_DEPLOY_STATUS.DEPLOYING);
        isFailed = isBuildFailed;
        isRunning = !isBuildFailed;
      } else {
        statusInfo = getAppDeployStatusInfo(latest.status || '');
        isFailed = (DEPLOY_FAILED_STATUSES as readonly string[]).includes(latest.status!);
        isRunning = latest.status === APP_DEPLOY_STATUS.DEPLOYING;
      }

      return {
        ...statusInfo,
        message: latest.message || undefined,
        isFailed,
        isRunning,
      };
    },
  );

  const latestDeployStatus = ref<LatestDeployStatus | null>(null);

  /** 构建状态 Alert 信息 */
  interface BuildAlertInfo {
    closable: boolean;
    status: BuildStatus;
    statusText: string;
    theme: BuildAlertTheme;
  }

  const buildAlertInfo = computed<BuildAlertInfo | null>(() => {
    const latest = latestDeployStatus.value;
    if (!latest?.isBuildAutoDeploy) return null;

    if (latest.stage === 'build') {
      const status = latest.status!;
      if (status === APP_BUILD_STATUS.RUNNING) {
        return { status: 'running', theme: 'info', closable: false, statusText: t('构建中...') };
      }
      if (status === APP_BUILD_STATUS.SUCCESS) {
        return { status: 'success', theme: 'success', closable: true, statusText: t('构建成功') };
      }
      if ((BUILD_INTERRUPT_STATUSES as readonly string[]).includes(status)) {
        return { status: 'warning', theme: 'warning', closable: true, statusText: t('构建中断') };
      }
      // FAILED / POLLING_TIMEOUT / POLLING_BROKEN
      return { status: 'failed', theme: 'error', closable: true, statusText: t('构建失败') };
    }
    // stage !== 'build' 且 isBuildAutoDeploy 为 true，构建已完成进入部署阶段
    return { status: 'success', theme: 'success', closable: true, statusText: t('构建成功') };
  });

  /** 统一组装构建日志信息，供日志侧滑和全量更新状态同步使用。 */
  const buildLogInfo = computed<BuildInfo>(() => ({
    buildID: latestDeployStatus.value?.buildID || '',
    imageTag: latestDeployStatus.value?.imageTag || '',
    operator: latestDeployStatus.value?.operator || '',
    pipelineID: latestDeployStatus.value?.pipelineID || '',
    revision: latestDeployStatus.value?.branch || '',
    status: buildAlertInfo.value?.status || 'failed',
  }));

  /** 构建状态 key：用于 useAlertVisibility 判断是否展示 */
  const buildStatusKey = computed(() => {
    const latest = latestDeployStatus.value;
    if (!latest?.isBuildAutoDeploy) return undefined;
    if (latest.stage === 'build') return latest.status;
    return APP_BUILD_STATUS.SUCCESS;
  });

  const { isVisible: isBuildAlertVisible } = useAlertVisibility(buildStatusKey, {
    seenKeys: [APP_BUILD_STATUS.RUNNING],
    alwaysShowKeys: [APP_BUILD_STATUS.FAILED, ...BUILD_INTERRUPT_STATUSES],
  });

  /** 获取当前环境最近一次部署状态，供实例页状态提示及轮询刷新使用。 */
  async function handleGetLatestDeployStatus() {
    const appID = appDetailStore.appID;
    const envName = trpcDeployStore.curEnvItem?.name;
    if (activeTab.value === TAB_NAMES.overview) return;
    if (!appID || !envName) return;
    try {
      const prevDeployStatus = latestDeployStatus.value?.status;
      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // 获取部署列表
      const res = await deployAPIs.listLatestDeployRecords!(
        {
          appID,
          envName,
        },
        { interceptorErr: false },
      );
      // 切换应用/环境期间可能存在未完成请求，过期响应直接丢弃。
      if (
        activeTab.value === TAB_NAMES.overview ||
        appID !== appDetailStore.appID ||
        envName !== trpcDeployStore.curEnvItem?.name
      ) {
        return;
      }
      const nextLatestDeployStatus = res as unknown as LatestDeployStatus;
      latestDeployStatus.value = nextLatestDeployStatus;
      if (prevDeployStatus && prevDeployStatus !== nextLatestDeployStatus.status) {
        await envSelectPanelRef.value?.refreshDeployStatuses?.();
      }
    } catch (err) {
      // 旧环境销毁后返回的 404 不应停止或污染新环境轮询。
      if (appID !== appDetailStore.appID || envName !== trpcDeployStore.curEnvItem?.name) return;
      stop();
      console.error(err);
    } finally {
      if (appID === appDetailStore.appID && envName === trpcDeployStore.curEnvItem?.name) {
        initLoading.value = false;
      }
    }
  }
  /** 打开最近一次源码构建对应的日志侧栏。 */
  function handleGotoPipeline() {
    showBuildLog.value = true;
  }

  /** 由于部署/移除部署后端存在延时，因此轮询请求数据
   *  由此更新 curReleaseName 从而刷新实例列表数据
   */
  const { start, stop, timer } = useInterval(handleGetLatestDeployStatus, 5000); // 轮询

  /** 同步单环境选择到页面状态和部署 Store，并触发当前环境数据重新加载。 */
  function handleEnvChange(env?: EnvOutput) {
    curEnvs.value = env?.name ? [env.name] : [];
    trpcDeployStore.updateCurEnvItem(env);
    latestDeployStatus.value = null;
    initLoading.value = true;
  }

  /** 将多选环境名称同步到共享 Store，供多环境实例表格请求数据。 */
  function handleEnvsChange(items: EnvOutput[]) {
    // 同步多选环境到 store
    envStore.updateSelectedEnvs(items.map(item => item?.name ?? ''));
  }

  const deployOverviewRef = ref<InstanceType<typeof DeployOverview>>();
  // undefined 表示从实例列表打开；数组（包括空数组）表示从总览打开并启用目标环境选择器。
  const overviewDeployTargets = ref<DeployOverviewDeployTarget[] | undefined>();

  /** 从总览打开复用的 QuicklyDeploy，并把当前可部署环境作为目标选项传入。 */
  function handleOverviewDeploy(targets: DeployOverviewDeployTarget[]) {
    overviewDeployTargets.value = targets;
    isShowQuicklyDeploy.value = true;
  }

  /** 环境列表可能在侧栏打开后才返回，仅在总览模式下持续更新侧栏选项。 */
  function handleOverviewDeployTargetsUpdate(targets: DeployOverviewDeployTarget[]) {
    if (overviewDeployTargets.value !== undefined) {
      overviewDeployTargets.value = targets;
    }
  }

  /** 部署成功后按入口刷新：总览刷新聚合数据，实例页刷新当前环境部署状态。 */
  async function handleQuickDeploySuccess() {
    if (overviewDeployTargets.value !== undefined) {
      await Promise.all([envSelectPanelRef.value?.refreshDeployStatuses?.(), deployOverviewRef.value?.load()]);
      fetchFeatureEnvList();
      return;
    }
    await handleGetLatestDeployStatus();
  }

  /** 从实例列表打开快速部署，并关闭总览入口专用的目标环境选择模式。 */
  async function handleShowQuicklyDeploy() {
    const envName = trpcDeployStore.curEnvItem?.name;
    if (!envName || precheckLoading.value) return;

    precheckLoading.value = true;
    try {
      const precheckPassed = await precheck(envName);
      if (!precheckPassed) return;

      // 清空总览目标是入口标记，保证实例列表仍读取当前环境，不显示目标环境选择器。
      overviewDeployTargets.value = undefined;
      isShowQuicklyDeploy.value = true;
    } catch (error) {
      console.warn(error);
    } finally {
      precheckLoading.value = false;
    }
  }

  /** 总览下钻：切到实例列表并把选中环境切成该环境 */
  function handleViewEnvInstances(envName: string) {
    const env = envList.value.find(item => item.name === envName);
    if (!env) return;
    isMultiEnvMode.value = false;
    curEnvs.value = [envName];
    envStore.updateCurrentEnv(envName);
    trpcDeployStore.updateCurEnvItem(env);
    latestDeployStatus.value = null;
    initLoading.value = true;
    // 环境选择由部署 Store 维护，URL 只锚定页签。
    const { env: _env, ...query } = route.query;
    router.replace({
      query: {
        ...query,
        activeTab: TAB_NAMES.instance,
      },
    });
  }

  const showFullUpdateDialog = ref(false);
  /** 打开当前环境的全量更新侧栏。 */
  async function handleShowFullUpdateDialog() {
    const envName = trpcDeployStore.curEnvItem?.name;
    if (!envName || precheckLoading.value) return;

    precheckLoading.value = true;
    try {
      const precheckPassed = await precheck(envName);
      if (precheckPassed) {
        showFullUpdateDialog.value = true;
      }
    } catch (error) {
      console.warn(error);
    } finally {
      precheckLoading.value = false;
    }
  }
  /** 全量更新成功后按构建模式决定是否关闭侧栏；源码构建需保留侧栏展示实时日志。 */
  function handleUpdateDeploySuccess(keepOpen: boolean) {
    if (!keepOpen) {
      showFullUpdateDialog.value = false;
    }
  }

  // 特性部署
  const isShowFeatureDeploy = ref(false);
  const featureEnvList = ref<FeatureEnvOutput[]>([]);
  const featureEnvLoading = ref(false);
  const featureEnvError = ref(false);
  const featureEnvCount = computed(() => featureEnvList.value.length);

  /** 获取应用关联的特性环境，并防止应用切换后的迟到响应污染新应用。 */
  async function fetchFeatureEnvList() {
    const appID = appDetailStore.appID;
    const requestCanManageFeatureEnvs = canManageFeatureEnvs.value;
    if (!requestCanManageFeatureEnvs || !appID) {
      featureEnvList.value = [];
      featureEnvError.value = false;
      featureEnvLoading.value = false;
      return;
    }

    featureEnvLoading.value = true;
    try {
      const list = await EnvService.listFeatureEnvs({
        appID,
        with_deploy_status: true,
      });
      // 应用类型或应用本身切换后，旧列表请求结果不再写入当前页面。
      if (appID !== appDetailStore.appID || requestCanManageFeatureEnvs !== canManageFeatureEnvs.value) return;
      featureEnvList.value = list;
      featureEnvError.value = false;
    } catch (err) {
      if (appID !== appDetailStore.appID || requestCanManageFeatureEnvs !== canManageFeatureEnvs.value) return;
      console.error(err);
      featureEnvList.value = [];
      featureEnvError.value = true;
    } finally {
      if (appID === appDetailStore.appID && requestCanManageFeatureEnvs === canManageFeatureEnvs.value) {
        featureEnvLoading.value = false;
      }
    }
  }

  /** 特性环境删除后选择回退环境：优先来源环境，其次任一可用标准环境。 */
  function getFeatureEnvDeleteFallbackEnv(payload: DeletedFeatureEnvPayload) {
    // 销毁当前特性环境后优先切回来源环境；来源不可用时再退到其它可用环境。
    const sourceEnv = payload.sourceEnvName
      ? envList.value.find(env => env.name === payload.sourceEnvName && env.status !== 'NotReady')
      : undefined;
    if (sourceEnv) return sourceEnv;

    return envList.value.find(env => env.name !== payload.envName && env.status !== 'NotReady');
  }

  /** 特性部署成功后切换到新环境的实例列表，并刷新总览及特性环境数量。 */
  function handleFeatureDeploySuccess(env?: EnvOutput) {
    if (env?.name) {
      envSelectRefreshKey.value += 1;
      envStore.updateCurrentEnv(env.name);
      trpcDeployStore.updateCurEnvItem(env);
      activeTab.value = 'instance';
    }
    deployOverviewRef.value?.load();
    fetchFeatureEnvList();
  }

  /** 删除特性环境后修正单/多选环境状态，避免后续继续请求已销毁环境。 */
  function handleFeatureEnvDeleted(payload: DeletedFeatureEnvPayload) {
    const fallbackEnv = getFeatureEnvDeleteFallbackEnv(payload);
    // 多环境选择中移除已销毁环境，并补入兜底环境，避免实例面板继续请求已删除环境。
    if (curEnvs.value.includes(payload.envName)) {
      const nextEnvSet = new Set(curEnvs.value.filter(envName => envName !== payload.envName));
      if (fallbackEnv?.name) {
        nextEnvSet.add(fallbackEnv.name);
      }
      curEnvs.value = Array.from(nextEnvSet);
      envStore.updateSelectedEnvs(curEnvs.value);
    }

    const isCurrentEnvDeleted =
      curEnv.value === payload.envName || trpcDeployStore.curEnvItem?.name === payload.envName;
    if (isCurrentEnvDeleted) {
      // 当前环境已被销毁，先停掉旧环境轮询，再切换到来源/兜底环境触发新环境加载。
      stop();
      latestDeployStatus.value = null;
      effectiveDeploySpec.value = undefined;
      activeTab.value = 'instance';

      if (fallbackEnv?.name) {
        initLoading.value = true;
        envStore.updateCurrentEnv(fallbackEnv.name);
        trpcDeployStore.updateCurEnvItem(fallbackEnv);
      } else {
        initLoading.value = false;
        envStore.updateCurrentEnv('');
        trpcDeployStore.updateCurEnvItem(undefined);
      }
    }

    refreshFeatureEnvData();
  }

  /** 权限与应用类型允许时打开复用的特性部署侧栏。 */
  function handleShowFeatureDeploy() {
    if (!canFeatureDeploy.value) return;
    isShowFeatureDeploy.value = true;
  }

  /** 重新创建环境选择器并刷新总览、特性环境列表，确保三处数据一致。 */
  function refreshFeatureEnvData() {
    envSelectRefreshKey.value += 1;
    deployOverviewRef.value?.load();
    fetchFeatureEnvList();
  }

  // 移除部署
  const morePopoverRef = ref<InstanceType<typeof Popover> | null>(null);
  /** 打开移除部署确认；表格行入口会先将目标环境同步为当前环境。 */
  function handleRemoveDeploy(env?: EnvOutput) {
    morePopoverRef.value?.hide();
    if (env) {
      trpcDeployStore.updateCurEnvItem(env);
    }
    isShowRemoveDeploy.value = true;
  }
  /** 移除部署成功后更新当前状态，并刷新环境状态、总览和轮询。 */
  async function handleRemoveDeploySuccess() {
    latestDeployStatus.value = {
      ...(latestDeployStatus.value || {}),
      hasDeployRecord: false,
      stage: 'deploy',
      status: APP_DEPLOY_STATUS.UNINSTALLED,
    };
    initLoading.value = false;
    await envSelectPanelRef.value?.refreshDeployStatuses?.();
    deployOverviewRef.value?.load();
    fetchFeatureEnvList();
    stop();
    start();
  }

  // 切换空间停止轮询并重置状态
  // 实例相关请求统一由“应用 + 当前环境 + Tab”驱动；总览不依赖单环境数据，进入后立即停止轮询。
  watch(
    () => spaceStore.currentSpace,
    (newSpace, oldSpace) => {
      if (oldSpace && newSpace !== oldSpace) {
        stop();
        // 清空当前环境项，避免跨工作空间请求
        trpcDeployStore.updateCurEnvItem(undefined);
      }
    },
  );

  watch(
    [() => appDetailStore.appID, () => trpcDeployStore.curEnvItem?.name, activeTab],
    async () => {
      latestDeployStatus.value = null;
      if (activeTab.value === TAB_NAMES.overview) {
        effectiveDeploySpec.value = undefined;
        initLoading.value = false;
        stop();
        return;
      }
      await handleGetLatestDeployStatus();
      await fetchEffectiveDeploySpec();
      if (activeTab.value === TAB_NAMES.overview) return;
      // 当前环境被销毁并清空时，不重新开启一个只会空转的轮询。
      if (appDetailStore.appID && trpcDeployStore.curEnvItem?.name && !timer.value) {
        start();
      }
    },
    { immediate: true },
  );

  watch([() => appDetailStore.appID, () => appDetailStore.appType], fetchFeatureEnvList, { immediate: true });

  watch(isShowFeatureEnvSideslider, show => {
    if (show) {
      // 侧栏每次打开都重新拉取带部署状态的列表，避免使用上一次打开时的缓存状态。
      fetchFeatureEnvList();
    }
  });

  onBeforeUnmount(() => {
    trpcDeployStore.updateCurEnvItem(undefined);
    stop();
  });
</script>
