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
 * S12：组件新建/编辑向导（component-management.vue）场景级测试。
 *
 * 覆盖六组用户可感知行为（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S12）：
 * A 模式回显 / B 表单校验 / C 试运行流转 / D 提交成败 / E 未保存离开确认 / F 可引用变量面板。
 *
 * 测行为不测实现：断言用户可见的标题、按钮、提示与弹层状态；
 * API 层（ComponentDefsService）与子模板（输入/输出模板）全部 mock 隔离。
 */

import { defineComponent, h, ref } from 'vue';
import type { Component } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ComponentManagement from '~/pages/marketplace/component-management.vue';

import { MockedApiError } from '../helpers/mocked-api-error';

// ── mock 与桩（vi.hoisted：vi.mock 工厂可引用） ──

const mocks = vi.hoisted(() => ({
  previewComponentDef: vi.fn(),
  createComponentDef: vi.fn(),
  patchComponentDef: vi.fn(),
  getComponentDefsBuiltinVars: vi.fn(),
  message: vi.fn(),
  infoBox: vi.fn(),
  /** 校验方法以 vi.fn 暴露：用例可断言「validateForm 已真实消费校验结果」（先于消极断言，防首检即过） */
  inputIsValid: vi.fn(() => Promise.resolve(mocks.inputValid.current)),
  outputIsValid: vi.fn(() => mocks.outputValid.current),
  /** 输入模板子组件的动态返回值（快照/脏检查/校验链共用） */
  inputValue: { current: [] as Array<Record<string, unknown>> },
  outputValue: { current: { patchers: [] as unknown[], specs: [] as unknown[] } },
  inputValid: { current: true },
  outputValid: { current: true },
}));

vi.mock('~/api/modules/v1', () => ({
  ComponentDefsService: {
    previewComponentDef: mocks.previewComponentDef,
    createComponentDef: mocks.createComponentDef,
    patchComponentDef: mocks.patchComponentDef,
    getComponentDefsBuiltinVars: mocks.getComponentDefsBuiltinVars,
  },
}));

vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-under-test' }),
}));

// InfoBox / Message 是命令式 API，mock 后由用例控制弹窗与断言调用参数；
// 其余 bkui-vue 组件保持真实渲染（本场景的核心验证目标之一）。
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
  InfoBox: mocks.infoBox,
}));

import { i18nPlugin } from '../helpers/mock-i18n';

// i18n 不在测试范围：useI18n 直译、$t 直译
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

// 输入/输出模板桩：仅提供被测组件依赖的契约方法（getValue/isValid/getFormData/resetData/getOutputData）
vi.mock('~/pages/marketplace/components/component-input/component-input-template.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'InputTemplateStub',
      props: { initialData: { type: Array, default: () => [] } },
      methods: {
        getValue: () => mocks.inputValue.current,
        getFormData: () => mocks.inputValue.current,
        // 真实契约（component-input-template.vue:172）为 async isValid(): Promise<boolean>
        // 注意：stub 返回值非响应式——面板用例依赖"挂载前赋值"；挂载后改值不会触发组件重算
        isValid: mocks.inputIsValid,
        resetData: () => {},
      },
      template: '<div data-testid="input-template">InputTemplate</div>',
    }),
  };
});

vi.mock('~/pages/marketplace/components/component-ouput/component-output-template.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'OutputTemplateStub',
      methods: {
        getOutputData: () => mocks.outputValue.current,
        // 真实契约（component-output-template.vue:195）为同步返回 boolean，勿写成 Promise
        isValid: mocks.outputIsValid,
      },
      template: '<div data-testid="output-template"><slot name="header-right" />OutputTemplate</div>',
    }),
  };
});

vi.mock('~/pages/marketplace/components/ref-var-panel.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'RefVarPanelStub',
      props: { inputVariableNames: { type: Array, default: () => [] } },
      emits: ['close'],
      template:
        '<div data-testid="ref-var-panel">' +
        '<span data-testid="ref-var-names">{{ inputVariableNames.map(v => v.name).join(",") }}</span>' +
        '<button data-testid="ref-var-close" @click="$emit(\'close\')">CloseRefVar</button>' +
        '</div>',
    }),
  };
});

vi.mock('~/pages/marketplace/components/component-ouput/patch-preview-card.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'PatchPreviewCardStub',
      props: { targetKind: { type: String, default: '' } },
      template: '<div data-testid="patch-preview">{{ targetKind }}</div>',
    }),
  };
});

