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

import type { PlatformBuildConfigInput } from '~/@types/v1/app';

export const MAX_PLATFORM_BUILD_ARTIFACT_DIR_LENGTH = 128;

export const ARTIFACT_DIR_SAFE_CHARS_REGEX = /^[A-Za-z0-9._/-]+$/;

export const ARTIFACT_DIR_MESSAGES = {
  ABSOLUTE_PATH: '产物目录必须是绝对路径（以 / 开头）',
  NO_PARENT_SEGMENT: '产物目录不能包含 .. 路径段',
  NOT_ROOT: '产物目录不能是根目录 /',
  SAFE_CHARS: '产物目录仅支持字母、数字、.、_、- 和 /',
  MAX_LENGTH: '产物目录长度不能超过 128 个字符',
} as const;

const ARTIFACT_DIR_RULE_DEFS = [
  {
    message: ARTIFACT_DIR_MESSAGES.ABSOLUTE_PATH,
    check: (value: string) => value.startsWith('/'),
  },
  {
    message: ARTIFACT_DIR_MESSAGES.NO_PARENT_SEGMENT,
    check: (value: string) => !value.split('/').includes('..'),
  },
  {
    message: ARTIFACT_DIR_MESSAGES.NOT_ROOT,
    check: (value: string) => value.split('/').some(segment => segment !== '' && segment !== '.'),
  },
  {
    message: ARTIFACT_DIR_MESSAGES.SAFE_CHARS,
    check: (value: string) => ARTIFACT_DIR_SAFE_CHARS_REGEX.test(value),
  },
  {
    message: ARTIFACT_DIR_MESSAGES.MAX_LENGTH,
    check: (value: string) => value.length <= MAX_PLATFORM_BUILD_ARTIFACT_DIR_LENGTH,
  },
] as const;

/** 创建平台通用构建「产物目录」的表单校验规则（与后端 NormalizePlatformBuildArtifactDir 同口径）。 */
export function createArtifactDirRules(t: (key: string) => string) {
  return ARTIFACT_DIR_RULE_DEFS.map(rule => ({
    message: t(rule.message),
    trigger: 'blur',
    validator: (value?: string) => !value || rule.check(value),
  }));
}

/** 创建空的平台通用构建默认配置对象，供创建页与编辑侧栏共用。 */
export function createPlatformBuildConfig() {
  return {
    artifactDir: '',
    builderImage: '',
    runnerImage: '',
    extraFiles: [] as string[],
    commands: { build: [] as string[], preBuild: [] as string[], runtimeEnv: [] as string[], start: '' },
  } satisfies PlatformBuildConfigInput;
}

/** 格式化只读详情页「产物目录」展示文本：已配置则原样展示，留空则显示语言默认目录。 */
export function formatArtifactDirDisplay(t: (key: string) => string, artifactDir?: string, language?: string): string {
  if (artifactDir) return artifactDir;
  return `${getDefaultArtifactDir(language)}${t('（默认）')}`;
}

/** 按语言返回「编译命令」输入框 placeholder。 */
export function getBuildCommandPlaceholder(t: (key: string) => string, language?: string, appName?: string): string {
  if (language === 'python') {
    return t('留空则自动识别 uv.lock / poetry.lock / requirements.txt 安装依赖');
  }
  return `${t('留空则使用平台默认：')}\ngo build -o \${BKMS_ARTIFACT_DIR}/${appName || '{{ appName }}'} .`;
}

/** 按语言返回「产物目录」默认值，用作输入框 placeholder。 */
export function getDefaultArtifactDir(language?: string): string {
  return language === 'python' ? '/app' : '/out';
}

/** 按单/多阶段返回「编译前置命令」输入框 placeholder。 */
export function getPreBuildCommandPlaceholder(t: (key: string) => string, requiresBuilderImage: boolean): string {
  return requiresBuilderImage
    ? t('在 go build 之前执行，如代码生成、安装工具等')
    : t('在安装依赖或执行构建命令前运行，如安装系统包、生成代码等');
}

/** 按单/多阶段返回「镜像入口命令」输入框 placeholder。 */
export function getStartCommandPlaceholder(t: (key: string) => string, requiresBuilderImage: boolean): string {
  return requiresBuilderImage
    ? t('留空则使用平台默认：运行编译产物二进制')
    : t('留空则沿用运行镜像默认入口，也可在部署时配置启动命令');
}

/** 校验「产物目录」输入；合法（含空串）返回 null，否则返回对应错误提示词条 key。 */
export function validateArtifactDir(value?: string): null | string {
  if (!value) return null;
  return ARTIFACT_DIR_RULE_DEFS.find(rule => !rule.check(value))?.message ?? null;
}
