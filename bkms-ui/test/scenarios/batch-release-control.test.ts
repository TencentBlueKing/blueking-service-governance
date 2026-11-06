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
 * 场景级测试：部署管理 - 实例列表分批发布控制横幅（ROUTES 未收录，本头即场景说明）
 *
 * V = 10
 * 判定节点：① 最新记录是否为分批发布（决定是否请求进度与渲染横幅）
 *           ② deploying 展示控制横幅；deployed 展示成功 Alert；failed 展示失败 + 回退/回滚；
 *              rolled-back 展示已回滚提示；卸载/取消等不展示
 *           ③ updating 态操作集合（暂停 + 回滚）与「更多」里的回退/跳过
 *           ④ manual-paused 态操作集合（继续发布 + 回滚 + 更多里的回退/跳过）
 *           ⑤ timed-paused 未到期提供「暂停」入口可提前转人工；到期/无倒计时与 manual-paused 一致
 *           ⑥ batchPhase 已有但 currentBatchIndex 尚未回填时仍要给出按钮
 *           ⑦ 需二次确认的操作（回滚/回退/跳过）与直接执行的操作（暂停/继续）
 *           ⑧ 操作成功提示文案（继续发布带批次号、回滚带目标版本）
 *           ⑨ 进行中 / 失败禁用顶部「部署」入口，完成 / 已回滚不阻塞
 *           ⑩ 操作在途时所有入口置灰，重复点击不重复发请求
 *           ⑪ 完成态关闭后按 deployID 记住
 *           ⑫ 点击横幅「查看发布进度」隐藏横幅并切换为灰度发布进度视图（独立组件，互斥渲染）
 *           ⑬ 进度视图内操作按钮全部平铺（无「更多」），复用横幅同一操作链路
 *           ⑭ 进度视图点「返回实例列表」复原回横幅态
 * 路径：① 更新中→横幅 + 部署禁用 ② 暂停 ③ 回滚确认/取消 ④ 暂停态回退 + 跳过
 *       ⑤ 定时暂停继续发布 ⑥ 批次序号未回填仍展示按钮 ⑦ 已完成→成功 Alert + 关闭记忆
 *       ⑧ 失败→回退上一批 ⑨ 已回滚→提示 + 部署可用 ⑩ 非分批发布不请求 ⑪ 操作失败保留状态
 *       ⑫ 请求挂起→按钮置灰且不重发→返回后恢复 ⑬ 未部署环境不请求进度
 *       ⑭ 查看发布进度→横幅隐藏 + 面板呈现 ⑮ 面板平铺按钮复用确认链路 ⑯ 返回实例列表
 * stub/mock：DeployService（进度/操作）、Message/InfoBox 命令式 API、Dropdown 只渲染插槽、
 *            InstanceList / 部署总览 / 部署历史等非验证目标子组件、EnvSelectPanel 仅 emit envList、
 *            CanaryInstanceTable 仅验证面板接线（灰度筛选口径由 resolveInstanceCanaryState 纯函数单测覆盖）。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import type { VNode } from 'vue';
import DeployPage from '~/pages/application/detail/deploy/deploy.vue';
import { useAppDetail } from '~/stores/app-detail';
import { useTrpcDeployStore } from '~/stores/trpc-deploy';

import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';

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
  progress: vi.fn(),
  operate: vi.fn(),
  message: vi.fn(),
  infoBox: vi.fn(),
}));

// 横幅文案由占位符拼装（批次号、实例数、版本迁移），这里做真实插值，断言才对齐用户可见文案。
vi.mock('vue-i18n', async importOriginal => {
  const interpolate = (text: string, params?: unknown) => {
    if (!params) return text;
    return text.replace(/\{(\w+)\}/g, (raw, key: string) => {
      const value = Array.isArray(params)
        ? (params as unknown[])[Number(key)]
        : (params as Record<string, unknown>)[key];
      return value === undefined ? raw : String(value);
    });
  };
  return {
    ...((await importOriginal<object>()) as object),
    useI18n: () => ({ te: () => true, t: interpolate }),
  };
});

const i18nGlobalMocks = {
  $t: (text: string, params?: unknown) => {
    if (!params) return text;
    return text.replace(/\{(\w+)\}/g, (raw, key: string) => {
      const value = Array.isArray(params)
        ? (params as unknown[])[Number(key)]
        : (params as Record<string, unknown>)[key];
      return value === undefined ? raw : String(value);
    });
  },
};

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

