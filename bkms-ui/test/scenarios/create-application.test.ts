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

import { defineComponent, h, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CreateApp from '~/pages/application/create.vue';
import BKMSAppWizard from '~/pages/application/template/bkms-app/index.vue';
import { buildBKMSAppPayload, createBKMSAppForm } from '~/pages/application/template/bkms-app/use-create-app';
import TemplateSelect from '~/pages/application/template/index.vue';

import { i18nGlobalMocks, i18nPlugin } from '../helpers/mock-i18n';

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  definitions: vi.fn(),
  suffix: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  paramValidate: vi.fn(),
  paramValue: vi.fn(),
  appValidate: vi.fn(),
  appValue: vi.fn(),
  setQuery: vi.fn(),
}));
const routeState = vi.hoisted(() => ({
  name: 'createBKMSAppTemplateApp',
  params: { space: 'ws-1' },
  query: { framework: 'trpc', language: 'go' },
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1/app', () => ({
  AppService: {
    getAppRuntimeDefinitions: mocks.definitions,
    getAppIDAutoSuffix: mocks.suffix,
    createBKMSApp: mocks.create,
  },
}));
vi.mock('vue-router', async () => {
  const { createRouterMock } = await import('../helpers/mock-router');
  const { reactive } = await import('vue');
  const route = reactive(routeState);
  mocks.setQuery.mockImplementation(query => {
    route.query = query;
  });
  const base = await createRouterMock({ push: mocks.push, route: routeState });
  return {
    ...base,
    useRoute: () => route,
    useRouter: () => ({ push: mocks.push, replace: mocks.replace, back: vi.fn() }),
  };
});
vi.mock('~/components/ms-header.vue', () => ({ default: defineComponent({ template: '<div><slot /></div>' }) }));
vi.mock('~/pages/application/template/bkms-app/param-config.vue', () => ({
  default: defineComponent({
    props: { form: { type: Object, required: true } },
    setup(props, { expose }) {
      const initialForm = props.form;
      expose({ validate: () => mocks.paramValidate(), getValue: () => mocks.paramValue(initialForm) });
      return () => h('div', { 'data-testid': 'parameters' });
    },
  }),
}));
vi.mock('~/pages/application/template/trpc/app-config.vue', () => ({
  default: defineComponent({
    setup(_, { expose }) {
      expose({ validate: () => mocks.appValidate(), getValue: () => mocks.appValue() });
      return () => h('div', { 'data-testid': 'framework-config' });
    },
  }),
}));
vi.mock('~/pages/application/template/taf/app-config.vue', () => ({
  default: defineComponent({
    setup(_, { expose }) {
      expose({ validate: () => mocks.appValidate(), getValue: () => mocks.appValue() });
      return () => h('div', { 'data-testid': 'framework-config' });
    },
  }),
}));

