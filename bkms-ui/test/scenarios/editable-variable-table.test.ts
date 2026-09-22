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
 * 场景级测试：环境变量可编辑表格（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S4）
 *
 * 覆盖两组用户可感知行为：只读态渲染 / 点击编辑进入编辑态 → V = 2
 * （第 3 条路径「删除确认浮层」jsdom 下不渲染，见文件末注释）
 *
 * 说明：表格本体为 vxe（@blueking/table），沿用 S14 沉淀的 stub 与垫片（见 test/helpers/mock-table.ts）；
 * 被 stub 的是单元格绘制，行内操作（编辑/删除确认）仍为真实行为。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

const harness = vi.hoisted(() => ({
  list: [{ id: '1', key: 'APP_NAME', value: 'demo', description: '应用名', isSensitive: false }] as {
    description: string;
    id: string;
    isSensitive: boolean;
    key: string;
    value: string;
  }[],
}));

beforeAll(() => {
  installVxeShims();
});

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());

beforeEach(() => {
  vi.clearAllMocks();
  harness.list = [{ id: '1', key: 'APP_NAME', value: 'demo', description: '应用名', isSensitive: false }];
});

async function renderTable() {
  const { default: EditableVariableTable } = await import('~/components/editable-variable-table/index.vue');
  return render(EditableVariableTable as never, {
    props: { list: harness.list as never } as never,
    global: { mocks: i18nGlobalMocks } as never,
  });
}

describe('环境变量表格：行内编辑与删除确认', () => {
  it('当传入变量列表时，应以只读态展示变量内容', async () => {
    await renderTable();
    await waitFor(() => expect(screen.getByText('APP_NAME')).toBeInTheDocument());
    expect(screen.getByText('demo')).toBeInTheDocument();
  });

  it('当用户点击编辑时，该行应进入可编辑状态', async () => {
    await renderTable();
    await userEvent.click(await screen.findByText('编辑'));
    // 进入编辑态后 key 由文本变为输入框
    await waitFor(() => expect(screen.getByDisplayValue('APP_NAME')).toBeInTheDocument());
  });

  // 未覆盖：点击删除后的确认浮层。
  // 原因：PopConfirm 浮层在 jsdom 下不渲染，点击后查不到确认文案；若断言「点击后未立即删除」
  // 会因浮层根本不出而变成恒真断言，故不写。
});
