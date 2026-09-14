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

import { onBeforeUnmount } from 'vue';

import type { LoadMode } from './use-deploy-overview';

interface DeployOverviewPollingOptions {
  getAppID: () => string | undefined;
  getInterval: () => number;
  load: (mode: LoadMode) => Promise<void>;
}

/**
 * 部署总览轮询调度：每轮请求完成后再按最新间隔排下一轮，并在页面隐藏时暂停。
 */
export function useDeployOverviewPolling(options: DeployOverviewPollingOptions) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  // 每次主动刷新或停止都会推进代数，让旧定时器和迟到回调失效。
  let generation = 0;

  function clearTimer() {
    if (!timer) return;
    clearTimeout(timer);
    timer = undefined;
  }

  function isEnabled(appID = options.getAppID()) {
    return !!appID && document.visibilityState !== 'hidden';
  }

  /** 主动刷新会抢占当前轮询链；请求结束后由这次刷新负责续排下一轮。 */
  async function refresh(mode: LoadMode = 'manual') {
    const appID = options.getAppID();
    const runGeneration = generation + 1;
    generation = runGeneration;
    clearTimer();
    try {
      await options.load(mode);
    } finally {
      if (runGeneration === generation && appID === options.getAppID() && isEnabled(appID)) {
        schedule(runGeneration);
      }
    }
  }

  /** 自动轮询只执行一轮请求，下一轮必须等本轮返回后再调度，避免请求堆叠。 */
  async function run(runGeneration: number) {
    if (runGeneration !== generation || !isEnabled()) return;
    await options.load('automatic');
    if (runGeneration === generation && isEnabled()) schedule(runGeneration);
  }

  /** 调度下一轮时才读取 interval，让刚返回的快照能立即影响下一次间隔。 */
  function schedule(runGeneration = generation) {
    clearTimer();
    if (runGeneration !== generation || !isEnabled()) return;
    timer = setTimeout(() => {
      timer = undefined;
      void run(runGeneration);
    }, options.getInterval());
  }

  function stop() {
    generation += 1;
    clearTimer();
  }

  /** 后台标签页不轮询；回到前台立即静默刷新一次并恢复调度。 */
  function handleVisibilityChange() {
    if (document.visibilityState === 'hidden') {
      stop();
      return;
    }
    void refresh('automatic');
  }

  document.addEventListener('visibilitychange', handleVisibilityChange);

  onBeforeUnmount(() => {
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    stop();
  });

  return {
    refresh,
    stop,
  };
}
