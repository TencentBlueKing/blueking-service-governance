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
 * 场景级测试：环境管理（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S9）
 *
 * 覆盖九组用户可感知行为（#208「去宽窄表」重构后校准 V 7 → 9）：
 *   列表有数据 / 空列表 → 列表 V = 2
 *   删除域 V = 7（入口已从列表行移至 basic-info.vue:457-482「删除环境」按钮 → delete-env-action.vue）：
 *   打开确认 / 名称不一致禁用 / 删除成功 / 删除失败保留 / 部署应用告警拦截 /
 *   获取详情失败提示 / 删除进行中防重复提交
 *
 * mock 边界：
 *   - @blueking/table stub（纯 field 列 jsdom 不产出 DOM，见 PLAYBOOK「vxe 垫片 + 表格 stub」）
 *   - 列表页 CreateEnv / PublicEnvVarsSideslider 契约 stub（非本场景验证对象，且前者依赖 pinia store）
 *   - 删除域以 harness 复刻 basic-info 的真实接线（button @click show(envData) + @deleted），
 *     DeleteEnvAction / DeleteEnvDialog / DeployedAppsWarning 全部真实渲染（bkui-vue 真实组件）
 *   - bkui-vue Message partial mock：删除成功 / 获取详情失败的手写 Message 是删除域的用户可感知信号
 *
 * 业务约定：HTTP 错误由 fetch interceptor 统一反馈，删除失败时组件静默保留弹窗（仅不关闭），按现状断言。
 * 注意：getEnv mock 必须返回对象（空对象即可）——delete-env-action.vue:81 直接解引用
 *       detail.appDeployStatuses，mock 成 undefined 会误入「获取详情失败」分支。
 */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';
import { MockedApiError } from '../helpers/mocked-api-error';

const mocks = vi.hoisted(() => ({
  listEnvs: vi.fn(),
  deleteEnv: vi.fn(),
  getEnv: vi.fn(),
  message: vi.fn(),
  deleted: vi.fn(),
  // 动态注入 harness 按钮传给 DeleteEnvAction.show() 的环境数据（复刻 basic-info 的 envData）
  envData: { current: null as null | Record<string, unknown> },
}));

beforeAll(() => {
  installVxeShims();
});

vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());
vi.mock('~/api/modules/v1', () => ({
  EnvService: { listEnvs: mocks.listEnvs, getEnv: mocks.getEnv, deleteEnv: mocks.deleteEnv },
  WorkspaceService: { getWorkspace: vi.fn().mockResolvedValue({}) },
  BkintegrationsBkmonitorService: { listApms: vi.fn().mockResolvedValue([]) },
}));
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({ route: { path: '/ws-1/env', name: 'env', params: { space: 'ws-1' } } });
});
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'ws-1' }) }));

// CreateEnv / PublicEnvVarsSideslider 不在本场景验证范围：
// 前者 setup 依赖 pinia store，后者挂载即调未 mock 的 EnvvarsService（会拖垮渲染树）→ 契约 stub
vi.mock('~/pages/env/create-env.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'CreateEnvStub',
      props: { isShow: Boolean },
      template: '<div data-testid="create-env-stub" />',
    }),
  };
});
vi.mock('~/pages/env/public-env-vars/public-env-vars-sideslider.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'PublicEnvVarsSidesliderStub',
      props: { visible: Boolean, space: String },
      template: '<div data-testid="public-env-vars-stub" />',
    }),
  };
});
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
// Message 命令式 API partial mock（保留 Dialog/Button/Input 等真实渲染）
vi.mock('bkui-vue', async importOriginal => ({
  ...((await importOriginal<object>()) as object),
  Message: mocks.message,
}));

// 各字段取不同值，避免多列渲染出相同文本导致查询歧义
const env = (name: string) => ({
  id: 'env-id-1',
  name,
  displayName: '环境A',
  type: 'test',
  appCount: 0,
  creator: 'tester',
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listEnvs.mockResolvedValue([env('env-a')]);
  mocks.getEnv.mockResolvedValue({ appDeployStatuses: [] });
  mocks.deleteEnv.mockResolvedValue({});
  mocks.envData.current = { id: 'env-id-1', name: 'env-a', displayName: '环境A' };
});

