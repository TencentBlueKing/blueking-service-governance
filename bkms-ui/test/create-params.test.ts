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
import { buildBKMSAppPayload, createBKMSAppForm } from '~/pages/application/template/bkms-app/use-create-app';

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

describe('创建启动配置提交', () => {
  it.each(['go', 'python', 'nodejs'])('blank + %s 提交时清理首尾空白与空行，保留内容中的空格且不修改表单', language => {
    const form = createBKMSAppForm('ws-1', 'blank', language);
    form.appModelSpec.command = [' \tpython3 \n', '', ' \t ', ' -u '];
    form.appModelSpec.args = [' /app/my script.py ', '', '\t', ' --message=hello world '];
    form.appModelSpec.envVars = [{ key: 'STALE', value: 'value' }];
    const original = structuredClone(form);

    const payload = buildBKMSAppPayload(form, { framework: 'blank', language });

    expect(payload.appModelSpec).toEqual({
      command: ['python3', '-u'],
      args: ['/app/my script.py', '--message=hello world'],
      shCommand: '',
      envVars: [],
    });
    expect(form).toEqual(original);
  });

  it.each(['go', 'python', 'nodejs'])('blank + %s 原样提交 shell 脚本，清空互斥字段且不修改表单', language => {
    const form = createBKMSAppForm('ws-1', 'blank', language);
    const script = '  cd /app && printf "%s\\n" "$PORT" | cat\nexec ./server --port="$PORT"\n';
    form.appModelSpec.command = ['/old/server'];
    form.appModelSpec.args = ['--old'];
    form.appModelSpec.shCommand = script;
    form.appModelSpec.envVars = [{ key: 'STALE', value: 'value' }];
    const original = structuredClone(form);

    expect(buildBKMSAppPayload(form, { framework: 'blank', language }).appModelSpec).toEqual({
      command: [],
      args: [],
      shCommand: script,
      envVars: [],
    });
    expect(form).toEqual(original);
  });

  it('blank 未填写命令或参数时保留空数组，由镜像默认启动配置处理', () => {
    const form = createBKMSAppForm('ws-1', 'blank', 'python');
    form.appModelSpec.command = [' ', '\t'];
    form.appModelSpec.args = [''];

    expect(buildBKMSAppPayload(form, {}).appModelSpec).toMatchObject({ command: [], args: [], shCommand: '' });
  });

  it.each(['trpc', 'taf'])('%s 沿用原有启动配置提交内容', framework => {
    const form = createBKMSAppForm('ws-1', framework, 'cpp');
    form.appModelSpec.command = [' /app/server '];
    form.appModelSpec.args = [' argument with spaces '];
    form.appModelSpec.envVars = [{ key: 'PORT', value: '8000' }];

    expect(buildBKMSAppPayload(form, {}).appModelSpec).toEqual(form.appModelSpec);
  });
});
