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
 * 场景级测试：创建 tRPC 应用的配置模板选择（ROUTES 未单独收录）
 *
 * V = 10（判定节点：默认模板选择、无默认模板回退、内置模板排序、TAF 参数、切换确认、取消/详情失败回滚、旧详情请求隔离、恢复场景不覆盖、恢复模板已删除回退）
 * 路径：① 有默认模板时优先默认且内置排前 ② 无默认模板时使用排序后的首项
 * ③ TAF 使用默认 Select trigger ④ 确认切换后带入新内容 ⑤ 取消切换恢复旧选项和内容
 * ⑥ 新详情加载失败恢复旧选项和内容 ⑦ 重新加载后忽略迟到的旧详情
 * ⑧ 无远端模板时通知宿主隐藏选择区域 ⑨ 外部传入已选模板 id 时仅回显不覆盖
 * ⑩ 已选模板不存在时回退默认模板并套用内容
 * stub/mock：仅替换 bkui-vue 的 Select 为原生可交互 select，InfoBox 为可触发确认/取消回调；模板 API 返回真实 TemplateOutputObj 形状。
 * backlog：请求失败提示由组件统一消息反馈处理，本文件只断言选项回滚和宿主可用性状态。
 */
/* eslint-disable vue/one-component-per-file */
import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type * as BkuiVue from 'bkui-vue';

const mocks = vi.hoisted(() => ({
  getTemplate: vi.fn(),
  listTemplates: vi.fn(),
  infoBox: vi.fn(),
  routerResolve: vi.fn(() => ({ href: '#/space/basic/frameworkTemplates' })),
}));

vi.mock('~/api/modules/v1', () => ({
  FrameworkConfigTemplatesService: {
    getWorkspaceFrameworkConfigTemplate: mocks.getTemplate,
    listWorkspaceFrameworkConfigTemplates: mocks.listTemplates,
  },
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({ resolve: mocks.routerResolve }),
}));

vi.mock('bkui-vue', async importOriginal => {
  const actual = await importOriginal<typeof BkuiVue>();
  const selectOptionStub = defineComponent({
    name: 'SelectOptionInteractiveStub',
    props: {
      label: { type: String, default: '' },
      value: { type: [String, Number], required: true },
    },
    setup(props) {
      return () => h('option', { value: String(props.value) }, props.label);
    },
  });
  const selectStub = defineComponent({
    name: 'SelectInteractiveStub',
    props: {
      disabled: { type: Boolean, default: false },
      modelValue: { type: [String, Number], default: '' },
    },
    emits: ['change', 'update:modelValue'],
    setup(props, { emit, slots }) {
      return () => {
        const select = h(
          'select',
          {
            'aria-label': '配置模板',
            disabled: props.disabled,
            value: String(props.modelValue),
            onChange: (event: Event) => {
              const value = (event.target as HTMLSelectElement).value;
              emit('update:modelValue', value);
              emit('change', value);
            },
          },
          slots.default?.(),
        );
        return h('div', [slots.trigger?.(), select, slots.extension?.()]);
      };
    },
  });
  return {
    ...actual,
    InfoBox: mocks.infoBox,
    Message: () => undefined,
    Select: Object.assign(selectStub, { Option: selectOptionStub }),
  };
});

import FrameworkTemplateSelect from '~/pages/application/template/components/framework-template-select.vue';

interface InfoBoxOptions {
  onCancel?: () => void;
  onConfirm?: () => void;
}

interface Template {
  builtin?: boolean;
  content?: string;
  fileName: string;
  id: string;
  isDefault?: boolean;
  name: string;
}

const templates = {
  builtin: { builtin: true, fileName: 'trpc_go.yaml', id: 'builtin-trpc-go', isDefault: false, name: '平台默认' },
  customA: { builtin: false, fileName: 'custom-a.yaml', id: 'custom-a', isDefault: false, name: '模板 A' },
  customB: { builtin: false, fileName: 'custom-b.yaml', id: 'custom-b', isDefault: false, name: '模板 B' },
} satisfies Record<string, Template>;

