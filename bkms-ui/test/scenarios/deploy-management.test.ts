/**
 * 场景级测试：提交部署 / 部署管理（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S2）
 *
 * 覆盖两组用户可感知行为（聚焦部署入口的权限分发）：
 *   模型类应用展示特性环境入口 / 非模型类应用不展示该入口 → V = 2
 *
 * 稳定策略：本轮只证明 TabHeader 上「特性环境入口」的 appType 分发，
 * 环境选择器 / 实例列表 / 拓扑 / 侧栏等重型子树一律 stub，避免冷编译与真实请求把用例推近 testTimeout。
 */
import { render, waitFor, within } from '@testing-library/vue';
import { createPinia } from 'pinia';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

const harness = vi.hoisted(() => ({
  appType: 'trpc' as string,
  listFeatureEnvs: vi.fn().mockResolvedValue([]),
}));

/** 轻量标记组件：避免拉起真实子树的编译与请求 */
const markStub = async (name: string) => {
  const { defineComponent } = await import('vue');
  return { default: defineComponent({ name, template: `<div data-testid="${name}"></div>` }) };
};

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({
    route: { path: '/ws-1/app/app-a/detail/deploy', name: 'deploy', params: { space: 'ws-1' } },
  });
});

// Vitest 对命名导出不会走 Proxy.get，须显式给出 EnvService / AppSpecService
vi.mock('~/api/modules/v1', () => ({
  EnvService: {
    listFeatureEnvs: harness.listFeatureEnvs,
    listAppEnvs: vi.fn().mockResolvedValue([]),
  },
  AppSpecService: {
    getEnvEffectiveAppSpecResources: vi.fn().mockResolvedValue(null),
  },
}));
vi.mock('~/api/modules/bkmsserver', async () => {
  const { createAnyServiceMock } = await import('../helpers/mock-service');
  return { ApiServerService: createAnyServiceMock('ApiServerService') };
});

vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({
    appID: 'app-a',
    appType: harness.appType,
    appDetail: {},
    loading: false,
    updateAppName: vi.fn(),
    updateAppID: vi.fn(),
  }),
}));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1' }),
}));
vi.mock('~/stores/trpc-deploy', () => ({
  useTrpcDeployStore: () => ({
    curEnvItem: null,
    updateCurEnvItem: vi.fn(),
  }),
}));
vi.mock('~/stores/deploy-env', () => ({
  useDeployEnvStore: () => ({
    currentEnv: '',
    selectedEnvs: [],
    envList: [],
    updateCurrentEnv: vi.fn(),
    updateSelectedEnvs: vi.fn(),
    updateEnvList: vi.fn(),
    getAppEnvSelection: () => null,
    updateAppEnvSelection: vi.fn(),
    clearCurrentEnv: vi.fn(),
  }),
}));

// 预检链路会继续拉联邦/环境变量接口，本场景不覆盖提交路径
vi.mock('~/pages/application/detail/deploy/use-deploy-precheck', async () => {
  const { ref } = await import('vue');
  return {
    useDeployPrecheck: () => ({
      cancelDeploy: vi.fn(),
      continueDeploy: vi.fn(),
      federationMismatches: ref([]),
      isShowPrecheckDialog: ref(false),
      precheck: vi.fn().mockResolvedValue(true),
      precheckEnvName: ref(''),
      undefinedVars: ref([]),
    }),
  };
});

// 与入口显隐无关的重型子树全部 stub，缩短 transform / 挂载时间
vi.mock('~/components/env-select-panel.vue', () => markStub('EnvSelectPanelStub'));
vi.mock('~/pages/application/detail/deploy/overview/deploy-overview.vue', () => markStub('DeployOverviewStub'));
vi.mock('~/pages/application/detail/deploy/deploy-history.vue', () => markStub('DeployHistoryStub'));
vi.mock('~/pages/application/detail/deploy/quickly-deploy.vue', () => markStub('QuicklyDeployStub'));
vi.mock('~/pages/application/detail/deploy/instance-list/full-update.vue', () => markStub('FullUpdateStub'));
vi.mock('~/pages/application/detail/deploy/instance-list/instance-list.vue', () => markStub('InstanceListStub'));
vi.mock('~/pages/application/detail/deploy/instance-list/multi-env-instance-table.vue', () =>
  markStub('MultiEnvInstanceTableStub'),
);
vi.mock('~/pages/application/detail/deploy/instance-list/scale-instance.vue', () => markStub('ScaleInstanceStub'));
vi.mock('~/pages/application/detail/deploy/deploy-event.vue', () => markStub('DeployEventStub'));
vi.mock('~/pages/application/components/topo/index.vue', () => markStub('ResourceTopologyStub'));
vi.mock('~/pages/application/detail/deploy/create-feature-env.vue', () => markStub('CreateFeatureEnvStub'));
vi.mock('~/pages/application/detail/deploy/feature-env-sideslider.vue', () => markStub('FeatureEnvSidesliderStub'));
vi.mock('~/pages/application/detail/deploy/remove-deploy.vue', () => markStub('RemoveDeployStub'));
vi.mock('~/pages/application/detail/deploy/env-var-precheck-dialog.vue', () => markStub('EnvVarPrecheckDialogStub'));
vi.mock('~/pages/application/detail/components/view-build-log/index.vue', () => markStub('ViewBuildLogStub'));

beforeEach(() => {
  vi.clearAllMocks();
  harness.appType = 'trpc';
  harness.listFeatureEnvs.mockResolvedValue([]);
});

/** 预热页面模块：冷 transform 从 it 里挪出，避免慢机撞 testTimeout */
beforeAll(async () => {
  await import('~/pages/application/detail/deploy/deploy.vue');
}, 60_000);

async function renderPage() {
  const { default: DeployPage } = await import('~/pages/application/detail/deploy/deploy.vue');
  return render(DeployPage as never, {
    global: {
      plugins: [createPinia()],
      mocks: i18nGlobalMocks,
      // 直译 keypath，保证「应用关联的特性环境」可被断言
      components: { 'i18n-t': I18nTStub },
    } as never,
  });
}

describe('部署管理：特性环境入口按应用类型分发', () => {
  // 全量并行时首包仍可能偏慢，单条放宽避免级联污染下一条
  it('当应用为模型类（trpc）时，应展示特性环境入口', async () => {
    const { container } = await renderPage();
    const view = within(container);
    await waitFor(() => expect(view.getByText(/应用关联的特性环境/)).toBeInTheDocument(), {
      timeout: 5_000,
    });
  }, 60_000);

  it('当应用非模型类时，不应展示特性环境入口', async () => {
    harness.appType = 'default';
    const { container } = await renderPage();
    const view = within(container);
    // 用 container 缩小查询范围，避免 Teleport/残留 DOM 造成「部署管理」多匹配
    await waitFor(() => expect(view.getByText('部署管理')).toBeInTheDocument(), { timeout: 5_000 });
    expect(view.queryByText(/应用关联的特性环境/)).not.toBeInTheDocument();
  }, 60_000);
});
