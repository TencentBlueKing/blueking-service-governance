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
  <div
    class="flex min-w-0 flex-1 flex-col gap-[16px] rounded-[2px] border border-solid border-[#dcdee5] bg-white p-[16px]"
    data-testid="batch-detail"
  >
    <div class="flex flex-wrap items-baseline gap-x-[12px] border-b border-b-solid border-b-[#f0f1f5] pb-[12px]">
      <span class="text-[14px] font-700 leading-[22px] text-[#313238]">
        {{ formatBatchCardTitle(batch, index, t) }}
      </span>
      <span class="text-[12px] leading-[20px] text-[#979ba5]">
        {{ t('修改后点「预览策略」可查看整体效果') }}
      </span>
    </div>

    <div class="flex flex-col gap-[12px]">
      <div class="text-[12px] font-700 leading-[20px] text-[#313238]">{{ t('目标资源') }}</div>
      <!-- 错误文案紧跟输入行（gap 4px），说明文案在其下方 -->
      <div class="flex flex-col gap-[4px]">
        <div class="flex items-stretch">
          <span
            class="flex h-[32px] shrink-0 items-center whitespace-nowrap rounded-[2px_0_0_2px] border border-solid border-[#c4c6cc] border-r-0 bg-[#fafbfd] px-[12px] text-[12px] text-[#63656e]"
          >
            {{ t('累计更新副本数') }}
          </span>
          <Input
            class="target-value w-[120px] ml-[-1px] relative hover:z-10"
            :class="{ 'border-[#EA3636] z-10': Boolean(error) }"
            data-testid="target-value"
            :disabled="readonly"
            :max="targetValueMax"
            :min="1"
            :model-value="batch.targetValue"
            type="number"
            @update:model-value="handleTargetChange"
          />
          <Select
            class="target-unit w-[96px] ml-[-1px] relative hover:z-10"
            :clearable="false"
            :disabled="readonly"
            :list="unitList"
            :model-value="batch.targetUnit"
            @update:model-value="handleUnitChange"
          />
        </div>
        <div
          v-if="error"
          class="text-[12px] leading-[20px] text-[#ea3636]"
          data-testid="target-error"
        >
          {{ error }}
        </div>
      </div>
      <div class="text-[12px] leading-[20px] text-[#979ba5]">
        <i18n-t keypath="填到本批结束时应更新的{total}（按累计值推进），不是本批新增数。新实例上线时北极星权重为 1。">
          <template #total>
            <span class="font-bold">{{ t('总量') }}</span>
          </template>
        </i18n-t>
      </div>
    </div>

    <div class="flex flex-col gap-[8px] border-t border-t-solid border-t-[#f0f1f5] pt-[16px]">
      <div class="text-[12px] font-700 leading-[20px] text-[#313238]">{{ t('暂停') }}</div>
      <Radio.Group
        class="flex flex-wrap"
        :disabled="readonly || isFullPercent"
        :model-value="batch.pauseMode"
        @update:model-value="handlePauseModeChange"
      >
        <Radio
          :disabled="readonly || isFullPercent"
          label="none"
        >
          {{ t('不停') }}
        </Radio>
        <Radio
          :disabled="readonly || isFullPercent"
          label="duration"
        >
          {{ t('暂停指定时长') }}
        </Radio>
        <Radio
          :disabled="readonly || isFullPercent"
          label="permanent"
        >
          {{ t('永久暂停，手动继续') }}
        </Radio>
      </Radio.Group>
      <div
        v-if="isFullPercent"
        class="text-[12px] leading-[20px] text-[#979ba5]"
        data-testid="full-percent-pause-tip"
      >
        {{ t('达到 100% 后全部实例已更新完毕，发布将直接结束，无需暂停。') }}
      </div>
      <div
        v-if="batch.pauseMode === 'duration'"
        class="mt-[4px]"
      >
        <Input
          class="w-[160px]"
          :disabled="readonly"
          :max="PAUSE_SECONDS_MAX"
          :min="PAUSE_SECONDS_MIN"
          :model-value="batch.pauseSeconds"
          :placeholder="t('30 ~ 86400 秒')"
          :suffix="t('秒')"
          type="number"
          @update:model-value="handlePauseSecondsChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Input, Radio, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import {
    DEFAULT_PAUSE_SECONDS,
    formatBatchCardTitle,
    PAUSE_SECONDS_MAX,
    PAUSE_SECONDS_MIN,
    PERCENT_TARGET_MAX,
  } from '../utils';

  import type { PauseMode, StrategyBatchForm, TargetUnit } from '../utils';

  interface Props {
    /** 行级校验错误文案 */
    error?: string;
    /** 批次的展示序号（从 0 开始） */
    index: number;
    /** 只读模式禁用所有输入 */
    readonly?: boolean;
  }

  withDefaults(defineProps<Props>(), {
    error: '',
    readonly: false,
  });

  const batch = defineModel<StrategyBatchForm>({ required: true });

  const { t } = useI18n();

  const unitList: { label: string; value: TargetUnit }[] = [
    { label: t('个'), value: 'count' },
    { label: t('%'), value: 'percent' },
  ];

  // 百分比单位时目标值上限为 100
  const targetValueMax = computed(() => (batch.value.targetUnit === 'percent' ? PERCENT_TARGET_MAX : undefined));

  /** 目标为 100% 的批次：发布到此批即全部完成，暂停选项锁定为「不停」 */
  const isFullPercent = computed(
    () => batch.value.targetUnit === 'percent' && batch.value.targetValue === PERCENT_TARGET_MAX,
  );

  function handlePauseModeChange(value: PauseMode) {
    batch.value.pauseMode = value;
    // 切到定时暂停时给出合法默认值，避免用户一进入即看到校验错误
    if (value === 'duration' && !batch.value.pauseSeconds) {
      batch.value.pauseSeconds = DEFAULT_PAUSE_SECONDS;
    }
  }

  function handlePauseSecondsChange(value: number | string) {
    batch.value.pauseSeconds = toNumberOrUndefined(value);
  }

  function handleTargetChange(value: number | string) {
    batch.value.targetValue = toNumberOrUndefined(value);
    syncPauseModeForFullPercent();
  }

  function handleUnitChange(value: TargetUnit) {
    batch.value.targetUnit = value;
    // 切到百分比时，若目标值超出上限则收敛到 100，避免带着非法值提交
    if (value === 'percent' && (batch.value.targetValue ?? 0) > PERCENT_TARGET_MAX) {
      batch.value.targetValue = PERCENT_TARGET_MAX;
    }
    syncPauseModeForFullPercent();
  }

  /** 目标达到 100% 时发布直接结束，暂停强制锁定为「不停」；改回 < 100% 时恢复可选 */
  function syncPauseModeForFullPercent() {
    if (isFullPercent.value) {
      batch.value.pauseMode = 'none';
    }
  }

  function toNumberOrUndefined(value: number | string | undefined): number | undefined {
    if (value === '' || value === undefined || value === null) return undefined;
    const num = Number(value);
    return Number.isNaN(num) ? undefined : num;
  }
</script>

<style lang="postcss" scoped>
  /* 三段拼接的输入组：靠深层选择器抹平组件库圆角，原子类无法穿透子组件根节点 */
  .target-value {
    width: 120px;
  }

  .target-value :deep(.bk-input) {
    border-radius: 0;
  }

  .target-unit {
    width: 96px;
  }

  .target-unit :deep(.bk-select) {
    border-radius: 0 2px 2px 0;
  }
</style>
