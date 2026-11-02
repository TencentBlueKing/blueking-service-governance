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

import { cloneDeep } from 'lodash-es';
import { AppService } from '~/api/modules/v1';
import { APP_TYPES } from '~/composables/app-type';
import { useAppDetail } from '~/stores/app-detail';

import type { AppModelSpecInput, UpdateAppStartupConfigInput } from '~/@types/v1/app';

/**
 * 根据真实应用类型保存启动配置。
 */
export default function useSpecField() {
  const appDetailStore = useAppDetail();

  // 新应用只更新启动字段，存量应用仍通过原 spec 接口保存完整配置。
  async function updateStartupConfig(
    startup: Required<Pick<UpdateAppStartupConfigInput, 'args' | 'command' | 'shCommand'>>,
  ) {
    const appID = appDetailStore.appID;
    const appType = appDetailStore.appType;
    if (appType === APP_TYPES.BKMS_APP) {
      // 新接口整体替换启动字段，command、args、shCommand 的空值也需完整提交。
      return AppService.updateAppStartupConfig({ appID, ...startup });
    }
    const currentSpec = cloneDeep(appDetailStore.appDetail?.appModelSpec) as AppModelSpecInput | undefined;
    const appModelSpec: AppModelSpecInput = { ...currentSpec, ...startup };
    if (appType === APP_TYPES.TAF) return AppService.updateAppTafSpec({ appID, appModelSpec });
    if (appType === APP_TYPES.TRPC) return AppService.updateAppTrpcSpec({ appID, appModelSpec });
    throw new Error('Application does not support startup configuration');
  }

  return {
    updateStartupConfig,
  };
}
