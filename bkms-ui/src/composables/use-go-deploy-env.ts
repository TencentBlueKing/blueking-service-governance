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

import { useRoute, useRouter } from 'vue-router';
import { useAppDetail } from '~/stores/app-detail';
import { useDeployEnvStore } from '~/stores/deploy-env';
import { useSpaceStore } from '~/stores/space';

/**
 * 新开标签页跳到部署页并定位到指定环境。
 * HostPort「去部署」等场景复用；与北极星侧实现保持行为一致，但不耦合其页面状态。
 */
export function useGoDeployEnv() {
  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const deployEnvStore = useDeployEnvStore();
  const spaceStore = useSpaceStore();

  function goDeployEnv(envName: string) {
    const spaceParam = route.params.space;
    const space = (Array.isArray(spaceParam) ? spaceParam[0] : spaceParam) || spaceStore.currentSpace;
    deployEnvStore.updateCurrentEnv(envName);
    // 部署页按 `${space}:${appName}` 恢复环境模式；指定环境时写成单环境，避免新标签页先读到 multi 缓存闪烁
    const scopeKey = space && appDetailStore.app ? `${space}:${appDetailStore.app}` : '';
    deployEnvStore.updateAppEnvSelection(scopeKey, { mode: 'single', selectedEnvs: [envName] });
    const resolved = router.resolve({
      name: 'detail',
      params: {
        space,
        name: appDetailStore.app,
        menuName: 'deployment',
        type: appDetailStore.appType,
      },
      query: { activeTab: 'instance', envName },
    });
    window.open(resolved.href, '_blank');
  }

  return { goDeployEnv };
}
