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

import { onScopeDispose, ref, shallowRef, watch } from 'vue';

import { type AppInstanceOutputObj, type WatchAppInstancesRequest } from '~/@types/v1/instance';
import { InstanceService } from '~/api/modules/v1';

import { notifyInstanceWatchError } from './instance-watch-notifier';
import { extractSseEventBlocks, parseInstanceSseBlock, reduceInstanceWatchEvent } from './instance-watch-utils';

import type { InstanceListSourceMode, InstanceWatchEvent } from '../types';

const RECONNECT_ENDED_REASONS = new Set(['watch timeout', 'cluster watch interrupted']);
// 异常断流 / 409 重连前的冷却时长，正常 ENDED 续流不受此限制。
const ABNORMAL_RECONNECT_DELAY_MS = 5000;
// 联邦环境暂不支持 Watch，退化为全量 List 轮询；所有轮询退避参数集中在此处，便于按集群容量调整。
const POLLING_DEFAULT_INTERVAL_MS = 10000;
const POLLING_INTERVAL_STEP_MS = 5000;
// 限制最大等待时间，避免持续慢响应使轮询无限退避。
const POLLING_MAX_INTERVAL_MS = 30000;
// 单轮 List 耗时达到默认间隔即判定为慢响应，下轮增加一个退避步长。
const POLLING_SLOW_RESPONSE_MS = POLLING_DEFAULT_INTERVAL_MS;
// 恢复阈值低于慢响应阈值，形成滞后区间，避免耗时在边界附近频繁切换间隔。
const POLLING_RECOVER_RESPONSE_MS = 8000;

interface InstanceWatchScope {
  appID: string;
  envName: string;
  trafficLaneName?: string;
}

interface UseInstanceListWatchOptions {
  enabled?: () => boolean;
  getMode?: () => InstanceListSourceMode;
  getScope: () => InstanceWatchScope;
}

