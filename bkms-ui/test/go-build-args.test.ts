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

import { defineComponent, ref } from 'vue';

import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import GoBuildArgs from '~/pages/application/components/go-build-args.vue';
import { useFrontendFeaturesStore } from '~/stores/frontend-features';

import { i18nGlobalMocks } from './helpers/mock-i18n';

const getFrontendFeatures = vi.hoisted(() => vi.fn());

vi.mock('vue-i18n', async () => (await import('./helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1/frontend-features', () => ({
  FrontendFeaturesService: { getFrontendFeatures },
}));

describe('GoBuildArgs feature flag', () => {
  beforeEach(() => {
    getFrontendFeatures.mockReset();
    vi.stubEnv('BK_GOLANG_PROXY_URL', 'https://goproxy.example.com');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  async function mountGoBuildArgs(enableGoProxy: boolean, initialArgs: Record<string, string> = {}) {
    const pinia = createPinia();
    setActivePinia(pinia);
    getFrontendFeatures.mockResolvedValue({ enableGoProxy });
    const store = useFrontendFeaturesStore();
    await store.getFrontendFeatures();

    const args = ref<Record<string, string>>({ ...initialArgs });
    const keyValueStub = defineComponent({
      name: 'KeyValue',
      props: {
        modelValue: { type: Object, default: undefined },
        valuePlaceholder: { type: [String, Object], default: undefined },
      },
      emits: ['update:modelValue'],
      template: '<div class="kv-stub" />',
    });

    const wrapper = mount(GoBuildArgs, {
      props: {
        language: 'go',
        modelValue: args.value,
        'onUpdate:modelValue': (val?: Record<string, string>) => {
          args.value = val ?? {};
          wrapper.setProps({ modelValue: args.value });
        },
      },
      global: {
        plugins: [pinia],
        mocks: i18nGlobalMocks,
        stubs: {
          KeyValue: keyValueStub,
          Button: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
          Share: true,
          'i18n-t': { template: '<span><slot /></span>' },
        },
      },
    });

    return { wrapper, args, keyValueStub };
  }

  it('hides GoProxy guide and custom placeholders when enableGoProxy is false', async () => {
    const { wrapper, keyValueStub } = await mountGoBuildArgs(false);

    expect(wrapper.find('a[href="https://goproxy.example.com"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('一键添加');
    expect(wrapper.findComponent(keyValueStub).props('valuePlaceholder')).toBeUndefined();
  });

  it('renders GoProxy guide, custom placeholders, and one-click add when enableGoProxy is true', async () => {
    const { wrapper, args, keyValueStub } = await mountGoBuildArgs(true, { GOPROXY: 'https://custom.example.com' });

    expect(wrapper.find('a[href="https://goproxy.example.com"]').exists()).toBe(true);
    expect(wrapper.findComponent(keyValueStub).props('valuePlaceholder')).toMatchObject({
      GOPROXY: expect.any(String),
      GOPRIVATE: expect.any(String),
      GOSUMDB: expect.any(String),
    });

    await wrapper.get('button').trigger('click');
    expect(args.value).toEqual({
      GOPROXY: 'https://custom.example.com',
      GOPRIVATE: '',
      GOSUMDB: '',
    });
  });
});
