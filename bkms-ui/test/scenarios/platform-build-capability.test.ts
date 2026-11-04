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
import CardRadio from '~/components/card-radio.vue';
import EditBuilderConfig from '~/pages/application/detail/base-info/trpc/edit-builder-config.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { PublicView } from '~/@types/v1/app';

const mocks = vi.hoisted(() => ({ validate: vi.fn(), getWorkspace: vi.fn() }));
let state: { runtime: null | PublicView };

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => state }));
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'workspace' }) }));
vi.mock('~/api/modules/v1', () => ({ BkintegrationsBkciService: { getBkCIPipelineVariables: vi.fn() } }));
vi.mock('~/api/modules/v1/workspace', () => ({ WorkspaceService: { getWorkspace: mocks.getWorkspace } }));
vi.mock('~/composables/use-leave-confirm', () => ({
  default: () => ({ confirmBox: vi.fn(), forceCleanDirtyTag: (callback?: () => void) => callback?.() }),
}));

enableAutoUnmount(afterEach);
beforeEach(() => {
  vi.resetAllMocks();
  state = reactive({ runtime: null as null | PublicView });
  mocks.validate.mockResolvedValue(true);
  mocks.getWorkspace.mockResolvedValue({});
});

async function openEditor() {
  const builderData = platformConfig();
  const wrapper = shallowMount(EditBuilderConfig, {
    props: { type: 'codeRepository', builderData, isShow: false },
    global: {
      mocks: i18nGlobalMocks,
      renderStubDefaultSlot: true,
      stubs: {
        Sideslider: { template: '<div><slot /><slot name="footer" /></div>' },
        Form: {
          template: '<div><slot /></div>',
          setup: () => ({ validate: mocks.validate }),
        },
        Button: {
          props: ['disabled'],
          template: '<button :disabled="disabled"><slot /></button>',
        },
        GitSelector: true,
      },
      directives: { bkloading: {}, 'bk-tooltips': {} },
    },
  });
  await wrapper.setProps({ isShow: true });
  await flushPromises();
  return { wrapper, builderData };
}

function platformConfig() {
  return {
    sourceType: 'codeRepository',
    repoBuildConfig: {
      repoURL: 'repository',
      defaultBranch: 'main',
      dockerfile: '',
      imageBuildMode: 'platform',
      platformBuildConfig: {
        builderImage: 'builder:v1',
        runnerImage: 'runner:v1',
        extraFiles: ['data'],
        commands: { build: ['go build'], preBuild: [], runtimeEnv: [], start: './server' },
      },
    },
  };
}

describe('构建配置的平台通用构建能力', () => {
  it('不支持时保留禁用选项，打开侧栏不改写已有平台构建配置', async () => {
    state.runtime = { mods: { platformBuild: { supported: false } } };
    const { wrapper, builderData } = await openEditor();

    expect(builderData).toEqual(platformConfig());
    expect(wrapper.findComponent(CardRadio).props('options')).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ value: 'platform', disabled: true, disabledTip: '当前应用不支持平台通用构建' }),
        expect.objectContaining({ value: 'repositoryDockerfile' }),
      ]),
    );
  });

  it('能力晚到或刷新为不支持时保留原配置，并阻止保存', async () => {
    const { wrapper, builderData } = await openEditor();
    state.runtime = { mods: { platformBuild: { supported: true } } };
    await flushPromises();
    state.runtime = { mods: { platformBuild: { supported: false } } };
    await flushPromises();

    expect(builderData).toEqual(platformConfig());
    const save = wrapper.findAll('button').find(button => button.text() === '保存')!;
    expect(save.attributes('disabled')).toBeDefined();
    await save.trigger('click');
    expect(mocks.validate).not.toHaveBeenCalled();
    expect(wrapper.emitted('confirm')).toBeUndefined();
  });

  it('用户主动改为仓库 Dockerfile 后可以保存', async () => {
    state.runtime = { mods: { platformBuild: { supported: false } } };
    const { wrapper, builderData } = await openEditor();
    wrapper.findComponent(CardRadio).vm.$emit('update:modelValue', 'repositoryDockerfile');
    await flushPromises();
    builderData.repoBuildConfig.dockerfile = 'Dockerfile';
    const save = wrapper.findAll('button').find(button => button.text() === '保存')!;
    expect(save.attributes('disabled')).toBeUndefined();
    await save.trigger('click');
    await flushPromises();

    expect(mocks.validate).toHaveBeenCalledOnce();
    expect(wrapper.emitted('confirm')).toHaveLength(1);
    expect(builderData.repoBuildConfig).toMatchObject({
      imageBuildMode: 'repositoryDockerfile',
      dockerfile: 'Dockerfile',
    });
    expect(builderData.repoBuildConfig.platformBuildConfig).toBeUndefined();
  });
});
