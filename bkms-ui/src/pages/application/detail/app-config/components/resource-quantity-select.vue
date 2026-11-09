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
      <Button
        v-if="!isDraftOpen"
        class="ml-[12px]"
        text
        theme="primary"
        @click="openDraft"
      >
        {{ $t('自定义') }}
      </Button>
      <div
        v-else
        class="flex w-full items-center"
        :class="kind === 'memory' ? 'gap-[8px] px-[12px]' : 'gap-[4px] px-[8px]'"
        @keydown.esc.prevent="closeDraft"
        @keydown.stop
      >
        <div
          class="flex min-w-0 flex-1 items-center rounded-[2px] border bg-white"
          :class="[
            kind === 'memory' ? 'h-[26px]' : 'h-[32px]',
            showDraftError ? 'border-[#EA3636]' : 'border-[#C4C6CC] focus-within:border-[#3A84FF]',
          ]"
        >
          <input
            ref="draftInputRef"
            v-model="draft"
            :aria-invalid="showDraftError"
            :aria-label="$t('自定义')"
            class="h-full min-w-0 flex-1 bg-transparent px-[10px] text-[12px] text-[#63656E] outline-none placeholder:text-[#C4C6CC]"
            inputmode="decimal"
            :placeholder="$t('请输入')"
            type="text"
            @keydown.enter="onDraftEnter"
          />
          <span
            v-if="showDraftError"
            v-bk-tooltips="draftError"
            :aria-label="draftError"
            class="mr-[8px] flex shrink-0 items-center text-[16px] text-[#EA3636]"
          >
            <ExclamationCircleShape />
          </span>
        </div>
        <div
          v-if="kind === 'memory'"
          :aria-label="$t('内存单位')"
          class="flex shrink-0 overflow-hidden rounded-[2px] border border-[#C4C6CC]"
          role="group"
        >
          <button
            v-for="memoryUnit in MEMORY_UNITS"
            :key="memoryUnit"
            :aria-pressed="unit === memoryUnit"
            class="h-[26px] cursor-pointer px-[8px] text-[12px] focus-visible:outline focus-visible:outline-[1px] focus-visible:outline-offset-[-1px] focus-visible:outline-[#3A84FF]"
            :class="unit === memoryUnit ? 'bg-[#E1ECFF] text-[#3A84FF]' : 'bg-white text-[#63656E]'"
            type="button"
            @click="unit = memoryUnit"
          >
            {{ memoryUnit }}
          </button>
        </div>
        <button
          :aria-label="$t('确定')"
          class="flex h-[24px] shrink-0 cursor-pointer items-center justify-center"
          :class="kind === 'memory' ? 'w-[16px]' : 'w-[20px]'"
          type="button"
          @click="confirmDraft"
        >
          <Done
            fill="#2CAF5E"
            :height="18"
            :width="18"
          />
        </button>
        <button
          :aria-label="$t('取消')"
          class="flex h-[24px] shrink-0 cursor-pointer items-center justify-center"
          :class="kind === 'memory' ? 'w-[14px]' : 'w-[20px]'"
          type="button"
          @click="closeDraft"
        >
          <Error
            fill="#979BA5"
            :height="14"
            :width="14"
          />
        </button>
      </div>
    </template>
  </Select>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref } from 'vue';

  import { Button, Select } from 'bkui-vue';
  import { Done, Error, ExclamationCircleShape } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { formatCpuCores, parseCpuCores, parseMemoryToMiB } from '~/common/resource-quantity';

  type MemoryUnit = 'Gi' | 'Mi';

  const CPU_OPTIONS = ['0.1', '0.2', '0.5', '1', '2', '4', '8', '16', '32'];
  const MEMORY_OPTIONS = ['256Mi', '512Mi', '1Gi', '2Gi', '4Gi', '8Gi', '16Gi', '32Gi', '64Gi'];
  const MEMORY_UNITS: MemoryUnit[] = ['Gi', 'Mi'];

  const props = defineProps<{ kind: 'cpu' | 'memory' }>();
  const model = defineModel<string>({ required: true });

  const { t } = useI18n();
  const selectRef = ref<InstanceType<typeof Select>>();
  const draftInputRef = ref<HTMLInputElement>();
  const isDraftOpen = ref(false);
  const draft = ref('');
  const unit = ref<MemoryUnit>('Gi');
  const triedConfirm = ref(false);

  const currentQuantity = computed(() => (props.kind === 'cpu' ? formatCpuCores(model.value) : model.value));
  const options = computed(() => {
    const presets = props.kind === 'cpu' ? CPU_OPTIONS : MEMORY_OPTIONS;
    if (!model.value || presets.includes(model.value)) return presets;
    return [...presets.filter(option => option !== currentQuantity.value), model.value];
  });

  const draftAmount = computed(() => draft.value.trim());
  const draftError = computed(() => getDraftError(draftAmount.value));
  const showDraftError = computed(() => Boolean(draftError.value && (draftAmount.value || triedConfirm.value)));

  function closeDraft() {
    isDraftOpen.value = false;
  }

  function confirmDraft() {
    triedConfirm.value = true;
    if (draftError.value) return;

    const amount = draftAmount.value;
    const quantity = props.kind === 'cpu' ? amount : `${amount}${unit.value}`;
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

  function onDraftEnter(event: KeyboardEvent) {
    if (event.isComposing) return;
    event.preventDefault();
    confirmDraft();
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
</script>
