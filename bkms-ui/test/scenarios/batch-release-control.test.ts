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
 * V = 11
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
 *           ⑫ 点击横幅「查看发布详情」隐藏横幅并切换为灰度发布进度视图（独立组件，互斥渲染；
 *              标题行策略段 + ⓘ 浮层收纳低频核对字段，元信息行仅失败态保留失败原因，步骤条标题带换算累计实例数）
 *           ⑬ 进度视图内操作按钮全部平铺（无「更多」），复用横幅同一操作链路
 *           ⑭ 进度视图点「返回实例列表」复原回横幅态
 *           ⑮ 骨架占位仅在「loading 且无进度快照」时展示：环境切换收旧横幅进骨架态；
 *              关闭后的完成态横幅在轮询补查期间不得闪出骨架
 *           ⑯ 终态关闭记忆按「应用+环境+部署记录」多条累加：多环境各自关闭互不覆盖，
 *              且只落 sessionStorage（重开浏览器后横幅恢复、不长期隐藏）
 * 路径：① 更新中→横幅 + 部署禁用 ② 暂停 ③ 回滚确认/取消 ④ 暂停态回退 + 跳过
 *       ⑤ 定时暂停继续发布 ⑥ 批次序号未回填仍展示按钮 ⑦ 已完成→成功 Alert + 关闭记忆
 *       ⑧ 失败→回退上一批 ⑨ 已回滚→提示 + 部署可用 ⑩ 非分批发布不请求 ⑪ 操作失败保留状态
 *       ⑫ 请求挂起→按钮置灰且不重发→返回后恢复 ⑬ 未部署环境不请求进度
 *       ⑭ 查看发布详情→横幅隐藏 + 面板呈现 ⑮ 面板平铺按钮复用确认链路 ⑯ 返回实例列表
 *       ⑰ 切换环境→旧横幅收起 + 骨架占位→新环境进度返回后展示新横幅 ⑱ 关闭后轮询补查→骨架不出现
 *       ⑲ 两个环境分别关闭→切回已关闭环境仍隐藏 + 记忆落 sessionStorage（localStorage 无残留）
 * 变异验证：M1 环境切换不重置快照（deploy.vue watch 注销）→ ⑰ 红；M2 showSkeleton 退化为只看
 *           loading → ⑱ 红；M3 resetForSwitch 保留旧进度快照 → ⑰ 红；M4 关闭记忆回到「单值覆盖 +
 *           localStorage」旧实现 → ⑲ 红；逐轮还原后 21/21 全绿。
 * stub/mock：DeployService（进度/操作）、Message/InfoBox 命令式 API、Dropdown 只渲染插槽、
 *            InstanceList / 部署总览 / 部署历史等非验证目标子组件、EnvSelectPanel 仅 emit envList、
 *            CanaryInstanceTable 仅验证面板接线（灰度筛选口径由 resolveInstanceCanaryState 纯函数单测覆盖）。
 */
import { h } from 'vue';
import type { SetupContext } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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

/** 递归收集节点纯文本（数字/版本号等插值内容），供 i18n-t stub 拼接。 */
function collectText(node: unknown): string {
  if (typeof node === 'string') return node;
  if (typeof node === 'number') return String(node);
  const children = (node as null | { children?: unknown })?.children;
  if (typeof children === 'string') return children;
  if (Array.isArray(children)) return children.map(collectText).join('');
  return '';
}

/**
 * i18n-t stub（本文件专用）：把 keypath 的 {0}/{1} 占位符替换为默认插槽子节点的纯文本，
 * 使带插值的文案可被 getByText 直接命中（共享 I18nTStub 只直译 keypath）。
 * 命名插槽（如横幅的 #index/#version）非断言目标，忽略。
 *
 * 用普通对象而非 defineComponent 声明：测试文件里已有多个 mock 组件，
 * 再调用 defineComponent 会触发 vue/one-component-per-file，普通对象同样支持 setup 返回渲染函数。
 */
const I18nTWithSlots = {
  name: 'batchReleaseI18nT',
  props: {
    keypath: { type: String, default: '' },
    tag: { type: String, default: 'span' },
  },
  setup(props: { keypath: string; tag: string }, { slots }: SetupContext) {
    return () => {
      const children = slots.default?.() ?? [];
      let text = props.keypath;
      for (let i = children.length - 1; i >= 0; i -= 1) {
        text = text.replace(new RegExp(`\\{${i}\\}`), collectText(children[i]));
      }
      return h(props.tag || 'span', text);
    };
  },
};

