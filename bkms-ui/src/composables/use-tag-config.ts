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

import type { TagConfig } from '~/@types/build';

/**
 * 获取 TagConfig 的显示文本
 * @param tagConfig - TagConfig 配置对象
 * @param t - i18n 翻译函数
 * @returns 显示文本
 */
export function getTagConfigDisplayText(tagConfig: null | TagConfig | undefined, t: (key: string) => string): string {
  if (!tagConfig?.type) return t('未开启');

  if (tagConfig.type === 'semver') {
    return t('语义化版本（格式：v1.0.0）');
  }

  if (tagConfig.type === 'custom') {
    const { prefix, withRevision, withBuildTime } = tagConfig.customOpts || {};
    const parts = [prefix, withRevision && `{${t('分支/Tag')}}`, withBuildTime && t('构建时间')].filter(Boolean);
    return parts.join('-') || t('自定义版本');
  }

  return '--';
}

/**
 * 标准化 tagConfig 值，当 type 为空字符串或不存在时返回 null。
 * @param tagConfig - TagConfig 配置对象
 * @returns 有效的 TagConfig 或 null
 */
export function normalizeTagConfig(tagConfig: null | TagConfig | undefined): null | TagConfig {
  if (!tagConfig?.type) return null;
  return tagConfig;
}
