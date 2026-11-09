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
 * 配置编辑保存（S3）场景级用例
 *
 * 覆盖 6 条用户可感知路径（台账 V=3，摸底校准为 6）：
 * 1. 点击编辑 → 进入编辑态（保存/取消按钮出现，编辑入口隐藏）
 * 2. 编辑后取消 → 回到查看态且恢复原值
 * 3. 校验不通过 → 展示校验提示且不调用保存接口
 * 4. 默认环境保存 → 走 setAppDefaultAppSpecResources 并回到查看态
 * 5. 普通环境保存 → 走 setEnvAppSpecResources（环境维度多态分支）
 * 6. 普通环境「恢复默认配置」→ 删除环境覆盖
 *
 * 回归覆盖确认等量继承值、低于预留拦截和已存值格式错误。
 *
 * 多态切换由通用 composable `use-app-spec-section.ts` 实现（handleEdit 存快照 /
 * handleCancelEdit 回滚 / handleSave 校验后按环境分派），本轮以资源规格子模块
 * （`components/resources-form.vue`）为该模式的代表进行覆盖。
 *
 * Mock 边界：
 * - AppSpecService 六个方法逐方法 mock（默认配置/环境生效/环境覆盖/写默认/写环境/删覆盖）
 * - Message 命令式 API mock；useGPAConfigPolling mock（自动扩缩容轮询非本场景判定分支）
 * - bkui-vue 真实渲染（表单校验、Select、Button 均为可感知行为载体）
 */
