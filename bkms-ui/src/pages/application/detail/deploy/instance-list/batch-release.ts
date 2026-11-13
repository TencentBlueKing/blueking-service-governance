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

import dayjs, { type Dayjs } from 'dayjs';
import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';

import {
  isFullTarget,
  parseBatchTarget,
  resolveBatchUpdatedCount,
  shouldAppendAutoFullBatch,
} from '../release-preview';

import type { BatchReleaseProgressOutputObj, LatestDeployStatus } from '~/@types/v1/deploy';
import type { AppInstanceOutputObj } from '~/@types/v1/instance';

/**
 * 分批发布控制横幅的纯函数层。
 *
 * 接口字段（status / batchPhase / currentBatchIndex）到「可见状态 + 可执行操作」的收敛全部集中在这里，
 * 组件只负责按 view 渲染，便于单测覆盖批次阶段映射、按钮集合与请求门控。
 */

/** 与后端 operateBatchRelease 的 action 一一对应。 */
export type BatchReleaseAction = 'backward' | 'continue' | 'pause' | 'rollback' | 'skip';

/** 需要二次确认的操作配置。 */
export interface BatchReleaseActionConfirm {
  /** 确认弹窗主按钮文案 */
  confirmTextKey: string;
  /** 确认弹窗主按钮主题：回滚用危险按钮，其余用主按钮 */
  confirmTheme: 'danger' | 'primary';
  /** 确认弹窗灰底说明 */
  descriptionKey: string;
  /** 确认弹窗标题 */
  titleKey: string;
}

export interface BatchReleaseBannerAction {
  action: BatchReleaseAction;
  /** 二次确认配置；缺省表示点击后直接执行 */
  confirm?: BatchReleaseActionConfirm;
  /** 不可用操作（如第 1 批的「回退上一批」）：置灰展示并拦截点击，表达「存在但不可用」 */
  disabled?: boolean;
  /** 按钮文案 i18n key */
  labelKey: string;
  /** primary=蓝底白字，danger=白底红边，default=白底灰边 */
  theme: 'danger' | 'default' | 'primary';
}

export interface BatchReleaseBannerView {
  /** 横幅右侧直接展示的操作按钮 */
  actions: BatchReleaseBannerAction[];
  /** 当前批次序号（从 1 开始）；未进入批次、回滚或全部批次已走完时为 null */
  batchIndex: null | number;
  /** 仅完成态可关闭：关闭后按 deployID 记住，下一次发布重新出现 */
  closable: boolean;
  /** 左侧图标语义：loading=过程态（更新中/回滚中），time=定时暂停，pause=手动/永久暂停，check=发布成功，warning=失败，info=已回滚 */
  iconKind: BatchReleaseIconKind;
  kind: BatchReleaseKind;
  /** 失败原因/批次提示，可能为空，由组件兜底文案 */
  message: string;
  newImageTag: string;
  oldImageTag: string;
  /** 收进「更多」下拉的低频操作 */
  overflowActions: BatchReleaseBannerAction[];
  replicas: number;
  /** 标题状态 Tag；已回滚为终态提示不展示（回滚中按设计稿展示「正在清理灰度资源」） */
  statusTag: BatchReleaseStatusTag | null;
  updatedReplicas: number;
}

/** 横幅左侧图标语义（设计稿图标规范），组件层映射为具体 bkms-icon 类名。 */
export type BatchReleaseIconKind = 'check' | 'info' | 'loading' | 'pause' | 'time' | 'warning';

/**
 * 横幅可见状态。
 * - completed：status=deployed（全部批次已完成，展示成功提示，可关闭）
 * - failed：status=failed（当前批次失败，需重试或回滚）
 * - paused：manual-paused / timed-paused（人工暂停或策略配置的永久暂停、定时暂停）
 * - rollingBack：currentBatchIndex=-1（正在回滚）
 * - rolledBack：status=rolled-back（回滚已完成，所有实例恢复旧版本，非错误态，可关闭）
 * - updating：updating / pending（本批更新中或尚未开始）
 */
export type BatchReleaseKind = 'completed' | 'failed' | 'paused' | 'rolledBack' | 'rollingBack' | 'updating';

/** 操作成功提示的 i18n 描述，交由组件层翻译。 */
export interface BatchReleaseMessageDescriptor {
  key: string;
  params: (number | string)[];
}

/** 标题旁的状态 Tag（设计稿七态：正在更新实例 / 正在全量更新 / 等待手动继续 / 暂停中 mm:ss / 已暂停 / 失败 / 已完成）。 */
export interface BatchReleaseStatusTag {
  key: string;
  params?: (number | string)[];
  /** 与步骤 Tag 同一套语义配色 */
  theme: 'danger' | 'default' | 'info' | 'success' | 'warning';
}

