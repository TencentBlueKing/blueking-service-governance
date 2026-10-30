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
 * 场景级测试：应用详情壳层路由守卫（ROUTES 未收录，本头即场景说明）
 *
 * 来源：detail.vue:134 onBeforeRouteUpdate（业务「特性环境支持设置环境变量」引入）
 * V = 3（判定节点 ×2：isSwitchingFeatureEnvApp 五条件与 / isCleanInstanceList 防重定向死循环）
 * 路径（V 主路径 1 条 + 放行等价类 3 条，共 4 用例）：
 *   ① 特性环境视图下切换不同应用 → 重定向 { name: 'detail', query: { activeTab: 'instance' }, replace: true }
 *   ② 目标已是干净 instance 列表（仅 activeTab 一个参数）→ 放行（防 replace 重定向死循环）
 *   ③ 非特性环境视图（view !== 'feature-envs'）下切换应用 → 放行（等价类）
 *   ④ 同应用内的路由更新（name 相同）→ 放行（等价类）
 * stub/mock：bkmsserver 最小集 + v1 Proxy 兜底 + vue-router 捕获守卫回调（守卫逻辑直接调用验证，
 *            真实导航链路归 E2E——指南非目标「整页 E2E」）+ vue-i18n useI18n 直译
 * 变异验证：3 处全部命中（恒真→③④红 / 死循环判定删除→②红 / query 串改→①红），已精确还原
 * backlog：无
 */
import { ref } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AppDetailShell from '~/pages/application/detail.vue';
import { useAppDetail } from '~/stores/app-detail';

import type { VueWrapper } from '@vue/test-utils';
import type { RouteLocationNormalizedGeneric } from 'vue-router';

const mocks = vi.hoisted(() => ({
  /** 捕获 detail.vue 注册的 onBeforeRouteUpdate 守卫回调（每次 mount 清空，取最新注册） */
  routeUpdateGuards: [] as Array<(to: RouteLocationNormalizedGeneric, from: RouteLocationNormalizedGeneric) => unknown>,
  space: '',
  apps: vi.fn(),
  detail: vi.fn(),
  runtime: vi.fn(),
}));

/** 挂载壳层并捕获守卫回调；调用方负责 unmount（§3.8 VTU 须自行清理） */
function mountShell(): {
  guard: NonNullable<(typeof mocks.routeUpdateGuards)[number]>;
  wrapper: VueWrapper;
} {
  const pinia = createPinia();
  setActivePinia(pinia);
  mocks.routeUpdateGuards.length = 0;
  const wrapper = mount(AppDetailShell, {
    shallow: true,
    global: {
      plugins: [pinia],
      // RouterView 由 vue-router mock 提供，但模板按全局组件解析——注册以消除 resolve 告警
      components: { RouterView: { template: '<div />' } },
      stubs: {
        CustomNavigation: { template: '<div><slot /><slot name="side-header" /></div>' },
        Alert: { template: '<div role="alert"><slot name="title" /><slot /></div>' },
        Button: { template: '<button><slot /></button>' },
      },
      mocks: { $t: (text: string) => text },
      // bkui-vue 指令垫片：未装 bkui-vue 全量插件，空指令消除解析警告
      directives: { bkloading: {}, 'bk-tooltips': {} },
    },
  });
  const guard = mocks.routeUpdateGuards.at(-1);
  expect(guard, 'detail.vue 应注册 onBeforeRouteUpdate 守卫').toBeDefined();
  return { wrapper, guard: guard as NonNullable<(typeof mocks.routeUpdateGuards)[number]> };
}

// ── 重依赖 mock：与 height-chain 同源（掐断 API 链，mount 只为注册守卫）──────────

// API mock 工厂见 helpers/mock-api.ts（§3.7 共享 helpers；同款 mock 亦用于 height-chain）
vi.mock('~/api/modules/bkmsserver', async () => {
  const base = (await import('../helpers/mock-api')).bkmsserverShellMockFactory();
  return { ApiServerService: { ...base.ApiServerService, GetApp: mocks.detail } };
});
vi.mock('~/api/modules/v1', async () => (await import('../helpers/mock-api')).v1ProxyMockFactory());
vi.mock('~/api/modules/v1/app', () => ({
  AppService: { listApps: mocks.apps, getAppRuntimeMods: mocks.runtime },
}));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: mocks.space }) }));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.space = '';
  mocks.apps.mockResolvedValue([]);
  mocks.detail.mockResolvedValue(undefined);
  mocks.runtime.mockResolvedValue(undefined);
});

// vue-router：捕获守卫回调（非 no-op——本场景的验证对象就是回调逻辑本身）
vi.mock('vue-router', () => {
  const currentRoute = ref({ params: {}, query: {}, name: '' });
  return {
    RouterView: { template: '<div />' },
    RouterLink: { template: '<a><slot /></a>' },
    useRoute: () => ({ params: {}, query: {}, currentRoute }),
    useRouter: () => ({
      push: vi.fn().mockResolvedValue(undefined),
      replace: vi.fn().mockResolvedValue(undefined),
      currentRoute,
    }),
    onBeforeRouteUpdate: (cb: (typeof mocks.routeUpdateGuards)[number]) => {
      mocks.routeUpdateGuards.push(cb);
    },
    onBeforeRouteLeave: vi.fn(),
    onBeforeRouteEnter: vi.fn(),
  };
});

vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  useI18n: () => ({ t: (s: string) => s, te: () => true }),
}));

/** 构造路由位置（仅填守卫用到的字段） */
function loc(overrides: {
  name?: string;
  params?: Record<string, string>;
  query?: Record<string, string>;
}): RouteLocationNormalizedGeneric {
  return {
    name: overrides.name ?? 'detail',
    params: overrides.params ?? {},
    query: overrides.query ?? {},
    fullPath: '',
    path: '',
    hash: '',
    matched: [],
    meta: {},
    redirectedFrom: undefined,
    href: '',
  } as unknown as RouteLocationNormalizedGeneric;
}

describe('应用详情壳层：特性环境应用切换守卫', () => {
  it('当用户在特性环境视图下切换不同应用时，应重定向到干净的 instance 列表', () => {
    const { wrapper, guard } = mountShell();
    const from = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-a' },
      query: { view: 'feature-envs' },
    });
    const to = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-b' },
      query: { view: 'feature-envs' },
    });

    expect(guard(to, from)).toEqual({
      name: 'detail',
      params: to.params,
      query: { activeTab: 'instance' },
      replace: true,
    });
    wrapper.unmount();
  });

  it('当目标路由已是干净的 instance 列表时，应放行以避免重定向死循环', () => {
    const { wrapper, guard } = mountShell();
    const from = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-a' },
      query: { view: 'feature-envs' },
    });
    const to = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-b' },
      query: { activeTab: 'instance' },
    });

    expect(guard(to, from)).toBeUndefined();
    wrapper.unmount();
  });

  it('当用户在非特性环境视图下切换应用时，应放行不重定向', () => {
    const { wrapper, guard } = mountShell();
    const from = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-a' },
      query: { view: 'overview' },
    });
    const to = loc({ name: 'detail', params: { menuName: 'deployment', name: 'app-b' }, query: { view: 'overview' } });

    expect(guard(to, from)).toBeUndefined();
    wrapper.unmount();
  });

  it('当用户在同一应用内路由更新时，应放行不重定向', () => {
    const { wrapper, guard } = mountShell();
    const from = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-a' },
      query: { view: 'feature-envs' },
    });
    const to = loc({
      name: 'detail',
      params: { menuName: 'deployment', name: 'app-a' },
      query: { view: 'feature-envs' },
    });

    expect(guard(to, from)).toBeUndefined();
    wrapper.unmount();
  });
});

describe('应用详情壳层：能力局部加载', () => {
  it('能力尚在加载时，详情完成即可挂载通用子页面', async () => {
    mocks.space = 'ws-1';
    mocks.apps.mockResolvedValue([{ id: 'a', name: 'a', type: 'bkmsApp', framework: 'blank', language: 'go' }]);
    mocks.detail.mockResolvedValue({ id: 'a', name: 'a', type: 'bkmsApp', framework: 'blank', language: 'go' });
    let complete!: (value: object) => void;
    mocks.runtime.mockReturnValue(
      new Promise(resolve => {
        complete = resolve;
      }),
    );
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      expect(mocks.detail).toHaveBeenCalledWith({ appID: 'a' });
      expect(mocks.runtime).toHaveBeenCalledWith({ appID: 'a' }, { interceptorErr: false });
      expect(useAppDetail().runtimeLoading).toBe(true);
      expect(wrapper.findComponent({ name: 'RouterView' }).exists()).toBe(true);
      complete({ type: 'bkmsApp', framework: 'blank', mods: {} });
      await flushPromises();
    } finally {
      wrapper.unmount();
    }
  });

  it('能力失败仍保留子页面，重试恢复后不覆盖真实应用类型', async () => {
    mocks.space = 'ws-1';
    mocks.apps.mockResolvedValue([{ id: 'a', name: 'a', type: 'trpc', framework: 'trpc', language: 'go' }]);
    mocks.detail.mockResolvedValue({ id: 'a', name: 'a', type: 'trpc', framework: 'trpc', language: 'go' });
    mocks.runtime.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({
      type: 'bkmsApp',
      framework: 'trpc',
      mods: { devMode: { supported: true } },
    });
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      expect(wrapper.findComponent({ name: 'RouterView' }).exists()).toBe(true);
      expect(wrapper.get('[role="alert"]').text()).toContain('应用能力加载失败，部分功能暂不可用');
      await wrapper.get('[role="alert"] button').trigger('click');
      await flushPromises();
      expect(wrapper.find('[role="alert"]').exists()).toBe(false);
      expect(useAppDetail().runtime?.mods?.devMode?.supported).toBe(true);
      expect(useAppDetail().appType).toBe('trpc');
    } finally {
      wrapper.unmount();
    }
  });
});
