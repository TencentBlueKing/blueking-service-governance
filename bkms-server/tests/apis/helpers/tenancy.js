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

const axios = require('axios');

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

/**
 * Reports whether the multi-tenant cases in `tenancy/` should run.
 *
 * @returns {boolean}
 */
function tenancyEnabled() {
  return Boolean(bru.getProcessEnv('RUN_MULTI_TENANT_API_TESTS'));
}

/**
 * Sends a request as the given test tenant from a script and returns the
 * response body, throwing on any non-200 status.
 *
 * @param {"A"|"B"} party
 * @param {string} method
 * @param {string} path - Path under `serviceURL`.
 * @param {Object} [data] - JSON body.
 * @returns {Promise<Object>}
 */
async function callAs(party, method, path, data) {
  const user = bru.getVar(`tenancyUser${party}`);
  const tenant = bru.getVar(`tenancyTenant${party}`);
  const response = await axios({
    method,
    url: `${bru.getEnvVar('serviceURL')}${path}`,
    data,
    headers: {
      'X-Bk-Authed-User-Info': JSON.stringify({userId: user, tenantId: tenant}),
      'X-Bk-Authed-User-Credential': '{"bkTicket": "xxx"}',
      'X-Bk-Tenant-Id': tenant
    },
    validateStatus: () => true
  });
  if (response.status !== 200) {
    throw new Error(`${method.toUpperCase()} ${path} as tenant ${party}: ${response.status} ` +
      JSON.stringify(response.data));
  }
  return response.data;
}

function helmAppBody(id) {
  return {
    name: id,
    id,
    type: 'helm',
    buildConfig: {
      sourceType: 'imageRegistry',
      pipelineType: '123',
      imageBuildConfig: {name: 'api-test-app'}
    },
    helmSpec: {
      helmSource: {
        repoType: 'HelmRepo',
        helmRepoConfig: {repoURL: 'http://www.example.com/foo', chartName: 'foobar'}
      }
    }
  };
}

/**
 * Creates the data the tenancy cases probe, and records its IDs in runtime
 * variables:
 *
 * - tenant A: a workspace (with its default `test` env), a tRPC app with an
 *   app env var, a workspace-scoped env var, a component def, and a helm app
 *   with a config file whose content is `owner: tenant-a`;
 * - tenant B: its own workspace and helm app, so B can pass the app check and
 *   reach the config file store with A's file ID.
 *
 * The collection script regenerates `randomSuffix` on every request, so the
 * suffix is read once here. Does nothing when the tenancy cases are disabled,
 * since a folder-level `skipRequest` does not stop the request's own script.
 *
 * @returns {Promise<void>}
 */
