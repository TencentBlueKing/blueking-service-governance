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
 * stub/mock：bkmsserver 最小集 + v1 Proxy 兜底 + vue-router 捕获守卫回调；历史栈回归使用
 *            真实 Vue Router / memory history，其余子页面仍隔离 + vue-i18n useI18n 直译
 * 变异验证：3 处全部命中（恒真→③④红 / 死循环判定删除→②红 / query 串改→①红），已精确还原
 * backlog：无
 */
import { ref } from 'vue';

import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { useRouter } from 'vue-router';
import { appNavigationConfig } from '~/config/navigation/app';
import AppDetailShell from '~/pages/application/detail.vue';
import { useAppDetail } from '~/stores/app-detail';

import { installVxeShims } from '../helpers/mock-table';

import type { VueWrapper } from '@vue/test-utils';
import type { NavigationGuard, RouteLocationNormalizedGeneric, Router } from 'vue-router';
import type * as VueRouter from 'vue-router';
import type { NavigationItem } from '~/config/navigation/types';

const mocks = vi.hoisted(() => ({
  /** 捕获 detail.vue 注册的 onBeforeRouteUpdate 守卫回调（每次 mount 清空，取最新注册） */
  routeUpdateGuards: [] as Array<
    (to: RouteLocationNormalizedGeneric, from: RouteLocationNormalizedGeneric) => ReturnType<NavigationGuard>
  >,
  space: '',
  apps: vi.fn(),
  listPageApps: vi.fn(),
  detail: vi.fn(),
  runtime: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  realRouter: null as null | Router,
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
        CustomNavigation: {
          name: 'CustomNavigation',
          props: ['list', 'activeKey'],
          template: '<div><slot /><slot name="side-header" /></div>',
        },
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
  return { ApiServerService: { ...base.ApiServerService, GetApp: mocks.detail, ListApps: mocks.listPageApps } };
});
vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());
vi.mock('~/api/modules/v1', async () => (await import('../helpers/mock-api')).v1ProxyMockFactory());
vi.mock('~/api/modules/v1/app', () => ({
  AppService: { listApps: mocks.apps, getAppRuntimeMods: mocks.runtime },
}));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: mocks.space }) }));

beforeAll(() => {
  installVxeShims();
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.realRouter = null;
  mocks.space = '';
  mocks.apps.mockResolvedValue([]);
  mocks.listPageApps.mockResolvedValue([]);
  mocks.detail.mockResolvedValue(undefined);
  mocks.runtime.mockResolvedValue(undefined);
  useRouter().currentRoute.value = loc({ name: '' });
});

// vue-router：捕获守卫回调（非 no-op——本场景的验证对象就是回调逻辑本身）
vi.mock('vue-router', async importOriginal => {
  const actual = await importOriginal<typeof VueRouter>();
  const currentRoute = ref({ params: {}, query: {}, name: '', meta: {} });
  const navigate = async (target: { params?: object; query?: object }) => {
    currentRoute.value = {
      ...currentRoute.value,
      ...target,
      params: { ...currentRoute.value.params, ...target.params },
      query: target.query || {},
    };
  };
  mocks.push.mockImplementation(navigate);
  mocks.replace.mockImplementation(navigate);
  return {
    ...actual,
    RouterView: { template: '<div />' },
    RouterLink: { template: '<a><slot /></a>' },
    useRoute: () => (mocks.realRouter ? actual.useRoute() : currentRoute.value),
    useRouter: () =>
      mocks.realRouter || {
        push: mocks.push,
        replace: mocks.replace,
        currentRoute,
      },
    onBeforeRouteUpdate: (cb: (typeof mocks.routeUpdateGuards)[number]) => {
      if (mocks.realRouter) return actual.onBeforeRouteUpdate(cb);
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

function menuKeys(wrapper: VueWrapper): string[] {
  const list = wrapper.findComponent({ name: 'CustomNavigation' }).props('list') as NavigationItem[];
  return list.flatMap(item => ('children' in item ? item.children.map(child => child.key) : [item.key]));
}

/** 挂载真实壳层和路由，仅隔离子页面；保留异步导航与前进/后退的历史语义。 */
async function mountShellWithHistory(
  name: string,
  {
    menuName = 'observation',
    type = 'bkmsApp',
    fromList = false,
    renderIcons = fromList,
  }: { fromList?: boolean; menuName?: string; renderIcons?: boolean; type?: string } = {},
) {
  const {
    createRouter,
    createMemoryHistory,
    RouterView: routerView,
  } = await vi.importActual<typeof VueRouter>('vue-router');
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/:space/app',
        name: 'app',
        component: fromList ? (await import('~/pages/application/application.vue')).default : { template: '<div />' },
        props: { space: 'ws-1' },
      },
      {
        path: '/:space/app/:name',
        component: AppDetailShell,
        children: [
          { path: ':type/:menuName', name: 'detail', component: { template: '<div data-test="application-page" />' } },
        ],
      },
    ],
  });
  mocks.realRouter = router;
  await router.push('/ws-1/app');
  if (!fromList) {
    await router.push({
      name: 'detail',
      params: { space: 'ws-1', name, type, menuName },
      query: { stale: 'observation-query' },
    });
  }
  const pinia = createPinia();
  setActivePinia(pinia);
  const wrapper = mount(routerView, {
    global: {
      plugins: [pinia, router],
      stubs: {
        CustomNavigation: {
          name: 'CustomNavigation',
          props: ['list', 'activeKey'],
          template: '<div><slot /><slot name="side-header" /></div>',
        },
        Select: {
          name: 'Select',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<div><slot name="trigger" /></div>',
        },
        Alert: { template: '<div role="alert"><slot name="title" /><slot /></div>' },
        Button: { template: '<button><slot /></button>' },
        Loading: true,
        TypeIcon: !renderIcons,
        ColorIcon: true,
        AngleDownFill: true,
        Plus: true,
      },
      mocks: { $t: (text: string) => text },
      directives: { bkloading: {}, 'bk-tooltips': {} },
    },
  });
  await flushPromises();
  return { router, wrapper };
}

