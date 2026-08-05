<template>
  <div class="flex flex-col gap-[12px]">
    <div
      v-for="item in options"
      :key="item.value"
      v-bk-tooltips="{
        content: item.disabledTip,
        disabled: !item.disabled || !item.disabledTip,
      }"
      class="flex items-center gap-[12px] px-[16px] py-[12px] rounded-[4px] border transition-all duration-200"
      :class="[
        item.disabled
          ? 'border-[#DCDEE5] bg-[#FAFBFD] cursor-not-allowed opacity-60'
          : modelValue === item.value
            ? 'border-[#3A84FF] bg-[#F0F5FF] cursor-pointer'
            : 'border-[#DCDEE5] bg-white hover:border-[#3A84FF] cursor-pointer',
      ]"
      @click="handleSelect(item)"
    >
      <div
        class="mt-[2px] w-[16px] h-[16px] rounded-full border-[1px] shrink-0 flex items-center justify-center transition-all duration-200"
        :class="[modelValue === item.value ? 'border-[#3A84FF]' : 'border-[#DCDEE5]']"
      >
        <div
          v-if="modelValue === item.value"
          class="w-[8px] h-[8px] rounded-full bg-[#3A84FF]"
        />
      </div>
      <div class="flex-1 min-w-0">
        <div
          class="text-[12px] leading-[20px] font-bold"
          :class="[modelValue === item.value ? 'text-[#3A84FF]' : 'text-[#313238]']"
        >
          {{ item.label }}
        </div>
        <div
          v-if="item.description"
          class="mt-[4px] text-[12px] leading-[20px]"
          :class="[modelValue === item.value ? 'text-[#699DF4]' : 'text-[#979BA5]']"
        >
          {{ item.description }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  export interface CardRadioOption {
    description?: string;
    /** 是否禁用该项；禁用后不可点击，并显示 disabledTip 提示 */
    disabled?: boolean;
    /** 禁用时的 hover 提示文案 */
    disabledTip?: string;
    label: string;
    value: string;
  }

  interface Props {
    options: CardRadioOption[];
  }

  defineProps<Props>();
  const modelValue = defineModel<string>('modelValue');
  const emit = defineEmits<{
    (e: 'change', value: string): void;
  }>();

  function handleSelect(item: CardRadioOption) {
    if (item.disabled) return;
    if (modelValue.value !== item.value) {
      modelValue.value = item.value;
      emit('change', item.value);
    }
  }
</script>