const combinations = [
  ['trpc', 'go'],
  ['trpc', 'cpp'],
  ['taf', 'cpp'],
  ['blank', 'go'],
  ['blank', 'python'],
  ['blank', 'nodejs'],
] as const;
function definitions() {
  return {
    frameworks: [
      { name: 'trpc', supportedLanguages: ['go', 'cpp'] },
      { name: 'taf', supportedLanguages: ['cpp'] },
      { name: 'blank', supportedLanguages: ['go', 'cpp', 'python', 'nodejs'] },
    ],
    matrix: [...combinations, ['blank', 'cpp']].map(([framework, language]) => ({
      framework,
      language,
      type: 'bkmsApp',
      schemaVersion: 1,
      mods: {
        frameworkFile: { supported: framework !== 'blank', props: { format: framework === 'taf' ? 'taf' : 'yaml' } },
        platformBuild: { supported: framework === 'trpc' && language === 'go' },
      },
    })),
  };
}
const WizardHarness = defineComponent({
  setup() {
    const step = ref(2);
    return () =>
      h(BKMSAppWizard, {
        step: step.value,
        'onUpdate-step': (value: number) => {
          step.value = value;
        },
      });
  },
});
function renderPage(component: Parameters<typeof render>[0]) {
  const routerView = defineComponent({
    setup(_, { slots }) {
      return () => slots.default?.({ Component: defineComponent({ render: () => null }) });
    },
  });
  return render(component, {
    global: { plugins: [createPinia(), i18nPlugin], mocks: i18nGlobalMocks, components: { RouterView: routerView } },
  });
}
beforeEach(() => {
  vi.clearAllMocks();
  routeState.name = 'createBKMSAppTemplateApp';
  routeState.query = { framework: 'trpc', language: 'go' };
  mocks.definitions.mockResolvedValue(definitions());
  mocks.suffix.mockResolvedValue({ suffix: '-1' });
  mocks.paramValidate.mockResolvedValue(true);
  mocks.paramValue.mockImplementation(form => ({ ...form, name: 'demo', id: 'demo-1' }));
  mocks.appValidate.mockResolvedValue(true);
  mocks.appValue.mockReturnValue({
    command: ['/server'],
    args: ['--port=8080'],
    shCommand: '',
    env: [{ key: 'PORT', value: '8080' }],
    content: 'server: demo',
    envVars: [{ key: 'PORT', value: '8080' }],
    tafSpec: { fileName: 'config.conf', filePath: '/usr/local/taf/conf', fileContent: '<taf />' },
  });
  mocks.create.mockResolvedValue({ id: 'demo-1', name: 'demo', type: 'bkmsApp' });
});

describe('创建应用模板', () => {
  it.each(['blank', 'trpc'])('%s向导刷新后正确恢复步骤条', async framework => {
    routeState.query.framework = framework;
    renderPage(CreateApp);
    expect(await screen.findByText('参数配置')).toBeVisible();
    if (framework === 'blank') expect(screen.queryByText('创建应用')).not.toBeInTheDocument();
    else expect(screen.getByText('创建应用')).toBeVisible();
  });
  it('同时展示开发框架和语言，选择语言后传递blank组合', async () => {
    renderPage(TemplateSelect);
    expect(screen.getByText('开发框架')).toBeVisible();
    expect(screen.getByText('按语言')).toBeVisible();
    await waitFor(() => expect(screen.getByRole('button', { name: /Python/ })).toBeEnabled());
    await userEvent.click(screen.getByRole('button', { name: /Python/ }));
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'createBKMSAppTemplateApp',
      query: { framework: 'blank', language: 'python' },
    });
    expect(screen.getByRole('button', { name: /tRPC/ })).toHaveAttribute('aria-pressed', 'false');
  });
  it('目录失败可重试，Helm创建仍可继续', async () => {
    mocks.definitions.mockRejectedValueOnce(new Error('offline'));
    renderPage(TemplateSelect);
    expect(await screen.findByText('应用运行时目录加载失败，请重试')).toBeVisible();
    expect(screen.getByRole('button', { name: '下一步' })).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: /Helm Chart/ }));
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.push).toHaveBeenCalledWith({ name: 'createHelmTemplateApp' });
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    await waitFor(() => expect(screen.queryByText('应用运行时目录加载失败，请重试')).not.toBeInTheDocument());
  });
});

