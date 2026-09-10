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

import { isAppModelAppType } from '~/composables/app-type';
import { useAppDetail } from '~/stores/app-detail';

import { useDeployAPIs } from './use-deploy';

import type { DeployPreCheckOutput } from '~/@types/v1/deploy';

/** 获取 tRPC / TAF 部署前的环境变量与必选集群组件检查结果。 */
export function useEnvVarPrecheck() {
  const appDetailStore = useAppDetail();

  async function check(envName: string): Promise<DeployPreCheckOutput> {
    const appType = appDetailStore.appType;
    const appID = appDetailStore.appID;
    if (!isAppModelAppType(appType)) return {};

    const preCheckDeploy = useDeployAPIs(appType).preCheckDeploy;
    if (!preCheckDeploy) return {};

    // 预检查接口直接返回结果对象，需要保留完整响应体，避免请求封装按默认规则读取 data 后得到 undefined。
    return preCheckDeploy(
      {
        appID,
        envName,
      },
      { needRes: true },
    );
  }

  return {
    check,
  };
}
