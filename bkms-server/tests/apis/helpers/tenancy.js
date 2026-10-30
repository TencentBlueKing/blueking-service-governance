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
 * Builds the `X-Bk-Authed-User-Info` header for the whole collection.
 *
 * When `BKMS_API_TEST_TENANT_ID` is set (see collection.bru), the user carries
 * that tenant so a multi-tenant server accepts it without asking bk-user;
 * otherwise the header is unchanged for single-tenant runs.
 *
 * @param {string} userId
 * @returns {string}
 */
function userInfo(userId) {
  const tenantId = bru.getVar("apiTestTenantID");
  return JSON.stringify(tenantId ? {userId, tenantId} : {userId});
}

/**
 * Identity helpers for the multi-tenant API tests in `tenancy/`.
 *
 * The user's own tenant goes into `X-Bk-Authed-User-Info` (what the login
 * state would carry), while `X-Bk-Tenant-Id` is the tenant the request asks
 * to act in. Passing a different `headerTenant` simulates a forged header.
 *
 * @param {Object} req - Bruno request object.
 * @param {"A"|"B"} party - Which test tenant's user sends the request.
 * @param {string} [headerTenant] - Tenant header; defaults to the user's tenant.
 */
function actAs(req, party, headerTenant) {
  const user = bru.getVar(`tenancyUser${party}`);
  const tenant = bru.getVar(`tenancyTenant${party}`);
  req.setHeader("X-Bk-Authed-User-Info", JSON.stringify({userId: user, tenantId: tenant}));
  req.setHeader("X-Bk-Tenant-Id", headerTenant || tenant);
}

module.exports = {
  actAs,
  userInfo
};