vi.mock('~/pages/marketplace/components/component-ouput/resource-card.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'ResourceCardStub',
      props: { kind: { type: String, default: '' } },
      template: '<div data-testid="resource-card">{{ kind }}</div>',
    }),
  };
});

// 可折叠布局桩：保留 main/aside 插槽与 v-model:is-collapsed 的显隐契约
vi.mock('~/components/collapsible-aside-layout.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'CollapsibleAsideLayoutStub',
      props: { isCollapsed: { type: Boolean, default: true } },
      template:
        '<div data-testid="aside-layout">' +
        '<div><slot name="main" /></div>' +
        '<div v-if="!isCollapsed" data-testid="aside-panel"><slot name="aside" /></div>' +
        '</div>',
    }),
  };
});

vi.mock('~/components/toggle-card.vue', async () => {
  const { defineComponent: dc } = await import('vue');
  return {
    default: dc({
      name: 'ToggleCardStub',
      template: '<div data-testid="toggle-card"><slot /></div>',
    }),
  };
});

// ── Harness：通过 defineExpose 的 open/close 驱动弹层，并捕获 refresh 事件 ──

const harness = vi.hoisted(() => ({
  open: undefined as ((type: 'component' | 'edit', data?: Record<string, unknown>) => void) | undefined,
  close: undefined as (() => void) | undefined,
  refreshSpy: undefined as ReturnType<typeof vi.fn> | undefined,
}));

const Harness = defineComponent({
  name: 'ComponentManagementHarness',
  setup() {
    const cmRef = ref();
    const refreshSpy = vi.fn();
    harness.open = (type, data) => cmRef.value?.open(type, data);
    harness.close = () => cmRef.value?.close();
    harness.refreshSpy = refreshSpy;
    return () =>
      h(ComponentManagement as Component, {
        ref: cmRef,
        onRefresh: () => refreshSpy(),
      });
  },
});

const NAME_PLACEHOLDER = '请输入 2-20 字符的字母、数字、中划线，以字母开头';
const EDIT_COMPONENT = {
  name: 'exist-comp',
  displayName: '已有组件',
  scopeType: 'global',
  description: '原始描述',
  properties: [],
  patchers: [],
  specs: [],
};

async function fillName(value: string) {
  const input = screen.getByPlaceholderText(NAME_PLACEHOLDER);
  await userEvent.clear(input);
  if (value) {
    await userEvent.type(input, value);
  }
}

/** 走到步骤 2（合法表单 + 试运行成功）；编辑模式组件 ID 禁用且已回显合法值，无需填写 */
async function gotoStep2(mode: 'component' | 'edit' = 'component') {
  if (mode === 'component') {
    await fillName('ab');
  }
  await userEvent.click(screen.getByRole('button', { name: '试运行' }));
  await screen.findByRole('button', { name: '提交' });
}

/** 打开弹层并等待步骤 1 内容（试运行按钮）就绪；额外等待脏检查快照完成 */
async function openWizard(mode: 'component' | 'edit' = 'component') {
  render(Harness, { global: { plugins: [createPinia(), i18nPlugin] } });
  harness.open?.(mode, mode === 'edit' ? EDIT_COMPONENT : undefined);
  await screen.findByRole('button', { name: '试运行' });
  // 快照 watch（flush post + nextTick）需两个微任务后才完成，避免"未修改却弹确认"的时序误判
  await waitFor(() => expect(screen.getByTestId('input-template')).toBeInTheDocument());
  await new Promise(resolve => setTimeout(resolve, 20));
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.inputValue.current = [];
  mocks.outputValue.current = { patchers: [], specs: [] };
  mocks.inputValid.current = true;
  mocks.outputValid.current = true;
  mocks.getComponentDefsBuiltinVars.mockResolvedValue([]);
  mocks.previewComponentDef.mockResolvedValue({
    patchPreview: [{ baseManifest: 'base', patchedManifest: 'patched', targetKind: 'Deployment' }],
    resources: [{ kind: 'Service', manifest: 'manifest', name: 'svc', apiVersion: 'v1' }],
  });
  mocks.createComponentDef.mockResolvedValue({});
  mocks.patchComponentDef.mockResolvedValue({});
  mocks.infoBox.mockImplementation(() => undefined);
});

