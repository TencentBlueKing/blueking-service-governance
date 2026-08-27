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

import { computed, toValue } from 'vue';
import type { MaybeRefOrGetter } from 'vue';

import type { EnvOutput } from '~/@types/v1/env';

type EnvWithCluster = Pick<EnvOutput, 'cluster'>;

/** 仅以环境接口返回的 isFederation=true 判定联邦环境。 */
export function isFederationEnv(env?: EnvWithCluster): boolean {
  return env?.cluster?.isFederation === true;
}

/** 根据响应式环境数据返回联邦环境状态。 */
export default function useIsFederationEnv(source: MaybeRefOrGetter<EnvWithCluster | undefined>) {
  return computed(() => isFederationEnv(toValue(source)));
}
