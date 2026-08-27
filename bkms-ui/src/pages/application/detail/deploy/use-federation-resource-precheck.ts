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

import { AppSpecService, EnvService } from '~/api/modules/v1';
import { isQuantityEqual, parseCpuCores, parseMemoryToMiB } from '~/common/resource-quantity';
import { isAppModelAppType } from '~/composables/app-type';
import { isFederationEnv } from '~/composables/use-is-federation-env';
import { useAppDetail } from '~/stores/app-detail';
import { useTrpcDeployStore } from '~/stores/trpc-deploy';

import type { AppSpecResourcesOutput } from '~/@types/v1/app-spec';
import type { EnvOutput } from '~/@types/v1/env';

/** 联邦集群下 Requests 与 Limits 不一致的单项资源。 */
export interface FederationResourceMismatch {
  /** 资源类型：CPU 或内存 */
  key: 'cpu' | 'memory';
  /** 当前 Limits 配置值（原始字符串，便于在弹窗中回显） */
  limits?: string;
  /** 当前 Requests 配置值（原始字符串，便于在弹窗中回显） */
  requests?: string;
}

/** 联邦集群部署前检查：CPU / 内存 Requests 必须与 Limits 一致。 */
export function useFederationResourcePrecheck() {
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  /** 逐项比较 CPU / 内存的 Requests 与 Limits，返回不一致的资源列表。 */
  function collectMismatches(spec?: AppSpecResourcesOutput | null): FederationResourceMismatch[] {
    const items: FederationResourceMismatch[] = [];
    if (!isQuantityEqual(spec?.cpuRequests, spec?.cpuLimits, parseCpuCores)) {
      items.push({
        key: 'cpu',
        limits: spec?.cpuLimits,
        requests: spec?.cpuRequests,
      });
    }
    if (!isQuantityEqual(spec?.memoryRequests, spec?.memoryLimits, parseMemoryToMiB)) {
      items.push({
        key: 'memory',
        limits: spec?.memoryLimits,
        requests: spec?.memoryRequests,
      });
    }
    return items;
  }

  /**
   * 解析指定环境名对应的 EnvOutput。
   * 查找优先级：传入的 targetEnv → 当前部署上下文已选环境 → 远端环境列表，命中即返回，避免不必要的请求。
   */
  async function resolveEnv(appID: string, envName: string, targetEnv?: EnvOutput): Promise<EnvOutput | undefined> {
    if (targetEnv?.name === envName) {
      return targetEnv;
    }
    if (trpcDeployStore.curEnvItem?.name === envName) {
      return trpcDeployStore.curEnvItem;
    }
    const envs = await EnvService.listAppEnvs({ appID });
    return (envs ?? []).find(item => item.name === envName);
  }

  /** 获取联邦环境中 Requests 与 Limits 不一致的资源。 */
  async function check(envName: string, targetEnv?: EnvOutput): Promise<FederationResourceMismatch[]> {
    // 仅 App 模型应用才走联邦资源预检，其它类型直接放行
    const appType = appDetailStore.appType;
    const appID = appDetailStore.appID;
    if (!isAppModelAppType(appType) || !appID) return [];

    const env = await resolveEnv(appID, envName, targetEnv);
    // 非联邦集群无需 Requests==Limits 约束，直接放行
    if (!isFederationEnv(env)) return [];

    const spec = await AppSpecService.getEnvEffectiveAppSpecResources({
      appID,
      envName,
    });

    return collectMismatches(spec);
  }

  return {
    check,
  };
}