async function renderPage() {
  const { default: EnvPage } = await import('~/pages/env/env.vue');
  // 页面内 CreateEnv 等子组件 setup 依赖 pinia store，必须自装 Pinia——
  // 否则依赖其他测试文件残留的全局 activePinia，在 CI 的 worker 调度下会随机失败
  const pinia = createPinia();
  setActivePinia(pinia);
  return render(EnvPage as never, {
    props: { space: 'ws-1' } as never,
    global: {
      plugins: [pinia],
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
    } as never,
  });
}

describe('环境管理：列表状态', () => {
  it('当环境列表加载完成时，应展示环境名称', async () => {
    await renderPage();
    await waitFor(() => expect(screen.getByText('env-a')).toBeInTheDocument());
  });

  it('当环境列表为空时，应不显示任何环境', async () => {
    mocks.listEnvs.mockResolvedValue([]);
    await renderPage();
    await waitFor(() => expect(mocks.listEnvs).toHaveBeenCalled());
    expect(screen.queryByText('env-a')).not.toBeInTheDocument();
  });
});

/* ------------------------------------------------------------------ */
/* 删除域：#208 重构后入口在 basic-info.vue 的「删除环境」按钮，          */
/* 实际逻辑在 delete-env-action.vue（defineExpose { show }）。           */
/* harness 复刻 basic-info.vue:457-482 的接线：button show(envData) + @deleted */
/* ------------------------------------------------------------------ */

const { default: DeleteEnvAction } = await import('~/pages/env/components/delete-env-action.vue');

const DeleteHarness = defineComponent({
  name: 'DeleteEnvActionHarness',
  setup() {
    const actionRef = ref();
    return () =>
      h('div', [
        h('button', { type: 'button', onClick: () => actionRef.value?.show(mocks.envData.current) }, '删除环境'),
        h(DeleteEnvAction as never, { ref: actionRef, onDeleted: () => mocks.deleted() }),
      ]);
  },
});

/** 点击「删除环境」（无部署应用基线）并等确认弹窗就绪，返回弹窗内删除按钮 */
async function openConfirmDialog() {
  renderHarness();
  await userEvent.click(screen.getByRole('button', { name: '删除环境' }));
  await waitFor(() => expect(screen.getByText(/确定删除环境/)).toBeInTheDocument());
  return screen.getByRole('button', { name: '删除' });
}

function renderHarness() {
  const options = {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
    },
  };
  return render(DeleteHarness as never, options as never);
}

