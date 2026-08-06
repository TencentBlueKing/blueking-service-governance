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

/** 应用构建状态 */
export const APP_BUILD_STATUS = {
  /** 构建记录已创建，状态尚未明确 */
  UNKNOWN: 'unknown',
  /** 构建中 */
  RUNNING: 'running',
  /** 构建成功（将自动进入部署阶段） */
  SUCCESS: 'success',
  /** 构建失败 */
  FAILED: 'failed',
  /** 构建被取消 */
  CANCELED: 'canceled',
  /** 构建状态轮询超时 */
  POLLING_TIMEOUT: 'polling-timeout',
  /** 构建状态轮询异常中断 */
  POLLING_BROKEN: 'polling-broken',
} as const;

export const BUILD_INTERRUPT_STATUSES = [
  APP_BUILD_STATUS.UNKNOWN,
  APP_BUILD_STATUS.CANCELED,
  APP_BUILD_STATUS.POLLING_TIMEOUT,
  APP_BUILD_STATUS.POLLING_BROKEN,
] as const;
