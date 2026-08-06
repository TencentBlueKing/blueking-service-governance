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
  <!-- 加密交互输入框 -->
  <div class="sensitive-value-input">
    <!-- 占位状态 -->
    <div
      v-if="isUnchanged"
      class="flex items-center h-[32px] px-[10px] border border-solid border-[#c4c6cc] rounded-[2px] cursor-text bg-white hover:border-[#979ba5]"
      @click="handlePlaceholderClick"
    >
      <SensitiveValuePlaceholder :count="unchangedDotCount" />
    </div>
    <!-- 聚焦-编辑态 -->
    <div
      v-else
      class="relative"
    >
      <Input
        ref="inputRef"
        v-model.trim="value"
        class="w-full relative z-99"
        :class="{ 'is-selected-placeholder': isSelected }"
        :clearable="props.mode === 'create'"
        :placeholder="isSelected ? '' : displayPlaceholder"
        @blur="handleBlur"
        @enter="emit('enter')"
        @focus="handleFocus"
        @keydown="handleKeydown"
        @paste="handlePaste"
      >
        <template
          v-if="props.mode === 'edit'"
          #suffix
        >
          <div class="flex items-center pr-[8px] bg-[#fff]">
            <i
              v-if="showResetIcon"
              v-bk-tooltips="$t('恢复')"
              class="bkms-icon bkms-icon-redo cursor-pointer text-[16px] hover:text-[#3a84ff]"
              @click.stop.prevent="reset"
              @mousedown.stop.prevent
            />
          </div>
        </template>
      </Input>
      <div
        v-if="isSelected"
        class="pointer-events-none absolute left-[1px] top-[1px] z-[100] flex h-[30px] items-center px-[9px]"
      >
        <div class="flex h-[24px] items-center bg-[#e1ecff] px-[4px]">
          <SensitiveValuePlaceholder :count="unchangedDotCount" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import { Input } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import SensitiveValuePlaceholder from './sensitive-value-placeholder.vue';

  // unchanged: 未触碰已有敏感值；selected: 聚焦后选中占位，但还未决定覆盖原值。
  type InputState = 'changed' | 'editing' | 'selected' | 'unchanged';

  interface IProps {
    mode: 'create' | 'edit';
    placeholder?: string;
    unchangedText?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    placeholder: '',
    unchangedText: '••••••',
  });

  const emit = defineEmits<{
    enter: [];
    reset: [];
  }>();

  const value = defineModel<string>({ default: '' });
  const modified = defineModel<boolean>('modified', { default: false });
  const { t } = useI18n();

  const inputRef = ref<InstanceType<typeof Input>>();
  const state = ref<InputState>(props.mode === 'edit' ? 'unchanged' : 'editing');
  let isResetting = false;

  const displayPlaceholder = computed(() => props.placeholder || t('请输入'));
  const isSelected = computed(() => props.mode === 'edit' && state.value === 'selected');
  const isUnchanged = computed(() => props.mode === 'edit' && state.value === 'unchanged');
  const showResetIcon = computed(() => props.mode === 'edit' && ['changed', 'editing'].includes(state.value));
  const unchangedDotCount = computed(() => Array.from(props.unchangedText).length || 6);

  // 只进入选中态，不标记 modified，避免用户仅聚焦后保存时误覆盖后端密文。
  function enterSelected() {
    if (props.mode !== 'edit' || state.value !== 'unchanged') {
      return;
    }
    state.value = 'selected';
    modified.value = false;
    value.value = '';
  }

  function handleBlur() {
    if (state.value === 'selected') {
      state.value = 'unchanged';
    }
  }

  function handleFocus() {
    enterSelected();
  }

  // selected 态下拦截首个输入键，先切到普通输入态再写值，避免占位和输入内容切换时闪动。
  function handleKeydown(valOrEvt: KeyboardEvent | string, evt?: KeyboardEvent) {
    const keyboardEvent = evt || (typeof valOrEvt === 'object' && 'key' in valOrEvt ? valOrEvt : undefined);
    if (state.value !== 'selected' || !shouldStartInput(keyboardEvent)) {
      return;
    }
    keyboardEvent?.preventDefault();
    startInput(keyboardEvent?.key.length === 1 ? keyboardEvent.key : '');
  }

  // 粘贴也视为开始覆盖原敏感值，走同一套 selected -> editing/changed 转换。
  function handlePaste(valOrEvt: ClipboardEvent | string, evt?: ClipboardEvent) {
    const clipboardEvent = evt || (typeof valOrEvt === 'object' && 'clipboardData' in valOrEvt ? valOrEvt : undefined);
    if (state.value !== 'selected') {
      return;
    }
    clipboardEvent?.preventDefault();
    startInput(clipboardEvent?.clipboardData?.getData('text') || '');
  }

  function handlePlaceholderClick() {
    enterSelected();
    nextTick(() => {
      inputRef.value?.focus();
    });
  }

  function markAsModified(val = value.value) {
    state.value = val ? 'changed' : 'editing';
    modified.value = true;
  }

  function reset() {
    isResetting = true;
    value.value = '';
    modified.value = false;
    state.value = props.mode === 'edit' ? 'unchanged' : 'editing';
    inputRef.value?.blur?.();
    emit('reset');
    nextTick(() => {
      isResetting = false;
    });
  }

  function shouldStartInput(evt?: KeyboardEvent) {
    if (!evt || evt.ctrlKey || evt.metaKey || evt.altKey) {
      return false;
    }
    return evt.key.length === 1 || ['Backspace', 'Delete'].includes(evt.key);
  }

  function startInput(val: string) {
    markAsModified(val);
    value.value = val;
  }

  watch(
    () => props.mode,
    mode => {
      state.value = mode === 'edit' ? 'unchanged' : 'editing';
      modified.value = false;
      value.value = '';
    },
  );

  watch(value, val => {
    if (props.mode === 'create' || isResetting || state.value === 'unchanged') {
      return;
    }
    if (state.value === 'selected' && val === '') {
      return;
    }
    markAsModified(val);
  });

  defineExpose({
    reset,
  });
</script>

<style lang="postcss" scoped>
  .sensitive-value-input {
    :deep(.is-selected-placeholder input) {
      caret-color: transparent;

      &::placeholder {
        color: transparent;
      }
    }
  }
</style>
