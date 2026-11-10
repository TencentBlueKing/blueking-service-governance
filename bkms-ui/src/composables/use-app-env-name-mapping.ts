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

import { ref } from 'vue';

import { EnvService } from '~/api/modules/v1/env';

/**
 * 本应用的环境名 → 环境展示名映射。
 * 列表接口只返回环境内部名，且特性环境只在 listAppEnvs 中返回，
 * 用空间级环境列表无法翻译，因此统一在这里按 appID 拉取。
 */
export default function useAppEnvNameMapping() {
  const envNameDisplayMap = ref<Record<string, string>>({});

  async function fetchEnvNameDisplayMap(appID: string) {
    if (!appID) {
      envNameDisplayMap.value = {};
      return envNameDisplayMap.value;
    }
    const envList = await EnvService.listAppEnvs({ appID }).catch(() => []);
    const displayMap: Record<string, string> = {};
    envList.forEach(env => {
      if (env.name) displayMap[env.name] = env.displayName || env.name;
    });
    envNameDisplayMap.value = displayMap;
    return displayMap;
  }

  /** 取环境展示名，环境已删除或未收录时回退内部名 */
  function getEnvDisplayName(envName: string) {
    return envNameDisplayMap.value[envName] || envName;
  }

  return {
    envNameDisplayMap,
    fetchEnvNameDisplayMap,
    getEnvDisplayName,
  };
}
