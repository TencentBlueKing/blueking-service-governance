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
 * 场景级测试：Helm 部署与预览回滚（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S15）
 *
 * 覆盖七条用户可感知路径 → V = 7：
 *   1. 部署进行中时更新按钮禁用并提示（index.vue 状态机）
 *   2. 部署表单必填校验拦截（deploy-application.vue）
 *   3. 填写完整 → 预览 → 确定部署成功（deploy-application.vue 正向主路径）
 *   4. 预览发现未定义变量 → 预检弹窗「仍然部署」进预览（deploy-application.vue）
 *   5. 预检弹窗「取消」停留配置步骤（deploy-application.vue）
 *   6. 部署历史第 1 页首行禁用回滚、非首行可点并弹出回滚弹窗（deploy-history.vue）
 *   7. 回滚弹窗确认成功（preview-rollback.vue）
 *
 * mock 边界：
 *   - bkui-vue 真实渲染（Form/Select/Button/Dialog/Sideslider）
 *   - 部署历史为 vxe 表格（@blueking/table）→ stub 表格本体 + 垫片并用（S9/S14 方案）
 *   - MsEditor（monaco diff）/ EnvUndefinedTips（i18n-t 插值）/ PreviewRollback（历史场景内）/
 *     index 重型容器子件（EnvSelectPanel/ResourceTopology/deploy-history/DeployApplication）→ 契约 stub
 *   - use-helm-deploy.ts 导出模块级单例 ref（deployHistoryList/chartList/latestDeployStatus），
 *     测试间会泄漏 → beforeEach 手动重置
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nPlugin } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

beforeAll(() => {
  installVxeShims();
});

const mocks = vi.hoisted(() => ({
  listHelmDeployRecords: vi.fn(),
  previewHelmDeploy: vi.fn(),
  createHelmDeploy: vi.fn(),
  previewRollbackHelmDeploy: vi.fn(),
  rollbackHelmDeploy: vi.fn(),
  listEnvTrafficLanes: vi.fn(),
  listAppConfigFiles: vi.fn(),
  listDeployableImageTags: vi.fn(),
  message: vi.fn(),
}));

vi.mock('~/api/modules/v1', () => ({
  DeployService: {
    listHelmDeployRecords: mocks.listHelmDeployRecords,
    previewHelmDeploy: mocks.previewHelmDeploy,
    createHelmDeploy: mocks.createHelmDeploy,
    previewRollbackHelmDeploy: mocks.previewRollbackHelmDeploy,
    rollbackHelmDeploy: mocks.rollbackHelmDeploy,
    deleteHelmDeploy: vi.fn(),
  },
  EnvService: { listEnvTrafficLanes: mocks.listEnvTrafficLanes },
  HelmChartsService: { listChartVersions: vi.fn().mockResolvedValue([]) },
  AppConfigFilesService: { listAppConfigFiles: mocks.listAppConfigFiles },
  ImagesService: { listDeployableImageTags: mocks.listDeployableImageTags },
}));

vi.mock('~/api/modules/bkmsserver', () => ({
  // fetchAppDetail 用：返回 null，避免覆盖测试预置的 store 状态
  ApiServerService: { GetApp: vi.fn().mockResolvedValue(null) },
}));

vi.mock('~/stores/space', () => ({
  // useHelmDeploy 仅消费 currentSpace 做请求参数与 truthy 判断
  useSpaceStore: () => ({ currentSpace: 'ws-1' }),
}));

vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  // 直译并处理 {0} 形式的插值（如「下一步：{0}」）
  useI18n: () => ({
    t: (s: string, params?: unknown[]) => {
      let out = s;
      if (Array.isArray(params)) params.forEach((p, i) => (out = out.replace(`{${i}}`, String(p))));
      return out;
    },
    te: () => true,
  }),
}));

vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({
    route: { path: '/ws-1/app/app-a/detail/helm-deploy', name: 'detail', params: { space: 'ws-1' } },
  });
});

vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
  InfoBox: vi.fn(),
}));

// monaco diff 编辑器：预览视图是否可见以标题标记承担
vi.mock('~/components/monaco-editor/ms-editor.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      props: { title: String, targetTitle: String },
      template: '<div data-testid="ms-editor-stub">{{ title }}|{{ targetTitle }}</div>',
    }),
  };
});

