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
 * 场景级测试：公共环境变量管理（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S16）
 *
 * 覆盖四组用户可感知行为（聚焦新建/编辑表单弹窗这一核心交互）：
 *   新建标题与可选作用域 / 编辑标题与作用域只读 / Key 格式非法被拦截 / 指定环境类型出现类型选择 → V = 4
 *
 * 说明：列表（Sideslider 内的变量表格）与删除确认另计，本场景先覆盖表单弹窗。
 */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

const harness = vi.hoisted(() => ({
  success: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1', async () => {
  const { createAnyServiceMock } = await import('../helpers/mock-service');
  return { EnvvarsService: createAnyServiceMock('EnvvarsService', {}) };
});

beforeEach(() => {
  vi.clearAllMocks();
});

async function renderDialog(editData: null | Record<string, unknown> = null) {
  const { default: EnvVarFormDialog } = await import('~/pages/env/public-env-vars/env-var-form-dialog.vue');
  return render(EnvVarFormDialog as never, {
    props: {
      isShow: true,
      workspaceId: 'ws-1',
      editData: editData as never,
    } as never,
    attrs: { 'onUpdate:isShow': () => {}, onSuccess: harness.success } as never,
    global: { mocks: i18nGlobalMocks } as never,
  });
}

describe('公共环境变量：新建与编辑表单', () => {
  it('当新建环境变量时，应展示新增标题并可选取作用域', async () => {
    await renderDialog();
    expect(await screen.findByText('新增环境变量')).toBeInTheDocument();
    expect(screen.getByText('所有')).toBeInTheDocument();
    expect(screen.getByText('指定环境类型')).toBeInTheDocument();
  });

  it('当编辑既有变量时，应展示编辑标题且作用域改为只读展示', async () => {
    await renderDialog({
      key: 'EXIST_KEY',
      value: 'v',
      isSensitive: false,
      scopeType: 'workspace',
      scopeValue: '',
      description: '',
    });
    expect(await screen.findByText('编辑环境变量')).toBeInTheDocument();
    expect(screen.queryByText('指定环境类型')).not.toBeInTheDocument();
  });

  it('当 Key 不符合命名规则时，提交应被拦截且不提示成功', async () => {
    await renderDialog();
    await userEvent.type(await screen.findByPlaceholderText('字母或下划线开头，仅允许字母、数字、下划线'), '1abc');
    await userEvent.click(screen.getByRole('button', { name: '确定' }));
    await waitFor(() => expect(screen.getByText('字母或下划线开头，仅允许字母、数字、下划线')).toBeInTheDocument());
    expect(harness.success).not.toHaveBeenCalled();
  });

  it('当用户选择指定环境类型时，应出现环境类型选项', async () => {
    await renderDialog();
    await userEvent.click(await screen.findByText('指定环境类型'));
    await waitFor(() => expect(screen.getByText('预发布')).toBeInTheDocument());
    expect(screen.getByText('生产')).toBeInTheDocument();
  });
});
