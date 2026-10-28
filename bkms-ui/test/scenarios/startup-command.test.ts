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

import { defineComponent, nextTick } from 'vue';

import userEvent from '@testing-library/user-event';
import { waitFor, within } from '@testing-library/vue';
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import StartupCommandInput from '~/components/startup-command-input.vue';
import ProgramConfig from '~/pages/application/detail/app-config/components/program-config.vue';
import TafAppConfig from '~/pages/application/template/taf/app-config.vue';
import TrpcAppConfig from '~/pages/application/template/trpc/app-config.vue';
import { useAppDetail } from '~/stores/app-detail';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

import type { AppModelSpecInput } from '~/@types/v1/app';

const mocks = vi.hoisted(() => ({
  updateTrpc: vi.fn(),
  updateTaf: vi.fn(),
  getApp: vi.fn(),
  message: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
}));
vi.mock('~/api/modules/v1', () => ({
  AppService: { updateAppTrpcSpec: mocks.updateTrpc, updateAppTafSpec: mocks.updateTaf },
}));
vi.mock('~/api/modules/v1/app', () => ({ AppService: {} }));
vi.mock('~/api/modules/bkmsserver', () => ({ ApiServerService: { GetApp: mocks.getApp } }));
vi.mock('~/composables/use-env-manager', async () => {
  const { ref } = await import('vue');
  return { default: () => ({ envList: ref([]), handleGetEnvList: vi.fn().mockResolvedValue([]) }) };
});
// 框架编辑器及侧栏布局不参与启动命令交互，保留父组件所需的公开契约。
vi.mock('~/components/monaco-editor/ms-editor.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      methods: { getValue: () => 'config: value' },
      template: '<div />',
    }),
  };
});

enableAutoUnmount(afterEach);
installVxeShims();

const SCRIPT = '  echo "ready" && printf "%s\\n" "$PORT" | cat\nexec /server --port="$PORT"\n';
const EXEC_SPEC = { command: ['/server'], args: ['--port=8080'], shCommand: '' };

beforeEach(() => {
  vi.clearAllMocks();
  mocks.updateTrpc.mockResolvedValue(undefined);
  mocks.updateTaf.mockResolvedValue(undefined);
});

describe('启动命令输入', () => {
  it('默认使用 exec，允许命令和参数列表均为空', async () => {
    const wrapper = mount(StartupCommandInput, { attachTo: document.body, global: { mocks: i18nGlobalMocks } });
    expect(wrapper.find('textarea').exists()).toBe(false);
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toEqual({ command: [], args: [], shCommand: '' });
  });

  it('切换时保留两种草稿，只校验和提交当前模式，并原样保留脚本', async () => {
    const wrapper = mount(StartupCommandInput, {
      attachTo: document.body,
      props: { value: EXEC_SPEC },
      global: { mocks: i18nGlobalMocks },
    });
    const ui = within(wrapper.get('div').element);
    const addButtons = ui.getAllByRole('button', { name: '添加' });
    expect(addButtons).toHaveLength(2);
    await userEvent.click(addButtons[0]);
    await userEvent.click(addButtons[1]);
    expect(await wrapper.vm.validate()).toBe(false);
    await userEvent.click(ui.getByText('shell', { exact: true }));
    await wrapper.get('textarea').setValue(SCRIPT);
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toEqual({ command: [], args: [], shCommand: SCRIPT });

    await userEvent.click(ui.getByText('exec', { exact: true }));
    expect(wrapper.findAll<HTMLInputElement>('input[type="text"]').map(input => input.element.value)).toEqual([
      '/server',
      '',
      '--port=8080',
      '',
    ]);
    await wrapper.findAll('input[type="text"]')[1].setValue('serve');
    await wrapper.findAll('input[type="text"]')[3].setValue('--verbose');
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toEqual({
      command: ['/server', 'serve'],
      args: [...EXEC_SPEC.args, '--verbose'],
      shCommand: '',
    });
    await userEvent.click(ui.getByText('shell', { exact: true }));
    expect(wrapper.get('textarea').element.value).toBe(SCRIPT);
    // 编辑草稿不修改传入的已保存配置。
    expect(EXEC_SPEC.command).toEqual(['/server']);
  });

  it.each(['', ' \n\t'])('shell 脚本为 %j 时阻止提交并提示必填', async script => {
    const wrapper = mount(StartupCommandInput, { attachTo: document.body, global: { mocks: i18nGlobalMocks } });
    await userEvent.click(within(wrapper.get('div').element).getByText('shell', { exact: true }));
    await wrapper.get('textarea').setValue(script);
    expect(await wrapper.vm.validate()).toBe(false);
    expect(wrapper.text()).toContain('必填项');
  });

  it('已有 shCommand 时直接回显 shell，并保留首尾空白和换行', () => {
    const wrapper = mount(StartupCommandInput, {
      attachTo: document.body,
      props: { value: { shCommand: SCRIPT } },
      global: { mocks: i18nGlobalMocks },
    });
    expect(wrapper.get('textarea').element.value).toBe(SCRIPT);
    expect(wrapper.vm.getValue()).toEqual({ command: [], args: [], shCommand: SCRIPT });
  });
});

