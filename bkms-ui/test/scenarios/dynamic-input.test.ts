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
 * 场景级测试：动态输入项（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S5）
 *
 * 覆盖五组用户可感知行为：字符串回传 / 数字框 / 布尔单选 / 文本域 / 禁用态 → V = 5
 * 说明：MAP 类型依赖 KeyValue 子组件（动态键值表格），其交互另计，本场景不覆盖。
 */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';


const harness = vi.hoisted(() => ({
  type: 'STRING' as string,
  value: '' as unknown,
  disabled: false as boolean,
  emitted: undefined as unknown,
}));

const DynamicInput = await import('~/components/dynamic-input.vue').then(m => m.default);

/** 包装组件：承载 v-model 与动态 type / disabled */
const Harness = defineComponent({
  setup() {
    const value = ref(harness.value);
    return () =>
      h(
        DynamicInput as never,
        {
          modelValue: value.value,
          'onUpdate:modelValue': (v: unknown) => {
            value.value = v as never;
            harness.emitted = v;
          },
          type: harness.type,
          disabled: harness.disabled,
          placeholder: '请输入值',
        } as never,
      );
  },
});

beforeEach(() => {
  harness.type = 'STRING';
  harness.value = '';
  harness.disabled = false;
  harness.emitted = undefined;
});

describe('动态输入项：按类型分发与回传', () => {
  it('当类型为字符串时，用户输入的值应回传给父组件', async () => {
    render(Harness);
    await userEvent.type(screen.getByPlaceholderText('请输入值'), 'hello');
    expect(harness.emitted).toBe('hello');
  });

  it('当类型为数字时，应提供数字输入形态并回传用户输入', async () => {
    harness.type = 'INT';
    render(Harness);
    const input = screen.getByPlaceholderText('请输入值');
    // 数字形态（数字键盘 / 浏览器数字校验）是用户可感知差异；组件当前回传字符串（'12' 而非 12）。
    // 已复核：下游无隐患——接口类型 properties 为 Record<string, unknown>（app.d.ts:1020），
    // 消费方定义的值类型为 boolean | number | string（components-sideSlider.vue:606），
    // 且模块内无任何数值转换（Number/parseInt/parseFloat），故按现状断言。
    expect(input).toHaveAttribute('type', 'number');
    await userEvent.type(input, '12');
    expect(harness.emitted).toBe('12');
  });

  it('当类型为布尔时，用户可选择 false 并回传', async () => {
    harness.type = 'BOOL';
    render(Harness);
    await userEvent.click(screen.getByRole('radio', { name: 'false' }));
    expect(harness.emitted).toBe(false);
  });

  it('当类型为长文本时，应渲染多行文本域', async () => {
    harness.type = 'TEXT';
    render(Harness);
    expect(screen.getByPlaceholderText('请输入值').tagName).toBe('TEXTAREA');
  });

  it('当输入项被禁用时，用户无法编辑', async () => {
    harness.disabled = true;
    render(Harness);
    expect(screen.getByPlaceholderText('请输入值')).toBeDisabled();
  });
});
