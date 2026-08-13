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

/** AppModel 部署来源（与后端 deploySource 枚举对齐） */
export const DEPLOY_SOURCE = {
  /** 直接部署 */
  DIRECT_DEPLOY: 'directDeploy',
  /** 构建完成后自动部署 */
  BUILD_AUTO_DEPLOY: 'buildAutoDeploy',
} as const;

/** AppModel 类型应用部署状态 */
export const APP_DEPLOY_STATUS = {
  /** 未知状态 */
  UNKNOWN: 'unknown',
  /** 部署中 */
  DEPLOYING: 'deploying',
  /** 已部署 */
  DEPLOYED: 'deployed',
  /** 卸载中 */
  UNINSTALLING: 'uninstalling',
  /** 已卸载 */
  UNINSTALLED: 'uninstalled',
  /** 部署失败 */
  FAILED: 'failed',
  /** 取消部署（用户取消：部署中时重新部署，自动取消之前的部署） */
  CANCELED: 'canceled',
  /** 轮询超时 */
  POLLING_TIMEOUT: 'polling-timeout',
  /** 轮询中断 */
  POLLING_BROKEN: 'polling-broken',
} as const;

/** Helm 类型应用部署状态 */
export const HELM_DEPLOY_STATUS = {
  /** 未知状态 */
  UNKNOWN: 'unknown',
  /** 已部署 */
  DEPLOYED: 'deployed',
  /** 已卸载 */
  UNINSTALLED: 'uninstalled',
  /** 卸载中 */
  UNINSTALLING: 'uninstalling',
  /** 已被取代 */
  SUPERSEDED: 'superseded',
  /** 部署失败 */
  FAILED: 'failed',
  /** 安装中 */
  PENDING_INSTALL: 'pending-install',
  /** 升级中 */
  PENDING_UPGRADE: 'pending-upgrade',
  /** 回滚中 */
  PENDING_ROLLBACK: 'pending-rollback',
} as const;

/** 失败类状态（deploy.vue、artifact-expand.vue 等多处复用） */
export const DEPLOY_FAILED_STATUSES = [
  APP_DEPLOY_STATUS.FAILED,
  APP_DEPLOY_STATUS.POLLING_TIMEOUT,
  APP_DEPLOY_STATUS.POLLING_BROKEN,
] as const;
