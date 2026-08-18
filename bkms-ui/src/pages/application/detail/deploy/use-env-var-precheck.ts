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

import { ref, watch } from 'vue';

import { isAppModelAppType } from '~/composables/app-type';
import { useAppDetail } from '~/stores/app-detail';

import { useDeployAPIs } from './use-deploy';

import type { UndefinedEnvVarOutput } from '~/@types/v1/deploy';

/**
 * 管理 tRPC / TAF 部署前环境变量检查及警告确认弹框状态。
 */
export function useEnvVarPrecheck() {
  const appDetailStore = useAppDetail();
  const isShowPrecheckDialog = ref(false);
  const precheckEnvName = ref('');
  const undefinedVars = ref<UndefinedEnvVarOutput[]>([]);
  let resolvePrecheck: ((passed: boolean) => void) | undefined;

  /** 完成本次预检查确认；未定义变量属于警告，用户可选择继续部署。 */
  function completePrecheck(passed: boolean) {
    const resolve = resolvePrecheck;
    resolvePrecheck = undefined;
    isShowPrecheckDialog.value = false;
    resolve?.(passed);
  }

  function continueDeploy() {
    completePrecheck(true);
  }

  function cancelDeploy() {
    completePrecheck(false);
  }

  async function precheck(envName: string): Promise<boolean> {
    const appType = appDetailStore.appType;
    if (!isAppModelAppType(appType)) return true;

    const preCheckDeployEnvVars = useDeployAPIs(appType).preCheckDeployEnvVars;
    if (!preCheckDeployEnvVars) return true;

    precheckEnvName.value = envName;
    undefinedVars.value = [];
    isShowPrecheckDialog.value = false;

    // 预检查接口直接返回 { undefinedVars }，需要保留完整响应体，避免请求封装按默认规则读取 data 后得到 undefined。
    const response = await preCheckDeployEnvVars(
      {
        appID: appDetailStore.appID,
        envName,
      },
      { needRes: true },
    );

    undefinedVars.value = response.undefinedVars ?? [];

    if (undefinedVars.value.length === 0) return true;

    isShowPrecheckDialog.value = true;
    return new Promise<boolean>(resolve => {
      resolvePrecheck = resolve;
    });
  }

  // 点击弹窗关闭图标时视为取消，避免部署流程一直等待用户确认。
  watch(isShowPrecheckDialog, show => {
    if (!show && resolvePrecheck) {
      const resolve = resolvePrecheck;
      resolvePrecheck = undefined;
      resolve(false);
    }
  });

  return {
    cancelDeploy,
    continueDeploy,
    isShowPrecheckDialog,
    precheck,
    precheckEnvName,
    undefinedVars,
  };
}
