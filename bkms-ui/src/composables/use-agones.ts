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

/**
 * Agones 应用类型判断 Hook
 *
 * Agones 复用 Helm 的 UI 和流程，差异仅在类型标识、构建配置（仅镜像）和展示文案。
 * 后续 Agones 差异化逻辑优先在此统一处理。
 */

import type { ComputedRef, MaybeRefOrGetter } from 'vue';
import { computed, toRef } from 'vue';

import { useRoute } from 'vue-router';

import { type HelmLikeAppType, APP_TYPES } from './app-type';

import type { AppDetailOutputObj } from '~/@types/app';

export { APP_TYPES as APP_TYPE, type HelmLikeAppType } from './app-type';

/** 详情页用：通过 appData.type 判断 */
export function useAgonesFromAppDetail(appData: MaybeRefOrGetter<AppDetailOutputObj | null | undefined>) {
  const appDataRef = toRef(appData);
  return useAgonesFrom(computed(() => appDataRef.value?.type === APP_TYPES.AGONES));
}

/** 创建流程用：通过路由名判断 */
export function useAgonesFromRoute() {
  const route = useRoute();
  return useAgonesFrom(computed(() => route.name === 'createAgonesTemplateApp'));
}

function useAgonesFrom(isAgones: ComputedRef<boolean>) {
  const appType = computed<HelmLikeAppType>(() => (isAgones.value ? APP_TYPES.AGONES : APP_TYPES.HELM));
  /** Agones 仅支持镜像，不支持代码仓库 */
  const shouldForceDisableCodeRepo = computed(() => isAgones.value);
  return { isAgones, appType, shouldForceDisableCodeRepo };
}
