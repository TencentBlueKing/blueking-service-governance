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
 * 共享路由 mock 工厂。
 *
 * 使用方式（vi.mock 内须用动态 import 避免 hoisting TDZ）：
 *   vi.mock('vue-router', async () => {
 *     const { createRouterMock } = await import('../helpers/mock-router');
 *     return createRouterMock({ route: { path: '/ws-1/app', name: 'app', params: { space: 'ws-1' } } });
 *   });
 *
 * 需要断言 push 时：
 *   const mocks = vi.hoisted(() => ({ push: vi.fn() }));
 *   vi.mock('vue-router', async () => {
 *     const { createRouterMock } = await import('../helpers/mock-router');
 *     return createRouterMock({ route: { ... }, push: mocks.push });
 *   });
 */

/** 路由属性覆写，所有字段可选 */
interface RouteOverrides {
  fullPath?: string;
  matched?: unknown[];
  meta?: Record<string, unknown>;
  name?: string;
  params?: Record<string, string>;
  path?: string;
  query?: Record<string, string>;
}

interface RouterMockOptions {
  /** 自定义 push mock（需要在用例中断言时传入 vi.hoisted 中的 vi.fn()） */
  push?: ReturnType<typeof vi.fn>;
  /** 自定义 replace mock */
  replace?: ReturnType<typeof vi.fn>;
  /** 路由属性覆写 */
  route?: RouteOverrides;
}

/**
 * 创建 vue-router mock 结果对象。
 * 内部使用 vi.importActual 获取原始模块，无需外部传入 importOriginal。
 */
export async function createRouterMock(opts?: RouterMockOptions) {
  const { push, replace, route: routeOverrides } = opts ?? {};

  const route: Record<string, unknown> = {
    path: '/',
    fullPath: '/',
    name: '',
    query: {},
    params: {},
    meta: {},
    matched: [],
    ...routeOverrides,
  };
  // 若未显式指定 fullPath，自动与 path 同步
  if (!routeOverrides?.fullPath) route.fullPath = route.path;

  return {
    ...((await vi.importActual('vue-router')) as object),
    useRouter: () => ({
      push: push ?? vi.fn(),
      replace: replace ?? vi.fn(),
      resolve: vi.fn().mockReturnValue({ href: '' }),
      // 暴露完整 route 对象，避免读 path/params/name 时静默拿到 undefined
      currentRoute: { value: route },
    }),
    useRoute: () => route,
  };
}