describe('应用详情壳层：首屏应用图标', () => {
  it.each([
    { type: 'bkmsApp', framework: 'blank', language: 'python', label: 'Python' },
    { type: 'bkmsApp', framework: 'trpc', language: 'go', label: 'tRPC' },
    { type: 'helm', framework: '', language: '', label: 'Helm Chart' },
  ])('从应用列表进入时，列表接口返回前即展示已知的 $label 图标', async ({ label, ...presentation }) => {
    mocks.space = 'ws-1';
    const app = { id: 'app-id', name: 'known-app', ...presentation };
    mocks.listPageApps.mockResolvedValue([app]);
    let completeList!: (value: object[]) => void;
    mocks.apps.mockReturnValue(new Promise(resolve => (completeList = resolve)));
    mocks.detail.mockResolvedValue(app);
    const { router, wrapper } = await mountShellWithHistory(app.name, { fromList: true });
    try {
      const applicationLink = wrapper.findAll('button').find(button => button.text() === app.name);
      expect(applicationLink).toBeDefined();
      await applicationLink!.trigger('click');
      await flushPromises();
      expect(router.currentRoute.value.name).toBe('detail');
      expect(mocks.apps).toHaveBeenCalledWith({ workspaceID: 'ws-1' });
      expect(mocks.detail).not.toHaveBeenCalled();
      expect(wrapper.find(`[title="${label}"]`).exists()).toBe(true);
      expect(wrapper.find('[title="平台托管应用"]').exists()).toBe(false);

      const listedApp = app.language === 'python' ? { ...app, language: 'nodejs' } : app;
      completeList([listedApp]);
      await flushPromises();
      const loadedLabel = listedApp.language === 'nodejs' ? 'Node.js' : label;
      expect(wrapper.find(`[title="${loadedLabel}"]`).exists()).toBe(true);
    } finally {
      wrapper.unmount();
    }
  });

  it.each([
    { space: 'ws-2', name: 'known-app' },
    { space: 'ws-1', name: 'another-app' },
  ])('进入 $space/$name 时，不复用其他空间或应用的图标', async params => {
    mocks.space = 'ws-1';
    const app = { id: 'app-id', name: 'known-app', type: 'bkmsApp', framework: 'blank', language: 'python' };
    mocks.listPageApps.mockResolvedValue([app]);
    mocks.apps.mockReturnValue(new Promise(() => {}));
    const { router, wrapper } = await mountShellWithHistory(app.name, { fromList: true });
    try {
      await wrapper
        .findAll('button')
        .find(button => button.text() === app.name)!
        .trigger('click');
      await flushPromises();
      expect(wrapper.find('[title="Python"]').exists()).toBe(true);
      await router.push({ name: 'detail', params: { ...params, type: 'bkmsApp', menuName: 'info' } });
      await flushPromises();
      expect(wrapper.find('[title="Python"]').exists()).toBe(false);
      expect(wrapper.find('[title="平台托管应用"]').exists()).toBe(true);
    } finally {
      wrapper.unmount();
    }
  });

  it('刷新或直接访问时，在无导航摘要的情况下按接口结果显示图标', async () => {
    mocks.space = 'ws-1';
    const app = { id: 'app-id', name: 'known-app', type: 'bkmsApp', framework: 'blank', language: 'nodejs' };
    let completeList!: (value: object[]) => void;
    mocks.apps.mockReturnValue(new Promise(resolve => (completeList = resolve)));
    mocks.detail.mockResolvedValue(app);
    const { wrapper } = await mountShellWithHistory(app.name, { menuName: 'info', renderIcons: true });
    try {
      expect(wrapper.find('[title="平台托管应用"]').exists()).toBe(true);
      completeList([app]);
      await flushPromises();
      expect(wrapper.find('[title="Node.js"]').exists()).toBe(true);
      expect(wrapper.find('[title="平台托管应用"]').exists()).toBe(false);
    } finally {
      wrapper.unmount();
    }
  });
});

