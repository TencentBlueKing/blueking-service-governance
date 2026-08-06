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

import { useI18n } from 'vue-i18n';
import { APP_DEPLOY_STATUS, HELM_DEPLOY_STATUS } from '~/common/enums/deploy';
import { type AppType, isHelmLikeAppType } from '~/composables/app-type';

/** AppModel 部署状态 → 文案 key（用于 t()） */
const APP_DEPLOY_STATUS_TEXT_KEY: Record<string, string> = {
  [APP_DEPLOY_STATUS.DEPLOYED]: '已部署',
  [APP_DEPLOY_STATUS.DEPLOYING]: '部署中',
  [APP_DEPLOY_STATUS.FAILED]: '部署失败',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: '部署失败',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: '部署失败',
  [APP_DEPLOY_STATUS.UNINSTALLING]: '卸载中',
  [APP_DEPLOY_STATUS.UNINSTALLED]: '已卸载',
  [APP_DEPLOY_STATUS.CANCELED]: '取消部署',
  [HELM_DEPLOY_STATUS.UNKNOWN]: '未知',
};

/** AppModel 部署状态 → ColorIcon 图标名 */
const APP_DEPLOY_STATUS_ICON: Record<string, string> = {
  [APP_DEPLOY_STATUS.DEPLOYED]: 'normal',
  [APP_DEPLOY_STATUS.DEPLOYING]: 'loading',
  [APP_DEPLOY_STATUS.UNINSTALLING]: 'loading',
  [APP_DEPLOY_STATUS.UNINSTALLED]: 'normal',
  [APP_DEPLOY_STATUS.FAILED]: 'abnormal',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: 'abnormal',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: 'abnormal',
  [APP_DEPLOY_STATUS.CANCELED]: 'warning-2',
  [HELM_DEPLOY_STATUS.UNKNOWN]: 'warning-2',
};

/** AppModel 部署状态 → 颜色 */
const APP_DEPLOY_STATUS_COLOR: Record<string, string> = {
  [APP_DEPLOY_STATUS.DEPLOYED]: 'green',
  [APP_DEPLOY_STATUS.DEPLOYING]: 'blue',
  [APP_DEPLOY_STATUS.FAILED]: 'red',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: 'red',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: 'red',
  [APP_DEPLOY_STATUS.UNINSTALLING]: 'orange',
  [APP_DEPLOY_STATUS.UNINSTALLED]: 'gray',
  [APP_DEPLOY_STATUS.CANCELED]: 'gray',
  [HELM_DEPLOY_STATUS.UNKNOWN]: 'orange',
};

/** Helm 部署状态 → 文案 key（用于 t()） */
const HELM_DEPLOY_STATUS_TEXT_KEY: Record<string, string> = {
  [HELM_DEPLOY_STATUS.DEPLOYED]: '已部署',
  [HELM_DEPLOY_STATUS.UNINSTALLED]: '已卸载',
  [HELM_DEPLOY_STATUS.SUPERSEDED]: '已废弃',
  [HELM_DEPLOY_STATUS.FAILED]: '部署失败',
  [HELM_DEPLOY_STATUS.UNINSTALLING]: '卸载中',
  [HELM_DEPLOY_STATUS.PENDING_INSTALL]: '安装中',
  [HELM_DEPLOY_STATUS.PENDING_UPGRADE]: '升级中',
  [HELM_DEPLOY_STATUS.PENDING_ROLLBACK]: '回滚中',
  [HELM_DEPLOY_STATUS.UNKNOWN]: '未知',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: '部署失败',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: '部署失败',
};

/** Helm 部署状态 → ColorIcon 图标名 */
const HELM_DEPLOY_STATUS_ICON: Record<string, string> = {
  [HELM_DEPLOY_STATUS.DEPLOYED]: 'normal',
  [HELM_DEPLOY_STATUS.UNINSTALLED]: 'normal',
  [HELM_DEPLOY_STATUS.SUPERSEDED]: 'warning-2',
  [HELM_DEPLOY_STATUS.FAILED]: 'abnormal',
  [HELM_DEPLOY_STATUS.UNINSTALLING]: 'loading',
  [HELM_DEPLOY_STATUS.PENDING_INSTALL]: 'loading',
  [HELM_DEPLOY_STATUS.PENDING_UPGRADE]: 'loading',
  [HELM_DEPLOY_STATUS.PENDING_ROLLBACK]: 'loading',
  [HELM_DEPLOY_STATUS.UNKNOWN]: 'warning-2',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: 'abnormal',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: 'abnormal',
};