vi.mock('~/api/modules/v1', () => ({
  EnvService: { listFeatureEnvs: vi.fn().mockResolvedValue([]) },
  AppSpecService: { getEnvEffectiveAppSpecResources: vi.fn().mockResolvedValue({ replicas: 2 }) },
  DeployService: {
    getBatchReleaseProgress: mocks.progress,
    operateBatchRelease: mocks.operate,
  },
}));

// Dropdown 只负责收纳「更多」菜单，弹层交互不是本场景验证目标：直接渲染两个插槽，
// 使菜单项在 jsdom 中可见可点，同时保留 Dropdown.DropdownMenu / Dropdown.DropdownItem 契约。
vi.mock('bkui-vue', async importOriginal => {
  const actual = await importOriginal<object>();
  const { defineComponent, h } = await import('vue');

  const DropdownMenu = defineComponent({
    name: 'DropdownMenuStub',
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.()),
  });
  const DropdownItem = defineComponent({
    name: 'DropdownItemStub',
    emits: ['click'],
    setup:
      (_, { slots, emit }) =>
      () =>
        h('div', { onClick: () => emit('click') }, slots.default?.()),
  });
  const Dropdown = defineComponent({
    name: 'DropdownStub',
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.default?.(), slots.content?.()]),
  }) as unknown as { DropdownItem: typeof DropdownItem; DropdownMenu: typeof DropdownMenu };
  Dropdown.DropdownMenu = DropdownMenu;
  Dropdown.DropdownItem = DropdownItem;

  return { ...actual, Dropdown, InfoBox: mocks.infoBox, Message: mocks.message };
});

vi.mock('~/pages/application/detail/deploy/use-deploy', () => ({
  useDeployAPIs: () => ({ listLatestDeployRecords: mocks.latest }),
}));
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
vi.mock('~/pages/application/detail/deploy/instance-list/instance-list.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'InstanceListStub',
      setup(_, { expose }) {
        // 真实组件 defineExpose({ handleListReleaseInstances })，横幅操作成功后依赖该契约刷新列表。
        expose({ handleListReleaseInstances: vi.fn() });
        return () => null;
      },
    }),
  };
});
// 灰度实例列表不是本文件的验证目标（灰度判定口径由 resolveInstanceCanaryState 纯函数单测覆盖），
// 这里仅保留可寻址的 testid，验证进度视图的挂载与互斥切换。
vi.mock('~/pages/application/detail/deploy/instance-list/components/canary-instance-table.vue', () => ({
  default: { template: '<div data-testid="canary-instance-table" />' },
}));
vi.mock('~/pages/application/detail/deploy/overview/deploy-overview.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/quickly-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/instance-list/multi-env-instance-table.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/scale-instance.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/full-update.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/deploy-history.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/deploy-event.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/components/topo/index.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/create-feature-env.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/remove-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/components/view-build-log/index.vue', () => ({ default: { template: '<div />' } }));

const FlexRowStub = { template: '<div><slot name="left" /><slot name="right" /></div>' };

/** 递归提取 vnode 的纯文本（数字/版本号等插值内容），供 i18n-t stub 拼接。 */
function vnodeText(node: VNode | string): string {
  if (typeof node === 'string') return node;
  const children = node.children;
  if (typeof children === 'string') return children;
  if (Array.isArray(children)) {
    return children.map(child => vnodeText(child as VNode | string)).join('');
  }
  return '';
}

/**
 * i18n-t stub（本文件专用）：把 keypath 的 {0}/{1} 占位符替换为默认插槽子节点的纯文本，
 * 使「共 3 批」「已更新 8/20 个实例」这类插值文案可被 getByText 直接命中
 * （共享 I18nTStub 只直译 keypath）。命名插槽（如横幅的 #index/#version）非断言目标，忽略。
 */
const I18nTInterp = defineComponent({
  name: 'I18nTInterp',
  props: {
    keypath: { type: String, default: '' },
    tag: { type: String, default: 'span' },
  },
  setup(props, { slots }) {
    return () => {
      const children = slots.default?.() ?? [];
      let text = props.keypath;
      for (let i = children.length - 1; i >= 0; i -= 1) {
        text = text.replace(new RegExp(`\\{${i}\\}`), vnodeText(children[i]));
      }
      return h(props.tag, text);
    };
  },
});