function openObservation(name = 'a') {
  useRouter().currentRoute.value = loc({
    params: { name, type: 'bkmsApp', menuName: 'observation' },
    query: { stale: 'observation-query' },
  });
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

describe('应用详情壳层：观测能力回退历史栈', () => {
  beforeEach(() => {
    mocks.space = 'ws-1';
    const apps = [
      { id: 'a', name: 'a', type: 'bkmsApp' },
      { id: 'b', name: 'b', type: 'bkmsApp' },
    ];
    mocks.apps.mockResolvedValue(apps);
    mocks.detail.mockImplementation(async ({ appID }: { appID: string }) => apps.find(app => app.id === appID));
  });

  it.each([true, false])(
    '从支持应用切到不支持应用后，一次后退回原观测页，前进回基本信息（能力延迟：%s）',
    async delayed => {
      let completeB!: (value: object) => void;
      mocks.runtime.mockImplementation(({ appID }: { appID: string }) =>
        appID === 'a'
          ? Promise.resolve({ mods: { apmServiceName: { supported: true } } })
          : delayed
            ? new Promise(resolve => (completeB = resolve))
            : Promise.resolve({ mods: { apmServiceName: { supported: false } } }),
      );
      const { router, wrapper } = await mountShellWithHistory('a');
      try {
        expect(menuKeys(wrapper)).toContain('observation');
        wrapper.findComponent({ name: 'Select' }).vm.$emit('update:modelValue', 'b');
        await flushPromises();
        if (delayed) {
          expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'observation' });
          completeB({ mods: { apmServiceName: { supported: false } } });
          await flushPromises();
        }
        expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'info' });
        expect(router.currentRoute.value.query).toEqual({});
        expect(menuKeys(wrapper)).not.toContain('observation');

        router.back();
        await flushPromises();
        expect(router.currentRoute.value.params).toMatchObject({ name: 'a', menuName: 'observation' });
        expect(menuKeys(wrapper)).toContain('observation');

        router.forward();
        await flushPromises();
        if (delayed) {
          completeB({ mods: { apmServiceName: { supported: false } } });
          await flushPromises();
        }
        expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'info' });
        expect(menuKeys(wrapper)).not.toContain('observation');
      } finally {
        wrapper.unmount();
      }
    },
  );

  it('直达不支持应用的旧链接时等待能力，并用基本信息替换该历史项', async () => {
    let complete!: (value: object) => void;
    mocks.runtime.mockReturnValue(new Promise(resolve => (complete = resolve)));
    const { router, wrapper } = await mountShellWithHistory('b');
    try {
      expect(useAppDetail().runtimeLoading).toBe(true);
      expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'observation' });
      expect(menuKeys(wrapper)).not.toContain('observation');
      complete({ mods: { apmServiceName: { supported: false } } });
      await flushPromises();
      expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'info' });
      expect(router.currentRoute.value.query).toEqual({});

      router.back();
      await flushPromises();
      expect(router.currentRoute.value.name).toBe('app');

      router.forward();
      await flushPromises();
      expect(router.currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'info' });
    } finally {
      wrapper.unmount();
    }
  });
});

