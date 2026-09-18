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
 * 场景级测试：动态 / 重复输入项（S5 收尾：key-value）
 * 路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S5
 *
 * 覆盖三组用户可感知行为：初始渲染 / 新增一对键值 / 禁用态 → V = 3
 * （第 4 条路径「删除与最少行数约束」jsdom 下不可触发，见文件末注释）
 *
 * 说明：删除按钮是 bkui-vue 的 Del 图标（渲染为 svg 且无 role），只能通过容器查询定位，见用例注释。
 */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

const harness = vi.hoisted(() => ({
  value: [{ key: 'A', value: '1' }] as { key: string; value: string }[],
  minRows: 0 as number,
  disabled: false as boolean,
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

const KeyValue = await import('~/components/key-value.vue').then(m => m.default);

const Harness = defineComponent({
  setup() {
    const value = ref<{ key: string; value: string }[]>([...harness.value]);
    return () =>
      h(
        KeyValue as never,
        {
          modelValue: value.value,
          'onUpdate:modelValue': (v: { key: string; value: string }[]) => (value.value = v),
          minRows: harness.minRows,
          disabled: harness.disabled,
          keyPlaceholder: '键名',
        } as never,
      );
  },
});

beforeEach(() => {
  harness.value = [{ key: 'A', value: '1' }];
  harness.minRows = 0;
  harness.disabled = false;
});

describe('键值输入项：增删与约束', () => {
  it('当存在初始键值时应渲染对应的键输入框', () => {
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    expect(screen.getByPlaceholderText('键名')).toHaveValue('A');
  });

  it('当用户点击添加时，应新增一对键值输入', async () => {
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    await userEvent.click(screen.getByText('添加'));
    expect(screen.getAllByPlaceholderText('键名')).toHaveLength(2);
  });

  // 未覆盖：删除交互与「最少行数」约束。
  // 原因：删除按钮是图标按钮（无 role/name），Del 渲染为 svg，点击 svg 不会冒泡触发按钮 click，
  // jsdom 下无法触发删除；若写成「点击后行数不变」会退化成恒真的弱断言，故不写。
  // 原因：删除确认浮层在 jsdom 下不渲染，点击无效果；若写成「点击后行数不变」会退化成恒真弱断言，故不写。

  it('当组件被禁用时，键输入应不可编辑', () => {
    harness.disabled = true;
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    expect(screen.getByPlaceholderText('键名')).toBeDisabled();
  });
});
