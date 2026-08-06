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
const {AUTH_HEADERS} = require('../common');
const {pollUntil} = require('./poll');

// Mirror of pkg/deploy/appmodel.Status terminal-ish set used by these tests:
// `deployed` is the success terminal, while `failed` / `uninstalled` are
// considered terminal so the poller stops instead of looping forever; callers
// then assert the precise status they expect.
const TERMINAL_STATUSES = ['deployed', 'failed', 'uninstalled'];

function kindToPathSegment(kind) {
  if (kind === 'trpc') return 'trpc-deploys';
  if (kind === 'taf') return 'taf-deploys';
  throw new Error(`unsupported appmodel deploy kind: ${kind}`);
}

function requireField(value, name) {
  if (!value) {
    throw new Error(`${name} is required`);
  }
}

/**
 * List AppModel (trpc/taf) deploy records via API.
 *
 * @param {Object} options
 * @param {('trpc'|'taf')} options.kind - Application kind (required).
 * @param {string} options.appID - Application ID (required).
 * @param {string} options.envName - Environment name (required).
 * @param {string} [options.trafficLaneName] - Traffic lane name filter.
 * @param {number} [options.page=1] - Page index.
 * @param {number} [options.pageSize=10] - Page size.
 * @returns {Promise<{count: number, results: Array<Object>}>}
 */
async function listDeploys(options = {}) {
  requireField(options.appID, 'Application ID');
  requireField(options.envName, 'Environment name');
  const segment = kindToPathSegment(options.kind);

  const serviceURL = bru.getEnvVar('serviceURL');
  const response = await axios.get(
    `${serviceURL}/apps/${options.appID}/envs/${options.envName}/${segment}`,
    {
      headers: AUTH_HEADERS,
      params: {
        page: options.page || 1,
        pageSize: options.pageSize || 10,
        trafficLaneName: options.trafficLaneName || ''
      }
    }
  );
  const data = response.data?.data || {};
  return {
    count: parseInt(data.count || 0, 10),
    results: data.results || []
  };
}

/**
 * Get latest AppModel deploy status (manual-deploy + build-auto-deploy merged).
 */
async function getLatestStatus(options = {}) {
  requireField(options.appID, 'Application ID');
  requireField(options.envName, 'Environment name');
  const segment = kindToPathSegment(options.kind);

  const serviceURL = bru.getEnvVar('serviceURL');
  const response = await axios.get(
    `${serviceURL}/apps/${options.appID}/envs/${options.envName}/${segment}/latest-status`,
    {
      headers: AUTH_HEADERS,
      params: {trafficLaneName: options.trafficLaneName || ''}
    }
  );
  return response.data?.data;
}

/**
 * List resource snapshot meta for a specific deploy record.
 */
async function listResourceSnapshots(options = {}) {
  requireField(options.appID, 'Application ID');
  requireField(options.envName, 'Environment name');
  requireField(options.deployID, 'Deploy ID');
  const segment = kindToPathSegment(options.kind);

  const serviceURL = bru.getEnvVar('serviceURL');
  const response = await axios.get(
    `${serviceURL}/apps/${options.appID}/envs/${options.envName}/${segment}/${options.deployID}/resource-snapshots`,
    {
      headers: AUTH_HEADERS,
      params: {
        page: options.page || 1,
        pageSize: options.pageSize || 10
      }
    }
  );
  const data = response.data?.data || {};
  return {
    count: parseInt(data.count || 0, 10),
    results: data.results || []
  };
}

/**
 * Get one resource snapshot (with manifest) by id.
 */
async function getResourceSnapshot(options = {}) {
  requireField(options.appID, 'Application ID');
  requireField(options.envName, 'Environment name');
  requireField(options.deployID, 'Deploy ID');
  requireField(options.snapshotID, 'Snapshot ID');
  const segment = kindToPathSegment(options.kind);

  const serviceURL = bru.getEnvVar('serviceURL');
  const url = `${serviceURL}/apps/${options.appID}/envs/${options.envName}` +
    `/${segment}/${options.deployID}/resource-snapshots/${options.snapshotID}`;
  const response = await axios.get(url, {headers: AUTH_HEADERS});
  return response.data?.snapshot;
}

/**
 * Wait until the latest AppModel deploy record reaches a terminal state.
 *
 * Matching rules (all optional, AND'd together once the record is terminal):
 *   - `expectedStatus`: status string or array (e.g. 'deployed'). If unset,
 *     any terminal status is accepted.
 *   - `imageTag`: latest record's imageTag must equal this value.
 *   - `excludeID`: latest record's id must differ from this value (i.e. wait
 *     for a NEW record after a previous deploy).
 *
 * The poller treats {deployed, failed, uninstalled} as terminal and stops
 * iterating once a terminal record is observed; callers then assert the
 * precise status they expect (this avoids waiting up to the full timeout
 * when a deploy clearly failed).
 *
 * @param {Object} options
 * @param {('trpc'|'taf')} options.kind - Application kind (required).
 * @param {string} options.appID - Application ID (required).
 * @param {string} options.envName - Environment name (required).
 * @param {string|string[]} [options.expectedStatus]
 * @param {string} [options.imageTag]
 * @param {string} [options.excludeID]
 * @param {string} [options.trafficLaneName]
 * @param {number} [options.timeoutMs=120000]
 * @param {number} [options.intervalMs=1000]
 * @returns {Promise<Object>} the matched record.
 */
async function waitForAppModelDeployTerminal(options = {}) {
  requireField(options.kind, 'Kind');
  const expectedStatuses = options.expectedStatus
    ? (Array.isArray(options.expectedStatus) ? options.expectedStatus : [options.expectedStatus])
    : null;
  const timeoutMs = options.timeoutMs || 120000;
  const label = `appmodel deploy [kind=${options.kind}, app=${options.appID}, env=${options.envName}, ` +
    `status=${expectedStatuses || 'terminal'}]`;

  return pollUntil({
    fn: async () => {
      const {results} = await listDeploys({
        kind: options.kind,
        appID: options.appID,
        envName: options.envName,
        trafficLaneName: options.trafficLaneName
      });
      return results[0];
    },
    predicate: (record) => {
      if (!record) return false;
      if (options.excludeID && record.id === options.excludeID) return false;
      if (options.imageTag && record.imageTag !== options.imageTag) return false;
      // Stop iterating only once the record is in a terminal state.
      if (!TERMINAL_STATUSES.includes(record.status)) return false;
      if (expectedStatuses && !expectedStatuses.includes(record.status)) {
        // Terminal but does not match the expected status — surface it so the
        // caller's assertions can fail fast rather than time out.
        return true;
      }
      return true;
    },
    timeoutMs,
    intervalMs: options.intervalMs || 1000,
    label
  });
}

module.exports = {
  listDeploys,
  getLatestStatus,
  listResourceSnapshots,
  getResourceSnapshot,
  waitForAppModelDeployTerminal,
  TERMINAL_STATUSES
};
