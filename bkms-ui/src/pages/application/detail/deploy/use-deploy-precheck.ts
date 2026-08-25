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

import { onScopeDispose, ref, watch } from 'vue';

import { useAppDetail } from '~/stores/app-detail';

import { useEnvVarPrecheck } from './use-env-var-precheck';
import { useFederationResourcePrecheck } from './use-federation-resource-precheck';

import type { FederationResourceMismatch } from './use-federation-resource-precheck';
import type { UndefinedEnvVarOutput } from '~/@types/v1/deploy';
import type { EnvOutput } from '~/@types/v1/env';

/** 将联邦资源硬阻断与环境变量警告汇总到同一次部署预检。 */
export function useDeployPrecheck() {
  const appDetailStore = useAppDetail();
  const envVarPrecheck = useEnvVarPrecheck();
  const federationPrecheck = useFederationResourcePrecheck();
  const isShowPrecheckDialog = ref(false);
  const precheckEnvName = ref('');
  const federationMismatches = ref<FederationResourceMismatch[]>([]);
  const undefinedVars = ref<UndefinedEnvVarOutput[]>([]);
  let precheckRunID = 0;
  let resolvePrecheck: ((passed: boolean) => void) | undefined;

  function completePrecheck(passed: boolean) {
    precheckRunID += 1;
    const resolve = resolvePrecheck;
    resolvePrecheck = undefined;
    isShowPrecheckDialog.value = false;
    resolve?.(passed);
  }

  function cancelDeploy() {
    completePrecheck(false);
  }

  function continueDeploy() {
    // 资源规格不一致属于硬阻断，不能通过事件绕过弹窗按钮状态继续部署。
    if (federationMismatches.value.length > 0) return;
    completePrecheck(true);
  }

  async function precheck(envName: string, targetEnv?: EnvOutput): Promise<boolean> {
    // 新一轮检查开始时结束旧弹窗流程，并让尚未完成的旧请求结果失效。
    if (resolvePrecheck) {
      completePrecheck(false);
    }
    const runID = ++precheckRunID;
    const appID = appDetailStore.appID;
    precheckEnvName.value = envName;
    federationMismatches.value = [];
    undefinedVars.value = [];
    isShowPrecheckDialog.value = false;

    // 顺序固定：先检查联邦资源规格，再检查环境变量，以便在一个弹窗中完整展示问题。
    const mismatches = await federationPrecheck.check(envName, targetEnv);
    if (runID !== precheckRunID || appID !== appDetailStore.appID) return false;
    federationMismatches.value = mismatches;

    const vars = await envVarPrecheck.check(envName);
    if (runID !== precheckRunID || appID !== appDetailStore.appID) return false;
    undefinedVars.value = vars;
    if (mismatches.length === 0 && vars.length === 0) return true;

    isShowPrecheckDialog.value = true;
    return new Promise<boolean>(resolve => {
      resolvePrecheck = resolve;
    });
  }

  // 点击弹窗关闭图标时视为取消，避免部署流程一直等待用户确认。
  watch(isShowPrecheckDialog, show => {
    if (!show && resolvePrecheck) {
      completePrecheck(false);
    }
  });

  // 页面离开时结束等待中的检查，避免部署处理函数长期悬挂。
  onScopeDispose(() => completePrecheck(false));

  return {
    cancelDeploy,
    continueDeploy,
    federationMismatches,
    isShowPrecheckDialog,
    precheck,
    precheckEnvName,
    undefinedVars,
  };
}
