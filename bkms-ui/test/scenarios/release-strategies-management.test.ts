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
 * 场景级测试：发布策略管理页（ROUTES 未收录，本头即场景说明）
 *
 * V = 19（判定节点：列表加载成功 / 加载失败 / 搜索过滤空态 / 名称点击打开抽屉 / 新建提交 /
 *          新建校验拦截 / 新增批次追加到末尾 / 仅剩一批时不可删除 / 累计值递增校验提示 /
 *          预览前置校验拦截 / 预览弹窗换算 / 编辑提交 / 查看只读 / 查看态克隆为空间策略 /
 *          克隆取消不创建 / 删除确认 / 删除取消 / 新建后回到首页并选中 / 克隆后回到首页并选中）
 * 路径：
 *   1. 列表加载成功：空间策略在前、内置策略带「平台内置」标签、批次摘要渲染
 *   2. 列表加载失败：顶部展示错误提示（保留旧数据）
 *   3. 搜索过滤：关键字命中单条；无命中展示搜索空态
 *   3.1 名称点击：空间策略打开编辑抽屉（可编辑回显），内置策略打开只读查看抽屉
 *   4. 新建策略：抽屉填写名称 + 批次累计值 → 提交调用 create 并提示「策略已创建」
 *   5. 新建校验拦截：名称为空提交 → 内联报错且不调用 create
 *   6. 新增批次：追加到批次列表末尾并自动选中（全量批由系统隐含补齐，不需要用户配置）
 *   7. 最少一批：仅剩一个批次时不提供删除入口
 *   8. 累计值递增校验：不大于上一批时卡片红字 + 输入框红框，未修正不提交
 *   9. 预览前置校验：表单未通过校验时点击「预览策略」不打开弹窗
 *   10. 预览弹窗：校验通过后打开，按实例数换算批次时间线（仅确认，无提交入口）
 *   11. 编辑策略：抽屉回显名称 → 提交调用 update
 *   12. 查看策略：抽屉只读，提供「克隆为空间策略 / 预览策略 / 关闭」
 *   13. 查看态克隆：切到克隆表单、预填「原名_克隆01」与全部批次，提交走 create
 *   14. 克隆取消：关闭抽屉，不产生新策略
 *   15. 删除策略：InfoBox 确认后调用 delete 并重新拉取列表
 *   16. 删除取消：InfoBox 不确认 → 不调用 delete
 *   17. 保存定位：新建 / 克隆成功后重新拉取列表、回到第一页并选中该行（关键词过滤掉时先清空搜索）
 *
 * stub/mock：
 *   - ReleaseStrategiesService（list/create/update/delete）→ harness vi.fn
 *   - vue-i18n useI18n → 直译 + {0} 插值
 *   - @blueking/table → tableMockFactory（行按插槽渲染，可断言行内操作/批次文案）
 *   - bkui-vue partial mock：Message / InfoBox 命令式 API stub，其余真实渲染（含 Popover/Select/Dialog）
 *   - FlexRow → 透传 stub（仅透传 left/right 插槽）
 *   - TableException → vi.mock 空 stub（空态仅断言 table-empty 区块存在）
 *
 * backlog（不确定项，待业务确认）：
 *   - 批次列明细 Popover 内容经 bkui Popover hover 触发，jsdom 无法稳定断言，仅覆盖默认插槽的「N 批 + 摘要」
 *   - 抽屉内「目标资源」为数字输入 + 单位下拉，jsdom 下 Select 展开交互不稳定，仅覆盖数字输入与本单位默认值
 */
import userEvent from '@testing-library/user-event';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/vue';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import ReleaseStrategiesPage from '~/pages/basic/release-strategies/index.vue';

import { I18nTStub } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

const harness = vi.hoisted(() => ({
  listReleaseStrategies: vi.fn(),
  createReleaseStrategy: vi.fn(),
  updateReleaseStrategy: vi.fn(),
  deleteReleaseStrategy: vi.fn(),
}));

