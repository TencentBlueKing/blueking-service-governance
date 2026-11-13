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
 *   7. 终态关闭记忆是「多条记录」：多个环境各自关闭互不覆盖，追加时去重并限制条数
 */
import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  type BatchReleaseBannerView,
  appendBatchReleaseDismissKey,
  buildBatchReleaseDismissKey,
  buildBatchReleaseSteps,
  formatBatchReleaseDuration,
  formatBatchReleaseStartedAt,
  hasCanaryDeployRecord,
  isBatchReleaseBannerDismissed,
  isBatchReleaseBusy,
  resolveBatchReleaseBanner,
  resolveBatchReleaseDurationSeconds,
  resolveBatchReleaseSuccessMessage,
  resolveInstanceCanaryState,
  resolvePercentAccumulatedCount,
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
    expect(view?.iconKind).toBe('check');
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    expect(view?.statusTag).toEqual({ key: '已完成', theme: 'success' });
  });

  it('已回滚（status=rolled-back）展示非错误提示且可关闭、无操作按钮', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ status: 'rolled-back' }));
    expect(view?.kind).toBe('rolledBack');
    expect(view?.closable).toBe(true);
    expect(view?.iconKind).toBe('info');
    expect(actionsOf(view)).toEqual([]);
    expect(overflowOf(view)).toEqual([]);
    // 回滚态设计稿未覆盖，不展示标题 Tag
    expect(view?.statusTag).toBeNull();
  });

  it('失败（status=failed）展示失败原因并提供重试与回滚，且不可关闭', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ message: '2 个实例健康检查未通过', status: 'failed' }));
    expect(view?.kind).toBe('failed');
    expect(view?.closable).toBe(false);
    expect(view?.message).toBe('2 个实例健康检查未通过');
    // 设计稿：失败态主入口「重试」（复用 continue 动作），回退上一批收进更多，回滚平铺
    expect(actionsOf(view)).toEqual(['continue', 'rollback']);
    expect(view?.actions[0].labelKey).toBe('重试');
    expect(view?.iconKind).toBe('warning');
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
      iconKind: 'loading',
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
    expect(view?.iconKind).toBe('pause');
    expect(actionsOf(view)).toEqual(['continue', 'rollback']);
    // 暂停态下没有暂停入口
    expect(actionsOf(view)).not.toContain('pause');
    expect(overflowOf(view)).toEqual(['backward', 'skip']);
    expect(view?.statusTag).toEqual({ key: '等待手动继续', theme: 'warning' });
  });

  it('timed-paused 已到期（或无倒计时数据）：按钮与人工暂停一致，Tag 为已暂停', () => {
    const view = resolveBatchReleaseBanner(buildProgress({ batchPhase: 'timed-paused' }));
    expect(view?.kind).toBe('paused');
    // 图标按暂停方式区分：定时暂停用时钟，人工/永久暂停用暂停符号
    expect(view?.iconKind).toBe('time');
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
    expect(view?.iconKind).toBe('loading');
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
    expect(isBatchReleaseBannerDismissed(completed, key, [key])).toBe(true);
    expect(isBatchReleaseBannerDismissed(rolledBack, key, [key])).toBe(true);
  });

  it('命中同一发布记录时隐藏，新 deployID 重新出现', () => {
    expect(isBatchReleaseBannerDismissed(completed, key, ['app-1:stag:deploy-2'])).toBe(false);
    expect(
      isBatchReleaseBannerDismissed(
        resolveBatchReleaseBanner(buildProgress({ deployID: 'deploy-2', status: 'deployed' })),
        buildBatchReleaseDismissKey('app-1', 'stag', 'deploy-2'),
        [key],
      ),
    ).toBe(false);
  });

  it('多个环境的关闭记忆同时生效，互不覆盖', () => {
    const keyStag = buildBatchReleaseDismissKey('app-1', 'stag', 'deploy-1');
    const keyProd = buildBatchReleaseDismissKey('app-1', 'prod', 'deploy-9');
    const dismissed = [keyStag, keyProd];

    expect(isBatchReleaseBannerDismissed(completed, keyStag, dismissed)).toBe(true);
    expect(isBatchReleaseBannerDismissed(completed, keyProd, dismissed)).toBe(true);
    // 未关闭过的环境照常展示
    const keyTest = buildBatchReleaseDismissKey('app-1', 'test', 'deploy-1');
    expect(isBatchReleaseBannerDismissed(completed, keyTest, dismissed)).toBe(false);
  });

  it('进行中 / 失败态不参与关闭记忆', () => {
    expect(isBatchReleaseBannerDismissed(resolveBatchReleaseBanner(buildProgress()), key, [key])).toBe(false);
    expect(
      isBatchReleaseBannerDismissed(resolveBatchReleaseBanner(buildProgress({ status: 'failed' })), key, [key]),
    ).toBe(false);
    expect(isBatchReleaseBannerDismissed(null, key, [key])).toBe(false);
    // 空 key（进度尚未返回）不得命中任何记录
    expect(isBatchReleaseBannerDismissed(completed, '', [''])).toBe(false);
  });

  it('追加关闭记忆时保留其它环境记录、去重后置末，并限制最大条数', () => {
    const keyA = buildBatchReleaseDismissKey('app-1', 'stag', 'deploy-1');
    const keyB = buildBatchReleaseDismissKey('app-1', 'prod', 'deploy-2');

    expect(appendBatchReleaseDismissKey([], keyA)).toEqual([keyA]);
    // 在环境 B 关闭不能冲掉环境 A 的记录
    expect(appendBatchReleaseDismissKey([keyA], keyB)).toEqual([keyA, keyB]);
    // 重复关闭同一记录只保留一条（且移到末尾）
    expect(appendBatchReleaseDismissKey([keyA, keyB], keyA)).toEqual([keyB, keyA]);
    // 空 key 不写入
    expect(appendBatchReleaseDismissKey([keyA], '')).toEqual([keyA]);

    // 超过上限时丢弃最旧记录，新记录必定保留
    const many = Array.from({ length: 50 }, (_, index) => `app-1:stag:deploy-${index}`);
    const appended = appendBatchReleaseDismissKey(many, 'app-1:stag:deploy-new');
    expect(appended).toHaveLength(50);
    expect(appended[0]).toBe('app-1:stag:deploy-1');
    expect(appended.at(-1)).toBe('app-1:stag:deploy-new');
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
        shouldFetchBatchReleaseProgress(buildLatest({ status, inProgressReleaseMode: '' }), buildProgress({ status })),
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

  it('标题口径：个数批直接带实例数，百分比批带百分比与换算后的累计实例数', () => {
    const steps = buildBatchReleaseSteps(buildProgress({ batchPhase: 'updating' }));
    // 第 1 批 target='1' → 1 个实例
    expect(steps[0].title).toEqual({ key: '{0} 个实例', params: [1] });
    // 第 2 批 target='50%' → 50%（10 个），20 × 50% = 10
    expect(steps[1].title).toEqual({ key: '{0}（{1} 个）', params: ['50%', 10] });
    // 第 3 批 target='100%' 全量 → 20 个实例
    expect(steps[2].title).toEqual({ key: '{0} 个实例', params: [20] });
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

describe('resolvePercentAccumulatedCount：百分比批步骤条标题的累计实例数（Math.round 口径）', () => {
  it('按 Math.round 换算：20 × 25% = 5，20 × 33% = 6.6 → 7', () => {
    expect(resolvePercentAccumulatedCount(25, 20)).toBe(5);
    expect(resolvePercentAccumulatedCount(33, 20)).toBe(7);
    expect(resolvePercentAccumulatedCount(50, 20)).toBe(10);
    expect(resolvePercentAccumulatedCount(100, 20)).toBe(20);
  });

  it('至少 1 个、不超过实例总数', () => {
    // 1% 向下舍入为 0 时收敛为 1
    expect(resolvePercentAccumulatedCount(1, 20)).toBe(1);
    // 超过 100% 或实例总数封顶
    expect(resolvePercentAccumulatedCount(120, 20)).toBe(20);
  });

  it('参数缺失或非法时返回 0（组件按空值兜底）', () => {
    expect(resolvePercentAccumulatedCount(undefined, 20)).toBe(0);
    expect(resolvePercentAccumulatedCount(Number.NaN, 20)).toBe(0);
    expect(resolvePercentAccumulatedCount(50, 0)).toBe(0);
  });
});

describe('formatBatchReleaseStartedAt：浮层开始时间（MM-DD HH:mm）', () => {
  it('按本地时区格式化为 MM-DD HH:mm', () => {
    const iso = '2026-11-13T10:19:05.826Z';
    expect(formatBatchReleaseStartedAt(iso)).toBe(dayjs(iso).format('MM-DD HH:mm'));
  });

  it('空值 / 非法值 / Go 零值时间展示 --', () => {
    expect(formatBatchReleaseStartedAt(undefined)).toBe('--');
    expect(formatBatchReleaseStartedAt('')).toBe('--');
    expect(formatBatchReleaseStartedAt('not-a-time')).toBe('--');
    expect(formatBatchReleaseStartedAt('0001-01-01T00:00:00Z')).toBe('--');
  });
});

describe('resolveBatchReleaseDurationSeconds：浮层耗时秒数', () => {
  const startedAt = '2026-11-13T10:00:00Z';

  it('进行中按 now - startedAt 换算（192 秒）', () => {
    const now = dayjs(startedAt).add(192, 'second').valueOf();
    expect(resolveBatchReleaseDurationSeconds({ endedAt: '', isTerminal: false, now, startedAt })).toBe(192);
    // 终态但 endedAt 缺失 / 零值时退化为 now 口径
    expect(resolveBatchReleaseDurationSeconds({ isTerminal: true, now, startedAt })).toBe(192);
    expect(
      resolveBatchReleaseDurationSeconds({ endedAt: '0001-01-01T00:00:00Z', isTerminal: true, now, startedAt }),
    ).toBe(192);
  });

  it('终态按 endedAt - startedAt 换算，now 不参与', () => {
    const now = dayjs(startedAt).add(999, 'second').valueOf();
    const endedAt = dayjs(startedAt).add(65, 'second').toISOString();
    expect(resolveBatchReleaseDurationSeconds({ endedAt, isTerminal: true, now, startedAt })).toBe(65);
  });

  it('结束早于开始（时钟偏移）收敛为 0；startedAt 无效返回 0', () => {
    const now = dayjs(startedAt).valueOf();
    const endedAt = dayjs(startedAt).subtract(10, 'second').toISOString();
    expect(resolveBatchReleaseDurationSeconds({ endedAt, isTerminal: true, now, startedAt })).toBe(0);
    expect(resolveBatchReleaseDurationSeconds({ isTerminal: false, now, startedAt: '' })).toBe(0);
  });
});

describe('formatBatchReleaseDuration：秒数 → 时长文案', () => {
  it('按时分秒逐级降位', () => {
    expect(formatBatchReleaseDuration(3672)).toEqual({ key: '{0} 时 {1} 分 {2} 秒', params: [1, 1, 12] });
    expect(formatBatchReleaseDuration(192)).toEqual({ key: '{0} 分 {1} 秒', params: [3, 12] });
    expect(formatBatchReleaseDuration(45)).toEqual({ key: '{0} 秒', params: [45] });
  });

  it('负数与 0 收敛为「0 秒」', () => {
    expect(formatBatchReleaseDuration(-5)).toEqual({ key: '{0} 秒', params: [0] });
    expect(formatBatchReleaseDuration(0)).toEqual({ key: '{0} 秒', params: [0] });
  });
});

describe('resolveInstanceCanaryState：实例灰度归属判定', () => {
  const progress = { newImageTag: 'v1.8.3', workloadUpdateRevision: 'app-7d9f8c5b6d' };

  it('实例工作负载版本等于本次新版本版本号时为灰度新起', () => {
    expect(
      resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.7.0', workloadRevision: 'app-7d9f8c5b6d' }, progress),
    ).toBe('new');
  });

  it('实例工作负载版本仍为旧版本版本号时为灰度前（即使镜像 Tag 恰好一致）', () => {
    expect(
      resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.3', workloadRevision: 'app-5c4b3a2f1e' }, progress),
    ).toBe('before');
  });

  it('灰度完成后镜像 Tag 与进度快照口径不一致时，仍按版本号判定为灰度新起', () => {
    // 回归：此前以镜像 Tag 判定，灰度已完成但 Tag 对不上时会整片误判为「灰度前」
    expect(
      resolveInstanceCanaryState(
        { image: 'registry.bk.com/app:202610281105', workloadRevision: 'app-7d9f8c5b6d' },
        { newImageTag: '202610281605', workloadUpdateRevision: 'app-7d9f8c5b6d' },
      ),
    ).toBe('new');
  });

  it('进度尚未回填工作负载版本时回退镜像 Tag 判定', () => {
    expect(resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.3' }, { newImageTag: 'v1.8.3' })).toBe('new');
    expect(resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.2' }, { newImageTag: 'v1.8.3' })).toBe(
      'before',
    );
    // 实例版本号缺失同样回退镜像 Tag
    expect(resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.3', workloadRevision: '' }, progress)).toBe(
      'new',
    );
  });

  it('灰度已完成且全量更新时，即使版本号与镜像 Tag 都对不上也判为灰度新起', () => {
    const completed = {
      status: 'deployed',
      replicas: 4,
      updatedReplicas: 4,
      newImageTag: '202610281605',
      workloadUpdateRevision: '',
    };
    expect(
      resolveInstanceCanaryState({ image: 'registry.bk.com/app:202610281105', workloadRevision: '' }, completed),
    ).toBe('new');
  });

  it('未完成全量更新或非已完成态时，不套用「已完成」兜底', () => {
    const deploying = { status: 'deploying', replicas: 4, updatedReplicas: 2, newImageTag: 'v1.8.3' };
    expect(resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.2' }, deploying)).toBe('before');
    // 已完成但总数未知（replicas=0）时不能直接判为新版本
    expect(
      resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.2' }, { status: 'deployed', replicas: 0 }),
    ).toBe('before');
  });

  it('版本号与镜像均缺失时一律按灰度前兜底', () => {
    expect(resolveInstanceCanaryState(undefined, progress)).toBe('before');
    expect(resolveInstanceCanaryState(null, null)).toBe('before');
    expect(resolveInstanceCanaryState({}, {})).toBe('before');
    expect(resolveInstanceCanaryState({ image: 'registry.bk.com/app:v1.8.3' }, {})).toBe('before');
  });
});