/**
 * 批次步骤视图模型（buildBatchReleaseSteps 产出，组件只负责渲染）。
 */
export interface BatchReleaseStepVM {
  /** 第 N 批，从 1 开始；系统补齐全量批沿用顺序编号 */
  batchIndex: number;
  /** 是否系统补齐全量批（标题固定「全量更新剩余实例」） */
  isAutoFull: boolean;
  /** bkui Steps per-step 状态；缺省 = 未进行（当前步蓝色数字） */
  stepStatus?: 'done' | 'error' | 'loading';
  /** 副文本 i18n 描述（暂停方式 · 继续方式） */
  subtitle: BatchReleaseMessageDescriptor;
  /** 状态 Tag 文案 key（已完成 / 正在更新 / 暂停中 {0} / 等待手动继续 / 更新失败 / 未开始） */
  tagKey: string;
  /** Tag 文案参数（定时暂停的 mm:ss 倒计时），随轮询刷新 */
  tagParams?: (number | string)[];
  /** Tag 主题：success=已完成 / info=正在更新 / warning=暂停 / danger=失败 / default=未开始 */
  tagTheme: 'danger' | 'default' | 'info' | 'success' | 'warning';
  /** 标题 i18n 描述（key + 参数） */
  title: BatchReleaseMessageDescriptor;
}

/** 灰度归属判定用到的实例字段：版本号为主口径，镜像 Tag 仅为兜底。 */
export type CanaryStateInstance = Pick<AppInstanceOutputObj, 'image' | 'workloadRevision'>;

/**
 * 灰度归属判定用到的进度字段：
 * - workloadUpdateRevision：工作负载 status.updateRevision，实例版本与之相等即为新版本实例；
 * - status / updatedReplicas / replicas：工作负载版本尚未回填时，用「已完成且全量更新」兜底；
 * - newImageTag：上述口径都不可用时的最后兜底（镜像 Tag 比对）。
 */
export type CanaryStateProgress = Pick<
  BatchReleaseProgressOutputObj,
  'newImageTag' | 'replicas' | 'status' | 'updatedReplicas' | 'workloadUpdateRevision'
>;

/** 实例灰度归属：new=灰度新起（已更新到本次新版本），before=灰度前（仍为旧版本） */
export type InstanceCanaryState = 'before' | 'new';

const ACTION_LABELS: Record<BatchReleaseAction, string> = {
  backward: '回退上一批',
  continue: '继续发布',
  pause: '暂停',
  rollback: '回滚',
  skip: '跳过剩余批次',
};

const ROLLBACK_CONFIRM: BatchReleaseActionConfirm = {
  confirmTextKey: '回滚',
  confirmTheme: 'danger',
  descriptionKey: '将结束本次灰度发布，实例恢复为旧版本镜像。',
  titleKey: '确认回滚到旧版本？',
};

const BACKWARD_CONFIRM: BatchReleaseActionConfirm = {
  confirmTextKey: '回退',
  confirmTheme: 'primary',
  descriptionKey: '当前批已更新的实例恢复为旧版本，流量权重回到上一批配置。',
  titleKey: '确认回退到上一批？',
};

const SKIP_CONFIRM: BatchReleaseActionConfirm = {
  confirmTextKey: '跳过并全量',
  confirmTheme: 'primary',
  descriptionKey: '将不再分批暂停，直接全量更新剩余实例，请确认当前版本已验证通过。',
  titleKey: '确认跳过剩余批次？',
};

/** 分批发布已结束的部署记录状态：进度不会再变化，同一 deployID 无需重复查询。 */
const TERMINAL_BATCH_RELEASE_STATUSES = new Set<string>([
  APP_DEPLOY_STATUS.DEPLOYED,
  APP_DEPLOY_STATUS.FAILED,
  APP_DEPLOY_STATUS.ROLLED_BACK,
]);

/** manual-paused：策略配置的永久暂停或人工暂停 */
const MANUAL_PAUSED_BATCH_PHASES = new Set(['manual-paused']);

/** timed-paused：定时暂停，可提前进入下一批 */
const TIMED_PAUSED_BATCH_PHASES = new Set(['timed-paused']);

/** 本批更新中或尚未开始：此时只能暂停（pending 阶段后端不支持暂停，按钮置灰） */
const UPDATING_BATCH_PHASES = new Set(['pending', 'updating']);