const mocks = vi.hoisted(() => ({
  message: vi.fn(),
  infoBox: vi.fn(),
}));

const strategyList = [
  {
    id: 'strategy-1',
    name: '生产谨慎发布',
    description: '谨慎放量',
    isBuiltin: false,
    workspaceID: 'ws-1',
    updatedAt: '2026-01-02T00:00:00Z',
    batches: [
      { target: '1', pauseMode: 'none' },
      { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
      { target: '50%', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
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
  useI18n: () => ({
    t: (s: string, params?: unknown[]) => {
      let out = s;
      if (Array.isArray(params)) params.forEach((p, i) => (out = out.replace(`{${i}}`, String(p))));
      return out;
    },
    te: () => true,
  }),
}));

vi.mock('~/api/modules/v1/release-strategies', () => ({
  ReleaseStrategiesService: {
    listReleaseStrategies: harness.listReleaseStrategies,
    createReleaseStrategy: harness.createReleaseStrategy,
    updateReleaseStrategy: harness.updateReleaseStrategy,
    deleteReleaseStrategy: harness.deleteReleaseStrategy,
  },
}));

vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1' }),
}));

vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());

vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
  InfoBox: mocks.infoBox,
}));

vi.mock('~/components/table-exception.vue', () => ({ default: { template: '<div data-testid="table-exception" />' } }));

const FlexRowStub = { template: '<div><slot name="left" /><slot name="right" /></div>' };

/** 必填项填写：名称 + 首批累计值 */
async function fillCreateForm(name: string, target: string) {
  const nameInput = screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
  await userEvent.type(nameInput, name);
  const targetInput = screen.getByTestId('target-value');
  await fireEvent.update(targetInput, target);
  return { nameInput, targetInput };
}

/** 打开「新建策略」抽屉并等待回填完成（默认一个待填写批次） */
async function openCreateDrawer() {
  await renderPage();
  await userEvent.click(await screen.findByRole('button', { name: '新建策略' }));
  await screen.findByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
}

async function renderPage() {
  return render(ReleaseStrategiesPage, {
    global: {
      mocks: { $t: translate },
      // 说明文案含加粗片段，走 <i18n-t> 插值组件；此处按 keypath 直译
      components: { 'i18n-t': I18nTStub },
      stubs: {
        FlexRow: FlexRowStub,
      },
    },
  });
}

function translate(key: string, params?: unknown[]) {
  let out = key;
  if (Array.isArray(params)) params.forEach((p, i) => (out = out.replace(`{${i}}`, String(p))));
  return out;
}

beforeAll(() => {
  installVxeShims();
});

beforeEach(() => {
  vi.clearAllMocks();
  harness.listReleaseStrategies.mockResolvedValue(strategyList);
  harness.createReleaseStrategy.mockResolvedValue({});
  harness.updateReleaseStrategy.mockResolvedValue({});
  harness.deleteReleaseStrategy.mockResolvedValue({});
  mocks.infoBox.mockImplementation(({ onConfirm }: { onConfirm?: () => void }) => onConfirm?.());
});

