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

import { ref, toValue, watch } from 'vue';
import type { MaybeRefOrGetter } from 'vue';

import { AppSpecService } from '~/api/modules/v1';
import { ENV_TYPE_PRODUCTION } from '~/common/const';
import { isRuntimeModSupported } from '~/composables/use-app-runtime';
import { useAppDetail } from '~/stores/app-detail';

interface UseEnvDevModeOptions {
  /** 目标环境名称；为空表示当前没有可查询的环境 */
  envName: MaybeRefOrGetter<string | undefined>;
  /** 环境类型，生产环境不支持开发模式 */
  envType?: MaybeRefOrGetter<string | undefined>;
  /** 额外的跳过条件，如默认环境不提供环境级开关；跳过时视为未开启 */
  skip?: MaybeRefOrGetter<boolean>;
}

/**
 * 环境生效的开发模式开关。作用域（应用/环境/运行能力）变化时先回落为未开启，
 * 新结果返回前不沿用上一个环境的状态；只允许最新一次请求写回结果。
 */
export function useEnvDevMode(options: UseEnvDevModeOptions) {
  const { envName, envType, skip } = options;
  const appDetailStore = useAppDetail();
  const isDevModeEnabled = ref(false);
  let revision = 0;

  async function loadDevMode() {
    const current = ++revision;
    const appID = appDetailStore.appID;
    const name = toValue(envName);
    // 后端 effective 不做生产环境过滤（限制只在渲染期生效），需在此显式兜底。
    if (
      !appID ||
      !name ||
      toValue(envType) === ENV_TYPE_PRODUCTION ||
      !isRuntimeModSupported(appDetailStore.runtime, 'devMode') ||
      toValue(skip)
    ) {
      isDevModeEnabled.value = false;
      return;
    }

    isDevModeEnabled.value = false;
    try {
      const spec = await AppSpecService.getEnvEffectiveAppSpecDevMode({ appID, envName: name });
      if (current === revision) isDevModeEnabled.value = spec?.enabled ?? false;
    } catch (error) {
      console.error(error);
      if (current === revision) isDevModeEnabled.value = false;
    }
  }

  watch(
    [
      () => appDetailStore.appID,
      () => toValue(envName),
      () => toValue(envType),
      () => appDetailStore.runtime,
      () => toValue(skip),
    ],
    () => void loadDevMode(),
    { immediate: true },
  );

  return { isDevModeEnabled };
}
