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

import { APP_DEPLOY_STATUS, DEPLOY_FAILED_STATUSES, HELM_DEPLOY_STATUS } from '~/common/enums/deploy';
import { buildStandardEnvMap, getFeatureSourceEnv } from '~/composables/use-env-manager';

import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
import type { EnvOutput } from '~/@types/v1/env';

export type RequiredVisibleEnvReason = 'deployment' | 'feature-parent';

const requiredDeployStatuses = new Set<string>([
  APP_DEPLOY_STATUS.DEPLOYING,
  APP_DEPLOY_STATUS.DEPLOYED,
  ...DEPLOY_FAILED_STATUSES,
  HELM_DEPLOY_STATUS.PENDING_INSTALL,
  HELM_DEPLOY_STATUS.PENDING_UPGRADE,
  HELM_DEPLOY_STATUS.PENDING_ROLLBACK,
]);

/** 返回必须保持可见的标准环境及原因；同一环境命中两种规则时优先展示部署提示。 */
export function getRequiredVisibleEnvReasons(envs: EnvOutput[], deployStatuses: AppDeployedEnvOutputObj[]) {
  const standardEnvMap = buildStandardEnvMap(envs.filter(env => (env.kind || 'standard') === 'standard'));
  const reasons = new Map<string, RequiredVisibleEnvReason>();

  envs
    .filter(env => env.kind === 'feature')
    .forEach(env => {
      const sourceEnv = getFeatureSourceEnv(env, standardEnvMap);
      if (sourceEnv?.name) reasons.set(sourceEnv.name, 'feature-parent');
    });

  deployStatuses.forEach(status => {
    if (!requiredDeployStatuses.has(status.deployStatus || '')) return;
    const env =
      (status.id ? standardEnvMap.byID.get(status.id) : undefined) ||
      (status.name ? standardEnvMap.byName.get(status.name) : undefined);
    if (env?.name) reasons.set(env.name, 'deployment');
  });

  return reasons;
}
