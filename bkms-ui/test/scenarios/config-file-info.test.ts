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

import { ref } from 'vue';

import { flushPromises, shallowMount } from '@vue/test-utils';
import { Radio } from 'bkui-vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ConfigFileListItem from '~/pages/application/detail/app-config/components/config-file-list-item.vue';
import ConfigFiles from '~/pages/application/detail/app-config/config-files.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { MockedApiError } from '../helpers/mocked-api-error';

import type * as BkuiVue from 'bkui-vue';
import type { PublicView } from '~/@types/v1/app';
import type { DefDetailObj, UpdateDefInput } from '~/@types/v1/app-config-file-defs';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  detail: vi.fn(),
  instances: vi.fn(),
  update: vi.fn(),
  create: vi.fn(),
  appDetail: vi.fn(),
  message: vi.fn(),
  appType: 'taf',
  editorValue: '',
  saveContent: vi.fn(),
}));
const runtimeState = ref<null | PublicView | undefined>();
vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  useI18n: () => ({
    t: (key: string, params: unknown[] = []) =>
      key.replace(/\{(\d+)\}/g, (_, index: string) => String(params[Number(index)])),
    te: () => true,
  }),
}));
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('vue-router', () => ({ onBeforeRouteLeave: vi.fn(), onBeforeRouteUpdate: vi.fn() }));
vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({
    appID: 'app-id',
    loading: false,
    get runtime() {
      return runtimeState.value === undefined
        ? {
            framework: mocks.appType,
            mods: { frameworkFile: { supported: true, props: { format: mocks.appType === 'taf' ? 'taf' : 'yaml' } } },
          }
        : runtimeState.value;
    },
    fetchAppDetail: mocks.appDetail,
    appDetail: { appModelSpec: { tafSpec: { fileName: 'old-spec.conf', filePath: '/old-spec' } } },
  }),
}));
vi.mock('~/composables/use-leave-confirm', () => ({ default: () => ({ confirmBox: async () => true }) }));
vi.mock('~/api/modules/v1', () => ({
  AppConfigFileDefsService: {
    listDefaultFilesWithDef: mocks.list,
    getAppConfigFileDefDetail: mocks.detail,
    listAppConfigFileDefEnvInstances: mocks.instances,
    updateAppConfigFileDef: mocks.update,
    createAppConfigFileDef: mocks.create,
    updateAppConfigFileDefContent: mocks.saveContent,
  },
  AppConfigFilesService: {},
  EnvService: { listAppEnvs: async () => [{ name: 'prod', displayName: 'Production', type: 'prod' }] },
  EnvvarsService: {},
}));
vi.mock('bkui-vue', async importOriginal => {
  const actual = await importOriginal<typeof BkuiVue>();
  return { ...actual, Message: mocks.message };
});
vi.mock('~/components/monaco-editor/ms-editor.vue', () => ({
  default: {
    name: 'MsEditor',
    template: '<div><slot name="title" /></div>',
    methods: {
      setValue(value: string) {
        mocks.editorValue = value;
      },
      getValue: () => mocks.editorValue,
    },
  },
}));
vi.mock('~/pages/application/detail/app-config/components/env-content-strategy-confirm-dialog.vue', () => ({
  default: { name: 'EnvContentStrategyConfirmDialog', template: '<div />', methods: { confirm: async () => true } },
}));
vi.mock('~/pages/application/detail/app-config/components/clear-file-content-dialog.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/save-version-confirm-dialog.vue', () => ({
  default: { name: 'SaveVersionConfirmDialog', props: ['isShow'], template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/create-config-file-sideslider.vue', () => ({
  default: { name: 'CreateConfigFileSideslider', props: ['isShow', 'loading'], template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/mount-preview-sideslider.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/version-list-sideslider.vue', () => ({
  default: { template: '<div />' },
}));

let wrapper: ReturnType<typeof shallowMount<typeof ConfigFiles>> | undefined;
let savedFile: DefDetailObj;
const container = { template: '<div><slot /></div>' };

beforeEach(() => {
  vi.resetAllMocks();
  mocks.appType = 'taf';
  runtimeState.value = undefined;
  mocks.editorValue = '';
  savedFile = {
    id: 'def-id',
    fileId: 'file-id',
    configKind: 'framework',
    name: 'app.conf',
    mountDir: '/config',
    currentVersion: 3,
    content: '',
    envContentStrategy: 'overlay',
  };
  mocks.list.mockImplementation(async () => ({ items: [savedFile] }));
  mocks.detail.mockImplementation(async () => ({ item: savedFile }));
  mocks.instances.mockResolvedValue({ items: [] });
  mocks.appDetail.mockResolvedValue({});
  mocks.update.mockImplementation(async (input: UpdateDefInput) => {
    savedFile = { ...savedFile, ...input };
    return {
      item: {
        id: savedFile.id,
        name: savedFile.name,
        mountDir: savedFile.mountDir,
        envContentStrategy: savedFile.envContentStrategy,
      },
    };
  });
});
afterEach(() => {
  wrapper?.unmount();
});

async function openEditor() {
  wrapper = shallowMount(ConfigFiles, {
    global: {
      mocks: i18nGlobalMocks,
      directives: { 'bk-tooltips': {}, bkloading: {} },
      stubs: {
        ResizeLayout: { template: '<div><slot name="aside" /><slot name="main" /></div>' },
        Skeleton: container,
        BkmsContent: {
          name: 'BkmsContent',
          props: ['title'],
          template:
            '<section><button data-edit :data-section="title || \'文件内容\'" @click="$emit(\'edit\')">编辑</button><slot /></section>',
        },
        FieldItem: { template: '<div><slot name="value" /></div>' },
        Form: false,
        FormItem: false,
        Input: false,
        Button: false,
        MsEditor: false,
        ConfigFileList: false,
        SaveVersionConfirmDialog: false,
        EnvContentStrategyConfirmDialog: false,
        CreateConfigFileSideslider: false,
      },
      renderStubDefaultSlot: true,
    },
  });
  await flushPromises();
  await wrapper.get('[data-edit]').trigger('click');
  await flushPromises();
  return wrapper;
}

async function save() {
  const button = wrapper!.findAll('button').find(item => item.text() === '保存');
  expect(button).toBeDefined();
  await button!.trigger('click');
  await flushPromises();
}

describe('配置文件信息编辑', () => {
  it.each<[string, null | PublicView]>([
    ['未加载', null],
    ['不支持', { framework: 'taf', mods: { frameworkFile: { supported: false, reasonCode: 'invalid_stack' } } }],
  ])('框架文件能力%s时，仍展示并优先选择接口返回的框架文件', async (_, runtime) => {
    runtimeState.value = runtime;
    const plainFile = { ...savedFile, id: 'plain-id', configKind: 'plain' };
    mocks.list.mockResolvedValue({ items: [plainFile, savedFile] });
    const view = await openEditor();

    const list = view.getComponent({ name: 'ConfigFileList' });
    expect(list.props('files').map((file: { id: string }) => file.id)).toEqual(['plain-id', 'def-id']);
    expect(list.findAllComponents(ConfigFileListItem).map(item => item.props('file').id)).toEqual([
      'def-id',
      'plain-id',
    ]);
    expect(list.props('activeId')).toBe('def-id');
    expect(mocks.detail).toHaveBeenCalledExactlyOnceWith(
      { appID: 'app-id', id: 'def-id', envName: '' },
      { needRes: true },
    );

    runtimeState.value = { framework: 'taf', mods: { frameworkFile: { supported: true, props: { format: 'taf' } } } };
    await flushPromises();
    expect(list.props('activeId')).toBe('def-id');
    expect(mocks.detail).toHaveBeenCalledTimes(1);
  });

  it('能力重试成功时，仍选中的普通文件保留内容编辑草稿', async () => {
    runtimeState.value = null;
    savedFile = {
      ...savedFile,
      id: 'plain-id',
      configKind: 'plain',
      content: 'original content',
      editableContentField: 'content',
    };
    const view = await openEditor();
    await view
      .findAll('button')
      .find(button => button.text() === '取消')!
      .trigger('click');
    await view.get('[data-section="文件内容"]').trigger('click');
    mocks.editorValue = 'unsaved content';
    view.getComponent({ name: 'MsEditor' }).vm.$emit('change', mocks.editorValue);
    await flushPromises();

    runtimeState.value = { framework: 'taf', mods: { frameworkFile: { supported: true, props: { format: 'taf' } } } };
    await flushPromises();

    expect(view.getComponent({ name: 'ConfigFileList' }).props('activeId')).toBe('plain-id');
    expect(mocks.editorValue).toBe('unsaved content');
    expect(mocks.detail).toHaveBeenCalledTimes(1);
    await save();
    const saveDialog = view.getComponent({ name: 'SaveVersionConfirmDialog' });
    expect(saveDialog.props('isShow')).toBe(true);
    saveDialog.vm.$emit('confirm', 'keep draft');
    await flushPromises();
    expect(mocks.saveContent).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'plain-id', content: 'unsaved content' }),
      expect.anything(),
    );
  });

  it('能力晚到时，仍选中的普通文件保留名称与目录草稿', async () => {
    runtimeState.value = null;
    savedFile = { ...savedFile, id: 'plain-id', configKind: 'plain' };
    const view = await openEditor();
    await view.findAll('input')[0].setValue('unsaved.conf');
    await view.findAll('input')[1].setValue('/unsaved');

    runtimeState.value = { framework: 'taf', mods: { frameworkFile: { supported: true, props: { format: 'taf' } } } };
    await flushPromises();

    expect(mocks.detail).toHaveBeenCalledTimes(1);
    expect(view.findAll('input')[0].element.value).toBe('unsaved.conf');
    expect(view.findAll('input')[1].element.value).toBe('/unsaved');
    await save();
    expect(mocks.update).toHaveBeenCalledWith(
      { appID: 'app-id', id: 'plain-id', name: 'unsaved.conf', mountDir: '/unsaved' },
      { interceptorErr: false, needRes: true },
    );
  });

  it('框架文件能力变为不支持时，不切换文件或丢弃编辑草稿', async () => {
    const plainFile = { ...savedFile, id: 'plain-id', configKind: 'plain', content: 'plain content' };
    mocks.list.mockImplementation(async () => ({ items: [savedFile, plainFile] }));
    mocks.detail.mockImplementation(async ({ id }: { id: string }) => ({
      item: id === 'plain-id' ? plainFile : savedFile,
    }));
    const view = await openEditor();
    await view.findAll('input')[0].setValue('unsaved.conf');
    await view.findAll('input')[1].setValue('/unsaved');
    runtimeState.value = {
      framework: 'taf',
      mods: { frameworkFile: { supported: false, reasonCode: 'invalid_stack' } },
    };
    await flushPromises();

    expect(view.getComponent({ name: 'ConfigFileList' }).props('activeId')).toBe('def-id');
    expect(mocks.detail).toHaveBeenCalledTimes(1);
    expect(view.findAll('input')[0].element.value).toBe('unsaved.conf');
    expect(view.findAll('input')[1].element.value).toBe('/unsaved');
  });

  it.each([
    ['a', true],
    ['a'.repeat(59) + '.yaml', true],
    ['a'.repeat(65), false],
    ['app/name.yaml', false],
    ['应用.conf', false],
    ['', false],
  ])('TAF 文件名校验：%s', async (name, valid) => {
    const view = await openEditor();
    await view.findAll('input')[0].setValue(name);
    await save();
    expect(mocks.update).toHaveBeenCalledTimes(valid ? 1 : 0);
  });

  it('TAF 使用 def 回填并仅提交变更，保存后展示服务端目录', async () => {
    const view = await openEditor();
    const inputs = view.findAll('input');
    expect(inputs[0].element.value).toBe('app.conf');
    expect(inputs[1].element.value).toBe('/config');
    await inputs[0].setValue('new.app.conf');
    await inputs[1].setValue('/new/');
    await save();

    expect(mocks.update).toHaveBeenCalledExactlyOnceWith(
      { appID: 'app-id', id: 'def-id', name: 'new.app.conf', mountDir: '/new' },
      { interceptorErr: false, needRes: true },
    );
    expect(view.text()).toContain('new.app.conf');
    expect(view.text()).toContain('/new');
    expect(mocks.appDetail).toHaveBeenCalledTimes(2);
  });

  it('tRPC 保存目录时不提交 name', async () => {
    mocks.appType = 'trpc';
    const view = await openEditor();
    expect(view.findAll('input')[0].attributes('disabled')).toBeDefined();
    await view.findAll('input')[1].setValue('/new');
    await save();
    expect(mocks.update).toHaveBeenCalledExactlyOnceWith(
      { appID: 'app-id', id: 'def-id', mountDir: '/new' },
      { interceptorErr: false, needRes: true },
    );
  });

  it('元信息与策略通过一次请求保存，随后刷新列表、详情及应用信息', async () => {
    const view = await openEditor();
    await view.findAll('input')[0].setValue('renamed.conf');
    await view.findAll('input')[1].setValue('/new/');
    view.getComponent(Radio.Group).vm.$emit('update:modelValue', 'overwrite');
    await flushPromises();
    await save();

    expect(mocks.update).toHaveBeenCalledExactlyOnceWith(
      { appID: 'app-id', id: 'def-id', name: 'renamed.conf', mountDir: '/new', envContentStrategy: 'overwrite' },
      { interceptorErr: false, needRes: true },
    );
    expect(mocks.list).toHaveBeenCalledTimes(2);
    expect(mocks.detail).toHaveBeenCalledTimes(2);
    expect(mocks.appDetail).toHaveBeenCalledTimes(2);
    expect(view.text()).toContain('renamed.conf');
    expect(view.text()).toContain('/new');
    expect(mocks.message).toHaveBeenCalledExactlyOnceWith({ theme: 'success', message: '操作成功' });
  });

  it('联合保存失败后保留输入，重试仍一次提交元信息与策略', async () => {
    const view = await openEditor();
    await view.findAll('input')[0].setValue('renamed.conf');
    await view.findAll('input')[1].setValue('/new/');
    view.getComponent(Radio.Group).vm.$emit('update:modelValue', 'overwrite');
    await flushPromises();
    mocks.update.mockRejectedValueOnce({ error: { message: 'save failed' } });
    await save();

    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.message).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ theme: 'error', message: expect.objectContaining({ overview: 'save failed' }) }),
    );
    expect(view.findAll('input')[0].element.value).toBe('renamed.conf');
    expect(view.findAll('input')[1].element.value).toBe('/new');
    expect(view.getComponent(Radio.Group).props('modelValue')).toBe('overwrite');
    expect(savedFile).toMatchObject({ name: 'app.conf', mountDir: '/config', envContentStrategy: 'overlay' });
    expect(mocks.list).toHaveBeenCalledTimes(1);
    expect(mocks.detail).toHaveBeenCalledTimes(1);
    expect(mocks.appDetail).toHaveBeenCalledTimes(1);
    await save();
    expect(mocks.update).toHaveBeenCalledTimes(2);
    expect(mocks.update.mock.calls[1][0]).toEqual({
      appID: 'app-id',
      id: 'def-id',
      name: 'renamed.conf',
      mountDir: '/new',
      envContentStrategy: 'overwrite',
    });
    expect(view.text()).toContain('renamed.conf');
    expect(mocks.list).toHaveBeenCalledTimes(2);
    expect(mocks.detail).toHaveBeenCalledTimes(2);
  });

  it.each(['framework', 'plain'])('%s 保存后刷新失败时，表单保留响应中的目录', async configKind => {
    savedFile.configKind = configKind;
    const view = await openEditor();
    await view.findAll('input')[1].setValue('/new/');
    mocks.list.mockRejectedValueOnce(new MockedApiError('list refresh unavailable'));
    await save();

    expect(mocks.update).toHaveBeenCalledExactlyOnceWith(
      { appID: 'app-id', id: 'def-id', mountDir: '/new' },
      { interceptorErr: false, needRes: true },
    );
    expect(view.findAll('input')[1].element.value).toBe('/new');
    expect(mocks.list).toHaveBeenCalledTimes(2);
    expect(mocks.detail).toHaveBeenCalledTimes(1);
  });

  it.each(['framework', 'plain'])('%s 文件目录冲突只提示一次，保留输入且可修改后重试', async configKind => {
    savedFile.configKind = configKind;
    const view = await openEditor();
    await view.findAll('input')[0].setValue('112x.conf');
    await view.findAll('input')[1].setValue('/data//conf/t4/');
    mocks.update.mockRejectedValueOnce({
      error: { details: [{ code: 'APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT' }] },
    });
    await save();
    expect(mocks.message).toHaveBeenCalledExactlyOnceWith({
      theme: 'error',
      delay: 8000,
      message: '目录 /data/conf/t4 下已存在文件 112x.conf，请修改文件名或挂载目录',
    });
    expect(view.findAll('input')[0].element.value).toBe('112x.conf');
    expect(view.findAll('input')[1].element.value).toBe('/data/conf/t4');
    await view.findAll('input')[1].setValue('/available');
    await save();
    expect(mocks.update).toHaveBeenCalledTimes(2);
    expect(savedFile.mountDir).toBe('/available');
  });

  it('普通文件创建冲突时不关闭侧栏或刷新当前内容', async () => {
    savedFile.configKind = 'plain';
    const view = await openEditor();
    view.getComponent({ name: 'ConfigFileList' }).vm.$emit('create');
    await flushPromises();
    const form = view.getComponent({ name: 'CreateConfigFileSideslider' });
    mocks.create.mockRejectedValue({
      error: { details: [{ code: 'APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT' }] },
    });
    form.vm.$emit('submit', { name: '112x.conf', mountDir: '/data/conf/t4', enableEnvVarRender: false });
    await flushPromises();
    expect(mocks.message).toHaveBeenCalledExactlyOnceWith({
      theme: 'error',
      delay: 8000,
      message: '目录 /data/conf/t4 下已存在文件 112x.conf，请修改文件名或挂载目录',
    });
    expect(form.props('isShow')).toBe(true);
    expect(mocks.list).toHaveBeenCalledTimes(1);
  });

  it('创建成功但列表刷新失败后，保留原环境内容并向原环境保存', async () => {
    savedFile = {
      ...savedFile,
      isUnifiedConfig: false,
      envContentStrategy: 'overwrite',
      editableContentField: 'content',
      content: 'default-content',
      currentVersion: 1,
    };
    mocks.detail.mockImplementation(async ({ envName }: { envName?: string }) => ({
      item:
        envName === 'prod'
          ? { ...savedFile, fileId: 'prod-file-id', content: 'prod-content', hasEnvInstance: true }
          : savedFile,
    }));
    const view = await openEditor();
    const environment = view.getComponent({ name: 'EnvPerspectiveSelect' });
    environment.vm.$emit('change', 'prod');
    await flushPromises();
    expect(environment.props('modelValue')).toBe('prod');
    expect(mocks.editorValue).toBe('prod-content');

    view.getComponent({ name: 'ConfigFileList' }).vm.$emit('create');
    await flushPromises();
    mocks.create.mockResolvedValueOnce({ item: { id: 'new-def-id' } });
    mocks.list.mockRejectedValueOnce(new MockedApiError('list refresh unavailable'));
    const form = view.getComponent({ name: 'CreateConfigFileSideslider' });
    form.vm.$emit('submit', { name: 'new.conf', mountDir: '/new', enableEnvVarRender: false });
    await flushPromises();

    expect(mocks.list).toHaveBeenCalledTimes(2);
    expect(form.props('loading')).toBe(false);
    expect(environment.props('modelValue')).toBe('prod');
    expect(view.getComponent({ name: 'ConfigFileList' }).props('activeId')).toBe('def-id');
    expect(mocks.editorValue).toBe('prod-content');
    expect(mocks.detail).toHaveBeenCalledTimes(2);

    await view.get('[data-section="文件内容"]').trigger('click');
    mocks.editorValue = 'prod-content-edited';
    await save();
    const saveDialog = view.getComponent({ name: 'SaveVersionConfirmDialog' });
    expect(saveDialog.props('isShow')).toBe(true);
    mocks.saveContent.mockResolvedValueOnce({ content: 'prod-content-edited', currentVersion: 2 });
    saveDialog.vm.$emit('confirm', 'update prod');
    await flushPromises();

    expect(mocks.saveContent).toHaveBeenCalledExactlyOnceWith(
      {
        appID: 'app-id',
        id: 'def-id',
        content: 'prod-content-edited',
        currentVersion: 1,
        description: 'update prod',
      },
      { interceptorErr: false, needRes: true, queryParams: { envName: 'prod' } },
    );
  });
});
