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

<template>
  <!-- 基于 bkui Alert：icon 插槽承载语义图标，title 插槽承载「状态文案 + 操作按钮」单行内容 -->
  <Alert
    class="batch-release-banner"
    :class="themeClass.container"
    :closable="view.closable"
    :data-kind="view.kind"
    data-testid="batch-release-banner"
    :show-icon="true"
    :theme="alertTheme"
    @close="emit('close')"
  >
    <template #icon>
      <!-- loading 为 SVG 动画图标（字体图标无法显示动画），需经 SvgIcon 渲染；其余沿用字体图标 -->
      <SvgIcon
        v-if="view.iconKind === 'loading'"
        :class="[themeClass.status, 'mr-[8px]']"
        height="16px"
        icon="bkms-icon-loading"
        width="16px"
      />
      <i
        v-else
        :class="['bkms-icon mr-[8px] text-[16px] leading-[1]', themeClass.status, iconClass]"
        data-testid="batch-release-banner-icon"
      ></i>
    </template>
    <template #title>
      <div class="flex items-center gap-[8px] text-[12px] leading-[20px]">
        <div class="min-w-0 flex-1 truncate text-[#4D4F56]">
          <span>{{ $t('当前环境灰度发布：') }}</span>
          <!-- 本批更新中 / 尚未开始 -->
          <template v-if="view.kind === 'updating'">
            <span :class="['font-bold', themeClass.status]">{{ $t('灰度发布中') }}</span>
            <i18n-t keypath=" · 第 {index} 批正在更新实例（{version}）">
              <template #index>
                <span class="font-bold">{{ view.batchIndex ?? '--' }}</span>
              </template>
              <template #version>
                <span class="font-bold">{{ forwardVersion }}</span>
              </template>
            </i18n-t>
          </template>
          <!-- 暂停态：等待手动继续 / 定时暂停为蓝色过程态（批次已完成）；仅「已暂停」兜底为黄色警示态 -->
          <template v-else-if="view.kind === 'paused'">
            <template v-if="pausedWaiting">
              <!-- 设计稿：定时暂停的状态词为橙色 #F59500，等待手动继续沿用蓝色；容器与图标仍为蓝色过程态 -->
              <span :class="['font-bold', pausedCountdown ? 'text-[#F59500]' : themeClass.status]">
                {{ $t('第 {0} 批已完成', [view.batchIndex ?? '--']) }}
              </span>
              <i18n-t
                v-if="pausedCountdown"
                keypath=" · 定时暂停中，{time} 后自动继续（{version}）"
              >
                <template #time>
                  <span class="font-bold">{{ pausedCountdown }}</span>
                </template>
                <template #version>
                  <span class="font-bold">{{ forwardVersion }}</span>
                </template>
              </i18n-t>
              <i18n-t
                v-else
                keypath=" · 等待手动继续（{version}）"
              >
                <template #version>
                  <span class="font-bold">{{ forwardVersion }}</span>
                </template>
              </i18n-t>
            </template>
            <template v-else>
              <span :class="['font-bold', themeClass.status]">
                {{ $t('第 {0} 批已暂停', [view.batchIndex ?? '--']) }}
              </span>
              <i18n-t keypath=" · 已更新 {replicas} 个实例（{version}）">
                <template #replicas>
                  <span class="font-bold">{{ replicaText }}</span>
                </template>
                <template #version>
                  <span class="font-bold">{{ forwardVersion }}</span>
                </template>
              </i18n-t>
            </template>
          </template>
          <!-- 第 N 批失败 -->
          <template v-else-if="view.kind === 'failed'">
            <span :class="['font-bold', themeClass.status]">
              {{ $t('第 {0} 批失败', [view.batchIndex ?? '--']) }}
            </span>
            <span> · </span>
            <span
              v-bk-tooltips="{ content: view.message, disabled: !view.message }"
              class="font-bold"
            >
              {{ view.message || $t('当前批次实例更新失败') }}
            </span>
            <i18n-t keypath="，已暂停当前批次（{version}）">
              <template #version>
                <span class="font-bold">{{ forwardVersion }}</span>
              </template>
            </i18n-t>
          </template>
          <!-- 回滚中 -->
          <template v-else-if="view.kind === 'rollingBack'">
            <span :class="['font-bold', themeClass.status]">{{ $t('正在回滚') }}</span>
            <i18n-t keypath=" · 已恢复 {replicas} 个实例，正在清理灰度资源（{version}）">
              <template #replicas>
                <span class="font-bold">{{ replicaText }}</span>
              </template>
              <template #version>
                <span class="font-bold">{{ backwardVersion }}</span>
              </template>
            </i18n-t>
          </template>
          <!-- 已回滚（终态，非错误态） -->
          <template v-else-if="view.kind === 'rolledBack'">
            <span :class="['font-bold', themeClass.status]">{{ $t('已回滚') }}</span>
            <i18n-t keypath=" · 已恢复 {replicas} 个实例（{version}）">
              <template #replicas>
                <span class="font-bold">{{ replicaText }}</span>
              </template>
              <template #version>
                <span class="font-bold">{{ backwardVersion }}</span>
              </template>
            </i18n-t>
          </template>
          <!-- 已完成 -->
          <template v-else>
            <span :class="['font-bold', themeClass.status]">{{ $t('灰度发布已完成') }}</span>
            <i18n-t keypath=" · 已更新 {replicas} 个实例（{version}）">
              <template #replicas>
                <span class="font-bold">{{ replicaText }}</span>
              </template>
              <template #version>
                <span class="font-bold">{{ forwardVersion }}</span>
              </template>
            </i18n-t>
          </template>
          <Button
            class="ml-[8px]"
            text
            theme="primary"
            @click="emit('viewProgress')"
          >
            {{ $t('查看发布详情') }}
          </Button>
        </div>
        <!-- 顺序：主操作 → 更多 → 危险操作（回滚固定在最后） -->
        <div
          v-if="inlineActions.length || view.overflowActions.length || dangerActions.length"
          class="flex shrink-0 items-center gap-[8px]"
        >
          <Button
            v-for="item in inlineActions"
            :key="item.action"
            :disabled="isBusy"
            :loading="pendingAction === item.action"
            size="small"
            :theme="buttonTheme(item)"
            @click="handleAction(item)"
          >
            {{ $t(item.labelKey) }}
          </Button>
          <Dropdown
            v-if="view.overflowActions.length"
            placement="bottom-end"
            :popover-options="{ boundary: 'body', clickContentAutoHide: true }"
            trigger="click"
          >
            <Button
              :disabled="isBusy"
              size="small"
              theme="default"
            >
              {{ $t('更多') }}
              <AngleDownLine class="ml-[4px] text-[12px]" />
            </Button>
            <template #content>
              <Dropdown.DropdownMenu class="!w-[116px] !p-0">
                <Dropdown.DropdownItem
                  v-for="item in view.overflowActions"
                  :key="`${item.action}-${item.labelKey}`"
                  class="!h-[32px] !px-[16px] !text-[12px] !leading-[32px]"
                  :class="
                    isActionDisabled(item)
                      ? '!pointer-events-none !text-[#C4C6CC]'
                      : '!text-[#4D4F56] hover:!bg-[#F0F1F5]'
                  "
                  @click="handleAction(item)"
                >
                  {{ $t(item.labelKey) }}
                </Dropdown.DropdownItem>
              </Dropdown.DropdownMenu>
            </template>
          </Dropdown>
          <Button
            v-for="item in dangerActions"
            :key="item.action"
            :disabled="isBusy"
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
    </template>
  </Alert>
