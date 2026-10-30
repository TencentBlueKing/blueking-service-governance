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

import { reactive } from 'vue';

import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DevModeForm from '~/pages/application/detail/app-config/components/dev-mode-form.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { PublicView } from '~/@types/v1/app';

const mocks = vi.hoisted(() => ({ getDevMode: vi.fn() }));
let state: { appID: string; runtime: null | PublicView };

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => state }));
vi.mock('~/api/modules/v1', () => ({
  AppSpecService: { getEnvEffectiveAppSpecDevMode: mocks.getDevMode },
}));
vi.mock('~/pages/application/detail/app-config/components/dev-mode-dialog.vue', () => ({
  default: { name: 'DevModeDialog', template: '<div />' },
}));
vi.mock('~/pages/application/detail/app-config/components/dev-mode-steps.vue', () => ({
  default: { name: 'DevModeSteps', template: '<div>publish steps</div>' },
}));

enableAutoUnmount(afterEach);
beforeEach(() => {
  vi.resetAllMocks();
  state = reactive({ appID: 'app-id', runtime: null as null | PublicView });
  mocks.getDevMode.mockResolvedValue({ enabled: false });
});

function mountForm() {
  return shallowMount(DevModeForm, {
    props: { currentEnv: { name: 'dev', type: 'development', isDefault: false, isModified: false } },
    global: { mocks: i18nGlobalMocks, renderStubDefaultSlot: true },
  });
}

describe('开发模式能力接入', () => {
  it('能力未知或明确不支持时不请求环境配置，能力晚到后加载当前环境', async () => {
    mocks.getDevMode.mockResolvedValue({ enabled: true });
    const wrapper = mountForm();
    await flushPromises();
    expect(mocks.getDevMode).not.toHaveBeenCalled();
    state.runtime = { mods: { devMode: { supported: false, reasonCode: 'incomplete_language' } } };
    await flushPromises();
    expect(mocks.getDevMode).not.toHaveBeenCalled();

    state.runtime = { mods: { devMode: { supported: true } } };
    await flushPromises();
    expect(mocks.getDevMode).toHaveBeenCalledExactlyOnceWith({ appID: 'app-id', envName: 'dev' });
    expect(wrapper.findComponent({ name: 'DevModeSteps' }).exists()).toBe(true);
  });

  it('切换环境后忽略旧环境迟到的已开启结果', async () => {
    let resolvePrevious!: (value: { enabled: boolean }) => void;
    mocks.getDevMode.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolvePrevious = resolve;
        }),
    );
    state.runtime = { mods: { devMode: { supported: true } } };
    const wrapper = mountForm();
    await flushPromises();
    await wrapper.setProps({ currentEnv: { name: 'test', type: 'test', isDefault: false, isModified: false } });
    await flushPromises();
    resolvePrevious({ enabled: true });
    await flushPromises();
    expect(mocks.getDevMode).toHaveBeenLastCalledWith({ appID: 'app-id', envName: 'test' });
    expect(wrapper.findComponent({ name: 'DevModeSteps' }).exists()).toBe(false);
  });
});
