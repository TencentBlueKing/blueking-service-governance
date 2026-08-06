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

import { ref, watch } from 'vue';

import { useLocalStorage } from '@vueuse/core';
import { fromPairs, orderBy, toPairs } from 'lodash-es';
import { defineStore } from 'pinia';

import { useSpaceStore } from './space';

import type { EnvOutputObj } from '~/@types/env';

interface AppEnvSelection {
  mode: 'multi' | 'single';
  selectedEnvs: string[];
  updatedAt: number;
}

const APP_ENV_SELECTION_STORAGE_KEY = 'bkms_deploy_env_app_selections';
const APP_ENV_SELECTION_LIMIT = 30;

export const useDeployEnvStore = defineStore('deploy-env', () => {
  // 当前选中的环境（单选模式）
  const currentEnv = ref<string>('');
  // 多选环境列表
  const selectedEnvs = ref<string[]>([]);
  // 环境列表（供多环境模式使用）
  const envList = ref<EnvOutputObj[]>([]);
  const appEnvSelections = useLocalStorage<Record<string, AppEnvSelection>>(APP_ENV_SELECTION_STORAGE_KEY, {});
  // 获取空间 store
  const spaceStore = useSpaceStore();

  function updateCurrentEnv(env: string) {
    currentEnv.value = env;
  }

  // 更新多选环境列表
  function updateSelectedEnvs(envs: string[]) {
    selectedEnvs.value = envs;
  }

  // 更新环境列表
  function updateEnvList(list: EnvOutputObj[]) {
    envList.value = list;
  }

  function getAppEnvSelection(scopeKey: string) {
    return appEnvSelections.value[scopeKey];
  }

  function updateAppEnvSelection(scopeKey: string, payload: Partial<Pick<AppEnvSelection, 'mode' | 'selectedEnvs'>>) {
    if (!scopeKey) return;
    const current = appEnvSelections.value[scopeKey];
    const nextSelections = {
      ...appEnvSelections.value,
      [scopeKey]: {
        mode: payload.mode ?? current?.mode ?? 'single',
        selectedEnvs: payload.selectedEnvs ?? current?.selectedEnvs ?? [],
        updatedAt: Date.now(),
      },
    };
    const limitedSelections = orderBy(toPairs(nextSelections), ([, selection]) => selection.updatedAt, 'desc').slice(
      0,
      APP_ENV_SELECTION_LIMIT,
    );
    appEnvSelections.value = fromPairs(limitedSelections) as Record<string, AppEnvSelection>;
  }

  function clearCurrentEnv() {
    currentEnv.value = '';
    selectedEnvs.value = [];
  }

  // 当空间切换时清空环境缓存
  watch(
    () => spaceStore.currentSpace,
    (newSpace, oldSpace) => {
      if (newSpace !== oldSpace) {
        clearCurrentEnv();
        envList.value = [];
      }
    },
  );

  return {
    currentEnv,
    selectedEnvs,
    envList,
    appEnvSelections,
    updateCurrentEnv,
    updateSelectedEnvs,
    updateEnvList,
    getAppEnvSelection,
    updateAppEnvSelection,
    clearCurrentEnv,
  };
});
