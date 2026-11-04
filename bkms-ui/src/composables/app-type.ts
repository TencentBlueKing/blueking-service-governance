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
  BKMS_APP: 'bkmsApp',
} as const;

/** 平台托管应用类型（包含存量 trpc / taf） */
export type AppModelAppType = typeof APP_TYPES.BKMS_APP | typeof APP_TYPES.TAF | typeof APP_TYPES.TRPC;

/** 应用类型（不含空字符串，用于导航、图标等场景） */
export type AppType = Exclude<IAppType, ''>;

/** Helm-like 应用类型（agones 复用 helm 流程） */
export type HelmLikeAppType = typeof APP_TYPES.AGONES | typeof APP_TYPES.HELM;

/** 应用类型（含空字符串，用于 store 等初始状态场景） */
export type IAppType = '' | (typeof APP_TYPES)[keyof typeof APP_TYPES];

/** 平台托管应用类型集合 */
export const APP_MODEL_APP_TYPES: readonly AppModelAppType[] = [
  APP_TYPES.TRPC,
  APP_TYPES.TAF,
  APP_TYPES.BKMS_APP,
] as const;

/** Helm-like 应用类型集合（agones 复用 helm 流程） */
export const HELM_LIKE_APP_TYPES: readonly HelmLikeAppType[] = [APP_TYPES.HELM, APP_TYPES.AGONES] as const;

export interface AppPresentationInput {
  framework?: string;
  language?: string;
  type?: string;
}

/** 展示和筛选使用技术栈分类，路由、接口仍使用应用真实 type。 */
export function getAppDisplayKind(app: AppPresentationInput): string {
  if (app.type !== APP_TYPES.BKMS_APP) return app.type || '';
  if (app.framework === 'trpc' || app.framework === 'taf') return app.framework;
  if (app.framework === 'blank' && ['go', 'cpp', 'python', 'nodejs', 'java'].includes(app.language || '')) {
    return app.language || APP_TYPES.BKMS_APP;
  }
  return APP_TYPES.BKMS_APP;
}

/** 返回展示文案；需要翻译的文案由调用组件翻译。 */
export function getAppDisplayLabel(kind: string): string {
  const labels: Record<string, string> = {
    trpc: 'tRPC',
    taf: 'TAF',
    helm: 'Helm Chart',
    agones: 'Agones',
    go: 'Go',
    cpp: 'C++',
    python: 'Python',
    nodejs: 'Node.js',
    java: 'Java',
    bkmsApp: '平台托管应用',
  };
  return labels[kind] || kind;
}

/** 判断是否为平台托管应用类型 */
export function isAppModelAppType(type: null | string | undefined): type is AppModelAppType {
  return !!type && APP_MODEL_APP_TYPES.includes(type as AppModelAppType);
}

/** 判断是否为 Helm-like 应用类型（helm / agones） */
export function isHelmLikeAppType(type: null | string | undefined): type is HelmLikeAppType {
  return !!type && HELM_LIKE_APP_TYPES.includes(type as HelmLikeAppType);
}
