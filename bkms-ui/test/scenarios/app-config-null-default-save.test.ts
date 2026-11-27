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
/**
 * 默认 section 未落库时，指定环境保存仍要带上相对表单默认值的差异。
 */
import { defineComponent, h, nextTick, ref } from 'vue';

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Lifecycle from '~/pages/application/detail/app-config/components/lifecycle.vue';
import UpdateStrategyForm from '~/pages/application/detail/app-config/components/update-strategy-form.vue';
import { useAppDetail } from '~/stores/app-detail';

import { i18nGlobalMocks, i18nPlugin } from '../helpers/mock-i18n';
import { installVxeShims } from '../helpers/mock-table';

import type { ExtendedEnv } from '~/pages/application/detail/app-config/components/types';

const ENV: ExtendedEnv = { name: 'prod', isDefault: false, isModified: false };

const mocks = vi.hoisted(() => ({
  getUpdateDefault: vi.fn(),
  getUpdateEffective: vi.fn(),
  getUpdateOverride: vi.fn(),
  setUpdateEnv: vi.fn(),
  deleteUpdateEnv: vi.fn(),
  getLifecycleDefault: vi.fn(),
  getLifecycleEffective: vi.fn(),
  getLifecycleOverride: vi.fn(),
  setLifecycleEnv: vi.fn(),
  message: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<object>()),
  Message: mocks.message,
}));

vi.mock('~/api/modules/v1', () => ({
  AppSpecService: {
    getAppDefaultAppSpecUpdateStrategy: mocks.getUpdateDefault,
    getEnvEffectiveAppSpecUpdateStrategy: mocks.getUpdateEffective,
    getEnvAppSpecUpdateStrategy: mocks.getUpdateOverride,
    setAppDefaultAppSpecUpdateStrategy: vi.fn(),
    setEnvAppSpecUpdateStrategy: mocks.setUpdateEnv,
    deleteEnvAppSpecUpdateStrategy: mocks.deleteUpdateEnv,
    getAppDefaultAppSpecLifecycle: mocks.getLifecycleDefault,
    getEnvEffectiveAppSpecLifecycle: mocks.getLifecycleEffective,
    getEnvAppSpecLifecycle: mocks.getLifecycleOverride,
    setAppDefaultAppSpecLifecycle: vi.fn(),
    setEnvAppSpecLifecycle: mocks.setLifecycleEnv,
    deleteEnvAppSpecLifecycle: vi.fn(),
  },
}));

const harnessState: { load?: (env: ExtendedEnv) => Promise<void> } = {};

const LifecycleHarness = defineComponent({
  name: 'LifecycleHarness',
  setup() {
    return () => h(Lifecycle, { currentEnv: ENV });
  },
});

const StrategyHarness = defineComponent({
  name: 'StrategyHarness',
  setup() {
    const sectionRef = ref();
    const env = ref<ExtendedEnv>(ENV);
    harnessState.load = async (next: ExtendedEnv) => {
      env.value = next;
      await nextTick();
      await sectionRef.value?.handleEnvChange(next);
    };
    return () => h(UpdateStrategyForm, { ref: sectionRef, currentEnv: env.value });
  },
});

function renderWithApp(component: object) {
  const pinia = createPinia();
  setActivePinia(pinia);
  const store = useAppDetail();
  store.updateAppID('app-1');
  store.updateAppType('trpc');
  return render(component, {
    global: { plugins: [pinia, i18nPlugin], mocks: i18nGlobalMocks },
  });
}

function strategyInput(label: string) {
  const input = screen.getByText(label, { exact: true }).closest('.bk-form-item')?.querySelector('input');
  if (!input) throw new Error(`未找到更新策略字段 ${label}`);
  return input;
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.getUpdateDefault.mockResolvedValue(null);
  mocks.getUpdateEffective.mockResolvedValue(null);
  mocks.getUpdateOverride.mockRejectedValue({ status: 404 });
  mocks.setUpdateEnv.mockResolvedValue({});
  mocks.getLifecycleDefault.mockResolvedValue(null);
  mocks.getLifecycleEffective.mockResolvedValue(null);
  mocks.getLifecycleOverride.mockRejectedValue({ status: 404 });
  mocks.setLifecycleEnv.mockResolvedValue({});
});

installVxeShims();
Element.prototype.scrollIntoView = vi.fn();

describe('默认配置未落库时的指定环境保存', () => {
  it('更新策略应把相对 25% 基线的差异写入环境覆盖', async () => {
    renderWithApp(StrategyHarness);
    await harnessState.load?.(ENV);
    expect(await screen.findAllByText('25%')).toHaveLength(2);

    await userEvent.click(await screen.findByRole('button', { name: '编辑' }));
    const surge = strategyInput('最大超出数量 (maxSurge)');
    await userEvent.clear(surge);
    await userEvent.type(surge, '10%');
    await userEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() =>
      expect(mocks.setUpdateEnv).toHaveBeenCalledWith({
        appID: 'app-1',
        envName: 'prod',
        appSpecUpdateStrategy: {
          maxSurge: '10%',
          maxUnavailable: null,
        },
      }),
    );
  });

  it('生命周期应把相对 30 秒基线的优雅退出时间写入环境覆盖', async () => {
    renderWithApp(LifecycleHarness);
    expect(await screen.findByText('30 秒')).toBeTruthy();

    await userEvent.click(await screen.findByRole('button', { name: '编辑' }));
    await userEvent.click(document.querySelector('.bk-switcher') as HTMLElement);
    const grace = (await screen.findByText('优雅退出时间 (terminationGracePeriodSeconds)'))
      .closest('.relative')
      ?.querySelector('input');
    if (!grace) throw new Error('未找到优雅退出时间输入框');
    await userEvent.clear(grace);
    await userEvent.type(grace, '45');
    await userEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() =>
      expect(mocks.setLifecycleEnv).toHaveBeenCalledWith({
        appID: 'app-1',
        envName: 'prod',
        appSpecLifecycle: {
          preStop: {
            type: 'EXEC',
            exec: { command: [], sleepSeconds: 30 },
          },
          terminationGracePeriodSeconds: 45,
        },
      }),
    );
  });
});
