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
import { render, screen, waitFor, within } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CreateApp from '~/pages/application/create.vue';
import BKMSAppWizard from '~/pages/application/template/bkms-app/index.vue';
import { buildBKMSAppPayload, createBKMSAppForm } from '~/pages/application/template/bkms-app/use-create-app';
import TemplateSelect from '~/pages/application/template/index.vue';
import { useAppDetail } from '~/stores/app-detail';
import { useAppStore } from '~/stores/apps';

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
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

async function openCreateStep(framework: string) {
  routeState.query = { framework, language: framework === 'taf' ? 'cpp' : 'go' };
  const page = renderPage(WizardHarness);
  await screen.findByTestId('parameters');
  if (framework !== 'blank') {
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    await screen.findByTestId('framework-config');
  }
  return page;
}

function renderPage(component: Parameters<typeof render>[0], pinia = createPinia()) {
  const routerView = defineComponent({
    setup(_, { slots }) {
      return () => slots.default?.({ Component: defineComponent({ render: () => null }) });
    },
  });
  return render(component, {
    global: { plugins: [pinia, i18nPlugin], mocks: i18nGlobalMocks, components: { RouterView: routerView } },
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
  it('展示框架和通用应用，选择通用应用后传递blank组合', async () => {
    renderPage(TemplateSelect);
    expect(screen.getByText('框架应用')).toBeVisible();
    expect(screen.getByText('通用应用')).toBeVisible();
    await waitFor(() => expect(screen.getByRole('button', { name: /Python/ })).toBeEnabled());
    await userEvent.click(screen.getByRole('button', { name: /Python/ }));
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'createBKMSAppTemplateApp',
      query: { framework: 'blank', language: 'python' },
    });
    expect(screen.getByRole('button', { name: /tRPC/ })).toHaveAttribute('aria-pressed', 'false');
  });
  it('切换框架语言后立即记住选择，重新进入模板页仍选中该语言', async () => {
    const pinia = createPinia();
    const appStore = useAppStore(pinia);
    const page = renderPage(TemplateSelect, pinia);
    await userEvent.click(await screen.findByRole('button', { name: 'C++' }));
    expect(appStore.lastAppTemplateID).toBe('trpc:cpp');

    page.unmount();
    renderPage(TemplateSelect, pinia);
    expect(await screen.findByRole('button', { name: 'C++' })).toHaveAttribute('aria-pressed', 'true');
  });
  it('从通用应用点击框架卡片中的语言时选中对应框架，单语言框架只显示语言文本', async () => {
    const pinia = createPinia();
    const appStore = useAppStore(pinia);
    renderPage(TemplateSelect, pinia);
    const pythonCard = screen.getByRole('button', { name: /Python/ });
    await waitFor(() => expect(pythonCard).toBeEnabled());
    await userEvent.click(pythonCard);

    await userEvent.click(screen.getByRole('button', { name: 'C++' }));
    expect(appStore.lastAppTemplateID).toBe('trpc:cpp');
    expect(screen.getByRole('button', { name: /tRPC/ })).toHaveAttribute('aria-pressed', 'true');
    expect(pythonCard).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'createBKMSAppTemplateApp',
      query: { framework: 'trpc', language: 'cpp' },
    });

    const tafCard = screen.getByRole('button', { name: /TAF/ });
    const tafContent = tafCard.closest<HTMLElement>('.template-card');
    if (!tafContent) throw new Error('TAF template card missing');
    const taf = within(tafContent);
    expect(taf.getByText('C++', { exact: true })).toBeVisible();
    expect(taf.queryByRole('button', { name: 'C++' })).not.toBeInTheDocument();
    await userEvent.click(tafCard);
    expect(appStore.lastAppTemplateID).toBe('taf:cpp');
    expect(tafCard).toHaveAttribute('aria-pressed', 'true');
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

describe('创建异步操作隔离', () => {
  it.each(['blank', 'trpc', 'taf'])('%s 校验期间连续点击只校验和创建一次', async framework => {
    await openCreateStep(framework);
    const validation = deferred<boolean>();
    const validate = framework === 'blank' ? mocks.paramValidate : mocks.appValidate;
    validate.mockReturnValue(validation.promise);

    await userEvent.dblClick(screen.getByRole('button', { name: '创建' }));
    expect(validate).toHaveBeenCalledTimes(1);
    expect(mocks.create).not.toHaveBeenCalled();
    validation.resolve(true);

    expect(await screen.findByText(/应用创建成功/)).toBeVisible();
    expect(mocks.create).toHaveBeenCalledTimes(1);
  });

  it.each(['false', 'throw'])('校验 %s 后释放提交锁，允许修正后再次创建', async failure => {
    await openCreateStep('blank');
    const validation = deferred<boolean>();
    mocks.paramValidate.mockReturnValueOnce(validation.promise);
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    if (failure === 'throw') validation.reject(new Error('validation unavailable'));
    else validation.resolve(false);
    await flushPromises();

    await waitFor(() => expect(screen.getByRole('button', { name: '创建' })).toBeEnabled());
    expect(mocks.create).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole('button', { name: '创建' }));

    expect(await screen.findByText(/应用创建成功/)).toBeVisible();
    expect(mocks.create).toHaveBeenCalledTimes(1);
  });

  it('离开创建页后，迟到的创建成功响应不能覆盖新应用上下文', async () => {
    routeState.query = { framework: 'blank', language: 'python' };
    const pinia = createPinia();
    const appDetail = useAppDetail(pinia);
    const creation = deferred<{ id: string; name: string; type: string }>();
    mocks.create.mockReturnValueOnce(creation.promise);
    const { unmount } = renderPage(WizardHarness, pinia);
    await screen.findByTestId('parameters');
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));

    unmount();
    appDetail.updateAppID('app-b');
    appDetail.updateAppName('application-b');
    appDetail.updateAppType('taf');
    const runtime = { framework: 'taf', mods: { adminCommand: { supported: true } } };
    appDetail.runtime = runtime;
    creation.resolve({ id: 'created-a', name: 'application-a', type: 'bkmsApp' });
    await flushPromises();

    expect(appDetail.appID).toBe('app-b');
    expect(appDetail.app).toBe('application-b');
    expect(appDetail.appType).toBe('taf');
    expect(appDetail.runtime).toEqual(runtime);
  });

  it.each(['blank', 'trpc', 'taf'])('%s 校验期间离开页面，校验完成后不发送创建请求', async framework => {
    const { unmount } = await openCreateStep(framework);
    const validation = deferred<boolean>();
    const validate = framework === 'blank' ? mocks.paramValidate : mocks.appValidate;
    validate.mockReturnValueOnce(validation.promise);
    await userEvent.click(screen.getByRole('button', { name: '创建' }));
    expect(validate).toHaveBeenCalledTimes(1);

    unmount();
    validation.resolve(true);
    await flushPromises();
    expect(mocks.create).not.toHaveBeenCalled();
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