describe('统一平台托管创建向导', () => {
  it.each(combinations)('%s + %s使用新创建请求，结果跳转真实类型', async (framework, language) => {
    routeState.query = { framework, language };
    renderPage(WizardHarness);
    await screen.findByTestId('parameters');
    if (framework !== 'blank') {
      await userEvent.click(screen.getByRole('button', { name: '下一步' }));
      await screen.findByTestId('framework-config');
    }
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(await screen.findByText(/应用创建成功/)).toBeVisible();
    const payload = mocks.create.mock.calls[0][0];
    expect(payload).toMatchObject({ workspaceID: 'ws-1', framework, language, name: 'demo' });
    expect(payload).not.toHaveProperty('type');
    expect(payload.appModelSpec).not.toHaveProperty('trpcSpec');
    expect(payload.appModelSpec).not.toHaveProperty('tafSpec');
    if (framework === 'blank') {
      expect(payload).not.toHaveProperty('frameworkConfigFile');
      expect(payload.buildConfig.repoBuildConfig).not.toHaveProperty('platformBuildConfig');
    } else {
      expect(payload.frameworkConfigFile.content).toBe(framework === 'taf' ? '<taf />' : 'server: demo');
    }
    await userEvent.click(screen.getByRole('button', { name: '查看应用' }));
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'detail',
      params: { name: 'demo', type: 'bkmsApp', menuName: 'info' },
    });
  });
  it('同一路由切换查询组合后提交新技术栈', async () => {
    renderPage(WizardHarness);
    await screen.findByTestId('parameters');
    mocks.setQuery({ framework: 'blank', language: 'python' });
    await waitFor(() => expect(screen.getByText('Python')).toBeVisible());
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create.mock.calls[0][0]).toMatchObject({ framework: 'blank', language: 'python' });
    expect(mocks.create.mock.calls[0][0]).not.toHaveProperty('frameworkConfigFile');
  });
  it('参数校验失败时停留，不能提交创建', async () => {
    mocks.paramValidate.mockResolvedValue(false);
    renderPage(WizardHarness);
    await screen.findByTestId('parameters');
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(screen.queryByTestId('framework-config')).not.toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });
  it('无效或首版未开放的组合返回模板页', async () => {
    routeState.query = { framework: 'blank', language: 'cpp' };
    renderPage(WizardHarness);
    await waitFor(() => expect(mocks.replace).toHaveBeenCalledWith({ name: 'createApplication' }));
    expect(mocks.create).not.toHaveBeenCalled();
  });
  it('失败后修改启动模式重试，提交完整的当前字段', async () => {
    mocks.create.mockRejectedValueOnce({ msg: '创建失败' });
    renderPage(WizardHarness);
    await screen.findByTestId('parameters');
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    await userEvent.click(await screen.findByRole('button', { name: '创建' }));
    expect(await screen.findByText('创建失败')).toBeVisible();
    await userEvent.click(screen.getByRole('button', { name: '重新创建' }));
    mocks.appValue.mockReturnValue({ command: [], args: [], shCommand: 'exec /server', env: [], content: 'a: 1' });
    await userEvent.click(await screen.findByRole('button', { name: '下一步' }));
    await userEvent.click(await screen.findByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(2));
    expect(mocks.create.mock.calls[1][0].appModelSpec).toEqual({
      command: [],
      args: [],
      shCommand: 'exec /server',
      envVars: [],
    });
  });
});

describe('创建请求隔离表单状态', () => {
  it('不支持平台构建时移除残留配置，流水线来源只提交流水线配置', () => {
    const form = createBKMSAppForm('ws-1', 'blank', 'go');
    form.buildConfig.repoBuildConfig.imageBuildMode = 'platform';
    form.buildConfig.repoBuildConfig.platformBuildConfig.builderImage = 'old-builder';
    const runtime = { mods: { platformBuild: { supported: false }, frameworkFile: { supported: false } } };
    expect(buildBKMSAppPayload(form, runtime).buildConfig.repoBuildConfig).toMatchObject({
      imageBuildMode: 'repositoryDockerfile',
    });
    expect(buildBKMSAppPayload(form, runtime).buildConfig.repoBuildConfig).not.toHaveProperty('platformBuildConfig');
    form.buildConfig.sourceType = 'pipeline';
    const payload = buildBKMSAppPayload(form, runtime);
    expect(payload.buildConfig).not.toHaveProperty('repoBuildConfig');
    expect(payload.buildConfig.pipelineBuildConfig).toEqual({ pipelineID: '', params: {} });
  });
});
