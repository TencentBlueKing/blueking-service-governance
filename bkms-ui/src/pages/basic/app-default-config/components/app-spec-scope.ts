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

/** 适用范围类型 */
export type AppSpecScopeType = 'all' | 'env_type' | 'specific_envs';

/** 侧滑打开场景：菜单新增配置项 / 卡片新增规则 / 编辑规则 */
export type AppSpecSliderScene = 'addConfig' | 'createRule' | 'editRule';

/** 全部环境类型 */
export const ALL_ENV_TYPES = ['development', 'test', 'staging', 'production'];

/** 开发模式规则可选的环境类型（production 不支持开发模式） */
export const DEV_MODE_SUPPORTED_ENV_TYPES = ['development', 'test', 'staging'];

export interface ScopeEnvTypesParams {
  envTypes: string[];
  occupiedEnvTypes: string[];
  scopeType: AppSpecScopeType;
  supportedEnvTypes: string[];
}

/**
 * 解析当前适用范围下实际生效的环境类型列表。
 * - scopeType 为 all：取 supportedEnvTypes
 * - scopeType 为 env_type：取 envTypes
 * 统一去重，并过滤掉不支持、已被其他规则占用的类型。
 */
export function getScopeSubmitEnvTypes(params: ScopeEnvTypesParams) {
  const { scopeType, supportedEnvTypes, envTypes, occupiedEnvTypes } = params;
  const occupiedSet = new Set(occupiedEnvTypes);
  const supportedSet = new Set(supportedEnvTypes);
  const selectedEnvTypes = scopeType === 'all' ? supportedEnvTypes : envTypes;

  return Array.from(new Set(selectedEnvTypes)).filter(
    envType => supportedSet.has(envType) && !occupiedSet.has(envType),
  );
}