</template>

<script lang="ts" setup>
  import { computed, onUnmounted, ref, watch } from 'vue';

  import { Alert, Button, Dropdown } from 'bkui-vue';
  import { AngleDownLine } from 'bkui-vue/lib/icon';

  import { formatPauseCountdown, resolveTimedPauseRemainingSeconds } from '../batch-release';

  import type { BatchReleaseProgressOutputObj } from '~/@types/v1/deploy';
  import type {
    BatchReleaseAction,
    BatchReleaseBannerAction,
    BatchReleaseBannerView,
    BatchReleaseIconKind,
  } from '../batch-release';

  const props = defineProps<{
    /** 原始进度快照：定时暂停倒计时按逐秒时间源实时换算（statusTag.params 仅随轮询刷新） */
    progress?: BatchReleaseProgressOutputObj | null;
    /** 正在执行的操作，对应按钮展示 loading */
    pendingAction?: BatchReleaseAction | null;
    /** 由 resolveBatchReleaseBanner 收敛出的展示模型 */
    view: BatchReleaseBannerView;
  }>();
  const emit = defineEmits<{
    action: [action: BatchReleaseBannerAction];
    close: [];
    /** 点击「查看发布详情」：由父组件切换到独立的灰度发布进度视图 */
    viewProgress: [];
  }>();

  interface BannerTheme {
    /** 覆盖 Alert 各主题默认底色与描边，对齐设计稿 */
    container: string;
    /** 状态词与图标共用的语义色 */
    status: string;
  }

  /**
   * 五套语义配色（设计稿）：
   * 绿（已完成）/ 红（失败）/ 黄（已暂停兜底、已回滚、回滚中）/ 蓝（更新中）。
   * 等待手动继续 / 定时暂停为蓝色过程态，见 PAUSED_WAITING_THEME。
   */
  const THEMES: Record<BatchReleaseBannerView['kind'], BannerTheme> = {
    completed: {
      container: '!border-[#A1E3BA] !bg-[#EBFAF0]',
      status: 'text-[#2CAF5E]',
    },
    failed: {
      container: '!border-[#F8B4B4] !bg-[#FFF0F0]',
      status: 'text-[#EA3636]',
    },
    paused: {
      container: '!border-[#F9D090] !bg-[#FDF4E8]',
      status: 'text-[#F59500]',
    },
    rolledBack: {
      container: '!border-[#F9D090] !bg-[#FDF4E8]',
      status: 'text-[#F59500]',
    },
    rollingBack: {
      container: '!border-[#F9D090] !bg-[#FDF4E8]',
      status: 'text-[#F59500]',
    },
    updating: {
      container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
      status: 'text-[#3A84FF]',
    },
  };

  /** 等待手动继续 / 定时暂停（批次已完成的暂停）：设计稿蓝色过程态，与「已暂停」黄色警示态区分。 */
  const PAUSED_WAITING_THEME: BannerTheme = {
    container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
    status: 'text-[#3A84FF]',
  };

  /** Alert 内置主题（决定默认无障碍语义，图标已由 icon 插槽覆盖）。 */
  const ALERT_THEMES: Record<BatchReleaseBannerView['kind'], 'danger' | 'info' | 'success' | 'warning'> = {
    completed: 'success',
    failed: 'danger',
    paused: 'warning',
    rolledBack: 'warning',
    rollingBack: 'warning',
    updating: 'info',
  };

  /**
   * 暂停态细分：statusTag 为「等待手动继续 / 定时暂停中」时按设计稿展示蓝色过程态
   * （批次已完成，等待继续）；「已暂停」（定时到期兜底等）保持黄色警示态。
   */
  const pausedWaiting = computed(() => {
    if (props.view.kind !== 'paused') return false;
    const key = props.view.statusTag?.key ?? '';
    return key === '等待手动继续' || key.startsWith('定时暂停中');
  });

  /**
   * 逐秒时间源：定时暂停的 mm:ss 倒计时按当前时间实时换算，不等 5s 轮询；
   * 仅在定时暂停态开启 1s 定时器，其余时间静止（取值无意义，相关计算不依赖它）。
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

  /**
   * 定时暂停倒计时（mm:ss）：按 pauseStartTime + pauseSeconds 逐秒换算，随秒跳动；
   * 非 timed-paused 或已到期时为空（到期后轮询会把 statusTag 校正为「已暂停」）。
   */
  const pausedCountdown = computed(() => {
    if (props.progress?.batchPhase !== 'timed-paused') return '';
    const remaining = resolveTimedPauseRemainingSeconds(props.progress, now.value);
    return remaining > 0 ? formatPauseCountdown(remaining) : '';
  });

  /**
   * 左侧字体图标按设计稿语义映射（view.iconKind 由纯函数层按状态收敛）：
   * time=定时暂停、pause=手动/永久暂停、check=发布成功、warning=失败、info=已回滚。
   * loading（更新中 / 回滚中）为 SVG 动画图标，经模板中的 SvgIcon 渲染，不在此列。
   */
  const ICONS: Record<Exclude<BatchReleaseIconKind, 'loading'>, string> = {
    check: 'bkms-icon-success-2',
    info: 'bkms-icon-circle-info',
    pause: 'bkms-icon-pause',
    time: 'bkms-icon-time-2',
    warning: 'bkms-icon-warning-circle',
  };

  const themeClass = computed(() => (pausedWaiting.value ? PAUSED_WAITING_THEME : THEMES[props.view.kind]));

  const alertTheme = computed(() => (pausedWaiting.value ? 'info' : ALERT_THEMES[props.view.kind]));

  /** 字体图标类名；loading 态由 SvgIcon 渲染，此处返回空串。 */
  const iconClass = computed(() => (props.view.iconKind === 'loading' ? '' : ICONS[props.view.iconKind]));

  /** 主操作与次要操作（回滚前面的按钮） */
  const inlineActions = computed(() => props.view.actions.filter(item => item.theme !== 'danger'));

  /** 危险操作（回滚），固定排在「更多」之后 */
  const dangerActions = computed(() => props.view.actions.filter(item => item.theme === 'danger'));

  /** 有操作在途：所有入口统一置灰，避免「按钮看着能点、点了没反应」。 */
  const isBusy = computed(() => !!props.pendingAction);

  /** 已更新实例数 / 总实例数 */
  const replicaText = computed(() => `${props.view.updatedReplicas} / ${props.view.replicas}`);

  /** 升级方向：旧版本 → 新版本 */
  const forwardVersion = computed(() => `${props.view.oldImageTag || '--'} → ${props.view.newImageTag || '--'}`);

  /** 回滚方向：新版本 → 旧版本 */
  const backwardVersion = computed(() => `${props.view.newImageTag || '--'} → ${props.view.oldImageTag || '--'}`);

  /** 把内部主题映射到 bkui Button 支持的 theme 取值 */
  function buttonTheme(item: BatchReleaseBannerAction): 'danger' | 'default' | 'primary' {
    if (item.theme === 'primary') return 'primary';
    if (item.theme === 'danger') return 'danger';
    return 'default';
  }

  function handleAction(item: BatchReleaseBannerAction) {
    if (isActionDisabled(item)) return;
    emit('action', item);
  }

  /** 「更多」菜单项：当前状态不支持 / 有操作在途时都按禁用处理（DropdownItem 无 disabled 属性，只能靠 class + 拦截）。 */
  function isActionDisabled(item: BatchReleaseBannerAction) {
    return !!item.disabled || isBusy.value;
  }
</script>

<style lang="postcss" scoped>
  /* Alert 默认是「标题 + 描述」两行结构，这里压成设计稿要求的 32px 单行信息条 */
  .batch-release-banner :deep(.bk-alert-wraper) {
    align-items: center;
    padding: 5px 12px;
  }

  .batch-release-banner :deep(.bk-alert-content) {
    overflow: visible;
  }

  .batch-release-banner :deep(.bk-alert-title) {
    line-height: 20px;
  }

  .batch-release-banner :deep(.bk-button-default) {
    background-color: #fff;
  }
</style>
