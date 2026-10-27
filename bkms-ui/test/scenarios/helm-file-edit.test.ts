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

import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import HelmFileList from '~/pages/application/template/helm-chart/helm-file-list.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { CreateDefInput, DefDetailObj } from '~/@types/v1/app-config-file-defs';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  detail: vi.fn(),
  update: vi.fn(),
  create: vi.fn(),
  message: vi.fn(),
  bscp: vi.fn(),
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'app-id', appType: 'helm' }) }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'space' }) }));
vi.mock('~/composables/use-leave-confirm', () => ({ default: () => ({ confirmBox: async () => true }) }));
vi.mock('~/api/modules/bkmsserver', () => ({ ApiServerService: { GetBSCPConfig: mocks.bscp } }));
vi.mock('~/api/modules/v1', () => ({
  AppConfigFileDefsService: {
    listDefaultFilesWithDef: mocks.list,
    getAppConfigFileDefDetail: mocks.detail,
    updateHelmFile: mocks.update,
    createAppConfigFileDef: mocks.create,
  },
}));
vi.mock('bkui-vue', () => {
  const container = { template: '<div><slot /></div>' };
  return {
    Button: container,
    Exception: container,
    Loading: container,
    Message: mocks.message,
    OverflowTitle: container,
    PopConfirm: container,
    Tag: container,
    Tree: {
      props: ['data', 'children'],
      template: '<div><slot v-for="item in data" name="node" v-bind="item" /></div>',
    },
  };
});
vi.mock('bkui-vue/lib/icon', () => ({
  Del: { template: '<button />' },
  InfoLine: { template: '<span />' },
  Plus: { template: '<span />' },
  EditLine: { template: '<button data-edit>Edit</button>' },
}));
vi.mock('~/pages/application/template/helm-chart/file-form-sideslider.vue', () => ({
  default: { name: 'FileFormSideslider', props: ['visible', 'currentFile', 'isEdit', 'loading'], template: '<div />' },
}));

let wrapper: ReturnType<typeof mount<typeof HelmFileList>> | undefined;
const binding = { bizID: '2', serviceID: '15', id: '301' };
let savedFile: DefDetailObj;
type FileFormData = Omit<CreateDefInput, 'configKind'>;

beforeEach(() => {
  vi.resetAllMocks();
  savedFile = {
    id: 'def-id',
    fileId: 'file-id',
    name: 'values.yaml',
    fileType: 'normal',
    contentSourceType: 'local',
    configKind: 'framework',
    currentVersion: 3,
    content: 'replicas: 1',
  };
  mocks.list.mockImplementation(async () => ({ items: [savedFile] }));
  mocks.detail.mockImplementation(async () => ({ item: savedFile }));
  mocks.bscp.mockResolvedValue({ content: 'replicas: 1' });
  mocks.update.mockImplementation(async (input: { name: string }) => {
    savedFile = { ...savedFile, name: input.name };
    return { item: savedFile };
  });
});
afterEach(() => {
  wrapper?.unmount();
});

function formData(): FileFormData {
  return {
    name: 'values-prod.yaml',
    fileFormat: 'yaml',
    fileType: 'normal',
    contentSourceType: 'local',
    baseAppConfigFileId: 'base-file-id',
    bscpConfig: { ...binding },
    description: 'update binding',
  };
}

async function openEditor() {
  wrapper = mount(HelmFileList, {
    global: {
      mocks: i18nGlobalMocks,
      directives: { 'bk-tooltips': {} },
      stubs: {
        Tree: false,
        EditLine: false,
        FileFormSideslider: false,
        FlexRow: { template: '<div><slot name="left" /><slot name="right" /></div>' },
      },
      renderStubDefaultSlot: true,
    },
  });
  await flushPromises();
  await wrapper.get('[data-edit]').trigger('click');
  await flushPromises();
  return wrapper.getComponent({ name: 'FileFormSideslider' });
}

