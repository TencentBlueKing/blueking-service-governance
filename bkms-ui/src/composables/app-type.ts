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

/**
 * 应用类型统一定义
 *
 * 新增应用类型只需修改此文件，其他地方自动同步。
 */

/** 应用类型常量 */
export const APP_TYPES = {
  HELM: 'helm',
  AGONES: 'agones',
  TRPC: 'trpc',
  TAF: 'taf',
} as const;

/** 应用模型类型（trpc / taf） */
export type AppModelAppType = typeof APP_TYPES.TAF | typeof APP_TYPES.TRPC;

/** 应用类型（不含空字符串，用于导航、图标等场景） */
export type AppType = Exclude<IAppType, ''>;

/** Helm-like 应用类型（agones 复用 helm 流程） */
export type HelmLikeAppType = typeof APP_TYPES.AGONES | typeof APP_TYPES.HELM;

/** 应用类型（含空字符串，用于 store 等初始状态场景） */
export type IAppType = '' | (typeof APP_TYPES)[keyof typeof APP_TYPES];

/** 应用模型类型集合（trpc / taf） */
export const APP_MODEL_APP_TYPES: readonly AppModelAppType[] = [APP_TYPES.TRPC, APP_TYPES.TAF] as const;

/** Helm-like 应用类型集合（agones 复用 helm 流程） */
export const HELM_LIKE_APP_TYPES: readonly HelmLikeAppType[] = [APP_TYPES.HELM, APP_TYPES.AGONES] as const;

/** 判断是否为应用模型类型（trpc / taf） */
export function isAppModelAppType(type: null | string | undefined): type is AppModelAppType {
  return !!type && APP_MODEL_APP_TYPES.includes(type as AppModelAppType);
}

/** 判断是否为 Helm-like 应用类型（helm / agones） */
export function isHelmLikeAppType(type: null | string | undefined): type is HelmLikeAppType {
  return !!type && HELM_LIKE_APP_TYPES.includes(type as HelmLikeAppType);
}
