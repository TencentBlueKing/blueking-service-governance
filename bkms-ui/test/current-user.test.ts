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

import { describe, expect, it } from 'vitest';
import { normalizeCurrentUser } from '~/api/modules/user';

describe('normalizeCurrentUser', () => {
  it('reads the camelCase account payload', () => {
    expect(
      normalizeCurrentUser({
        userID: 'alice',
        tenantID: 'system',
        timeZone: 'Asia/Shanghai',
        personalCenterURL: 'https://bkuser.example.com/personal-center',
      }),
    ).toEqual({
      user_id: 'alice',
      tenant_id: 'system',
      timeZone: 'Asia/Shanghai',
      personalCenterURL: 'https://bkuser.example.com/personal-center',
    });
  });

  it('keeps the legacy snake_case payload usable', () => {
    expect(normalizeCurrentUser({ user_id: 'alice', tenant_id: 'tenant-a' })).toEqual({
      user_id: 'alice',
      tenant_id: 'tenant-a',
    });
  });

  it('keeps an empty tenant and drops blank optional urls', () => {
    expect(
      normalizeCurrentUser({
        userID: 'alice',
        tenantID: '',
        timeZone: '  ',
        personalCenterURL: '',
      }),
    ).toEqual({
      user_id: 'alice',
      tenant_id: '',
    });
  });
});