const createPageStubs = {
  ToggleCard: { template: '<div><slot /></div>' },
  ResizeLayout: defineComponent({
    setup(_props, { expose }) {
      expose({ asideRef: { hidden: true } });
    },
    template: '<div><slot name="main" /></div>',
  }),
};

describe('创建应用的启动配置', () => {
  it('tRPC 创建表单默认使用 shell，校验并将原始脚本交给创建流程', async () => {
    const wrapper = mount(TrpcAppConfig, {
      attachTo: document.body,
      props: { value: { appModelSpec: { command: [], args: [] } } },
      global: { mocks: i18nGlobalMocks, stubs: createPageStubs },
    });
    await flushPromises();
    expect(wrapper.get('textarea').attributes('placeholder')).toBe('请输入');
    expect(await wrapper.vm.validate()).toBe(false);
    await wrapper.get('textarea').setValue(SCRIPT);
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toMatchObject({ command: [], args: [], shCommand: SCRIPT, content: 'config: value' });
  });

  it('TAF 创建表单默认使用 shell，切换到 exec 后允许空列表', async () => {
    const wrapper = mount(TafAppConfig, {
      attachTo: document.body,
      props: {
        value: {
          command: [],
          args: [],
          envVars: [],
          tafSpec: { fileName: 'app.conf', filePath: '/etc/app', fileContent: 'config' },
        },
      },
      global: { mocks: i18nGlobalMocks, stubs: createPageStubs },
    });
    await flushPromises();
    expect(wrapper.get('textarea').attributes('placeholder')).toBe('请输入');
    expect(await wrapper.vm.validate()).toBe(false);
    await userEvent.click(within(wrapper.get('form').element).getByText('exec', { exact: true }));
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toMatchObject({ command: [], args: [], shCommand: '' });
  });

  it('TAF 创建表单从已有 shell 回显，切回 exec 时清空提交脚本', async () => {
    const wrapper = mount(TafAppConfig, {
      attachTo: document.body,
      props: {
        value: {
          command: [],
          args: [],
          shCommand: SCRIPT,
          envVars: [],
          tafSpec: { fileName: 'app.conf', filePath: '/etc/app', fileContent: 'config' },
        },
      },
      global: { mocks: i18nGlobalMocks, stubs: createPageStubs },
    });
    await flushPromises();
    expect(wrapper.get<HTMLTextAreaElement>('textarea').element.value).toBe(SCRIPT);
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toMatchObject({ command: [], args: [], shCommand: SCRIPT });
    await userEvent.click(within(wrapper.get('form').element).getByText('exec', { exact: true }));
    expect(await wrapper.vm.validate()).toBe(true);
    expect(wrapper.vm.getValue()).toMatchObject({ command: [], args: [], shCommand: '' });
  });
});

function mountProgram(type: 'taf' | 'trpc', spec: AppModelSpecInput = EXEC_SPEC) {
  const pinia = createPinia();
  setActivePinia(pinia);
  const store = useAppDetail();
  store.updateAppID('app-1');
  store.updateAppType(type);
  store.appDetail = { id: 'app-1', type, appModelSpec: spec };
  const wrapper = mount(ProgramConfig, {
    attachTo: document.body,
    props: { currentEnv: { name: 'default', isDefault: true, isModified: false } },
    global: { plugins: [pinia], mocks: i18nGlobalMocks, directives: { 'bk-tooltips': {} } },
  });
  return { wrapper, store, ui: within(wrapper.get('div').element) };
}