/** 步骤条暂停态批次阶段：manual-paused / timed-paused，节点保留数字由 curStep 高亮 */
const PAUSED_PHASES = new Set(['manual-paused', 'timed-paused']);

/** 关闭记忆 key：应用 + 环境 + 部署记录，任一变化都视为新的发布结果。 */
export function buildBatchReleaseDismissKey(appID: string, envName: string, deployID: string): string {
  if (!appID || !envName || !deployID) return '';
  return `${appID}:${envName}:${deployID}`;
}

/**
 * 按下发策略快照与当前进度构建批次步骤视图模型。
 *
 * - 末批非全量时追加「全量更新剩余实例」节点，与发布预览口径一致（shouldAppendAutoFullBatch）
 * - 回滚中（currentBatchIndex=-1）与已回滚 / 已完成终态：全部批次节点按已完成处理
 * - 失败态：当前批次 error，之前批次 done，之后批次未开始
 * - 进行中：当前批次按 batchPhase 区分 loading（更新中）/ 暂停（蓝色数字，仅 curStep 高亮）
 */
export function buildBatchReleaseSteps(
  progress: BatchReleaseProgressOutputObj | null,
  /** 计算倒计时的当前时间；组件传入逐秒刷新的时间源即可让倒计时随秒跳动 */
  now = Date.now(),
): BatchReleaseStepVM[] {
  if (!progress) return [];

  const batches = progress.batches ?? [];
  const replicas = progress.replicas ?? 0;
  const status = progress.status ?? '';
  const currentIndex = progress.currentBatchIndex;
  const batchPhase = progress.batchPhase ?? '';
  // 统一收敛为可比较的局部值，避免在分支里反复做非空断言
  const currentIdx = typeof currentIndex === 'number' && currentIndex >= 0 ? currentIndex : -1;
  const hasCurrentIndex = currentIdx >= 0;
  // 终态 / 回滚中：所有节点统一收敛为已完成，设计稿未对回滚态做批次级区分。
  const isSettled =
    currentIndex === -1 || status === APP_DEPLOY_STATUS.DEPLOYED || status === APP_DEPLOY_STATUS.ROLLED_BACK;
  const isFailed = status === APP_DEPLOY_STATUS.FAILED;

  const steps: BatchReleaseStepVM[] = batches.map((batch, index) => {
    const parsed = parseBatchTarget(batch.target);
    const isPercentPartial = parsed.unit === 'percent' && !isFullTarget(batch.target);
    const title: BatchReleaseMessageDescriptor = isPercentPartial
      ? { key: '{0}（{1} 个）', params: [parsed.text, resolvePercentAccumulatedCount(parsed.value, replicas)] }
      : { key: '{0} 个实例', params: [resolveBatchUpdatedCount(batch.target, replicas)] };

    let tagKey = '未开始';
    let tagParams: (number | string)[] | undefined;
    let tagTheme: BatchReleaseStepVM['tagTheme'] = 'default';
    let stepStatus: BatchReleaseStepVM['stepStatus'];
    if (isSettled || (hasCurrentIndex && index < currentIdx)) {
      tagKey = '已完成';
      tagTheme = 'success';
      stepStatus = 'done';
    } else if (isFailed && hasCurrentIndex && index === currentIdx) {
      tagKey = '失败';
      tagTheme = 'danger';
      stepStatus = 'error';
    } else if (hasCurrentIndex && index === currentIdx) {
      if (batchPhase === 'completed') {
        tagKey = '已完成';
        tagTheme = 'success';
        stepStatus = 'done';
      } else if (PAUSED_PHASES.has(batchPhase)) {
        // 暂停态不带 per-step 状态：保留数字节点，由 curStep 高亮为蓝色。
        // 定时暂停带 mm:ss 倒计时，手动 / 永久暂停展示「等待手动继续」。
        const pauseTag = resolvePausedStepTag(progress, now);
        tagKey = pauseTag.key;
        tagParams = pauseTag.params;
        tagTheme = 'warning';
      } else {
        tagKey = '正在更新';
        tagTheme = 'info';
        stepStatus = 'loading';
      }
    }

    return {
      batchIndex: index + 1,
      isAutoFull: false,
      stepStatus,
      subtitle: resolveBatchPauseSubtitle(batch),
      tagKey,
      ...(tagParams ? { tagParams } : {}),
      tagTheme,
      title,
    };
  });

  // 末批非全量时后端会自动补齐全量批（UpdatingPrimary 阶段），与发布预览保持一致。
  if (shouldAppendAutoFullBatch(batches.map(batch => batch.target))) {
    const autoFullIndex = batches.length;
    let tagKey = '未开始';
    let tagTheme: BatchReleaseStepVM['tagTheme'] = 'default';
    let stepStatus: BatchReleaseStepVM['stepStatus'];
    if (isSettled) {
      tagKey = '已完成';
      tagTheme = 'success';
      stepStatus = 'done';
    } else if (hasCurrentIndex && currentIdx >= autoFullIndex) {
      tagKey = '正在更新';
      tagTheme = 'info';
      stepStatus = 'loading';
    }
    steps.push({
      batchIndex: autoFullIndex + 1,
      isAutoFull: true,
      stepStatus,
      subtitle: { key: '系统自动补齐全量', params: [] },
      tagKey,
      tagTheme,
      title: { key: '全量更新剩余实例', params: [] },
    });
  }

  return steps;
}