describe('组件管理：新建/编辑向导', () => {
  describe('打开与模式回显', () => {
    it('当用户新建组件时，应显示"新建组件"标题、组件 ID 可输入且使用范围默认"仅本空间可见"', async () => {
      await openWizard('component');

      expect(screen.getByText('新建组件')).toBeInTheDocument();
      const nameInput = screen.getByPlaceholderText(NAME_PLACEHOLDER);
      expect(nameInput).toBeEnabled();
      expect(screen.getByRole('radio', { name: '仅本空间可见' })).toBeChecked();
      expect(screen.getByRole('radio', { name: '所有空间可见' })).not.toBeChecked();
    });

    it('当用户编辑组件时，应显示"编辑组件"及组件名、组件 ID 不可修改且回显使用范围与描述', async () => {
      await openWizard('edit');

      expect(screen.getByText('编辑组件')).toBeInTheDocument();
      expect(screen.getByText('已有组件')).toBeInTheDocument();
      const nameInput = screen.getByPlaceholderText(NAME_PLACEHOLDER);
      expect(nameInput).toBeDisabled();
      expect(nameInput).toHaveValue('exist-comp');
      expect(screen.getByRole('radio', { name: '所有空间可见' })).toBeChecked();
      expect(screen.getByDisplayValue('原始描述')).toBeInTheDocument();
    });
  });

  describe('表单校验', () => {
    it('当组件 ID 为空时，试运行应被拦截并提示"请输入组件ID"，不调用预览接口', async () => {
      await openWizard('component');
      await fillName('');

      await userEvent.click(screen.getByRole('button', { name: '试运行' }));

      // 空值时 bkui-vue 依次执行 required 与自定义 rules，最终展示的错误文案为 rules 末条的格式提示
      await waitFor(() =>
        expect(screen.getByText(/以字母开头，可包含字母、数字、中划线，长度2-20位/)).toBeInTheDocument(),
      );
      // 校验文案（正向信号）就位后再做消极断言，避免 waitFor(not) 空转即过的假绿
      await waitFor(() => expect(mocks.previewComponentDef).not.toHaveBeenCalled());
      expect(screen.queryByText('试运行结果预览')).not.toBeInTheDocument();
    });

    it.each([
      ['1abc', '数字开头'],
      ['ab#cd', '含非法字符'],
      ['ab-', '以中划线结尾'],
    ])('当组件 ID %s（%s）不满足命名规则时，应显示格式错误提示且不进入预览', async (value, _label) => {
      await openWizard('component');
      await fillName(value);

      await userEvent.click(screen.getByRole('button', { name: '试运行' }));

      await waitFor(() => expect(mocks.previewComponentDef).not.toHaveBeenCalled());
      expect(screen.getByText(/以字母开头，可包含字母、数字、中划线，长度2-20位/)).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();
    });

    it('当组件 ID 为边界长度（2 位/20 位）时应通过校验进入预览，1 位/21 位应被拦截', async () => {
      await openWizard('component');
      await fillName('ab');
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await screen.findByRole('button', { name: '提交' });
      expect(mocks.previewComponentDef).toHaveBeenCalledTimes(1);

      await userEvent.click(screen.getByRole('button', { name: '上一步' }));
      await fillName('a'.repeat(19) + '0');
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await screen.findByRole('button', { name: '提交' });
      expect(mocks.previewComponentDef).toHaveBeenCalledTimes(2);

      await userEvent.click(screen.getByRole('button', { name: '上一步' }));
      await fillName('a');
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await waitFor(() => expect(mocks.previewComponentDef).toHaveBeenCalledTimes(2));
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();

      await fillName('a'.repeat(20) + '0');
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await waitFor(() => expect(mocks.previewComponentDef).toHaveBeenCalledTimes(2));
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();
    });

    it('当输入或输出模板校验失败时，试运行应被拦截且不调用预览接口', async () => {
      await openWizard('component');
      await fillName('ab');

      // 消极断言防"首检即过"：先等校验方法被真实消费（validateForm 已执行到消费校验结果，
      // 校验失败时按钮不进 loading，无 DOM 信号可等），再断言预览未被调用——
      // 若回归 bug 忽略子模板校验直接调 preview，调用晚于消费点，会被 waitFor 轮询捕获
      mocks.inputValid.current = false;
      mocks.inputIsValid.mockClear();
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await waitFor(() => expect(mocks.inputIsValid).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(mocks.previewComponentDef).not.toHaveBeenCalled());
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();

      mocks.inputValid.current = true;
      mocks.outputValid.current = false;
      mocks.outputIsValid.mockClear();
      await userEvent.click(screen.getByRole('button', { name: '试运行' }));
      await waitFor(() => expect(mocks.outputIsValid).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(mocks.previewComponentDef).not.toHaveBeenCalled());
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();
    });
  });

  describe('试运行流转', () => {
    it('当用户填写合法表单并点击试运行时，应调用预览接口并进入步骤 2 展示试运行结果', async () => {
      await openWizard('component');
      await gotoStep2();

      expect(screen.getByText('试运行结果预览')).toBeInTheDocument();
      expect(screen.getByTestId('patch-preview')).toHaveTextContent('Deployment');
      expect(screen.getByTestId('resource-card')).toHaveTextContent('Service');
      expect(mocks.previewComponentDef).toHaveBeenCalledWith(
        expect.objectContaining({ compDefName: 'ab' }),
        expect.objectContaining({ needRes: true }),
      );
    });

    it('当用户在步骤 2 点击上一步时，应回到表单填写且已填内容保留', async () => {
      await openWizard('component');
      await gotoStep2();

      await userEvent.click(screen.getByRole('button', { name: '上一步' }));

      await screen.findByRole('button', { name: '试运行' });
      expect(screen.getByPlaceholderText(NAME_PLACEHOLDER)).toHaveValue('ab');
      expect(screen.queryByText('试运行结果预览')).not.toBeInTheDocument();
    });

    it('当试运行接口失败时，应停留在表单步骤且可重新发起试运行（错误反馈由请求拦截器统一处理）', async () => {
      mocks.previewComponentDef.mockRejectedValue(new MockedApiError());
      await openWizard('component');
      await fillName('ab');

      await userEvent.click(screen.getByRole('button', { name: '试运行' }));

      await waitFor(() => expect(mocks.previewComponentDef).toHaveBeenCalledTimes(1));
      expect(screen.queryByRole('button', { name: '提交' })).not.toBeInTheDocument();
      await waitFor(() => expect(screen.getByRole('button', { name: '试运行' })).toBeEnabled());
      expect(screen.queryByText('试运行结果预览')).not.toBeInTheDocument();
    });
  });

  describe('提交', () => {
    it('当用户新建提交成功时，应提示"组件新建成功"、关闭弹层并刷新列表', async () => {
      await openWizard('component');
      await gotoStep2();

      await userEvent.click(screen.getByRole('button', { name: '提交' }));

      await waitFor(() =>
        expect(mocks.message).toHaveBeenCalledWith(
          expect.objectContaining({ theme: 'success', message: '组件新建成功' }),
        ),
      );
      expect(mocks.createComponentDef).toHaveBeenCalledWith(
        expect.objectContaining({
          compDefName: 'ab',
          scopeType: 'workspace',
          scopeWorkspaceIDs: ['ws-under-test'],
        }),
      );
      expect(harness.refreshSpy).toHaveBeenCalledTimes(1);
      await waitFor(() => expect(screen.queryByRole('button', { name: '试运行' })).not.toBeInTheDocument());
    });

    it('当用户新建提交失败时，应提示"组件新建失败"且弹层保持打开', async () => {
      mocks.createComponentDef.mockRejectedValue(new Error('server error'));
      await openWizard('component');
      await gotoStep2();

      await userEvent.click(screen.getByRole('button', { name: '提交' }));

      await waitFor(() =>
        expect(mocks.message).toHaveBeenCalledWith(
          expect.objectContaining({ theme: 'danger', message: '组件新建失败' }),
        ),
      );
      expect(screen.getByRole('button', { name: '上一步' })).toBeInTheDocument();
      expect(harness.refreshSpy).not.toHaveBeenCalled();
    });

    it('当用户编辑提交成功时，应提示"组件更新成功"、关闭弹层并刷新列表', async () => {
      await openWizard('edit');
      await gotoStep2('edit');

      await userEvent.click(screen.getByRole('button', { name: '提交' }));

      await waitFor(() =>
        expect(mocks.message).toHaveBeenCalledWith(
          expect.objectContaining({ theme: 'success', message: '组件更新成功' }),
        ),
      );
      expect(mocks.patchComponentDef).toHaveBeenCalledWith(
        expect.objectContaining({
          compDefName: 'exist-comp',
          scopeType: 'global',
          scopeWorkspaceIDs: [],
        }),
      );
      expect(mocks.createComponentDef).not.toHaveBeenCalled();
      expect(harness.refreshSpy).toHaveBeenCalledTimes(1);
      await waitFor(() => expect(screen.queryByRole('button', { name: '上一步' })).not.toBeInTheDocument());
    });

    it('当用户编辑提交失败时，应提示"组件更新失败"且弹层保持打开', async () => {
      mocks.patchComponentDef.mockRejectedValue(new Error('server error'));
      await openWizard('edit');
      await gotoStep2('edit');

      await userEvent.click(screen.getByRole('button', { name: '提交' }));

      await waitFor(() =>
        expect(mocks.message).toHaveBeenCalledWith(
          expect.objectContaining({ theme: 'danger', message: '组件更新失败' }),
        ),
      );
      expect(screen.getByRole('button', { name: '上一步' })).toBeInTheDocument();
      expect(harness.refreshSpy).not.toHaveBeenCalled();
    });
  });

  describe('未保存离开确认', () => {
    it('当用户未修改任何内容时，点击取消应直接关闭弹层且不弹确认', async () => {
      await openWizard('component');

      await userEvent.click(screen.getByRole('button', { name: '取消' }));

      await waitFor(() => expect(screen.queryByRole('button', { name: '试运行' })).not.toBeInTheDocument());
      expect(mocks.infoBox).not.toHaveBeenCalled();
    });

    it('当用户已填写内容后点击取消时，应先弹出离开确认', async () => {
      await openWizard('component');
      await fillName('ab');

      await userEvent.click(screen.getByRole('button', { name: '取消' }));

      await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));
      expect(mocks.infoBox).toHaveBeenCalledWith(expect.objectContaining({ title: '确认离开当前页？' }));
      expect(screen.getByRole('button', { name: '试运行' })).toBeInTheDocument();
    });

    it('当用户在离开确认中放弃离开时，应保留弹层与已填内容', async () => {
      await openWizard('component');
      await fillName('ab');
      await userEvent.click(screen.getByRole('button', { name: '取消' }));
      await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));

      const infoBoxOptions = mocks.infoBox.mock.calls[0][0] as { onCancel: () => void };
      infoBoxOptions.onCancel();

      await waitFor(() => expect(screen.getByRole('button', { name: '试运行' })).toBeInTheDocument());
      expect(screen.getByPlaceholderText(NAME_PLACEHOLDER)).toHaveValue('ab');
    });

    it('当用户在离开确认中确认离开时，应关闭弹层且不刷新列表', async () => {
      await openWizard('component');
      await fillName('ab');
      await userEvent.click(screen.getByRole('button', { name: '取消' }));
      await waitFor(() => expect(mocks.infoBox).toHaveBeenCalledTimes(1));

      const infoBoxOptions = mocks.infoBox.mock.calls[0][0] as { onConfirm: () => void };
      infoBoxOptions.onConfirm();

      await waitFor(() => expect(screen.queryByRole('button', { name: '试运行' })).not.toBeInTheDocument());
      expect(harness.refreshSpy).not.toHaveBeenCalled();
    });
  });

  describe('可引用变量面板', () => {
    it('当用户点击"可引用变量"时，应展开面板并展示已输入变量的名称列表', async () => {
      mocks.inputValue.current = [{ name: 'cluster', description: '目标集群' }];
      await openWizard('component');

      await userEvent.click(screen.getByRole('button', { name: '可引用变量' }));

      await waitFor(() => expect(screen.getByTestId('aside-panel')).toBeInTheDocument());
      expect(screen.getByTestId('ref-var-names')).toHaveTextContent('cluster');
    });

    it('当用户关闭面板时，面板收起且弹层恢复默认宽度', async () => {
      await openWizard('component');
      await userEvent.click(screen.getByRole('button', { name: '可引用变量' }));
      await waitFor(() => expect(screen.getByTestId('aside-panel')).toBeInTheDocument());

      await userEvent.click(screen.getByTestId('ref-var-close'));

      await waitFor(() => expect(screen.queryByTestId('aside-panel')).not.toBeInTheDocument());
      expect(screen.getByRole('button', { name: '试运行' })).toBeInTheDocument();
    });
  });
});
