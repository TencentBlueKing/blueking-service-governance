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
 * We undertake not to change the open source license (MIT license) that applies to the current version
 * of the project delivered to anyone in the future.
 */
/**
 * 场景级测试：部署表单发布策略选择（ROUTES 未收录，本头即场景说明）
 *
 * 覆盖二十一条用户可感知路径 → V = 21：
 *   1. 默认「单批发布」提交：部署参数不带 releaseStrategyID
 *   2. 切「分批发布」：展开策略下拉，下拉底部有「新建策略」、右侧有「查看」
 *   3. 分批发布未选策略提交：表单校验拦截，不发起部署
 *   4. 选中策略：发布预览出现（策略说明 + 摘要四项 + 批次时间线 + 进度方块 + 暂停标签）
 *   5. 分批发布已选策略提交：部署参数携带 releaseStrategyID
 *   6. 切回「单批发布」：预览收起、策略清空，提交不带 releaseStrategyID
 *   7. 联邦集群环境：分批发布按钮禁用
 *   8. 从源码构建：分批发布按钮禁用，且 canary 自动回落为单批发布
 *   9. Helm 应用：分批发布按钮禁用（仅应用模型应用支持）
 *   10. 主工作负载为 Deployment：分批发布按钮禁用（仅 GameDeployment 支持）
 *   11. 主工作负载为 GameDeployment：分批发布可用
 *   12. 点下拉内「新建策略」：叠加新建策略抽屉；保存后抽屉关闭、自动选中新策略并刷新发布预览
 *   13. 点「查看」：打开已选策略查看抽屉；未选策略时置灰不可点；部署页查看抽屉为纯查看，页脚只保留「关闭」
 *   14. 策略末批未到全量：提示收尾为「最后自动全量」，预览不补齐全量批、底部提示剩余实例自动更新
 *   15. 两批策略（25% → 100%）：提示保留首批灰度阶段，不误判为单批全量
 *   16. 主工作负载资格检查完成前：分批发布禁用；返回 Deployment 后仍禁用
 *   17. 首次部署（无记录 / 只有构建记录 / 已下线）：分批发布禁用，仍可单批发布
 *   18. 切「从源码构建」：默认分支回填到修订选择器（验证模板 ref 真实挂载，prepare 不被静默跳过）
 *   19. 发布方式展示单批/分批说明：默认单批选中，切换时两项始终互斥
 *   20. 键盘选择：方向键切换焦点与选中状态，Tab 和空格可操作单选项
 *   21. 分批不可用：方向键及 Tab 跳过禁用项，保留单批选中状态
 *
 * 变异验证：固定选择为单批、移除分批禁用、移除方向键选中更新均被捕获（各 1 条失败，bail=1），已精确还原。
 *
 * stub/mock：
 *   - bkui-vue 真实渲染（Form/Button/Radio/Select/Input/Tag）；嵌套的新建策略抽屉同样真实渲染
 *   - ReleaseStrategiesService.listReleaseStrategies → harness 假数据（4 批：1 个 → 25% 暂停 600s → 50% 永久暂停 → 100%）
 *   - ReleaseStrategiesService.createReleaseStrategy → harness 返回新建结果（含 id，供「自动选中」断言）
 *   - use-deploy 的 useDeployAPIs → harness.createDeployDirectly / buildAndCreateDeploy / listDeployRecords（断言部署参数与主工作负载）/ listLatestDeployRecords（首次部署判定）
 *   - use-app-repo-ref-select 保持真实实现（仅 stores 打桩）：ref 挂载契约本身是 ⑱ 的验证目标，不能整体 stub
 *   - use-deploy-precheck / use-recommend-tag / use-app-visible-envs / use-leave-confirm / stores → 契约 stub
 *   - vue-router → mock-router 工厂（业务已不直接跳页，保留路由依赖兜底）
 *   - ImageSelect → 可填值 stub（真实组件依赖镜像列表接口）；CodeRevisionField 按模块 mock 为记录型 stub（暴露 RepoRefHandle，
 *     它由 unplugin-vue-components 编译期自动引入，global.components 覆盖不到）
 *
 * backlog（不确定项，待业务确认）：
 *   - 「构建部署/联邦集群暂不支持分批发布」tooltip 文案经 bkui tooltips 渲染 jsdom 不可达，仅断言 disabled 属性
 *   - 「请选择发布策略」校验错误文案经 bkui tooltip 渲染，以 .bk-form-item.is-error 态断言（组件库内部类名，jsdom 例外）
 *   - 新建策略抽屉叠加层级由 bkui Modal 的全局递增 zIndex 保证，jsdom 无视觉信号，仅断言两层抽屉同时存在于 DOM
 */
