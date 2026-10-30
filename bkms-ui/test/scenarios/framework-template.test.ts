import { defineComponent, h, onMounted, onUpdated, ref } from 'vue';

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
 * 场景级测试：空间框架配置模板（新模块，未纳入历史 ROUTES 台账）
 * 路径清单：V = 6
 * 1. 打开新建模板侧滑表单；2. 空表单确认显示校验；3. 新建和编辑时文件名不可修改；
 * 4. 切换 TAF/C++；5. 完整填写后确认新增；6. 默认模板切换时同语言旧默认失效。
 * Stub 边界：bkui-vue 的 Input/Select/Checkbox 换为可交互 stub（全局 stub 为纯 slot），MsEditor mock 为 textarea，Sideslider/Radio/Form 使用全局结构 stub。
 * Backlog：真实 API 接入后补充接口失败、权限和持久化刷新路径。
 */
/* eslint-disable vue/one-component-per-file */
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const apiMocks = vi.hoisted(() => ({
  create: vi.fn().mockResolvedValue({}),
  get: vi.fn().mockResolvedValue({}),
  update: vi.fn().mockResolvedValue({}),
}));

vi.mock('~/api/modules/v1', () => ({
  FrameworkConfigTemplatesService: {
    createWorkspaceFrameworkConfigTemplate: apiMocks.create,
    getWorkspaceFrameworkConfigTemplate: apiMocks.get,
    updateWorkspaceFrameworkConfigTemplate: apiMocks.update,
  },
}));

vi.mock('vue-i18n', async importOriginal => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    useI18n: () => ({ t: (key: string) => key }),
  };
});

vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'test-space' }),
}));

vi.mock('~/components/divider-header.vue', () => ({
  default: defineComponent({
    name: 'DividerHeaderMock',
    setup(_, { slots }) {
      return () => h('div', [slots.title?.(), slots.default?.()]);
    },
  }),
}));

vi.mock('~/components/monaco-editor/ms-editor.vue', () => ({
  default: defineComponent({
    name: 'MsEditorMock',
    props: {
      modelValue: { type: String, default: '' },
      lang: { type: String, default: 'yaml' },
      title: { type: String, default: '' },
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('textarea', {
          'data-testid': 'ms-editor',
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
        });
    },
  }),
}));

vi.mock('bkui-vue', async importOriginal => {
  const actual = await importOriginal<Record<string, unknown>>();
  const inputStub = defineComponent({
    name: 'InputInteractiveStub',
    props: {
      disabled: { type: Boolean, default: false },
      modelValue: { type: String, default: '' },
      placeholder: { type: String, default: '' },
      type: { type: String, default: 'text' },
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('input', {
          disabled: props.disabled,
          placeholder: props.placeholder,
          type: props.type,
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        });
    },
  });
  const selectOptionStub = defineComponent({
    name: 'SelectOptionInteractiveStub',
    props: { label: { type: String, default: '' }, value: { type: [String, Number], required: true } },
    setup(props) {
      return () => h('option', { value: props.value }, props.label);
    },
  });
  const selectStub = defineComponent({
    name: 'SelectInteractiveStub',
    props: { modelValue: { type: [String, Number], default: '' } },
    emits: ['update:modelValue'],
    setup(props, { emit, slots }) {
      const selectRef = ref<HTMLSelectElement>();
      function syncValue() {
        if (selectRef.value) selectRef.value.value = String(props.modelValue);
      }
      onMounted(syncValue);
      onUpdated(syncValue);
      return () =>
        h(
          'select',
          {
            onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
            ref: selectRef,
          },
          slots.default?.(),
        );
    },
  });
  const checkboxStub = defineComponent({
    name: 'CheckboxInteractiveStub',
    props: { modelValue: { type: Boolean, default: false } },
    emits: ['update:modelValue'],
    setup(props, { emit, slots }) {
      return () =>
        h('label', [
          h('input', {
            checked: props.modelValue,
            type: 'checkbox',
            onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
          }),
          slots.default?.(),
        ]);
    },
  });
  const formStub = defineComponent({
    name: 'FormInteractiveStub',
    setup(_, { expose, slots }) {
      expose({ clearValidate: () => undefined, validate: async () => true });
      return () => h('form', slots.default?.());
    },
  });
  const formItem = (actual.Form as { FormItem: unknown }).FormItem;
  return {
    ...actual,
    Checkbox: checkboxStub,
    Form: Object.assign(formStub, { FormItem: formItem }),
    Message: () => undefined,
    Input: inputStub,
    Select: Object.assign(selectStub, { Option: selectOptionStub }),
  };
});

import FrameworkTemplateEditor from '~/pages/basic/framework-templates/template-editor.vue';

const editorProps = {
  appType: 'trpc' as const,
  isShow: true,
  scene: 'create' as const,
  template: null,
  templates: [],
};

const editProps = {
  appType: 'trpc' as const,
  isShow: true,
  scene: 'edit' as const,
  template: {
    appType: 'trpc' as const,
    createdAt: '',
    fileContent: 'global: {}',
    fileName: 'trpc_go.yaml',
    id: 'template-1',
    isDefault: false,
    language: 'go' as const,
    name: '模板 1',
    source: 'workspace' as const,
  },
  templates: [],
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('框架配置模板：新建与默认模板', () => {
  it('打开表单后，空内容确认应显示必填校验', async () => {
    render(FrameworkTemplateEditor, { props: editorProps });

    await userEvent.click(screen.getByText('确定'));

    expect(screen.getByText('请输入配置文件内容')).toBeInTheDocument();
  });

  it('新建和编辑时均不允许修改配置文件名', async () => {
    const createView = render(FrameworkTemplateEditor, { props: editorProps });
    await waitFor(() => expect(createView.getByDisplayValue('trpc_go.yaml')).toBeDisabled());
    createView.unmount();

    render(FrameworkTemplateEditor, { props: editProps });
    await waitFor(() => expect(screen.getByDisplayValue('trpc_go.yaml')).toBeDisabled());
  });

  it('切换到 TAF 时应隐藏语言选择并切换配置文件默认名', async () => {
    render(FrameworkTemplateEditor, { props: editorProps });

    await userEvent.click(screen.getAllByRole('radio')[1]);

    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.getByDisplayValue('config.conf')).toBeInTheDocument();
    expect(screen.getByDisplayValue('config.conf')).toBeDisabled();
  });

  it('完整填写 tRPC 模板后确认应提交模板数据', async () => {
    render(FrameworkTemplateEditor, { props: editorProps });

    await userEvent.type(screen.getByPlaceholderText('请输入模板名称'), 'tRPC 基础模板');
    await userEvent.type(screen.getByTestId('ms-editor'), 'global:\n  namespace: Development');
    await userEvent.click(screen.getByText('确定'));

    expect(apiMocks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        content: 'global:\n  namespace: Development',
        fileName: 'trpc_go.yaml',
        framework: 'trpc',
        language: 'go',
        name: 'tRPC 基础模板',
      }),
    );
  });

  it('完整填写 TAF 模板后确认应提交 TAF 配置文件名', async () => {
    render(FrameworkTemplateEditor, { props: editorProps });

    await userEvent.click(screen.getAllByRole('radio')[1]);
    await userEvent.type(screen.getByPlaceholderText('请输入模板名称'), 'TAF 基础模板');
    await userEvent.type(screen.getByTestId('ms-editor'), '<taf />');
    await userEvent.click(screen.getByText('确定'));

    expect(apiMocks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '<taf />',
        fileName: 'config.conf',
        framework: 'taf',
        language: 'cpp',
        name: 'TAF 基础模板',
      }),
    );
  });
});
