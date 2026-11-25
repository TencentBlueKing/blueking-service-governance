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

import { nextTick } from 'vue';

import { enableAutoUnmount, mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import UserSelector from '~/components/user-selector.vue';
import { useUserStore } from '~/stores/user';

vi.mock('@blueking/bk-user-selector', () => ({
  default: {
    name: 'BkUserSelector',
    props: ['modelValue', 'tenantId', 'apiBaseUrl'],
    emits: ['update:modelValue'],
    render: () => null,
  },
}));

enableAutoUnmount(afterEach);

afterEach(() => {
  vi.unstubAllEnvs();
});

describe('UserSelector', () => {
  it.each([
    { dev: true, template: 'https://{api_name}.example.test', expected: '/api-bk-user-selector' },
    { dev: true, template: '', expected: '' },
    { dev: false, template: ' https://{api_name}.example.test/ ', expected: 'https://bk-user-web.example.test/prod' },
    {
      dev: false,
      template: 'https://bk-user-web.example.test/prod/',
      expected: 'https://bk-user-web.example.test/prod',
    },
    { dev: false, template: '', expected: '' },
  ])('uses $expected with DEV=$dev and template=$template', ({ dev, template, expected }) => {
    vi.stubEnv('DEV', dev);
    vi.stubEnv('BK_API_URL_TMPL', template);
    const pinia = createPinia();
    const wrapper = mount(UserSelector, { global: { plugins: [pinia] } });

    expect(wrapper.getComponent({ name: 'BkUserSelector' }).props('apiBaseUrl')).toBe(expected);
  });

  it('prefers the explicit tenant, then the signed-in tenant, then default', async () => {
    const pinia = createPinia();
    const userStore = useUserStore(pinia);
    userStore.setUserInfo({ user_id: 'viewer', tenant_id: 'tenant-a' });
    const wrapper = mount(UserSelector, {
      props: { tenantId: 'tenant-b' },
      global: { plugins: [pinia] },
    });
    const selector = wrapper.getComponent({ name: 'BkUserSelector' });

    expect(selector.props('tenantId')).toBe('tenant-b');

    await wrapper.setProps({ tenantId: undefined });
    expect(selector.props('tenantId')).toBe('tenant-a');

    userStore.setUserInfo({ user_id: 'legacy-user' });
    await nextTick();
    expect(selector.props('tenantId')).toBe('default');
  });

  it('emits the selected user ID unchanged as a single-element array', () => {
    const pinia = createPinia();
    useUserStore(pinia).setUserInfo({ user_id: 'viewer', tenant_id: 'tenant-a' });
    const wrapper = mount(UserSelector, {
      props: { modelValue: ['original-id'] },
      attrs: { multiple: false },
      global: { plugins: [pinia] },
    });
    const selector = wrapper.getComponent({ name: 'BkUserSelector' });

    expect(selector.props('modelValue')).toBe('original-id');
    selector.vm.$emit('update:modelValue', 'selected-id');
    expect(wrapper.emitted('update:modelValue')).toEqual([[['selected-id']]]);
  });
});
