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
  <Select
    ref="selectRef"
    v-model="model"
    filterable
    :with-validate="false"
    @toggle="onToggle"
  >
    <Select.Option
      v-for="option in options"
      :key="option"
      :label="kind === 'cpu' ? $t('{0} 核', [formatCpuCores(option)]) : option"
      :value="option"
    />
    <template #extension>
      <button
        v-if="!isDraftOpen"
        class="flex h-full w-full cursor-pointer items-center justify-center gap-[6px] px-[12px] text-[12px] text-[#4D4F56] hover:bg-[#F0F1F5]"
        type="button"
        @click="openDraft"
      >
        <i
          aria-hidden="true"
          class="bkms-icon bkms-icon-plus-circle text-[14px] text-[#979BA5]"
        ></i>
        {{ $t('自定义') }}
      </button>
      <div
        v-else
        class="flex w-full items-center gap-[4px] px-[12px]"
        @keydown.esc.prevent="onDraftEscape"
        @keydown.stop
      >
        <div
          class="flex h-[32px] min-w-0 flex-1 items-center rounded-[2px] border bg-white text-[12px] text-[#63656E]"
          :class="showDraftError ? 'border-[#EA3636]' : 'border-[#C4C6CC] focus-within:border-[#3A84FF]'"
        >
          <input
            ref="draftInputRef"
            :aria-invalid="showDraftError"
            :aria-label="$t('自定义')"
            class="draft-number-input h-full min-w-0 flex-1 bg-transparent px-[10px] outline-none placeholder:text-[#C4C6CC]"
            inputmode="decimal"
            min="0"
            :placeholder="$t('请输入')"
            step="any"
            type="number"
            :value="draft"
            @beforeinput="onBeforeDraftInput"
            @focus="unitSelectRef?.hidePopover()"
            @input="onDraftInput"
            @keydown.down.prevent="stepDraft(-1)"
            @keydown.enter="onDraftEnter"
            @keydown.up.prevent="stepDraft(1)"
          />
          <span
            v-if="showDraftError"
            v-bk-tooltips="draftError"
            :aria-label="draftError"
            class="mr-[8px] flex shrink-0 items-center text-[16px] text-[#EA3636]"
          >
            <ExclamationCircleShape />
          </span>
          <Select
            v-if="kind === 'memory'"
            ref="unitSelectRef"
            v-model="unit"
            class="shrink-0 border-l border-[#DCDEE5]"
            :clearable="false"
            :popover-min-width="64"
            :popover-options="{ boundary: 'parent' }"
            :with-validate="false"
            @toggle="isUnitSelectOpen = $event"
          >
            <template #trigger>
              <button
                :aria-expanded="isUnitSelectOpen"
                :aria-label="`${$t('内存单位')} ${unit}`"
                class="flex h-[30px] w-[47px] cursor-pointer items-center justify-between pl-[8px] pr-[4px] outline-none focus-visible:bg-[#E1ECFF]"
                type="button"
                @keydown.down.prevent="unit = 'Mi'"
                @keydown.tab="unitSelectRef?.hidePopover()"
                @keydown.up.prevent="unit = 'Gi'"
              >
                {{ unit }}
                <AngleDown
                  aria-hidden="true"
                  class="text-[#979BA5]"
                  :class="{ 'rotate-180': isUnitSelectOpen }"
                  :height="18"
                  :width="18"
                />
              </button>
            </template>
            <Select.Option
              v-for="memoryUnit in MEMORY_UNITS"
              :key="memoryUnit"
              :label="memoryUnit"
              :value="memoryUnit"
            />
          </Select>
        </div>
        <button
          :aria-label="$t('确定')"
          class="flex h-[24px] w-[24px] shrink-0 cursor-pointer items-center justify-center text-[#2CAF5E] hover:text-[#299E56]"
          type="button"
          @click="confirmDraft"
        >
          <Done
            fill="currentColor"
            :height="24"
            :width="24"
          />
        </button>
        <button
          :aria-label="$t('取消')"
          class="flex h-[24px] w-[24px] shrink-0 cursor-pointer items-center justify-center text-[#979BA5] hover:text-[#63656E]"
          type="button"
          @click="closeDraft"
        >
          <Error
            fill="currentColor"
            :height="18"
            :width="18"
          />
        </button>
      </div>
    </template>
  </Select>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref } from 'vue';

  import { Select } from 'bkui-vue';
  import { AngleDown, Done, Error, ExclamationCircleShape } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { formatCpuCores, parseCpuCores, parseMemoryToMiB } from '~/common/resource-quantity';

  type MemoryUnit = 'Gi' | 'Mi';

  const CPU_OPTIONS = ['0.1', '0.2', '0.5', '1', '2', '4', '8', '16', '32'];
  const MEMORY_OPTIONS = ['256Mi', '512Mi', '1Gi', '2Gi', '4Gi', '8Gi', '16Gi', '32Gi', '64Gi'];
  const MEMORY_UNITS: MemoryUnit[] = ['Gi', 'Mi'];
  const DRAFT_INPUT_PATTERN = /^\d*\.?\d*$/;

  const props = defineProps<{ kind: 'cpu' | 'memory' }>();
  const model = defineModel<string>({ required: true });

  const { t } = useI18n();
  const selectRef = ref<InstanceType<typeof Select>>();
  const unitSelectRef = ref<InstanceType<typeof Select>>();
  const draftInputRef = ref<HTMLInputElement>();
  const isDraftOpen = ref(false);
  const isUnitSelectOpen = ref(false);
  const draft = ref('');
  const unit = ref<MemoryUnit>('Gi');
  const triedConfirm = ref(false);

  const currentQuantity = computed(() => (props.kind === 'cpu' ? formatCpuCores(model.value) : model.value));
  const options = computed(() => {
    const presets = props.kind === 'cpu' ? CPU_OPTIONS : MEMORY_OPTIONS;
    if (!model.value || presets.includes(model.value)) return presets;
    return [...presets.filter(option => option !== currentQuantity.value), model.value];
  });

  const draftError = computed(() => getDraftError(draft.value));
  const showDraftError = computed(() => Boolean(draftError.value && (draft.value || triedConfirm.value)));

  function closeDraft() {
    isUnitSelectOpen.value = false;
    isDraftOpen.value = false;
  }

  function confirmDraft() {
    triedConfirm.value = true;
    if (draftError.value) return;

    const quantity = props.kind === 'cpu' ? draft.value : `${draft.value}${unit.value}`;
    // 等量 CPU 保留原始写法，避免把继承的毫核值变成环境覆盖。
    if (quantity !== currentQuantity.value) model.value = quantity;
    closeDraft();
    if (props.kind === 'cpu') selectRef.value?.hidePopover();
  }

  function getDraftError(value: string) {
    if (!value) return t('不能为空');
    if (props.kind === 'cpu') {
      // 正则已限定纯数字，?? 0 仅做 null 收窄，不会实际命中。
      return /^\d+(?:\.\d)?$/.test(value) && (parseCpuCores(value) ?? 0) >= 0.1
        ? ''
        : t('请输入不小于 0.1 核且最多一位小数的数字');
    }
    return /^\d+(?:\.\d+)?$/.test(value) && (parseMemoryToMiB(`${value}${unit.value}`) ?? 0) >= 256
      ? ''
      : t('请输入不小于 256Mi 的数字');
  }

  function onBeforeDraftInput(event: InputEvent) {
    const text = event.data ?? event.dataTransfer?.getData('text/plain');
    if (text && !DRAFT_INPUT_PATTERN.test(text)) event.preventDefault();
  }

  function onDraftEnter(event: KeyboardEvent) {
    if (event.isComposing) return;
    event.preventDefault();
    confirmDraft();
  }

  function onDraftEscape() {
    if (isUnitSelectOpen.value) {
      unitSelectRef.value?.hidePopover();
    } else {
      closeDraft();
    }
  }

  function onDraftInput() {
    const input = draftInputRef.value;
    if (!input) return;
    if (DRAFT_INPUT_PATTERN.test(input.value)) {
      draft.value = input.value;
    } else {
      input.value = draft.value;
    }
  }

  function onToggle(open: boolean) {
    if (!open) closeDraft();
  }

  function openDraft() {
    unit.value = model.value.trim().endsWith('Mi') ? 'Mi' : 'Gi';
    const currentAmount = currentQuantity.value;
    draft.value = props.kind === 'cpu' && !getDraftError(currentAmount) ? currentAmount : '';
    triedConfirm.value = false;
    isDraftOpen.value = true;
    nextTick(() => draftInputRef.value?.focus());
  }

  function stepDraft(step: -1 | 1) {
    // 只调整整数部分，避免小数步进时产生浮点误差。
    const [integer, fraction] = draft.value.split('.');
    const nextInteger = BigInt(integer) + BigInt(step);
    const nextDraft = fraction ? `${nextInteger}.${fraction}` : String(nextInteger);
    const nextAmount = Number(nextDraft);
    if (Number.isFinite(nextAmount) && nextAmount > 0) draft.value = nextDraft;
  }
</script>

<style scoped>
  .draft-number-input {
    appearance: textfield;
  }

  .draft-number-input::-webkit-inner-spin-button,
  .draft-number-input::-webkit-outer-spin-button {
    appearance: none;
  }
</style>
