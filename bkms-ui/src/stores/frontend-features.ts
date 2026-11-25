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

import { computed, ref } from 'vue';

import { defineStore } from 'pinia';
import { FrontendFeaturesService } from '~/api/modules/v1/frontend-features';

import type { FrontendFeaturesOutput } from '~/@types/v1/frontend-features';

/**
 * 部署配置决定的前端功能开关。
 * 只控制页面入口和展示，后端接口仍会各自校验。
 * 请求失败时保持缺省值：BSCP 与 KubeInsight 开启，其余关闭。
 */
export const useFrontendFeaturesStore = defineStore('frontend-features', () => {
  const features = ref<FrontendFeaturesOutput | null>(null);
  let pendingRequest: null | Promise<FrontendFeaturesOutput | null> = null;

  const polarisWorkspaceAccessPoint = ref(false);
  const enableMultiTenantMode = ref(false);
  const enableIndependentBkRepo = ref(false);
  const enableWeTerm = ref(false);
  const enableBSCP = ref(true);
  const enableKubeInsight = ref(true);

  // 默认关闭，仅在接口显式返回 enableGoProxyBuildArgsTips: true 时开启
  const enableGoProxyBuildArgsTips = computed(() => Boolean(features.value?.enableGoProxyBuildArgsTips));

  function apply(features?: FrontendFeaturesOutput) {
    polarisWorkspaceAccessPoint.value = !!features?.polarisWorkspaceAccessPoint;
    enableMultiTenantMode.value = !!features?.enableMultiTenantMode;
    enableIndependentBkRepo.value = !!features?.enableIndependentBkRepo;
    enableWeTerm.value = !!features?.enableWeTerm;
    enableBSCP.value = features?.enableBSCP !== false;
    enableKubeInsight.value = features?.enableKubeInsight !== false;
  }

  async function getFrontendFeatures() {
    if (pendingRequest) {
      return pendingRequest;
    }

    pendingRequest = FrontendFeaturesService.getFrontendFeatures(undefined, { interceptorErr: false })
      .then(res => {
        features.value = res ?? null;
        apply(res);
        return features.value;
      })
      .catch(err => {
        console.error('获取前端功能开关失败:', err);
        return null;
      })
      .finally(() => {
        pendingRequest = null;
      });

    return pendingRequest;
  }

  async function load() {
    await getFrontendFeatures();
  }

  return {
    features,
    enableGoProxyBuildArgsTips,
    getFrontendFeatures,
    polarisWorkspaceAccessPoint,
    enableMultiTenantMode,
    enableIndependentBkRepo,
    enableWeTerm,
    enableBSCP,
    enableKubeInsight,
    load,
  };
});
