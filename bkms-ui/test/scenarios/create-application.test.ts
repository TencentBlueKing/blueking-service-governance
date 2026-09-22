/**
 * 创建应用向导（S1）场景级用例
 *
 * 覆盖 8 条用户可感知路径（台账预估 V=4~5，摸底校准为 8）：
 * 1. tRPC 模板流程步骤条三步
 * 2. Helm 模板流程步骤条两步（路由级分支：STEPS_CONFIG 按 appType 取）
 * 3. 模板搜索无匹配的空态
 * 4. 选中模板 → 下一步跳转对应向导
 * 5. 参数配置校验不通过 → 停留不前进
 * 6. 参数配置校验通过 → 进入应用配置步骤（创建按钮出现）
 * 7. 创建成功 → 结果页成功态 + 查看应用入口
 * 8. 创建失败 → 结果页失败态 + 失败原因 + 重新创建
 *
 * Mock 边界：
 * - ApiServerService（创建应用/ID 后缀）vi.mock 隔离
 * - vue-router：基于共享 createRouterMock，额外覆写 useRoute（本场景需按用例切换 route.name）
 *   并补充 RouterView 占位（容器页只断言步骤条，子路由内容不在本轮范围）
 * - 两个表单子件（param-config / app-config）按真实 defineExpose 契约 stub
 * - MsHeader 契约 stub（渲染插槽，步骤条在其插槽内）
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks, i18nPlugin } from '../helpers/mock-i18n';

import CreateApp from '~/pages/application/create.vue';
import TemplateSelect from '~/pages/application/template/index.vue';
import TrpcWizard from '~/pages/application/template/trpc/index.vue';

const mocks = vi.hoisted(() => ({
  createApp: vi.fn(),
  getAppIDAutoSuffix: vi.fn(),
  push: vi.fn(),
  back: vi.fn(),
  paramValidate: vi.fn(),
  paramGetValue: vi.fn(),
  appValidate: vi.fn(),
  appGetValue: vi.fn(),
  resetStatus: vi.fn(),
  routerBack: vi.fn(),
  infoBox: vi.fn(),
}));

/** 按用例切换的路由状态（步骤条读取 route.name 决定步数） */
const routeState = vi.hoisted(() => ({ name: 'createTrpcTemplateApp', params: { space: 'ws-1' } }));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

// use-leave-confirm 用 InfoBox 弹离开确认（命令式 API），mock 后即可断言「是否弹过」
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  InfoBox: mocks.infoBox,
}));

vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  const base = await createRouterMock({ push: mocks.push, route: { params: routeState.params } });
  return {
    ...base,
    // 向导取消走 router.back()（trpc/index.vue:198），共享 helper 未暴露 back，此处补上
    useRouter: () => ({ ...base.useRouter(), back: mocks.routerBack }),
    // 共享 mock 的 route 为静态对象；本场景需按用例切换 route.name（create.vue:101 依此取步骤配置）
    useRoute: () => ({
      name: routeState.name,
      params: routeState.params,
      query: {},
      path: '/ws-1/create',
      fullPath: '/ws-1/create',
      meta: {},
      matched: [],
    }),
  };
});

/**
 * create.vue 模板里的 <RouterView> 是全局注册组件（resolveComponent 解析，不走模块 mock），
 * 需在 render 时全局注册桩：<RouterView v-slot="{ Component }"> 必须回传插槽参数，
 * 否则解构 undefined 直接崩；传入占位组件供 <component :is> 渲染。
 */
const RouterPlaceholder = defineComponent({ name: 'RouterPlaceholder', template: '<div />' });
const RouterViewStub = defineComponent({
  name: 'RouterViewStub',
  setup(_props, { slots }) {
    return () => h('div', { 'data-testid': 'router-view-stub' }, slots.default?.({ Component: RouterPlaceholder }));
  },
});

vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: {
    CreateApp: mocks.createApp,
    GetAppIDAutoSuffix: mocks.getAppIDAutoSuffix,
  },
}));

// 真实契约：create.vue 的步骤条位于 MsHeader 插槽内 → stub 渲染插槽
vi.mock('~/components/ms-header.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'MsHeaderStub',
      template: '<div data-testid="ms-header-stub"><slot /></div>',
    }),
  };
});

// 真实契约（param-config.vue:390 / :527 / :589）：validate 异步 boolean、getValue 同步对象、getAppIDAutoSuffix 异步
vi.mock('~/pages/application/template/trpc/param-config.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'ParamConfigStub',
      methods: {
        validate: () => mocks.paramValidate(),
        getValue: () => mocks.paramGetValue(),
        getAppIDAutoSuffix: () => mocks.getAppIDAutoSuffix(),
      },
      template: '<div data-testid="param-config-stub" />',
    }),
  };
});

// 真实契约（app-config.vue:170 / :194 / :201）：getValue 同步对象、validate 异步 boolean、resetStatus 同步
vi.mock('~/pages/application/template/trpc/app-config.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'AppConfigStub',
      methods: {
        getValue: () => mocks.appGetValue(),
        validate: () => mocks.appValidate(),
        resetStatus: () => mocks.resetStatus(),
      },
      template: '<div data-testid="app-config-stub" />',
    }),
  };
});

function renderCreatePage() {
  return render(CreateApp, {
    global: {
      plugins: [createPinia(), i18nPlugin],
      mocks: i18nGlobalMocks,
      components: { RouterView: RouterViewStub },
    },
  });
}

function renderTemplateSelect() {
  const pinia = createPinia();
  setActivePinia(pinia);
  return render(TemplateSelect, {
    global: { plugins: [pinia, i18nPlugin], mocks: i18nGlobalMocks },
  });
}

/**
 * 向导的 step 由父级容器传入（create.vue 持有 curStep 并监听 next/pre/update-step），
 * 单测中用 harness 复现该状态流转，否则 emits('next') 不会推进步骤。
 */
const WizardHarness = defineComponent({
  name: 'WizardHarness',
  setup() {
    const step = ref(2);
    return () =>
      h(TrpcWizard, {
        step: step.value,
        onNext: () => {
          step.value += 1;
        },
        onPre: () => {
          step.value -= 1;
        },
        'onUpdate-step': (value: number) => {
          step.value = value;
        },
      });
  },
});

function renderTrpcWizard() {
  return render(WizardHarness, {
    global: { plugins: [createPinia(), i18nPlugin], mocks: i18nGlobalMocks },
  });
}

describe('创建应用向导：步骤条随模板类型', () => {
  it('当用户进入 tRPC 模板创建流程时，步骤条应展示选择模板/参数配置/创建应用三步', async () => {
    routeState.name = 'createTrpcTemplateApp';
    renderCreatePage();
    expect(await screen.findByText('选择模板')).toBeTruthy();
    expect(screen.getByText('参数配置')).toBeTruthy();
    expect(screen.getByText('创建应用')).toBeTruthy();
  });

  it('当用户进入 Helm 模板创建流程时，步骤条应只展示选择模板与参数配置两步', async () => {
    routeState.name = 'createHelmTemplateApp';
    renderCreatePage();
    expect(await screen.findByText('选择模板')).toBeTruthy();
    expect(screen.getByText('参数配置')).toBeTruthy();
    expect(screen.queryByText('创建应用')).toBeNull();
  });
});