const Harness = defineComponent({
  props: {
    appType: { type: String, default: 'trpc' },
    language: { type: String, default: 'go' },
    selectedTemplateId: { type: String, default: '' },
    variant: { type: String, default: 'text' },
  },
  setup(props) {
    const selectedContent = ref('');
    const selectedFileName = ref('');
    const selectedId = ref('');
    const templateAvailability = ref<'available' | 'empty' | 'loading'>('loading');
    const templateLoading = ref<'loading' | 'ready'>('loading');
    return () =>
      h('div', [
        h(FrameworkTemplateSelect, {
          appType: props.appType as 'taf' | 'trpc',
          language: props.language,
          selectedTemplateId: props.selectedTemplateId,
          variant: props.variant as 'field' | 'text',
          workspaceId: 'workspace-1',
          onAvailability: (available: boolean) => {
            templateAvailability.value = available ? 'available' : 'empty';
          },
          onLoading: (loading: boolean) => {
            templateLoading.value = loading ? 'loading' : 'ready';
          },
          onSelect: (template: Template) => {
            selectedContent.value = template.content || '';
            selectedFileName.value = template.fileName;
            selectedId.value = template.id;
          },
        }),
        h('output', { 'data-testid': 'selected-id' }, selectedId.value),
        h('output', { 'data-testid': 'selected-content' }, selectedContent.value),
        h('output', { 'data-testid': 'selected-file-name' }, selectedFileName.value),
        h('output', { 'data-testid': 'template-availability' }, templateAvailability.value),
        h('output', { 'data-testid': 'template-loading' }, templateLoading.value),
      ]);
  },
});

let activeInfoBox: InfoBoxOptions | undefined;

async function chooseTemplate(id: string) {
  await userEvent.selectOptions(screen.getByRole('combobox', { name: '配置模板' }), id);
  await waitFor(() => expect(mocks.infoBox).toHaveBeenCalled());
  activeInfoBox = mocks.infoBox.mock.lastCall?.[0] as InfoBoxOptions;
}

async function renderTemplateSelect(
  items: Template[],
  appType: 'taf' | 'trpc' = 'trpc',
  language = 'go',
  variant: 'field' | 'text' = 'text',
) {
  setTemplates(items);
  const view = render(Harness, {
    props: { appType, language, variant },
    global: {
      mocks: { $t: (key: string) => key },
    },
  });
  await waitFor(() => expect(screen.getByTestId('selected-content')).not.toHaveTextContent(''));
  await waitFor(() => expect(screen.getByTestId('template-loading')).toHaveTextContent('ready'));
  return view;
}

function setTemplates(items: Template[]) {
  mocks.listTemplates.mockResolvedValue(items);
  mocks.getTemplate.mockImplementation(({ templateID }: { templateID: string }) =>
    Promise.resolve({ ...items.find(item => item.id === templateID), content: `content-${templateID}` }),
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  activeInfoBox = undefined;
  mocks.infoBox.mockImplementation((options: InfoBoxOptions) => {
    activeInfoBox = options;
  });
});

