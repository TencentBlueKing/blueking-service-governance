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
 * 部署总览灰度发布中状态：进度收敛纯函数单测
 *
 * 覆盖规则（对应 TAPD 1070165342139528266 部署状态展示逻辑）：
 *   1. 分批发布进行中（status=deploying）且已进入批次 → 第 N/M 批（currentBatchIndex 从 0 转 1 起）
 *   2. 尚未进入批次（currentBatchIndex=null）→ 只显示灰度发布中（无批次后缀）
 *   3. 回滚中（currentBatchIndex=-1 且 status=deploying）→ 只显示灰度发布中
 *   4. 进度为终态（deployed / rolled-back）→ null（即使总览行仍是 deploying，如构建记录较新）
 *   5. 无进度（data null）或批次缺失 → null
 */
import { describe, expect, it } from 'vitest';
import { toCanaryProgress } from '~/pages/application/detail/deploy/overview/use-deploy-overview';

import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';

function buildProgress(overrides: Partial<BatchReleaseProgressOutputObj> = {}): BatchReleaseProgressOutputObj {
  return {
    batchPhase: 'updating',
    batches: [{ target: '1', pauseMode: 'permanent' }, { target: '100%', pauseMode: 'none' }],
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