/** 管理单个应用环境的实例全量 List 与后续 Watch/轮询数据源。 */
export function useInstanceListWatch(options: UseInstanceListWatchOptions) {
  // 对页面暴露的状态：实例始终保存全量快照，筛选、排序和分页由消费方在本地完成。
  const instances = ref<AppInstanceOutputObj[]>([]);
  const isInitialLoading = ref(false);
  // 仅 Watch 模式在 SSE 建连成功后为 true；联邦轮询模式没有长连接，isWatching 恒为 false。
  const isWatching = ref(false);
  const lastError = shallowRef<unknown>();
  // 联邦轮询的自适应间隔：慢响应逐次退避，恢复后逐步回落，避免大量 Pod 场景请求堆积。
  const pollingIntervalMs = ref(POLLING_DEFAULT_INTERVAL_MS);

  // 每次停止或刷新都会递增 generation，异步返回值只有代次一致时才允许写入当前页面。
  let activeController: AbortController | undefined;
  let disposed = false;
  let generation = 0;
  // 异常重连冷却期间的挂起定时器，代次失效或停止时需清除以避免过期后误触发重连。
  let abnormalReconnectTimer: ReturnType<typeof setTimeout> | undefined;
  // 联邦轮询的下一轮定时器，停止、隐藏或切换环境时必须同步清理。
  let pollingTimer: ReturnType<typeof setTimeout> | undefined;
  // 同一作用域拿到过首包后，后续手动刷新或正常续流不再遮住现有列表展示 Skeleton。
  let hasLoadedSnapshot = false;
  let lastScopeKey = '';

  function getMode() {
    return options.getMode?.() ?? 'watch';
  }

  /** 只有页面可见、作用域完整且业务允许请求时才能启动实例数据源。 */
  function isEnabled() {
    const scope = options.getScope();
    return (
      !disposed &&
      (options.enabled?.() ?? true) &&
      document.visibilityState !== 'hidden' &&
      Boolean(scope.appID && scope.envName)
    );
  }

  /** 清除挂起的异常重连定时器，防止冷却结束后触发已失效的重连。 */
  function clearAbnormalReconnectTimer() {
    if (!abnormalReconnectTimer) return;
    clearTimeout(abnormalReconnectTimer);
    abnormalReconnectTimer = undefined;
  }

  /** 清除联邦轮询定时器，防止停止后继续发起下一轮 List。 */
  function clearPollingTimer() {
    if (!pollingTimer) return;
    clearTimeout(pollingTimer);
    pollingTimer = undefined;
  }

  /** 使当前代次失效并中止 List 请求、Watch 流或等待中的定时任务。 */
  function invalidateCurrentConnection() {
    generation += 1;
    clearAbnormalReconnectTimer();
    clearPollingTimer();
    activeController?.abort();
    activeController = undefined;
    isWatching.value = false;
  }

  /** 停止连接但保留最后一次实例快照，供隐藏页面或折叠环境恢复前继续展示。 */
  function stop() {
    invalidateCurrentConnection();
    isInitialLoading.value = false;
  }

  /** 清空当前作用域的实例首包；下一次 List 重新进入首次加载状态。 */
  function clear() {
    instances.value = [];
    hasLoadedSnapshot = false;
    // 切换作用域视为全新环境，轮询间隔回到基准，避免沿用上一环境的降级状态。
    pollingIntervalMs.value = POLLING_DEFAULT_INTERVAL_MS;
  }

  /** 依据本轮 List 实际耗时逐步调整下一轮间隔；滞后区间避免在阈值附近抖动。 */
  function adjustPollingInterval(elapsedMs: number) {
    if (elapsedMs >= POLLING_SLOW_RESPONSE_MS) {
      pollingIntervalMs.value = Math.min(pollingIntervalMs.value + POLLING_INTERVAL_STEP_MS, POLLING_MAX_INTERVAL_MS);
    } else if (elapsedMs <= POLLING_RECOVER_RESPONSE_MS) {
      pollingIntervalMs.value = Math.max(
        pollingIntervalMs.value - POLLING_INTERVAL_STEP_MS,
        POLLING_DEFAULT_INTERVAL_MS,
      );
    }
  }

  /** 主动 abort 属于正常生命周期清理，不作为请求失败提示给用户。 */
  function isAbortError(error: unknown) {
    return error instanceof DOMException ? error.name === 'AbortError' : (error as Error)?.name === 'AbortError';
  }

  /** Watch 建连只有 HTTP 409 代表位点失效，需要重新 List 获取新的 resourceVersion。 */
  function isWatchConflict(error: unknown) {
    return (error as { status?: number })?.status === 409;
  }

  /** 非重连类异常只记录并展示接口返回的错误，不进行定时重试。 */
  function handleFailure(error: unknown, runGeneration: number) {
    if (runGeneration !== generation || isAbortError(error) || !isEnabled()) return;
    isWatching.value = false;
    activeController = undefined;
    lastError.value = error;
    notifyInstanceWatchError(error);
  }

  /** 联邦轮询失败时保留现有列表，并展示本次 List 接口返回的错误，后台仍等待下一轮恢复。 */
  function handlePollingFailure(error: unknown, runGeneration: number) {
    if (runGeneration !== generation || isAbortError(error) || !isEnabled() || getMode() !== 'polling') return;
    activeController = undefined;
    lastError.value = error;
    notifyInstanceWatchError(error);
  }

  /**
   * 断流、标准 ENDED 或 409 都必须完整重新 List，禁止复用旧 resourceVersion。
   * delay=true 时先冷却再重连（异常断流 / 409 专用），默认立即续流（正常 ENDED）。
   */
  function reconnect(runGeneration: number, error?: unknown, options: { delay?: boolean } = {}) {
    if (runGeneration !== generation || isAbortError(error) || !isEnabled()) return;
    isWatching.value = false;
    lastError.value = undefined;

    if (options.delay) {
      // 只有异常断流和 409 做冷却，避免服务端持续立即断开时前端空转打 List + Watch。
      clearAbnormalReconnectTimer();
      activeController?.abort();
      activeController = undefined;
      abnormalReconnectTimer = setTimeout(() => {
        abnormalReconnectTimer = undefined;
        // 冷却结束后再次校验代次与可见性，期间若已停止或切换作用域则放弃本次重连。
        if (runGeneration !== generation || !isEnabled()) return;
        void refresh();
      }, ABNORMAL_RECONNECT_DELAY_MS);
      return;
    }

    void refresh();
  }

  /** 用纯函数归并事件，保证所有页面共用一致的 ADDED/MODIFIED/DELETED/PLUGIN 语义。 */
  function applyEvent(event: InstanceWatchEvent) {
    instances.value = reduceInstanceWatchEvent(instances.value, event);
  }

  /** 拉取实例全量快照；Watch 与联邦轮询都复用该入口，保证请求参数一致。 */
  async function listAllInstances(scope: InstanceWatchScope, controller: AbortController, runGeneration: number) {
    const listResult = await InstanceService.listAppInstances(
      {
        appID: scope.appID,
        envName: scope.envName,
        all: true,
        ...(scope.trafficLaneName ? { trafficLaneName: scope.trafficLaneName } : {}),
      },
      { interceptorErr: false, signal: controller.signal },
    );
    if (runGeneration !== generation || controller.signal.aborted) return undefined;

    instances.value = (listResult.results || []) as AppInstanceOutputObj[];
    hasLoadedSnapshot = true;
    // Watch 返回响应头可能较慢，List 首包可展示后立即关闭 loading；轮询首包同样不等待下一轮。
    isInitialLoading.value = false;
    lastError.value = undefined;
    return listResult;
  }

  /** 按块消费 SSE；返回 ENDED.reason，读流、解析或异常关流由调用方触发重新 List。 */
  async function consumeWatchStream(response: Response, controller: AbortController, runGeneration: number) {
    if (!response.body) throw new Error('instance watch response body is empty');

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    // reader.read() 的边界与 SSE 事件边界无关，buffer 用于保留被拆开的 UTF-8 文本和事件尾部。
    let buffer = '';
    let ended = false;
    let endedReason: string | undefined;
    let streamCompleted = false;

    try {
      while (!controller.signal.aborted && runGeneration === generation) {
        const { done, value } = await reader.read();
        buffer += decoder.decode(value, { stream: !done });

        const extracted = extractSseEventBlocks(buffer);
        buffer = extracted.rest;
        // 异常 EOF 前仍解析最后一段完整 JSON，随后按非正常关流处理，避免静默停在旧数据。
        const blocks = done && buffer.trim() ? [...extracted.blocks, buffer] : extracted.blocks;
        if (done) buffer = '';

        for (const block of blocks) {
          const parsed = parseInstanceSseBlock(block);
          // 心跳注释不含业务数据，仅用于保持连接存活。
          if (!parsed.event) continue;
          if (parsed.event.type === 'ENDED') {
            // 任意 ENDED 都必须收流；reason 只决定它是正常 timeout 还是需要报错的异常结束。
            ended = true;
            endedReason = parsed.event.reason;
            break;
          }
          applyEvent(parsed.event);
        }

        if (ended) return endedReason;
        if (done) {
          streamCompleted = true;
          throw new Error('instance watch stream closed unexpectedly');
        }
      }

      return undefined;
    } finally {
      // ENDED 和解析异常需要主动释放服务端流；abort 场景由 AbortController 负责取消。
      if (!streamCompleted && !controller.signal.aborted) {
        await reader.cancel().catch(() => undefined);
      }
      reader.releaseLock();
    }
  }

  /** 调度下一次联邦轮询；每轮 List 结束后再排下一轮，避免请求重叠。间隔按上一轮响应耗时自适应。 */
  function schedulePolling(runGeneration: number) {
    clearPollingTimer();
    if (runGeneration !== generation || !isEnabled() || getMode() !== 'polling') return;
    pollingTimer = setTimeout(() => {
      pollingTimer = undefined;
      if (runGeneration !== generation || !isEnabled() || getMode() !== 'polling') return;
      const controller = new AbortController();
      activeController = controller;
      void poll(runGeneration, controller);
    }, pollingIntervalMs.value);
  }

  /** 联邦环境轮询路径：只做全量 List，不建 Watch，不依赖 resourceVersion。 */
  async function poll(runGeneration: number, controller: AbortController) {
    const scope = options.getScope();
    // 记录本轮 List 起点，用于在结束后按实际响应耗时调整下一轮间隔。
    const startedAt = Date.now();
    try {
      // 首次失败后后台轮询不再反复展示 Skeleton，保留错误空态直到下一次成功快照到达。
      isInitialLoading.value = !hasLoadedSnapshot && !lastError.value;
      const listResult = await listAllInstances(scope, controller, runGeneration);
      if (!listResult) return;
    } catch (error) {
      handlePollingFailure(error, runGeneration);
    } finally {
      if (runGeneration === generation) {
        isInitialLoading.value = false;
        if (activeController === controller) {
          activeController = undefined;
        }
        // 无论成功或失败都以本轮实际耗时评估，接口变慢（含超时）时同样触发降级。
        adjustPollingInterval(Date.now() - startedAt);
        schedulePolling(runGeneration);
      }
    }
  }

  /** 先用 List(all=true) 替换全量快照，再从同一次响应的 resourceVersion 建立 Watch。 */
  async function connectWatch(runGeneration: number, controller: AbortController) {
    const scope = options.getScope();
    try {
      isInitialLoading.value = !hasLoadedSnapshot;
      const listResult = await listAllInstances(scope, controller, runGeneration);
      if (!listResult) return;
      const resourceVersion = listResult.resourceVersion;
      if (!resourceVersion) throw new Error('instance list response is missing resourceVersion');

      const request: WatchAppInstancesRequest = {
        appID: scope.appID,
        envName: scope.envName,
        resourceVersion,
        ...(scope.trafficLaneName ? { trafficLaneName: scope.trafficLaneName } : {}),
      };
      let response: Response;
      try {
        // 必须拿原始 Response 才能通过 body.getReader() 持续消费 SSE 数据。
        response = await InstanceService.watchAppInstances<WatchAppInstancesRequest, Response>(request, {
          interceptorErr: false,
          needStatus: true,
          originalResponse: true,
          signal: controller.signal,
        });
      } catch (error) {
        if (isWatchConflict(error)) {
          // 409 说明位点失效，若 List 返回的 RV 持续冲突需冷却，避免高频空转重连。
          reconnect(runGeneration, error, { delay: true });
        } else {
          handleFailure(error, runGeneration);
        }
        return;
      }
      if (runGeneration !== generation || controller.signal.aborted) return;

      isWatching.value = true;
      void consumeWatchStream(response, controller, runGeneration)
        .then(reason => {
          if (runGeneration !== generation || controller.signal.aborted) return;
          isWatching.value = false;
          if (reason && RECONNECT_ENDED_REASONS.has(reason)) {
            // timeout 是正常到期，interrupted 是服务端主动中断；两者都从新的 List 位点恢复。
            reconnect(runGeneration);
            return;
          }
          // 未知 ENDED 不属于契约规定的重连条件，展示实际 reason 后停止。
          handleFailure(new Error(reason || 'instance watch stream ended'), runGeneration);
        })
        // reader.read、SSE 解析和异常 EOF 都可能遗漏增量，必须重新 List 补齐后再 Watch。
        .catch(error => reconnect(runGeneration, error, { delay: true }));
    } catch (error) {
      handleFailure(error, runGeneration);
    } finally {
      if (runGeneration === generation) {
        isInitialLoading.value = false;
      }
    }
  }

  /** 完整刷新会先终止旧代次，确保同一作用域最多只有一条有效 Watch 或轮询。 */
  async function refresh() {
    invalidateCurrentConnection();
    if (!isEnabled()) return;

    const runGeneration = generation;
    const controller = new AbortController();
    activeController = controller;
    if (getMode() === 'polling') {
      await poll(runGeneration, controller);
    } else {
      await connectWatch(runGeneration, controller);
    }
  }

  /** 页面隐藏时立即断流；重新可见时从新的 List 位点恢复，避免补读旧连接。 */
  function handleVisibilityChange() {
    if (document.visibilityState === 'hidden') {
      stop();
      return;
    }
    if (isEnabled()) void refresh();
  }

  document.addEventListener('visibilitychange', handleVisibilityChange);

  // 应用、环境、泳道、enabled 或数据源模式变化时重建连接；切换作用域必须清空旧环境快照。
  watch(
    () => {
      const scope = options.getScope();
      return [scope.appID, scope.envName, scope.trafficLaneName || '', options.enabled?.() ?? true, getMode()] as const;
    },
    ([appID, envName, trafficLaneName, enabled, _mode]) => {
      const scopeKey = [appID, envName, trafficLaneName].join('\u0000');
      if (scopeKey !== lastScopeKey) {
        lastScopeKey = scopeKey;
        stop();
        clear();
        lastError.value = undefined;
      }

      if (enabled && appID && envName && document.visibilityState !== 'hidden') {
        void refresh();
      } else {
        stop();
      }
    },
    { immediate: true },
  );

  // 组件卸载时移除全局监听并中止所有尚未完成的请求或 reader。
  onScopeDispose(() => {
    disposed = true;
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    stop();
  });

  return {
    clear,
    instances,
    isInitialLoading,
    isWatching,
    lastError,
    pollingIntervalMs,
    refresh,
    stop,
  };
}
