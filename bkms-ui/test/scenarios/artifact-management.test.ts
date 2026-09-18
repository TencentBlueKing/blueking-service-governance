/**
 * 场景级测试：制品管理（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S11）
 *
 * 覆盖三组用户可感知行为：Helm-like 应用显示双页签 / 非 Helm-like 不显示页签
 *   直接展示容器镜像 / 点击页签切换到对应制品内容 → V = 3
 *   （页签切换曾判为「jsdom 不可行」，复核后证伪：让 mock 路由的 query 可写且
 *     replace 写回即可，见本文件 vue-router mock 注释）
 *
 * 说明：被测核心是 index.vue 的「应用类型 → 视图分发」与 Tab 切换逻辑；
 * container-image / helm-chart 为重型子页（含表格与上传交互），此处 stub 为标记文本，
 * 其行为留待各自场景覆盖。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const harness = vi.hoisted(() => ({
  appType: 'helm' as string,
}));

import { i18nGlobalMocks } from '../helpers/mock-i18n';

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  const { reactive } = await import('vue');
  const base = await createRouterMock({
    route: { path: '/ws-1/app/app-a/detail/artifact', name: 'artifact', params: { space: 'ws-1' } },
  });
  // useUrlQuerySync 写侧走 router.replace({ query })、读侧读 route.query（use-url-query-sync.ts:74-80）；
  // 让两者共享同一个响应式 query 对象，页签切换即可在 jsdom 下驱动
  const query = reactive<Record<string, string>>({});
  const route = { ...base.useRoute(), query };
  return {
    ...base,
    useRoute: () => route,
    useRouter: () => ({
      ...base.useRouter(),
      replace: (target: { query?: Record<string, unknown> }) => {
        Object.keys(query).forEach(key => delete query[key]);
        Object.assign(query, target?.query ?? {});
        return Promise.resolve();
      },
    }),
  };
});
vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({
    appType: harness.appType,
    updateAppName: vi.fn(),
    updateAppID: vi.fn(),
  }),
}));
// 重型子页 stub：仅保留可断言的标记文本
vi.mock('~/pages/application/detail/artifact/container-image.vue', async () => {
  const { defineComponent } = await import('vue');
  return { default: defineComponent({ name: 'ContainerImageStub', template: '<div>container-image-content</div>' }) };
});
vi.mock('~/pages/application/detail/artifact/helm-chart.vue', async () => {
  const { defineComponent } = await import('vue');
  return { default: defineComponent({ name: 'HelmChartStub', template: '<div>helm-chart-content</div>' }) };
});

beforeEach(() => {
  vi.clearAllMocks();
  harness.appType = 'helm';
});

async function renderPage() {
  const { default: ArtifactPage } = await import('~/pages/application/detail/artifact/index.vue');
  return render(ArtifactPage as never, {
    global: { mocks: i18nGlobalMocks } as never,
  });
}

describe('制品管理：按应用类型分发视图', () => {
  it('当应用为 Helm-like 类型时，应展示容器镜像与 Helm Chart 两个制品页签', async () => {
    await renderPage();
    await waitFor(() => expect(screen.getByText('容器镜像')).toBeInTheDocument());
    expect(screen.getByText('Helm Chart')).toBeInTheDocument();
    // 默认落在第一个页签：容器镜像
    expect(screen.getByText('container-image-content')).toBeInTheDocument();
  });

  it('当应用非 Helm-like 类型时，应直接展示容器镜像而不显示页签', async () => {
    harness.appType = 'default';
    await renderPage();
    await waitFor(() => expect(screen.getByText('container-image-content')).toBeInTheDocument());
    expect(screen.queryByText('Helm Chart')).not.toBeInTheDocument();
  });

  it('当用户点击 Helm Chart 页签时，应展示 Helm Chart 制品内容', async () => {
    await renderPage();
    await waitFor(() => expect(screen.getByText('container-image-content')).toBeInTheDocument());
    await userEvent.click(screen.getByText('Helm Chart'));
    // 正向消费信号：页签切换后渲染目标子页（原「jsdom 不可测」结论已证伪，
    // 关键是让 mock 路由的 query 可写且 replace 写回，见本文件 vue-router mock 注释）
    await waitFor(() => expect(screen.getByText('helm-chart-content')).toBeInTheDocument());
    expect(screen.queryByText('container-image-content')).not.toBeInTheDocument();
  });
});
