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

import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';

import { type TargetTexts, formatOptionTarget } from './release-preview';

import type { LatestDeployStatus } from '~/@types/v1/deploy';
import type { BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

/**
 * 分批发布准入被拒的原因；空串表示允许分批发布。
 * 顺序即展示优先级（见 resolveCanaryBlockReason），文案由使用方按 key 取 i18n。
 */
export type CanaryBlockReason =
  | ''
  | 'buildFromSource'
  | 'checkingWorkloadKind'
  | 'deploymentWorkload'
  | 'federationEnv'
  | 'firstDeploy'
  | 'inplaceRestart'
  | 'notAppModelApp'
  | 'replicasChanged';

export interface CanaryEligibilityInput {
  /** 是否为应用模型应用（tRPC / TAF） */
  isAppModelApp: boolean;
  /** 本次是否走「从源码构建」部署（构建部署不支持分批） */
  isBuildFromSource: boolean;
  isFederationEnv: boolean;
  /** 目标环境 / 泳道是否还没有可复用的部署（首次部署没有存量实例可分批） */
  isFirstDeploy: boolean;
  /** 本次是否走「仅镜像（原地重启）」（实例更新接口不支持分批） */
  isInplaceRestart: boolean;
  /** 主工作负载类型；空串表示未知（不因此禁用，由后端兜底校验） */
  mainWorkloadKind: string;
  /** 实例数是否已被改成与当前生效值不同（分批更新沿用现有实例数，不能同时扩缩容） */
  replicasChanged: boolean;
  /** 分批发布资格检查（主工作负载 / 首次部署）是否进行中 */
  workloadKindLoading: boolean;
}

/** 发布方式：一次更新全部实例 / 按策略分批更新（枚举与部署记录共用 ~/common/enums/deploy） */
export type { ReleaseMode } from '~/common/enums/deploy';

/** 发布策略下拉选项 */
export interface ReleaseStrategyOption {
  /** 平台内置策略（不可编辑，仅可克隆） */
  isBuiltin: boolean;
  label: string;
  value: string;
}

/**
 * 策略下拉选项：空间自定义策略在前、平台内置策略在后（各自保持接口返回顺序）。
 * sort 在现代引擎中稳定，不会打乱同组内的创建时间序。
 */
export function buildStrategyOptions(
  strategies: ReleaseStrategyOutputObj[] | undefined,
  texts: TargetTexts,
): ReleaseStrategyOption[] {
  return (strategies ?? [])
    .map(item => ({
      isBuiltin: item.isBuiltin === true,
      value: item.id ?? '',
      label: formatStrategyOptionLabel(item.name ?? '', item.batches, texts),
    }))
    .sort((a, b) => Number(a.isBuiltin) - Number(b.isBuiltin));
}

/** 选项摘要拼装：策略名 + 批次序列，如「生产谨慎发布(1个,25%,50%,全量)」 */
export function formatStrategyOptionLabel(
  name: string,
  batches: BatchOutputObj[] | undefined,
  texts: TargetTexts,
): string {
  const summary = (batches ?? []).map(batch => formatOptionTarget(batch.target, texts)).join(',');
  return summary ? `${name}(${summary})` : name;
}

/**
 * 判断目标环境 / 泳道是否属于首次部署（分批发布的准入前置检查）。
 *
 * 三种情况都按首次部署处理：
 * - 记录为空（接口 data 为 null）：既没有部署记录也没有构建记录；
 * - hasDeployRecord 为 false：只有构建记录（构建失败 / 未走到部署），工作负载从未创建；
 * - status 为 uninstalled：已下线，工作负载已被删除，本次部署等同重新创建。
 *
 * latest 为 undefined 表示调用方没能拿到状态（接口不可用或请求失败），同样无法证明存在存量实例，
 * 由调用方决定是否按首次部署拦截。
 */
export function isFirstDeployFromLatestStatus(latest?: LatestDeployStatus | null): boolean {
  if (!latest) return true;
  if (latest.hasDeployRecord === false) return true;
  return latest.status === APP_DEPLOY_STATUS.UNINSTALLED;
}

/**
 * 分批发布准入规则（实例列表「全量更新」与「快速部署」两个入口共用的唯一事实源）。
 * 返回首个不满足项的原因 key；空串表示允许分批发布。
 * 主工作负载未知（无部署记录）时不拦截，交由后端兜底校验。
 */
export function resolveCanaryBlockReason(input: CanaryEligibilityInput): CanaryBlockReason {
  if (!input.isAppModelApp) return 'notAppModelApp';
  if (input.isFederationEnv) return 'federationEnv';
  // 资格检查完成前不允许选择分批发布，避免「先选后禁」留下已选策略透传给后端
  if (input.workloadKindLoading) return 'checkingWorkloadKind';
  // 首次部署没有存量实例可灰度，所有实例本来就是一次创建，分批没有意义
  if (input.isFirstDeploy) return 'firstDeploy';
  if (input.mainWorkloadKind === 'Deployment') return 'deploymentWorkload';
  if (input.isInplaceRestart) return 'inplaceRestart';
  if (input.isBuildFromSource) return 'buildFromSource';
  // 分批更新不能与修改实例数同时进行；改回当前生效实例数后即可再次选择分批，因此排在最后
  if (input.replicasChanged) return 'replicasChanged';
  return '';
}
