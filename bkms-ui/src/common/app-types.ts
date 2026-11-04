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

/** 应用管理类型，trpc / taf 为存量类型。 */
export const APP_TYPES = {
  HELM: 'helm',
  AGONES: 'agones',
  TRPC: 'trpc',
  TAF: 'taf',
  BKMS_APP: 'bkmsApp',
} as const;

/** 平台托管应用的框架，与应用管理类型分开维护。 */
export const APP_FRAMEWORKS = {
  BLANK: 'blank',
  TRPC: 'trpc',
  TAF: 'taf',
} as const;

/** 展示兼容语言；canCreateBlank 仅控制 blank 创建入口，实际组合仍须由后端目录校验。 */
export const APP_LANGUAGES = [
  { id: 'go', label: 'Go', description: '适合 Gin、Echo 等 Web 服务', canCreateBlank: true },
  { id: 'cpp', label: 'C++', description: '', canCreateBlank: false },
  { id: 'python', label: 'Python', description: '适合数据服务与 AI 应用', canCreateBlank: true },
  { id: 'nodejs', label: 'Node.js', description: '适合接口与前后端一体服务', canCreateBlank: true },
  { id: 'java', label: 'Java', description: '', canCreateBlank: false },
] as const;

// 首版 blank 创建入口仅开放 Go、Python、Node.js，展示范围保持兼容。
export const BLANK_LANGUAGES = APP_LANGUAGES.filter(item => item.canCreateBlank).map(item => item.id);

/** 模板卡片与创建摘要共用的名称、描述。 */
export const APP_TEMPLATES = [
  {
    id: APP_FRAMEWORKS.TRPC,
    label: 'tRPC',
    description: '适用于使用 tRPC 协议的服务',
  },
  {
    id: APP_FRAMEWORKS.TAF,
    label: 'TAF',
    description: '采用 TAF 框架实现的服务端应用',
  },
  {
    id: APP_TYPES.HELM,
    label: 'Helm Chart',
    description: '自定义应用编排，拥有灵活的可定制性',
  },
  {
    id: APP_TYPES.AGONES,
    label: 'Agones',
    description: '适用于多人在线游戏的专用服务器编排',
  },
] as const;
