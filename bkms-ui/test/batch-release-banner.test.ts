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
 * 实例列表分批发布控制横幅：状态映射与请求门控纯函数单测
 *
 * 覆盖规则：
 *   1. 仅 status=deploying 时展示横幅；其余状态一律隐藏
 *   2. 批次阶段 → 按钮集合（manual-paused / timed-paused / updating / pending / 回滚中）
 *   3. 「更多」下拉的低频操作与置灰规则
 *   4. 进度补查门控：非分批发布不查、同一 deployID 终态不重复查、新部署重新查
 *   5. 灰度发布进度视图的批次步骤视图模型（buildBatchReleaseSteps）
 *   6. 实例灰度归属判定（resolveInstanceCanaryState，与进度视图灰度列表筛选同口径）
 */
import { describe, expect, it } from 'vitest';
import {
  type BatchReleaseBannerView,
  buildBatchReleaseDismissKey,
  buildBatchReleaseSteps,
  hasCanaryDeployRecord,
  isBatchReleaseBannerDismissed,
  isBatchReleaseBusy,
  resolveBatchReleaseBanner,
  resolveBatchReleaseSuccessMessage,
  resolveInstanceCanaryState,
  shouldFetchBatchReleaseProgress,
} from '~/pages/application/detail/deploy/instance-list/batch-release';

import type { BatchReleaseProgressOutputObj, LatestDeployStatus } from '~/@types/v1/deploy';

function actionsOf(view: BatchReleaseBannerView | null) {
  return (view?.actions ?? []).map(item => item.action);
}

function buildLatest(overrides: Partial<LatestDeployStatus> = {}): LatestDeployStatus {
  return {
    deployID: 'deploy-1',
    inProgressReleaseMode: 'canary',
    releaseMode: 'canary',
    stage: 'deploy',
    status: 'deploying',
    ...overrides,
  };
}

// 接口契约里 currentBatchIndex 在批次序号未回填时为 null（d.ts 仅标注 number），测试按真实取值放宽
function buildProgress(
  overrides: Omit<Partial<BatchReleaseProgressOutputObj>, 'currentBatchIndex'> & {
    currentBatchIndex?: null | number;
  } = {},
): BatchReleaseProgressOutputObj {
  return {
    batchPhase: 'updating',
    batches: [
      { target: '1', pauseMode: 'permanent' },
      { target: '50%', pauseMode: 'permanent' },
      { target: '100%', pauseMode: 'none' },
    ],
    currentBatchIndex: 2,
    deployID: 'deploy-1',
    newImageTag: 'v1.8.3',
    oldImageTag: 'v1.8.2',
    replicas: 20,
    status: 'deploying',
    updatedReplicas: 8,
    ...overrides,
  };
}

function overflowOf(view: BatchReleaseBannerView | null) {
  return (view?.overflowActions ?? []).map(item => `${item.action}${item.disabled ? ':disabled' : ''}`);
}

