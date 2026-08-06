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

import { computed, onBeforeUnmount, readonly, ref, shallowRef, watch } from 'vue';
import type { Ref } from 'vue';

import { GpaService } from '~/api/modules/v1';

import type { GPAConfigOutputObj } from '~/@types/v1/gpa';

/** 响应式 getter，用于支持 appID / envName 动态变化 */
type Getter<T> = () => T;

/** 按 appID + envName 缓存的轮询条目 */
interface GPAConfigPollingEntry {
  appID: string;
  /** 缓存的最新配置 */
  config: Ref<GPAConfigOutputObj | null>;
  /** 进行中的请求 Promise，防并发 */
  currentRefresh: null | Promise<GPAConfigOutputObj | null>;
  envName: string;
  /** 缓存 key */
  key: string;
  needStatus: boolean;
  /** 轮询订阅者集合，非空时触发定时刷新 */
  pollingOwners: Set<symbol>;
  /** 轮询间隔（ms） */
  pollInterval: number;
  /** 下次刷新定时器 */
  timer: null | number;
  /** 使用者集合，用于引用计数 */
  users: Set<symbol>;
}

/** 轮询配置选项 */
interface UseGPAConfigPollingOptions {
  /** 是否激活，返回 false 时暂停轮询 */
  active?: Getter<boolean>;
  appID: Getter<string | undefined>;
  envName: Getter<string | undefined>;
  /** 是否请求 status 字段，默认 true */
  needStatus?: boolean;
  /** 轮询间隔（ms），默认 5000 */
  pollInterval?: number;
}

const DEFAULT_POLL_INTERVAL = 5000;

/**
 * 全局轮询缓存，同 appID + envName 共享一个 entry，
 * 避免重复创建定时器和请求。
 */
const gpaConfigPollingMap = new Map<string, GPAConfigPollingEntry>();

/**
 * GPA 配置轮询 composable。
 * 同 appID + envName 的调用方共享轮询任务，全部卸载后自动清理。
 */
export function useGPAConfigPolling(options: UseGPAConfigPollingOptions) {
  /** 当前调用方标识 */
  const owner = Symbol('gpa-config-polling-owner');
  /** 当前轮询条目 */
  const currentEntry = shallowRef<GPAConfigPollingEntry | null>(null);
  const isLoading = ref(false);
  const needStatus = options.needStatus ?? true;
  const pollInterval = options.pollInterval ?? DEFAULT_POLL_INTERVAL;

  /** 当前配置 */
  const config = computed(() => currentEntry.value?.config.value ?? null);
  /** 是否已启用 */
  const enabled = computed(() => !!config.value?.enabled);
  /** 当前状态 */
  const status = computed(() => config.value?.status || null);

  /** 手动刷新一次配置 */
  async function refresh() {
    const entry = currentEntry.value;
    if (!entry) return null;
    isLoading.value = true;
    try {
      return await refreshEntry(entry);
    } finally {
      // entry 未切换时才重置 loading
      if (currentEntry.value === entry) {
        isLoading.value = false;
      }
    }
  }

  /** 开始轮询 */
  function startPolling() {
    const entry = currentEntry.value;
    if (!entry) return;
    entry.pollingOwners.add(owner);
    scheduleNextRefresh(entry);
  }

  /** 停止轮询 */
  function stopPolling() {
    const entry = currentEntry.value;
    if (!entry) return;
    entry.pollingOwners.delete(owner);
    if (entry.pollingOwners.size === 0) {
      clearPollingTimer(entry);
    }
  }

  /** 根据条件切换轮询，通常 GPA 启用时开启、禁用时停止 */
  function updatePolling(shouldPoll = enabled.value) {
    if (shouldPoll) {
      startPolling();
    } else {
      stopPolling();
    }
  }

  /** 移除 owner 引用，无人使用时清理定时器和缓存 */
  function releaseEntry() {
    const entry = currentEntry.value;
    if (!entry) return;
    entry.pollingOwners.delete(owner);
    entry.users.delete(owner);
    if (entry.pollingOwners.size === 0) {
      clearPollingTimer(entry);
    }
    deleteEntryIfUnused(entry);
    currentEntry.value = null;
  }

  /** 获取或创建 appID + envName 对应的 entry */
  function acquireEntry(appID: string, envName: string) {
    const key = buildKey(appID, envName);
    if (currentEntry.value?.key === key) return;

    releaseEntry();

    let entry = gpaConfigPollingMap.get(key);
    if (!entry) {
      entry = createEntry(key, appID, envName, needStatus, pollInterval);
      gpaConfigPollingMap.set(key, entry);
    }
    entry.users.add(owner);
    currentEntry.value = entry;
  }

  // appID / envName / active 变化时自动切换目标并刷新
  watch(
    () => [options.appID(), options.envName(), options.active?.() ?? true] as const,
    ([appID, envName, active]) => {
      if (!active || !appID || !envName) {
        releaseEntry();
        return;
      }

      acquireEntry(appID, envName);
      refresh();
    },
    { immediate: true, flush: 'sync' },
  );

  onBeforeUnmount(() => {
    releaseEntry();
  });

  return {
    config,
    enabled,
    isLoading: readonly(isLoading),
    refresh,
    startPolling,
    status,
    stopPolling,
    updatePolling,
  };
}