/**
 * 部署记录是否为分批发布（进行中或已结束）。
 *
 * 后端约定：判断是否展示分批发布面板必须用 `inProgressReleaseMode`。
 * 最新的「构建 + 自动部署」记录可能比部署记录更新（此时 latest-status 返回的是构建记录），
 * 而集群里的分批发布仍在进行，因此不能按 `stage`/`status` 过滤。
 */
export function hasCanaryDeployRecord(latest: LatestDeployStatus | null): boolean {
  if (!latest) return false;
  // 移除部署后 latest-status 仍会带出历史上那条 canary 记录（hasDeployRecord=false），
  // 此时已没有任何批次或实例可控制，不能继续搭车轮询进度接口。
  // build 阶段沿用实例列表口径：构建不删旧实例，旧的分批发布可能仍在运行。
  if (latest.hasDeployRecord === false && latest.stage !== 'build') return false;
  if (latest.inProgressReleaseMode === 'canary') return true;
  // 进行中标记为空时，再看最新记录本身是否为分批发布，用于完成一次进度查询后清理状态。
  return latest.releaseMode === 'canary';
}

/** 已结束且无需干预的终态：可关闭，也不再阻塞新的部署入口。 */
const DISMISSIBLE_KINDS = new Set<BatchReleaseKind>(['completed', 'rolledBack']);

/** 关闭记忆最多保留的条数：会话内持续累积，加上限避免无限增长。 */
const MAX_DISMISSED_KEYS = 50;

/** 追加一条关闭记忆：已存在则移到末尾，超过上限丢弃最旧的一条。 */
export function appendBatchReleaseDismissKey(dismissedKeys: readonly string[], key: string): string[] {
  const rest = dismissedKeys.filter(item => item && item !== key);
  return key ? [...rest, key].slice(-MAX_DISMISSED_KEYS) : rest;
}

/** 秒数 → 时长文案：按时分秒逐级降位，0 秒展示「0 秒」。 */
export function formatBatchReleaseDuration(seconds: number): BatchReleaseMessageDescriptor {
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) return { key: '{0} 时 {1} 分 {2} 秒', params: [hours, minutes, secs] };
  if (minutes > 0) return { key: '{0} 分 {1} 秒', params: [minutes, secs] };
  return { key: '{0} 秒', params: [secs] };
}

/**
 * 浮层「开始时间」：MM-DD HH:mm（当年发布场景下省略年份，与设计稿一致）。
 * 无数据 / 非法值展示 '--'。
 */
export function formatBatchReleaseStartedAt(startedAt?: string): string {
  const time = parseServerTime(startedAt);
  return time ? time.format('MM-DD HH:mm') : '--';
}

/** 剩余秒 → mm:ss（超过 1 小时分钟位自然进位，如 75:00）。 */
export function formatPauseCountdown(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const mm = Math.floor(total / 60);
  const ss = total % 60;
  return `${String(mm).padStart(2, '0')}:${String(ss).padStart(2, '0')}`;
}

/**
 * 仅完成 / 已回滚这两个终态可关闭；关闭记忆命中同一发布记录时隐藏横幅。
 *
 * 记忆是「多条记录」而不是单条：key 已带环境维度，多个环境各自关闭后要互不覆盖，
 * 否则在环境 B 关闭会冲掉环境 A 的记录，切回环境 A 时横幅又冒出来。
 */
export function isBatchReleaseBannerDismissed(
  view: BatchReleaseBannerView | null,
  currentKey: string,
  dismissedKeys: readonly string[],
): boolean {
  if (!view || !DISMISSIBLE_KINDS.has(view.kind)) return false;
  return !!currentKey && dismissedKeys.includes(currentKey);
}

/**
 * 分批发布是否仍在占用当前环境。
 * 完成 / 已回滚属于无待处理动作的终态，不阻塞新的部署入口；
 * 失败态仍阻塞，因为需要用户先重试或回滚把这次灰度收尾。
 */