describe('发布策略管理：列表与搜索', () => {
  it('当用户进入页面时，应加载策略列表并渲染名称、内置标签与批次摘要', async () => {
    await renderPage();

    expect(await screen.findByText('生产谨慎发布')).toBeInTheDocument();
    expect(screen.getByText('分三批灰度')).toBeInTheDocument();
    expect(screen.getByText('平台内置')).toBeInTheDocument();
    // 批次摘要：4 批策略按 (token) 拼接
    expect(screen.getByText('(1个) · (25%) · (50%) · (全量)')).toBeInTheDocument();
    await waitFor(() => expect(harness.listReleaseStrategies).toHaveBeenCalledWith({ workspaceID: 'ws-1' }));
  });

  it('当列表加载失败时，应展示顶部错误提示', async () => {
    harness.listReleaseStrategies.mockRejectedValue(new Error('boom'));
    await renderPage();

    expect(await screen.findByText('发布策略加载失败，请稍后重试')).toBeInTheDocument();
  });

  it('当用户按名称搜索时，应过滤列表；无命中时展示空态', async () => {
    await renderPage();
    await screen.findByText('生产谨慎发布');

    const searchInput = screen.getByPlaceholderText('搜索策略名称关键字');
    await userEvent.type(searchInput, '生产谨慎');
    expect(screen.getByText('生产谨慎发布')).toBeInTheDocument();
    expect(screen.queryByText('分三批灰度')).not.toBeInTheDocument();

    await userEvent.clear(searchInput);
    await userEvent.type(searchInput, '不存在');
    expect(screen.queryByText('生产谨慎发布')).not.toBeInTheDocument();
    expect(screen.getByTestId('table-exception')).toBeInTheDocument();
  });

  it('当用户点击策略名称时，空间策略打开编辑抽屉、内置策略打开只读查看抽屉', async () => {
    await renderPage();
    await userEvent.click(await screen.findByRole('button', { name: '生产谨慎发布' }));

    const header = await screen.findByTestId('strategy-sideslider-header');
    expect(within(header).getByText('编辑策略')).toBeInTheDocument();
    const nameInput = screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
    expect(nameInput).toHaveValue('生产谨慎发布');
    expect(nameInput).not.toBeDisabled();

    await userEvent.click(screen.getByRole('button', { name: '取消' }));
    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: '分三批灰度' }));

    const viewHeader = await screen.findByTestId('strategy-sideslider-header');
    expect(within(viewHeader).getByText('查看策略')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符')).toBeDisabled();
  });
});

