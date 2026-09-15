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

import { cloneDeep } from 'lodash-es';
import { defineStore } from 'pinia';

import type { ISearchValue } from '~/composables/use-search';

export interface EnvListState {
  current: number;
  limit: number;
  scrollLeft: number;
  scrollTop: number;
  search: ISearchValue[];
  sortField: string;
  sortOrder: 'asc' | 'desc';
}

/** 仅在当前应用会话中保留各空间列表视图，不缓存环境数据。 */
export const useEnvListStateStore = defineStore('envListState', () => {
  const states = ref<Record<string, EnvListState>>({});
  function save(space: string, state: EnvListState) {
    states.value[space] = cloneDeep(state);
  }
  function get(space: string) {
    return cloneDeep(states.value[space]);
  }
  return { save, get };
});