export function isBatchReleaseBusy(view: BatchReleaseBannerView | null): boolean {
  return !!view && !DISMISSIBLE_KINDS.has(view.kind);
}

/** 进度是否已进入终态（用于判断横幅是否还需要继续轮询刷新）。 */
export function isTerminalBatchReleaseStatus(status?: string): boolean {
  return TERMINAL_BATCH_RELEASE_STATUSES.has(status ?? '');
}

/**
 * 将进度接口响应收敛为横幅展示模型。
 *
 * - `status=deploying`：展示批次控制横幅（按钮集合见下方规则）；`currentBatchIndex=-1` 时为回滚中
 * - `status=deployed`：展示「灰度发布已完成」成功提示，可关闭
 * - `status=failed`：展示失败原因，提供重试（主入口）/ 回滚入口，回退上一批收进「更多」
 * - `status=rolled-back`：展示「已回滚」提示（非错误态），可关闭
 * - 其余状态（卸载相关 / 取消 / 轮询异常 / 未知）：不展示横幅，结果由部署状态列与实例列表呈现
 */
export function resolveBatchReleaseBanner(
  progress: BatchReleaseProgressOutputObj | null,
): BatchReleaseBannerView | null {
  if (!progress?.status) return null;

  const { currentBatchIndex } = progress;
  const base = {
    batchIndex: typeof currentBatchIndex === 'number' && currentBatchIndex >= 0 ? currentBatchIndex + 1 : null,
    closable: false,
    message: (progress.message ?? '').trim(),
    newImageTag: progress.newImageTag ?? '',
    oldImageTag: progress.oldImageTag ?? '',
    replicas: progress.replicas ?? 0,
    updatedReplicas: progress.updatedReplicas ?? 0,
  };

  // 全部批次已完成：展示成功提示（绿色 Alert + 关闭入口），不再提供批次操作。
  if (progress.status === APP_DEPLOY_STATUS.DEPLOYED) {
    return {
      ...base,
      actions: [],
      closable: true,
      iconKind: 'check',
      kind: 'completed',
      overflowActions: [],
      statusTag: { key: '已完成', theme: 'success' },
    };
  }

  // 回滚已完成：所有实例已恢复旧版本，非错误态，展示提示后允许关闭。
  if (progress.status === APP_DEPLOY_STATUS.ROLLED_BACK) {
    return {
      ...base,
      actions: [],
      closable: true,
      iconKind: 'info',
      kind: 'rolledBack',
      overflowActions: [],
      statusTag: null,
    };
  }

  // 当前批次失败：主入口「重试」（复用 continue 动作）+ 回滚，回退上一批收进「更多」。
  if (progress.status === APP_DEPLOY_STATUS.FAILED) {
    return {
      ...base,
      actions: [
        createAction('continue', 'primary', { labelKey: '重试' }),
        createAction('rollback', 'danger', { confirm: ROLLBACK_CONFIRM }),
      ],
      iconKind: 'warning',
      kind: 'failed',
      overflowActions: [buildBackwardAction(progress)],
      statusTag: { key: '失败', theme: 'danger' },
    };
  }

  if (progress.status !== APP_DEPLOY_STATUS.DEPLOYING) return null;

  // currentBatchIndex=-1 表示正在回滚，此时没有可执行操作。
  if (currentBatchIndex === -1) {
    return {
      ...base,
      actions: [],
      iconKind: 'loading',
      kind: 'rollingBack',
      overflowActions: [],
      statusTag: { key: '正在清理灰度资源', theme: 'info' },
    };
  }

  const batchPhase = progress.batchPhase ?? '';
  // 后端约定：currentBatchIndex 在「尚未进入批次」时可能还没回填（null），
  // 因此按钮是否可用只看 batchPhase，绝不能拿批次序号做门控，否则 updating 态会退化成无任何入口。
  const hasBatchIndex = typeof currentBatchIndex === 'number' && currentBatchIndex >= 0;
  const isManualPaused = MANUAL_PAUSED_BATCH_PHASES.has(batchPhase);
  const isTimedPaused = TIMED_PAUSED_BATCH_PHASES.has(batchPhase);
  const isUpdating = UPDATING_BATCH_PHASES.has(batchPhase);

  // 全量更新中：灰度批走完后系统补齐全量批（batchPhase=completed，或批次序号越过灰度批数）。
  // 该阶段自动收尾，不提供任何操作，避免干扰。
  const isFullUpdating =
    batchPhase === 'completed' ||
    (hasBatchIndex && (progress.batches?.length ?? 0) > 0 && currentBatchIndex >= (progress.batches?.length ?? 0));
  if (isFullUpdating) {
    return {
      ...base,
      actions: [],
      iconKind: 'loading',
      kind: 'updating',
      overflowActions: [],
      statusTag: { key: '正在全量更新', theme: 'info' },
    };
  }

  // batchPhase 缺失但有当前批次号时按「本批更新中」兜底，避免老数据下横幅没有任何操作。
  const inFlight = isManualPaused || isTimedPaused || isUpdating || hasBatchIndex;
  if (!inFlight) {
    return {
      ...base,
      actions: [],
      iconKind: 'loading',
      kind: 'updating',
      overflowActions: [],
      statusTag: { key: '正在更新实例', theme: 'info' },
    };
  }

  /**
   * 本层按钮集合（设计稿「第二层：有批次在执行时」）：
   * - 继续发布：manual-paused / timed-paused
   * - 暂停：updating / pending；timed-paused 未到期可提前转人工暂停
   * - 回退上一批：任意阶段展示，第 1 批（或批次未知）置灰
   * - 跳过剩余批次 / 回滚：任意批次阶段
   */
  const overflowActions = [buildBackwardAction(progress), createAction('skip', 'default', { confirm: SKIP_CONFIRM })];
  const rollbackAction = createAction('rollback', 'danger', { confirm: ROLLBACK_CONFIRM });

  // manual-paused（策略配置的永久暂停或人工暂停）/ timed-paused（定时暂停，可提前进入下一批）
  if (isManualPaused || isTimedPaused) {
    const remaining = isTimedPaused ? resolveTimedPauseRemainingSeconds(progress) : 0;
    return {
      ...base,
      // 定时暂停未到期可「暂停」提前转人工；到期后与永久/手动暂停按钮集合一致。
      actions: [
        createAction('continue', 'primary'),
        ...(isTimedPaused && remaining > 0 ? [createAction('pause', 'default')] : []),
        rollbackAction,
      ],
      // 图标区分暂停方式：定时暂停用时钟，人工/永久暂停用暂停符号。
      iconKind: isTimedPaused ? 'time' : 'pause',
      kind: 'paused',
      overflowActions,
      statusTag: resolvePausedStatusTag(progress, remaining),
    };
  }

  // updating / pending（本批更新中或尚未开始）：暂停 + 回滚
  return {
    ...base,
    actions: [createAction('pause', 'default'), rollbackAction],
    iconKind: 'loading',
    kind: 'updating',
    overflowActions,
    statusTag: { key: '正在更新实例', theme: 'info' },
  };
}