import { defineComponent, h, ref } from 'vue';
import type { SetupContext } from 'vue';

import userEvent from '@testing-library/user-event';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/vue';
import { bkTooltips } from 'bkui-vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import QuicklyDeployForm from '~/pages/application/detail/deploy/quickly-deploy-form.vue';

import type { EnvOutput } from '~/@types/v1/env';

/* 场景接线需要多个内联桩组件（ImageSelectStub / CodeRevisionFieldStub / Harness），与 instance-weight.test.ts 同款豁免。 */
/* eslint-disable vue/one-component-per-file */

const harness = vi.hoisted(() => ({
  checkDeployEnv: vi.fn().mockReturnValue(true),
  createDeployDirectly: vi.fn().mockResolvedValue({}),
  buildAndCreateDeploy: vi.fn().mockResolvedValue({}),
  // 最近一条部署记录：默认无记录（主工作负载未知）
  listDeployRecords: vi.fn().mockResolvedValue({ count: 0, results: [] }),
  // 最新部署状态：默认已有部署记录（非首次部署）
  listLatestDeployRecords: vi.fn().mockResolvedValue({ hasDeployRecord: true, status: 'deployed' }),
  listReleaseStrategies: vi.fn(),
  createReleaseStrategy: vi.fn(),
}));

/** vue-router mock 工厂的 push 桩（「查看」已改为打开查看抽屉，不再断言跳页，保留兜底） */
const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));

/** 可变应用类型：helm / taf / trpc，供应用类型禁用用例切换 */
const appState = vi.hoisted(() => ({ appType: 'trpc' }));

const strategyList = [
  {
    id: 'strategy-1',
    name: '生产谨慎发布',
    description: '谨慎放量',
    isBuiltin: false,
    workspaceID: 'ws-1',
    batches: [
      { target: '1', pauseMode: 'none' },
      { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
      { target: '50%', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
    ],
  },
  {
    id: 'strategy-two-batch',
    name: '两批放量',
    description: '25% 后全量',
    isBuiltin: false,
    workspaceID: 'ws-1',
    batches: [
      { target: '25%', pauseMode: 'none' },
      { target: '100%', pauseMode: 'none' },
    ],
  },
  {
    id: 'strategy-partial',
    name: '末批部分放量',
    description: '末批未到全量，由系统补齐',
    isBuiltin: false,
    workspaceID: 'ws-1',
    batches: [
      { target: '1', pauseMode: 'none' },
      { target: '25%', pauseMode: 'none' },
      { target: '50%', pauseMode: 'none' },
    ],
  },
  {
    id: 'strategy-builtin',
    name: '分三批灰度',
    description: '平台内置策略',
    isBuiltin: true,
    workspaceID: 'ws-1',
    batches: [
      { target: '1', pauseMode: 'none' },
      { target: '50%', pauseMode: 'none' },
      { target: '100%', pauseMode: 'none' },
    ],
  },
];

vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  // 直译并处理 {0} 形式的插值（如「第 {0} 批」）
  useI18n: () => ({
    t: (s: string, params?: unknown[]) => {
      let out = s;
      if (Array.isArray(params)) {
        params.forEach((p, i) => {
          out = out.replace(`{${i}}`, String(p));
        });
      }
      return out;
    },
    te: () => true,
  }),
}));

vi.mock('~/api/modules/v1/release-strategies', () => ({
  ReleaseStrategiesService: {
    listReleaseStrategies: harness.listReleaseStrategies,
    createReleaseStrategy: harness.createReleaseStrategy,
  },
}));

vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  return createRouterMock({
    route: { path: '/ws-1/app/demo/detail', name: 'detail', params: { space: 'ws-1' } },
    push: routerMocks.push,
  });
});

vi.mock('~/pages/application/detail/deploy/use-deploy', async importOriginal => ({
  ...(await importOriginal<object>()),
  useDeployAPIs: () => ({
    createDeployDirectly: harness.createDeployDirectly,
    buildAndCreateDeploy: harness.buildAndCreateDeploy,
    listDeployRecords: harness.listDeployRecords,
    listLatestDeployRecords: harness.listLatestDeployRecords,
  }),
}));