// 预检弹窗（i18n-t 插值组件）：契约 stub。真实组件（env-undefined-tips.vue:207-220）
// 在取消/仍部署/前往修改时均先置 isShow=false 再 emit，stub 保持同一契约
vi.mock('~/pages/application/detail/helm-deploy/env-undefined-tips.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      props: { isShow: Boolean },
      emits: ['update:isShow', 'cancel', 'go-modify', 'still-deploy'],
      template: `<div data-testid="env-undefined-tips-stub" v-if="isShow">
        <button data-testid="still-deploy-btn" @click="$emit('update:isShow', false); $emit('still-deploy')">仍然部署</button>
        <button data-testid="cancel-tips-btn" @click="$emit('update:isShow', false); $emit('cancel')">取消</button>
      </div>`,
    }),
  };
});

// 回滚弹窗：deploy-history 场景内 stub（其本体行为在独立 describe 真实渲染）
vi.mock('~/pages/application/detail/helm-deploy/preview-rollback.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      props: { isShow: Boolean },
      emits: ['success'],
      template: '<div data-testid="preview-rollback-stub" v-if="isShow">回滚应用</div>',
    }),
  };
});

// —— index.vue 的重型容器子件契约 stub ——

// 环境选择：mounted 自动 emit 指定环境（真实组件行为是拉环境列表 + 自动选首环境）
vi.mock('~/components/env-select-panel.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      props: { modelValue: String, type: String },
      emits: ['update:modelValue', 'update:item'],
      async mounted() {
        await Promise.resolve();
        this.$emit('update:modelValue', 'env-1');
        this.$emit('update:item', { name: 'env-1', displayName: '环境一', type: 'dev' });
      },
      template: '<div data-testid="env-select-panel-stub">env-select</div>',
    }),
  };
});

vi.mock('~/components/status-icon.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      props: ['status'],
      template: '<span data-testid="status-icon-stub">{{ status }}</span>',
    }),
  };
});

vi.mock('~/pages/application/components/topo/index.vue', async () => {
  const { defineComponent } = await import('vue');
  return { default: defineComponent({ name: 'ResourceTopologyStub', template: '<div>资源拓扑</div>' }) };
});

vi.mock('~/pages/application/detail/helm-deploy/deploy-history.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'DeployHistoryStub',
      template: '<div data-testid="deploy-history-stub">部署历史</div>',
    }),
  };
});

// 注意：deploy-application.vue 不做文件级 mock——部署流程用例直接渲染真实组件；
// index 场景下其 isShow 初始为 false 且无 mount 请求，闲置无害。

// —— vxe 表格 stub（conventions 模板，部署历史列为插槽渲染）——
vi.mock('@blueking/table', async () => {
  const { defineComponent, h, Comment } = await import('vue');
  const TableColumnStub = defineComponent({
    props: { field: String, label: String, type: String },
    setup(_p, { slots }) {
      return () => (slots.default ? h('span', slots.default({ row: {}, rowIndex: 0 })) : h(Comment, ''));
    },
  });
  const TableStub = defineComponent({
    props: { data: { type: Array, default: () => [] } },
    setup(props, { slots }) {
      return () => {
        const columns = (slots.default?.() ?? []).filter(Boolean);
        const rows = props.data as Record<string, unknown>[];
        const body = rows.length
          ? rows.map((row, rowIndex) =>
              h(
                'div',
                { key: rowIndex, 'data-testid': 'table-row' },
                columns.map((col: { children?: { default?: (s: unknown) => unknown } }, i: number) =>
                  h('span', { key: i }, [col.children?.default ? col.children.default({ row, rowIndex }) : '']),
                ),
              ),
            )
          : slots.empty?.();
        return h('div', { 'data-testid': 'table-stub' }, [body]);
      };
    },
  });
  return { Table: TableStub, TableColumn: TableColumnStub };
});

import { useHelmDeploy } from '~/pages/application/detail/helm-deploy/use-helm-deploy';
import { useAppDetail } from '~/stores/app-detail';

// 全局 $t 直译（使用共享 i18nPlugin）

const envItem = { name: 'env-1', displayName: '环境一', type: 'dev' };

const latestRecord = {
  id: 'deploy-1',
  chartVersion: '2.0.0',
  imageTag: 'img-2',
  status: 'success',
  values: 'latest-values',
  valuesFileID: 'vf-1',
  operator: 'op-a',
  createdAt: '2026-09-08T10:00:00Z',
  updatedAt: '2026-09-08T10:00:00Z',
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listAppConfigFiles.mockResolvedValue({ items: [] });
  mocks.listDeployableImageTags.mockResolvedValue({ results: [] });
  // useHelmDeploy 内部消费 pinia store，先激活
  setActivePinia(createPinia());
  // use-helm-deploy 模块级单例 ref：测试间手动重置，防状态泄漏
  const { deployHistoryList, chartList, latestDeployStatus } = useHelmDeploy();
  deployHistoryList.value = [];
  chartList.value = [];
  latestDeployStatus.value = '';
});