/**
 * 浮层「已耗时 / 总耗时」的秒数换算。
 * - 进行中（非终态）：now - startedAt，随轮询 / 逐秒时间源刷新；
 * - 终态（已完成 / 失败 / 已回滚）：endedAt - startedAt；endedAt 缺失或零值时退化为 now 口径。
 * startedAt 无效时返回 0（调用方按无数据隐藏该行）。
 */
export function resolveBatchReleaseDurationSeconds(options: {
  endedAt?: string;
  isTerminal: boolean;
  now?: number;
  startedAt?: string;
}): number {
  const start = parseServerTime(options.startedAt);
  if (!start) return 0;
  const end = options.isTerminal ? parseServerTime(options.endedAt) : null;
  const endTime = end ?? dayjs(options.now ?? Date.now());
  return Math.max(0, endTime.diff(start, 'second'));
}

/** 操作成功后的提示文案（继续发布需带批次号，回滚需带目标版本）。 */
export function resolveBatchReleaseSuccessMessage(
  action: BatchReleaseAction,
  view: BatchReleaseBannerView | null,
): BatchReleaseMessageDescriptor {
  const batchIndex = view?.batchIndex ?? '--';
  if (action === 'rollback') {
    return { key: '已开始回滚，实例将恢复为 {0}', params: [view?.oldImageTag || '--'] };
  }
  if (action === 'continue') {
    // 失败态复用 continue 动作但入口文案是「重试」，提示需要区分，否则用户会以为在继续后面的批次。
    if (view?.kind === 'failed') {
      return { key: '已重试，开始执行第 {0} 批', params: [batchIndex] };
    }
    return { key: '已继续发布，开始执行第 {0} 批', params: [batchIndex] };
  }
  if (action === 'backward') return { key: '已回退到上一批', params: [] };
  if (action === 'skip') return { key: '已跳过剩余批次，正在全量更新剩余实例', params: [] };
  return { key: '已暂停分批发布', params: [] };
}

