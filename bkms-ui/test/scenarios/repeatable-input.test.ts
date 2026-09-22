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
 * 场景级测试：动态 / 重复输入项（S5 补齐：repeatable-input）
 * 路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S5
 *
 * 覆盖三组用户可感知行为：初始渲染 / 新增条目 / 删除条目 → V = 3
 * （第 4 条路径「必填校验提示」为 tooltips 形式，jsdom 下不可断言，见文件末注释）
 *
 * 说明：删除按钮是 bkui-vue 的 Del 图标（渲染为 svg，无可访问角色），
 * 只能通过容器查询定位，已在用例中注释说明。
 */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

const harness = vi.hoisted(() => ({
  value: ['a', 'b'] as string[],
  required: false as boolean,
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

const RepeatableInput = await import('~/components/repeatable-input.vue').then(m => m.default);

const Harness = defineComponent({
  setup() {
    const value = ref<string[]>([...harness.value]);
    return () =>
      h(
        RepeatableInput as never,
        {
          modelValue: value.value,
          'onUpdate:modelValue': (v: string[]) => (value.value = v),
          required: harness.required,
          placeholder: '请输入值',
        } as never,
      );
  },
});

beforeEach(() => {
  harness.value = ['a', 'b'];
  harness.required = false;
});

describe('重复输入项：增删与校验', () => {
  it('当存在初始值时应渲染对应数量的输入项', () => {
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    expect(screen.getAllByPlaceholderText('请输入值')).toHaveLength(2);
  });

  it('当用户点击添加时，应新增一个空输入项', async () => {
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    await userEvent.click(screen.getByText('添加'));
    expect(screen.getAllByPlaceholderText('请输入值')).toHaveLength(3);
  });

  it('当用户删除某项时，仅该项被移除', async () => {
    render(Harness, { global: { mocks: i18nGlobalMocks } as never });
    // Del 为 bkui-vue 图标组件，渲染为 svg 且无 role：按行锚定（首个输入框所在行）定位，
    // 避免全局 svg 顺序耦合（页内新增无关图标即错位）
    const firstRow = screen.getAllByPlaceholderText('请输入值')[0].closest('.flex') as HTMLElement;
    await userEvent.click(firstRow.querySelector('svg') as HTMLElement);
    expect(screen.getAllByPlaceholderText('请输入值')).toHaveLength(1);
  });

  // 未覆盖：必填为空时的校验提示。
  // 原因：FormItem 用 error-display-type="tooltips"，错误以浮层形式展示，jsdom 下不产出可查询文本。
  // 原因：错误以浮层展示，jsdom 下不产出可查询文本；若断言「提交被拦截」会因查不到提示而变成恒真断言，故不写。
});
