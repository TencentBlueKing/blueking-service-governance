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
  /** 按属性名缓存 mock 实例：同名方法稳定可断言调用次数（每次访问新建 spy 则永远断言到 0） */
  const methodMocks = new Map<string, ReturnType<typeof vi.fn>>();
  return new Proxy({} as Record<string, unknown>, {
    get: (_target, prop: string | symbol) => {
      // 过滤 Symbol prop（如 Symbol.toPrimitive / then），防止被当作 thenable 误触发
      if (typeof prop !== 'string') return undefined;
      if (!calledMethods.has(prop)) {
        calledMethods.add(prop);
        console.warn(`[anyService] ${serviceName}.${prop}() 被 Proxy 兜底调用`);
      }
      let fn = methodMocks.get(prop);
      if (!fn) {
        // 每次调用 resolve 独立副本（structuredClone 在调用时执行），防用例 mutate 串染
        fn = vi.fn(() => Promise.resolve(structuredClone(defaultReturn)));
        methodMocks.set(prop, fn);
      }
      return fn;
    },
  });
}
