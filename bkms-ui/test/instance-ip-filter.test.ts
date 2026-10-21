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
 */

import { describe, expect, it } from 'vitest';
import { matchesInstanceIPFilter } from '~/pages/application/detail/deploy/instance-list/instance-utils';

import type { AppInstanceOutputObj } from '~/@types/v1/instance';

const instance = (ip?: string, nodeIP?: string) => {
  const value: AppInstanceOutputObj = { ip, nodeIP };
  return value;
};

describe('实例列表 IP 筛选', () => {
  it('未传筛选值时不过滤', () => {
    expect(matchesInstanceIPFilter(instance('10.0.0.1', '192.168.0.1'))).toBe(true);
  });

  it('Pod IP 使用单值精确匹配，不命中前缀或包含值', () => {
    expect(matchesInstanceIPFilter(instance('10.0.0.2'), ['10.0.0.2'])).toBe(true);
    expect(matchesInstanceIPFilter(instance('10.0.0.10'), ['10.0.0.1'])).toBe(false);
  });

  it('Node IP 使用单值精确匹配', () => {
    expect(matchesInstanceIPFilter(instance(undefined, '192.168.0.2'), [], ['192.168.0.2'])).toBe(true);
  });

  it('Pod IP 与 Node IP 之间按 OR', () => {
    expect(matchesInstanceIPFilter(instance('10.0.0.9', '192.168.0.2'), ['10.0.0.1'], ['192.168.0.2'])).toBe(true);
    expect(matchesInstanceIPFilter(instance('10.0.0.9', '192.168.0.9'), ['10.0.0.1'], ['192.168.0.2'])).toBe(false);
  });
});
