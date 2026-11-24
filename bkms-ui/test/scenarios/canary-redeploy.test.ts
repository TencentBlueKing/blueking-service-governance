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
 * 场景级测试：部署管理 - 灰度发布中允许重新部署（ROUTES 未收录，本头即场景说明）
 *
 * 灰度进行中「部署」保持可用：提交前需二次确认，确认后本次灰度结束、实例按本次配置更新；
 * 侧栏「仅镜像（原地重启）」在灰度中不可用；隐藏环境在灰度中只剩原地重启，入口保持禁用。
 *
 * V = 5
 * 判定节点：
 * ① 顶部「部署」是否可用：只看 latest-status 的 inProgressReleaseMode，灰度进行中不再禁用
 * ② 侧栏顶部提示：灰度中展示「重新部署将结束本次灰度」说明
 * ③ 原地重启入口：灰度中「仅镜像（原地重启）」禁用（联邦集群同样禁用）
 * ④ 提交前置确认：灰度中点「部署」先弹确认框，取消不提交、确认才提交
 * ⑤ 隐藏环境 + 灰度：只剩原地重启可用 → 部署入口保持禁用
 * 路径：① 可见环境灰度中 → 部署可点 → 侧栏提示 + 原地重启禁用 → 确认后按填写内容提交
 *       ② 确认框取消 → 不发起部署
 *       ③ 非灰度 → 不弹确认框，直接提交（回归保护）
 *       ④ 隐藏环境灰度中 → 部署入口禁用
 * stub/mock：DeployService（灰度进度）、InfoBox / Message 命令式 API、部署侧栏内的镜像选择与
 *            分支选择、EnvSelectPanel 仅 emit envList、实例列表等非验证目标子组件。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeployPage from '~/pages/application/detail/deploy/deploy.vue';
import { useAppDetail } from '~/stores/app-detail';
import { useDeployEnvStore } from '~/stores/deploy-env';
import { useTrpcDeployStore } from '~/stores/trpc-deploy';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';

const mocks = vi.hoisted(() => ({
  env: {
    cluster: { isFederation: false },
    displayName: '生产环境',
    kind: 'standard',
    name: 'prod',
    status: 'Ready',
    type: 'production',
  },
  latest: vi.fn(),
  records: vi.fn(),
  create: vi.fn(),
  progress: vi.fn(),
  operate: vi.fn(),
  infoBox: vi.fn(),
  message: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue');
  const route = reactive({ params: { space: 'ws-1', name: 'app-a' }, query: { activeTab: 'instance' } });
  return {
    useRoute: () => route,
    useRouter: () => ({
      replace: ({ query }: { query: Record<string, unknown> }) => {
        route.query = query as { activeTab: string };
      },
    }),
  };
});
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'ws-1' }) }));
// 确认框是命令式 API，需要断言标题 / 说明 / 按钮文案与「确认后才提交」的时序。
vi.mock('bkui-vue', async importOriginal => {
  const actual = await importOriginal<object>();
  return { ...actual, InfoBox: mocks.infoBox, Message: mocks.message };
});
vi.mock('~/api/modules/v1', () => ({
  EnvService: { listFeatureEnvs: vi.fn().mockResolvedValue([]) },
  AppSpecService: { getEnvEffectiveAppSpecResources: vi.fn().mockResolvedValue({ replicas: 2 }) },
  DeployService: {
    getBatchReleaseProgress: mocks.progress,
    operateBatchRelease: mocks.operate,
  },
  InstanceService: { updateAppInstances: vi.fn().mockResolvedValue({}) },
}));
vi.mock('~/pages/application/detail/deploy/use-deploy', () => ({
  useDeployAPIs: () => ({
    listLatestDeployRecords: mocks.latest,
    listDeployRecords: mocks.records,
    createDeployDirectly: mocks.create,
    buildAndCreateDeploy: vi.fn(),
  }),
}));
vi.mock('~/pages/application/detail/deploy/use-federation-resource-precheck', () => ({
  useFederationResourcePrecheck: () => ({ check: vi.fn().mockResolvedValue([]) }),
}));
vi.mock('~/pages/application/detail/deploy/use-env-var-precheck', () => ({
  useEnvVarPrecheck: () => ({ check: vi.fn().mockResolvedValue({}) }),
}));
vi.mock('~/pages/application/components/image-select.vue', () => ({
  default: {
    props: { value: String },
    emits: ['update:value'],
    template: '<input aria-label="镜像 Tag" :value="value" @input="$emit(\'update:value\', $event.target.value)" />',
  },
}));
vi.mock('~/components/code-revision/code-revision-field.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/components/env-select-panel/env-select-panel.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      emits: ['update:envList'],
      mounted() {
        this.$emit('update:envList', [mocks.env]);
      },
      template: '<div />',
    }),
  };
});
vi.mock('~/pages/application/detail/deploy/instance-list/instance-list.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/multi-env-instance-table.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/scale-instance.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/overview/deploy-overview.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/quickly-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/remove-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/deploy-history.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/deploy-event.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/components/topo/index.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/create-feature-env.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/feature-env-sideslider.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/components/view-build-log/index.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/components/view-build-log/build-log-panel.vue', () => ({
  default: { template: '<div />' },
}));