vi.mock('~/pages/application/detail/deploy/use-deploy-precheck', async () => {
  const { ref: createRef } = await import('vue');
  return {
    useDeployPrecheck: () => ({
      cancelDeploy: vi.fn(),
      continueDeploy: vi.fn(),
      federationMismatches: createRef([]),
      isShowPrecheckDialog: createRef(false),
      missingRequiredClusterAddons: createRef([]),
      precheck: vi.fn().mockResolvedValue(true),
      precheckEnvId: createRef(''),
      precheckEnvName: createRef(''),
      undefinedVars: createRef([]),
    }),
  };
});

vi.mock('~/composables/use-app-visible-envs', () => ({
  useAppVisibleEnvs: () => ({ checkDeployEnv: harness.checkDeployEnv }),
}));

vi.mock('~/composables/use-leave-confirm', () => ({
  default: () => ({
    confirmBox: vi.fn().mockResolvedValue(true),
    forceCleanDirtyTag: vi.fn(),
    withPausedWatch: (fn: () => void) => fn(),
  }),
}));

vi.mock('~/composables/use-recommend-tag', () => ({
  useRecommendTag: () => ({
    getDefaultBranch: () => 'master',
    fetchRecommendTag: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({ appID: 'app-a', appType: appState.appType, appDetail: {} }),
}));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1' }),
}));
vi.mock('~/stores/trpc-deploy', () => ({
  useTrpcDeployStore: () => ({ curEnvItem: null }),
}));

vi.mock('~/pages/application/components/image-select.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      name: 'ImageSelectStub',
      props: { value: { type: String, default: '' } },
      emits: ['update:value'],
      template:
        '<input data-testid="image-select-stub" :value="value" @input="$emit(\'update:value\', $event.target.value)" />',
    }),
  };
});

/**
 * 修订选择器桩：暴露与真实 CodeRevisionField 一致的操作（RepoRefHandle），
 * 用于验证 use-app-repo-ref-select 的 ref 是否真的挂到了组件实例上
 * （模板 ref 缺少同名顶层绑定时不会写回，prepare 会静默空转）。
 *
 * 该组件由 unplugin-vue-components 在编译期自动引入，global.components 覆盖不到，只能按模块 mock。
 */
const codeRevisionCalls = vi.hoisted(() => ({
  hidePopover: vi.fn(),
  prepare: vi.fn((preferred?: string) => preferred ?? ''),
  reset: vi.fn(),
}));

vi.mock('~/components/code-revision/code-revision-field.vue', async () => {
  const { defineComponent: createComponent, h: createElement } = await import('vue');
  return {
    default: createComponent({
      name: 'CodeRevisionFieldStub',
      setup(_, { expose }) {
        expose({
          dismissIfOutside: () => undefined,
          hidePopover: () => codeRevisionCalls.hidePopover(),
          prepare: (preferred?: string) => codeRevisionCalls.prepare(preferred),
          reset: () => codeRevisionCalls.reset(),
        });
        return () => createElement('div', { 'data-testid': 'code-revision-field-stub' });
      },
    }),
  };
});

/** 递归收集插槽子节点纯文本（批次实例数值），供 i18n-t stub 拼接插值。 */
function collectText(node: unknown): string {
  if (typeof node === 'string') return node;
  if (typeof node === 'number') return String(node);
  const children = (node as null | { children?: unknown })?.children;
  if (typeof children === 'string') return children;
  if (Array.isArray(children)) return children.map(collectText).join('');
  return '';
}

/**
 * 发布预览的实例分数文案走 `<i18n-t keypath="新版本 {0} · 旧版本 {1}">`，
 * 共享 I18nTStub 只直译 keypath、不处理插值，这里把默认插槽子节点文本按序替换占位符。
 */
