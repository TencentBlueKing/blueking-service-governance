/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) that applies to the current version
 * of the project delivered to anyone in the future.
 */

import type { LatestDeployStatus } from '~/@types/v1/deploy';
import type { EnvOutput } from '~/@types/v1/env';

/**
 * 构建阶段不会新建部署记录，也不会删除上一轮实例；latest-status 在该阶段聚合出的
 * hasDeployRecord 可能为 false 或缺失。构建尚未进入部署、旧实例仍在运行时，实例列表
 * 必须继续按上一轮部署记录查询，因此 build 阶段对实例列表恒视为「有部署记录」，
 * 避免被当作无部署记录而停止实例监听并清空列表。
 */
export function resolveInstanceListHasDeployRecord(latest: LatestDeployStatus | null | undefined): boolean | undefined {
  if (!latest) return undefined;
  if (latest.stage === 'build') return true;
  return latest.hasDeployRecord;
}

/**
 * 环境选择器 refresh 后会重抛当前环境；环境未变化时应保留最新部署状态与实例快照，
 * 避免构建中把状态置空导致实例列表被清空，只能等下一轮轮询才恢复。
 * 注意：prevEnvName 必须在更新部署 Store 的当前环境之前读取，否则恒判为「未变化」。
 */
export function shouldResetDeployStatusOnEnvChange(
  env: EnvOutput | undefined,
  prevEnvName: string | undefined,
): boolean {
  return !env?.name || env.name !== prevEnvName;
}
