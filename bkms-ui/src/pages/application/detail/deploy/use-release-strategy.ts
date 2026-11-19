/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { computed, onMounted, ref, toValue, watch } from 'vue';
import type { MaybeRefOrGetter } from 'vue';

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
import { RELEASE_MODE } from '~/common/enums/deploy';
import { useAppDetail } from '~/stores/app-detail';
import { useSpaceStore } from '~/stores/space';

import { type ReleasePreviewResult, buildReleasePreview, buildStrategyHint } from './release-preview';
import {
  type CanaryBlockReason,
  type ReleaseMode,
  type ReleaseStrategyOption,
  buildStrategyOptions,
  isFirstDeployFromLatestStatus,
  resolveCanaryBlockReason,
} from './release-strategy';
import { type DeployableAppType, useDeployAPIs } from './use-deploy';

import type { AppModelDeployRecordOutputObj } from '~/@types/v1/deploy';
import type { ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

/** 分批发布准入被拒原因对应的文案 */
const CANARY_BLOCK_MESSAGES: Record<Exclude<CanaryBlockReason, ''>, string> = {
  notAppModelApp: '仅应用模型应用支持分批发布',
  federationEnv: '联邦集群环境暂不支持分批发布',
  checkingWorkloadKind: '正在检查分批发布资格，请稍候',
  firstDeploy: '首次部署暂不支持分批发布',
  deploymentWorkload: '仅主工作负载为 GameDeployment 的应用支持分批发布',
  inplaceRestart: '原地重启暂不支持分批发布',
  buildFromSource: '构建部署暂不支持分批发布',
};

export interface ReleaseStrategyModel {
  releaseMode: ReleaseMode;
  releaseStrategyID: string;
  /** 当前生效实例数：发布预览按它换算批次时间线 */
  replicas?: number;
}

export interface UseReleaseStrategyOptions {
  /** 目标环境名；为空时跳过主工作负载资格检查 */
  envName: MaybeRefOrGetter<string>;
  /** 是否为应用模型应用（tRPC / TAF） */
  isAppModelApp: MaybeRefOrGetter<boolean>;
  /** 本次是否走「从源码构建」部署 */
  isBuildFromSource: MaybeRefOrGetter<boolean>;
  isFederationEnv: MaybeRefOrGetter<boolean>;
  /** 本次是否走「仅镜像（原地重启）」；不传视为否 */
  isInplaceRestart?: MaybeRefOrGetter<boolean>;
  /** 表单模型：直接读写 releaseMode / releaseStrategyID / replicas */
  model: ReleaseStrategyModel;
  /** 目标泳道名；不传表示不使用泳道 */
  trafficLaneName?: MaybeRefOrGetter<string | undefined>;
}

/**
 * 发布方式 / 发布策略的共享状态与行为。
 * 实例列表「全量更新」与「快速部署」两个入口的准入规则、策略列表、发布预览都从这里取，
 * 避免分批发布资格这类会持续演进的业务规则在两侧各维护一份。
 */
export function useReleaseStrategy(options: UseReleaseStrategyOptions) {
  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const appDetailStore = useAppDetail();
  const { model } = options;

  /** 主工作负载类型（GameDeployment / Deployment），来自最近一次部署记录；空串表示未知 */
  const mainWorkloadKind = ref('');
  /** 主工作负载资格检查是否进行中：检查完成前禁用分批发布，避免选择后再被回退 */
  const workloadKindLoading = ref(false);
  /** 环境切换时丢弃上一轮的主工作负载检查结果 */
  let workloadKindToken = 0;
  /** 目标环境 / 泳道是否首次部署：首次部署没有存量实例可灰度，不允许选择分批发布 */
  const isFirstDeploy = ref(false);
  /** 首次部署检查是否进行中：与主工作负载检查共用「资格检查中」禁用态 */
  const deployStatusLoading = ref(false);
  /** 环境切换时丢弃上一轮的首次部署检查结果 */
  let deployStatusToken = 0;

  // 工作空间发布策略列表（含平台内置策略，isBuiltin 标识）
  const releaseStrategies = ref<ReleaseStrategyOutputObj[]>([]);
  const strategyLoading = ref(false);

  const canaryBlockReason = computed(() =>
    resolveCanaryBlockReason({
      isAppModelApp: Boolean(toValue(options.isAppModelApp)),
      isFederationEnv: Boolean(toValue(options.isFederationEnv)),
      isInplaceRestart: Boolean(toValue(options.isInplaceRestart ?? false)),
      isBuildFromSource: Boolean(toValue(options.isBuildFromSource)),
      isFirstDeploy: isFirstDeploy.value,
      mainWorkloadKind: mainWorkloadKind.value,
      // 两个资格检查任一未完成都视为「检查中」，避免先选分批再被回退
      workloadKindLoading: workloadKindLoading.value || deployStatusLoading.value,
    }),
  );

  /** 分批发布不可用的原因；为空表示可用 */
  const canaryDisabledReason = computed(() => {
    const reason = canaryBlockReason.value;
    return reason ? t(CANARY_BLOCK_MESSAGES[reason]) : '';
  });

  const strategyOptions = computed<ReleaseStrategyOption[]>(() =>
    buildStrategyOptions(releaseStrategies.value, { full: t('全量'), unit: t('个') }),
  );

  /** 空间内已有策略名称：新建抽屉用于名称唯一性校验 */
  const existingStrategyNames = computed(() => releaseStrategies.value.map(item => item.name ?? ''));

  const selectedStrategy = computed(() => releaseStrategies.value.find(item => item.id === model.releaseStrategyID));

  /** 发布预览：选中策略后按实例数实时换算批次时间线与摘要 */
  const releasePreview = computed<null | ReleasePreviewResult>(() => {
    if (model.releaseMode !== RELEASE_MODE.CANARY || !selectedStrategy.value) return null;
    return buildReleasePreview(selectedStrategy.value.batches, Number(model.replicas));
  });

  /**
   * 策略说明文案：先 1 个实例，确认后累计到 25%、50%，最后自动全量。
   * 末批非全量时后端仍会自动更新剩余实例（UpdatingPrimary 阶段），因此收尾固定为「最后自动全量」；
   * 预览时间线不再补齐该全量批，改为底部「剩余未更新的实例将自动更新」提示。
   */
  const strategyHintText = computed(() => {
    const hint = buildStrategyHint(selectedStrategy.value?.batches);
    if (!hint) return '';
    const parts: string[] = [];
    // 仅「真的只有一个批次且为全量」才只提示「最后自动全量」，避免「先更新到 100%」的重复表述；
    // 两批策略（如 25% → 100%）不能用中间批次数判断，否则首批灰度阶段会被漏掉。
    const isSingleFull = hint.batchCount === 1 && hint.lastIsFull;
    if (!isSingleFull) {
      parts.push(hint.firstIsPercent ? t('先更新到 {0}', [hint.firstTarget]) : t('先 {0} 个实例', [hint.firstTarget]));
    }
    if (hint.middleTargets.length > 0) {
      parts.push(t('确认后累计到 {0}', [hint.middleTargets.join('、')]));
    }
    parts.push(t('最后自动全量'));
    return parts.join('，');
  });

  /** 拉取最近一条部署记录以获取主工作负载类型；仅应用模型应用且环境已知时执行 */
  async function fetchMainWorkloadKind() {
    const envName = toValue(options.envName);
    if (!toValue(options.isAppModelApp) || !envName) return;
    const token = ++workloadKindToken;
    workloadKindLoading.value = true;
    try {
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      const res = await deployAPIs
        .listDeployRecords({
          appID: appDetailStore.appID,
          envName,
          page: 1,
          pageSize: 1,
        })
        .catch(() => ({ count: 0, results: [] as AppModelDeployRecordOutputObj[] }));
      // 环境切换后旧请求可能晚到，不能覆盖新环境的主工作负载类型
      if (token !== workloadKindToken) return;
      mainWorkloadKind.value = res?.results?.[0]?.workloadKind ?? '';
    } finally {
      if (token === workloadKindToken) workloadKindLoading.value = false;
    }
  }

  /**
   * 检查目标环境 / 泳道是否首次部署。
   *
   * 用最新部署状态接口判断：无记录（data 为 null）、只有构建记录（hasDeployRecord=false）、
   * 已下线（status=uninstalled）三种情况都按首次部署处理。首次部署没有存量实例可灰度，
   * 分批更新按钮需置灰。
   *
   * 请求失败不拦截：无法确认时交给后端兜底，避免接口抖动把合法入口一起禁用。
   */
  async function fetchDeployStatus() {
    const envName = toValue(options.envName);
    if (!toValue(options.isAppModelApp) || !envName) return;
    const token = ++deployStatusToken;
    deployStatusLoading.value = true;
    try {
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // Helm 等应用类型没有该接口（分批发布本身已由 isAppModelApp 拦截），保持可用交由后端校验
      if (!deployAPIs.listLatestDeployRecords) {
        isFirstDeploy.value = false;
        return;
      }
      const latest = await deployAPIs.listLatestDeployRecords({
        appID: appDetailStore.appID,
        envName,
        trafficLaneName: toValue(options.trafficLaneName) || '',
      });
      if (token !== deployStatusToken) return;
      isFirstDeploy.value = isFirstDeployFromLatestStatus(latest);
    } catch (error) {
      console.error(error);
      if (token === deployStatusToken) isFirstDeploy.value = false;
    } finally {
      if (token === deployStatusToken) deployStatusLoading.value = false;
    }
  }

  /** 拉取工作空间发布策略；两个入口的侧栏都随挂载加载一次 */
  async function refresh() {
    if (!spaceStore.currentSpace) return;
    strategyLoading.value = true;
    try {
      releaseStrategies.value =
        (await ReleaseStrategiesService.listReleaseStrategies({ workspaceID: spaceStore.currentSpace })) ?? [];
    } catch (err) {
      console.error(err);
    } finally {
      strategyLoading.value = false;
    }
  }

  /** 静默回落到「一次更新全部实例」并清空已选策略（切换镜像来源 / 更新内容等必然导致分批不可用） */
  function resetToAllInstances() {
    if (model.releaseMode === RELEASE_MODE.ALL && !model.releaseStrategyID) return;
    model.releaseMode = RELEASE_MODE.ALL;
    model.releaseStrategyID = '';
  }

  /** 切换发布方式；切回「一次更新全部」时清空已选策略 */
  function handleChangeReleaseMode(mode: ReleaseMode) {
    if (mode === RELEASE_MODE.ALL) {
      resetToAllInstances();
      return;
    }
    model.releaseMode = mode;
  }

  /** 新建策略成功：刷新策略列表并选中新策略，发布预览随选中项自动重算 */
  async function handleStrategyCreated(strategy?: ReleaseStrategyOutputObj) {
    await refresh();
    if (!strategy?.id) return;
    model.releaseStrategyID = strategy.id;
  }

  // 资格异步返回导致已选分批模式失效时，重置为一次更新全部并提示，避免带着后端必拒的参数提交。
  watch(canaryDisabledReason, reason => {
    if (!reason || model.releaseMode !== RELEASE_MODE.CANARY) return;
    resetToAllInstances();
    Message({ theme: 'warning', message: t('当前环境不支持分批发布，已重置为一次更新全部实例') });
  });

  /** 两项准入资格检查：主工作负载类型 + 是否首次部署 */
  function fetchReleaseEligibility() {
    fetchMainWorkloadKind();
    fetchDeployStatus();
  }

  // 快速部署侧栏允许在侧栏内切换目标环境，环境变化后必须重跑资格检查，
  // 否则会把上一个环境（如已部署环境）的判定套用到新的首次部署环境上。
  watch(() => toValue(options.envName), fetchReleaseEligibility);

  onMounted(() => {
    refresh();
    fetchReleaseEligibility();
  });

  return {
    canaryDisabledReason,
    existingStrategyNames,
    handleChangeReleaseMode,
    handleStrategyCreated,
    releasePreview,
    refresh,
    resetToAllInstances,
    selectedStrategy,
    strategyHintText,
    strategyLoading,
    strategyOptions,
  };
}
