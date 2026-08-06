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

import { ref, shallowRef } from 'vue';

import { defineStore } from 'pinia';
import { type AppSpecResourcesOutput } from '~/@types/appspec_resources';
import { type EnvOutputObj } from '~/@types/env';
import { type DeployableImageTagOutputObj } from '~/@types/image';
import { ApiServerService } from '~/api/modules/bkmsserver';
import { useAppDetail } from '~/stores/app-detail';

export const useTrpcDeployStore = defineStore('trpcDeploy', () => {
  const curEnvItem = ref<EnvOutputObj>();
  const imageList = ref<DeployableImageTagOutputObj[]>([]);
  const appDetailStore = useAppDetail();
  const deploySpec = shallowRef<AppSpecResourcesOutput | null>();

  /**
   * 根据镜像 tag 获取完整的镜像信息
   */
  function getCurrentImage(currentImageTag: string) {
    return imageList.value.find(item => item.tag === currentImageTag);
  }

  function updateCurEnvItem(item?: EnvOutputObj) {
    curEnvItem.value = item;
  }

  /**
   * 获取部署规格数据
   */
  async function getDeploySpec() {
    deploySpec.value = await ApiServerService.GetAppDefaultAppSpecResources({
      appID: appDetailStore.appID,
    }).catch(() => null);
  }

  return {
    curEnvItem,
    imageList,
    deploySpec,
    getCurrentImage,
    updateCurEnvItem,
    getDeploySpec,
  };
});
