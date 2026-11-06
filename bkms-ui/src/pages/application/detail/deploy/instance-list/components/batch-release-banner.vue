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
      <i
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
          <!-- 已暂停（人工暂停 / 策略永久暂停 / 定时暂停） -->
          <template v-else-if="view.kind === 'paused'">
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
            {{ $t('查看发布进度') }}
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
  import { computed } from 'vue';

  import { Alert, Button, Dropdown } from 'bkui-vue';
  import { AngleDownLine } from 'bkui-vue/lib/icon';

  import type { BatchReleaseAction, BatchReleaseBannerAction, BatchReleaseBannerView } from '../batch-release';

  const props = defineProps<{
    /** 正在执行的操作，对应按钮展示 loading */
    pendingAction?: BatchReleaseAction | null;
    /** 由 resolveBatchReleaseBanner 收敛出的展示模型 */
    view: BatchReleaseBannerView;
  }>();
  const emit = defineEmits<{
    action: [action: BatchReleaseBannerAction];
    close: [];
    /** 点击「查看发布进度」：由父组件切换到独立的灰度发布进度视图 */
    viewProgress: [];
  }>();

  interface BannerTheme {
    /** 覆盖 Alert 各主题默认底色与描边，对齐设计稿 */
    container: string;
    /** 状态词与图标共用的语义色 */
    status: string;
  }

  /** 五套语义配色：绿（已完成）/ 红（失败）/ 黄（暂停、已回滚）/ 蓝（更新中、回滚中——回滚中按设计稿为过程态蓝色）。 */
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
      container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
      status: 'text-[#3A84FF]',
    },
    updating: {
      container: '!border-[#A3C5FD] !bg-[#F0F5FF]',
      status: 'text-[#3A84FF]',
    },
  };

  /** Alert 内置主题（决定默认无障碍语义，图标已由 icon 插槽覆盖）。 */
  const ALERT_THEMES: Record<BatchReleaseBannerView['kind'], 'danger' | 'info' | 'success' | 'warning'> = {
    completed: 'success',
    failed: 'danger',
    paused: 'warning',
    rolledBack: 'warning',
    rollingBack: 'info',
    updating: 'info',
  };

  /** 每种状态的左侧图标。 */
  const ICONS: Record<BatchReleaseBannerView['kind'], string> = {
    completed: 'bkms-icon-circle-info',
    failed: 'bkms-icon-warning-circle',
    paused: 'bkms-icon-warning-circle',
    rolledBack: 'bkms-icon-warning-circle',
    rollingBack: 'bkms-icon-warning-circle',
    updating: 'bkms-icon-circle-info',
  };

  const themeClass = computed(() => THEMES[props.view.kind]);

  const alertTheme = computed(() => ALERT_THEMES[props.view.kind]);

  const iconClass = computed(() => ICONS[props.view.kind]);

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
  function buttonTheme(item: BatchReleaseBannerAction): 'danger' | 'primary' | undefined {
    if (item.theme === 'primary') return 'primary';
    if (item.theme === 'danger') return 'danger';
    return undefined;
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
</style>
