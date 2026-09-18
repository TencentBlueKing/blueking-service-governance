/**
 * 场景级测试：应用列表页（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S14）
 *
 * 覆盖：有数据 / 空列表（含空态区块）/ 加载失败 / 进入详情 → V = 4
 *
 * 表格处理：vxe（@blueking/table）依赖元素尺寸与纯字段列 DOM，jsdom 下均不可用，
 * 采用「共享垫片 installVxeShims + 表格 stub」方案（见 test/helpers/mock-table.ts），
 * 垫片仅在本文件 beforeAll 安装，不改全局 setup，避免影响既有测试。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';
import { MockedApiError } from '../helpers/mocked-api-error';

const mocks = vi.hoisted(() => ({
  listApps: vi.fn(),
  push: vi.fn(),
}));

vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: { ListApps: mocks.listApps },
}));
vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({ route: { path: '/ws-1/app', name: 'app', params: { space: 'ws-1' } }, push: mocks.push });
});
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'ws-1' }) }));
vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({ updateAppName: vi.fn(), updateAppID: vi.fn() }),
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

beforeAll(() => {
  installVxeShims();
});

const app = (name: string) => ({
  name,
  type: 'default',
  language: 'go',
  creator: 'tester',
  deployedEnvs: [],
  createdAt: '2026-09-01T10:00:00Z',
  lastOperatedAt: '2026-09-01T10:00:00Z',
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listApps.mockResolvedValue([app('app-a')]);
});

async function renderPage() {
  const { default: Application } = await import('~/pages/application/application.vue');
  return render(Application as never, {
    props: { space: 'ws-1' } as never,
    global: {
      mocks: i18nGlobalMocks,
      // 空态/异常态文案由 <i18n-t> 插值组件渲染，直译为 keypath 供断言
      components: { 'i18n-t': I18nTStub },
    } as never,
  });
}

describe('应用列表：数据状态与导航', () => {
  it('当应用列表加载完成时，应展示应用名称', async () => {
    await renderPage();
    await waitFor(() => expect(screen.getByRole('button', { name: 'app-a' })).toBeInTheDocument());
  });

  it('当应用列表为空时，应结束加载、渲染空态区块且不显示任何应用', async () => {
    mocks.listApps.mockResolvedValue([]);
    await renderPage();
    await waitFor(() => expect(screen.getByText('创建应用')).toBeInTheDocument());
    // #empty 插槽（TableException 空态组件）随表格 stub 渲染，空态区块成为可断言信号
    expect(screen.getByTestId('table-empty')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'app-a' })).not.toBeInTheDocument();
  });

  it('当应用列表加载失败时，应结束加载状态且不渲染任何应用数据', async () => {
    mocks.listApps.mockRejectedValue(new MockedApiError());
    await renderPage();
    // 失败后必须退出骨架屏（源码 .catch 复位 isLoading），否则用户会一直看到加载中
    await waitFor(() => expect(screen.getByText('创建应用')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: 'app-a' })).not.toBeInTheDocument();
  });

  it('当用户点击应用名称时，应进入该应用详情', async () => {
    await renderPage();
    await userEvent.click(await screen.findByRole('button', { name: 'app-a' }));
    expect(mocks.push).toHaveBeenCalled();
  });
});
