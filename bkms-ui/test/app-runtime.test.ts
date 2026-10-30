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

import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getAppDisplayKind, getAppDisplayLabel, isAppModelAppType } from '~/composables/app-type';
import { getFrameworkFileFormat, isRuntimeModSupported } from '~/composables/use-app-runtime';
import { useAppDetail } from '~/stores/app-detail';

import type { PublicView } from '~/@types/v1/app';

const mocks = vi.hoisted(() => ({ runtime: vi.fn(), detail: vi.fn() }));
vi.mock('~/api/modules/v1/app', () => ({ AppService: { getAppRuntimeMods: mocks.runtime } }));
vi.mock('~/api/modules/bkmsserver', () => ({ ApiServerService: { GetApp: mocks.detail } }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => {
    resolve = done;
  });
  return { promise, resolve };
}
const runtime = (framework = 'trpc'): PublicView => ({
  type: 'bkmsApp',
  framework,
  language: 'go',
  mods: { frameworkFile: { supported: true, props: { format: 'yaml' } } },
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.resetAllMocks();
});

describe('应用 runtime 状态', () => {
  it('合并同一应用的在途请求，保留真实应用类型', async () => {
    const store = useAppDetail();
    store.updateAppID('a');
    store.updateAppType('trpc');
    const pending = deferred<PublicView>();
    mocks.runtime.mockReturnValue(pending.promise);
    const first = store.fetchAppRuntimeMods();
    const second = store.fetchAppRuntimeMods();
    expect(store.runtimeLoading).toBe(true);
    expect(mocks.runtime).toHaveBeenCalledTimes(1);
    expect(mocks.runtime).toHaveBeenCalledWith({ appID: 'a' }, { interceptorErr: false });
    pending.resolve(runtime());
    await Promise.all([first, second]);
    expect(store.runtime?.framework).toBe('trpc');
    expect(store.appType).toBe('trpc');
    expect(store.runtimeLoading).toBe(false);
  });

  it('能力失败保留详情，允许重新请求恢复', async () => {
    const store = useAppDetail();
    store.updateAppID('a');
    store.appDetail = { id: 'a', name: 'app-a' };
    mocks.runtime.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(runtime());
    await store.fetchAppRuntimeMods();
    expect(store.runtimeError).toBe(true);
    expect(store.appDetail?.name).toBe('app-a');
    expect(isRuntimeModSupported(store.runtime, 'frameworkFile')).toBe(false);
    await store.fetchAppRuntimeMods();
    expect(store.runtimeError).toBe(false);
    expect(isRuntimeModSupported(store.runtime, 'frameworkFile')).toBe(true);
  });

  it('A→B→A 后清空旧能力，最初 A 的迟到响应不能覆盖本轮结果或加载状态', async () => {
    const store = useAppDetail();
    store.updateAppID('a');
    const stale = deferred<PublicView>();
    const current = deferred<PublicView>();
    mocks.runtime.mockReturnValueOnce(stale.promise).mockReturnValueOnce(current.promise);
    const staleFetching = store.fetchAppRuntimeMods();
    store.runtime = runtime();
    store.updateAppID('b');
    expect(store.runtime).toBeNull();
    expect(store.runtimeLoading).toBe(false);
    store.updateAppID('a');
    const currentFetching = store.fetchAppRuntimeMods();
    stale.resolve(runtime('taf'));
    await staleFetching;
    expect(store.runtime).toBeNull();
    expect(store.runtimeLoading).toBe(true);
    current.resolve(runtime('blank'));
    await currentFetching;
    expect(store.runtime?.framework).toBe('blank');
  });

  it('reset 后迟到响应不得恢复旧能力', async () => {
    const store = useAppDetail();
    store.updateAppID('a');
    const pending = deferred<PublicView>();
    mocks.runtime.mockReturnValueOnce(pending.promise);
    const fetching = store.fetchAppRuntimeMods();
    store.reset();
    pending.resolve(runtime());
    await fetching;
    expect(store.runtime).toBeNull();
    expect(store.runtimeError).toBe(false);
    expect(store.runtimeLoading).toBe(false);
  });
});

describe('能力与技术栈展示', () => {
  it('只启用后端明确支持的能力，并对未知参数安全读取', () => {
    expect(isRuntimeModSupported(undefined, 'devMode')).toBe(false);
    expect(isRuntimeModSupported({ mods: { devMode: { supported: false } } }, 'devMode')).toBe(false);
    expect(getFrameworkFileFormat(runtime())).toBe('yaml');
    expect(getFrameworkFileFormat({ mods: { frameworkFile: { supported: true, props: { format: 'taf' } } } })).toBe(
      'taf',
    );
    for (const props of [null, 'yaml', {}, { format: 'json' }]) {
      expect(getFrameworkFileFormat({ mods: { frameworkFile: { supported: true, props } } })).toBeUndefined();
    }
    expect(
      getFrameworkFileFormat({ mods: { frameworkFile: { supported: false, props: { format: 'yaml' } } } }),
    ).toBeUndefined();
  });

  it('新旧框架应用合并展示分类，blank 按语言显示但真实类型保留', () => {
    for (const framework of ['trpc', 'taf']) {
      expect(getAppDisplayKind({ type: framework })).toBe(framework);
      expect(getAppDisplayKind({ type: 'bkmsApp', framework })).toBe(framework);
    }
    for (const language of ['go', 'python', 'nodejs', 'cpp']) {
      expect(getAppDisplayKind({ type: 'bkmsApp', framework: 'blank', language })).toBe(language);
    }
    expect(getAppDisplayKind({ type: 'bkmsApp' })).toBe('bkmsApp');
    expect(getAppDisplayLabel('cpp')).toBe('C++');
    expect(getAppDisplayLabel('nodejs')).toBe('Node.js');
    expect(isAppModelAppType('bkmsApp')).toBe(true);
    expect(isAppModelAppType('helm')).toBe(false);
  });
});