// 接口契约里 currentBatchIndex 在批次序号未回填时为 null（d.ts 仅标注 number），测试按真实取值放宽
function buildProgress(
  overrides: Partial<BatchReleaseProgressOutputObj> & { currentBatchIndex?: null | number } = {},
): BatchReleaseProgressOutputObj {
  return {
    batchPhase: 'updating',
    batches: [
      { target: '1', pauseMode: 'permanent' },
      { target: '50%', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
    ],
    currentBatchIndex: 2,
    deployID: 'deploy-1',
    message: '',
    newImageTag: 'v1.8.3',
    oldImageTag: 'v1.8.2',
    replicas: 20,
    status: 'deploying',
    updatedReplicas: 8,
    ...overrides,
  };
}

/** 横幅内按钮文案（按 DOM 顺序），用于校验「主操作 → 更多 → 回滚」的排布；无按钮时返回空数组。 */
function buttonLabelsOf(banner: HTMLElement) {
  return within(banner)
    .queryAllByRole('button')
    .map(button => button.textContent?.trim() ?? '');
}

/** 写操作统一带 { interceptorErr: false }，错误由横幅自己提示。 */
function expectOperateCalledWith(action: string) {
  expect(mocks.operate).toHaveBeenCalledWith({ action, appID: 'app-a', envName: 'prod' }, { interceptorErr: false });
}

/** 挂载后等待首轮「最新部署状态 + 进度」返回，再返回横幅节点。 */
async function findBanner() {
  return await screen.findByTestId('batch-release-banner');
}

function mountPage() {
  return render(DeployPage, {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTInterp },
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
}

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  setActivePinia(createPinia());
  const appStore = useAppDetail();
  appStore.updateAppID('app-a');
  appStore.updateAppType('trpc');
  appStore.appDetail = { id: 'app-a', visibleEnvNames: [] };
  mocks.env.cluster.isFederation = false;
  useTrpcDeployStore().updateCurEnvItem(mocks.env as never);

  mocks.latest.mockResolvedValue({
    deployID: 'deploy-1',
    inProgressReleaseMode: 'canary',
    releaseMode: 'canary',
    stage: 'deploy',
    status: 'deploying',
  });
  mocks.progress.mockResolvedValue(buildProgress());
  mocks.operate.mockResolvedValue({});
});

