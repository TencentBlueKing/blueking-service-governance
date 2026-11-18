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

import { type TargetTexts, formatOptionTarget } from './release-preview';

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
  | 'inplaceRestart'
  | 'notAppModelApp';

export interface CanaryEligibilityInput {
  /** 是否为应用模型应用（tRPC / TAF） */
  isAppModelApp: boolean;
  /** 本次是否走「从源码构建」部署（构建部署不支持分批） */
  isBuildFromSource: boolean;
  isFederationEnv: boolean;
  /** 本次是否走「仅镜像（原地重启）」（实例更新接口不支持分批） */
  isInplaceRestart: boolean;
  /** 主工作负载类型；空串表示未知（不因此禁用，由后端兜底校验） */
  mainWorkloadKind: string;
  /** 主工作负载资格检查是否进行中 */
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
 * 分批发布准入规则（实例列表「全量更新」与「快速部署」两个入口共用的唯一事实源）。
 * 返回首个不满足项的原因 key；空串表示允许分批发布。
 * 主工作负载未知（无部署记录）时不拦截，交由后端兜底校验。
 */
export function resolveCanaryBlockReason(input: CanaryEligibilityInput): CanaryBlockReason {
  if (!input.isAppModelApp) return 'notAppModelApp';
  if (input.isFederationEnv) return 'federationEnv';
  // 资格检查完成前不允许选择分批发布，避免「先选后禁」留下已选策略透传给后端
  if (input.workloadKindLoading) return 'checkingWorkloadKind';
  if (input.mainWorkloadKind === 'Deployment') return 'deploymentWorkload';
  if (input.isInplaceRestart) return 'inplaceRestart';
  if (input.isBuildFromSource) return 'buildFromSource';
  return '';
}
