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

import { type Ref, computed, ref, watch } from 'vue';

interface UseAlertVisibilityOptions {
  /** 无论如何都展示的状态 */
  alwaysShowKeys: string[];
  /** 看到这些状态 = 用户在场，后续结果才展示 */
  seenKeys: string[];
}

export function useAlertVisibility(key: Ref<string | undefined>, options: UseAlertVisibilityOptions) {
  const { seenKeys, alwaysShowKeys } = options;
  const hasSeen = ref(false);

  watch(key, newKey => {
    if (newKey && seenKeys.includes(newKey)) {
      hasSeen.value = true;
    }
  });

  const isVisible = computed(() => {
    const currentKey = key.value;
    if (!currentKey) return false;
    if (seenKeys.includes(currentKey)) return true;
    if (alwaysShowKeys.includes(currentKey)) return true;
    return hasSeen.value;
  });

  return { isVisible };
}