describe('发布策略管理：新建 / 编辑 / 查看', () => {
  it('当用户新建策略并提交时，应调用 create 并提示成功', async () => {
    await openCreateDrawer();
    await fillCreateForm('灰度放量', '10');

    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    await waitFor(() => expect(harness.createReleaseStrategy).toHaveBeenCalledTimes(1));
    expect(harness.createReleaseStrategy).toHaveBeenCalledWith(
      expect.objectContaining({
        workspaceID: 'ws-1',
        name: '灰度放量',
        // 仅提交用户配置的批次（默认永久暂停），全量由系统在末批确认后自动补齐
        batches: [{ target: '10', pauseMode: 'permanent' }],
      }),
      { interceptorErr: false },
    );
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '策略已创建' });
  });

  it('新建策略保存成功后，应回到第一页并选中该策略行', async () => {
    // 12 条历史策略：首页只切出 10 行，用于验证保存后回到第一页
    const others = Array.from({ length: 12 }, (_, index) => ({
      id: `strategy-old-${index}`,
      name: `历史策略${index}`,
      description: '',
      isBuiltin: false,
      workspaceID: 'ws-1',
      updatedAt: '2026-01-01T00:00:00Z',
      batches: [{ target: '1', pauseMode: 'none' }],
    }));
    const created = {
      id: 'strategy-new',
      name: '灰度放量',
      description: '',
      isBuiltin: false,
      workspaceID: 'ws-1',
      updatedAt: '2026-02-01T00:00:00Z',
      batches: [{ target: '3', pauseMode: 'permanent' }],
    };
    harness.listReleaseStrategies.mockResolvedValueOnce(others).mockResolvedValue([created, ...others]);
    harness.createReleaseStrategy.mockResolvedValue(created);

    await openCreateDrawer();
    await fillCreateForm('灰度放量', '3');
    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    // 保存后重新拉取列表
    await waitFor(() => expect(harness.listReleaseStrategies).toHaveBeenCalledTimes(2));
    // 选中刚保存的行（vxe row-config.isCurrent）
    const currentRow = await waitFor(() => {
      const node = document.querySelector('[data-current="true"]');
      if (!node) throw new Error('尚未选中新建的策略行');
      return node as HTMLElement;
    });
    expect(within(currentRow).getByText('灰度放量')).toBeInTheDocument();
    // 回到第一页：13 条数据只切出 10 行，新建行（更新时间最新）排在首行
    const rows = screen.getAllByTestId('table-row');
    expect(rows).toHaveLength(10);
    expect(rows[0]).toBe(currentRow);
  });

  it('克隆策略保存成功后，应回到第一页并选中克隆出的策略行', async () => {
    const others = Array.from({ length: 12 }, (_, index) => ({
      id: `strategy-old-${index}`,
      name: `历史策略${index}`,
      description: '',
      isBuiltin: false,
      workspaceID: 'ws-1',
      updatedAt: '2026-01-01T00:00:00Z',
      batches: [{ target: '1', pauseMode: 'none' }],
    }));
    const cloned = {
      id: 'strategy-cloned',
      name: '生产谨慎发布_克隆01',
      description: '谨慎放量',
      isBuiltin: false,
      workspaceID: 'ws-1',
      updatedAt: '2026-02-01T00:00:00Z',
      batches: [{ target: '1', pauseMode: 'none' }],
    };
    harness.listReleaseStrategies
      .mockResolvedValueOnce([strategyList[0], ...others])
      .mockResolvedValue([cloned, strategyList[0], ...others]);
    harness.createReleaseStrategy.mockResolvedValue(cloned);

    await renderPage();
    await screen.findByText('生产谨慎发布');
    await userEvent.click(screen.getAllByRole('button', { name: '克隆' })[0]);
    // 克隆态名称已预填，直接提交
    await userEvent.click(await screen.findByRole('button', { name: '提交' }));

    await waitFor(() => expect(harness.createReleaseStrategy).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(harness.listReleaseStrategies).toHaveBeenCalledTimes(2));
    const currentRow = await waitFor(() => {
      const node = document.querySelector('[data-current="true"]');
      if (!node) throw new Error('尚未选中克隆出的策略行');
      return node as HTMLElement;
    });
    expect(within(currentRow).getByText('生产谨慎发布_克隆01')).toBeInTheDocument();
    expect(screen.getAllByTestId('table-row')).toHaveLength(10);
  });

  it('当用户新建策略名称为空时，应被校验拦截且不调用 create', async () => {
    await openCreateDrawer();

    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    expect(await screen.findByText('策略名称不能为空')).toBeInTheDocument();
    // 校验失败定位到第一个错误：焦点回到名称输入框
    await waitFor(() => expect(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符')).toHaveFocus());
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当新建名称与当前空间已有策略重名时，失焦即提示并在提交时拦截', async () => {
    await openCreateDrawer();
    await fillCreateForm('生产谨慎发布', '10');

    // 失焦即按列表数据同步提示，无需等提交
    await fireEvent.blur(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'));
    expect(await screen.findByText('策略名称在当前空间已存在')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    expect(await screen.findByText('策略名称在当前空间已存在')).toBeInTheDocument();
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当编辑策略未改名时不应提示重名，改成其他已存在名称时才拦截', async () => {
    harness.listReleaseStrategies.mockResolvedValue([
      ...strategyList,
      {
        id: 'strategy-2',
        name: '快速全量',
        description: '',
        isBuiltin: false,
        workspaceID: 'ws-1',
        updatedAt: '2026-01-01T00:00:00Z',
        batches: [{ target: '100%', pauseMode: 'none' }],
      },
    ]);
    await renderPage();
    await screen.findByText('生产谨慎发布');
    // 两行空间策略各有一个「编辑」入口，按更新时间倒序取第一行（生产谨慎发布）
    await userEvent.click(screen.getAllByRole('button', { name: '编辑' })[0]);

    const nameInput = await screen.findByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
    expect(nameInput).toHaveValue('生产谨慎发布');

    // 未改名：与自身同名不算重复
    await fireEvent.blur(nameInput);
    expect(screen.queryByText('策略名称在当前空间已存在')).not.toBeInTheDocument();

    // 改成另一条已有策略名称：提示重名并拦截提交
    await fireEvent.update(nameInput, '快速全量');
    await fireEvent.blur(nameInput);
    expect(await screen.findByText('策略名称在当前空间已存在')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '提交' }));
    expect(await screen.findByText('策略名称在当前空间已存在')).toBeInTheDocument();
    expect(harness.updateReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当用户新增批次时，应追加到末尾并自动选中新批次', async () => {
    await openCreateDrawer();

    // 默认一个批次：累计值留空待填写
    expect(screen.getAllByTestId('batch-card')).toHaveLength(1);

    await userEvent.click(screen.getByRole('button', { name: '发布批次' }));

    const cards = screen.getAllByTestId('batch-card');
    expect(cards).toHaveLength(2);
    expect(within(cards[0]).getByText('第 1 批 · 待填写')).toBeInTheDocument();
    // 新增批次同样不预填累计值，追加在末尾
    expect(within(cards[1]).getByText('第 2 批 · 待填写')).toBeInTheDocument();
    // 新增批次后自动选中，详情面板头部同步为第 2 批
    expect(within(screen.getByTestId('batch-detail')).getByText('第 2 批 · 待填写')).toBeInTheDocument();
  });

  it('当仅剩一个批次时不提供删除入口，新增后恢复且默认隐藏（hover 才出现）', async () => {
    await openCreateDrawer();

    // 至少保留一批：默认只有一个批次时不可删除
    expect(within(screen.getByTestId('batch-card')).queryByTestId('batch-remove')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '发布批次' }));

    const cards = screen.getAllByTestId('batch-card');
    expect(cards).toHaveLength(2);
    const removeButton = within(cards[0]).getByTestId('batch-remove');
    // 默认态（即使处于选中态）不可见，仅 hover 卡片时出现
    expect(removeButton.className).toContain('opacity-0');
    expect(removeButton.className).toContain('group-hover:opacity-100');
    expect(within(cards[1]).getByTestId('batch-remove')).toBeInTheDocument();
  });

  it('当用户删除批次时，应移除该批次并选中相邻批次，删到仅剩一批后不再可删', async () => {
    await openCreateDrawer();
    await userEvent.click(screen.getByRole('button', { name: '发布批次' }));

    const cards = screen.getAllByTestId('batch-card');
    expect(cards).toHaveLength(2);
    // 删除第 1 批后，剩余批次顺位补为第 1 批并自动选中
    await userEvent.click(within(cards[0]).getByTestId('batch-remove'));

    const restCards = screen.getAllByTestId('batch-card');
    expect(restCards).toHaveLength(1);
    expect(within(restCards[0]).getByText('第 1 批 · 待填写')).toBeInTheDocument();
    expect(within(screen.getByTestId('batch-detail')).getByText('第 1 批 · 待填写')).toBeInTheDocument();
    expect(within(restCards[0]).queryByTestId('batch-remove')).not.toBeInTheDocument();
  });

  it('当批次累计值不大于上一批时，批次卡片与目标资源输入框应同时标红提示', async () => {
    await openCreateDrawer();
    await userEvent.type(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'), '灰度放量');
    // 第 1 批累计 10 个
    await fireEvent.update(screen.getByTestId('target-value'), '10');
    // 新增第 2 批并填写 5 个，小于上一批，触发「需大于上一批」
    await userEvent.click(screen.getByRole('button', { name: '发布批次' }));
    await fireEvent.update(screen.getByTestId('target-value'), '5');

    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    // 卡片第二行用简短文案替换暂停说明
    expect(await screen.findByTestId('batch-card-error')).toHaveTextContent('累计更新数需大于上一批');
    // 校验失败定位到第一个错误批次：详情面板切到出错的第 2 批
    expect(within(screen.getByTestId('batch-detail')).getByText('第 2 批 · 累计 5 个')).toBeInTheDocument();
    // 详情面板补上上一批累计值
    expect(screen.getByTestId('target-error')).toHaveTextContent('累计更新数需大于上一批（10 个）');
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当表单未通过校验时，点击「预览策略」应拦截并展示校验错误', async () => {
    await openCreateDrawer();

    await userEvent.click(screen.getByRole('button', { name: '预览策略' }));

    expect(await screen.findByText('策略名称不能为空')).toBeInTheDocument();
    expect(screen.queryByTestId('preview-row')).not.toBeInTheDocument();
  });

  it('预览弹窗应展示批次时间线且不提供实例总数输入', async () => {
    await openCreateDrawer();
    await fillCreateForm('灰度放量', '3');

    await userEvent.click(screen.getByRole('button', { name: '预览策略' }));

    const rows = await screen.findAllByTestId('preview-row');
    // 用户只配了 1 批（累计 3 个），末批未到全量，系统补齐一个全量批 → 2 行
    expect(rows).toHaveLength(2);
    // 首批展示配置的目标值 + 「首批」说明
    expect(within(rows[0]).getByText('3 个')).toBeInTheDocument();
    expect(within(rows[0]).getByText('首批')).toBeInTheDocument();
    // 系统补齐的全量批带「全量」标签
    expect(within(rows[1]).getByTestId('preview-row-full')).toBeInTheDocument();
    // 摘要栏四项指标：批次数 / 暂停次数 / 首批数量 / 流量批次
    expect(screen.getByText('首批数量')).toBeInTheDocument();
    expect(screen.getByText('流量批次')).toBeInTheDocument();
    // 本期无失败策略、无实例总数换算，改为按配置比例示意
    expect(screen.queryByText('失败策略：暂停当前批次')).not.toBeInTheDocument();
    expect(screen.queryByText('实例总数')).not.toBeInTheDocument();
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });

  it('预览弹窗只做确认，不提供提交入口，关闭后需回到抽屉提交', async () => {
    await openCreateDrawer();
    await fillCreateForm('灰度放量', '3');

    await userEvent.click(screen.getByRole('button', { name: '预览策略' }));
    await screen.findAllByTestId('preview-row');
    // 预览仅用于确认配置，弹窗内不再提供提交按钮
    expect(screen.queryByTestId('preview-submit')).not.toBeInTheDocument();

    await userEvent.click(screen.getByTestId('preview-close'));
    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    await waitFor(() => expect(harness.createReleaseStrategy).toHaveBeenCalledTimes(1));
    expect(harness.createReleaseStrategy).toHaveBeenCalledWith(
      expect.objectContaining({
        name: '灰度放量',
        batches: [{ target: '3', pauseMode: 'permanent' }],
      }),
      { interceptorErr: false },
    );
  });

  it('当用户编辑策略时，应回显名称并调用 update', async () => {
    await renderPage();
    await screen.findByText('生产谨慎发布');
    await userEvent.click(screen.getByRole('button', { name: '编辑' }));

    const nameInput = await screen.findByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
    expect(nameInput).toHaveValue('生产谨慎发布');
    // 标题：编辑策略 | 策略名称
    const header = await screen.findByTestId('strategy-sideslider-header');
    expect(within(header).getByText('编辑策略')).toBeInTheDocument();
    expect(within(header).getByText('生产谨慎发布')).toBeInTheDocument();
    await fireEvent.update(nameInput, '生产谨慎发布v2');
    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    await waitFor(() => expect(harness.updateReleaseStrategy).toHaveBeenCalledTimes(1));
    expect(harness.updateReleaseStrategy).toHaveBeenCalledWith(
      expect.objectContaining({
        strategyID: 'strategy-1',
        name: '生产谨慎发布v2',
        batches: [
          { target: '1', pauseMode: 'none' },
          { target: '25%', pauseMode: 'duration', pauseSeconds: 600 },
          { target: '50%', pauseMode: 'permanent' },
          { target: '100%', pauseMode: 'none' },
        ],
      }),
      { interceptorErr: false },
    );
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '策略已更新' });
  });

  it('当用户查看内置策略时，抽屉应只读并提供克隆与预览入口', async () => {
    await renderPage();
    await screen.findByText('分三批灰度');
    await userEvent.click(screen.getByRole('button', { name: '查看' }));

    expect(await screen.findByText('查看策略')).toBeInTheDocument();
    // 标题：查看策略 | 策略名称 + 平台内置标签
    const header = await screen.findByTestId('strategy-sideslider-header');
    expect(within(header).getByText('分三批灰度')).toBeInTheDocument();
    expect(within(header).getByText('平台内置')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符')).toBeDisabled();
    expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '发布批次' })).not.toBeInTheDocument();
    // 查看态 footer：克隆为空间策略 / 预览策略 / 关闭
    expect(screen.getByRole('button', { name: '克隆为空间策略' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '预览策略' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '关闭' })).toBeInTheDocument();
  });

  it('当用户在查看态点击「克隆为空间策略」时，应切换为克隆表单并预填不重名的默认名称', async () => {
    await renderPage();
    await screen.findByText('分三批灰度');
    await userEvent.click(screen.getByRole('button', { name: '查看' }));
    await userEvent.click(await screen.findByRole('button', { name: '克隆为空间策略' }));

    // 标题切到克隆策略，原策略仍作为副标题展示
    const header = await screen.findByTestId('strategy-sideslider-header');
    expect(within(header).getByText('克隆策略')).toBeInTheDocument();
    expect(within(header).getByText('分三批灰度')).toBeInTheDocument();

    // 名称预填「原名_克隆01」且可编辑；批次配置全部预填
    const nameInput = screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符');
    expect(nameInput).toHaveValue('分三批灰度_克隆01');
    expect(nameInput).not.toBeDisabled();
    expect(screen.getAllByTestId('batch-card')).toHaveLength(3);
    expect(screen.getByRole('button', { name: '提交' })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '提交' }));

    await waitFor(() => expect(harness.createReleaseStrategy).toHaveBeenCalledTimes(1));
    expect(harness.createReleaseStrategy).toHaveBeenCalledWith(
      expect.objectContaining({
        workspaceID: 'ws-1',
        name: '分三批灰度_克隆01',
        description: '平台内置策略',
        batches: [
          { target: '1', pauseMode: 'none' },
          { target: '50%', pauseMode: 'none' },
          { target: '100%', pauseMode: 'none' },
        ],
      }),
      { interceptorErr: false },
    );
    // 克隆走新建，不得修改原策略
    expect(harness.updateReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当用户在克隆表单点击「取消」时，应直接关闭且不生成新策略', async () => {
    await renderPage();
    await screen.findByText('分三批灰度');
    await userEvent.click(screen.getByRole('button', { name: '查看' }));
    await userEvent.click(await screen.findByRole('button', { name: '克隆为空间策略' }));

    await userEvent.click(screen.getByRole('button', { name: '取消' }));

    // 先等正向信号（抽屉关闭），再做消极断言
    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });
});