import { defineComponent, h, nextTick, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ResourcesForm from '~/pages/application/detail/app-config/components/resources-form.vue';
import { useAppDetail } from '~/stores/app-detail';

import { i18nGlobalMocks, i18nPlugin } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

import type { ExtendedEnv } from '~/pages/application/detail/app-config/components/types';

const DEFAULT_SPEC = {
  replicas: 2,
  cpuRequests: '1',
  cpuLimits: '2',
  memoryRequests: '1Gi',
  memoryLimits: '2Gi',
};

const DEFAULT_ENV: ExtendedEnv = { name: 'default', isDefault: true, isModified: true };
const PROD_ENV: ExtendedEnv = { name: 'prod', isDefault: false, isModified: true };

const mocks = vi.hoisted(() => ({
  getDefault: vi.fn(),
  getEnvEffective: vi.fn(),
  getEnvOverride: vi.fn(),
  setDefault: vi.fn(),
  setEnv: vi.fn(),
  deleteEnv: vi.fn(),
  message: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
}));

vi.mock('~/api/modules/v1', () => ({
  AppSpecService: {
    getAppDefaultAppSpecResources: mocks.getDefault,
    getEnvEffectiveAppSpecResources: mocks.getEnvEffective,
    getEnvAppSpecResources: mocks.getEnvOverride,
    setAppDefaultAppSpecResources: mocks.setDefault,
    setEnvAppSpecResources: mocks.setEnv,
    deleteEnvAppSpecResources: mocks.deleteEnv,
  },
}));

// GPA 自动扩缩容配置含轮询，非本场景判定分支 → 固定返回「未启用」
vi.mock('~/composables/use-gpa-config-polling', async () => {
  const { ref } = await import('vue');
  return {
    useGPAConfigPolling: () => ({
      config: ref(null),
      enabled: ref(false),
      refresh: vi.fn(),
      updatePolling: vi.fn(),
    }),
  };
});

/**
 * 数据加载由父容器驱动（resources-form.vue:593 defineExpose 暴露 handleEnvChange，
 * loadEnvData 不在组件内自触发），故用 harness 持有 ref 驱动环境加载。
 */
const harnessState: { load?: (env: ExtendedEnv) => Promise<void> } = {};

const SectionHarness = defineComponent({
  name: 'SectionHarness',
  setup() {
    const sectionRef = ref();
    const env = ref<ExtendedEnv>(DEFAULT_ENV);
    harnessState.load = async (next: ExtendedEnv) => {
      env.value = next;
      await nextTick();
      await sectionRef.value?.handleEnvChange(next);
    };
    return () => h(ResourcesForm, { ref: sectionRef, currentEnv: env.value });
  },
});

async function enterEditMode() {
  await userEvent.click(await screen.findByRole('button', { name: '编辑' }));
  expect(await screen.findByRole('button', { name: '保存' })).toBeTruthy();
}

function getQuantityControl(label: string, selector: string) {
  const control = screen
    .getByText(label, { exact: true })
    .closest('.bk-form-item')
    ?.querySelector<HTMLElement>(selector);
  if (!control) throw new Error(`未找到资源规格字段 ${label} 的控件`);
  return control;
}

/** 编辑态下唯一的类型为 number 的输入框即「实例数」（bkui FormItem label 未做 aria 关联） */
function getReplicasInput() {
  return screen.getByRole('spinbutton');
}

/** 驱动环境数据加载，并等待查看态渲染出实例数（正向信号：数据已回填） */
async function loadEnv(env: ExtendedEnv, expectedReplicas: string) {
  await harnessState.load?.(env);
  expect(await screen.findByText(expectedReplicas, { exact: true })).toBeTruthy();
}

async function openCustomQuantity(label: string) {
  await userEvent.click(getQuantityControl(label, '.bk-select-trigger'));
  // BKUI 下拉弹层没有可查询的 role，用可见状态定位当前菜单。
  const popup = await waitFor(() => {
    const openMenu = document.querySelector<HTMLElement>('.bk-select-popover.visible');
    if (!openMenu) throw new Error('资源规格下拉未打开');
    return within(openMenu);
  });
  await userEvent.click(popup.getByRole('button', { name: '自定义' }));
  return popup;
}

function renderResourcesForm() {
  const pinia = createPinia();
  setActivePinia(pinia);
  const store = useAppDetail();
  // use-app-spec-section.ts:386 watch 仅在 appType ∈ {trpc, taf} 时拉取默认配置
  store.updateAppID('app-1');
  store.updateAppType('trpc');

  return render(SectionHarness, {
    global: { plugins: [pinia, i18nPlugin], mocks: i18nGlobalMocks },
  });
}

async function setCustomQuantity(label: string, amount: string) {
  const popup = await openCustomQuantity(label);
  const input = popup.getByRole('textbox', { name: '自定义' });
  await userEvent.clear(input);
  await userEvent.type(input, amount);
  await userEvent.click(popup.getByRole('button', { name: '确定' }));
}

beforeEach(() => {
  vi.clearAllMocks();
});

// 页面链路触碰 @blueking/table 的 DOM 工具（HTMLDocument 等），沿用共享垫片
installVxeShims();
Element.prototype.scrollIntoView = vi.fn();

describe('配置编辑：资源规格多态切换', () => {
  it('当用户点击编辑时，应进入编辑态并展示保存与取消操作', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    expect(screen.getByRole('button', { name: '取消' })).toBeTruthy();
    // 编辑态下不再展示进入编辑的入口（BkmsContent :show-edit-icon="!isEditing"）
    expect(screen.queryByRole('button', { name: '编辑' })).toBeNull();
  });

  it('当用户修改后点击取消时，应回到查看态且恢复原配置', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    const replicasInput = getReplicasInput();
    await userEvent.clear(replicasInput);
    await userEvent.type(replicasInput, '9');
    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    // 正向消费信号：回到查看态（编辑入口回归）
    expect(await screen.findByRole('button', { name: '编辑' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: '保存' })).toBeNull();
    expect(screen.getByText('2', { exact: true })).toBeTruthy();
  });

  it('当保存接口失败时，应停留在编辑态且不提示保存成功', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    mocks.setDefault.mockRejectedValue(new Error('保存失败'));
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    // 正向消费信号：保存请求已发出（失败分支在 handleSave 内 catch）
    await waitFor(() => expect(mocks.setDefault).toHaveBeenCalledTimes(1));
    expect(screen.getByRole('button', { name: '保存' })).toBeTruthy();
    await waitFor(() => expect(mocks.message).not.toHaveBeenCalled());
  });

  it('当用户在默认环境保存有效配置时，应写入默认配置并回到查看态', async () => {
    let storedSpec = { ...DEFAULT_SPEC, cpuRequests: '200m', memoryRequests: '128Mi' };
    mocks.getDefault.mockImplementation(async () => storedSpec);
    mocks.setDefault.mockImplementation(async ({ appSpecResources }) => {
      storedSpec = appSpecResources;
      return {};
    });
    const { unmount } = renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    await setCustomQuantity('CPU 限制 (Limits)', '89');
    await setCustomQuantity('内存限制 (Limits)', '128');
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.setDefault).toHaveBeenCalledWith({
        appID: 'app-1',
        appSpecResources: {
          replicas: 2,
          cpuRequests: '200m',
          cpuLimits: '89',
          memoryRequests: '128Mi',
          memoryLimits: '128Gi',
        },
      }),
    );
    expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'success' }));
    expect(await screen.findByRole('button', { name: '编辑' })).toBeTruthy();

    unmount();
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    expect(screen.getByText('89 核')).toBeInTheDocument();
    expect(screen.getByText('128Gi')).toBeInTheDocument();
  });

  it('当用户切换内存单位时，应保留输入并在确认后回填', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();

    const requestLabel = '内存预留 (Requests)';
    const requestPopup = await openCustomQuantity(requestLabel);
    const input = requestPopup.getByRole('textbox', { name: '自定义' });
    const units = within(requestPopup.getByRole('group', { name: '内存单位' }));
    expect(input).toHaveValue('');
    expect(units.getByRole('button', { name: 'Gi', pressed: true })).toBeInTheDocument();

    await userEvent.type(input, '768');
    await userEvent.click(units.getByRole('button', { name: 'Mi' }));
    expect(units.getByRole('button', { name: 'Mi', pressed: true })).toBeInTheDocument();
    expect(input).toHaveValue('768');
    expect(getQuantityControl(requestLabel, '.bk-select-trigger input')).toHaveValue('1Gi');

    await userEvent.click(requestPopup.getByRole('button', { name: '确定' }));
    expect(getQuantityControl(requestLabel, '.bk-select-trigger input')).toHaveValue('768Mi');
    expect(requestPopup.getByRole('button', { name: '自定义' })).toBeVisible();

    const limitLabel = '内存限制 (Limits)';
    const limitPopup = await openCustomQuantity(limitLabel);
    expect(limitPopup.getByRole('button', { name: 'Gi', pressed: true })).toBeInTheDocument();
    await userEvent.click(limitPopup.getByRole('button', { name: 'Mi' }));
    await userEvent.type(limitPopup.getByRole('textbox', { name: '自定义' }), '1536');
    await userEvent.click(limitPopup.getByRole('button', { name: '取消' }));
    expect(getQuantityControl(limitLabel, '.bk-select-trigger input')).toHaveValue('2Gi');
  });

  it('当用户在普通环境保存配置时，应写入该环境的覆盖配置', async () => {
    const defaults = { ...DEFAULT_SPEC, cpuRequests: '200m' };
    let overrides: Partial<typeof DEFAULT_SPEC> = { replicas: 4, cpuRequests: '1.5', cpuLimits: '3' };
    mocks.getDefault.mockResolvedValue(defaults);
    mocks.getEnvEffective.mockImplementation(async () => ({ ...defaults, ...overrides }));
    mocks.getEnvOverride.mockImplementation(async () => overrides);
    mocks.setEnv.mockImplementation(async ({ appSpecResources }) => {
      overrides = Object.fromEntries(Object.entries(appSpecResources).filter(([, value]) => value !== null));
      return {};
    });
    const { unmount } = renderResourcesForm();
    await loadEnv(PROD_ENV, '4');
    await enterEditMode();
    // 单项重置图标没有可查询的角色，用所属表单字段定位。
    await userEvent.click(getQuantityControl('实例数', '.bkms-icon-return'));
    const replicasInput = getReplicasInput();
    expect(replicasInput).toHaveValue(2);
    await userEvent.clear(replicasInput);
    await userEvent.type(replicasInput, '5');
    await userEvent.click(getQuantityControl('CPU 预留 (Requests)', '.bkms-icon-return'));
    const requestDraft = await openCustomQuantity('CPU 预留 (Requests)');
    expect(requestDraft.getByRole('textbox', { name: '自定义' })).toHaveValue('0.2');
    await userEvent.click(requestDraft.getByRole('button', { name: '确定' }));
    await userEvent.click(getQuantityControl('CPU 限制 (Limits)', '.bkms-icon-return'));
    await setCustomQuantity('CPU 限制 (Limits)', '89');
    await setCustomQuantity('内存限制 (Limits)', '128');
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.setEnv).toHaveBeenCalledWith({
        appID: 'app-1',
        envName: 'prod',
        appSpecResources: {
          replicas: 5,
          cpuRequests: null,
          cpuLimits: '89',
          memoryRequests: null,
          memoryLimits: '128Gi',
        },
      }),
    );
    expect(mocks.setDefault).not.toHaveBeenCalled();
    expect(await screen.findByRole('button', { name: '编辑' })).toBeTruthy();

    unmount();
    renderResourcesForm();
    await loadEnv(PROD_ENV, '5');
    expect(screen.getByText('0.2 核')).toBeInTheDocument();
    expect(screen.getByText('89 核')).toBeInTheDocument();
    expect(screen.getByText('128Gi')).toBeInTheDocument();

    await enterEditMode();
    await userEvent.click(getQuantityControl('实例数', '.bkms-icon-return'));
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.setEnv).toHaveBeenLastCalledWith(
        expect.objectContaining({ appSpecResources: expect.objectContaining({ replicas: null }) }),
      ),
    );
  });

  it('当用户将限制设得低于预留时，应拦截保存', async () => {
    const defaults = { ...DEFAULT_SPEC, cpuRequests: '2', memoryRequests: '1Gi' };
    mocks.getDefault.mockResolvedValue(defaults);
    mocks.getEnvEffective.mockResolvedValue({ ...defaults, replicas: 4 });
    mocks.getEnvOverride.mockResolvedValue({ replicas: 4 });
    renderResourcesForm();
    await loadEnv(PROD_ENV, '4');
    await enterEditMode();

    // 打开自定义预填当前值，点取消丢弃草稿后已选值保持原样
    const requestDraft = await openCustomQuantity('CPU 预留 (Requests)');
    expect(requestDraft.getByRole('textbox', { name: '自定义' })).toHaveValue('2');
    await userEvent.click(requestDraft.getByRole('button', { name: '取消' }));
    await setCustomQuantity('CPU 限制 (Limits)', '1');
    await setCustomQuantity('内存限制 (Limits)', '0.5');
    await userEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => expect(screen.getAllByText('{0} Limits 不能小于 Requests')).toHaveLength(2));
    expect(mocks.setEnv).not.toHaveBeenCalled();
  });

  it('当用户保存含无效资源值的配置时，应提示格式错误并拦截保存', async () => {
    mocks.getDefault.mockResolvedValue({ ...DEFAULT_SPEC, cpuRequests: 'invalid' });
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));

    expect(await screen.findByText('格式错误')).toBeInTheDocument();
    expect(mocks.setDefault).not.toHaveBeenCalled();
  });

  it('当用户在普通环境点击恢复默认配置时，应删除该环境的覆盖配置', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    mocks.getEnvEffective.mockResolvedValue({ ...DEFAULT_SPEC, replicas: 4 });
    mocks.getEnvOverride.mockResolvedValue({ replicas: 4 });
    mocks.deleteEnv.mockResolvedValue({});
    renderResourcesForm();
    await loadEnv(PROD_ENV, '4');
    await enterEditMode();
    await userEvent.click(await screen.findByRole('button', { name: '恢复默认配置' }));
    await waitFor(() =>
      expect(mocks.deleteEnv).toHaveBeenCalledWith(expect.objectContaining({ appID: 'app-1', envName: 'prod' })),
    );
    expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'success' }));
  });
});
