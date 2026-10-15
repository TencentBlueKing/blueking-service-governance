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

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { useAppDetail } from '~/stores/app-detail';

import type { EnvOutput } from '~/@types/v1/env';

/** 应用详情是可见环境名单的唯一来源；完整环境列表仍由各页面维护。 */
export function useAppVisibleEnvs() {
  const appStore = useAppDetail();
  const { t } = useI18n();
  const visibleEnvNames = computed(() =>
    appStore.appDetail?.id === appStore.appID ? appStore.appDetail?.visibleEnvNames || [] : [],
  );
  const isRestricted = computed(() => visibleEnvNames.value.length > 0);
  const visibleNames = computed(() => new Set(visibleEnvNames.value));

  function isEnvVisible(env: EnvOutput) {
    if (env.kind === 'feature') return env.ownerAppID === appStore.appID;
    return !isRestricted.value || (!!env.name && visibleNames.value.has(env.name));
  }

  function getEnvDeployDisabledReason(env?: EnvOutput) {
    return env && !isEnvVisible(env) ? t('该应用未配置部署到此环境') : '';
  }

  function checkDeployEnv(env: EnvOutput) {
    const allowed = isEnvVisible(env);
    if (!allowed) Message({ theme: 'warning', message: t('该应用未配置部署到此环境') });
    return allowed;
  }

  return { checkDeployEnv, getEnvDeployDisabledReason, isEnvVisible, isRestricted, visibleEnvNames };
}
