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

import { flushPromises, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import FileFormSideslider from '~/pages/application/template/helm-chart/file-form-sideslider.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type * as BkuiVue from 'bkui-vue';

const mocks = vi.hoisted(() => ({ infoBox: vi.fn() }));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<typeof BkuiVue>()),
  InfoBox: mocks.infoBox,
}));
vi.mock('~/pages/application/template/helm-chart/bcsp-config-selector.vue', () => ({
  default: { template: '<div />' },
}));

let wrapper: ReturnType<typeof shallowMount<typeof FileFormSideslider>> | undefined;

beforeEach(() => {
  vi.resetAllMocks();
  mocks.infoBox.mockImplementation(({ onCancel }: { onCancel: () => void }) => onCancel());
});
afterEach(() => wrapper?.unmount());

async function clickButton(label: string) {
  const button = wrapper!.findAll('button').find(item => item.text() === label);
  expect(button).toBeDefined();
  await button!.trigger('click');
  await flushPromises();
}

async function openEditor() {
  wrapper = shallowMount(FileFormSideslider, {
    props: {
      baseFileOptions: [],
      currentFile: { name: 'values.yaml', fileType: 'normal', contentSourceType: 'local' },
      isEdit: true,
      visible: true,
      workspaceName: 'space',
    },
    global: {
      mocks: i18nGlobalMocks,
      directives: { 'bk-tooltips': {} },
      stubs: {
        // 只替代侧栏动画；表单、校验、输入控件和 useLeaveConfirm 均使用真实实现。
        Sideslider: { name: 'Sideslider', template: '<div><slot /><slot name="footer" /></div>' },
        Form: false,
        FormItem: false,
        Input: false,
        Button: false,
      },
      renderStubDefaultSlot: true,
    },
  });
  wrapper.getComponent({ name: 'Sideslider' }).vm.$emit('shown');
  await flushPromises();
  return wrapper;
}

describe('Helm 文件编辑表单', () => {
  it('允许含点号的 64 位文件名，并提交本地文件表单', async () => {
    const view = await openEditor();
    const name = `${'a'.repeat(59)}.yaml`;
    expect(view.get('input').attributes('maxlength')).toBe('64');
    await view.get('input').setValue(name);
    await clickButton('保存');

    expect(view.emitted('submit')).toEqual([
      [expect.objectContaining({ name, fileType: 'normal', contentSourceType: 'local' })],
    ]);
    expect(view.emitted('submit')?.[0]?.[0]).not.toHaveProperty('bscpConfig');
  });

  it.each(['a'.repeat(65), 'app/name.yaml'])('拒绝不合法的文件名：%s', async name => {
    const view = await openEditor();
    await view.get('input').setValue(name);
    await clickButton('保存');

    expect(view.emitted('submit')).toBeUndefined();
    expect(view.text()).toContain('文件名仅支持数字、字母、下划线(_)、中划线(-)、点(.)，长度 1-64 之间');
  });

  it('保存未关闭侧栏时保留未保存提醒，关闭并重开后清除提醒', async () => {
    const view = await openEditor();
    await view.get('input').setValue('renamed.yaml');
    await clickButton('保存');
    expect(view.emitted('submit')).toHaveLength(1);

    // 父级保存失败时保持侧栏打开，取消离开确认后输入仍保留。
    await clickButton('取消');
    expect(mocks.infoBox).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ content: '离开将会导致未保存信息丢失' }),
    );
    expect(view.emitted('cancel')).toBeUndefined();
    expect(view.get('input').element.value).toBe('renamed.yaml');

    await view.setProps({ visible: false });
    view.getComponent({ name: 'Sideslider' }).vm.$emit('hidden');
    await flushPromises();
    await view.setProps({ visible: true });
    view.getComponent({ name: 'Sideslider' }).vm.$emit('shown');
    await flushPromises();
    mocks.infoBox.mockClear();
    expect(view.get('input').element.value).toBe('values.yaml');

    await clickButton('取消');
    expect(mocks.infoBox).not.toHaveBeenCalled();
    expect(view.emitted('cancel')).toHaveLength(1);
  });
});
