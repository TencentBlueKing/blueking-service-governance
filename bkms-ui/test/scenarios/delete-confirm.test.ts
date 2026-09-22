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
 * 场景级测试：删除二次确认（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S6）
 *
 * 覆盖四组用户可感知行为：显示内容 / 确认 / 取消 / 删除进行中禁用取消 → V = 4
 * 连带说明：业务级删除入口（如 delete-app-dialog.vue）复用本组件，其交互随本场景覆盖。
 */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  confirm: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

const harness = vi.hoisted(() => ({
  loading: false,
}));

const DeleteConfirm = await import('~/components/delete-comfirm.vue').then(m => m.default);

/** 包装组件：承载 v-model:isShow 与 loading 动态值 */
const Harness = defineComponent({
  setup() {
    const isShow = ref(true);
    const loading = ref(harness.loading);
    return () =>
      h(
        DeleteConfirm as never,
        {
          isShow: isShow.value,
          'onUpdate:isShow': (v: boolean) => (isShow.value = v),
          loading: loading.value,
          title: '删除应用',
          onConfirm: mocks.confirm,
        } as never,
        { default: () => '删除后不可恢复' },
      );
  },
});

beforeEach(() => {
  vi.clearAllMocks();
  harness.loading = false;
});

describe('删除确认：二次确认流程', () => {
  it('当删除确认弹窗打开时，应展示标题与风险说明', async () => {
    render(Harness);
    expect(await screen.findByText('删除应用')).toBeInTheDocument();
    expect(screen.getByText('删除后不可恢复')).toBeInTheDocument();
  });

  it('当用户点击确定时，应触发删除且弹窗交由父组件决定是否关闭', async () => {
    render(Harness);
    await userEvent.click(await screen.findByRole('button', { name: '确定' }));
    expect(mocks.confirm).toHaveBeenCalledTimes(1);
    // 组件只 emit confirm，不自行关闭——避免父组件删除失败时弹窗已消失
    expect(screen.getByText('删除应用')).toBeInTheDocument();
  });

  it('当用户点击取消时，应关闭弹窗且不触发删除', async () => {
    render(Harness);
    await userEvent.click(await screen.findByRole('button', { name: '取消' }));
    await waitFor(() => expect(screen.queryByText('删除应用')).not.toBeInTheDocument());
    expect(mocks.confirm).not.toHaveBeenCalled();
  });

  it('当删除正在进行时，应禁用取消按钮避免中途关闭', async () => {
    harness.loading = true;
    render(Harness);
    expect(await screen.findByRole('button', { name: '取消' })).toBeDisabled();
  });
});
