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

import { vi } from 'vitest';

/**
 * 共享 i18n mock 工具集。
 *
 * 使用方式（vi.mock 内须用动态 import 避免 hoisting TDZ）：
 *   vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
 *   import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';
 *   render(Comp, { global: { mocks: i18nGlobalMocks, components: { 'i18n-t': I18nTStub } } });
 */

/**
 * vi.mock('vue-i18n') 的标准 factory：保留原始导出，useI18n 直译。
 * 内部使用 vi.importActual 获取原始模块，无需外部传入 importOriginal。
 */
export async function i18nMockFactory() {
  return {
    ...((await vi.importActual('vue-i18n')) as object),
    useI18n: () => ({ t: (s: string) => s, te: () => true }),
  };
}

/** render() 的 global.mocks 对象：模板中的 $t 直译 */
export const i18nGlobalMocks = { $t: (s: string) => s };

/**
 * render() 的 global.plugins 插件：通过 globalProperties 注入 $t 直译。
 * 适用于不通过 mocks 而通过 plugin 注入的场景（如 cluster-components / component-management）。
 */
export const i18nPlugin = {
  install(app: { config: { globalProperties: Record<string, unknown> } }) {
    app.config.globalProperties.$t = (s: string) => s;
  },
};

/** <i18n-t> 组件 stub：直译 keypath，可在断言中查找 keypath 文本 */
export const I18nTStub = {
  props: { keypath: { type: String, default: '' } },
  template: '<span>{{ keypath }}</span>',
};
