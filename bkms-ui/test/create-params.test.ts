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

import { nextTick, reactive } from 'vue';

import { flushPromises, shallowMount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ParamConfig from '~/pages/application/template/bkms-app/param-config.vue';
import { createBKMSAppForm } from '~/pages/application/template/bkms-app/use-create-app';

import { i18nGlobalMocks, i18nPlugin } from './helpers/mock-i18n';

const suffix = vi.hoisted(() => vi.fn());
vi.mock('vue-i18n', async () => (await import('./helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1/app', () => ({ AppService: { getAppIDAutoSuffix: suffix } }));

function mountParameters(form: ReturnType<typeof createBKMSAppForm>) {
  return shallowMount(ParamConfig, {
    props: { form, platformBuildSupported: false },
    global: { plugins: [createPinia(), i18nPlugin], mocks: i18nGlobalMocks },
  });
}
beforeEach(() => suffix.mockReset());

describe('创建参数页往返', () => {
  it('从配置步骤返回、失败重试重挂载时保留应用ID和流水线参数', async () => {
    const form = createBKMSAppForm('ws-1', 'trpc', 'go');
    form.name = 'demo';
    form.id = 'demo-abc123';
    form.buildConfig.sourceType = 'pipeline';
    form.buildConfig.pipelineBuildConfig = { pipelineID: 'pipeline-1', params: { VERSION: 'v2' } };
    let wrapper = mountParameters(form);
    await nextTick();
    const saved = wrapper.vm.getValue();
    wrapper.unmount();
    wrapper = mountParameters(saved);
    await nextTick();
    expect(wrapper.vm.getValue()).toMatchObject({
      id: 'demo-abc123',
      buildConfig: {
        sourceType: 'pipeline',
        pipelineBuildConfig: { pipelineID: 'pipeline-1', params: { VERSION: 'v2' } },
      },
    });
    expect(suffix).not.toHaveBeenCalled();
    wrapper.unmount();
  });
  it('名称先于后缀接口填写时，后缀返回后同步完整ID', async () => {
    let resolveSuffix: (value: { suffix: string }) => void = () => undefined;
    suffix.mockReturnValue(
      new Promise(resolve => {
        resolveSuffix = resolve;
      }),
    );
    const form = reactive(createBKMSAppForm('ws-1', 'blank', 'go'));
    const wrapper = mountParameters(form);
    form.name = 'demo';
    await nextTick();
    resolveSuffix({ suffix: '-xyz' });
    await flushPromises();
    expect(suffix).toHaveBeenCalledWith({}, { needRes: true });
    expect(wrapper.vm.getValue().id).toBe('demo-xyz');
    wrapper.unmount();
  });
});