describe('环境管理：删除二次确认', () => {
  it('当用户点击删除环境且环境未部署应用时，应打开删除确认弹窗', async () => {
    const confirmBtn = await openConfirmDialog();
    expect(confirmBtn).toBeDisabled();
    expect(screen.getByText('删除环境后，所有配置、环境变量和操作记录将永久删除')).toBeInTheDocument();
  });

  it('当用户输入的名称与环境不一致时，删除应被禁用', async () => {
    await openConfirmDialog();
    await userEvent.type(screen.getByPlaceholderText('请输入待删除环境名称'), 'env-b');
    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled();
  });

  it('当用户输入正确名称并确认删除时，应删除成功并提示且通知返回列表', async () => {
    await openConfirmDialog();
    await userEvent.type(screen.getByPlaceholderText('请输入待删除环境名称'), 'env-a');
    const confirmBtn = screen.getByRole('button', { name: '删除' });
    await waitFor(() => expect(confirmBtn).toBeEnabled());
    await userEvent.click(confirmBtn);
    // 正向消费信号：删除接口按选中环境调用 + 手写「删除成功」Message（delete-env-action.vue:63）
    await waitFor(() => expect(mocks.deleteEnv).toHaveBeenCalledWith({ envID: 'env-id-1' }));
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ message: '删除成功', theme: 'success' }));
    // 成功后 emit('deleted') → 宿主页路由返回列表（basic-info.vue:481）
    expect(mocks.deleted).toHaveBeenCalledTimes(1);
    // 关闭断言用「不可见」而非「DOM 移除」：delete-env-dialog 未设 render-directive="if"，
    // bkui Dialog 关闭走默认 'show' 模式，DOM 以 v-show 隐藏保留（modal/index.js:473-476）
    await waitFor(() => expect(screen.getByText(/确定删除环境/)).not.toBeVisible());
  });

  it('当删除接口失败时，应保留弹窗让用户可重试', async () => {
    mocks.deleteEnv.mockRejectedValue(new MockedApiError());
    await openConfirmDialog();
    await userEvent.type(screen.getByPlaceholderText('请输入待删除环境名称'), 'env-a');
    const confirmBtn = screen.getByRole('button', { name: '删除' });
    await waitFor(() => expect(confirmBtn).toBeEnabled());
    await userEvent.click(confirmBtn);
    await waitFor(() => expect(mocks.deleteEnv).toHaveBeenCalledWith({ envID: 'env-id-1' }));
    // 失败链路走完的稳定信号：确认按钮 loading 复位（handleConfirm 的 finally），
    // 保证断言时 rejection 已传播完毕，不因「弹窗本就开着」而假绿
    await waitFor(() => expect(confirmBtn).toBeEnabled());
    // 现状行为：失败不关闭弹窗（错误由 interceptor 统一反馈），用户可修正后重试
    expect(screen.getByText(/确定删除环境/)).toBeInTheDocument();
    expect(mocks.message).not.toHaveBeenCalledWith({ message: '删除成功', theme: 'success' });
  });

  it('当环境已部署应用时，应拦截删除并提示先卸载应用', async () => {
    mocks.getEnv.mockResolvedValue({
      appDeployStatuses: [{ appName: 'app-a', appType: 'test', deployStatus: 'running' }],
    });
    renderHarness();
    await userEvent.click(screen.getByRole('button', { name: '删除环境' }));
    // 正向消费信号：delete-env-action.vue:81-84 命中「有已部署应用」分支 → 告警弹窗（含卸载引导与应用列表）
    expect(await screen.findByText('无法删除环境')).toBeInTheDocument();
    expect(screen.getByText('该环境已部署应用，请先卸载后再删除环境')).toBeInTheDocument();
    // 删除确认弹窗不应出现（硬拦截，非警告后仍可继续）
    expect(screen.queryByText(/确定删除环境/)).not.toBeInTheDocument();
    expect(mocks.deleteEnv).not.toHaveBeenCalled();
  });

  it('当获取环境详情失败时，应提示失败且不弹出任何确认框', async () => {
    mocks.getEnv.mockRejectedValue(new MockedApiError());
    renderHarness();
    await userEvent.click(screen.getByRole('button', { name: '删除环境' }));
    // 正向消费信号：delete-env-action.vue:88 手写失败 Message
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith({
        message: '获取环境详情失败，无法验证部署状态',
        theme: 'error',
      }),
    );
    expect(screen.queryByText(/确定删除环境/)).not.toBeInTheDocument();
    expect(screen.queryByText('无法删除环境')).not.toBeInTheDocument();
    expect(mocks.deleteEnv).not.toHaveBeenCalled();
  });

  it('当删除请求进行中重复点击确认时，应只触发一次删除', async () => {
    let resolveDelete!: (value: unknown) => void;
    mocks.deleteEnv.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveDelete = resolve;
        }),
    );
    await openConfirmDialog();
    await userEvent.type(screen.getByPlaceholderText('请输入待删除环境名称'), 'env-a');
    const confirmBtn = screen.getByRole('button', { name: '删除' });
    await waitFor(() => expect(confirmBtn).toBeEnabled());
    await userEvent.click(confirmBtn);
    // 第一次请求仍 pending（deleting 守卫生效），再次点击不得重复触发
    await userEvent.click(confirmBtn);
    resolveDelete({});
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ message: '删除成功', theme: 'success' }));
    expect(mocks.deleteEnv).toHaveBeenCalledTimes(1);
    expect(mocks.deleted).toHaveBeenCalledTimes(1);
  });
});
