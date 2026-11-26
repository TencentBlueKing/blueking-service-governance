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
 * composable 级测试：useAppConfigCommon 的模板加载守卫（M1 回归）
 *
 * 模板切换/初始化未完成（templateLoading 为 true）时 ensureTemplateReady 返回 false 并弹出提示，
 * 防止窗口期内提交把旧模板内容带入创建请求；加载完成后恢复放行且不再提示。
 *
 * stub/mock：bkui-vue 的 Message、vue-i18n、use-env-manager、bkmsserver API。
 */
import { defineComponent, h } from 'vue';

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  message: vi.fn(),
}));

vi.mock('bkui-vue', () => ({ Message: mocks.message }));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('~/composables/use-env-manager', () => ({
  default: () => ({ envList: { value: [] }, handleGetEnvList: vi.fn() }),
}));

vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: { ListEnvAvailableEnvVars: vi.fn() },
}));

import { useAppConfigCommon } from '~/pages/application/template/composables/use-app-config-common';

type AppConfigCommon = ReturnType<typeof useAppConfigCommon>;

function setupCommon(): AppConfigCommon {
  let exposed!: AppConfigCommon;
  const hostComponent = defineComponent({
    name: 'AppConfigCommonHost',
    setup() {
      exposed = useAppConfigCommon();
      return () => h('div');
    },
  });
  mount(hostComponent);
  return exposed;
}

describe('useAppConfigCommon 模板加载守卫', () => {
  beforeEach(() => {
    mocks.message.mockClear();
  });

  it('初始处于模板加载态：ensureTemplateReady 阻止提交并提示', () => {
    const common = setupCommon();
    expect(common.templateLoading.value).toBe(true);

    expect(common.ensureTemplateReady()).toBe(false);
    expect(mocks.message).toHaveBeenCalledTimes(1);
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'warning', message: '模板配置加载中，请稍后' });
  });

  it('模板加载完成后恢复放行且不提示', () => {
    const common = setupCommon();
    common.handleTemplateLoading(false);

    expect(common.templateLoading.value).toBe(false);
    expect(common.ensureTemplateReady()).toBe(true);
    expect(mocks.message).not.toHaveBeenCalled();
  });

  it('切换模板重新进入加载态时再次阻止提交（每次提示一次）', () => {
    const common = setupCommon();
    common.handleTemplateLoading(false);
    common.handleTemplateLoading(true);

    expect(common.ensureTemplateReady()).toBe(false);
    expect(mocks.message).toHaveBeenCalledTimes(1);
  });
});