describe('创建应用向导：模板选择', () => {
  it('当用户搜索无匹配模板时，应展示搜索为空状态', async () => {
    renderTemplateSelect();
    // 搜索框为 input[type=search]，ARIA role 是 searchbox
    await userEvent.type(await screen.findByRole('searchbox'), '不存在的模板');
    expect(await screen.findByText('搜索为空')).toBeTruthy();
    expect(screen.queryByRole('button', { name: '下一步' })).toBeNull();
  });

  it('当用户选择 Helm 模板并点下一步时，应进入 Helm 创建向导', async () => {
    renderTemplateSelect();
    await userEvent.click(await screen.findByText('Helm 应用'));
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    // 正向消费信号：路由跳转至 Helm 模板向导
    await waitFor(() =>
      expect(mocks.push).toHaveBeenCalledWith(expect.objectContaining({ name: 'createHelmTemplateApp' })),
    );
  });
});

describe('创建应用向导：tRPC 参数与应用配置', () => {
  it('当参数配置校验不通过时，应停留在参数配置步骤且不创建应用', async () => {
    mocks.paramValidate.mockResolvedValue(false);
    renderTrpcWizard();
    await userEvent.click(await screen.findByRole('button', { name: '下一步' }));
    // 正向消费信号：校验被消费
    await waitFor(() => expect(mocks.paramValidate).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole('button', { name: '创建' })).toBeNull();
    expect(mocks.createApp).not.toHaveBeenCalled();
  });

  it('当用户在向导中点击取消时，应直接回退上一步且不弹离开确认', async () => {
    mocks.paramValidate.mockResolvedValue(true);
    mocks.paramGetValue.mockReturnValue({});
    renderTrpcWizard();
    await userEvent.click(await screen.findByRole('button', { name: '取消' }));
    // 正向消费信号：trpc/index.vue:194-199 cancel → router.back()
    await waitFor(() => expect(mocks.routerBack).toHaveBeenCalledTimes(1));
    // 现状：useLeaveConfirm() 未传 formModel（:103），isDirty 恒 false → 不会弹 InfoBox
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('当参数配置校验通过时，应进入应用配置步骤并展示创建按钮', async () => {
    mocks.paramValidate.mockResolvedValue(true);
    mocks.paramGetValue.mockReturnValue({});
    renderTrpcWizard();
    await userEvent.click(await screen.findByRole('button', { name: '下一步' }));
    expect(await screen.findByRole('button', { name: '创建' })).toBeTruthy();
    expect(screen.getByTestId('app-config-stub')).toBeTruthy();
  });

  it('当创建应用成功时，应展示创建成功结果页与查看应用入口', async () => {
    mocks.paramValidate.mockResolvedValue(true);
    mocks.paramGetValue.mockReturnValue({});
    mocks.appValidate.mockResolvedValue(true);
    mocks.appGetValue.mockReturnValue({ content: 'a: 1', command: [], args: [], env: [] });
    mocks.createApp.mockResolvedValue({});
    renderTrpcWizard();
    // 参数配置 → 应用配置
    await userEvent.click(await screen.findByRole('button', { name: '下一步' }));
    await userEvent.click(await screen.findByRole('button', { name: '创建' }));
    expect(await screen.findByText(/应用创建成功/)).toBeTruthy();
    expect(screen.getByRole('button', { name: '查看应用' })).toBeTruthy();
  });

  it('当创建应用失败时，应展示创建失败结果页与失败原因并可重新创建', async () => {
    mocks.paramValidate.mockResolvedValue(true);
    mocks.paramGetValue.mockReturnValue({});
    mocks.appValidate.mockResolvedValue(true);
    mocks.appGetValue.mockReturnValue({ content: 'a: 1', command: [], args: [], env: [] });
    mocks.createApp.mockRejectedValue({ msg: '代码仓库不可达' });
    renderTrpcWizard();
    await userEvent.click(await screen.findByRole('button', { name: '下一步' }));
    await userEvent.click(await screen.findByRole('button', { name: '创建' }));
    expect(await screen.findByText(/应用创建失败/)).toBeTruthy();
    expect(screen.getByText('代码仓库不可达')).toBeTruthy();
    expect(screen.getByRole('button', { name: '重新创建' })).toBeTruthy();
  });
});