describe('发布策略管理：抽屉离开确认', () => {
  /** 关闭确认弹窗：不确认（用户点「取消」留在抽屉） */
  function keepOpen() {
    mocks.infoBox.mockImplementation(() => {});
  }

  it('当表单未改动时，点击「取消」应直接关闭且不弹确认', async () => {
    await openCreateDrawer();

    await userEvent.click(screen.getByRole('button', { name: '取消' }));

    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('当表单有改动时，点击「取消」应先弹确认；确认离开后才关闭', async () => {
    await openCreateDrawer();
    // 只输入名称即视为有改动，无需表单校验通过
    await userEvent.type(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'), '灰度');

    await userEvent.click(screen.getByRole('button', { name: '取消' }));

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    // 默认 mock 会直接 confirm → 关闭抽屉，且不产生请求
    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(harness.createReleaseStrategy).not.toHaveBeenCalled();
  });

  it('当表单有改动且用户在确认弹窗点「取消」时，抽屉保持打开', async () => {
    keepOpen();
    await openCreateDrawer();
    await userEvent.type(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'), '灰度');

    await userEvent.click(screen.getByRole('button', { name: '取消' }));

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId('strategy-sideslider-header')).toBeInTheDocument();
  });

  it('当表单有改动时，点击右上角 × 也应先弹确认', async () => {
    await openCreateDrawer();
    await userEvent.type(screen.getByPlaceholderText('请输入策略名称，1 ~ 32 个字符'), '灰度');

    const closeIcon = document.querySelector<HTMLElement>('.bk-sideslider-close');
    expect(closeIcon).not.toBeNull();
    await fireEvent.click(closeIcon as HTMLElement);

    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
  });

  it('当表单回填后未改动时，不应因回填而误判为有改动', async () => {
    await renderPage();
    await screen.findByText('生产谨慎发布');
    await userEvent.click(screen.getByRole('button', { name: '编辑' }));
    await screen.findByPlaceholderText('请输入策略名称，1 ~ 32 个字符');

    await userEvent.click(screen.getByRole('button', { name: '取消' }));

    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });

  it('查看态只读，点击「关闭」无需确认', async () => {
    await renderPage();
    await screen.findByText('分三批灰度');
    await userEvent.click(screen.getByRole('button', { name: '查看' }));
    await screen.findByRole('button', { name: '关闭' });

    await userEvent.click(screen.getByRole('button', { name: '关闭' }));

    await waitFor(() => expect(screen.queryByTestId('strategy-sideslider-header')).not.toBeInTheDocument());
    expect(mocks.infoBox).not.toHaveBeenCalled();
  });
});

describe('发布策略管理：删除与内置策略操作', () => {
  it('当用户删除策略并确认时，应调用 delete 并重新拉取列表', async () => {
    await renderPage();
    await screen.findByText('生产谨慎发布');
    await userEvent.click(screen.getByRole('button', { name: '删除' }));

    await waitFor(() => expect(harness.deleteReleaseStrategy).toHaveBeenCalledTimes(1));
    expect(harness.deleteReleaseStrategy).toHaveBeenCalledWith({ workspaceID: 'ws-1', strategyID: 'strategy-1' });
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'success', message: '策略已删除' });
    // 删除成功后重新拉取列表
    await waitFor(() => expect(harness.listReleaseStrategies).toHaveBeenCalledTimes(2));
  });

  it('当用户取消删除时，不应调用 delete', async () => {
    mocks.infoBox.mockImplementation(() => {});
    await renderPage();
    await screen.findByText('生产谨慎发布');
    await userEvent.click(screen.getByRole('button', { name: '删除' }));

    // 先等正向消费信号（InfoBox 已弹出），再做消极断言
    await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
    expect(harness.deleteReleaseStrategy).not.toHaveBeenCalled();
  });

  it('内置策略应仅展示「查看/克隆」，无「编辑/删除」', async () => {
    await renderPage();
    await screen.findByText('分三批灰度');

    const rows = screen.getAllByTestId('table-row');
    // 空间策略在前、内置策略在后
    const spaceRow = rows[0];
    const builtinRow = rows[1];
    expect(within(spaceRow).getByRole('button', { name: '编辑' })).toBeInTheDocument();
    expect(within(spaceRow).getByRole('button', { name: '删除' })).toBeInTheDocument();
    expect(within(builtinRow).getByRole('button', { name: '查看' })).toBeInTheDocument();
    expect(within(builtinRow).getByRole('button', { name: '克隆' })).toBeInTheDocument();
    expect(within(builtinRow).queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
    expect(within(builtinRow).queryByRole('button', { name: '删除' })).not.toBeInTheDocument();
  });
});