describe('部署启动配置', () => {
  it.each(['trpc', 'taf'] as const)('%s 从 exec 保存为 shell，清空旧字段并保留其他配置', async type => {
    const frameworkSpec =
      type === 'trpc' ? { trpcSpec: { language: 'go' as const } } : { tafSpec: { fileContent: 'config' } };
    const { wrapper, ui } = mountProgram(type, { ...EXEC_SPEC, ...frameworkSpec });
    mocks.getApp.mockResolvedValue({ id: 'app-1', type, appModelSpec: { ...frameworkSpec, shCommand: SCRIPT } });
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    await userEvent.click(ui.getByText('shell', { exact: true }));
    await wrapper.get('textarea').setValue(SCRIPT);
    await userEvent.click(ui.getByRole('button', { name: '保存' }));
    const update = type === 'trpc' ? mocks.updateTrpc : mocks.updateTaf;
    await waitFor(() =>
      expect(update).toHaveBeenCalledWith({
        appID: 'app-1',
        appModelSpec: { ...frameworkSpec, command: [], args: [], shCommand: SCRIPT },
      }),
    );
    await waitFor(() => expect(wrapper.find('textarea').exists()).toBe(false));
    expect(ui.getByText('shell', { exact: true })).toBeVisible();
    expect(ui.getByText(SCRIPT, { normalizer: text => text })).toBeVisible();
    expect(mocks.getApp).toHaveBeenCalledWith({ appID: 'app-1' });
  });

  it('从 shell 保存为空 exec 配置时清空脚本', async () => {
    const { wrapper, ui } = mountProgram('trpc', { shCommand: SCRIPT });
    mocks.getApp.mockResolvedValue({
      id: 'app-1',
      type: 'trpc',
      appModelSpec: { command: [], args: [], shCommand: '' },
    });
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    await userEvent.click(ui.getByText('exec', { exact: true }));
    await userEvent.click(ui.getByRole('button', { name: '保存' }));
    await waitFor(() =>
      expect(mocks.updateTrpc).toHaveBeenCalledWith({
        appID: 'app-1',
        appModelSpec: { command: [], args: [], shCommand: '' },
      }),
    );
    await waitFor(() => expect(wrapper.find('textarea').exists()).toBe(false));
    expect(ui.getByText('exec', { exact: true })).toBeVisible();
    expect(ui.queryByText(SCRIPT, { normalizer: text => text })).not.toBeInTheDocument();
  });

  it('取消编辑后恢复已保存模式和内容', async () => {
    const { wrapper, ui, store } = mountProgram('trpc', { shCommand: SCRIPT });
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    await wrapper.get('textarea').setValue('changed');
    await userEvent.click(ui.getByText('exec', { exact: true }));
    await userEvent.click(ui.getByRole('button', { name: '取消' }));
    expect(ui.getByText('shell', { exact: true })).toBeVisible();
    expect(ui.getByText(SCRIPT, { normalizer: text => text })).toBeVisible();
    expect(store.appDetail?.appModelSpec?.shCommand).toBe(SCRIPT);
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    expect(wrapper.get('textarea').element.value).toBe(SCRIPT);
    expect(mocks.updateTrpc).not.toHaveBeenCalled();
  });

  it('保存失败后保留草稿并允许再次保存', async () => {
    const { wrapper, ui } = mountProgram('trpc', { shCommand: SCRIPT });
    mocks.updateTrpc.mockRejectedValueOnce(new Error('save failed'));
    mocks.getApp.mockResolvedValue({ id: 'app-1', type: 'trpc', appModelSpec: { shCommand: 'echo retry' } });
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    await wrapper.get('textarea').setValue('echo retry');
    await userEvent.click(ui.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(mocks.updateTrpc).toHaveBeenCalledTimes(1));
    expect(wrapper.get('textarea').element.value).toBe('echo retry');
    await userEvent.click(ui.getByRole('button', { name: '保存' }));
    await waitFor(() => expect(mocks.updateTrpc).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(wrapper.find('textarea').exists()).toBe(false));
  });

  it('普通环境禁止编辑；编辑过程中切换到普通环境也不能保存', async () => {
    const { wrapper, ui } = mountProgram('trpc');
    await userEvent.click(ui.getByRole('button', { name: '编辑' }));
    await wrapper.setProps({ currentEnv: { name: 'prod', isDefault: false, isModified: false } });
    await userEvent.click(ui.getByRole('button', { name: '保存' }));
    await nextTick();
    expect(mocks.updateTrpc).not.toHaveBeenCalled();
    await userEvent.click(ui.getByRole('button', { name: '取消' }));
    expect(ui.getByRole('button', { name: '编辑' })).toBeDisabled();
  });
});