/**
 * 实例灰度归属判定：以实例所属工作负载版本（Pod 的 controller-revision-hash）为准，
 * 实例版本等于本次新版本 `workloadUpdateRevision` 即为灰度新起，否则为灰度前。
 *
 * 不再以镜像 Tag 为主口径：Tag 只是镜像地址的一部分，重新构建复用同一 Tag、
 * 或进度快照里的 Tag 与实例实际 Tag 口径不一致时，灰度完成后仍会被误判为「灰度前」。
 * 版本号由后端随进度下发，与横幅「已更新 X/Y」同源，灰度结束后也不会回退。
 *
 * 后端字段未回填时的两级兜底（顺序不可调换）：
 * 1. 进度已是「已完成且全量更新」：全部实例必然是新版本，直接判为灰度新起；
 * 2. 否则退化为镜像 Tag 比对（仅用于灰度进行中且工作负载版本尚未读到）。
 */
export function resolveInstanceCanaryState(
  instance: CanaryStateInstance | null | undefined,
  progress: CanaryStateProgress | null | undefined,
): InstanceCanaryState {
  const instanceRevision = instance?.workloadRevision?.trim() ?? '';
  const updateRevision = progress?.workloadUpdateRevision?.trim() ?? '';
  if (instanceRevision && updateRevision) return instanceRevision === updateRevision ? 'new' : 'before';

  if (isCanaryReleaseFullyUpdated(progress)) return 'new';

  const image = instance?.image ?? '';
  const newImageTag = progress?.newImageTag?.trim() ?? '';
  if (!image || !newImageTag) return 'before';
  return (image.split(':').pop() ?? '') === newImageTag ? 'new' : 'before';
}

/** 标题状态 Tag：定时暂停展示「定时暂停中 · mm:ss 后自动继续」，手动 / 永久暂停展示「等待手动继续」。 */
export function resolvePausedStatusTag(
  progress: BatchReleaseProgressOutputObj,
  remainingSeconds: number,
): BatchReleaseStatusTag {
  if (progress.batchPhase !== 'timed-paused') return { key: '等待手动继续', theme: 'warning' };
  if (remainingSeconds <= 0) return { key: '已暂停', theme: 'warning' };
  return {
    key: '定时暂停中 · {0} 后自动继续',
    params: [formatPauseCountdown(remainingSeconds)],
    theme: 'warning',
  };
}

/**
 * 百分比批步骤条标题的累计实例数：Math.round 口径，至少 1、不超过实例总数。
 *
 * 仅用于进度面板步骤条标题展示（设计稿「25%（3 个）」），数字递增天然表达累计含义；
 * 发布预览沿用 release-preview.resolveBatchUpdatedCount 的 ceil 口径，两者互不影响。
 */
export function resolvePercentAccumulatedCount(pct: number | undefined, replicas: number): number {
  if (!pct || !Number.isFinite(pct) || replicas <= 0) return 0;
  return Math.min(replicas, Math.max(1, Math.round((replicas * pct) / 100)));
}

/**
 * timed-paused 剩余秒数：按 pauseStartTime + pauseSeconds 换算。
 * `now` 由调用方注入：组件传逐秒刷新的时间源即可让倒计时随秒跳动，轮询仅负责校准起点。
 * 非 timed-paused、数据缺失或已到期时返回 0。
 */
export function resolveTimedPauseRemainingSeconds(progress: BatchReleaseProgressOutputObj, now = Date.now()): number {
  if (progress.batchPhase !== 'timed-paused') return 0;
  const currentIndex = progress.currentBatchIndex;
  const batch = progress.batches?.[typeof currentIndex === 'number' && currentIndex >= 0 ? currentIndex : -1];
  const startAt = progress.pauseStartTime ? Date.parse(progress.pauseStartTime) : NaN;
  const seconds = Number(batch?.pauseSeconds) || 0;
  if (Number.isNaN(startAt) || seconds <= 0) return 0;
  return Math.max(0, Math.ceil((startAt + seconds * 1000 - now) / 1000));
}

/**
 * 是否需要补查分批发布进度。
 *
 * 与旧实现只查 deploying 不同：分批发布结束后需要再查一次才能确认结果，
 * 因此这里按「部署记录本身是分批发布」判断；同一 deployID 的终态进度命中缓存后不再重复请求。
 *
 * @param emptyQueriedDeployID 已确认「接口无进度」的 deployID：接口返回空是稳定结果，
 * 不记住就会随 latest-status 每 5s 重复请求一次空进度。
 */
