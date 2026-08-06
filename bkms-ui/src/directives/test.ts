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

import type { Directive, DirectiveBinding } from 'vue';

/**
 * v-test 指令：在非生产环境下为元素添加 data-testid 属性，用于 E2E 测试定位。
 *
 * 命名规范: {模块}-{组件}-{行为}
 *
 * @example
 * <!-- 字符串用法 -->
 * <input v-test="'login-email-input'" />
 * <button v-test="'login-submit-btn'" />
 *
 * <!-- 对象用法，支持动态 id 拼接 -->
 * <tr v-for="item in list" v-test="{ id: `app-list-row-${item.id}` }" />
 */

const isProd = import.meta.env.PROD;

function applyTestId(el: HTMLElement, binding: DirectiveBinding) {
  if (isProd) return;

  const value = binding.value;
  if (!value) return;

  const testId = typeof value === 'string' ? value : value.id;
  if (testId) {
    el.setAttribute('data-testid', testId);
  }
}

const TestDirective: Directive = {
  mounted: applyTestId,
  updated: applyTestId,
};

export default TestDirective;