const FlexRowStub = { template: '<div><slot name="left" /><slot name="right" /></div>' };

/** 灰度进行中的 latest-status：inProgressReleaseMode 是后端判定「灰度占用中」的唯一口径。 */
const CANARY_IN_PROGRESS = {
  deployID: 'deploy-1',
  inProgressReleaseMode: 'canary',
  releaseMode: 'canary',
  stage: 'deploy',
  status: 'deploying',
};

/** 侧栏底部的主按钮（与顶部按钮同名，打开侧栏后取最后一个）。 */
async function clickSidebarDeploy() {
  const buttons = await screen.findAllByRole('button', { name: '部署' });
  await userEvent.click(buttons[buttons.length - 1]);
}

/** 顶部「部署」按钮（打开部署侧栏的入口）。 */
async function findDeployEntry() {
  return await screen.findByRole('button', { name: '部署' });
}

function mountPage() {
  const view = render(DeployPage, {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
      stubs: {
        ColorIcon: true,
        FlexRow: FlexRowStub,
        KeyValueBadge: true,
        Popover: { template: '<div><slot /></div>' },
        TabHeader: true,
      },
      directives: { 'bk-tooltips': {} },
    },
  });
  // 环境选择由环境列表在挂载后回填，这里在挂载完成后指定当前环境，等价于真实页面的环境切换结果
  useDeployEnvStore().updateCurrentEnv(mocks.env.name);
  return view;
}

/** 打开部署侧栏并填写镜像 Tag，进入「可提交」状态。 */
async function openSidebarAndFillTag() {
  await userEvent.click(await findDeployEntry());
  expect(
    await screen.findByText('当前环境正在灰度发布，重新部署将结束本次灰度，并按本次配置更新实例'),
  ).toBeInTheDocument();
  await userEvent.type(await screen.findByRole('textbox', { name: '镜像 Tag' }), 'new-tag');
}

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  window.sessionStorage.clear();
  setActivePinia(createPinia());
  const appStore = useAppDetail();
  appStore.updateAppID('app-a');
  appStore.updateAppType('trpc');
  appStore.appDetail = { id: 'app-a', visibleEnvNames: [] };
  mocks.env.cluster.isFederation = false;
  useTrpcDeployStore().updateCurEnvItem(mocks.env as never);

  mocks.latest.mockResolvedValue(CANARY_IN_PROGRESS);
  mocks.records.mockResolvedValue({ results: [{ imageTag: 'image:old' }] });
  mocks.create.mockResolvedValue({});
  // 进度快照为空：本文件只验证部署入口与侧栏交互，横幅内容由 batch-release-banner 单测覆盖。
  mocks.progress.mockResolvedValue(null);
  mocks.operate.mockResolvedValue({});
});

