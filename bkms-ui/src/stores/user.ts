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

import { computed, ref } from 'vue';

import { defineStore } from 'pinia';
import { getUser } from '~/api/modules/user';
import { AccountService } from '~/api/modules/v1/account';

import type { IUser } from '~/@types/api';
import type { RoleInfo } from '~/@types/v1/account';

export const useUserStore = defineStore('user', () => {
  const userInfo = ref<IUser>({
    user_id: '',
  });
  const roleInfo = ref<null | RoleInfo>(null);

  // 租户 ID
  const tenantId = computed(() => userInfo.value.tenant_id?.trim() || '');
  const currentUsername = computed(() => roleInfo.value?.username || userInfo.value.user_id || '');
  const hasPlatformRole = computed(() => !!roleInfo.value?.platRoleCode);

  function setUserInfo(user: IUser) {
    userInfo.value = user;
    // 图表组件读 window.timezone。接口没给时区时保留 main.ts 里的浏览器兜底。
    if (user.timeZone) window.timezone = user.timeZone;
  }

  async function getUserInfo() {
    const user = await getUser().catch(() => ({ user_id: '' }));
    setUserInfo(user);
  }

  async function getRoleInfo() {
    roleInfo.value = await AccountService.getRole(undefined, { interceptorErr: false }).catch(() => null);
  }

  return {
    userInfo,
    roleInfo,
    tenantId,
    currentUsername,
    hasPlatformRole,
    getUserInfo,
    getRoleInfo,
    setUserInfo,
  };
});
