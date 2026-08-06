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

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import FlexRow from '../src/components/flex-row.vue';

describe('FlexRow.vue', () => {
  it('renders correctly', () => {
    const wrapper = mount(FlexRow);
    expect(wrapper.exists()).toBe(true);
  });

  it('has two div elements', () => {
    const wrapper = mount(FlexRow);
    expect(wrapper.findAll('div')).toHaveLength(3); // One parent div and two child divs
  });

  it('renders left slot content', () => {
    const wrapper = mount(FlexRow, {
      slots: {
        left: '<span>Left Content</span>',
      },
    });
    expect(wrapper.find('div > div:first-child').html()).toContain('Left Content');
  });

  it('renders right slot content', () => {
    const wrapper = mount(FlexRow, {
      slots: {
        right: '<span>Right Content</span>',
      },
    });
    expect(wrapper.find('div > div:last-child').html()).toContain('Right Content');
  });

  it('has correct CSS classes', () => {
    const wrapper = mount(FlexRow);
    expect(wrapper.classes()).toContain('flex');
    expect(wrapper.classes()).toContain('items-center');
    expect(wrapper.classes()).toContain('place-content-between');
  });
});
