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
import { ref } from 'vue';
import type { App } from 'vue';

import { mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it, vi } from 'vitest';
import AppDetailShell from '~/pages/application/detail.vue';
import FrameworkConfig from '~/pages/application/detail/app-config/framework-config.vue';
import AppConfigIndex from '~/pages/application/detail/app-config/index.vue';
import AppDetailObservation from '~/pages/application/detail/observation.vue';

/**
 * 高度链契约测试（jsdom，DOM 契约：mount 真实组件 → 断言渲染后 DOM 的高度角色 class）。
 *
 * 每条高度链的每一环只需保证「自己声明了高度角色」（h-full / flex-1 + min-h-0 / 任意值定高），
 * 跨页面的链路由 e2e/tests/layout.spec.ts 的真实渲染高度断言兜底（jsdom 无布局引擎，测不了像素）。
 * 本套用例在 class 层面提供毫秒级回归反馈：壳层丢 h-full（#135 类回归）在此直接变红。
 *
 * 断言对象是「被高度链依赖的容器」的 class（mount 后真实 DOM），页面结构调整导致 selector
 * 失效时测试会红——此时按新结构更新 selector，并确认新结构仍满足高度角色声明。
 */

// ── 重依赖 mock：掐断 API / store / router / i18n 链，页面 mount 只为得到真实模板 DOM ──
// （jsdom 环境垫片 ResizeObserver / PointerEvent 在 test/setup.ts，经 setupFiles 先于组件加载执行）

vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: {
    ListApps: vi.fn().mockResolvedValue([]),
    GetWorkspace: vi.fn().mockResolvedValue(undefined),
    GetApp: vi.fn().mockResolvedValue(undefined),
    // observation.vue fetchApmData 会调用；当前测试环境 store 为空靠前置 guard 短路，
    // 此处显式 stub，避免未来 guard 调整后以 TypeError 形式挂掉
    GetApmServiceName: vi.fn().mockResolvedValue(null),
  },
}));

// v1 API 模块导出大量 service 且部分子组件在顶层 .bind() 引用：
// 用自相似 Proxy 兜底——任意导出名均得到可调用、可 bind 的节点（测试路径不触达其返回值）。
// 注意：symbol/then/__esModule 必须透传 undefined，否则 vi.mock 的 await 会把 factory
// 结果误判为 thenable 导致 vitest 挂死；模块层须为 object（vitest 校验 factory 返回值）
vi.mock('~/api/modules/v1', () => {
  const makeNode = (): unknown =>
    new Proxy(function stub() {} as unknown as Record<PropertyKey, unknown>, {
      get: (target, prop) => {
        if (typeof prop === 'symbol' || prop === 'then' || prop === '__esModule' || prop === 'default') {
          return undefined;
        }
        if (!(prop in target)) {
          target[prop] = makeNode();
        }
        return target[prop];
      },
      apply: () => Promise.resolve(undefined),
    });
  const moduleCache: Record<PropertyKey, unknown> = {};
  return new Proxy(moduleCache, {
    get: (target, prop) => {
      if (typeof prop === 'symbol' || prop === 'then' || prop === '__esModule') {
        return undefined;
      }
      if (!(prop in target)) {
        target[prop] = makeNode();
      }
      return target[prop];
    },
  });
});

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
  };
});

// 部分覆盖：保留真实 createI18n（modules/i18n 顶层创建实例），仅替换组件内 useI18n（免 app 上下文）
vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  useI18n: () => ({ t: (s: string) => s, te: () => true }),
}));

/** 模板 $t 直译占位（i18n 不在测试范围，仅为让 mount 不报错） */
const i18nStub = {
  install(app: App) {
    app.config.globalProperties.$t = (s: string) => s;
  },
};

/**
 * 公共 mount 配置：pinia（真实 store 定义，API 已 mock）+ 子组件全 stub（shallow）。
 * mock 只有 4 个，全部是「让组件可 import/mount」的最小集：
 * API ×2（顶层 import，不 stub 则加载整条请求链）、vue-router、vue-i18n（setup 调用 useRoute/useI18n）。
 * ResizeObserver/$t 是 jsdom 环境垫片（非业务 mock）。
 */
