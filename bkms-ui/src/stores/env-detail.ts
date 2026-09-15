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
import { EnvService } from '~/api/modules/v1';

import type { EnvDetailOutput, EnvOutput } from '~/@types/v1/env';

/** 环境详情页 store：环境列表与当前环境详情的加载、同步与重置。 */
export const useEnvDetailStore = defineStore('envDetail', () => {
  // 环境列表（按 workspace 维度）
  const envList = ref<EnvOutput[]>([]);
  // 当前查看的环境详情
  const currentEnv = ref<EnvDetailOutput | null>(null);
  // 当前 workspace ID
  const workspace = ref('');
  // 详情加载状态与错误类型（notFound：环境不存在；request：请求失败）
  const loading = ref(false);
  const listLoading = ref(false);
  const error = ref<'notFound' | 'request' | null>(null);
  const listError = ref(false);
  // 自增序列号，用于丢弃过期请求的响应，避免竞态覆盖
  let detailSequence = 0;
  let listSequence = 0;
  // 最近一次请求详情的环境 ID，用于校验响应归属
  let requestedEnvId = '';

  /** 按 workspace 拉取环境列表；workspace 变化时重置详情。 */
  async function fetchEnvList(space = workspace.value) {
    const sequence = ++listSequence;
    if (space !== workspace.value) {
      resetDetail();
      envList.value = [];
      workspace.value = space;
    }
    listLoading.value = true;
    listError.value = false;
    try {
      const list = await EnvService.listEnvs({ workspaceID: space });
      if (sequence === listSequence) envList.value = list;
    } catch {
      if (sequence === listSequence) listError.value = true;
    } finally {
      if (sequence === listSequence) listLoading.value = false;
    }
  }

  /** 拉取当前环境详情；404 归类为 notFound，其余归为 request 错误。 */
  async function fetchCurrentEnv(envId: string) {
    const sequence = ++detailSequence;
    requestedEnvId = envId;
    currentEnv.value = null;
    error.value = null;
    loading.value = true;
    try {
      const data = await EnvService.getEnv({ envID: envId }, { needStatus: true });
      if (sequence !== detailSequence) return;
      if (!data?.id) error.value = 'notFound';
      else setCurrentEnv(data);
    } catch (reason) {
      if (sequence !== detailSequence) return;
      error.value =
        typeof reason === 'object' && reason !== null && 'status' in reason && reason.status === 404
          ? 'notFound'
          : 'request';
    } finally {
      if (sequence === detailSequence) loading.value = false;
    }
  }

  /** 校验环境 ID 后写入详情，并同步更新列表中的对应项。 */
  function setCurrentEnv(env: EnvDetailOutput) {
    if (env.id !== requestedEnvId) return;
    currentEnv.value = env;
    envList.value = envList.value.map(item => (item.id === env.id ? { ...item, ...env } : item));
  }

  /** 提交成功后同步已保存字段；后续刷新失败也不能退回旧值。 */
  async function syncSavedEnv(space: string, envId: string, changes: Partial<EnvDetailOutput>) {
    if (space !== workspace.value) return;
    envList.value = envList.value.map(item => (item.id === envId ? { ...item, ...changes } : item));
    if (envId !== requestedEnvId) return;
    if (!currentEnv.value) {
      await fetchCurrentEnv(envId);
      return;
    }

    setCurrentEnv({ ...currentEnv.value, ...changes });
    const sequence = ++detailSequence;
    try {
      const data = await EnvService.getEnv({ envID: envId }, { irrevocable: true });
      if (sequence === detailSequence && data?.id === envId) setCurrentEnv(data);
    } catch {
      // 拦截器提示刷新错误，保留服务端已确认保存的字段。
    }
  }

  /** 重置详情状态，并使未完成的详情请求失效。 */
  function resetDetail() {
    ++detailSequence;
    requestedEnvId = '';
    currentEnv.value = null;
    loading.value = false;
    error.value = null;
  }

  /** 重置整个 store（详情 + 列表）。 */
  function reset() {
    resetDetail();
    ++listSequence;
    envList.value = [];
    workspace.value = '';
    listLoading.value = false;
    listError.value = false;
  }

  return {
    envList,
    currentEnv,
    loading,
    listLoading,
    error,
    listError,
    fetchEnvList,
    fetchCurrentEnv,
    setCurrentEnv,
    syncSavedEnv,
    reset,
  };
});
