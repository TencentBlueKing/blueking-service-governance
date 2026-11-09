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

import { computed, h, ref } from 'vue';

import { useSessionStorage } from '@vueuse/core';
import { InfoBox, Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { DeployService } from '~/api/modules/v1';

import {
  type BatchReleaseAction,
  type BatchReleaseBannerAction,
  type BatchReleaseBannerView,
  appendBatchReleaseDismissKey,
  buildBatchReleaseDismissKey,
  hasCanaryDeployRecord,
  isBatchReleaseBannerDismissed,
  isBatchReleaseBusy,
  resolveBatchReleaseBanner,
  resolveBatchReleaseSuccessMessage,
  shouldFetchBatchReleaseProgress,
} from '../batch-release';

import type { BatchReleaseProgressOutputObj, LatestDeployStatus } from '~/@types/v1/deploy';

/**
 * 进度回查的等待上限。
 * 请求挂起（网关无响应、连接被中间件 hold）时必须主动放弃等待，否则 pendingAction 一直非空，
 * 横幅上所有按钮都会被静默拦截，表现为「状态更新后怎么点都没反应」。
 */
const PROGRESS_REFETCH_TIMEOUT = 3000;

/**
 * 单次操作的兜底释放时间。
 * 写请求本身没有超时（项目未配全局 timeout），一旦连接被挂住，pendingAction 永远不回落，
 * 横幅按钮就永久「点不动」，因此用看门狗强制解锁（此时结果未知，仍以服务端进度为准）。
 */
const PENDING_ACTION_WATCHDOG = 30_000;

export interface UseBatchReleaseControlOptions {
  /** 当前应用 ID */
  getAppID: () => string;
  /** 当前环境名称 */
  getEnvName: () => string;
  /** 是否允许展示与请求：单环境 + 应用模型 + 停留在实例 Tab */
  isEnabled: () => boolean;
  /** 操作成功后刷新环境部署状态（可选） */
  refreshDeployStatus?: () => Promise<void> | void;
  /** 操作成功后刷新实例列表 */
  refreshInstances: () => Promise<void> | void;
}

/**
 * 分批发布控制横幅的控制器：负责进度补查、五类操作、二次确认与结果提示。
 *
 * 所有写操作都以服务端为准：操作成功后立刻回查进度并刷新实例列表，
 * 失败时只提示并回查纠偏，不做本地乐观状态机，避免前后端状态分叉。
 */
export function useBatchReleaseControl(options: UseBatchReleaseControlOptions) {
  const { t } = useI18n();
  // 完成态关闭记忆：已关闭的 `${appID}:${envName}:${deployID}` 列表，换应用/环境或新部署时自然失效。
  // 列表而非单值：多个环境各自关闭互不覆盖；放 sessionStorage，重新打开浏览器后横幅重新出现。
  const dismissedKeys = useSessionStorage<string[]>('batch-release-dismissed', []);

  const progress = ref<BatchReleaseProgressOutputObj | null>(null);
  const pendingAction = ref<BatchReleaseAction | null>(null);
  // 进度回查进行中：横幅渲染骨架占位，避免数据返回时横幅突然弹出造成布局跳动。
  const loading = ref(false);
  // 接口已确认「无进度」的部署记录：空结果是稳定态，记住后不再随 latest-status 每 5s 空转。
  const emptyQueriedDeployID = ref('');
  // 请求世代：环境切换或新一轮请求开始后丢弃过期响应。
  let requestToken = 0;

  const rawView = computed<BatchReleaseBannerView | null>(() => resolveBatchReleaseBanner(progress.value));

  /**
   * 骨架占位可见性：仅在「还没有任何进度快照但正在回查」时展示（环境切换 / 首次加载）。
   * 快照仍在（如已关闭的完成态横幅）时回查不打扰页面，否则关闭后每轮轮询都会闪出骨架。
   */
  const showSkeleton = computed(() => loading.value && !rawView.value);

  const dismissKey = computed(() =>
    buildBatchReleaseDismissKey(options.getAppID(), options.getEnvName(), progress.value?.deployID ?? ''),
  );

  /** 部署中展示批次控制横幅，已完成展示成功提示；关闭过的成功提示按 deployID 保持隐藏。 */
  const view = computed<BatchReleaseBannerView | null>(() => {
    if (!options.isEnabled()) return null;
    if (isBatchReleaseBannerDismissed(rawView.value, dismissKey.value, dismissedKeys.value)) return null;
    return rawView.value;
  });

  /** 是否仍在进行中（完成态不再阻塞新的部署入口）。 */
  const isBusy = computed(() => isBatchReleaseBusy(view.value));

  /** 顶部「部署」按钮被禁用时的提示原因。 */
  const busyReason = computed(() => (isBusy.value ? t('分批发布进行中，请先完成、跳过或回滚') : ''));

  /** 清空横幅：环境切换、多环境、非实例 Tab 或最新记录不是分批发布时调用。 */
  function reset() {
    requestToken += 1;
    progress.value = null;
    loading.value = false;
    // 换应用 / 环境时旧的「空进度」记忆失效，否则新环境可能被误判为无需查询。
    emptyQueriedDeployID.value = '';
  }

  /**
   * 应用 / 环境切换时调用：立刻丢弃旧环境的进度快照并进入骨架等待态，
   * 避免旧横幅一直展示到新数据返回时被「直接替换」。
   * 后续 refresh 若确认无分批发布（reset）或回查完成（finally），会自然收掉 loading。
   */
  function resetForSwitch() {
    reset();
    loading.value = true;
  }

  /**
   * 直接回查进度（不做门控），用于操作成功后的即时刷新。
   * @param latest 本轮最新的部署状态，仅用于判断「空进度」是否为稳定结果
   */
  async function fetchProgress(latest?: LatestDeployStatus | null) {
    const appID = options.getAppID();
    const envName = options.getEnvName();
    if (!options.isEnabled() || !appID || !envName) {
      reset();
      return;
    }
    const token = ++requestToken;
    const result = await DeployService.getBatchReleaseProgress({ appID, envName }, { interceptorErr: false }).catch(
      error => {
        // 失败保留上一次快照，不打断页面。
        console.error(error);
        return undefined;
      },
    );
    if (token !== requestToken) return;
    if (appID !== options.getAppID() || envName !== options.getEnvName() || !options.isEnabled()) return;
    if (result === undefined) return;
    progress.value = result ?? null;
    if (result) return;
    // 仅在「后端也不认为有进行中的分批发布」时记忆空结果：刚发起发布、进度尚未落库时
    // inProgressReleaseMode 已是 canary，此时空结果是暂时的，下一轮必须重查。
    const deployID = latest?.deployID ?? '';
    if (deployID && latest?.inProgressReleaseMode !== 'canary') {
      emptyQueriedDeployID.value = deployID;
    }
  }

  /**
   * 跟随最新的部署状态轮询补查进度。
   * 非分批发布记录直接清空；同一 deployID 的终态进度或「已确认为空」的结果命中后不再重复请求。
   */
  async function refresh(latest: LatestDeployStatus | null) {
    if (!options.isEnabled() || !hasCanaryDeployRecord(latest)) {
      reset();
      return;
    }
    if (!shouldFetchBatchReleaseProgress(latest, progress.value, emptyQueriedDeployID.value)) return;
    // 加超时避免进度请求挂起时把上层「最新部署状态」轮询一起卡住。
    // 过期响应不回写 loading：真正的收尾由最新一轮请求或 reset 负责。
    const token = ++requestToken;
    loading.value = true;
    try {
      await withTimeout(fetchProgress(latest), PROGRESS_REFETCH_TIMEOUT);
    } finally {
      if (token === requestToken) loading.value = false;
    }
  }

  /** 后台刷新：不占用操作收尾，失败只记录日志。 */
  function runRefresh(task?: () => Promise<void> | void) {
    if (!task) return;
    try {
      Promise.resolve(task()).catch(error => console.warn(error));
    } catch (error) {
      console.warn(error);
    }
  }

  /** 请求失败时提示并回查纠偏，不改变本地进度快照。 */
  async function operate(action: BatchReleaseAction) {
    const appID = options.getAppID();
    const envName = options.getEnvName();
    if (!appID || !envName || pendingAction.value) return;

    const currentView = view.value;
    pendingAction.value = action;
    const watchdog = setTimeout(() => {
      pendingAction.value = null;
    }, PENDING_ACTION_WATCHDOG);
    try {
      await DeployService.operateBatchRelease({ action, appID, envName }, { interceptorErr: false });
      // 操作期间切换了应用/环境：结果已不属于当前页面，直接丢弃。
      if (appID !== options.getAppID() || envName !== options.getEnvName()) return;

      const descriptor = resolveBatchReleaseSuccessMessage(action, currentView);
      Message({
        theme: 'success',
        message: t(descriptor.key, descriptor.params),
      });

      // 只同步等待「进度回查」（带超时），让横幅文案立刻跟上；
      // 实例列表刷新走的是长连接 Watch，必须放后台，否则一次点击要等整轮建连，
      // 期间 pendingAction 不释放，所有按钮（含「更多」菜单项）都会被静默拦截。
      await withTimeout(fetchProgress(), PROGRESS_REFETCH_TIMEOUT);
      runRefresh(options.refreshInstances);
      runRefresh(options.refreshDeployStatus);
    } catch (error) {
      console.error(error);
      Message({
        theme: 'error',
        message: t('操作失败，请稍后重试'),
      });
      // 400/409 等前置条件不满足时以服务端状态为准。
      await withTimeout(fetchProgress(), PROGRESS_REFETCH_TIMEOUT);
    } finally {
      clearTimeout(watchdog);
      pendingAction.value = null;
    }
  }

  /** 点击横幅操作：需要二次确认的先弹 InfoBox，其余直接执行。 */
  function handleAction(item: BatchReleaseBannerAction) {
    if (item.disabled || pendingAction.value) return;
    if (!item.confirm) {
      void operate(item.action);
      return;
    }
    const { confirm } = item;
    InfoBox({
      title: t(confirm.titleKey),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'center',
      content: h(
        'div',
        {
          class: 'rounded-[2px] bg-[#F5F7FA] px-[12px] py-[16px] text-[14px] leading-[22px] text-[#4D4F56]',
        },
        t(confirm.descriptionKey),
      ),
      confirmText: t(confirm.confirmTextKey),
      cancelText: t('取消'),
      confirmButtonTheme: confirm.confirmTheme,
      onConfirm: () => {
        void operate(item.action);
      },
    });
  }

  /** 完成态横幅关闭：按 `应用:环境:deployID` 追加记录（保留其它环境的记忆），下一次发布重新出现。 */
  function handleClose() {
    if (!dismissKey.value) return;
    dismissedKeys.value = appendBatchReleaseDismissKey(dismissedKeys.value, dismissKey.value);
  }

  return {
    busyReason,
    handleAction,
    handleClose,
    isBusy,
    loading,
    pendingAction,
    progress,
    refresh,
    reset,
    resetForSwitch,
    showSkeleton,
    view,
  };
}

/** 给可能挂起的请求加超时：超时或失败都返回 undefined，由后续轮询兜底。 */
function withTimeout<T>(task: Promise<T>, ms: number): Promise<T | undefined> {
  return new Promise<T | undefined>(resolve => {
    const timer = setTimeout(() => resolve(undefined), ms);
    task.then(
      value => {
        clearTimeout(timer);
        resolve(value);
      },
      () => {
        clearTimeout(timer);
        resolve(undefined);
      },
    );
  });
}
