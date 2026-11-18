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
  /** 已回滚（分批发布回滚完成，所有实例恢复旧版本；终态，归为非错误态） */
  ROLLED_BACK: 'rolled-back',
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

/**
 * 发布方式（表单 releaseMode / 部署记录 releaseMode / latest-status inProgressReleaseMode）。
 * canary 表示按发布策略分批更新，all 表示一次更新全部实例。
 */
export const RELEASE_MODE = {
  ALL: 'all',
  CANARY: 'canary',
} as const;

/** 分批发布的当前批次阶段（进度接口 batchPhase） */
export const BATCH_PHASE = {
  /** 本批更新中 */
  UPDATING: 'updating',
  /** 本批尚未开始 */
  PENDING: 'pending',
  /** 本批已完成，等策略配置的定时暂停到期 */
  TIMED_PAUSED: 'timed-paused',
  /** 本批已完成，等待人工继续（含策略配置的永久暂停） */
  MANUAL_PAUSED: 'manual-paused',
  /** 所有批次已走完，进入全量收尾 */
  COMPLETED: 'completed',
} as const;

/** 分批发布的 Rollout 阶段（进度接口 rolloutPhase，与后端 batchrelease 状态对齐） */
export const ROLLOUT_PHASE = {
  /** 正在执行发布步骤：后端只在该阶段接受手动操作 */
  RUNNING: 'Running',
  /** 所有步骤已完成，正在全量更新剩余旧版本实例 */
  UPDATING_PRIMARY: 'UpdatingPrimary',
  /** 正在清理灰度资源（发布开始与结束时都会出现） */
  CLEANING_CANARY: 'CleaningCanary',
} as const;

/** 批次完成后的暂停方式（策略配置 pauseMode） */
export const BATCH_PAUSE_MODE = {
  /** 不暂停，直接进入下一批 */
  NONE: 'none',
  /** 定时暂停，到期自动继续 */
  DURATION: 'duration',
  /** 永久暂停，等待人工继续 */
  PERMANENT: 'permanent',
} as const;

export type BatchPauseMode = (typeof BATCH_PAUSE_MODE)[keyof typeof BATCH_PAUSE_MODE];

export type ReleaseMode = (typeof RELEASE_MODE)[keyof typeof RELEASE_MODE];
