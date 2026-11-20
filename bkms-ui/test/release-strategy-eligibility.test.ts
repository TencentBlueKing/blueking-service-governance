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
import { describe, expect, it } from 'vitest';
import {
  type CanaryEligibilityInput,
  buildStrategyOptions,
  formatStrategyOptionLabel,
  isFirstDeployFromLatestStatus,
  resolveCanaryBlockReason,
} from '~/pages/application/detail/deploy/release-strategy';

/** 全部分批发布条件都满足的基线，逐项打开以验证单条规则的拦截效果 */
function eligible(overrides: Partial<CanaryEligibilityInput> = {}): CanaryEligibilityInput {
  return {
    isAppModelApp: true,
    isBuildFromSource: false,
    isFederationEnv: false,
    isFirstDeploy: false,
    isInplaceRestart: false,
    mainWorkloadKind: 'GameDeployment',
    workloadKindLoading: false,
    ...overrides,
  };
}

describe('分批发布准入规则', () => {
  it('条件全部满足时允许分批发布（返回空串）', () => {
    expect(resolveCanaryBlockReason(eligible())).toBe('');
  });

  it('主工作负载未知（无部署记录）时不拦截，交由后端兜底校验', () => {
    expect(resolveCanaryBlockReason(eligible({ mainWorkloadKind: '' }))).toBe('');
  });

  it.each([
    ['helm 等非应用模型应用', { isAppModelApp: false }, 'notAppModelApp'],
    ['联邦集群环境', { isFederationEnv: true }, 'federationEnv'],
    ['资格检查进行中', { workloadKindLoading: true }, 'checkingWorkloadKind'],
    ['首次部署', { isFirstDeploy: true }, 'firstDeploy'],
    ['主工作负载为 Deployment', { mainWorkloadKind: 'Deployment' }, 'deploymentWorkload'],
    ['原地重启（仅镜像）', { isInplaceRestart: true }, 'inplaceRestart'],
    ['从源码构建部署', { isBuildFromSource: true }, 'buildFromSource'],
  ])('%s 时被拦截', (_name, overrides, reason) => {
    expect(resolveCanaryBlockReason(eligible(overrides as Partial<CanaryEligibilityInput>))).toBe(reason);
  });

  it('多个条件同时不满足时按展示优先级返回首个原因', () => {
    // 非应用模型应用优先级最高：此时主工作负载、构建方式等检查结果都不具备参考意义
    expect(
      resolveCanaryBlockReason(
        eligible({ isAppModelApp: false, isBuildFromSource: true, mainWorkloadKind: 'Deployment' }),
      ),
    ).toBe('notAppModelApp');
    // 联邦集群先于原地重启 / 构建部署
    expect(resolveCanaryBlockReason(eligible({ isFederationEnv: true, isBuildFromSource: true }))).toBe(
      'federationEnv',
    );
    // 资格检查未完成时无法判断是否首次部署，检查中优先于首次部署
    expect(resolveCanaryBlockReason(eligible({ workloadKindLoading: true, isFirstDeploy: true }))).toBe(
      'checkingWorkloadKind',
    );
    // 首次部署先于主工作负载判定：此时工作负载尚未创建，判定结果没有参考意义
    expect(resolveCanaryBlockReason(eligible({ isFirstDeploy: true, mainWorkloadKind: 'Deployment' }))).toBe(
      'firstDeploy',
    );
    // 原地重启先于构建部署（两者互斥，但优先级需稳定）
    expect(resolveCanaryBlockReason(eligible({ isInplaceRestart: true, isBuildFromSource: true }))).toBe(
      'inplaceRestart',
    );
  });
});

describe('首次部署判定', () => {
  it.each([
    ['最新状态为空（data === null），既无部署也无构建记录', null, true],
    ['只有构建记录（hasDeployRecord === false）', { hasDeployRecord: false, status: 'failed' }, true],
    ['已下线（status === uninstalled）', { hasDeployRecord: true, status: 'uninstalled' }, true],
    ['已部署', { hasDeployRecord: true, status: 'deployed' }, false],
    ['部署失败但存在部署记录（可基于旧版本灰度）', { hasDeployRecord: true, status: 'failed' }, false],
    ['未标注 hasDeployRecord 的历史记录按已有部署处理', { status: 'deployed' }, false],
  ])('%s：%s，首次部署应为 %s', (_name, latest, expected) => {
    expect(isFirstDeployFromLatestStatus(latest)).toBe(expected);
  });

  it('完全拿不到状态（undefined）时按首次部署处理，由调用方决定是否拦截', () => {
    expect(isFirstDeployFromLatestStatus(undefined)).toBe(true);
  });
});

describe('发布策略下拉选项', () => {
  const texts = { full: '全量', unit: '个' };

  it('选项摘要按「策略名(批次序列)」拼装，空 target 记为全量', () => {
    const label = formatStrategyOptionLabel(
      '生产谨慎发布',
      [{ target: '1' }, { target: '25%' }, { target: '' }],
      texts,
    );
    expect(label).toBe('生产谨慎发布(1个,25%,全量)');
  });

  it('策略无批次时只展示策略名', () => {
    expect(formatStrategyOptionLabel('空策略', [], texts)).toBe('空策略');
    expect(formatStrategyOptionLabel('空策略', undefined, texts)).toBe('空策略');
  });

  it('空间自定义策略在前、平台内置策略在后，同组内保持接口顺序', () => {
    const options = buildStrategyOptions(
      [
        { id: 'builtin-1', name: '内置A', isBuiltin: true, batches: [{ target: '100%' }] },
        { id: 'custom-1', name: '自定义A', isBuiltin: false, batches: [{ target: '50%' }] },
        { id: 'builtin-2', name: '内置B', isBuiltin: true, batches: [{ target: '100%' }] },
        { id: 'custom-2', name: '自定义B', isBuiltin: false, batches: [{ target: '25%' }] },
      ],
      texts,
    );
    expect(options.map(option => option.value)).toEqual(['custom-1', 'custom-2', 'builtin-1', 'builtin-2']);
    expect(options[0]).toEqual({ isBuiltin: false, label: '自定义A(50%)', value: 'custom-1' });
  });

  it('空列表返回空选项', () => {
    expect(buildStrategyOptions([], texts)).toEqual([]);
    expect(buildStrategyOptions(undefined, texts)).toEqual([]);
  });
});
