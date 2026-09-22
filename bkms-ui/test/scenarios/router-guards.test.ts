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
 * 场景级测试：路由智能返回 + 空间权限守卫（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S13）
 *
 * 覆盖两组用户可感知行为：
 * A. 智能返回（smartGoBack）：有/无浏览历史 × fallback × 上级路由推导 → V = 4
 *    （第 5 条路径「推导不出上级路由 → 浏览器后退」需构造「父级路由记录无 name 且无默认子路由」
 *     的场景才能覆盖，本轮未构造）
 * B. 空间权限守卫（beforeEach）：无空间参数 / 就绪 / 无权限 / 未就绪 / 空列表先拉取 → V = 5
 *
 * 说明：本场景为纯路由逻辑（无 DOM 交互），用户可感知结果 = 最终停留的路由与浏览器历史行为，
 * 因此断言 currentRoute 与 window.history.back 调用，而非 DOM 查询。
 */
import { createApp } from 'vue';

import { waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { install as installRouter } from '~/modules/router';

import type { Router } from 'vue-router';

const spaceState = vi.hoisted(() => ({
  list: [] as Array<{ id: string; state: string }>,
  fetch: vi.fn(async () => [] as Array<{ id: string; state: string }>),
}));

vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({
    list: spaceState.list,
    // 与 src/stores/space.ts:44-47 的真实枚举保持一致（Ready / Disabled，首字母大写）
    spaceState: { Ready: 'Ready', Disabled: 'Disabled' },
    handleGetWorkspaceList: async () => {
      const next = await spaceState.fetch();
      spaceState.list = next;
      return next;
    },
  }),
}));

/** 覆写浏览器历史状态：back 非空 = 有浏览历史 */
function setHistory(back: null | string) {
  window.history.replaceState({ back, current: location.hash }, '', location.href);
}

/** 挂一个空 app 安装真实 router 模块，取出 router 实例（install 内部自建 router，需经 app 获取） */
async function setup(initial?: Record<string, unknown> | string) {
  const app = createApp({ render: () => null });
  installRouter({ app } as never);
  const router = app.config.globalProperties.$router as Router;
  if (initial) await router.push(initial as string);
  return { app, router };
}

/** 浏览器后退：vue-router 的 back 底层走 history.go(-1)，非 history.back() */
function spyBrowserBack() {
  return vi.spyOn(window.history, 'go').mockImplementation(() => {});
}

beforeEach(() => {
  spaceState.list = [{ id: 'ws-1', state: 'Ready' }];
  spaceState.fetch.mockResolvedValue([{ id: 'ws-1', state: 'Ready' }]);
});

describe('路由：智能返回', () => {
  it('当用户有浏览历史时点击返回，应执行浏览器后退回到上一页', async () => {
    const { router } = await setup('/ws-1/app');
    setHistory('/previous-page');
    const backSpy = spyBrowserBack();
    (router.back as unknown as () => void)();
    expect(backSpy).toHaveBeenCalledWith(-1);
  });

  it('当用户有浏览历史且调用方也指定了回退目标时，应优先执行浏览器后退', async () => {
    const { router } = await setup('/ws-1/app');
    setHistory('/previous-page');
    const backSpy = spyBrowserBack();
    (router.back as unknown as (f: unknown) => void)({ name: 'env', params: { space: 'ws-1' } });
    expect(backSpy).toHaveBeenCalledWith(-1);
    expect(router.currentRoute.value.name).toBe('app');
  });

  it('当用户无浏览历史但调用方指定了回退目标时，应跳转到该目标', async () => {
    const { router } = await setup('/ws-1/app');
    setHistory(null);
    (router.back as unknown as (f: unknown) => void)({ name: 'env', params: { space: 'ws-1' } });
    await waitFor(() => expect(router.currentRoute.value.name).toBe('env'));
  });

  it('当用户无浏览历史且在子页面时，应回退到有名称的上级页面', async () => {
    const { router } = await setup('/ws-1/app/myapp/detail/deploy');
    expect(router.currentRoute.value.name).toBe('detail');
    setHistory(null);
    (router.back as unknown as () => void)();
    await waitFor(() => expect(router.currentRoute.value.name).toBe('appNavigation'));
  });

  it('当上级路由无名称时，应回退到它的默认子路由', async () => {
    const { router } = await setup('/ws-1/app/create/trpc');
    expect(router.currentRoute.value.name).toBe('createTrpcTemplateApp');
    setHistory(null);
    (router.back as unknown as () => void)();
    await waitFor(() => expect(router.currentRoute.value.name).toBe('createApplication'));
  });

  // 说明：smartGoBack 中「推导不出上级 → 退回浏览器后退」的分支在真实路由表下不可达——
  // resolveParent 在 matched.length < 2（router.ts:303）或「父级记录无 name 且无默认子路由」
  // （router.ts:311-316）时返回 undefined 才会走到该分支；真实路由表下后者需专门构造，
  // 本轮未构造，故不为其单独立用例。
});

describe('路由：空间权限守卫', () => {
  it('当用户访问不含空间参数的路由时，应直接放行', async () => {
    const { router } = await setup();
    await router.push('/403');
    expect(router.currentRoute.value.name).toBe('403');
  });

  it('当用户访问的空间在列表中且已就绪时，应放行到目标页面', async () => {
    const { router } = await setup();
    await router.push('/ws-1/app');
    expect(router.currentRoute.value.name).toBe('app');
  });

  it('当用户访问无权访问的空间时，应跳转 403 并携带回跳地址与空间 ID', async () => {
    spaceState.list = [];
    spaceState.fetch.mockResolvedValue([]);
    const { router } = await setup();
    await router.push('/ws-9/app');
    expect(router.currentRoute.value.name).toBe('403');
    expect(router.currentRoute.value.query.redirect).toBe('/ws-9/app');
    expect(router.currentRoute.value.query.workspaceID).toBe('ws-9');
  });

  it('当用户访问的空间尚未就绪时，应跳转 404', async () => {
    // 未就绪态用真实枚举值 Disabled（原用 'creating' 非真实取值，已按复核修正）
    spaceState.list = [{ id: 'ws-1', state: 'Disabled' }];
    const { router } = await setup();
    await router.push('/ws-1/app');
    expect(router.currentRoute.value.name).toBe('404');
  });

  it('当空间列表尚未加载时，应先拉取列表再判定放行', async () => {
    spaceState.list = [];
    spaceState.fetch.mockResolvedValue([{ id: 'ws-1', state: 'Ready' }]);
    const { router } = await setup();
    await router.push('/ws-1/app');
    // 守卫在一次导航中可能执行多次（重定向链），此处断言「确实拉取过列表」而非具体次数
    expect(spaceState.fetch).toHaveBeenCalled();
    expect(router.currentRoute.value.name).toBe('app');
  });
});