/** 剩余秒 → mm:ss，与步骤条 Tag 的倒计时格式一致。 */
function remainingFormatted(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`;
}

/** 倒计时用例的固定当前时间：传入 buildBatchReleaseSteps 的 now 参数，验证逐秒递减。 */
const TIMESTAMP_NOW = Date.now();

describe('resolveBatchReleaseBanner：可见状态', () => {
  it('已完成（status=deployed）展示成功提示且可关闭、无操作按钮', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'completed', status: 'deployed' }));
    expect(view?.kind).toBe('completed');
    expect(view?.closable).toBe(true);
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    expect(view?.statusTag).toEqual({ key: '已完成', theme: 'success' });
  });

  it('已回滚（status=rolled-back）展示非错误提示且可关闭、无操作按钮', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ status: 'rolled-back' }));
    expect(view?.kind).toBe('rolledBack');
    expect(view?.closable).toBe(true);
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    // 回滚态设计稿未覆盖，不展示标题 Tag
    expect(view?.statusTag).toBeNull();
  });

  it('失败（status=failed）展示失败原因并提供回退上一批与回滚，且不可关闭', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ message: '2 个实例健康检查未通过', status: 'failed' }));
    expect(view?.kind).toBe('failed');
    expect(view?.closable).toBe(false);
    expect(view?.message).toBe('2 个实例健康检查未通过');
    // 设计稿：失败态无「重试」，回退上一批收进更多，回滚平铺
    expect(actionsOf(view)).toEqual(['rollback']);
    expect(view?.statusTag).toEqual({ key: '失败', theme: 'danger' });
    expect(overflowOf(view)).toEqual(['backward']);
  });

  it('无进度 / 未知状态 / 卸载类状态都不展示横幅', () => {
    expect(resolveBatchReleaseBanner(buildProgress({ status: 'unknown' }))).toBeNull();
    expect(resolveBatchReleaseBanner(buildProgress({ status: 'uninstalled' }))).toBeNull();
    expect(resolveBatchReleaseBanner(buildProgress({ status: 'canceled' }))).toBeNull();
    expect(resolveBatchReleaseBanner(buildProgress({ status: undefined }))).toBeNull();
    expect(resolveBatchReleaseBanner(null)).toBeNull();
  });

  it('deploying 时按批次阶段展示，其余字段透传且不可关闭', () => {
    const view = resolveBatchReleaseBanner(buildProgress());
    expect(view).toMatchObject({
      batchIndex: 3,
      closable: false,
      kind: 'updating',
      newImageTag: 'v1.8.3',
      oldImageTag: 'v1.8.2',
      replicas: 20,
      statusTag: { key: '正在更新实例', theme: 'info' },
      updatedReplicas: 8,
    });
  });
});

describe('resolveBatchReleaseBanner：按 batchPhase 决定按钮', () => {
  it('manual-paused：继续发布可用、暂停不可用，回退/跳过在更多里', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'manual-paused' }));
    expect(view?.kind).toBe('paused');
    expect(actionsOf(view)).toEqual(['continue', 'rollback']);
    // 暂停态下没有暂停入口
    expect(actionsOf(view)).not.toContain('pause');
    expect(overflowOf(view)).toEqual(['backward', 'skip']);
    expect(view?.statusTag).toEqual({ key: '等待手动继续', theme: 'warning' });
  });

  it('timed-paused 已到期（或无倒计时数据）：按钮与人工暂停一致，Tag 为已暂停', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'timed-paused' }));
    expect(view?.kind).toBe('paused');
    expect(actionsOf(view)).toEqual(['continue', 'rollback']);
    expect(overflowOf(view)).toEqual(['backward', 'skip']);
    expect(view?.statusTag).toEqual({ key: '已暂停', theme: 'warning' });
  });

  it('timed-paused 未到期：额外提供暂停入口可提前转人工，标题 Tag 为「定时暂停中 · mm:ss 后自动继续」', () => {
    const pauseStartTime = new Date(Date.now() - 10_000).toISOString();
    const view = resolveBatchReleaseBanner(
      buildProgress({
        batchPhase: 'timed-paused',
        batches: [
          { target: '1', pauseMode: 'permanent' },
          { target: '50%', pauseMode: 'permanent' },
          { target: '100%', pauseMode: 'duration', pauseSeconds: 30 },
        ],
        pauseStartTime,
      }),
    );
    expect(view?.kind).toBe('paused');
    expect(actionsOf(view)).toEqual(['continue', 'pause', 'rollback']);
    const remaining = Math.ceil((Date.parse(pauseStartTime) + 30_000 - Date.now()) / 1000);
    expect(view?.statusTag).toEqual({
      key: '定时暂停中 · {0} 后自动继续',
      params: [remainingFormatted(remaining)],
      theme: 'warning',
    });
  });

  it('第 1 批暂停时没有上一批可回退，回退上一批置灰展示', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'manual-paused', currentBatchIndex: 0 }));
    expect(actionsOf(view)).toEqual(['continue', 'rollback']);
    expect(overflowOf(view)).toEqual(['backward:disabled', 'skip']);
  });

  it('updating：可暂停、可回滚，无继续发布', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'updating' }));
    expect(view?.kind).toBe('updating');
    expect(actionsOf(view)).toEqual(['pause', 'rollback']);
    expect(view?.actions[0].labelKey).toBe('暂停');
    expect(overflowOf(view)).toEqual(['backward', 'skip']);
  });

  it('pending（本批尚未开始）：按设计稿仍可暂停、跳过与回滚', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'pending', currentBatchIndex: null }));
    expect(view?.kind).toBe('updating');
    expect(view?.batchIndex).toBeNull();
    expect(actionsOf(view)).toEqual(['pause', 'rollback']);
    // 批次序号未回填时无法确认存在上一批，回退上一批置灰展示
    expect(overflowOf(view)).toEqual(['backward:disabled', 'skip']);
  });

  it('batchPhase=updating 但 currentBatchIndex 尚未回填时，仍必须展示暂停与回滚', () => {
    // 线上真实响应：status=deploying、batchPhase=updating、currentBatchIndex 还没落库（null）。
    // 此时不能因为拿不到批次序号就把所有入口掐掉，否则横幅右侧会空无一物。
    const view = resolveBatchReleaseBanner(
      buildProgress({ batchPhase: 'updating', currentBatchIndex: null, message: 'All steps are completed' }),
    );
    expect(view?.kind).toBe('updating');
    expect(actionsOf(view)).toEqual(['pause', 'rollback']);
  });

  it('全部批次已走完（batchPhase=completed）时进入全量更新态，不提供批次操作', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'completed', currentBatchIndex: null }));
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    expect(view?.statusTag).toEqual({ key: '正在全量更新', theme: 'info' });
  });

  it('灰度批走完推进到全量批（currentBatchIndex 越过批次数）时同样进入全量更新态', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ currentBatchIndex: 3 }));
    expect(view?.kind).toBe('updating');
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    expect(view?.statusTag).toEqual({ key: '正在全量更新', theme: 'info' });
  });

  it('回滚中（currentBatchIndex=-1）：展示且没有可执行操作，标题 Tag 为正在清理灰度资源', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ currentBatchIndex: -1 }));
    expect(view?.kind).toBe('rollingBack');
    expect(view?.batchIndex).toBeNull();
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    expect(view?.statusTag).toEqual({ key: '正在清理灰度资源', theme: 'info' });
  });

  it('第 1 批 updating 时回退上一批置灰展示', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'updating', currentBatchIndex: 0 }));
    expect(overflowOf(view)).toEqual(['backward:disabled', 'skip']);
  });
});

describe('isBatchReleaseBusy', () => {
  it('部署中 / 暂停 / 回滚中 / 失败都阻塞部署入口', () => {
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress()))).toBe(true);
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress({ batchPhase: 'manual-paused' })))).toBe(true);
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress({ currentBatchIndex: -1 })))).toBe(true);
    // 失败后必须先重试或回滚收尾，此时不允许并发发起新部署
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress({ status: 'failed' })))).toBe(true);
  });

  it('完成 / 已回滚与无横幅都不阻塞', () => {
    expect(isBatchReleaseBusy(null)).toBe(false);
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress({ status: 'deployed' })))).toBe(false);
    expect(isBatchReleaseBusy(resolveBatchReleaseBanner(buildProgress({ status: 'rolled-back' })))).toBe(false);
  });
});

describe('终态关闭记忆', () => {
  const completed = resolveBatchReleaseBanner(buildProgress({ status: 'deployed' }));
  const rolledBack = resolveBatchReleaseBanner(buildProgress({ status: 'rolled-back' }));
  const key = buildBatchReleaseDismissKey('app-1', 'stag', 'deploy-1');

  it('key 由应用 + 环境 + 部署记录拼接，缺一返回空串', () => {
    expect(key).toBe('app-1:stag:deploy-1');
    expect(buildBatchReleaseDismissKey('', 'stag', 'deploy-1')).toBe('');
    expect(buildBatchReleaseDismissKey('app-1', '', 'deploy-1')).toBe('');
    expect(buildBatchReleaseDismissKey('app-1', 'stag', '')).toBe('');
  });

  it('已完成与已回滚都参与关闭记忆', () => {
    expect(isBatchReleaseBannerDismissed(completed, key, key)).toBe(true);
    expect(isBatchReleaseBannerDismissed(rolledBack, key, key)).toBe(true);
  });

  it('命中同一发布记录时隐藏，新 deployID 重新出现', () => {
    expect(isBatchReleaseBannerDismissed(completed, key, 'app-1:stag:deploy-2')).toBe(false);
    expect(
      isBatchReleaseBannerDismissed(
        resolveBatchReleaseBanner(buildProgress({ deployID: 'deploy-2', status: 'deployed' })),
        buildBatchReleaseDismissKey('app-1', 'stag', 'deploy-2'),
        key,
      ),
    ).toBe(false);
  });

  it('进行中 / 失败态不参与关闭记忆', () => {
    expect(isBatchReleaseBannerDismissed(resolveBatchReleaseBanner(buildProgress()), key, key)).toBe(false);
    expect(
      isBatchReleaseBannerDismissed(resolveBatchReleaseBanner(buildProgress({ status: 'failed' })), key, key),
    ).toBe(false);
    expect(isBatchReleaseBannerDismissed(null, key, key)).toBe(false);
  });
});

describe('进度补查门控', () => {
  it('没有部署记录或非分批发布记录时都不补查', () => {
    expect(shouldFetchBatchReleaseProgress(null, null)).toBe(false);
    expect(shouldFetchBatchReleaseProgress(buildLatest({ inProgressReleaseMode: '', releaseMode: 'all' }), null)).toBe(
      false,
    );
    expect(
      shouldFetchBatchReleaseProgress(
        buildLatest({ inProgressReleaseMode: '', releaseMode: 'all', stage: 'build' }),
        null,
      ),
    ).toBe(false);
  });

  it('构建 + 自动部署记录比部署记录新时，仍按 inProgressReleaseMode 判定为分批发布进行中', () => {
    const latest = buildLatest({ releaseMode: 'all', stage: 'build' });
    expect(hasCanaryDeployRecord(latest)).toBe(true);
    expect(shouldFetchBatchReleaseProgress(latest, null)).toBe(true);
  });

  it('进行中分批发布每轮都补查', () => {
    expect(shouldFetchBatchReleaseProgress(buildLatest(), null)).toBe(true);
    expect(shouldFetchBatchReleaseProgress(buildLatest(), buildProgress())).toBe(true);
  });

  it('同一 deployID 的终态进度命中缓存后不再重复请求', () => {
    ['deployed', 'failed', 'rolled-back'].forEach(status => {
      // 真实数据：分批发布结束后后端会清空 inProgressReleaseMode。
      expect(
        shouldFetchBatchReleaseProgress(
          buildLatest({ status, inProgressReleaseMode: '' }),
          buildProgress({ status }),
        ),
      ).toBe(false);
    });
  });

  it('重新部署复用同一 deployID 且后端标记灰度进行中时，终态缓存过期必须重新补查', () => {
    expect(
      shouldFetchBatchReleaseProgress(
        buildLatest({ status: 'deploying' }),
        buildProgress({ status: 'deployed', batchPhase: 'completed' }),
      ),
    ).toBe(true);
  });

  it('新部署产生新 deployID 时重新补查', () => {
    expect(
      shouldFetchBatchReleaseProgress(
        buildLatest({ deployID: 'deploy-2' }),
        buildProgress({ deployID: 'deploy-1', status: 'deployed' }),
      ),
    ).toBe(true);
  });

  it('分批发布已结束（releaseMode=canary 且无进行中标记）仍补查一次以清理横幅', () => {
    const latest = buildLatest({ inProgressReleaseMode: '', releaseMode: 'canary', status: 'deployed' });
    expect(hasCanaryDeployRecord(latest)).toBe(true);
    expect(shouldFetchBatchReleaseProgress(latest, null)).toBe(true);
  });

  it('未部署（移除部署后 latest-status 仍带历史 canary 记录）时不补查', () => {
    const uninstalled = buildLatest({ hasDeployRecord: false, status: 'uninstalled' });
    expect(hasCanaryDeployRecord(uninstalled)).toBe(false);
    expect(shouldFetchBatchReleaseProgress(uninstalled, null)).toBe(false);
  });

  it('构建阶段 hasDeployRecord=false 时仍视为有部署记录（旧实例可能仍在运行）', () => {
    const building = buildLatest({ hasDeployRecord: false, stage: 'build' });
    expect(hasCanaryDeployRecord(building)).toBe(true);
    expect(shouldFetchBatchReleaseProgress(building, null)).toBe(true);
  });

  it('同一 deployID 已确认无进度后不再重复请求，换 deployID 后重新请求', () => {
    const finished = buildLatest({ inProgressReleaseMode: '', releaseMode: 'canary', status: 'deployed' });
    expect(shouldFetchBatchReleaseProgress(finished, null, 'deploy-1')).toBe(false);
    expect(shouldFetchBatchReleaseProgress(finished, null, 'deploy-0')).toBe(true);
    expect(
      shouldFetchBatchReleaseProgress(
        buildLatest({ deployID: 'deploy-2', inProgressReleaseMode: '', releaseMode: 'canary', status: 'deployed' }),
        null,
        'deploy-1',
      ),
    ).toBe(true);
    // 后端标记为进行中的分批发布不受「空结果」记忆影响，避免刚发起的发布被挡住
    expect(shouldFetchBatchReleaseProgress(buildLatest(), null, 'deploy-1')).toBe(true);
  });
});

describe('resolveBatchReleaseSuccessMessage', () => {
  it('继续发布带当前批次号，失败态复用重试文案', () => {
    const paused = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'manual-paused' }));
    expect(resolveBatchReleaseSuccessMessage('continue', paused)).toEqual({
      key: '已继续发布，开始执行第 {0} 批',
      params: [3],
    });
    const failed = resolveBatchReleaseBanner(buildProgress({ status: 'failed' }));
    expect(resolveBatchReleaseSuccessMessage('continue', failed)).toEqual({
      key: '已重试，开始执行第 {0} 批',
      params: [3],
    });
  });

  it('回滚提示带回滚目标版本', () => {
    const view = resolveBatchReleaseBanner(buildProgress());
    expect(resolveBatchReleaseSuccessMessage('rollback', view)).toEqual({
      key: '已开始回滚，实例将恢复为 {0}',
      params: ['v1.8.2'],
    });
  });

  it('回退 / 跳过 / 暂停各自有独立提示', () => {
    const view = resolveBatchReleaseBanner(buildProgress());
    expect(resolveBatchReleaseSuccessMessage('backward', view).key).toBe('已回退到上一批');
    expect(resolveBatchReleaseSuccessMessage('skip', view).key).toBe('已跳过剩余批次，正在全量更新剩余实例');
    expect(resolveBatchReleaseSuccessMessage('pause', view).key).toBe('已暂停分批发布');
  });
});

describe('buildBatchReleaseSteps：批次步骤视图模型', () => {
  it('进行中：当前批 loading，之前批已完成，之后批未开始', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'updating' }));
    expect(steps).toHaveLength(3);
    expect(steps[0]).toMatchObject({ batchIndex: 1, stepStatus: 'done', tagKey: '已完成', tagTheme: 'success' });
    expect(steps[1]).toMatchObject({ batchIndex: 2, stepStatus: 'done', tagKey: '已完成', tagTheme: 'success' });
    expect(steps[2]).toMatchObject({ batchIndex: 3, stepStatus: 'loading', tagKey: '正在更新', tagTheme: 'info' });
  });

  it('暂停态：当前批不带 per-step 状态（保留数字节点由 curStep 高亮），手动暂停 Tag 为等待手动继续', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'manual-paused' }));
    expect(steps[2]).toMatchObject({ stepStatus: undefined, tagKey: '等待手动继续', tagTheme: 'warning' });
    expect(steps[0].stepStatus).toBe('done');
  });

  it('定时暂停：Tag 带 mm:ss 倒计时，换算不出时回退等待手动继续', () => {
    const pauseStartTime = new Date(Date.now() - 12_000).toISOString();
    const steps = buildBatchReleaseSteps(
      buildProgress({
        batchPhase: 'timed-paused',
        batches: [
          { target: '1', pauseMode: 'permanent' },
          { target: '50%', pauseMode: 'permanent' },
          { target: '100%', pauseMode: 'duration', pauseSeconds: 30 },
        ],
        pauseStartTime,
      }),
    );
    const remaining = Math.ceil((Date.parse(pauseStartTime) + 30_000 - Date.now()) / 1000);
    expect(steps[2]).toMatchObject({
      stepStatus: undefined,
      tagKey: '暂停中 {0}',
      tagParams: [remainingFormatted(remaining)],
      tagTheme: 'warning',
    });
    // pauseStartTime 缺失 / 已到期时回退为已暂停
    const fallback = buildBatchReleaseSteps(buildProgress({ batchPhase: 'timed-paused' }));
    expect(fallback[2]).toMatchObject({ tagKey: '已暂停' });
  });

  it('倒计时随注入时间逐秒递减，不等轮询', () => {
    const pauseStartTime = new Date(TIMESTAMP_NOW - 10_000).toISOString();
    const progress = buildProgress({
      batchPhase: 'timed-paused',
      batches: [
        { target: '1', pauseMode: 'permanent' },
        { target: '50%', pauseMode: 'permanent' },
        { target: '100%', pauseMode: 'duration', pauseSeconds: 30 },
      ],
      pauseStartTime,
    });
    const first = buildBatchReleaseSteps(progress, TIMESTAMP_NOW)[2];
    const second = buildBatchReleaseSteps(progress, TIMESTAMP_NOW + 1000)[2];
    expect(first.tagParams).toEqual(['00:20']);
    expect(second.tagParams).toEqual(['00:19']);
  });

  it('标题状态 Tag 与步骤 Tag 文案区分：标题为「定时暂停中 · mm:ss 后自动继续」，步骤为「暂停中 mm:ss」', () => {
    const pauseStartTime = new Date(TIMESTAMP_NOW - 10_000).toISOString();
    const progress = buildProgress({
      batchPhase: 'timed-paused',
      batches: [
        { target: '1', pauseMode: 'permanent' },
        { target: '50%', pauseMode: 'permanent' },
        { target: '100%', pauseMode: 'duration', pauseSeconds: 30 },
      ],
      pauseStartTime,
    });
    const view = resolveBatchReleaseBanner(progress);
    expect(view?.statusTag).toMatchObject({ key: '定时暂停中 · {0} 后自动继续', params: ['00:20'] });
    const steps = buildBatchReleaseSteps(progress, TIMESTAMP_NOW);
    expect(steps[2]).toMatchObject({ tagKey: '暂停中 {0}', tagParams: ['00:20'], tagTheme: 'warning' });
  });

  it('批次序号尚未回填时全部批次未开始', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'updating', currentBatchIndex: null }));
    expect(steps.map(step => step.tagKey)).toEqual(['未开始', '未开始', '未开始']);
    expect(steps.every(step => step.stepStatus === undefined)).toBe(true);
  });

  it('失败态：当前批 error，之前批已完成，之后批未开始', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ status: 'failed' }));
    expect(steps[0]).toMatchObject({ stepStatus: 'done', tagKey: '已完成' });
    expect(steps[1]).toMatchObject({ stepStatus: 'done', tagKey: '已完成' });
    expect(steps[2]).toMatchObject({ stepStatus: 'error', tagKey: '失败', tagTheme: 'danger' });
  });

  it('终态与回滚中：全部批次统一收敛为已完成', () => {
    for (const overrides of [
      { batchPhase: 'completed', status: 'deployed' },
      { batchPhase: 'completed', status: 'rolled-back' },
      { currentBatchIndex: -1 },
    ]) {
      const steps = buildBatchReleaseSteps(buildProgress(overrides));
      expect(steps.every(step => step.stepStatus === 'done' && step.tagKey === '已完成')).toBe(true);
    }
  });

  it('末批非全量时追加「全量更新剩余实例」节点，口径与发布预览一致', () => {
    // 末批 50% 非全量 → 系统补齐全量批；当前批（index=1）仍在补齐全量批之前，该节点未开始
    const steps = buildBatchReleaseSteps(
      buildProgress({
        batches: [
          { target: '1', pauseMode: 'none' },
          { target: '50%', pauseMode: 'duration', pauseSeconds: 30 },
        ],
        currentBatchIndex: 1,
      }),
    );
    expect(steps).toHaveLength(3);
    const autoFull = steps[2];
    expect(autoFull).toMatchObject({
      batchIndex: 3,
      isAutoFull: true,
      stepStatus: undefined,
      tagKey: '未开始',
      title: { key: '全量更新剩余实例' },
      subtitle: { key: '系统自动补齐全量' },
    });
    // 当前批推进到补齐全量批（currentBatchIndex=2）时该节点转为更新中
    const updating = buildBatchReleaseSteps(
      buildProgress({
        batches: [
          { target: '1', pauseMode: 'none' },
          { target: '50%', pauseMode: 'none' },
        ],
      }),
    );
    expect(updating[2]).toMatchObject({ stepStatus: 'loading', tagKey: '正在更新' });
  });

  it('标题口径：百分比部分批「累计更新至 X%」，数量批按实例总数换算', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'updating' }));
    // 第 1 批 target='1' → 累计更新 1 个实例
    expect(steps[0].title).toEqual({ key: '累计更新 {0} 个实例', params: [1] });
    // 第 2 批 target='50%' → 累计更新至 50%
    expect(steps[1].title).toEqual({ key: '累计更新至 {0}', params: ['50%'] });
    // 第 3 批 target='100%' 全量 → 累计更新 20 个实例
    expect(steps[2].title).toEqual({ key: '累计更新 {0} 个实例', params: [20] });
  });

  it('副文本按批次暂停方式区分：永久 / 定时 / 不停', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'updating' }));
    expect(steps[0].subtitle).toEqual({ key: '永久暂停 · 手动继续', params: [] });
    expect(steps[2].subtitle).toEqual({ key: '不停 · 自动继续', params: [] });

    const timed = buildBatchReleaseSteps(
      buildProgress({ batches: [{ target: '100%', pauseMode: 'duration', pauseSeconds: 30 }] }),
    );
    expect(timed[0].subtitle).toEqual({ key: '暂停 {0} 秒 · 自动继续', params: [30] });
  });

  it('无进度时返回空数组', () => {
    expect(buildBatchReleaseSteps(null)).toEqual([]);
  });
});

describe('resolveInstanceCanaryState：实例灰度归属判定', () => {
  it('镜像 Tag 等于本次新版本时为灰度新起', () => {
    expect(resolveInstanceCanaryState('registry.bk.com/app:v1.8.3', 'v1.8.3')).toBe('new');
  });

  it('镜像仍为旧版本时为灰度前', () => {
    expect(resolveInstanceCanaryState('registry.bk.com/app:v1.8.2', 'v1.8.3')).toBe('before');
  });

  it('实例镜像缺失或新版本 Tag 缺失时一律按灰度前兜底', () => {
    expect(resolveInstanceCanaryState(undefined, 'v1.8.3')).toBe('before');
    expect(resolveInstanceCanaryState('', 'v1.8.3')).toBe('before');
    expect(resolveInstanceCanaryState('registry.bk.com/app:v1.8.3', '')).toBe('before');
    expect(resolveInstanceCanaryState('registry.bk.com/app:v1.8.3', undefined as unknown as string)).toBe('before');
  });
});