describe('实例列表分批发布控制：横幅与操作', () => {
  it('当分批发布进行中（updating）时，应展示发布中横幅并禁用顶部部署入口', async () => {
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'updating');
    expect(within(banner).getByText('灰度发布中')).toBeInTheDocument();
    // 本批更新中：直接按钮为「暂停 → 更多 → 回滚」
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '暂停', '更多', '回滚']);
    expect(within(banner).getByRole('button', { name: '暂停' })).toBeEnabled();
    expect(within(banner).getByRole('button', { name: '回滚' })).toBeEnabled();
    expect(within(banner).queryByRole('button', { name: '继续发布' })).toBeNull();
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();

    // 更多里保留回退上一批与跳过剩余批次
    expect(within(banner).getByText('回退上一批')).toBeInTheDocument();
    expect(within(banner).getByText('跳过剩余批次')).toBeInTheDocument();
  });

  it('当用户点击暂停时，应直接调用分批发布接口并提示已暂停', async () => {
    mountPage();
    const banner = await findBanner();

    await userEvent.click(within(banner).getByRole('button', { name: '暂停' }));

    await waitFor(() => expectOperateCalledWith('pause'));
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '已暂停分批发布' }));
    // 暂停不需要二次确认
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('当用户点击回滚时，应二次确认后才调用接口，取消则不发起请求', async () => {
    mountPage();
    const banner = await findBanner();

    await userEvent.click(within(banner).getByRole('button', { name: '回滚' }));

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    const confirmConfig = mocks.infoBox.mock.calls[0][0];
    expect(confirmConfig.title).toBe('确认回滚到旧版本？');
    // 未确认前不发起写操作
    expect(mocks.operate).not.toHaveBeenCalled();

    confirmConfig.onConfirm();
    await waitFor(() => expectOperateCalledWith('rollback'));
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith({
        theme: 'success',
        message: '已开始回滚，实例将恢复为 v1.8.2',
      }),
    );
  });

  it('当分批发布暂停时，应支持回退上一批与更多里的跳过剩余批次，且都需二次确认', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'manual-paused' }));
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'paused');
    expect(within(banner).getByText('第 3 批已暂停')).toBeInTheDocument();
    expect(within(banner).getByRole('button', { name: '继续发布' })).toBeEnabled();
    // 暂停态没有暂停入口；更多位于「继续发布」与「回滚」之间
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '继续发布', '更多', '回滚']);
    expect(within(banner).queryByRole('button', { name: '暂停' })).toBeNull();
    expect(within(banner).queryByRole('button', { name: '回退上一批' })).toBeNull();

    await userEvent.click(within(banner).getByText('回退上一批'));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(mocks.infoBox.mock.calls[0][0].title).toBe('确认回退到上一批？');
    expect(mocks.operate).not.toHaveBeenCalled();
    mocks.infoBox.mock.calls[0][0].onConfirm();
    await waitFor(() => expectOperateCalledWith('backward'));

    await userEvent.click(within(banner).getByText('跳过剩余批次'));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(2));
    expect(mocks.infoBox.mock.calls[1][0].title).toBe('确认跳过剩余批次？');
    mocks.infoBox.mock.calls[1][0].onConfirm();
    await waitFor(() => expectOperateCalledWith('skip'));
  });

  it('当定时暂停已到期（无倒计时数据）时，按钮与人工暂停一致，继续发布直接进入下一批', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'timed-paused' }));
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'paused');
    expect(within(banner).getByRole('button', { name: '继续发布' })).toBeEnabled();
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '继续发布', '更多', '回滚']);

    await userEvent.click(within(banner).getByRole('button', { name: '继续发布' }));
    await waitFor(() => expectOperateCalledWith('continue'));
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('当定时暂停未到期时，应额外提供暂停入口支持提前转人工暂停', async () => {
    mocks.progress.mockResolvedValue(
      buildProgress({
        batchPhase: 'timed-paused',
        batches: [
          { target: '1', pauseMode: 'permanent' },
          { target: '50%', pauseMode: 'permanent' },
          { target: '100%', pauseMode: 'duration', pauseSeconds: 30 },
        ],
        pauseStartTime: new Date(Date.now() - 5_000).toISOString(),
      }),
    );
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'paused');
    // 顺序：主操作（继续发布）→ 暂停 → 更多 → 回滚
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '继续发布', '暂停', '更多', '回滚']);

    await userEvent.click(within(banner).getByRole('button', { name: '暂停' }));
    await waitFor(() => expectOperateCalledWith('pause'));
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '已暂停分批发布' }));
  });

  it('当批次序号尚未回填（batchPhase=updating、currentBatchIndex=null）时，仍应展示暂停与回滚', async () => {
    mocks.progress.mockResolvedValue(
      buildProgress({
        batchPhase: 'updating',
        currentBatchIndex: null,
        message: 'All steps are completed, updating primary workload',
      }),
    );
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'updating');
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '暂停', '更多', '回滚']);
    expect(within(banner).queryByRole('button', { name: '继续发布' })).toBeNull();
    // 拿不到批次序号时无法确认存在上一批：回退上一批置灰展示，点击不发起请求
    expect(within(banner).getByText('回退上一批')).toHaveClass('!pointer-events-none');
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();
  });

  it('当本批尚未开始（pending）时，应展示暂停与回滚，更多里没有回退上一批', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'pending', currentBatchIndex: null }));
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'updating');
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '暂停', '更多', '回滚']);
    expect(within(banner).queryByRole('button', { name: '继续发布' })).toBeNull();
    expect(within(banner).getByText('跳过剩余批次')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();
  });

  it('当分批发布已完成时，应展示成功 Alert 且不阻塞部署入口，关闭后不再出现', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'completed', status: 'deployed' }));
    // 完成后 inProgressReleaseMode 已清空，仅靠 releaseMode=canary 补查一次以拿到完成态结果。
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      inProgressReleaseMode: '',
      releaseMode: 'canary',
      stage: 'deploy',
      status: 'deployed',
    });
    const first = mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'completed');
    expect(within(banner).getByText('灰度发布已完成')).toBeInTheDocument();
    // 完成态没有任何批次操作按钮，只剩右上角无文字的关闭图标按钮（bk-alert 内置 .bk-alert-close）
    expect(buttonLabelsOf(banner).filter(Boolean)).toEqual(['查看发布进度']);
    const closeButton = banner.querySelector('.bk-alert-close');
    expect(closeButton).not.toBeNull();
    // 完成态不再阻塞新的部署入口
    expect(screen.getByRole('button', { name: '部署' })).toBeEnabled();

    await userEvent.click(closeButton as HTMLElement);
    await waitFor(() => expect(screen.queryByTestId('batch-release-banner')).toBeNull());
    first.unmount();

    // 卸载会清空当前环境选择，重新挂载前需要恢复，模拟用户重新进入页面。
    useTrpcDeployStore().updateCurEnvItem(mocks.env as never);
    mountPage();
    await waitFor(() => expect(mocks.progress).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();
  });

  it('当分批发布失败时，应展示失败原因并提供回退上一批与回滚入口（无重试）', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ message: '2 个实例健康检查未通过', status: 'failed' }));
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      inProgressReleaseMode: '',
      releaseMode: 'canary',
      stage: 'deploy',
      status: 'failed',
    });
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'failed');
    expect(within(banner).getByText('2 个实例健康检查未通过')).toBeInTheDocument();
    // 设计稿：失败态无「重试」，回退上一批收进「更多」，回滚平铺
    expect(within(banner).queryByRole('button', { name: '重试' })).toBeNull();
    expect(buttonLabelsOf(banner)).toEqual(['查看发布进度', '更多', '回滚']);
    // 失败后仍需先收尾，顶部部署入口保持禁用
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();

    await userEvent.click(within(banner).getByText('回退上一批'));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(mocks.infoBox.mock.calls[0][0].title).toBe('确认回退到上一批？');
    expect(mocks.operate).not.toHaveBeenCalled();
    mocks.infoBox.mock.calls[0][0].onConfirm();
    await waitFor(() => expectOperateCalledWith('backward'));
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '已回退到上一批' }));
  });

  it('当批次回滚完成时，应展示已回滚提示且不阻塞部署入口', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'completed', status: 'rolled-back' }));
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      inProgressReleaseMode: '',
      releaseMode: 'canary',
      stage: 'deploy',
      status: 'rolled-back',
    });
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'rolledBack');
    expect(within(banner).getByText('已回滚')).toBeInTheDocument();
    // 回滚完成没有任何批次操作，只剩关闭入口（bk-alert 内置 .bk-alert-close）
    expect(buttonLabelsOf(banner).filter(Boolean)).toEqual(['查看发布进度']);
    expect(banner.querySelector('.bk-alert-close')).not.toBeNull();
    // 已回滚为终态，环境已恢复旧版本，不再阻塞部署入口
    expect(screen.getByRole('button', { name: '部署' })).toBeEnabled();
  });

  it('当最新记录不是分批发布时，不应请求进度也不展示横幅', async () => {
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      inProgressReleaseMode: '',
      releaseMode: 'all',
      stage: 'deploy',
      status: 'deployed',
    });
    mountPage();

    // 完成信号：顶部部署入口已渲染（说明 latest 已回填）
    await screen.findByRole('button', { name: '部署' });
    expect(mocks.latest).toHaveBeenCalled();
    expect(mocks.progress).not.toHaveBeenCalled();
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();
  });

  it('当所选环境尚未部署（latest 仅剩历史 canary 记录）时，不应请求进度也不展示横幅', async () => {
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      hasDeployRecord: false,
      inProgressReleaseMode: '',
      releaseMode: 'canary',
      stage: 'deploy',
      status: 'uninstalled',
    });
    mountPage();

    // 完成信号：未部署空态渲染（说明 latest 已回填且页面进入未部署分支）
    await screen.findByText('该环境尚未部署应用');
    expect(mocks.latest).toHaveBeenCalled();
    // 移除部署后不能继续搭车 latest-status 每 5s 轮询进度接口
    expect(mocks.progress).not.toHaveBeenCalled();
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();
  });

  it('当操作失败时，应提示失败并保留横幅当前状态', async () => {
    mocks.operate.mockRejectedValueOnce(new Error('rollout changed'));
    mountPage();
    const banner = await findBanner();

    await userEvent.click(within(banner).getByRole('button', { name: '暂停' }));

    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith({ theme: 'error', message: '操作失败，请稍后重试' }),
    );
    await waitFor(() => expect(mocks.progress).toHaveBeenCalledTimes(2));
    expect(screen.getByTestId('batch-release-banner')).toHaveAttribute('data-kind', 'updating');
  });

  it('当操作请求未返回时，横幅操作应置灰且不重复发起请求', async () => {
    let release: (value?: unknown) => void = () => {};
    mocks.operate.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          release = resolve;
        }),
    );
    mountPage();
    const banner = await findBanner();

    await userEvent.click(within(banner).getByRole('button', { name: '暂停' }));
    await waitFor(() => expect(mocks.operate).toHaveBeenCalledTimes(1));

    // 请求在途：所有入口统一置灰（而不是「看着能点、点了没反应」）
    await waitFor(() => expect(within(banner).getByRole('button', { name: '更多' })).toBeDisabled());
    expect(within(banner).getByRole('button', { name: '回滚' })).toBeDisabled();
    await userEvent.click(within(banner).getByText('回退上一批'));
    expect(mocks.operate).toHaveBeenCalledTimes(1);

    // 请求返回后立即恢复可点，不被实例列表的 Watch 刷新拖住
    release({});
    await waitFor(() => expect(within(banner).getByRole('button', { name: '更多' })).toBeEnabled());
    expect(within(banner).getByRole('button', { name: '回滚' })).toBeEnabled();
  });
});