describe('配置模板选择', () => {
  it('进入时优先选择默认模板，同时把平台内置模板排在前面', async () => {
    await renderTemplateSelect([{ ...templates.customA, isDefault: true }, templates.builtin]);

    const select = screen.getByRole('combobox', { name: '配置模板' });
    expect(select).toHaveValue('custom-a');
    expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a');
    expect(screen.getByTestId('selected-file-name')).toHaveTextContent('custom-a.yaml');
    expect(screen.getByText('配置模板：模板 A')).toBeInTheDocument();
    expect(
      within(select)
        .getAllByRole('option')
        .map(option => option.textContent),
    ).toEqual(['平台默认', '模板 A']);
  });

  it('没有默认模板时选择排序后的第一项', async () => {
    await renderTemplateSelect([templates.customA, templates.builtin, templates.customB]);

    expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('builtin-trpc-go');
    expect(screen.getByTestId('selected-content')).toHaveTextContent('content-builtin-trpc-go');
    expect(screen.getByTestId('selected-file-name')).toHaveTextContent('trpc_go.yaml');
  });

  it('TAF 使用默认 Select trigger，并按 taf/cpp 请求模板', async () => {
    await renderTemplateSelect([templates.customA], 'taf', 'cpp', 'field');

    expect(mocks.listTemplates).toHaveBeenCalledWith(
      { workspaceID: 'workspace-1', framework: 'taf', language: 'cpp' },
      expect.objectContaining({ id: expect.any(String) }),
    );
    expect(screen.queryByText('配置模板：')).not.toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('custom-a');
  });

  it('点击新建模板后在新标签页打开空间框架配置模板', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    await renderTemplateSelect([templates.customA]);

    await userEvent.click(screen.getByRole('button', { name: '新建模板' }));

    expect(mocks.routerResolve).toHaveBeenCalledWith({
      name: 'basicItem',
      params: { menuName: 'frameworkTemplates', space: 'workspace-1' },
    });
    expect(openSpy).toHaveBeenCalledWith('#/space/basic/frameworkTemplates', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('无远端模板时通知宿主隐藏模板选择区域', async () => {
    setTemplates([]);
    render(Harness, {
      global: {
        mocks: { $t: (key: string) => key },
      },
    });

    await waitFor(() => expect(screen.getByTestId('template-availability')).toHaveTextContent('empty'));
    await waitFor(() => expect(screen.getByTestId('template-loading')).toHaveTextContent('ready'));
  });

  it('详情加载期间保持只读，确认切换后带入新模板内容', async () => {
    let resolveDetail!: (value: Template) => void;
    const detail = new Promise<Template>(resolve => {
      resolveDetail = resolve;
    });
    setTemplates([templates.customA, templates.customB]);
    mocks.getTemplate.mockImplementation(({ templateID }: { templateID: string }) =>
      templateID === 'custom-b' ? detail : Promise.resolve({ ...templates.customA, content: 'content-custom-a' }),
    );
    render(Harness, {
      global: {
        mocks: { $t: (key: string) => key },
      },
    });
    await waitFor(() => expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a'));

    await chooseTemplate('custom-b');
    activeInfoBox?.onConfirm?.();

    await waitFor(() => expect(screen.getByTestId('template-loading')).toHaveTextContent('loading'));
    expect(screen.getByRole('combobox', { name: '配置模板' })).toBeDisabled();
    expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a');

    resolveDetail({ ...templates.customB, content: 'content-custom-b' });
    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('custom-b');
      expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-b');
      expect(screen.getByTestId('selected-file-name')).toHaveTextContent('custom-b.yaml');
      expect(screen.getByTestId('template-loading')).toHaveTextContent('ready');
    });
  });

  it.each([
    ['取消切换', false],
    ['详情加载失败', true],
  ])('%s时恢复旧选项和旧内容', async (_caseName, detailFails) => {
    await renderTemplateSelect([templates.customA, templates.customB]);
    if (detailFails) {
      vi.spyOn(console, 'error').mockImplementation(() => undefined);
      mocks.getTemplate.mockRejectedValueOnce(new Error('detail unavailable'));
    }

    await chooseTemplate('custom-b');
    if (detailFails) activeInfoBox?.onConfirm?.();
    else activeInfoBox?.onCancel?.();

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('custom-a');
      expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a');
    });
  });

  it('重新加载后忽略迟到的旧模板详情', async () => {
    let resolveOldDetail!: (value: Template) => void;
    const oldDetail = new Promise<Template>(resolve => {
      resolveOldDetail = resolve;
    });
    setTemplates([templates.customA, templates.customB]);
    mocks.getTemplate.mockImplementation(({ templateID }: { templateID: string }) => {
      if (templateID === 'custom-b') return oldDetail;
      return Promise.resolve({ ...templates.customA, content: 'content-custom-a' });
    });
    const view = render(Harness, {
      global: {
        mocks: { $t: (key: string) => key },
      },
    });
    await waitFor(() => expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a'));

    await chooseTemplate('custom-b');
    activeInfoBox?.onConfirm?.();
    await waitFor(() =>
      expect(mocks.getTemplate).toHaveBeenCalledWith(
        { workspaceID: 'workspace-1', templateID: 'custom-b' },
        expect.objectContaining({ id: expect.any(String) }),
      ),
    );

    mocks.listTemplates.mockResolvedValue([templates.customA, templates.customB]);
    await view.rerender({ language: 'cpp' });
    await waitFor(() => expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a'));

    resolveOldDetail({ ...templates.customB, content: 'late-old-content' });
    await flushPromises();
    expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a');
    expect(screen.getByTestId('selected-content')).not.toHaveTextContent('late-old-content');
  });

  it('恢复已选模板时仅回显选中项，不重新套用模板内容', async () => {
    // 模拟向导「第 3 步 → 上一步 → 再进第 3 步」：宿主已持有选中模板 id 与已保存的编辑内容
    setTemplates([templates.customA, templates.customB]);
    render(Harness, {
      props: { selectedTemplateId: 'custom-b' },
      global: {
        mocks: { $t: (key: string) => key },
      },
    });

    await waitFor(() => expect(screen.getByTestId('template-loading')).toHaveTextContent('ready'));
    expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('custom-b');
    expect(screen.getByText('配置模板：模板 B')).toBeInTheDocument();
    // 未 emit select、未请求详情，宿主已保存的编辑内容不被默认模板覆盖
    expect(screen.getByTestId('selected-content')).toHaveTextContent('');
    expect(mocks.getTemplate).not.toHaveBeenCalled();
  });

  it('已选模板不在列表中时回退为默认模板并套用其内容', async () => {
    setTemplates([{ ...templates.customA, isDefault: true }, templates.customB]);
    render(Harness, {
      props: { selectedTemplateId: 'deleted-template' },
      global: {
        mocks: { $t: (key: string) => key },
      },
    });

    await waitFor(() => expect(screen.getByTestId('selected-content')).toHaveTextContent('content-custom-a'));
    expect(screen.getByRole('combobox', { name: '配置模板' })).toHaveValue('custom-a');
  });
});