/** 生成全局缓存 key */
function buildKey(appID: string, envName: string) {
  return `${appID}::${envName}`;
}

/** 清除定时器 */
function clearPollingTimer(entry: GPAConfigPollingEntry) {
  if (entry.timer) {
    clearTimeout(entry.timer);
    entry.timer = null;
  }
}

/** 创建轮询条目 */
function createEntry(
  key: string,
  appID: string,
  envName: string,
  needStatus: boolean,
  pollInterval: number,
): GPAConfigPollingEntry {
  return {
    appID,
    envName,
    key,
    needStatus,
    pollInterval,
    config: ref<GPAConfigOutputObj | null>(null),
    currentRefresh: null,
    pollingOwners: new Set(),
    timer: null,
    users: new Set(),
  };
}

/** 无 user 时从全局 Map 移除 entry */
function deleteEntryIfUnused(entry: GPAConfigPollingEntry) {
  if (entry.users.size > 0) return;
  clearPollingTimer(entry);
  gpaConfigPollingMap.delete(entry.key);
}

/** 刷新配置（已有进行中请求时复用，避免并发重复） */
function refreshEntry(entry: GPAConfigPollingEntry) {
  if (entry.currentRefresh) return entry.currentRefresh;

  entry.currentRefresh = requestConfig(entry)
    .then(config => {
      entry.config.value = config;
      return config;
    })
    .catch(error => {
      console.warn(error);
      entry.config.value = null;
      return null;
    })
    .finally(() => {
      entry.currentRefresh = null;
    });
  return entry.currentRefresh;
}

/** 请求配置 API，404 视为无配置返回 null */
function requestConfig(entry: GPAConfigPollingEntry) {
  return GpaService.getAppEnvGPAConfig(
    {
      appID: entry.appID,
      envName: entry.envName,
    },
    {
      interceptorErr: false,
      needStatus: entry.needStatus,
    },
  ).catch(error => {
    if ((error as { status?: number })?.status === 404) {
      return null;
    }
    throw error;
  });
}

/** 调度下一次轮询，刷新完成后递归自身形成循环；已有定时器或无订阅者时跳过 */
function scheduleNextRefresh(entry: GPAConfigPollingEntry) {
  if (entry.timer || entry.pollingOwners.size === 0) return;
  entry.timer = setTimeout(() => {
    entry.timer = null;
    refreshEntry(entry).finally(() => scheduleNextRefresh(entry));
  }, entry.pollInterval) as unknown as number;
}
