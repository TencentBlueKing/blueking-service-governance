<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<!-- 灰度发布进度视图：返回栏 + 语义色大横幅（状态 / 元信息 / 平铺操作按钮）+ Steps 批次步骤条 + 灰度实例列表 -->
<template>
  <div
    class="flex flex-col gap-[16px]"
    data-testid="batch-release-progress-panel"
  >
    <!-- 返回栏 -->
    <FlexRow class="h-[48px] py-[8px] px-[16px] shrink-0 bg-[#fff] shadow-[0_2px_4px_0_#0000001a]">
      <template #left>
        <div class="flex items-center gap-[12px]">
          <Button
            class="!px-0"
            data-testid="batch-release-progress-back"
            text
            theme="primary"
            @click="emit('back')"
          >
            <i class="bkms-icon bkms-icon-angle-left text-[14px] mr-[2px] fw-bold"></i>
            {{ $t('返回实例列表') }}
          </Button>
          <Divider
            class="!mx-0 !h-[14px]"
            color="#EAEBF0"
            direction="vertical"
            type="solid"
          />
          <span class="text-[14px] font-bold leading-[22px] text-[#313238]">{{ $t('灰度发布进度') }}</span>
          <span class="text-[12px] leading-[20px] text-[#979BA5]">
            {{ $t('环境：') }}
            <span class="text-[#4D4F56]">{{ envDisplayName || '--' }}</span>
          </span>
          <!-- 环境类型 Tag + 特性环境 Tag，与环境选择器展示口径一致 -->
          <Tag
            v-if="envItem?.type && envTypeMap[envItem.type]"
            :class="envTypeTagClassMap[envItem.type]"
            size="small"
          >
            {{ envTypeMap[envItem.type]?.name || '' }}
          </Tag>
          <Tag
            v-if="isFeatureEnv"
            class="bg-[#E2F5F7] text-[#3A9EAA] shrink-0"
            size="small"
          >
            {{ $t('特性') }}
          </Tag>
        </div>
      </template>
    </FlexRow>

    <!-- 语义色大横幅 -->
    <div
      :class="['rounded-[2px] border px-[16px] py-[16px]', themeClass.container]"
      data-testid="batch-release-progress-banner"
    >
      <div class="flex items-center justify-between gap-[16px]">
        <div class="flex min-w-0 items-center gap-[8px]">
          <span
            class="shrink-0 text-[14px] font-bold leading-[22px] text-[#313238]"
            data-testid="batch-release-progress-status"
          >
            {{ statusText }}
          </span>
          <Tag
            v-if="statusTag"
            class="shrink-0"
            :class="TAG_CLASSES[statusTag.theme]"
            data-testid="batch-release-progress-pause-hint"
            size="small"
          >
            {{ $t(statusTag.key, statusTag.params ?? []) }}
          </Tag>
        </div>
        <!-- 操作按钮全部平铺：主操作 → 回退上一批 / 跳过剩余批次 → 回滚，复用横幅的二次确认链路 -->
        <div
          v-if="flatActions.length"
          class="flex shrink-0 items-center gap-[8px]"
        >
          <Button
            v-for="item in flatActions"
            :key="`${item.action}-${item.labelKey}`"
            :disabled="isBusy || item.disabled"
            :loading="pendingAction === item.action"
            :outline="item.theme === 'danger'"
            size="small"
            :theme="buttonTheme(item)"
            @click="handleAction(item)"
          >
            {{ $t(item.labelKey) }}
          </Button>
        </div>
      </div>
      <!-- 元信息行：仅展示有数据支撑的字段；标签灰色、数值加粗深色，对齐设计稿 -->
      <div
        class="mt-[6px] flex flex-wrap items-center gap-x-[24px] gap-y-[4px] text-[12px] leading-[20px] text-[#4D4F56]"
      >
        <span
          v-if="view.message && view.kind === 'failed'"
          class="flex items-center gap-[4px]"
        >
          <span>{{ $t('失败原因') }}</span>
          <span
            v-bk-tooltips="{ content: view.message, disabled: !view.message }"
            class="max-w-[360px] truncate font-bold text-[#313238]"
          >
            {{ view.message }}
          </span>
        </span>
        <span
          v-if="strategyName"
          class="flex items-center gap-[4px]"
        >
          <span>{{ $t('发布策略') }}</span>
          <span class="text-[#313238]">{{ strategyName }}</span>
        </span>
        <i18n-t
          class="flex items-center gap-[4px]"
          keypath="共 {0} 批"
          tag="span"
        >
          <span class="font-bold text-[#313238]">{{ stepsData.length }}</span>
        </i18n-t>
        <i18n-t
          v-if="view.batchIndex != null"
          class="flex items-center gap-[4px]"
          keypath="当前第 {0} 批"
          tag="span"
        >
          <span class="font-bold text-[#313238]">{{ view.batchIndex }}</span>
        </i18n-t>
        <i18n-t
          class="flex items-center gap-[4px]"
          keypath="已更新 {0}/{1} 个实例"
          tag="span"
        >
          <span class="font-bold text-[#313238]">{{ view.updatedReplicas }}</span>
          <span class="font-bold text-[#313238]">{{ view.replicas }}</span>
        </i18n-t>
        <span class="flex items-center gap-[4px]">
          <span>{{ $t('镜像') }}</span>
          <span class="font-bold text-[#313238]">{{ versionText }}</span>
        </span>
      </div>
      <!-- Steps 批次步骤条 -->
      <div
        v-if="stepsData.length"
        :class="['mt-[12px] border-t pt-[12px]', themeClass.divider]"
      >
        <Steps
          :cur-step="curStep"
          line-type="solid"
          :status="stepsStatus"
          :steps="stepsData"
        />
      </div>
    </div>

    <!-- 灰度实例列表：全部 / 灰度新起 / 灰度前，与横幅「已更新」同口径 -->
    <CanaryInstanceTable
      :env-name="envName"
      :new-image-tag="view.newImageTag"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, onUnmounted, ref, watch } from 'vue';

  import { Button, Divider, Steps, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import {
    type BatchReleaseAction,
    type BatchReleaseBannerAction,
    type BatchReleaseBannerView,
    type BatchReleaseStepVM,
    buildBatchReleaseSteps,
    resolvePausedStatusTag,
    resolveTimedPauseRemainingSeconds,
  } from '../batch-release';
  import CanaryInstanceTable from './canary-instance-table.vue';

  import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';
  import type { EnvOutput } from '~/@types/v1/env';

  const props = defineProps<{
    /** 当前环境对象（展示名 / 类型 Tag / 特性 Tag 数据来源） */
    envItem?: EnvOutput | null;
    /** 当前环境名（灰度实例列表请求作用域） */
    envName: string;
    /** 正在执行的操作，对应按钮展示 loading */
    pendingAction?: BatchReleaseAction | null;
    /** 分批发布进度接口数据 */
    progress: BatchReleaseProgressOutputObj | null;
    /** 由 resolveBatchReleaseBanner 收敛出的展示模型 */
    view: BatchReleaseBannerView;
  }>();

  const emit = defineEmits<{
    action: [action: BatchReleaseBannerAction];
    back: [];
  }>();

  const { t } = useI18n();

  /** 环境展示名；envItem 缺失时回退环境名。 */
  const envDisplayName = computed(() => props.envItem?.displayName || props.envName);

  /** 是否特性环境（返回栏「特性」Tag）。 */
  const isFeatureEnv = computed(() => props.envItem?.kind === 'feature');

  interface PanelTheme {
    container: string;
    divider: string;
    status: string;
  }

  /** 六态语义配色，与单行横幅 THEMES 保持一致；大横幅额外需要分隔线颜色。
   *  回滚中按设计稿为蓝色系（过程态，先恢复实例再清理灰度资源），已回滚 / 暂停为橙色。 */
  const THEMES: Record<BatchReleaseBannerView['kind'], PanelTheme> = {
    completed: {
      container: '!border-[#A1E3BA] !bg-[#EBFAF0]',
      divider: '!border-[#A1E3BA]',
      status: 'text-[#2CAF5E]',
    },
    failed: {
      container: '!border-[#F8B4B4] !bg-[#FFF0F0]',
      divider: '!border-[#F8B4B4]',
      status: 'text-[#EA3636]',
    },
    paused: {
      container: '!border-[#F9D090] !bg-[#FDF4E8]',
      divider: '!border-[#F9D090]',
      status: 'text-[#F59500]',
    },
    rolledBack: {
      container: '!border-[#F9D090] !bg-[#FDF4E8]',
      divider: '!border-[#F9D090]',
      status: 'text-[#F59500]',
    },
    rollingBack: {
      container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
      divider: '!border-[#A3C5FD]',
      status: 'text-[#3A84FF]',
    },
    updating: {
      container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
      divider: '!border-[#A3C5FD]',
      status: 'text-[#3A84FF]',
    },
  };

  /** 状态 Tag 主题 → bkui Tag 配色。 */
  const TAG_CLASSES = {
    danger: '!border-none !bg-[#FFEBEB] !text-[#EA3636]',
    default: '!border-none !bg-[#F0F1F5] !text-[#63656E]',
    info: '!border-none !bg-[#E1ECFF] !text-[#3A84FF]',
    success: '!border-none !bg-[#E5F6EA] !text-[#2CAF5E]',
    warning: '!border-none !bg-[#FDEED8] !text-[#F59500]',
  } as const;

  const themeClass = computed(() => THEMES[props.view.kind]);

  const isBusy = computed(() => !!props.pendingAction);

  /**
   * 逐秒时间源：定时暂停的 mm:ss 倒计时按当前时间实时换算，不等 5s 轮询；
   * 仅在定时暂停态开启 1s 定时器，其余时间静止（取值无意义，相关计算不依赖它）。
   * 只跟随 progress.batchPhase：view 由外层收敛门控，与倒计时是否展示无关。
   */
  const now = ref(Date.now());
  let countdownTimer: ReturnType<typeof setInterval> | undefined;
  watch(
    () => props.progress?.batchPhase === 'timed-paused',
    isTimedPaused => {
      if (isTimedPaused && !countdownTimer) {
        now.value = Date.now();
        countdownTimer = setInterval(() => {
          now.value = Date.now();
        }, 1000);
      } else if (!isTimedPaused && countdownTimer) {
        clearInterval(countdownTimer);
        countdownTimer = undefined;
      }
    },
    { immediate: true },
  );
  onUnmounted(() => clearInterval(countdownTimer));

  /** 标题状态 Tag：定时暂停时按逐秒时间源重算倒计时，其余状态沿用 view 收敛结果。 */
  const statusTag = computed(() => {
    const { statusTag } = props.view;
    if (!statusTag) return null;
    const progress = props.progress;
    if (progress?.batchPhase === 'timed-paused') {
      return resolvePausedStatusTag(progress, resolveTimedPauseRemainingSeconds(progress, now.value));
    }
    return statusTag;
  });

  const strategyName = computed(() => props.progress?.strategyName || '');

  /**
   * 状态标题文案（对齐设计稿：进行中/暂停/失败统一「灰度发布中」，批次信息由元信息行与步骤条承载）。
   */
  const statusText = computed(() => {
    switch (props.view.kind) {
      case 'completed':
        return t('灰度发布已完成');
      case 'rolledBack':
        return t('已回滚');
      case 'rollingBack':
        return t('正在回滚');
      default:
        return t('灰度发布中');
    }
  });

  /** 升级方向：回滚中 / 已回滚为新 → 旧，其余为旧 → 新。 */
  const versionText = computed(() => {
    const { oldImageTag, newImageTag } = props.view;
    const isBackward = props.view.kind === 'rolledBack' || props.view.kind === 'rollingBack';
    return isBackward
      ? `${newImageTag || '--'} → ${oldImageTag || '--'}`
      : `${oldImageTag || '--'} → ${newImageTag || '--'}`;
  });

  /**
   * 标题按状态区分颜色（对齐设计稿，仅数字加粗）：
   * 当前批次（更新中 loading / 暂停中 warning）蓝色；失败红色；其余深灰，未开始浅灰。
   */
  function stepTitleClass(step: BatchReleaseStepVM): string {
    if (step.stepStatus === 'loading' || step.tagTheme === 'warning') return 'text-[#3A84FF]';
    if (step.stepStatus === 'error') return 'text-[#EA3636]';
    if (step.stepStatus === 'done') return 'text-[#313238]';
    return 'text-[#979BA5]';
  }

  /** 占位标记：先把参数替换成标记再翻译，用于把标题切分为「普通文本 + 加粗参数」片段。 */
  const TITLE_PARAM_MARK = '\uE000';

  /** 标题片段：普通文本或加粗参数 VNode。 */
  type TitleSegment = ReturnType<typeof h> | string;

  /** 标题片段：常规批带「第 N 批 ·」前缀，系统补齐全量批不带；仅参数数字加粗。 */
  function stepTitleSegments(step: BatchReleaseStepVM): TitleSegment[] {
    const body = titleSegments(step.title.key, step.title.params);
    return step.isAutoFull ? body : [...titleSegments('第 {0} 批', [step.batchIndex]), ' · ', ...body];
  }

  /** 单参数文案片段：数字/数值加粗，前后文本保持普通字重。 */
  function titleSegments(key: string, params: (number | string)[]): TitleSegment[] {
    if (!params.length) return [t(key)];
    const [before, after] = t(
      key,
      params.map(() => TITLE_PARAM_MARK),
    ).split(TITLE_PARAM_MARK);
    return [before, h('span', { class: 'fw-bold' }, String(params[0])), after];
  }

  /** Steps 批次步骤条数据：title/description 传 VNode 承载「标题 + 状态 Tag」与副文本。 */
  const stepsData = computed(() =>
    buildBatchReleaseSteps(props.progress, now.value).map(step => ({
      ...(step.stepStatus ? { status: step.stepStatus } : {}),
      description: h(
        'div',
        { class: 'text-[12px] leading-[18px] text-[#C4C6CC]' },
        t(step.subtitle.key, step.subtitle.params),
      ),
      title: h('div', { class: 'flex items-center gap-[4px] text-[12px] leading-[20px]' }, [
        h('span', { class: stepTitleClass(step) }, stepTitleSegments(step)),
        h(Tag, { class: TAG_CLASSES[step.tagTheme], size: 'small' }, () => t(step.tagKey, step.tagParams ?? [])),
      ]),
    })),
  );

  /** bkui Steps 的 curStep 从 1 开始；未进入批次落在第 1 批；终态越过最后一个节点表达「全部已完成」。 */
  const curStep = computed(() => {
    const total = stepsData.value.length || 1;
    if (props.view.batchIndex == null) {
      // 终态（已完成 / 回滚中 / 已回滚）：curStep = 总批数 + 1，越过最后一个节点，
      // 配合全部节点 done 避免最后一批被高亮为当前步骤；未进入批次（pending）落在第 1 批。
      return props.view.kind === 'updating' ? 1 : total + 1;
    }
    return Math.min(props.view.batchIndex, total);
  });

  /** Steps 组件级状态：当前批正在更新时传 loading，保证当前节点显示旋转 loading 图标。 */
  const stepsStatus = computed(() => {
    const current = props.view.batchIndex != null ? stepsData.value[props.view.batchIndex - 1] : undefined;
    return current?.status === 'loading' ? 'loading' : '';
  });

  /** 平铺按钮顺序：主操作 → 回退上一批 / 跳过剩余批次 → 回滚。 */
  const flatActions = computed(() => [
    ...props.view.actions.filter(item => item.theme !== 'danger'),
    ...props.view.overflowActions,
    ...props.view.actions.filter(item => item.theme === 'danger'),
  ]);

  /** 把内部主题映射到 bkui Button 支持的 theme 取值。 */
  function buttonTheme(item: BatchReleaseBannerAction): 'danger' | 'primary' | undefined {
    if (item.theme === 'primary') return 'primary';
    if (item.theme === 'danger') return 'danger';
    return undefined;
  }

  function handleAction(item: BatchReleaseBannerAction) {
    if (isBusy.value) return;
    emit('action', item);
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-step) {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .icon-loading::before {
      content: '';
      display: none;
    }

    .bk-step-title::after {
      display: none;
    }

    .bk-step-number {
      position: relative;
    }

    &.done:not(.current) .bk-step-number {
      border-color: #e1ecff;
      background-color: #e1ecff;
    }

    &:not(.current):not(.done) .bk-step-number {
      border-color: #eaebf0;
      background-color: #eaebf0;
      color: #4d4f56;
    }

    &.done .bk-step-number::after {
      background-image: none;
    }

    .bk-step-number::after {
      position: absolute;
      top: 11px;
      left: 30px;
      width: 99999px;
      height: 1px;
      background-color: #dcdee5;
      content: '';
    }

    /* 最后一个步骤右侧不再延伸连接线 */
    &:last-child .bk-step-number::after {
      display: none;
    }
  }
</style>
