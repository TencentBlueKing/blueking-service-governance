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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useConfigFileDefs } from '~/pages/application/detail/app-config/use-config-file-defs';

import type { DefDetailObj } from '~/@types/v1/app-config-file-defs';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  detail: vi.fn(),
  instances: vi.fn(),
  update: vi.fn(),
  create: vi.fn(),
}));
vi.mock('~/api/modules/v1', () => ({
  AppConfigFileDefsService: {
    listDefaultFilesWithDef: mocks.list,
    getAppConfigFileDefDetail: mocks.detail,
    listAppConfigFileDefEnvInstances: mocks.instances,
    updateAppConfigFileDef: mocks.update,
    createAppConfigFileDef: mocks.create,
  },
}));

const file: DefDetailObj = {
  id: 'def-id',
  fileId: 'default-file-id',
  configKind: 'framework',
  name: 'app.conf',
  mountDir: '/old',
  currentVersion: 3,
  content: 'default content',
  envContentStrategy: 'overlay',
};

beforeEach(() => {
  vi.resetAllMocks();
  mocks.list.mockResolvedValue({ items: [file] });
  mocks.detail.mockResolvedValue({
    item: { ...file, fileId: 'env-file-id', currentVersion: 5, content: 'env content', hasEnvInstance: true },
  });
  mocks.instances.mockResolvedValue({ items: [] });
});

describe('配置文件元信息保存', () => {
  it('使用 def ID 保存并返回服务端元信息，不合并或刷新列表与详情', async () => {
    const state = useConfigFileDefs('app-id');
    await state.fetchDefs();
    await state.refreshActive('dev');
    const originalDefs = state.defs.value;
    const originalDetail = state.detail.value;
    mocks.update.mockResolvedValue({ item: { id: 'def-id', name: 'renamed.conf', mountDir: '/new' } });

    const result = await state.updateDef({ name: 'renamed.conf', mountDir: '/new/' }, { interceptorErr: false });

    expect(mocks.update).toHaveBeenCalledWith(
      { appID: 'app-id', id: 'def-id', name: 'renamed.conf', mountDir: '/new/' },
      { interceptorErr: false, needRes: true },
    );
    expect(result).toEqual({ id: 'def-id', name: 'renamed.conf', mountDir: '/new' });
    expect(state.defs.value).toBe(originalDefs);
    expect(state.detail.value).toBe(originalDetail);
    expect(mocks.list).toHaveBeenCalledTimes(1);
    expect(mocks.detail).toHaveBeenCalledTimes(1);
    expect(mocks.instances).toHaveBeenCalledTimes(1);
    expect(state.activeDef.value).toMatchObject({
      name: 'app.conf',
      mountDir: '/old',
      content: 'default content',
      currentVersion: 3,
      fileId: 'default-file-id',
    });
    expect(state.detail.value).toMatchObject({
      name: 'app.conf',
      mountDir: '/old',
      content: 'env content',
      currentVersion: 5,
      fileId: 'env-file-id',
    });
  });

  it('保存失败透传异常并保留已有列表与详情', async () => {
    const state = useConfigFileDefs('app-id');
    await state.fetchDefs();
    await state.refreshActive('dev');
    const originalDefs = state.defs.value;
    const originalDetail = state.detail.value;
    const error = { error: { message: 'save failed' } };
    mocks.update.mockRejectedValueOnce(error);
    await expect(state.updateDef({ envContentStrategy: 'overwrite' })).rejects.toBe(error);
    expect(state.defs.value).toBe(originalDefs);
    expect(state.detail.value).toBe(originalDetail);
  });

  it('创建失败透传冲突，不刷新列表或改变当前选中项', async () => {
    const state = useConfigFileDefs('app-id');
    await state.fetchDefs();
    const conflict = { error: { details: [{ code: 'APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT' }] } };
    mocks.create.mockRejectedValue(conflict);
    await expect(
      state.createPlain({
        name: 'app.conf',
        mountDir: '/old',
        enableEnvVarRender: false,
      }),
    ).rejects.toBe(conflict);
    expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ name: 'app.conf', mountDir: '/old' }), {
      interceptorErr: false,
      needRes: true,
    });
    expect(mocks.list).toHaveBeenCalledTimes(1);
    expect(state.activeDefID.value).toBe('def-id');
  });
});
