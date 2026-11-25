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

import Fetch from '~/api/fetch';

import type { IUser } from '~/@types/api';

const fetch = new Fetch({
  prefix: `${import.meta.env.BK_API_PREFIX}`,
});

const requestCurrentUser = fetch.get<unknown, unknown>('/simple_account/info');

/** 响应体没有 data 包裹，固定取完整 body。 */
export function getUser() {
  return requestCurrentUser(undefined, { needRes: true }).then(normalizeCurrentUser);
}

/**
 * /simple_account/info 现返回驼峰字段，旧环境仍返回下划线字段。
 * 统一成页面已在使用的 user_id / tenant_id，并保留新增的时区和个人中心地址。
 */
export function normalizeCurrentUser(raw: unknown): IUser {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {};
  const tenant = source.tenantID ?? source.tenant_id;
  const timeZone = readText(source, 'timeZone', 'time_zone');
  const personalCenterURL = readText(source, 'personalCenterURL', 'personal_center_url');

  return {
    user_id: readText(source, 'userID', 'user_id'),
    ...(typeof tenant === 'string' ? { tenant_id: tenant.trim() } : {}),
    ...(timeZone ? { timeZone } : {}),
    ...(personalCenterURL ? { personalCenterURL } : {}),
  };
}

function readText(source: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = source[key];
    if (typeof value === 'string' && value.trim()) return value.trim();
  }
  return '';
}
