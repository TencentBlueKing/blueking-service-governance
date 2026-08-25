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

/** 两侧都为空视为未配置，不算不一致；一侧为空或数值不等则不一致。 */
export function isQuantityEqual(request?: string, limit?: string, parse: (value?: string) => null | number = Number) {
  const hasRequest = !!request?.trim();
  const hasLimit = !!limit?.trim();
  if (!hasRequest && !hasLimit) return true;
  if (!hasRequest || !hasLimit) return false;
  const requestValue = parse(request);
  const limitValue = parse(limit);
  if (requestValue === null || limitValue === null) return request?.trim() === limit?.trim();
  return requestValue === limitValue;
}

/** CPU 值转核数；兼容毫核（200m = 0.2）。无法解析时返回 null。 */
export function parseCpuCores(value?: string): null | number {
  const normalized = value?.trim();
  if (!normalized) return null;
  const milliCoreMatch = normalized.match(/^(\d+(?:\.\d+)?)m$/);
  if (milliCoreMatch) return Number(milliCoreMatch[1]) / 1000;
  const cores = Number(normalized);
  return Number.isFinite(cores) ? cores : null;
}

const MEMORY_TO_MIB: Record<string, number> = {
  '': 1 / 1024 ** 2,
  Ei: 1024 ** 4,
  Pi: 1024 ** 3,
  Ti: 1024 ** 2,
  Gi: 1024,
  Mi: 1,
  Ki: 1 / 1024,
  E: 1000 ** 6 / 1024 ** 2,
  P: 1000 ** 5 / 1024 ** 2,
  T: 1000 ** 4 / 1024 ** 2,
  G: 1000 ** 3 / 1024 ** 2,
  M: 1000 ** 2 / 1024 ** 2,
  K: 1000 / 1024 ** 2,
};

/** 内存值转 MiB，用于 Requests / Limits 等量比较。无法解析时返回 null。 */
export function parseMemoryToMiB(value?: string): null | number {
  const normalized = value?.trim();
  if (!normalized) return null;
  const match = normalized.match(/^(\d+(?:\.\d+)?)(Ei|Pi|Ti|Gi|Mi|Ki|E|P|T|G|M|K)?$/);
  if (!match) return null;
  const amount = Number(match[1]);
  if (!Number.isFinite(amount)) return null;
  const unit = match[2] || '';
  return amount * (MEMORY_TO_MIB[unit] ?? 1);
}
