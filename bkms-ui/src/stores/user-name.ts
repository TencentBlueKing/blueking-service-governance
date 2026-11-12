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

import { ref } from 'vue';

import { defineStore } from 'pinia';
import { getBkUserApiUrl } from '~/common/bk-user-api';
import { useUserStore } from '~/stores/user';

/** bk-user display_info 接口返回的用户结构 */
interface BkUserInfo {
  bk_username: string;
  display_name: string;
}

interface DisplayInfoResponse {
  code?: number;
  data?: BkUserInfo[];
  message?: string;
}

/** 缓存有效期（毫秒），与 @blueking/bk-user-display-name 组件保持一致 */
const CACHE_DURATION = 5 * 60 * 1000;
const MISSING_USER_CACHE_DURATION = 30 * 1000;

/** 批量接口单次请求的 userid 上限 */
const BATCH_SIZE = 100;

/**
 * userid -> 用户显示名称 缓存
 * 多租户环境下接口仅返回 userid（如 75flxt5zyapg61mn），
 * 通过 bk-user 网关的 display_info 批量接口反查 display_name（形如 "v_wmhawang(王XX)"），
 * 供搜索、筛选等需要按姓名匹配的场景使用（展示场景仍由 UserDisplay 组件负责）。
 */
export const useUserNameStore = defineStore('user-name', () => {
  /** userid -> display_name */
  const nameMap = ref<Record<string, string>>({});
  /** 各 userid 的缓存过期时间 */
  const expiresAt = new Map<string, number>();
  /** 待请求的 userid 队列（用于合并并发调用） */
  const waitingIds = new Set<string>();
  /** 当前正在进行的批量请求 */
  let inflight: null | Promise<void> = null;

  function getDisplayName(id: string) {
    return nameMap.value[id] ?? '';
  }

  /** 是否具备解析条件：已配置用户网关且为多租户（非 default）租户 */
  function canResolve() {
    const tenantId = useUserStore().tenantId;
    return !!getBkUserApiUrl() && !!tenantId && tenantId !== 'default';
  }

  async function fetchBatch(ids: string[]) {
    const apiBaseUrl = getBkUserApiUrl();
    const tenantId = useUserStore().tenantId;
    const query = ids.map(encodeURIComponent).join(',');
    const response = await fetch(`${apiBaseUrl}/api/v3/open-web/tenant/users/-/display_info/?bk_usernames=${query}`, {
      credentials: 'include',
      headers: { 'X-Bk-Tenant-Id': tenantId },
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const res: DisplayInfoResponse = await response.json();
    if (res?.code !== undefined && res.code !== 0) {
      throw new Error(res.message || `code ${res.code}`);
    }
    if (!Array.isArray(res?.data)) {
      throw new Error('Invalid user display info response');
    }

    const displayNames = new Map(res.data.map(user => [user.bk_username, user.display_name]));
    const now = Date.now();
    for (const id of ids) {
      const displayName = displayNames.get(id);
      nameMap.value[id] = displayName || id;
      expiresAt.set(id, now + (displayName ? CACHE_DURATION : MISSING_USER_CACHE_DURATION));
    }
  }

  /** 消费等待队列，分批请求直至清空 */
  async function flush() {
    try {
      while (waitingIds.size) {
        const batch = Array.from(waitingIds).slice(0, BATCH_SIZE);
        batch.forEach(id => waitingIds.delete(id));
        try {
          await fetchBatch(batch);
        } catch (err) {
          // 失败的批次不写入缓存，等待下次调用重试
          console.error('获取用户显示名称失败:', err);
        }
      }
    } finally {
      inflight = null;
    }
  }

  /**
   * 批量解析 userid -> display_name（并发调用会自动合并为同一轮请求）
   * 未配置用户网关或非多租户环境时静默跳过，nameMap 保持为空，由调用方降级按 userid 处理
   */
  async function resolveUserNames(ids: (null | string | undefined)[]) {
    if (!canResolve()) return;
    const now = Date.now();
    ids.forEach(id => {
      const key = id?.trim();
      if (key && now >= (expiresAt.get(key) ?? 0)) {
        waitingIds.add(key);
      }
    });
    if (!waitingIds.size) return;
    inflight ??= flush();
    await inflight;
  }

  return {
    nameMap,
    getDisplayName,
    resolveUserNames,
  };
});
