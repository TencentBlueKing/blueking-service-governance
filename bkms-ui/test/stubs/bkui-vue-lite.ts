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

import { defineComponent } from 'vue';

/**
 * 单测用轻量 bkui-vue stub：只覆盖 RepoRefSelect Input 路径所需的 Input / Select / Button。
 * 由测试文件 vi.mock('bkui-vue') 按需引用，不改动 vite.config 全局 alias。
 * stub 契约标记用 data-testid（指南条款 4），禁止依赖 CSS 类名。
 */
export const Input = defineComponent({
  name: 'Input',
  props: {
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  template:
    '<input data-testid="repo-ref-input-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
});

const Empty = defineComponent({ name: 'EmptyStub', template: '<div />' });

/** Select 带 Group/Option 占位，满足 repo-ref-select 模板引用 */
export const Select = Object.assign(Empty, {
  Group: Empty,
  Option: Empty,
});

export const Button = defineComponent({ name: 'Button', template: '<button />' });
