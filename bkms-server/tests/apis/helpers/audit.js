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
const {authHeaders} = require('../common');
const {pollUntil} = require('./poll');

/**
 * Waits until the operation records queried by `req` stop growing.
 *
 * Handlers write operation records in a goroutine after responding, so a
 * record query issued right after the last write may miss it. Instead of a
 * fixed sleep, this polls the same URL until the total count is non-zero and
 * unchanged between two consecutive rounds; `intervalMs` must therefore
 * exceed the time a single record write takes.
 *
 * @param {Object} req - Bruno request object of the operation-records query.
 * @param {Object} [options]
 * @param {number} [options.intervalMs=300] - Sleep between rounds.
 * @param {number} [options.timeoutMs=10000] - Total wait budget.
 * @returns {Promise<void>}
 */
async function waitForOperationRecords(req, {intervalMs = 300, timeoutMs = 10000} = {}) {
  const url = bru.interpolate(req.getUrl());
  let previous = -1;
  await pollUntil({
    fn: async () => {
      const response = await axios.get(url, {headers: authHeaders()});
      return response.data?.data?.count ?? 0;
    },
    predicate: (count) => {
      const settled = count > 0 && count === previous;
      previous = count;
      return settled;
    },
    intervalMs,
    timeoutMs,
    label: 'operation records to settle'
  });
}

module.exports = {
  waitForOperationRecords
};