/** 打开侧滑（isShow false → true 触发数据加载 watch），等数据加载完成后返回 */
async function openDeployApplication() {
  const utils = await renderDeployApplication();
  await utils.rerender({ isShow: true, deployType: 'RollingUpdate', envItem, laneName: '' } as never);
  // 完成信号：侧滑打开时的两个列表请求都已发起
  await waitFor(() => expect(mocks.listAppConfigFiles).toHaveBeenCalled());
  return utils;
}

/** 打开侧滑并预填完整表单（历史回填 Chart 版本/Values 文件 + UI 选择镜像 Tag） */
async function openFilledDeployApplication(previewResult: Record<string, unknown>) {
  const { deployHistoryList } = useHelmDeploy();
  deployHistoryList.value = [latestRecord];
  mocks.listDeployableImageTags.mockResolvedValue({ results: [{ tag: 'v1.2.3' }] });
  mocks.previewHelmDeploy.mockResolvedValue(previewResult);
  const utils = await openDeployApplication();
  // 镜像 Tag 无回填，需用户从下拉选择（bkui Select 真实渲染）
  const imageTagLabel = screen.getByText('镜像 Tag');
  const formItem = imageTagLabel.closest('.bk-form-item') as HTMLElement;
  await userEvent.click(formItem.querySelector('.bk-select') as HTMLElement);
  await userEvent.click(await screen.findByText('v1.2.3'));
  return utils;
}

async function renderDeployApplication() {
  const { default: DeployApplication } = await import('~/pages/application/detail/helm-deploy/deploy-application.vue');
  return render(DeployApplication as never, {
    props: { isShow: false, deployType: 'RollingUpdate', envItem, laneName: '' },
    global: { plugins: [createPinia(), i18nPlugin] } as never,
  });
}

describe('Helm 部署/更新侧滑：部署流程', () => {
  it('当用户未选必填项点「下一步」时，应显示校验错误态且不调用预览接口', async () => {
    const utils = await openDeployApplication();
    // 回填参数来自部署历史，此测试保持历史为空 → 表单三项均为空
    // 侧滑 wrapper 显隐由 bkui modal 内部 setTimeout 置位，findBy 自带重试
    await userEvent.click(await screen.findByRole('button', { name: '下一步：预览部署' }));
    // 正向消费信号：表单项进入错误态（bkui 错误文案经 tooltip 展示，jsdom 不可断言，
    // 以 form-item 的 is-error 态为可见信号）
    await waitFor(() => expect(document.body.querySelector('.bk-form-item.is-error')).toBeTruthy(), { timeout: 5000 });
    expect(mocks.previewHelmDeploy).not.toHaveBeenCalled();
  });

  it('当用户填写完整并确定部署时，应经预览调用创建部署接口并提示部署成功', async () => {
    const utils = await openFilledDeployApplication({ current: 'a: 1', target: 'a: 2' });
    // 下一步 → 预览视图出现
    await userEvent.click(await screen.findByRole('button', { name: '下一步：预览部署' }));
    await waitFor(() => expect(mocks.previewHelmDeploy).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId('ms-editor-stub')).toBeTruthy();
    // 确定部署
    await userEvent.click(await screen.findByRole('button', { name: '确定部署' }));
    await waitFor(() => expect(mocks.createHelmDeploy).toHaveBeenCalledTimes(1));
    expect(mocks.createHelmDeploy).toHaveBeenCalledWith(
      expect.objectContaining({ imageTag: 'v1.2.3', chartVersion: '2.0.0' }),
      expect.anything(),
    );
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'success', message: '部署成功' })),
    );
    // 通知父组件刷新部署历史
    await waitFor(() => expect(utils.emitted('deploy')).toBeTruthy());
  });

  it('当预览发现未定义变量时，应弹出预检弹窗；点「仍然部署」应进入预览', async () => {
    await openFilledDeployApplication({ current: 'a: 1', target: 'a: 2', missingEnvVars: ['VAR_X'], missingVars: [] });
    await userEvent.click(await screen.findByRole('button', { name: '下一步：预览部署' }));
    expect(await screen.findByTestId('env-undefined-tips-stub')).toBeTruthy();
    await userEvent.click(screen.getByTestId('still-deploy-btn'));
    expect(await screen.findByTestId('ms-editor-stub')).toBeTruthy();
  });

  it('当用户在预检弹窗点「取消」时，应停留在配置步骤且不进入预览', async () => {
    await openFilledDeployApplication({ current: 'a: 1', target: 'a: 2', missingEnvVars: [], missingVars: ['VAR_Y'] });
    await userEvent.click(await screen.findByRole('button', { name: '下一步：预览部署' }));
    await screen.findByTestId('env-undefined-tips-stub');
    await userEvent.click(screen.getByTestId('cancel-tips-btn'));
    // 正向信号：预检弹窗关闭
    await waitFor(() => expect(screen.queryByTestId('env-undefined-tips-stub')).toBeNull());
    expect(screen.queryByTestId('ms-editor-stub')).toBeNull();
  });
});

