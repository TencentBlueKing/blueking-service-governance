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
 * Generic polling utility used by the API tests.
 *
 * Bruno's stateless test cases routinely need to wait for asynchronous
 * server-side state transitions (e.g. helm deploy reaching `deployed`).
 * `pollUntil` repeatedly invokes `fn` and returns the first observation
 * for which `predicate(value)` is truthy. On timeout, the most recent
 * observation is embedded in the error message to aid debugging.
 *
 * @template T
 * @param {Object} options
 * @param {() => Promise<T>} options.fn - Async observer invoked each round.
 * @param {(value: T) => boolean} options.predicate - Returns true to stop polling.
 * @param {number} [options.timeoutMs=120000] - Total wait budget.
 * @param {number} [options.intervalMs=1000] - Sleep between rounds.
 * @param {string} [options.label='pollUntil'] - Label used in timeout errors.
 * @returns {Promise<T>}
 */
async function pollUntil({fn, predicate, timeoutMs = 120000, intervalMs = 1000, label = 'pollUntil'}) {
  if (typeof fn !== 'function') {
    throw new Error('pollUntil: `fn` must be a function');
  }
  if (typeof predicate !== 'function') {
    throw new Error('pollUntil: `predicate` must be a function');
  }

  const deadline = Date.now() + timeoutMs;
  let lastValue;
  let lastError;

  // eslint-disable-next-line no-constant-condition
  while (true) {
    try {
      lastValue = await fn();
      lastError = undefined;
      if (predicate(lastValue)) {
        return lastValue;
      }
    } catch (err) {
      // Swallow per-round errors — common for "resource not yet visible"
      // scenarios. The final timeout error includes the last error message.
      lastError = err;
    }
    if (Date.now() >= deadline) {
      const lastObs = lastError
        ? `last error: ${lastError.message}`
        : `last value: ${safeStringify(lastValue)}`;
      throw new Error(`Timed out (${timeoutMs}ms) waiting for ${label}; ${lastObs}`);
    }
    await sleep(intervalMs);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function safeStringify(value) {
  try {
    return JSON.stringify(value);
  } catch (_err) {
    return String(value);
  }
}

module.exports = {
  pollUntil
};
