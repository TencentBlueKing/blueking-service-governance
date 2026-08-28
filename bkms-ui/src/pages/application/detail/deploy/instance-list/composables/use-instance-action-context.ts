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

import { type InjectionKey, inject, provide } from 'vue';

import { Message } from 'bkui-vue';
import { i18n } from '~/modules/i18n';

import type { InstanceActionContext } from '../types';

export const INSTANCE_ACTION_CONTEXT_KEY: InjectionKey<InstanceActionContext> = Symbol('instanceActionContext');

export function provideInstanceActionContext(context: InstanceActionContext) {
  provide(INSTANCE_ACTION_CONTEXT_KEY, context);
}

/** 操作提交成功后的统一副作用：提示并刷新 List + Watch。 */
export async function runActionSuccess(context: InstanceActionContext, options?: { clearSelection?: boolean }) {
  Message({
    theme: 'success',
    message: i18n.global.t('操作成功'),
  });

  if (options?.clearSelection) {
    context.clearSelections();
  }

  await context.refreshData();
}

export function useInstanceActionContext(): InstanceActionContext {
  const context = inject(INSTANCE_ACTION_CONTEXT_KEY);
  if (!context) {
    throw new Error('useInstanceActionContext must be used within a provider');
  }
  return context;
}
