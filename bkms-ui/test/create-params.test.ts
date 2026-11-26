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

import { flushPromises, mount, shallowMount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CardRadio from '~/components/card-radio.vue';
import BuildOutputHint from '~/pages/application/components/build-output-hint.vue';
import ImageModeSelect from '~/pages/application/components/image-mode-select.vue';
import ParamConfig from '~/pages/application/template/bkms-app/param-config.vue';
import { buildBKMSAppPayload, createBKMSAppForm } from '~/pages/application/template/bkms-app/use-create-app';

import { i18nGlobalMocks, i18nPlugin, I18nTStub } from './helpers/mock-i18n';

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

describe('平台通用构建产物目录与 Python 单阶段表单联动', () => {
  it('不支持平台通用构建时展示禁用卡片与暂不支持提示，并回退到仓库 Dockerfile', () => {
    const form = createBKMSAppForm('ws-1', 'trpc', 'cpp');
    form.buildConfig.repoBuildConfig.imageBuildMode = 'platform';
    const wrapper = shallowMount(ParamConfig, {
      props: { form, platformBuildSupported: false },
      global: {
        plugins: [createPinia(), i18nPlugin],
        mocks: i18nGlobalMocks,
        renderStubDefaultSlot: true,
        directives: { bkloading: {}, 'bk-xss-html': {} },
      },
    });

    expect(form.buildConfig.repoBuildConfig.imageBuildMode).toBe('repositoryDockerfile');
    expect(wrapper.findComponent(CardRadio).props('options')).toEqual(
      expect.arrayContaining([expect.objectContaining({ value: 'platform', disabled: true, disabledTip: '暂不支持' })]),
    );
    wrapper.unmount();
  });

  it('Go 与 Python 切换时保留已填产物目录，更新 placeholder 并按单阶段显隐构建镜像与复制示例', async () => {
    const form = reactive(createBKMSAppForm('ws-1', 'blank', 'go'));
    form.name = 'demo';
    form.id = 'demo-1';
    form.buildConfig.repoBuildConfig.imageBuildMode = 'platform';
    form.buildConfig.repoBuildConfig.platformBuildConfig.builderImage = 'golang:1.24';
    form.buildConfig.repoBuildConfig.platformBuildConfig.runnerImage = 'debian:12';
    form.buildConfig.repoBuildConfig.platformBuildConfig.artifactDir = ' /dist ';
    form.buildConfig.repoBuildConfig.platformBuildConfig.extraFiles = ['certs/'];
    const clearValidate = vi.fn();

    const wrapper = shallowMount(ParamConfig, {
      props: { form, platformBuildSupported: true, requiresBuilderImage: true },
      global: {
        plugins: [createPinia(), i18nPlugin],
        mocks: i18nGlobalMocks,
        renderStubDefaultSlot: true,
        stubs: {
          Form: {
            template: '<div><slot /></div>',
            setup(_, { expose }) {
              expose({ clearValidate, validate: async () => true });
              return {};
            },
          },
          StartupCommandInput: {
            setup(_, { expose }) {
              expose({ getValue: () => ({}), validate: async () => true });
              return () => null;
            },
          },
        },
        directives: { bkloading: {}, 'bk-xss-html': {} },
      },
    });
    await nextTick();

    const collectImageTypes = () => wrapper.findAllComponents(ImageModeSelect).map(c => c.props('type'));
    const collectPlaceholders = () => wrapper.findAll('input-stub').map(node => node.attributes('placeholder'));
    expect(collectImageTypes()).toEqual(['builder', 'runner']);
    expect(collectPlaceholders()).toContain('/out');
    expect(collectPlaceholders()).toContain('留空则使用平台默认：\ngo build -o ${BKMS_ARTIFACT_DIR}/demo .');
    expect(wrapper.html()).not.toContain('/out/demo');
    expect(wrapper.text()).toContain('打包额外文件');

    const artifactDirInput = wrapper.find('input-stub[placeholder="/out"]');
    await artifactDirInput.trigger('blur');
    expect(form.buildConfig.repoBuildConfig.platformBuildConfig.artifactDir).toBe('/dist');
    await artifactDirInput.trigger('clear');
    expect(clearValidate).toHaveBeenCalledWith('buildConfig.repoBuildConfig.platformBuildConfig.artifactDir');

    // 切到 Python 单阶段：保留已填 /dist，切换 placeholder，隐藏构建镜像与打包额外文件，提交清空残留字段
    form.language = 'python';
    await wrapper.setProps({ requiresBuilderImage: false });
    await nextTick();

    expect(form.buildConfig.repoBuildConfig.platformBuildConfig.artifactDir).toBe('/dist');
    expect(collectImageTypes()).toEqual(['runner']);
    expect(collectPlaceholders()).toContain('/app');
    expect(collectPlaceholders()).toContain('留空则自动识别 uv.lock / poetry.lock / requirements.txt 安装依赖');
    expect(wrapper.text()).not.toContain('打包额外文件');

    const pythonRuntime = {
      mods: { platformBuild: { supported: true, props: { requiresBuilderImage: false } } },
    };
    const payload = buildBKMSAppPayload(wrapper.vm.getValue(), pythonRuntime);
    expect(payload.buildConfig.repoBuildConfig?.platformBuildConfig).toMatchObject({
      artifactDir: '/dist',
      builderImage: '',
      runnerImage: 'debian:12',
      extraFiles: [],
    });
    wrapper.unmount();

    const goHint = mount(BuildOutputHint, {
      props: { appName: 'demo', requiresBuilderImage: true },
      global: { mocks: i18nGlobalMocks, components: { 'i18n-t': I18nTStub } },
    });
    expect(goHint.text()).toContain('go build -o ${BKMS_ARTIFACT_DIR}/demo .');
    expect(goHint.text()).not.toContain('/out/');
    goHint.unmount();

    const pyHint = mount(BuildOutputHint, {
      props: { appName: 'demo', requiresBuilderImage: false },
      global: { mocks: i18nGlobalMocks, components: { 'i18n-t': I18nTStub } },
    });
    expect(pyHint.find('code').exists()).toBe(false);
    pyHint.unmount();
  });
});