describe('应用详情壳层：跨类型应用切换菜单', () => {
  it.each([
    { fromType: 'trpc', menuName: 'dashboard', toType: 'helm', expected: 'info' },
    { fromType: 'taf', menuName: 'alert', toType: 'agones', expected: 'info' },
    { fromType: 'helm', menuName: 'orchestrate', toType: 'trpc', expected: 'info' },
    { fromType: 'agones', menuName: 'orchestrate', toType: 'bkmsApp', expected: 'info' },
    { fromType: 'trpc', menuName: 'build', toType: 'helm', expected: 'build' },
    { fromType: 'helm', menuName: 'network', toType: 'taf', expected: 'network' },
  ])(
    '$fromType 的 $menuName 切换 $toType 后选中 $expected，后退恢复来源页',
    async ({ fromType, menuName, toType, expected }) => {
      mocks.space = 'ws-1';
      const apps = [
        { id: 'a', name: 'a', type: fromType },
        { id: 'b', name: 'b', type: toType },
      ];
      mocks.apps.mockResolvedValue(apps);
      mocks.detail.mockImplementation(async ({ appID }: { appID: string }) => apps.find(app => app.id === appID));
      mocks.runtime.mockResolvedValue({ mods: { apmServiceName: { supported: true } } });
      const { router, wrapper } = await mountShellWithHistory('a', { menuName, type: fromType });
      try {
        wrapper.findComponent({ name: 'Select' }).vm.$emit('update:modelValue', 'b');
        await flushPromises();
        expect(router.currentRoute.value.params).toMatchObject({ name: 'b', type: toType, menuName: expected });
        expect(menuKeys(wrapper)).toContain(expected);
        expect(wrapper.findComponent({ name: 'CustomNavigation' }).props('activeKey')).toBe(expected);

        router.back();
        await flushPromises();
        expect(router.currentRoute.value.params).toMatchObject({ name: 'a', type: fromType, menuName });
        expect(wrapper.findComponent({ name: 'CustomNavigation' }).props('activeKey')).toBe(menuName);
      } finally {
        wrapper.unmount();
      }
    },
  );

  it('直达 Helm 无效菜单时，在列表返回后选中基本信息，后退离开详情', async () => {
    mocks.space = 'ws-1';
    let completeList!: (value: object[]) => void;
    mocks.apps.mockReturnValue(new Promise(resolve => (completeList = resolve)));
    const app = { id: 'a', name: 'a', type: 'helm' };
    mocks.detail.mockResolvedValue(app);
    const { router, wrapper } = await mountShellWithHistory('a', { type: 'helm', menuName: 'dashboard' });
    try {
      expect(router.currentRoute.value.params.menuName).toBe('dashboard');
      completeList([app]);
      await flushPromises();
      expect(router.currentRoute.value.params).toMatchObject({ name: 'a', type: 'helm', menuName: 'info' });
      expect(wrapper.findComponent({ name: 'CustomNavigation' }).props('activeKey')).toBe('info');

      router.back();
      await flushPromises();
      expect(router.currentRoute.value.name).toBe('app');
    } finally {
      wrapper.unmount();
    }
  });
});

