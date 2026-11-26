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

import { APP_FRAMEWORKS } from '~/common/app-types';

import type { TemplateOutputObj } from '~/@types/v1/framework-config-templates';

/** 框架配置模板支持的语言（tRPC 按语言区分，TAF 固定 cpp）。 */
export const TEMPLATE_LANGUAGES = ['go', 'cpp'] as const;

export type TemplateLanguage = (typeof TEMPLATE_LANGUAGES)[number];

/** 模板语言展示名。 */
export const TEMPLATE_LANGUAGE_LABELS: Record<TemplateLanguage, string> = {
  cpp: 'C++',
  go: 'Go',
};

/** 模板来源：平台内置 / 空间自定义。 */
export const TEMPLATE_SOURCES = {
  PLATFORM: 'platform',
  WORKSPACE: 'workspace',
} as const;

export type FrameworkAppType = FrameworkType;

export interface FrameworkTemplateRecord {
  appType: FrameworkAppType;
  createdAt: string;
  fileContent: string;
  fileName: string;
  id: string;
  isDefault: boolean;
  language: '' | TemplateLanguage;
  name: string;
  source: FrameworkTemplateSource;
  updatedAt?: string;
  updater?: string;
}

export type FrameworkTemplateSource = 'platform' | 'workspace';

export type FrameworkType = 'taf' | 'trpc';

/** 新建模板时的默认配置文件名。 */
export function defaultTemplateFileName(appType: FrameworkAppType, language: string): string {
  if (appType === APP_FRAMEWORKS.TAF) return 'config.conf';
  return `trpc_${language || 'go'}.yaml`;
}

/** TAF 模板语言固定为 C++，tRPC 模板默认 Go。 */
export function defaultTemplateLanguage(appType: FrameworkAppType): TemplateLanguage {
  return appType === APP_FRAMEWORKS.TAF ? 'cpp' : 'go';
}

/**
 * TemplateOutputObj → 页面记录的统一映射：
 * 语言收窄为 go/cpp（非法值回退到传入的 fallback）、builtin → source。
 * 列表页与模板编辑器共用，避免映射逻辑分散漂移。
 */
export function normalizeTemplate(
  template: TemplateOutputObj,
  languageFallback: '' | TemplateLanguage = '',
): FrameworkTemplateRecord {
  const language = template.language === 'go' || template.language === 'cpp' ? template.language : languageFallback;
  return {
    appType: template.framework === APP_FRAMEWORKS.TAF ? 'taf' : 'trpc',
    createdAt: template.createdAt || '',
    fileContent: template.content || '',
    fileName: template.fileName || '',
    id: template.id || '',
    isDefault: Boolean(template.isDefault),
    language,
    name: template.name || '',
    source: template.builtin ? TEMPLATE_SOURCES.PLATFORM : TEMPLATE_SOURCES.WORKSPACE,
    updatedAt: template.updatedAt,
    updater: template.updater,
  };
}