const I18nTWithSlots = {
  name: 'releaseStrategyI18nT',
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

/** 复刻宿主接线：点击「提交部署」经 defineExpose.submit 走完整校验 + 部署链路 */
const Harness = defineComponent({
  components: { QuicklyDeployForm },
  props: { isFederationEnv: { type: Boolean, default: false } },
  setup() {
    const formRef = ref<{ submit: (env: EnvOutput) => Promise<boolean> }>();
    const targetEnv: EnvOutput = { name: 'stag', displayName: 'stag' };
    const handleSubmit = () => formRef.value?.submit(targetEnv);
    return { formRef, handleSubmit };
  },
  // 文案区别于新建策略抽屉的「提交」，避免两者按 role 名称查询时撞车
  template: `
    <div>
      <QuicklyDeployForm
        ref="formRef"
        env-name="stag"
        env-type="staging"
        :is-federation-env="isFederationEnv"
        :precheck-before-submit="false"
        use-provided-env
      />
      <button data-testid="submit-trigger" type="button" @click="handleSubmit">提交部署</button>
    </div>
  `,
});

/** 打开策略下拉并选择指定策略（bkui Select 真实渲染；选项 label 含批次摘要，用正则匹配） */
async function chooseStrategy(namePattern: RegExp) {
  const formItem = screen.getByText('发布策略').closest('.bk-form-item') as HTMLElement;
  await userEvent.click(formItem.querySelector('.bk-select') as HTMLElement);
  await userEvent.click(await screen.findByText(namePattern));
}

/** 等主工作负载资格检查完成（默认无部署记录，不因负载禁用）后切到分批发布 */
async function enableCanaryMode() {
  const button = screen.getByRole('radio', { name: /^分批发布/ });
  await waitFor(() => expect(button).not.toBeDisabled());
  await userEvent.click(button);
}

/** 填写实例数与镜像 Tag，保证除发布策略外的必填项通过校验 */
async function fillBaseForm() {
  // 直接更新输入值，避免逐字输入造成「120」或 clear 触发校验提示
  fireEvent.update(screen.getByRole('spinbutton'), '20');
  await userEvent.type(screen.getByTestId('image-select-stub'), 'v1.0');
}

/** 打开策略下拉后点击底部的「新建策略」，等新建策略抽屉打开 */
async function openCreateStrategyDrawer() {
  const formItem = screen.getByText('发布策略').closest('.bk-form-item') as HTMLElement;
  await userEvent.click(formItem.querySelector('.bk-select') as HTMLElement);
  await userEvent.click(await screen.findByText('新建策略'));
  return await screen.findByTestId('strategy-sideslider-header');
}

async function renderForm(isFederationEnv = false) {
  return render(Harness, {
    props: { isFederationEnv },
    global: {
      components: { 'i18n-t': I18nTWithSlots },
      // 与 src/modules/global.ts 对齐：注册 v-bk-tooltips 指令
      directives: { 'bk-tooltips': bkTooltips },
      mocks: { $t: translate },
    },
  });
}

async function submitForm() {
  await userEvent.click(screen.getByTestId('submit-trigger'));
}

/** 与 useI18n mock 保持一致：直译并处理 {0} 插值 */
function translate(key: string, params?: unknown[]) {
  let out = key;
  if (Array.isArray(params)) {
    params.forEach((p, i) => {
      out = out.replace(`{${i}}`, String(p));
    });
  }
  return out;
}

beforeEach(() => {
  vi.clearAllMocks();
  appState.appType = 'trpc';
  harness.listReleaseStrategies.mockResolvedValue(strategyList);
  harness.listDeployRecords.mockResolvedValue({ count: 0, results: [] });
  harness.listLatestDeployRecords.mockResolvedValue({ hasDeployRecord: true, status: 'deployed' });
});

describe('部署表单：发布策略选择', () => {
  it('当用户选择发布方式时，应展示单批与分批说明并保持单选状态', async () => {
    await renderForm();
    const releaseModes = within(screen.getByRole('radiogroup', { name: '发布方式' }));
    const singleBatch = releaseModes.getByRole('radio', { name: /^单批发布/ });
    const batchRelease = releaseModes.getByRole('radio', { name: /^分批发布/ });

    expect(singleBatch).toBeChecked();
    expect(batchRelease).not.toBeChecked();
    expect(releaseModes.getByText('中途不暂停，适合日常发布')).toBeVisible();
    expect(
      releaseModes.getByText('按发布策略分批更新，每批完成后可暂停确认，适合需要先小范围验证的发布'),
    ).toBeVisible();

    await enableCanaryMode();
    expect(batchRelease).toBeChecked();
    expect(singleBatch).not.toBeChecked();
    await userEvent.click(singleBatch);
    expect(singleBatch).toBeChecked();
    expect(batchRelease).not.toBeChecked();
  });

  it('当用户用方向键切换发布方式时，应移动焦点并更新选中状态', async () => {
    await renderForm();
    await enableCanaryMode();
    const singleBatch = screen.getByRole('radio', { name: /^单批发布/ });
    const batchRelease = screen.getByRole('radio', { name: /^分批发布/ });

    singleBatch.focus();
    await userEvent.keyboard('{ArrowDown}');
    expect(batchRelease).toHaveFocus();
    expect(batchRelease).toBeChecked();
    await userEvent.keyboard('{ArrowUp}');
    expect(singleBatch).toHaveFocus();
    expect(singleBatch).toBeChecked();
    await userEvent.tab();
    expect(batchRelease).toHaveFocus();
    await userEvent.keyboard(' ');
    expect(batchRelease).toBeChecked();
  });

  it('当用户通过键盘选择发布方式且分批不可用时，应跳过禁用项', async () => {
    await renderForm(true);
    const singleBatch = screen.getByRole('radio', { name: /^单批发布/ });
    const batchRelease = screen.getByRole('radio', { name: /^分批发布/ });

    singleBatch.focus();
    await userEvent.keyboard('{ArrowDown}');
    expect(singleBatch).toHaveFocus();
    expect(singleBatch).toBeChecked();
    await userEvent.tab();
    expect(screen.getByTestId('submit-trigger')).toHaveFocus();
    expect(batchRelease).not.toBeChecked();
  });

  it('当用户保持默认「单批发布」提交时，部署参数不应携带发布策略', async () => {
    await renderForm();
    await fillBaseForm();
    await submitForm();

    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly.mock.calls[0][0]).toMatchObject({
      appID: 'app-a',
      envName: 'stag',
      imageTag: 'v1.0',
      replicas: 20,
    });
    expect(harness.createDeployDirectly.mock.calls[0][0]).not.toHaveProperty('releaseStrategyID');
  });

  it('当用户切换到「分批发布」时，应展开策略下拉并展示「新建策略」与「查看」入口', async () => {
    await renderForm();
    await enableCanaryMode();

    expect(await screen.findByPlaceholderText('请选择发布策略')).toBeInTheDocument();
    expect(screen.getByText('查看')).toBeInTheDocument();
    await waitFor(() => expect(harness.listReleaseStrategies).toHaveBeenCalledWith({ workspaceID: 'ws-1' }));

    // 下拉选项：空间自定义策略在前、平台内置策略在后；内置选项带「平台内置」标签
    await userEvent.click(screen.getByPlaceholderText('请选择发布策略'));
    const spaceOption = screen.getByText(/生产谨慎发布/);
    const builtinOption = screen.getByText(/分三批灰度/);
    // DOCUMENT_POSITION_FOLLOWING：内置策略选项位于空间策略选项之后
    expect(spaceOption.compareDocumentPosition(builtinOption) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByText('平台内置')).toBeInTheDocument();
    // 下拉底部固定入口：新建策略
    expect(screen.getByText('新建策略')).toBeInTheDocument();
    // 关闭下拉，避免影响后续用例的浮层查询
    await userEvent.click(document.body);
  });

  it('当用户分批发布未选策略就提交时，应被表单校验拦截且不发起部署', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await submitForm();

    // bkui 校验错误态为可见信号（错误文案经 tooltip 渲染，jsdom 不可断言）
    await waitFor(() => expect(document.body.querySelector('.bk-form-item.is-error')).toBeTruthy());
    expect(harness.createDeployDirectly).not.toHaveBeenCalled();
    expect(harness.buildAndCreateDeploy).not.toHaveBeenCalled();
  });

  it('当用户选中策略时，应展示发布预览（策略说明 + 摘要三项 + 批次时间线 + 进度方块 + 暂停标签）', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/生产谨慎发布/);

    expect(await screen.findByText('发布预览')).toBeInTheDocument();
    // 标题旁换算说明：按所选环境的 20 个实例换算
    expect(screen.getByText('按所选环境的 20 个实例换算')).toBeInTheDocument();
    // 策略下拉下方说明文案（设计稿：先 1 个实例，确认后累计到 25%、50%，最后自动全量）
    expect(screen.getByText('先 1 个实例，确认后累计到 25%、50%，最后自动全量')).toBeInTheDocument();
    // 摘要栏三项（label 上、数值+单位下）：4 批 / 2 次 / 20 个（流量批次暂不展示）
    const summaryCard = screen.getByTestId('release-summary');
    expect(within(summaryCard).getByText('批次数')).toBeInTheDocument();
    expect(within(summaryCard).getByText('暂停次数')).toBeInTheDocument();
    expect(within(summaryCard).getByText('实例总数')).toBeInTheDocument();
    expect(within(summaryCard).queryByText('流量批次')).not.toBeInTheDocument();
    expect(within(summaryCard).getByText('4')).toBeInTheDocument();
    expect(within(summaryCard).getByText('批')).toBeInTheDocument();
    expect(within(summaryCard).getByText('2')).toBeInTheDocument();
    expect(within(summaryCard).getByText('次')).toBeInTheDocument();
    expect(within(summaryCard).getByText('20')).toBeInTheDocument();
    // 批次时间线：首批数值 target 展示「N 个实例」，其余展示「累计 N 个（P%）」，末批带全量标签
    expect(screen.getByText('第 1 批 · 1 个实例')).toBeInTheDocument();
    expect(screen.getByText('第 2 批 · 累计 5 个（25%）')).toBeInTheDocument();
    expect(screen.getByText('第 3 批 · 累计 10 个（50%）')).toBeInTheDocument();
    expect(screen.getByText('第 4 批 · 累计 20 个（100%）')).toBeInTheDocument();
    expect(screen.getByText('全量')).toBeInTheDocument();
    // 末批已是全量，底部不展示「剩余实例自动更新」提示
    expect(screen.queryByTestId('release-remaining-tip')).not.toBeInTheDocument();
    // 右侧实例分数：每批一行「新版本 X · 旧版本 Y」（发布预览面板改版后的口径）
    expect(screen.getByText('新版本 1 · 旧版本 19')).toBeInTheDocument();
    expect(screen.getByText('新版本 5 · 旧版本 15')).toBeInTheDocument();
    expect(screen.getByText('新版本 10 · 旧版本 10')).toBeInTheDocument();
    expect(screen.getByText('新版本 20 · 旧版本 0')).toBeInTheDocument();
    // 无暂停配置的批次（除末批）带「不停」标签：4 批中仅批次 1
    expect(screen.getAllByText('不停，直接进入下一批')).toHaveLength(1);
    // 实例进度方块：每批一行 20 个方块，宽度按实例比例均分（flex-1）；
    // 颜色：浅蓝=本批要发布（首批首方块），深蓝=此前批次已发布（次批首方块）
    const squareRows = screen.getAllByTestId('batch-squares');
    expect(squareRows).toHaveLength(4);
    for (const row of squareRows) {
      expect(row.children).toHaveLength(20);
    }
    expect(squareRows[0].children[0].className).toContain('bg-[#A3C5FD]');
    expect(squareRows[1].children[0].className).toContain('bg-[#3A84FF]');
    // 暂停标签：permanent → 橙色（warning）；duration → 蓝色（info）
    expect(screen.getByText('暂停 600 秒后自动继续')).toBeInTheDocument();
    expect(screen.getByText('永久暂停 · 手动继续')).toBeInTheDocument();
  });

  it('当策略末批未到全量时，预览不补齐全量批而在底部提示剩余实例自动更新', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/末批部分放量/);

    // 提示收尾统一为「最后自动全量」：末批 50% 归入累计段，不再出现「最后累计到 50%」
    expect(await screen.findByText('先 1 个实例，确认后累计到 25%、50%，最后自动全量')).toBeInTheDocument();
    // 只展示用户配置的 3 批，不再追加第 4 批系统补齐的全量批（与策略预览弹窗一致）
    expect(screen.getByText('第 3 批 · 累计 10 个（50%）')).toBeInTheDocument();
    expect(screen.queryByText('第 4 批 · 累计 20 个（100%）')).not.toBeInTheDocument();
    expect(screen.getAllByTestId('batch-squares')).toHaveLength(3);
    // 底部提示剩余实例在最后一批完成后自动更新
    expect(screen.getByTestId('release-remaining-tip')).toHaveTextContent(
      '最后一批完成后，剩余未更新的实例将自动更新。',
    );
    // 摘要批次数按用户配置的 3 批统计，与策略预览摘要口径一致
    const summaryCard = screen.getByTestId('release-summary');
    expect(within(summaryCard).getByText('批次数')).toBeInTheDocument();
    expect(within(summaryCard).getByText('3')).toBeInTheDocument();
  });

  it('当用户分批发布并已选策略提交时，部署参数应携带发布策略 ID', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/生产谨慎发布/);
    await submitForm();

    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly).toHaveBeenCalledWith(
      expect.objectContaining({
        appID: 'app-a',
        envName: 'stag',
        imageTag: 'v1.0',
        replicas: 20,
        releaseStrategyID: 'strategy-1',
      }),
    );
  });

  it('当用户切回「单批发布」时，应收起预览并清空已选策略，提交不带发布策略', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/生产谨慎发布/);
    await screen.findByText('发布预览');

    await userEvent.click(screen.getByRole('radio', { name: /^单批发布/ }));
    expect(screen.queryByText('发布预览')).not.toBeInTheDocument();

    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly.mock.calls[0][0]).not.toHaveProperty('releaseStrategyID');
  });

  it('当目标环境为联邦集群时，应禁用「分批发布」按钮', async () => {
    await renderForm(true);
    await fillBaseForm();

    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
    // 单批发布仍可用，提交链路不受影响
    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
  });

  it('当用户切换到从源码构建时，应禁用分批发布，且已选的 canary 自动回落为单批发布', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/生产谨慎发布/);
    await screen.findByText('发布预览');

    await userEvent.click(screen.getByRole('button', { name: '从源码构建' }));

    // 回落：预览收起，分批按钮进入禁用态
    await waitFor(() => expect(screen.queryByText('发布预览')).not.toBeInTheDocument());
    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
  });

  it('当用户切换到从源码构建时，应把默认分支回填给修订选择器', async () => {
    await renderForm();
    // 切到源码模式后修订选择器才通过 v-if 挂载：必须等下一轮 tick 再 prepare，
    // 且模板 ref 只有存在同名顶层绑定时才会写回（缺解构时 prepare 会静默空转）。
    await userEvent.click(screen.getByRole('button', { name: '从源码构建' }));

    expect(await screen.findByTestId('code-revision-field-stub')).toBeInTheDocument();
    await waitFor(() => expect(codeRevisionCalls.prepare).toHaveBeenCalledWith('master'));
  });

  it('当应用为 Helm 应用时，应禁用「分批发布」按钮（仅应用模型应用支持）', async () => {
    appState.appType = 'helm';
    await renderForm();
    await fillBaseForm();

    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
    // 单批发布仍可用，提交链路不受影响
    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
  });

  it('当主工作负载为 Deployment 时，应禁用「分批发布」按钮（仅 GameDeployment 支持）', async () => {
    harness.listDeployRecords.mockResolvedValue({ count: 1, results: [{ workloadKind: 'Deployment' }] });
    await renderForm();
    await fillBaseForm();

    await waitFor(() => expect(harness.listDeployRecords).toHaveBeenCalled());
    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
    // 分批发布已失效时提交不应透传 releaseStrategyID（当前为单批发布，正常部署）
    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly.mock.calls[0][0]).not.toHaveProperty('releaseStrategyID');
  });

  it('当主工作负载资格检查完成前，应先禁用分批发布；返回 Deployment 后保持禁用', async () => {
    let resolveRecords!: (value: { count: number; results: unknown[] }) => void;
    harness.listDeployRecords.mockReturnValue(
      new Promise(resolve => {
        resolveRecords = resolve;
      }),
    );
    await renderForm();
    await fillBaseForm();

    // 资格未知时不允许先选分批再被回退：检查完成前直接禁用
    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
    expect(screen.queryByText('发布预览')).not.toBeInTheDocument();

    resolveRecords({ count: 1, results: [{ workloadKind: 'Deployment' }] });
    await waitFor(() => expect(harness.listDeployRecords).toHaveBeenCalled());
    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();
  });

  it('当主工作负载为 GameDeployment 时，应允许「分批发布」', async () => {
    harness.listDeployRecords.mockResolvedValue({ count: 1, results: [{ workloadKind: 'GameDeployment' }] });
    await renderForm();
    await fillBaseForm();

    await waitFor(() => expect(harness.listDeployRecords).toHaveBeenCalled());
    // 可正常展开策略下拉
    await enableCanaryMode();
    expect(await screen.findByPlaceholderText('请选择发布策略')).toBeInTheDocument();
  });

  it.each([
    ['无部署 / 构建记录（data 为 null）', null],
    ['只有构建记录（hasDeployRecord=false）', { hasDeployRecord: false, status: 'failed' }],
    ['工作负载已下线（status=uninstalled）', { hasDeployRecord: true, status: 'uninstalled' }],
  ])('当目标环境为首次部署（%s）时，应禁用「分批发布」', async (_name, latest) => {
    harness.listLatestDeployRecords.mockResolvedValue(latest);
    await renderForm();
    await fillBaseForm();

    // 首部署检查按环境 / 泳道查询，空泳道传空串
    await waitFor(() => expect(harness.listLatestDeployRecords).toHaveBeenCalled());
    expect(harness.listLatestDeployRecords).toHaveBeenCalledWith({
      appID: 'app-a',
      envName: 'stag',
      trafficLaneName: '',
    });
    // 等检查结果落地（promise 微任务），确认不是仅因「检查中」而禁用
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(screen.getByRole('radio', { name: /^分批发布/ })).toBeDisabled();

    // 单批发布仍可用，提交链路不受影响
    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly.mock.calls[0][0]).not.toHaveProperty('releaseStrategyID');
  });

  it('当策略为两批（25% → 100%）时，提示与预览应保留首批灰度阶段，不误判为单批全量', async () => {
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await chooseStrategy(/两批放量/);

    // 首批 25% 不能被「单批全量」误判吞掉
    expect(await screen.findByText('先更新到 25%，最后自动全量')).toBeInTheDocument();
    expect(screen.queryByText('最后自动全量，先更新到 25%')).not.toBeInTheDocument();
    // 两批均有批次行：25% → 100%（末批已全量，无需系统补齐第 3 批）
    expect(screen.getByText('第 1 批 · 累计 5 个（25%）')).toBeInTheDocument();
    expect(screen.getByText('第 2 批 · 累计 20 个（100%）')).toBeInTheDocument();
    expect(screen.getAllByTestId('batch-squares')).toHaveLength(2);
  });
});