describe('实例列表分批发布控制：灰度发布进度视图', () => {
  /** 点击横幅「查看发布进度」并等待进度面板出现，返回面板节点。 */
  async function openProgressPanel() {
    const banner = await findBanner();
    await userEvent.click(within(banner).getByRole('button', { name: '查看发布进度' }));
    return await screen.findByTestId('batch-release-progress-panel');
  }

  it('当点击「查看发布进度」时，应隐藏横幅并展示进度视图，点返回后复原横幅态', async () => {
    mountPage();
    const panel = await openProgressPanel();

    // 横幅与进度视图互斥渲染
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();
    const progressBanner = within(panel).getByTestId('batch-release-progress-banner');
    expect(within(panel).getByText('灰度发布中')).toBeInTheDocument();
    // 标题 Tag：更新中为「正在更新实例」
    expect(within(panel).getByText('正在更新实例')).toBeInTheDocument();
    // 元信息行：策略快照缺失时只展示有数据的字段
    expect(within(panel).getByText('共 3 批')).toBeInTheDocument();
    expect(within(panel).getByText('当前第 3 批')).toBeInTheDocument();
    expect(within(panel).getByText('已更新 8/20 个实例')).toBeInTheDocument();
    expect(within(panel).getByText('v1.8.2 → v1.8.3')).toBeInTheDocument();
    // 步骤条渲染批次节点（含状态 Tag）
    expect(within(progressBanner).getAllByText('已完成').length).toBeGreaterThanOrEqual(2);
    expect(within(progressBanner).getByText('正在更新')).toBeInTheDocument();

    // 返回实例列表：进度视图隐藏，横幅恢复
    await userEvent.click(within(panel).getByRole('button', { name: '返回实例列表' }));
    await waitFor(() => expect(screen.queryByTestId('batch-release-progress-panel')).toBeNull());
    expect(await findBanner()).toBeInTheDocument();
  });

  it('当进度视图展开时，操作按钮应全部平铺（无「更多」）并复用同一操作链路', async () => {
    mountPage();
    const panel = await openProgressPanel();
    const progressBanner = within(panel).getByTestId('batch-release-progress-banner');

    // 顺序：主操作 → 回退上一批 → 跳过剩余批次 → 回滚，不再收纳进「更多」
    const labels = within(progressBanner)
      .queryAllByRole('button')
      .map(button => button.textContent?.trim() ?? '');
    expect(labels).toEqual(['暂停', '回退上一批', '跳过剩余批次', '回滚']);
    expect(within(panel).queryByRole('button', { name: '更多' })).toBeNull();

    // 平铺按钮复用横幅的二次确认链路
    await userEvent.click(within(progressBanner).getByRole('button', { name: '回退上一批' }));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(mocks.infoBox.mock.calls[0][0].title).toBe('确认回退到上一批？');
    expect(mocks.operate).not.toHaveBeenCalled();
    mocks.infoBox.mock.calls[0][0].onConfirm();
    await waitFor(() => expectOperateCalledWith('backward'));
  });

  it('当暂停态展开进度视图时，应展示暂停小标且主操作为继续发布', async () => {
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'manual-paused' }));
    mountPage();
    const panel = await openProgressPanel();
    const progressBanner = within(panel).getByTestId('batch-release-progress-banner');

    expect(within(panel).getByText('灰度发布中')).toBeInTheDocument();
    // 标题 Tag 与步骤 Tag 同文案，断言至少各出现一处
    expect(within(panel).getAllByText('等待手动继续').length).toBeGreaterThanOrEqual(2);
    const labels = within(progressBanner)
      .queryAllByRole('button')
      .map(button => button.textContent?.trim() ?? '');
    expect(labels).toEqual(['继续发布', '回退上一批', '跳过剩余批次', '回滚']);

    await userEvent.click(within(progressBanner).getByRole('button', { name: '继续发布' }));
    await waitFor(() => expectOperateCalledWith('continue'));
  });
});
