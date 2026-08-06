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

import { computed } from 'vue';

import { AppService } from '~/api/modules/v1';
import { useAppDetail } from '~/stores/app-detail';

/**
 * 根据应用类型获取对应的 spec 字段名和更新 API
 */
export default function useSpecField() {
  const appDetailStore = useAppDetail();

  const appType = computed(() => appDetailStore.appType);

  // 根据应用类型获取对应的 spec 字段名
  const specFieldName = computed(() => (appType.value === 'taf' ? 'tafSpec' : 'trpcSpec'));

  // 根据应用类型获取对应的 API 更新方法
  const updateSpecApi = computed(() => {
    return appType.value === 'taf' ? AppService.updateAppTafSpec : AppService.updateAppTrpcSpec;
  });

  return {
    appType,
    specFieldName,
    updateSpecApi,
  };
}