function mountOptions(extraStubs: Record<string, boolean | { template: string }> = {}) {
  return {
    shallow: true,
    global: {
      plugins: [createPinia(), i18nStub],
      // bkui-vue 指令垫片：未装 bkui-vue 全量插件，空指令消除解析警告
      directives: { bkloading: {}, 'bk-tooltips': {} },
      stubs: {
        // 透传 slot 的壳：保留被测容器 DOM，隔离子组件树
        CustomNavigation: { template: '<div data-test="shell"><slot /></div>' },
        FlexRow: { template: '<div data-test="flex-row"><slot name="left" /><slot name="right" /></div>' },
        RouterView: { template: '<div data-test="router-view" />' },
        // <component :is> 动态组件 shallow 拦不住，显式封住三个子页
        DeployConfig: true,
        FrameworkConfig: true,
        EnvVariable: true,
        // 宿主容器藏在多层 slot 里，透传渲染
        Skeleton: { template: '<div data-test="skeleton"><slot /></div>' },
        BkmsContent: { template: '<div data-test="bkms-content"><slot /><slot name="action" /></div>' },
        ...extraStubs,
      },
    },
  };
}

describe('高度链契约（关键页面的高度角色声明）', () => {
  it('应用详情壳层 detail.vue：v-bkloading 容器声明 h-full + min-h-full（#135 原始回归点）', () => {
    const wrapper = mount(AppDetailShell, mountOptions());
    // CustomNavigation stub 的默认 slot 即被测容器（RouterView 的直接父级），静态渲染无需等待
    const container = wrapper.find('[data-test="shell"] > div');
    expect(container.exists()).toBe(true);
    expect(container.classes(), '壳层容器必须同时声明 h-full 与 min-h-full，缺失即整棵子页树塌陷').toEqual(
      expect.arrayContaining(['h-full', 'min-h-full']),
    );
    wrapper.unmount();
  });

  it('观测数据页 observation.vue：根容器声明 flex h-full min-h-0 flex-col', () => {
    const wrapper = mount(AppDetailObservation, mountOptions());
    expect(wrapper.classes(), '观测页根容器必须撑满父级并允许收缩').toEqual(
      expect.arrayContaining(['flex', 'flex-col', 'h-full', 'min-h-0']),
    );
    wrapper.unmount();
  });

  it('观测数据页 observation.vue：iframe 容器声明 flex-1 + min-h-0（高度链收口环）', () => {
    const wrapper = mount(AppDetailObservation, mountOptions());
    const container = wrapper.find('div.bg-white');
    expect(container.exists()).toBe(true);
    expect(container.classes(), 'iframe 容器必须 flex-1 撑满且 min-h-0 可收缩').toEqual(
      expect.arrayContaining(['flex-1', 'min-h-0']),
    );
    wrapper.unmount();
  });

  it('应用配置页 app-config/index.vue：根容器声明 flex-col h-full，内容区声明 flex-1 min-h-0', () => {
    const wrapper = mount(AppConfigIndex, mountOptions());
    expect(wrapper.classes(), '配置页根容器必须纵向撑满').toEqual(
      expect.arrayContaining(['flex', 'flex-col', 'h-full']),
    );
    const content = wrapper.find('div.min-h-0');
    expect(content.exists()).toBe(true);
    expect(content.classes(), '配置页内容区必须 flex-1 撑满且 min-h-0 可收缩').toEqual(
      expect.arrayContaining(['flex-1', 'min-h-0']),
    );
    wrapper.unmount();
  });

  it('框架配置文件 framework-config.vue：根容器声明 flex h-full min-h-0 flex-col', () => {
    const wrapper = mount(FrameworkConfig, mountOptions());
    expect(wrapper.classes(), '框架配置页根容器必须撑满父级并允许收缩').toEqual(
      expect.arrayContaining(['flex', 'flex-col', 'h-full', 'min-h-0']),
    );
    wrapper.unmount();
  });

  it('框架配置文件 framework-config.vue：编辑器宿主区声明 flex-1 + min-h-0（Monaco 高度链收口环）', () => {
    const wrapper = mount(FrameworkConfig, mountOptions());
    const host = wrapper.find('div.flex-1.min-h-0');
    expect(host.exists(), '编辑器宿主区必须存在且声明 flex-1 min-h-0').toBe(true);
    wrapper.unmount();
  });
});