/** Helm 部署状态 → 颜色 */
const HELM_DEPLOY_STATUS_COLOR: Record<string, string> = {
  [HELM_DEPLOY_STATUS.DEPLOYED]: 'green',
  [HELM_DEPLOY_STATUS.UNINSTALLED]: 'gray',
  [HELM_DEPLOY_STATUS.SUPERSEDED]: 'orange',
  [HELM_DEPLOY_STATUS.FAILED]: 'red',
  [HELM_DEPLOY_STATUS.UNINSTALLING]: 'orange',
  [HELM_DEPLOY_STATUS.PENDING_INSTALL]: 'orange',
  [HELM_DEPLOY_STATUS.PENDING_UPGRADE]: 'orange',
  [HELM_DEPLOY_STATUS.PENDING_ROLLBACK]: 'orange',
  [HELM_DEPLOY_STATUS.UNKNOWN]: 'orange',
  [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: 'red',
  [APP_DEPLOY_STATUS.POLLING_BROKEN]: 'red',
};

/** 合并后的部署状态 → 文案 key（AppModel 优先，补充 Helm 独有状态） */
const MERGED_DEPLOY_STATUS_TEXT_KEY: Record<string, string> = {
  ...HELM_DEPLOY_STATUS_TEXT_KEY,
  ...APP_DEPLOY_STATUS_TEXT_KEY,
};

export interface DeployStatusInfo {
  icon: string;
  text: string;
}

/**
 * 部署状态映射 composable
 * - textMap: 已翻译的文案映射（computed，i18n locale 切换时自动响应）
 * - colorMap: 颜色映射（静态常量，无需响应式）
 */
export function useDeployStatusMap() {
  const { t } = useI18n();

  const appStatusTextMap = computed(() =>
    Object.fromEntries(Object.entries(APP_DEPLOY_STATUS_TEXT_KEY).map(([k, v]) => [k, t(v)])),
  );

  const helmStatusTextMap = computed(() =>
    Object.fromEntries(Object.entries(HELM_DEPLOY_STATUS_TEXT_KEY).map(([k, v]) => [k, t(v)])),
  );

  /** 合并后的状态文案映射（AppModel 优先，覆盖 Helm 同 key） */
  const mergedStatusTextMap = computed(() =>
    Object.fromEntries(Object.entries(MERGED_DEPLOY_STATUS_TEXT_KEY).map(([k, v]) => [k, t(v)])),
  );

  /** 根据 AppModel 部署状态获取 icon + 文案 + 颜色 */
  function getAppDeployStatusInfo(status: string): DeployStatusInfo {
    return {
      icon: APP_DEPLOY_STATUS_ICON[status] || '',
      text: appStatusTextMap.value[status] ? appStatusTextMap.value[status] : '',
    };
  }

  /** 根据 Helm 部署状态获取 icon + 文案 */
  function getHelmDeployStatusInfo(status: string): DeployStatusInfo {
    return {
      icon: HELM_DEPLOY_STATUS_ICON[status] || '',
      text: helmStatusTextMap.value[status] ? helmStatusTextMap.value[status] : '',
    };
  }

  /** 根据应用类型自动选择对应的状态映射，获取 icon + 文案 */
  function getDeployStatusInfo(appType: AppType | null | undefined, status: string): DeployStatusInfo {
    return isHelmLikeAppType(appType) ? getHelmDeployStatusInfo(status) : getAppDeployStatusInfo(status);
  }

  /** 根据应用类型自动选择对应的状态映射表（colorMap + textMap） */
  function getDeployStatusMaps(appType: AppType | null | undefined) {
    return isHelmLikeAppType(appType)
      ? { statusColorMap: HELM_DEPLOY_STATUS_COLOR, statusTextMap: helmStatusTextMap.value }
      : { statusColorMap: APP_DEPLOY_STATUS_COLOR, statusTextMap: appStatusTextMap.value };
  }

  return {
    appStatusTextMap,
    appStatusColorMap: APP_DEPLOY_STATUS_COLOR,
    appStatusIconMap: APP_DEPLOY_STATUS_ICON,
    helmStatusTextMap,
    helmStatusColorMap: HELM_DEPLOY_STATUS_COLOR,
    helmStatusIconMap: HELM_DEPLOY_STATUS_ICON,
    mergedStatusTextMap,
    getAppDeployStatusInfo,
    getHelmDeployStatusInfo,
    getDeployStatusInfo,
    getDeployStatusMaps,
  };
}
