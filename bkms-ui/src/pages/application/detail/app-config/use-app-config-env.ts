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

import { useLocalStorage } from '@vueuse/core';
import { useAppDetail } from '~/stores/app-detail';
import { useSpaceStore } from '~/stores/space';

export const DEFAULT_ENV_NAME = '__default__';

/** 应用配置页签共用的环境选择；文件本地回退到默认配置时不应写入这里。 */
export function useAppConfigEnv() {
  const spaceStore = useSpaceStore();
  const appDetailStore = useAppDetail();
  const storedEnvNames = useLocalStorage<Record<string, string>>('bkms_app_config_deploy_env', {});

  function getEnvStorageKey() {
    return `${spaceStore.currentSpace}:${appDetailStore.appID}`;
  }

  function getStoredEnvName() {
    return storedEnvNames.value[getEnvStorageKey()] || '';
  }

  function saveEnvNameToStorage(envName: string) {
    const realName = envName === DEFAULT_ENV_NAME ? '' : envName;
    const storageKey = getEnvStorageKey();
    const nextStoredEnvNames = { ...storedEnvNames.value };
    if (realName) {
      nextStoredEnvNames[storageKey] = realName;
    } else {
      delete nextStoredEnvNames[storageKey];
    }
    storedEnvNames.value = nextStoredEnvNames;
  }

  return { getEnvStorageKey, getStoredEnvName, saveEnvNameToStorage };
}
