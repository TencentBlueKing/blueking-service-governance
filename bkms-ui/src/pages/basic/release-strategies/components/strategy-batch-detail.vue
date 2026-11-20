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
        <i18n-t keypath="填到本批结束时应更新的{total}（按累计值推进），不是本批新增数。">
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
        <Form
          ref="pauseFormRef"
          class="pause-form"
          form-type="vertical"
          :model="pauseFormModel"
          :rules="pauseRules"
        >
          <Form.FormItem property="pauseSeconds">
            <!-- 不传 min / max：组件库会在 change 时把越界值静默收敛到边界，用户看不到原因 -->
            <Input
              class="pause-seconds w-[160px]"
              data-testid="pause-seconds"
              :disabled="readonly"
              :model-value="batch.pauseSeconds"
              :suffix="t('秒')"
              type="number"
              @blur="handlePauseBlur"
              @update:model-value="handlePauseSecondsChange"
            />
            <!-- 与错误文案同一位置：校验通过时展示取值说明，失败时由 error 插槽替换 -->
            <div
              v-if="!pauseInvalid"
              class="pt-[2px] text-[12px] leading-[20px] text-[#979ba5]"
              data-testid="pause-hint"
            >
              {{ translate(t, '可填 {0} ~ {1} 秒的整数（最长 24 小时）。', [PAUSE_SECONDS_MIN, PAUSE_SECONDS_MAX]) }}
            </div>
            <template #error="message">
              <span data-testid="pause-error">{{ message }}</span>
            </template>
          </Form.FormItem>
        </Form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Form, Input, Radio, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import {
    DEFAULT_PAUSE_SECONDS,
    formatBatchCardTitle,
    PAUSE_SECONDS_MAX,
    PAUSE_SECONDS_MIN,
    PERCENT_TARGET_MAX,
    translate,
    validatePauseSeconds,
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

  /** 暂停时长的 Form 实例：失焦 / 提交时由它做校验并内联展示错误 */
  const pauseFormRef = ref<InstanceType<typeof Form> | null>(null);

  /**
   * 暂停时长的表单模型：只承载这一项，交给 Form 统一校验（不传 min/max，避免值被静默收敛）。
   * 必须是稳定引用并被就地更新——Form 校验从 model 取值，若每次渲染换新对象，
   * 校验触发时读到的还是上一次渲染的旧值，会出现「已经改成 60 却报 5 的错误」。
   */
  const pauseFormModel = reactive<{ pauseSeconds: number | undefined }>({
    pauseSeconds: batch.value.pauseSeconds,
  });

  /** 是否已进入校验态（失焦过，或提交/预览时由抽屉触发过）：之后值变化实时重算 */
  const pauseValidated = ref(false);

  /**
   * 暂停项当前是否校验不通过：直接取 Form 校验的结果，保证与 FormItem 的错误渲染始终一致。
   * 不能用「值 + 已校验」自己推算——校验是异步的，自己推算会先隐藏说明文案、错误却还没渲染出来。
   */
  const pauseInvalid = ref(false);

  const pauseRules = computed(() => ({
    pauseSeconds: [
      {
        trigger: 'blur',
        validator: (value: number | undefined) => {
          const error = validatePauseSeconds(value);
          return error ? translate(t, error.key, error.params) : true;
        },
      },
    ],
  }));

  // 切换批次或暂停方式：回到「先不报错」的初始态，并清掉上一个批次的错误文案
  watch(
    () => [batch.value.key, batch.value.pauseMode],
    () => {
      pauseValidated.value = false;
      pauseInvalid.value = false;
      pauseFormModel.pauseSeconds = batch.value.pauseSeconds;
      pauseFormRef.value?.clearValidate();
    },
  );

  // 值改动：先把新值同步给 Form 模型，再按需重算；改回合法值即刻恢复，不必等再次失焦
  watch(
    () => batch.value.pauseSeconds,
    value => {
      pauseFormModel.pauseSeconds = value;
      if (pauseValidated.value) void validatePause();
    },
  );

  function handlePauseBlur() {
    void validatePause();
  }

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

  /**
   * 主动校验暂停时长（失焦 / 提交 / 预览时调用）。
   * Form.validate() 通过时 resolve、失败时 reject：用 try/catch 承接并驱动 pauseInvalid，
   * 让「取值说明」与「错误文案」严格互斥；异常已消化，调用方不必再处理 rejection。
   */
  async function validatePause() {
    pauseValidated.value = true;
    try {
      await pauseFormRef.value?.validate('pauseSeconds');
      pauseInvalid.value = false;
    } catch {
      pauseInvalid.value = true;
    }
  }

  defineExpose({ validatePause });
</script>

<style lang="postcss" scoped>
  /* 暂停时长：FormItem 只承担校验，标签与间距都不需要 */
  .pause-form :deep(.bk-form-item) {
    margin-bottom: 0;
  }

  /* 错误文案改为文档流内展示，默认绝对定位会浮在下一项上方 */
  .pause-form :deep(.bk-form-error) {
    position: static;
    padding-top: 2px;
    font-size: 12px;
    line-height: 20px;
  }

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