describe('Helm 文件保存', () => {
  it.each([
    ['normal', 'local'],
    ['overlay', 'local'],
    ['normal', 'bscp'],
    ['overlay', 'bscp'],
  ])('编辑 %s / %s 时使用 def ID，并按原文件类型提交字段', async (fileType, contentSourceType) => {
    savedFile = { ...savedFile, fileType, contentSourceType, bscpConfig: binding, baseAppConfigFileId: 'base-file-id' };
    const form = await openEditor();
    form.vm.$emit('submit', formData());
    await flushPromises();

    expect(mocks.update).toHaveBeenCalledWith(
      {
        appID: 'app-id',
        id: 'def-id',
        name: 'values-prod.yaml',
        currentVersion: 3,
        description: 'update binding',
        ...(fileType === 'overlay' ? { baseAppConfigFileId: 'base-file-id' } : {}),
        ...(contentSourceType === 'bscp' ? { bscpConfig: binding } : {}),
      },
      { interceptorErr: false, needRes: true },
    );
    expect(form.props('visible')).toBe(false);
    expect(wrapper?.emitted('file-list-change')?.at(-1)?.[0]).toEqual([savedFile]);
    expect(savedFile.currentVersion).toBe(3);
  });

  it('保存使用打开编辑时的版本，后续列表刷新不改变并发检查基准', async () => {
    const form = await openEditor();
    savedFile = { ...savedFile, currentVersion: 4 };
    await wrapper?.vm.fetchFileList();
    form.vm.$emit('submit', formData());
    await flushPromises();
    expect(mocks.update.mock.calls[0][0].currentVersion).toBe(3);
  });

  it.each(['base', 'bscp'])('修改 %s 绑定后展示服务端返回的新版本', async changedField => {
    savedFile = {
      ...savedFile,
      fileType: 'overlay',
      contentSourceType: 'bscp',
      baseAppConfigFileId: 'base-file-id',
      bscpConfig: binding,
    };
    const form = await openEditor();
    const data = formData();
    if (changedField === 'base') data.baseAppConfigFileId = 'new-base-file-id';
    else data.bscpConfig = { ...binding, id: '302' };
    mocks.update.mockImplementationOnce(async () => {
      savedFile = { ...savedFile, ...data, currentVersion: 4 };
      return { item: savedFile };
    });
    form.vm.$emit('submit', data);
    await flushPromises();
    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.update.mock.calls[0][0]).toMatchObject({
      name: data.name,
      baseAppConfigFileId: data.baseAppConfigFileId,
      bscpConfig: data.bscpConfig,
    });
    expect(wrapper?.emitted('file-detail-change')?.at(-1)?.[0]).toMatchObject({
      fileInfo: { name: data.name, currentVersion: 4 },
    });
  });

  it.each([
    ['APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT', '当前应用已存在同名文件'],
    ['APP_CONFIG_FILE_VERSION_CONFLICT', '当前配置已被他人更新。为避免数据被覆盖，请刷新页面获取最新版本后重新编辑。'],
  ])('编辑遇到 %s 时仅提示一次并保留表单', async (code, message) => {
    const form = await openEditor();
    mocks.update.mockRejectedValue({ error: { details: [{ code }] } });
    form.vm.$emit('submit', formData());
    await flushPromises();

    expect(mocks.message).toHaveBeenCalledExactlyOnceWith({ theme: 'error', message });
    expect(form.props('visible')).toBe(true);
    expect(form.props('loading')).toBe(false);
    expect(mocks.list).toHaveBeenCalledTimes(1);
    expect(savedFile.name).toBe('values.yaml');
  });

  it('添加同名文件时保留弹窗并给出业务提示', async () => {
    const form = await openEditor();
    const add = wrapper!
      .findAll('div')
      .find(node => node.classes().includes('cursor-pointer') && node.text() === '添加文件');
    expect(add).toBeDefined();
    await add!.trigger('click');
    mocks.create.mockRejectedValue({
      error: { details: [{ code: 'APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT' }] },
    });
    form.vm.$emit('submit', formData());
    await flushPromises();
    expect(mocks.create).toHaveBeenCalledWith(expect.any(Object), { interceptorErr: false, needRes: true });
    expect(mocks.message).toHaveBeenCalledExactlyOnceWith({ theme: 'error', message: '当前应用已存在同名文件' });
    expect(form.props('visible')).toBe(true);
  });

  it('其他保存错误保留后端原因，不显示成功或刷新列表', async () => {
    const form = await openEditor();
    mocks.update.mockRejectedValue({ error: { message: 'BSCP configuration is invalid' } });
    form.vm.$emit('submit', formData());
    await flushPromises();
    expect(mocks.message).toHaveBeenCalledTimes(1);
    expect(mocks.message.mock.calls[0][0].message.overview).toBe('BSCP configuration is invalid');
    expect(form.props('visible')).toBe(true);
    expect(mocks.list).toHaveBeenCalledTimes(1);
  });
});
