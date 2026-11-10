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
/**
 * 部署总览灰度发布中状态：进度收敛与轮询回填纯函数单测
 *
 * 覆盖规则（对应 TAPD 1070165342139528266 部署状态展示逻辑）：
 *   1. 分批发布进行中（status=deploying）且已进入批次 → 第 N/M 批（currentBatchIndex 从 0 转 1 起）
 *   2. 尚未进入批次（currentBatchIndex=null）→ 只显示灰度发布中（无批次后缀）
 *   3. 回滚中（currentBatchIndex=-1 且 status=deploying）→ 只显示灰度发布中
 *   4. 进度为终态（deployed / rolled-back）→ null（即使总览行仍是 deploying，如构建记录较新）
 *   5. 无进度（data null）或批次缺失 → null
 *   6. 轮询回填：仅当前仍为 deploying 的行沿用上一轮进度，环境进入终态后必须清除
 */
import { describe, expect, it } from 'vitest';
import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';
import {
  type DeployOverviewRow,
  carryOverCanaryProgress,
  toCanaryProgress,
} from '~/pages/application/detail/deploy/overview/use-deploy-overview';

import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';

function buildProgress(overrides: Partial<BatchReleaseProgressOutputObj> = {}): BatchReleaseProgressOutputObj {
  return {
    batchPhase: 'updating',
    batches: [
      { target: '1', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
    ],
    currentBatchIndex: 0,
    deployID: 'deploy-1',
    status: 'deploying',
    ...overrides,
  };
}

describe('toCanaryProgress', () => {
  it('分批发布进行中且已进入批次时，返回 1 起的当前批次与批次总数', () => {
    // currentBatchIndex=1（第 2 批）共 4 批 → 展示「第 2/4 批」
    expect(toCanaryProgress(buildProgress({ currentBatchIndex: 1, batches: Array.from({ length: 4 }) }))).toEqual({
      currentBatch: 2,
      totalBatches: 4,
    });
  });

  it('尚未进入批次（currentBatchIndex=null）时，仅标记进行中不带批次后缀', () => {
    expect(toCanaryProgress(buildProgress({ currentBatchIndex: null, batchPhase: 'pending' }))).toEqual({
      currentBatch: null,
      totalBatches: 2,
    });
  });

  it('回滚中（currentBatchIndex=-1 且 status=deploying）时，仅标记进行中', () => {
    expect(toCanaryProgress(buildProgress({ currentBatchIndex: -1 }))).toEqual({
      currentBatch: null,
      totalBatches: 2,
    });
  });

  it('进度已到终态时返回 null，即使总览行仍是部署中（如构建记录较新）', () => {
    expect(toCanaryProgress(buildProgress({ status: 'deployed' }))).toBeNull();
    expect(toCanaryProgress(buildProgress({ status: 'rolled-back' }))).toBeNull();
    expect(toCanaryProgress(buildProgress({ status: 'failed' }))).toBeNull();
  });

  it('无进度或批次缺失时返回 null', () => {
    expect(toCanaryProgress(null)).toBeNull();
    expect(toCanaryProgress(buildProgress({ batches: [] }))).toBeNull();
    expect(toCanaryProgress(buildProgress({ batches: undefined }))).toBeNull();
  });
});

function buildRow(overrides: Partial<DeployOverviewRow> = {}): DeployOverviewRow {
  return {
    abnormalCount: 0,
    autoScale: { abnormal: false, configured: false, enabled: false, status: null, tips: [] },
    canaryProgress: null,
    clusterID: 'cls-1',
    clusterName: '集群 A',
    cpuLimits: '',
    cpuRequests: '',
    deployedAt: '',
    deployStatus: APP_DEPLOY_STATUS.DEPLOYING,
    desiredCount: 4,
    displayName: 'stag',
    envID: 'env-1',
    imageTag: 'v1.0',
    isFeature: false,
    memoryLimits: '',
    memoryRequests: '',
    name: 'stag',
    runningCount: 4,
    type: 'staging',
    ...overrides,
  };
}

describe('carryOverCanaryProgress', () => {
  const progress = { currentBatch: 2, totalBatches: 4 };

  it('环境仍为部署中时沿用上一轮进度，避免状态列在进度补查返回前闪烁回「部署中」', () => {
    const next = carryOverCanaryProgress([buildRow({ canaryProgress: progress })], [buildRow()]);
    expect(next[0].canaryProgress).toEqual(progress);
  });

  it('环境从部署中变为已部署后，上一轮进度应被清除', () => {
    const previous = [buildRow({ canaryProgress: progress })];
    const firstRound = carryOverCanaryProgress(previous, [buildRow({ deployStatus: APP_DEPLOY_STATUS.DEPLOYED })]);
    expect(firstRound[0].canaryProgress).toBeNull();

    // 旧实现每轮无条件回填：再用终态行跑一轮，验证不会把上一轮进度带回来
    const secondRound = carryOverCanaryProgress(firstRound, [buildRow({ deployStatus: APP_DEPLOY_STATUS.DEPLOYED })]);
    expect(secondRound[0].canaryProgress).toBeNull();
  });

  it('环境进入失败 / 取消等非部署中状态时同样清除进度', () => {
    const previous = [buildRow({ canaryProgress: progress })];
    const statuses = [APP_DEPLOY_STATUS.FAILED, APP_DEPLOY_STATUS.CANCELED, APP_DEPLOY_STATUS.UNKNOWN];
    statuses.forEach(status => {
      const [row] = carryOverCanaryProgress(previous, [buildRow({ deployStatus: status })]);
      expect(row.canaryProgress).toBeNull();
    });
  });

  it('上一轮没有该环境或没有进度时保持 null', () => {
    const next = carryOverCanaryProgress(
      [buildRow({ canaryProgress: null }), buildRow({ name: 'prod' })],
      [buildRow(), buildRow({ name: 'prod2' })],
    );
    expect(next.map(row => row.canaryProgress)).toEqual([null, null]);
  });
});