export function shouldFetchBatchReleaseProgress(
  latest: LatestDeployStatus | null,
  cached: BatchReleaseProgressOutputObj | null,
  emptyQueriedDeployID = '',
): boolean {
  if (!hasCanaryDeployRecord(latest)) return false;
  const latestDeployID = latest?.deployID ?? '';
  // deployID 缺失时无法判断是否为同一次发布，退化为「仅终态停止轮询」，避免每 5s 重复请求。
  if (!cached?.deployID || !latestDeployID) {
    // 同一 deployID 已被服务端确认「无进度」时不再重查；但后端标记为进行中的分批发布必须重新补查，
    // 避免「先前空结果」把刚发起的发布挡在门外。
    const confirmedEmpty =
      !!latestDeployID && latestDeployID === emptyQueriedDeployID && latest?.inProgressReleaseMode !== 'canary';
    if (confirmedEmpty) return false;
    return !isTerminalBatchReleaseStatus(cached?.status);
  }
  // 新部署产生新 deployID，必须重新补查。
  if (cached.deployID !== latestDeployID) return true;
  // 同一部署记录被重新发起灰度发布（重新部署复用 deployID，进度已被重置）：
  // 后端标记进行中时终态缓存已过期，必须重查，否则横幅会一直停留在上一次的「已完成 / 已回滚」。
  if (latest?.inProgressReleaseMode === 'canary') return true;
  return !isTerminalBatchReleaseStatus(cached.status);
}

/**
 * 「回退上一批」：所有批次阶段统一展示，第 1 批（或批次未知）置灰，表达「存在但不可用」。
 */
function buildBackwardAction(progress: BatchReleaseProgressOutputObj): BatchReleaseBannerAction {
  const { currentBatchIndex } = progress;
  const hasPrevBatch = typeof currentBatchIndex === 'number' && currentBatchIndex >= 1;
  return createAction('backward', 'default', {
    confirm: BACKWARD_CONFIRM,
    ...(hasPrevBatch ? {} : { disabled: true }),
  });
}

function createAction(
  action: BatchReleaseAction,
  theme: BatchReleaseBannerAction['theme'],
  options: { confirm?: BatchReleaseActionConfirm; disabled?: boolean; labelKey?: string } = {},
): BatchReleaseBannerAction {
  return {
    action,
    theme,
    labelKey: options.labelKey ?? ACTION_LABELS[action],
    ...(options.confirm ? { confirm: options.confirm } : {}),
    ...(options.disabled ? { disabled: true } : {}),
  };
}

/** 分批发布是否已完成全量更新：终态 deployed 且已更新实例数覆盖总实例数。 */
function isCanaryReleaseFullyUpdated(progress: CanaryStateProgress | null | undefined): boolean {
  if (progress?.status !== APP_DEPLOY_STATUS.DEPLOYED) return false;
  const replicas = progress.replicas ?? 0;
  return replicas > 0 && (progress.updatedReplicas ?? 0) >= replicas;
}

/**
 * 解析服务端时间：空值、非法值或 Go 零值时间（0001-01-01T00:00:00Z，表示字段未设置）
 * 统一收敛为 null，调用方按「无数据」兜底。
 */
function parseServerTime(value?: string): Dayjs | null {
  if (!value) return null;
  const time = dayjs(value);
  return time.isValid() && time.year() >= 2000 ? time : null;
}

/** 批次暂停方式的副文本描述。 */
function resolveBatchPauseSubtitle(batch: {
  pauseMode?: string;
  pauseSeconds?: number;
}): BatchReleaseMessageDescriptor {
  if (batch.pauseMode === 'permanent') {
    return { key: '永久暂停 · 手动继续', params: [] };
  }
  if (batch.pauseMode === 'duration') {
    return { key: '暂停 {0} 秒 · 自动继续', params: [batch.pauseSeconds ?? 0] };
  }
  return { key: '不停 · 自动继续', params: [] };
}

/**
 * 暂停态批次的状态 Tag：定时暂停按剩余时间展示「暂停中 mm:ss」（逐秒倒计时）；
 * 到期或换算不出展示「已暂停」；手动 / 永久暂停展示「等待手动继续」。
 */
function resolvePausedStepTag(progress: BatchReleaseProgressOutputObj, now: number): BatchReleaseMessageDescriptor {
  if (progress.batchPhase !== 'timed-paused') return { key: '等待手动继续', params: [] };
  const remaining = resolveTimedPauseRemainingSeconds(progress, now);
  if (remaining <= 0) return { key: '已暂停', params: [] };
  return { key: '暂停中 {0}', params: [formatPauseCountdown(remaining)] };
}