async function setupTenancyData() {
  if (!tenancyEnabled()) {
    return;
  }
  const suffix = bru.getVar('randomSuffix');

  const ws = await callAs('A', 'post', '/workspaces', {
    id: `tenancy-ws-${suffix}`,
    displayName: `多租户隔离 ${suffix}`,
    description: '多租户隔离 API 测试',
    bkCIProjectID: `tenancy-bkci-${suffix}`
  });
  const wsID = ws.data.id;
  bru.setVar('tenancyWorkspaceID', wsID);

  const app = await callAs('A', 'post', `/workspaces/${wsID}/apps`, {
    name: `tenancy-app-${suffix}`,
    id: `tenancy-app-${suffix}`,
    type: 'trpc',
    appModelSpec: {
      trpcSpec: {
        language: 'go',
        fileName: 'trpc_go.yaml',
        filePath: '/usr/local/trpc/bin/',
        fileContent: 'dGVzdAo='
      }
    },
    buildConfig: {
      sourceType: 'pipeline',
      pipelineType: 'pipeline',
      pipelineBuildConfig: {pipelineID: 'p-tenancy-pipeline', params: {}}
    }
  });
  const appID = app.data.id;
  bru.setVar('tenancyAppID', appID);

  const envs = await callAs('A', 'get', `/workspaces/${wsID}/envs`);
  const env = (envs.data || []).find((e) => e.name === 'test');
  if (!env) {
    throw new Error(`default test env not found in workspace ${wsID}: ${JSON.stringify(envs.data)}`);
  }
  bru.setVar('tenancyEnvID', env.id);
  bru.setVar('tenancyEnvDisplayName', env.displayName);

  const scopedVar = await callAs('A', 'post', `/workspaces/${wsID}/scoped-env-vars`, {
    scopeType: 'workspace',
    scopeValue: '',
    key: `TENANCY_SCOPED_${suffix}`,
    value: 'tenant-a-value',
    description: '多租户隔离 API 测试',
    isSensitive: false
  });
  bru.setVar('tenancyScopedEnvVarID', scopedVar.data.id);

  const appVar = await callAs('A', 'post', `/apps/${appID}/env-vars`, {
    key: `TENANCY_APP_${suffix}`,
    value: 'tenant-a-value',
    description: '多租户隔离 API 测试',
    isSensitive: false
  });
  bru.setVar('tenancyAppEnvVarKey', appVar.data.key);

  const compDefName = `tenancy-comp-${suffix}`;
  await callAs('A', 'post', '/component-defs', {
    compDefName,
    displayName: 'Tenancy Component A',
    description: 'created by tenant A',
    properties: [
      {name: 'serviceName', type: 'STRING', defaultValue: 'my-service', description: 'Service name'}
    ],
    patchers: [
      'spec:\n  template:\n    spec:\n      containers:\n        - name: "{{ .bkmsContainerName }}"\n' +
      '          env:\n            - name: SERVICE_NAME\n              value: "{{ .serviceName }}"'
    ],
    specs: [
      'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "{{ .name }}-config"\n' +
      'data:\n  serviceName: "{{ .serviceName }}"'
    ],
    scopeType: 'global',
    scopeWorkspaceIDs: []
  });
  bru.setVar('tenancyCompDefName', compDefName);

  // tRPC apps allow only one framework config file, so config files live on a helm app
  const helmA = await callAs('A', 'post', `/workspaces/${wsID}/apps`, helmAppBody(`tenancy-helm-a-${suffix}`));
  const helmAppIDA = helmA.data.id;
  bru.setVar('tenancyHelmAppIDA', helmAppIDA);

  const file = await callAs('A', 'post', `/apps/${helmAppIDA}/app-config-files`, {
    name: `tenancy-cfg-${suffix}`,
    type: 'normal',
    contentSourceType: 'local',
    fileFormat: 'yaml',
    versionDescription: '初始版本'
  });
  const fileID = file.item.id;
  bru.setVar('tenancyConfigFileID', fileID);
  await callAs('A', 'put', `/apps/${helmAppIDA}/app-config-files/${fileID}/content`, {
    content: 'owner: tenant-a',
    versionDescription: '租户 A 写入'
  });

  const wsB = await callAs('B', 'post', '/workspaces', {
    id: `tenancy-ws-b-${suffix}`,
    displayName: `多租户隔离 B ${suffix}`,
    description: '多租户隔离 API 测试',
    bkCIProjectID: `tenancy-bkci-b-${suffix}`
  });
  const wsIDB = wsB.data.id;
  bru.setVar('tenancyWorkspaceIDB', wsIDB);

  const helmB = await callAs('B', 'post', `/workspaces/${wsIDB}/apps`, helmAppBody(`tenancy-helm-b-${suffix}`));
  bru.setVar('tenancyHelmAppIDB', helmB.data.id);
}

/**
 * Deletes what `setupTenancyData` created. Every step is attempted even if
 * an earlier one fails, so a partial setup is still cleaned up; failures are
 * reported together at the end. Does nothing when the tenancy cases are
 * disabled, as with `setupTenancyData`.
 *
 * @returns {Promise<void>}
 */
async function cleanupTenancyData() {
  if (!tenancyEnabled()) {
    return;
  }
  const steps = [
    ['A', 'tenancyCompDefName', (v) => `/component-defs/${v}`],
    ['A', 'tenancyHelmAppIDA', (v) => `/apps/${v}`],
    ['A', 'tenancyAppID', (v) => `/apps/${v}`],
    ['A', 'tenancyWorkspaceID', (v) => `/workspaces/${v}`],
    ['B', 'tenancyHelmAppIDB', (v) => `/apps/${v}`],
    ['B', 'tenancyWorkspaceIDB', (v) => `/workspaces/${v}`]
  ];
  const errors = [];
  for (const [party, varName, path] of steps) {
    const value = bru.getVar(varName);
    if (!value) {
      continue;
    }
    try {
      await callAs(party, 'delete', path(value));
    } catch (err) {
      errors.push(err.message);
    }
  }
  if (errors.length > 0) {
    throw new Error(`tenancy cleanup failed:\n${errors.join('\n')}`);
  }
}

module.exports = {
  actAs,
  cleanupTenancyData,
  setupTenancyData,
  tenancyEnabled,
  userInfo
};