// 接口契约里 currentBatchIndex 在批次序号未回填时为 null（d.ts 仅标注 number），测试按真实取值放宽
type ProgressOverrides = Partial<Omit<BatchReleaseProgressOutputObj, 'currentBatchIndex'>> & {
  currentBatchIndex?: null | number;
};

function buildProgress(overrides: ProgressOverrides = {}): BatchReleaseProgressOutputObj {
  const { currentBatchIndex, ...rest } = overrides;
  const progress: BatchReleaseProgressOutputObj = {
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
    strategyName: '分四批灰度',
    updatedReplicas: 8,
    ...rest,
  };
  // 未显式传入时保留上面的默认批次序号；null（未回填）归一到 undefined，与 d.ts 的 number 契约对齐，
  // 组件本身也只用 typeof number / === -1 判定，两者行为一致。
  if (currentBatchIndex !== undefined) progress.currentBatchIndex = currentBatchIndex ?? undefined;
  return progress;
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
      components: { 'i18n-t': I18nTWithSlots },
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
  window.sessionStorage.clear();
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
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '暂停', '更多', '回滚']);
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
    // 设计稿：等待手动继续为蓝色过程态，状态词展示「第 N 批已完成」
    expect(within(banner).getByText('第 3 批已完成')).toBeInTheDocument();
    expect(within(banner).getByRole('button', { name: '继续发布' })).toBeEnabled();
    // 暂停态没有暂停入口；更多位于「继续发布」与「回滚」之间
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '继续发布', '更多', '回滚']);
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
    // 到期后 statusTag 退化为「已暂停」，横幅按黄色警示态展示「第 N 批已暂停」
    expect(within(banner).getByText('第 3 批已暂停')).toBeInTheDocument();
    expect(within(banner).getByRole('button', { name: '继续发布' })).toBeEnabled();
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '继续发布', '更多', '回滚']);

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
    // 设计稿：定时暂停为蓝色过程态，状态词「第 N 批已完成」+ 加粗倒计时
    // （i18n-t 插槽插值在测试 mock 下不渲染，倒计时值由纯函数单测的 statusTag.params 断言覆盖）
    expect(within(banner).getByText('第 3 批已完成')).toBeInTheDocument();
    expect(within(banner).queryByText('第 3 批已暂停')).toBeNull();
    // 顺序：主操作（继续发布）→ 暂停 → 更多 → 回滚
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '继续发布', '暂停', '更多', '回滚']);

    await userEvent.click(within(banner).getByRole('button', { name: '暂停' }));
    await waitFor(() => expectOperateCalledWith('pause'));
    await waitFor(() => expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '已暂停分批发布' }));
  });

  it('当批次序号尚未回填（batchPhase=updating、currentBatchIndex=null）时，仍应展示暂停与回滚', async () => {
    mocks.progress.mockResolvedValue(
      buildProgress({
        batchPhase: 'updating',
        // 接口在批次序号未回填时返回 null，d.ts 未覆盖该取值（由 ProgressOverrides 放宽）
        currentBatchIndex: null,
        message: 'All steps are completed, updating primary workload',
      }),
    );
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'updating');
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '暂停', '更多', '回滚']);
    expect(within(banner).queryByRole('button', { name: '继续发布' })).toBeNull();
    // 拿不到批次序号时无法确认存在上一批：回退上一批置灰展示，点击不发起请求
    expect(within(banner).getByText('回退上一批')).toHaveClass('!pointer-events-none');
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();
  });

  it('当本批尚未开始（pending）时，应展示暂停与回滚，更多里没有回退上一批', async () => {
    // 同上：本批尚未开始时 currentBatchIndex 仍未回填（null）
    mocks.progress.mockResolvedValue(buildProgress({ batchPhase: 'pending', currentBatchIndex: null }));
    mountPage();
    const banner = await findBanner();

    expect(banner).toHaveAttribute('data-kind', 'updating');
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '暂停', '更多', '回滚']);
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
    expect(buttonLabelsOf(banner).filter(Boolean)).toEqual(['查看发布详情']);
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

  it('当在多个环境分别关闭完成态横幅后，切回已关闭的环境仍保持关闭，且记忆只存在 sessionStorage', async () => {
    const envB = { ...mocks.env, name: 'prod-b' };
    // 每个环境各自一条完成态的分批发布记录
    mocks.progress.mockImplementation((params: { envName: string }) =>
      Promise.resolve(
        buildProgress({
          batchPhase: 'completed',
          deployID: params.envName === 'prod-b' ? 'deploy-b' : 'deploy-1',
          status: 'deployed',
        }),
      ),
    );
    // 完成后 inProgressReleaseMode 已清空，仅靠 releaseMode=canary 补查一次以拿到完成态结果。
    mocks.latest.mockResolvedValue({
      deployID: 'deploy-1',
      inProgressReleaseMode: '',
      releaseMode: 'canary',
      stage: 'deploy',
      status: 'deployed',
    });
    const progressCallsFor = (envName: string) =>
      mocks.progress.mock.calls.filter(([params]) => (params as { envName: string }).envName === envName).length;

    mountPage();

    // 环境 A 关闭完成态横幅
    const bannerA = await findBanner();
    expect(bannerA).toHaveAttribute('data-kind', 'completed');
    await userEvent.click(bannerA.querySelector('.bk-alert-close') as HTMLElement);
    await waitFor(() => expect(screen.queryByTestId('batch-release-banner')).toBeNull());

    // 切到环境 B：新横幅正常展示，关闭它
    useTrpcDeployStore().updateCurEnvItem(envB as never);
    const bannerB = await findBanner();
    expect(bannerB).toHaveAttribute('data-kind', 'completed');
    await userEvent.click(bannerB.querySelector('.bk-alert-close') as HTMLElement);
    await waitFor(() => expect(screen.queryByTestId('batch-release-banner')).toBeNull());

    // 切回环境 A：记忆必须按环境各自保留，不能因为环境 B 的关闭被冲掉
    useTrpcDeployStore().updateCurEnvItem(mocks.env as never);
    await waitFor(() => expect(progressCallsFor('prod')).toBeGreaterThanOrEqual(2));
    await waitFor(() => expect(screen.queryByTestId('batch-release-skeleton')).toBeNull());
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();

    // 关闭记忆落在 sessionStorage（重开浏览器即清空、横幅重新出现），不再写 localStorage 长期隐藏
    const sessionValue = window.sessionStorage.getItem('batch-release-dismissed') ?? '';
    expect(sessionValue).toContain('app-a:prod:deploy-1');
    expect(sessionValue).toContain('app-a:prod-b:deploy-b');
    expect(window.localStorage.getItem('batch-release-dismissed')).toBeNull();
    expect(window.localStorage.getItem('global_batch-release-dismissed')).toBeNull();
  });

  it('当切换环境重新加载横幅时，应先展示骨架占位而不是直接替换旧横幅', async () => {
    const envB = { ...mocks.env, name: 'prod-b' };
    // 第二次进度请求（新环境）挂起，保证骨架窗口可观测
    let releaseB: (value?: BatchReleaseProgressOutputObj) => void = () => {};
    mocks.progress.mockResolvedValueOnce(buildProgress()).mockImplementationOnce(
      () =>
        new Promise(resolve => {
          releaseB = resolve;
        }),
    );
    mountPage();
    const bannerA = await findBanner();
    expect(bannerA).toHaveAttribute('data-kind', 'updating');

    // 切换环境：旧环境的横幅立刻收起，进入骨架等待态
    useTrpcDeployStore().updateCurEnvItem(envB as never);
    await waitFor(() => expect(screen.queryByTestId('batch-release-banner')).toBeNull());
    expect(await screen.findByTestId('batch-release-skeleton')).toBeInTheDocument();

    // 新环境进度返回：骨架收起，展示新环境的横幅（manual-paused）
    releaseB(buildProgress({ currentBatchIndex: 1, batchPhase: 'manual-paused' }));
    const bannerB = await findBanner();
    expect(bannerB).toHaveAttribute('data-kind', 'paused');
    expect(screen.queryByTestId('batch-release-skeleton')).toBeNull();
  });

  it('完成态横幅关闭后，即使轮询仍在补查进度也不应闪出骨架屏', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      mocks.progress
        .mockResolvedValueOnce(buildProgress({ batchPhase: 'completed', status: 'deployed' }))
        // 完成后 inProgressReleaseMode 仍为 canary 时每轮轮询都会补查；让补查挂起，
        // 若骨架条件只看 loading，骨架会一直挂在页面上（回归信号稳定可断言）。
        .mockImplementationOnce(() => new Promise(() => {}));
      mocks.latest.mockResolvedValue({
        deployID: 'deploy-1',
        inProgressReleaseMode: 'canary',
        releaseMode: 'canary',
        stage: 'deploy',
        status: 'deployed',
      });
      mountPage();
      const banner = await findBanner();
      expect(banner).toHaveAttribute('data-kind', 'completed');

      await userEvent.click(banner.querySelector('.bk-alert-close') as HTMLElement);
      await waitFor(() => expect(screen.queryByTestId('batch-release-banner')).toBeNull());

      // 推进一个轮询周期：补查已发起（完成信号），但骨架不得出现，横幅保持关闭
      await vi.advanceTimersByTimeAsync(5000);
      await waitFor(() => expect(mocks.progress).toHaveBeenCalledTimes(2));
      expect(screen.queryByTestId('batch-release-skeleton')).toBeNull();
      expect(screen.queryByTestId('batch-release-banner')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it('当分批发布失败时，应展示失败原因并提供重试与回滚入口', async () => {
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
    // 设计稿：失败态主入口「重试」+ 回滚平铺，回退上一批收进「更多」
    expect(within(banner).getByRole('button', { name: '重试' })).toBeEnabled();
    expect(buttonLabelsOf(banner)).toEqual(['查看发布详情', '重试', '更多', '回滚']);
    // 失败后仍需先收尾，顶部部署入口保持禁用
    expect(screen.getByRole('button', { name: '部署' })).toBeDisabled();

    await userEvent.click(within(banner).getByRole('button', { name: '重试' }));
    await waitFor(() => expectOperateCalledWith('continue'));
    await waitFor(() =>
      expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '已重试，开始执行第 3 批' }),
    );

    await userEvent.click(within(banner).getByText('回退上一批'));
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(mocks.infoBox.mock.calls[0][0].title).toBe('确认回退到上一批？');
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
    expect(buttonLabelsOf(banner).filter(Boolean)).toEqual(['查看发布详情']);
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
  /** 点击横幅「查看发布详情」并等待进度面板出现，返回面板节点。 */
  async function openProgressPanel() {
    const banner = await findBanner();
    await userEvent.click(within(banner).getByRole('button', { name: '查看发布详情' }));
    return await screen.findByTestId('batch-release-progress-panel');
  }

  it('当点击「查看发布详情」时，应隐藏横幅并展示精简进度视图，点返回后复原横幅态', async () => {
    mountPage();
    const panel = await openProgressPanel();

    // 横幅与进度视图互斥渲染
    expect(screen.queryByTestId('batch-release-banner')).toBeNull();
    const progressBanner = within(panel).getByTestId('batch-release-progress-banner');
    expect(within(panel).getByText('灰度发布中')).toBeInTheDocument();
    // 标题 Tag：更新中为「正在更新实例」
    expect(within(panel).getByText('正在更新实例')).toBeInTheDocument();
    // 标题行策略段：状态 Tag 右侧展示「发布策略：<策略名>」（ⓘ 浮层明细由纯函数与交互单测覆盖）
    expect(within(panel).getByTestId('batch-release-progress-strategy')).toHaveTextContent('发布策略：分四批灰度');
    // 步骤条渲染批次节点（含状态 Tag），标题直接带换算后的累计实例数
    expect(within(progressBanner).getAllByText('已完成').length).toBeGreaterThanOrEqual(2);
    expect(within(progressBanner).getByText('正在更新')).toBeInTheDocument();
    // 标题参数（批次号 / 实例数）加粗为独立节点，按整块用户可见文本断言
    expect(progressBanner.textContent).toContain('第 1 批 · 1 个实例');
    expect(progressBanner.textContent).toContain('第 2 批 · 50%（10 个）');
    expect(progressBanner.textContent).toContain('第 3 批 · 20 个实例');
    // 元信息行已精简：共 X 批 / 当前第 X 批 / 已更新 X/Y / 镜像不再展示（步骤条与实例表筛选胶囊承载）
    expect(within(panel).queryByText('共 3 批')).toBeNull();
    expect(within(panel).queryByText('当前第 3 批')).toBeNull();
    expect(within(panel).queryByText('已更新 8/20 个实例')).toBeNull();
    expect(within(panel).queryByText('v1.8.2 → v1.8.3')).toBeNull();

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