describe('灰度发布中的重新部署', () => {
  it('当环境正在灰度发布时，顶部部署仍可点击，侧栏提示会结束灰度且不再提供原地重启', async () => {
    mountPage();
    expect(await findDeployEntry()).toBeEnabled();

    await openSidebarAndFillTag();

    // 原地重启会打断灰度，侧栏里只能选择镜像+配置 / 仅配置
    expect(screen.getByRole('radio', { name: '镜像+配置' })).toBeEnabled();
    expect(screen.getByRole('radio', { name: '仅配置' })).toBeEnabled();
    expect(screen.getByRole('radio', { name: '仅镜像（原地重启）' })).toBeDisabled();
  });

  it('当灰度中提交部署时，应先确认结束灰度，确认后才按本次配置提交', async () => {
    mountPage();
    await openSidebarAndFillTag();
    await clickSidebarDeploy();

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    const confirmConfig = mocks.infoBox.mock.calls[0][0];
    expect(confirmConfig.confirmText).toBe('重新部署');
    // 弹窗形态对齐设计稿：42px 警示图标 + 居中标题 + 自带浅色块的说明文字
    expect(confirmConfig.extCls).toBe('canary-redeploy-infobox');
    expect(confirmConfig.width).toBe(480);
    expect(confirmConfig.showContentBgColor).toBe(true);
    expect(confirmConfig.content).toBe('本次灰度发布将结束，实例按本次配置更新');
    expect(confirmConfig.headerAlign).toBe('center');
    expect(confirmConfig.footerAlign).toBe('center');
    const titleVNode = confirmConfig.title as {
      children: { children: unknown; props: Record<string, unknown> }[];
      props: { class: string };
    };
    expect(titleVNode.props.class).toBe('canary-redeploy-warning-title');
    const [iconVNode, titleTextVNode] = titleVNode.children;
    expect(iconVNode.props.icon).toBe('bkms-icon-tishi');
    expect(iconVNode.props.width).toBe(42);
    expect(iconVNode.props.height).toBe(42);
    expect(titleTextVNode.children).toBe('确认结束灰度并重新部署？');
    // 未确认前不发起部署
    expect(mocks.create).not.toHaveBeenCalled();

    confirmConfig.onConfirm();
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({
        appID: 'app-a',
        envName: 'prod',
        imageTag: 'new-tag',
        replicas: 2,
      }),
    );
  });

  it('当用户在确认框选择取消时，不应发起部署', async () => {
    mountPage();
    await openSidebarAndFillTag();
    await clickSidebarDeploy();

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    // 取消即不调用 onConfirm：部署接口保持未调用
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('当环境未在灰度中时，提交部署不弹确认框，直接按填写内容部署', async () => {
    mocks.latest.mockResolvedValue({ deployID: 'deploy-1', stage: 'deploy', status: 'deployed' });
    mountPage();
    await userEvent.click(await findDeployEntry());
    // 侧栏打开后再断言：非灰度场景既不提示「结束灰度」，也不弹确认框
    expect(await screen.findByRole('radio', { name: '镜像+配置' })).toBeChecked();
    expect(screen.queryByText('当前环境正在灰度发布，重新部署将结束本次灰度，并按本次配置更新实例')).toBeNull();
    await userEvent.type(await screen.findByRole('textbox', { name: '镜像 Tag' }), 'new-tag');
    await clickSidebarDeploy();

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({
        appID: 'app-a',
        envName: 'prod',
        imageTag: 'new-tag',
        replicas: 2,
      }),
    );
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('当隐藏环境正在灰度发布时，只剩原地重启可用，部署入口保持禁用', async () => {
    useAppDetail().appDetail = { id: 'app-a', visibleEnvNames: ['test'] };
    mountPage();

    expect(await findDeployEntry()).toBeDisabled();
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