describe('部署表单：抽屉内新建发布策略', () => {
  /** 新建抽屉保存后接口回带的策略：用于断言「自动选中」 */
  const createdStrategy = {
    id: 'strategy-new',
    name: '灰度两批',
    description: '',
    isBuiltin: false,
    workspaceID: 'ws-1',
    batches: [{ target: '10', pauseMode: 'permanent' }],
  };

  it('当用户点「新建策略」时，应在部署表单之上叠加新建策略抽屉', async () => {
    await renderForm();
    await enableCanaryMode();

    const header = await openCreateStrategyDrawer();

    // 标题：新建策略（与下拉里的同名入口区分，限定在抽屉头内断言）
    expect(within(header).getByText('新建策略')).toBeInTheDocument();
    // 部署表单仍在抽屉之下保持可交互（叠加而非替换）
    expect(screen.getByRole('radio', { name: /^单批发布/ })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('请选择发布策略')).toBeInTheDocument();
  });

  it('当用户在新建抽屉保存成功后，应关闭抽屉、自动选中新策略并刷新发布预览', async () => {
    harness.createReleaseStrategy.mockResolvedValue(createdStrategy);
    await renderForm();
    await fillBaseForm();
    await enableCanaryMode();
    await openCreateStrategyDrawer();

    // 保存后列表刷新带回新策略（第一次挂载时的列表不含它）
    harness.listReleaseStrategies.mockResolvedValue([...strategyList, createdStrategy]);

    await userEvent.type(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'), '灰度两批');
    await fireEvent.update(screen.getByTestId('target-value'), '10');
    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    await waitFor(() =>
      expect(harness.createReleaseStrategy).toHaveBeenCalledWith(
        {
          workspaceID: 'ws-1',
          name: '灰度两批',
          description: '',
          batches: [{ target: '10', pauseMode: 'permanent' }],
        },
        { interceptorErr: false },
      ),
    );

    // 抽屉关闭 + 策略列表重拉（挂载时 1 次 + 保存后 1 次）
    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(harness.listReleaseStrategies).toHaveBeenCalledTimes(2);
    // 自动选中新策略 → 发布预览按 20 个实例换算出 1 批 10 个
    expect(await screen.findByText('发布预览')).toBeInTheDocument();
    expect(screen.getByText('第 1 批 · 10 个实例')).toBeInTheDocument();

    // 自动选中生效：提交部署时携带新策略 ID
    await submitForm();
    await waitFor(() => expect(harness.createDeployDirectly).toHaveBeenCalledTimes(1));
    expect(harness.createDeployDirectly.mock.calls[0][0]).toMatchObject({ releaseStrategyID: 'strategy-new' });
  });

  it('当用户点「查看」时，应打开已选策略的查看抽屉', async () => {
    await renderForm();
    await enableCanaryMode();

    // 未选中策略时「查看」置灰不可点
    expect(screen.getByRole('button', { name: '查看' })).toBeDisabled();

    // 选中策略后点击「查看」：叠加查看抽屉，展示所选策略
    await chooseStrategy(/生产谨慎发布/);
    await userEvent.click(screen.getByRole('button', { name: '查看' }));
    const header = await screen.findByTestId('strategy-sideslider-header');
    expect(within(header).getByText('查看策略')).toBeInTheDocument();
    expect(within(header).getByText('生产谨慎发布')).toBeInTheDocument();
    // 部署页的查看抽屉为纯查看（readonly）：页脚只保留「关闭」，不提供克隆 / 预览入口，
    // 克隆入口在「发布策略」管理页，避免在部署流程中派生新策略
    expect(screen.getByRole('button', { name: '关闭' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '克隆为空间策略' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '预览策略' })).not.toBeInTheDocument();
  });
});
