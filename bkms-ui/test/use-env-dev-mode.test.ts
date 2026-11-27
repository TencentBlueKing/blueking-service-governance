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

import { effectScope, nextTick, reactive, ref } from 'vue';
import type { EffectScope } from 'vue';

import { flushPromises } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useEnvDevMode } from '~/composables/use-env-dev-mode';

import type { PublicView } from '~/@types/v1/app';

const mocks = vi.hoisted(() => ({ getDevMode: vi.fn() }));
let state: { appID: string; runtime: null | PublicView };

vi.mock('~/api/modules/v1', () => ({
  AppSpecService: { getEnvEffectiveAppSpecDevMode: mocks.getDevMode },
}));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => state }));

const SUPPORTED_RUNTIME = { mods: { devMode: { supported: true } } } as unknown as PublicView;
let scopes: EffectScope[] = [];

function run(
  envName: { value: string | undefined },
  envType?: { value: string | undefined },
  skip?: { value: boolean },
) {
  const scope = effectScope();
  scopes.push(scope);
  const result = scope.run(() =>
    useEnvDevMode({
      envName: () => envName.value,
      envType: () => envType?.value,
      skip: () => skip?.value ?? false,
    }),
  );
  if (!result) throw new Error('composable 未初始化');
  return result;
}

beforeEach(() => {
  vi.resetAllMocks();
  state = reactive({ appID: 'app-id', runtime: SUPPORTED_RUNTIME }) as typeof state;
  mocks.getDevMode.mockResolvedValue({ enabled: false });
});

afterEach(() => {
  scopes.forEach(scope => scope.stop());
  scopes = [];
});

describe('useEnvDevMode：环境生效的开发模式', () => {
  it('能力未知或明确不支持时不请求，能力晚到后按当前环境加载', async () => {
    mocks.getDevMode.mockResolvedValue({ enabled: true });
    state.runtime = null;
    const envName = ref<string | undefined>('dev');
    const { isDevModeEnabled } = run(envName);
    await flushPromises();
    expect(mocks.getDevMode).not.toHaveBeenCalled();
    expect(isDevModeEnabled.value).toBe(false);

    state.runtime = SUPPORTED_RUNTIME;
    await flushPromises();
    expect(mocks.getDevMode).toHaveBeenCalledExactlyOnceWith({ appID: 'app-id', envName: 'dev' });
    expect(isDevModeEnabled.value).toBe(true);
  });

  it('生产环境与显式跳过条件不请求，并视为未开启', async () => {
    const envName = ref<string | undefined>('prod');
    const envType = ref<string | undefined>('production');
    const skip = ref(false);
    const { isDevModeEnabled } = run(envName, envType, skip);
    await flushPromises();
    expect(mocks.getDevMode).not.toHaveBeenCalled();
    expect(isDevModeEnabled.value).toBe(false);

    envType.value = 'development';
    skip.value = true;
    await flushPromises();
    expect(mocks.getDevMode).not.toHaveBeenCalled();
  });

  it('切换环境后立即回落为未开启，不等新请求返回', async () => {
    mocks.getDevMode.mockResolvedValue({ enabled: true });
    const envName = ref<string | undefined>('dev');
    const { isDevModeEnabled } = run(envName);
    await flushPromises();
    expect(isDevModeEnabled.value).toBe(true);

    mocks.getDevMode.mockImplementationOnce(() => new Promise(() => undefined));
    envName.value = 'test';
    await nextTick();
    expect(isDevModeEnabled.value).toBe(false);
  });

  it('旧环境迟到的已开启结果不落到新环境', async () => {
    let resolvePrevious!: (value: { enabled: boolean }) => void;
    mocks.getDevMode.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolvePrevious = resolve;
        }),
    );
    const envName = ref<string | undefined>('dev');
    const { isDevModeEnabled } = run(envName);
    await flushPromises();

    envName.value = 'test';
    await nextTick();
    resolvePrevious({ enabled: true });
    await flushPromises();

    expect(mocks.getDevMode).toHaveBeenLastCalledWith({ appID: 'app-id', envName: 'test' });
    expect(isDevModeEnabled.value).toBe(false);
  });

  it('请求失败视为未开启', async () => {
    mocks.getDevMode.mockRejectedValue(new Error('boom'));
    const envName = ref<string | undefined>('dev');
    const { isDevModeEnabled } = run(envName);
    await flushPromises();
    expect(isDevModeEnabled.value).toBe(false);
  });
});
