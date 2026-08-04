<template>
  <div class="flex flex-col h-full">
    <!-- 自定义 Header -->
    <div class="h-[52px] flex items-center justify-between px-[24px] bg-[#FFF] shadow-[0_3px_4px_0_#0000000a]">
      <div class="text-[16px] text-[#313238]">{{ $t('部署管理') }}</div>
      <Button
        v-if="canManageFeatureEnvs"
        class="feature-env-entry"
        text
        theme="primary"
        @click="isShowFeatureEnvSideslider = true"
      >
        <i18n-t keypath="应用关联的特性环境：{0} 个">
          <span class="font-bold">{{ featureEnvCount || '--' }}&nbsp;</span>
        </i18n-t>
      </Button>
    </div>
    <div class="flex-1 min-h-0 px-[24px] py-[20px] flex flex-col">
      <FlexRow class="bg-[#EAEBF0] shadow-[0_2px_4px_0_#0000001a] px-[12px] py-[8px] mb-[16px]">
        <template #left>
          <div class="flex">
            <EnvSelectPanel
              :key="envSelectRefreshKey"
              ref="envSelectPanelRef"
              v-model="curEnv"
              v-model:model-values="curEnvs"
              class="mr-[16px]"
              init-first-env-when-empty
              :mode="envSelectMode"
              multi-selectable
              preserve-missing-model-value
              @update:deploy-status-list="deployStatusList = $event"
              @update:env-list="envList = $event"
              @update:item="handleEnvChange"
              @update:items="handleEnvsChange"
              @update:loading="envListLoading = $event"
              @update:mode="isMultiEnvMode = $event === 'multi'"
            />
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
              />
              <span class="leading-[20px] text-[#4D4F56]">{{ curDeployStatus?.text }}</span>
              <i
                v-if="curDeployStatus?.isFailed && curDeployStatus?.message"
                v-bk-tooltips="{
                  content: curDeployStatus?.message,
                  theme: 'light',
                }"
                class="bkms-icon bkms-icon-circle-info text-[16px] text-[#C4C6CC] ml-[4px] cursor-pointer"
              >
              </i>
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
        v-if="!hasAvailableEnv && !isEnvListLoading"
        class="large-exception"
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
      <Tab
        v-else
        v-model:active="activeTab"
        class="flex-1 overflow-hidden"
        :label-height="40"
        type="unborder-card"
      >
        <Tab.TabPanel
          name="instance"
          render-directive="if"
        >
          <template #label>
            {{ $t('实例列表') }}
          </template>
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
                :show-feature-deploy="canFeatureDeploy"
                @deploy="isShowQuicklyDeploy = true"
                @feature-deploy="handleShowFeatureDeploy"
              />
            </Exception>
          </Skeleton>
        </Tab.TabPanel>
        <Tab.TabPanel
          v-if="!isMultiEnvMode"
          name="topo"
          render-directive="if"
        >
          <template #label>
            {{ $t('资源拓扑') }}
          </template>
          <ResourceTopology :env-name="curEnv" />
        </Tab.TabPanel>
        <Tab.TabPanel
          v-if="!isMultiEnvMode"
          name="event"
          render-directive="if"
        >
          <template #label>
            {{ $t('事件') }}
          </template>
          <DeployEvent />
        </Tab.TabPanel>
        <Tab.TabPanel
          v-if="!isMultiEnvMode"
          name="history"
          render-directive="if"
        >
          <template #label>
            {{ $t('部署历史') }}
          </template>
          <DeployHistory />
        </Tab.TabPanel>
      </Tab>
    </div>
    <!-- 立即部署 -->
    <QuicklyDeploy
      v-model:is-show="isShowQuicklyDeploy"
      :effective-replicas="effectiveDeploySpec?.replicas"
      :is-prod-env="isProdEnv"
      @update="handleGetLatestDeployStatus"
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

  import { Alert, Button, Exception, Popover, Tab } from 'bkui-vue';
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
  import { isAppModelAppType, isHelmLikeAppType } from '~/composables/app-type';
  import { useAlertVisibility } from '~/composables/use-alert-visibility';
  import { type DeployStatusInfo, useDeployStatusMap } from '~/composables/use-deploy-status';
  import useInterval from '~/composables/use-interval';
  import ViewBuildLog from '~/pages/application/detail/components/view-build-log/index.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';
  import { useSpaceStore } from '~/stores/space';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import KeyValueBadge from '../../components/key-value-badge.vue';
  import ResourceTopology from '../../components/topo/index.vue';
  import DeployActionButton from './deploy-action-button.vue';
  import DeployEvent from './deploy-event.vue';
  import DeployHistory from './deploy-history.vue';
  import FeatureDeploy from './feature-deploy.vue';
  import FeatureEnvSideslider from './feature-env-sideslider.vue';
  import FullUpdate from './instance-list/full-update.vue';
  import InstanceList from './instance-list/instance-list.vue';
  import MultiEnvInstanceTable from './instance-list/multi-env-instance-table.vue';
  import ScaleInstance from './instance-list/scale-instance.vue';
  import QuicklyDeploy from './quickly-deploy.vue';
  import RemoveDeploy from './remove-deploy.vue';
  import { DeployableAppType, useDeployAPIs } from './use-deploy';

  import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
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

  // 环境生效的部署规格
  const effectiveDeploySpec = ref<AppSpecResourcesOutput>();
  async function fetchEffectiveDeploySpec() {
    const appID = appDetailStore.appID;
    const envName = curEnv.value;
    const curEnvItemName = trpcDeployStore.curEnvItem?.name;
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
    if (appID !== appDetailStore.appID || envName !== curEnv.value || envName !== trpcDeployStore.curEnvItem?.name) {
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

  // 获取路由参数
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

  const targetEnvName = computed(() => {
    const envName = route.query.envName;
    return Array.isArray(envName) ? envName[0] || '' : envName || '';
  });

  /**
   * envName 仅用于跨页面跳转时的首次环境定位。
   * 定位完成后移除该参数，避免它在后续返回页面时持续覆盖用户的环境选择。
   */
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
  const isShowQuicklyDeploy = ref(false);
  const isShowFeatureEnvSideslider = ref(false);
  const isShowRemoveDeploy = ref(false);
  const activeTab = ref<string>('instance');
  const isMultiEnvMode = ref(initialEnvSelection.value?.mode === 'multi');
  const isRestoringEnvSelection = ref(false);
  const envSelectMode = computed(() => (isMultiEnvMode.value ? 'multi' : 'single'));

  function restoreAppEnvSelection() {
    if (!envSelectionScopeKey.value) return;
    isRestoringEnvSelection.value = true;
    const selection = envStore.getAppEnvSelection(envSelectionScopeKey.value);
    const routeEnvName = targetEnvName.value;
    const selectedEnvs = routeEnvName ? [routeEnvName] : [...(selection?.selectedEnvs || [])];
    curEnvs.value = selectedEnvs;
    isMultiEnvMode.value = routeEnvName ? false : selection?.mode === 'multi';
    if (!isMultiEnvMode.value) {
      envStore.updateCurrentEnv(selectedEnvs[0] || '');
    }
    envStore.updateSelectedEnvs(curEnvs.value);
    if (routeEnvName) {
      // 先将路由指定环境写入缓存，再清理 URL；新标签页首次加载和后续手动选择均可正确恢复。
      envStore.updateAppEnvSelection(envSelectionScopeKey.value, {
        mode: 'single',
        selectedEnvs,
      });
    }
    nextTick(() => {
      isRestoringEnvSelection.value = false;
      if (routeEnvName) {
        clearRouteEnvName();
      }
    });
  }

  watch(isMultiEnvMode, val => {
    if (val && activeTab.value !== 'instance') {
      activeTab.value = 'instance';
    }
  });

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
    [envSelectionScopeKey, targetEnvName],
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

  async function handleGetLatestDeployStatus() {
    const appID = appDetailStore.appID;
    const envName = trpcDeployStore.curEnvItem?.name;
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
      if (appID !== appDetailStore.appID || envName !== trpcDeployStore.curEnvItem?.name) return;
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
  // 打开构建日志侧滑
  function handleGotoPipeline() {
    showBuildLog.value = true;
  }

  /** 由于部署/移除部署后端存在延时，因此轮询请求数据
   *  由此更新 curReleaseName 从而刷新实例列表数据
   */
  const { start, stop, timer } = useInterval(handleGetLatestDeployStatus, 5000); // 轮询

  /** 需要环境select有值后才能获取到数据 */
  function handleEnvChange(env?: EnvOutput) {
    curEnvs.value = env?.name ? [env.name] : [];
    trpcDeployStore.updateCurEnvItem(env);
    latestDeployStatus.value = null;
    initLoading.value = true;
  }

  /** 多选环境变化 */
  function handleEnvsChange(items: EnvOutput[]) {
    // 同步多选环境到 store
    envStore.updateSelectedEnvs(items.map(item => item?.name ?? ''));
  }

  // 全量更新
  const showFullUpdateDialog = ref(false);
  function handleShowFullUpdateDialog() {
    showFullUpdateDialog.value = true;
  }
  // 全量更新成功；源码构建场景需要保留侧滑以继续展示实时日志。
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

  // 获取特性环境列表
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

  function getFeatureEnvDeleteFallbackEnv(payload: DeletedFeatureEnvPayload) {
    // 销毁当前特性环境后优先切回来源环境；来源不可用时再退到其它可用环境。
    const sourceEnv = payload.sourceEnvName
      ? envList.value.find(env => env.name === payload.sourceEnvName && env.status !== 'NotReady')
      : undefined;
    if (sourceEnv) return sourceEnv;

    return envList.value.find(env => env.name !== payload.envName && env.status !== 'NotReady');
  }

  function handleFeatureDeploySuccess(env?: EnvOutput) {
    if (env?.name) {
      envSelectRefreshKey.value += 1;
      envStore.updateCurrentEnv(env.name);
      trpcDeployStore.updateCurEnvItem(env);
      activeTab.value = 'instance';
    }
    fetchFeatureEnvList();
  }

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

  function handleShowFeatureDeploy() {
    if (!canFeatureDeploy.value) return;
    isShowFeatureDeploy.value = true;
  }

  function refreshFeatureEnvData() {
    envSelectRefreshKey.value += 1;
    fetchFeatureEnvList();
  }

  // 移除部署
  const morePopoverRef = ref<InstanceType<typeof Popover> | null>(null);
  function handleRemoveDeploy(env?: EnvOutput) {
    morePopoverRef.value?.hide();
    if (env) {
      trpcDeployStore.updateCurEnvItem(env);
    }
    isShowRemoveDeploy.value = true;
  }
  async function handleRemoveDeploySuccess() {
    latestDeployStatus.value = {
      ...(latestDeployStatus.value || {}),
      hasDeployRecord: false,
      stage: 'deploy',
      status: APP_DEPLOY_STATUS.UNINSTALLED,
    };
    initLoading.value = false;
    await envSelectPanelRef.value?.refreshDeployStatuses?.();
    fetchFeatureEnvList();
    stop();
    start();
  }

  // 切换空间停止轮询并重置状态
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
    [() => appDetailStore.appID, () => trpcDeployStore.curEnvItem?.name],
    async () => {
      latestDeployStatus.value = null;
      await handleGetLatestDeployStatus();
      await fetchEffectiveDeploySpec();
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

<style lang="postcss" scoped>
  :deep(.bk-tab-header) {
    background-color: #fff;
    padding: 0 24px;
  }
  :deep(.bk-tab-header-nav .bk-tab-header-item) {
    padding: 0 !important;
    margin-right: 32px !important;
    font-size: 14px;
  }
  :deep(.bk-tab-content) {
    background-color: #fff;
    height: 100%;
    min-height: 0;
    overflow: auto;
    &:has(.custom-resource-topology) {
      padding: 0 !important;
    }
  }
</style>
