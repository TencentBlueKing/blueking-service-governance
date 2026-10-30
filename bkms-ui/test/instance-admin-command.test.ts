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

import { shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import InstanceBatchToolbar from '~/pages/application/detail/deploy/instance-list/components/instance-batch-toolbar.vue';

import type { PublicView } from '~/@types/v1/app';

const state = vi.hoisted(() => ({ runtime: null as null | PublicView }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => state }));
vi.mock('vue-i18n', async () => (await import('./helpers/mock-i18n')).i18nMockFactory());

function mountToolbar() {
  return shallowMount(InstanceBatchToolbar, {
    props: { canAdjustWeight: false, canGrayDeploy: false, isAllInstancesSelected: false, selectedCount: 1 },
    global: {
      mocks: { $t: (key: string) => key },
      directives: { 'bk-tooltips': {} },
      stubs: {
        Button: { emits: ['click'], template: `<button @click="$emit('click')"><slot /></button>` },
      },
    },
  });
}

describe('实例批量操作：管理命令能力', () => {
  it('当新应用支持管理命令时，应展示并触发管理命令操作', async () => {
    state.runtime = { type: 'bkmsApp', framework: 'taf', mods: { adminCommand: { supported: true } } };
    const wrapper = mountToolbar();
    const button = wrapper.findAll('button').find(item => item.text() === '管理命令');
    expect(button).toBeDefined();
    await button!.trigger('click');
    expect(wrapper.emitted('admin-command')).toHaveLength(1);
    wrapper.unmount();
  });

  it.each([null, { mods: { adminCommand: { supported: false, reasonCode: 'unsupported_stack' } } }] as const)(
    '当能力尚未加载或不受支持时，应隐藏管理命令并保留通用监控操作',
    runtime => {
      state.runtime = runtime;
      const wrapper = mountToolbar();
      expect(wrapper.text()).not.toContain('管理命令');
      expect(wrapper.text()).toContain('监控');
      wrapper.unmount();
    },
  );
});
