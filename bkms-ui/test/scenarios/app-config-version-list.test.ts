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
import VersionListSideslider from '~/pages/application/detail/app-config/version-list-sideslider.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';
import { TableStub } from '../helpers/mock-table';

import type { EnvInstanceObj } from '~/@types/v1/app-config-file-defs';

const mocks = vi.hoisted(() => ({ listVersions: vi.fn() }));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('@blueking/table', async () => (await import('../helpers/mock-table')).tableMockFactory());
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'app-test' }) }));
vi.mock('~/api/modules/v1', () => ({
  AppConfigFilesService: { listAppConfigFileVersions: mocks.listVersions },
}));

// 保留真实侧栏数据流与操作插槽，仅替换 UI 容器及无关弹窗。
vi.mock('bkui-vue', () => {
  const container = { template: '<div><slot /></div>' };
  return {
    Button: {
      props: { disabled: Boolean },
      template: '<button :disabled="disabled"><slot /></button>',
    },
    Divider: container,
    PopConfirm: container,
    Popover: container,
    Radio: { Group: { name: 'RadioGroup', template: '<div><slot /></div>' }, Button: container },
    SearchSelect: { name: 'SearchSelect', template: '<div />' },
    Sideslider: container,
    Tag: container,
  };
});
vi.mock('~/components/overflow-ctrl.vue', () => ({ default: { template: '<div><slot /></div>' } }));
vi.mock('~/components/table-exception.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/app-config/components/rollback-confirm-dialog.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/version-compare-dialog.vue', () => ({
  default: { template: '<div />' },
}));

type VersionListWrapper = ReturnType<typeof mount<typeof VersionListSideslider>>;

function instances(type = 'normal'): EnvInstanceObj[] {
  return [
    { fileId: 'default-file', envName: '', currentVersion: 24, type: 'normal' },
    { fileId: 'dev-file', envName: 'dev', currentVersion: 12, type },
  ];
}
let wrapper: VersionListWrapper;

async function openVersionList(props = {}) {
  wrapper = mount(VersionListSideslider, {
    props: {
      visible: false,
      configFileDefId: 'def-test',
      currentEnvName: 'dev',
      defaultFileId: 'default-file',
      defaultVersion: 24,
      enableEnvConfig: true,
      envInstances: instances(),
      envList: [
        { id: 'dev-id', name: 'dev', displayName: 'Dev' },
        { id: 'empty-id', name: 'empty', displayName: 'Empty' },
      ],
      ...props,
    },
    global: {
      mocks: i18nGlobalMocks,
      directives: { bkloading: () => {}, 'bk-tooltips': () => {} },
    },
  });
  await wrapper.setProps({ visible: true });
  await flushPromises();
  mocks.listVersions.mockClear();
  return wrapper;
}

function rollbackButton(version: number) {
  const button = versionRow(version)
    .findAll('button')
    .find(item => item.text() === '回滚到此版本');
  if (!button) throw new Error(`Missing rollback button: ${version}`);
  return button;
}

function versionRow(version: number) {
  const row = wrapper
    .findAll('[data-testid="table-row"]')
    .find(item => item.text().match(/^V(\d+)/)?.[1] === String(version));
  if (!row) throw new Error(`Missing version row: ${version}`);
  return row;
}

describe('配置文件版本回滚类型限制', () => {
  beforeEach(() => {
    mocks.listVersions.mockReset();
    mocks.listVersions.mockResolvedValue({
      results: [
        { version: 12, type: 'overlay' },
        { version: 11, type: 'normal' },
        { version: 10, type: 'overlay' },
      ],
      count: 30,
    });
  });

  afterEach(() => wrapper?.unmount());

  it.each([
    ['overlay', 'normal', 11, 10],
    ['normal', 'overlay', 10, 11],
  ])('版本号不变而类型从 %s 改为 %s 时，按实时类型限制回滚', async (oldType, type, allowed, blocked) => {
    mocks.listVersions.mockResolvedValue({
      results: [
        { version: 12, type: oldType },
        { version: 11, type: 'normal' },
        { version: 10, type: 'overlay' },
      ],
      count: 30,
    });
    await openVersionList({ envInstances: instances(type) });

    expect(rollbackButton(allowed).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(blocked).attributes('disabled')).toBeDefined();
    expect(versionRow(12).text()).toContain('当前版本');
    expect(versionRow(12).findAll('button')).toHaveLength(0);
    expect(versionRow(12).text()).toContain('--');
  });

  it('实例刷新后版本号未变，按钮仍同步使用新类型', async () => {
    await openVersionList({ envInstances: instances('overlay') });
    expect(rollbackButton(11).attributes('disabled')).toBeDefined();

    await wrapper.setProps({ envInstances: instances('normal') });
    await flushPromises();

    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(10).attributes('disabled')).toBeDefined();
  });

  it('搜索、翻页结果不含当前版本时，不再补查历史类型', async () => {
    await openVersionList();
    mocks.listVersions.mockResolvedValue({ results: [{ version: 11, type: 'normal' }], count: 30 });
    wrapper
      .findComponent({ name: 'SearchSelect' })
      .vm.$emit('update:modelValue', [{ id: 'description', values: [{ id: 'test' }] }]);
    await flushPromises();

    expect(mocks.listVersions).toHaveBeenCalledTimes(1);
    expect(mocks.listVersions).toHaveBeenLastCalledWith(expect.objectContaining({ description: 'test', page: 1 }));
    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();

    wrapper.findComponent(TableStub).vm.$emit('page-value-change', 2);
    await flushPromises();

    expect(mocks.listVersions).toHaveBeenCalledTimes(2);
    expect(mocks.listVersions).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }));
    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
  });

  it('切换环境时匹配对应实例，无独立实例时使用默认文件类型', async () => {
    await openVersionList({ envInstances: instances('overlay') });
    expect(rollbackButton(11).attributes('disabled')).toBeDefined();

    wrapper.findComponent({ name: 'RadioGroup' }).vm.$emit('update:modelValue', 'empty-id');
    await flushPromises();

    expect(mocks.listVersions).toHaveBeenLastCalledWith(
      expect.objectContaining({ appConfigFileID: 'default-file', envName: 'empty' }),
    );
    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(10).attributes('disabled')).toBeDefined();

    wrapper.findComponent({ name: 'RadioGroup' }).vm.$emit('update:modelValue', '__default__');
    await flushPromises();
    expect(mocks.listVersions).toHaveBeenLastCalledWith(
      expect.objectContaining({ appConfigFileID: 'default-file', envName: '' }),
    );
    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(10).attributes('disabled')).toBeDefined();
  });

  it('实时类型缺失时不回退到历史类型', async () => {
    await openVersionList({ envInstances: instances().map(item => ({ ...item, type: undefined })) });

    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(10).attributes('disabled')).toBeUndefined();
  });

  it('Helm 按文件模式不增加类型限制', async () => {
    await openVersionList({ configFileDefId: undefined, currentFileId: 'helm-file', currentVersion: 12 });

    expect(rollbackButton(11).attributes('disabled')).toBeUndefined();
    expect(rollbackButton(10).attributes('disabled')).toBeUndefined();
  });
});
