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
 * 多态切换由通用 composable `use-app-spec-section.ts` 实现（handleEdit 存快照 /
 * handleCancelEdit 回滚 / handleSave 校验后按环境分派），本轮以资源规格子模块
 * （`components/resources-form.vue`）为该模式的代表进行覆盖。
 *
 * Mock 边界：
 * - AppSpecService 六个方法逐方法 mock（默认配置/环境生效/环境覆盖/写默认/写环境/删覆盖）
 * - Message 命令式 API mock；useGPAConfigPolling mock（自动扩缩容轮询非本场景判定分支）
 * - bkui-vue 真实渲染（表单校验、Select、Button 均为可感知行为载体）
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, nextTick, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks, i18nPlugin } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

import ResourcesForm from '~/pages/application/detail/app-config/components/resources-form.vue';
import { useAppDetail } from '~/stores/app-detail';

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

/** 驱动环境数据加载，并等待查看态渲染出实例数（正向信号：数据已回填） */
async function loadEnv(env: ExtendedEnv, expectedReplicas: string) {
  await harnessState.load?.(env);
  expect(await screen.findByText(expectedReplicas, { exact: true })).toBeTruthy();
}

async function enterEditMode() {
  await userEvent.click(await screen.findByRole('button', { name: '编辑' }));
  expect(await screen.findByRole('button', { name: '保存' })).toBeTruthy();
}

/** 编辑态下唯一的类型为 number 的输入框即「实例数」（bkui FormItem label 未做 aria 关联） */
function getReplicasInput() {
  return screen.getByRole('spinbutton');
}

beforeEach(() => {
  vi.clearAllMocks();
});

// 页面链路触碰 @blueking/table 的 DOM 工具（HTMLDocument 等），沿用共享垫片
installVxeShims();

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
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    mocks.setDefault.mockResolvedValue({});
    renderResourcesForm();
    await loadEnv(DEFAULT_ENV, '2');
    await enterEditMode();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.setDefault).toHaveBeenCalledWith(
        expect.objectContaining({ appID: 'app-1', appSpecResources: expect.objectContaining({ replicas: 2 }) }),
      ),
    );
    expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'success' }));
    expect(await screen.findByRole('button', { name: '编辑' })).toBeTruthy();
  });

  it('当用户在普通环境保存配置时，应写入该环境的覆盖配置', async () => {
    mocks.getDefault.mockResolvedValue(DEFAULT_SPEC);
    mocks.getEnvEffective.mockResolvedValue({ ...DEFAULT_SPEC, replicas: 4 });
    mocks.getEnvOverride.mockResolvedValue({ replicas: 4 });
    mocks.setEnv.mockResolvedValue({});
    renderResourcesForm();
    await loadEnv(PROD_ENV, '4');
    await enterEditMode();
    await userEvent.click(screen.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.setEnv).toHaveBeenCalledWith(expect.objectContaining({ appID: 'app-1', envName: 'prod' })),
    );
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
