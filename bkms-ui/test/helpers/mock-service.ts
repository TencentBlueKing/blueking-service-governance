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

import { vi } from 'vitest';

/**
 * Proxy 兜底 service mock（带 warning 追踪）。
 *
 * 所有方法调用均返回 resolved 空值，但**每个方法首次被调用时**会打印 console.warn，
 * 方便排查是否有未预期的 API 被默默吞掉。
 *
 * @param serviceName - 服务名称，会出现在 warning 日志中，便于定位来源
 * @param defaultReturn - 每个方法 resolve 的默认值
 * @returns proxy 对象（直接用作 vi.mock 的返回值）
 *
 * 使用方式：
 *   vi.mock('~/api/modules/v1', async () => {
 *     const { createAnyServiceMock } = await import('../helpers/mock-service');
 *     return { BuildsService: createAnyServiceMock('BuildsService') };
 *   });
 */
export function createAnyServiceMock(serviceName = 'UnnamedService', defaultReturn: unknown = { list: [], total: 0 }) {
  /** 记录已调用过的方法名，同名方法只 warn 一次 */
  const calledMethods = new Set<string>();
  return new Proxy({} as Record<string, unknown>, {
    get: (_target, prop: string | symbol) => {
      // 过滤 Symbol prop（如 Symbol.toPrimitive / then），防止被当作 thenable 误触发
      if (typeof prop !== 'string') return undefined;
      if (!calledMethods.has(prop)) {
        calledMethods.add(prop);
        console.warn(`[anyService] ${serviceName}.${prop}() 被 Proxy 兜底调用`);
      }
      return vi.fn().mockResolvedValue(defaultReturn);
    },
  });
}
