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

import { getActivePinia } from 'pinia';
import { useFrontendFeaturesStore } from '~/stores/frontend-features';

export function getBkUserApiUrl() {
  const template = import.meta.env.BK_API_URL_TMPL?.trim().replace(/\/+$/, '');
  if (!template) return '';
  if (import.meta.env.DEV) return '/api-bk-user-selector';
  return template.includes('{api_name}') ? `${template.replace('{api_name}', 'bk-user-web')}/prod` : template;
}

/** 多租户模式且已配置用户网关、当前用户有租户时返回姓名组件配置，否则返回 undefined 沿用原文本 */
export function getUserDisplayConfig(tenantId?: string) {
  return isMultiTenant(tenantId) ? { tenantId, apiBaseUrl: getBkUserApiUrl() } : undefined;
}

/**
 * 是否按多租户处理当前用户：部署开关 enableMultiTenantMode 为 true，且已配置用户网关、租户 ID 非空。
 * 人员展示、远程搜索、批量解析共用此判定。
 */
export function isMultiTenant(tenantId?: string): tenantId is string {
  return isMultiTenantModeEnabled() && !!getBkUserApiUrl() && !!tenantId;
}

function isMultiTenantModeEnabled() {
  if (!getActivePinia()) return false;
  return useFrontendFeaturesStore().enableMultiTenantMode;
}
