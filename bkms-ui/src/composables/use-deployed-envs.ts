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

import { envTypeMap, envTypeTagClassMap } from './use-env-manager';

/** 特性环境类别取值，与接口 envKind 保持一致 */
export const FEATURE_ENV_KIND = 'feature';

/** 标准环境类型展示顺序 */
export const DEPLOYED_ENV_TYPES = ['development', 'test', 'staging', 'production'] as const;

/** 镜像与 Helm Chart 列表返回的已部署环境最小结构 */
export interface DeployedEnvItem {
  envDisplayName?: string;
  envKind?: string;
  envName?: string;
  envType?: string;
}

/** 「已部署环境」列的分组结果 */
export interface DeployedEnvTag {
  count: number;
  /** 组内环境的展示名称，用于悬浮提示 */
  displayNames: string[];
  key: DeployedEnvType;
  label: string;
  tagClass: string;
}

export type DeployedEnvType = (typeof DEPLOYED_ENV_TYPES)[number];

/**
 * 对已部署环境按环境分类归组。特性环境的 envType 是它继承自来源环境的类型，
 * 「已部署环境」列只表达环境分类，因此特性环境与其继承的类型合并计数，不再单列。
 */
export function groupDeployedEnvs(
  deployedEnvs: DeployedEnvItem[] | undefined,
  translate: (envName: string) => string = envName => envName,
): DeployedEnvTag[] {
  if (!deployedEnvs?.length) return [];

  const typeDisplayNames: Record<DeployedEnvType, string[]> = {
    development: [],
    staging: [],
    production: [],
    test: [],
  };

  deployedEnvs.forEach(env => {
    if (!isDeployedEnvType(env.envType)) return;
    typeDisplayNames[env.envType].push(resolveDeployedEnvDisplayName(env, translate));
  });

  return DEPLOYED_ENV_TYPES.filter(envType => typeDisplayNames[envType].length > 0).map(envType => ({
    count: typeDisplayNames[envType].length,
    displayNames: typeDisplayNames[envType],
    key: envType,
    label: envTypeMap[envType]?.name || envType,
    tagClass: envTypeTagClassMap[envType],
  }));
}

/** 判断环境类型是否为可展示的标准类型 */
export function isDeployedEnvType(envType?: string): envType is DeployedEnvType {
  return Boolean(envType) && DEPLOYED_ENV_TYPES.includes(envType as DeployedEnvType);
}

export function isFeatureEnvKind(envKind?: string) {
  return envKind === FEATURE_ENV_KIND;
}

/**
 * 取环境展示名称：优先使用接口返回的展示名，其次由调用方按应用环境列表翻译，最后回退内部名。
 */
export function resolveDeployedEnvDisplayName(
  env: DeployedEnvItem,
  translate: (envName: string) => string = envName => envName,
) {
  if (env.envDisplayName) return env.envDisplayName;
  return env.envName ? translate(env.envName) : '--';
}
