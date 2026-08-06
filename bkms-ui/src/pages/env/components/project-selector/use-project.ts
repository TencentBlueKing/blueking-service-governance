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

import type { Ref } from 'vue';
import { ref } from 'vue';

import { type BCSProjectOutput } from '~/@types/v1/bkintegrations-bcs';
import { BkintegrationsBcsService } from '~/api/modules/v1';

export default function useProject(_searchKey: Ref<string>) {
  const projectList = ref<BCSProjectOutput[]>([]);

  async function handleInitProjectList() {
    const res = await BkintegrationsBcsService.listBCSAuthorizedProjects({}, { validateCode: false }).catch(() => []);
    projectList.value = res || [];
  }

  // 远程搜索
  // const selectRef = ref();
  // const searchKey = useDebouncedRef('', 600);
  // const remoteSearch = (key: string) => {
  //   selectRef.value.searchLoading = false;
  //   searchKey.value = key;
  // };

  return {
    projectList,
    handleInitProjectList,
    // remoteSearch,
  };
}