describe('部署历史与回滚：入口与确认', () => {
  it('当部署记录为第 1 页首行时，「回滚到此版本」应禁用；非首行点击应弹出回滚弹窗', async () => {
    const oldRecord = { ...latestRecord, id: 'deploy-0', chartVersion: '1.0.0', updatedAt: '2026-09-07T10:00:00Z' };
    mocks.listHelmDeployRecords.mockResolvedValue({ count: '2', results: [latestRecord, oldRecord] });
    // deploy-history 的文件级 stub 仅供 index 场景使用，此处取真实组件
    const { default: DeployHistory } = await vi.importActual<
      typeof import('~/pages/application/detail/helm-deploy/deploy-history.vue')
    >('~/pages/application/detail/helm-deploy/deploy-history.vue');
    render(DeployHistory as never, {
      props: { envName: 'env-1', laneName: '', skipInitialFetch: false },
      global: { plugins: [createPinia(), i18nPlugin] } as never,
    });
    // 表格行渲染完成（stub 表格 + 插槽列）
    const rows = await screen.findAllByTestId('table-row');
    expect(rows.length).toBe(2);
    const rollbackBtns = screen.getAllByRole('button', { name: '回滚到此版本' });
    expect(rollbackBtns[0]).toBeDisabled();
    expect(rollbackBtns[1]).toBeEnabled();
    await userEvent.click(rollbackBtns[1]);
    expect(await screen.findByTestId('preview-rollback-stub')).toBeTruthy();
  });

  it('当用户在回滚弹窗确认时，应调用回滚接口、提示回滚成功并关闭弹窗', async () => {
    mocks.previewRollbackHelmDeploy.mockResolvedValue({ current: 'a: 1', target: 'a: 0' });
    // preview-rollback 的文件级 stub 仅供 deploy-history 场景使用，此处取真实组件
    const { default: PreviewRollback } = await vi.importActual<
      typeof import('~/pages/application/detail/helm-deploy/preview-rollback.vue')
    >('~/pages/application/detail/helm-deploy/preview-rollback.vue');
    const { emitted } = render(PreviewRollback as never, {
      props: { isShow: true, appName: 'app-a', envName: 'env-1', deployID: 'deploy-0', trafficLaneName: '' },
      global: { plugins: [createPinia(), i18nPlugin] } as never,
    });
    // 打开即拉取回滚预览（Dialog @shown），diff 就绪后确定按钮可用
    await waitFor(() => expect(mocks.previewRollbackHelmDeploy).toHaveBeenCalled());
    const confirmBtn = await screen.findByRole('button', { name: '确定' });
    await waitFor(() => expect(confirmBtn).toBeEnabled());
    await userEvent.click(confirmBtn);
    await waitFor(() => expect(mocks.rollbackHelmDeploy).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({ theme: 'success', message: '回滚成功' })),
    );
    await waitFor(() => expect(emitted('success')).toBeTruthy());
  });
});

describe('Helm 部署容器页：更新入口状态机', () => {
  it('当最新部署状态为进行中（pending-upgrade）时，更新按钮应禁用', async () => {
    // 预置应用上下文（index.vue 依赖 appID/appType 触发历史拉取）
    const appDetailStore = useAppDetail();
    appDetailStore.updateAppID('app-1');
    appDetailStore.updateAppType('helm');
    appDetailStore.updateAppName('app-a');
    mocks.listEnvTrafficLanes.mockResolvedValue([]);
    mocks.listHelmDeployRecords.mockResolvedValue({
      count: '1',
      results: [{ ...latestRecord, status: 'pending-upgrade' }],
    });
    const { default: HelmDeploy } = await import('~/pages/application/detail/helm-deploy/index.vue');
    render(HelmDeploy as never, {
      global: { plugins: [i18nPlugin] } as never,
    });
    // 环境选择 stub 自动 emit → 拉泳道与部署历史 → latestDeployStatus = pending-upgrade
    const updateBtn = await screen.findByRole('button', { name: '更新' });
    await waitFor(() => expect(updateBtn).toBeDisabled());
  });
});