describe('应用详情壳层：能力局部加载', () => {
  it('加载中隐藏观测入口，确认支持后恢复并保留观测链接', async () => {
    mocks.space = 'ws-1';
    openObservation();
    mocks.apps.mockResolvedValue([{ id: 'a', name: 'a', type: 'bkmsApp', framework: 'trpc', language: 'go' }]);
    mocks.detail.mockResolvedValue({ id: 'a', name: 'a', type: 'bkmsApp', framework: 'trpc', language: 'go' });
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
      expect(menuKeys(wrapper)).not.toContain('observation');
      expect(menuKeys(wrapper)).toEqual(expect.arrayContaining(['dashboard', 'alert']));
      expect(useRouter().currentRoute.value.params.menuName).toBe('observation');
      complete({ type: 'bkmsApp', framework: 'trpc', mods: { apmServiceName: { supported: true } } });
      await flushPromises();
      expect(menuKeys(wrapper)).toContain('observation');
      expect(useRouter().currentRoute.value.params.menuName).toBe('observation');
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
      mods: { devMode: { supported: true }, apmServiceName: { supported: true } },
    });
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      expect(wrapper.findComponent({ name: 'RouterView' }).exists()).toBe(true);
      expect(wrapper.get('[role="alert"]').text()).toContain('应用能力加载失败，部分功能暂不可用');
      expect(menuKeys(wrapper)).not.toContain('observation');
      await wrapper.get('[role="alert"] button').trigger('click');
      await flushPromises();
      expect(wrapper.find('[role="alert"]').exists()).toBe(false);
      expect(useAppDetail().runtime?.mods?.devMode?.supported).toBe(true);
      expect(useAppDetail().appType).toBe('trpc');
      expect(menuKeys(wrapper)).toContain('observation');
    } finally {
      wrapper.unmount();
    }
  });

  it('直接进入观测数据时等待能力结果，不支持后回退基本信息', async () => {
    mocks.space = 'ws-1';
    openObservation();
    mocks.apps.mockResolvedValue([{ id: 'a', name: 'a', type: 'bkmsApp' }]);
    mocks.detail.mockResolvedValue({ id: 'a', name: 'a', type: 'bkmsApp' });
    let complete!: (value: object) => void;
    mocks.runtime.mockReturnValue(new Promise(resolve => (complete = resolve)));
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      expect(useRouter().currentRoute.value.params.menuName).toBe('observation');
      complete({ mods: { apmServiceName: { supported: false } } });
      await flushPromises();
      expect(menuKeys(wrapper)).not.toContain('observation');
      expect(menuKeys(wrapper)).toEqual(expect.arrayContaining(['dashboard', 'alert']));
      expect(useRouter().currentRoute.value.params.menuName).toBe('info');
      expect(useRouter().currentRoute.value.query).toEqual({});
    } finally {
      wrapper.unmount();
    }
  });

  it('切换支持与不支持的应用时更新菜单，且不修改共享导航定义', async () => {
    mocks.space = 'ws-1';
    useRouter().currentRoute.value = loc({ params: { name: 'a', type: 'bkmsApp', menuName: 'info' } });
    const apps = [
      { id: 'a', name: 'a', type: 'bkmsApp' },
      { id: 'b', name: 'b', type: 'bkmsApp' },
    ];
    mocks.apps.mockResolvedValue(apps);
    mocks.detail.mockImplementation(async ({ appID }: { appID: string }) => apps.find(app => app.id === appID));
    mocks.runtime.mockImplementation(async ({ appID }: { appID: string }) => ({
      mods: { apmServiceName: { supported: appID === 'a' } },
    }));
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      expect(menuKeys(wrapper)).toContain('observation');
      useRouter().currentRoute.value = loc({ params: { name: 'b', type: 'bkmsApp', menuName: 'info' } });
      await flushPromises();
      expect(useAppDetail().appID).toBe('b');
      expect(menuKeys(wrapper)).not.toContain('observation');
      expect(appNavigationConfig.bkmsApp.flatMap(item => ('children' in item ? item.children : [item]))).toEqual(
        expect.arrayContaining([expect.objectContaining({ key: 'observation' })]),
      );
      useRouter().currentRoute.value = loc({ params: { name: 'a', type: 'bkmsApp', menuName: 'info' } });
      await flushPromises();
      expect(menuKeys(wrapper)).toContain('observation');
    } finally {
      wrapper.unmount();
    }
  });

  it('旧应用迟到的不支持响应不能将新应用从观测数据重定向', async () => {
    mocks.space = 'ws-1';
    openObservation();
    const apps = [
      { id: 'a', name: 'a', type: 'bkmsApp' },
      { id: 'b', name: 'b', type: 'bkmsApp' },
    ];
    mocks.apps.mockResolvedValue(apps);
    mocks.detail.mockImplementation(async ({ appID }: { appID: string }) => apps.find(app => app.id === appID));
    let completeOld!: (value: object) => void;
    mocks.runtime.mockImplementation(({ appID }: { appID: string }) =>
      appID === 'a'
        ? new Promise(resolve => (completeOld = resolve))
        : Promise.resolve({ mods: { apmServiceName: { supported: true } } }),
    );
    const { wrapper } = mountShell();
    try {
      await flushPromises();
      openObservation('b');
      await flushPromises();
      expect(useAppDetail().appID).toBe('b');
      expect(menuKeys(wrapper)).toContain('observation');
      completeOld({ mods: { apmServiceName: { supported: false } } });
      await flushPromises();
      expect(menuKeys(wrapper)).toContain('observation');
      expect(useRouter().currentRoute.value.params).toMatchObject({ name: 'b', menuName: 'observation' });
    } finally {
      wrapper.unmount();
    }
  });
});
