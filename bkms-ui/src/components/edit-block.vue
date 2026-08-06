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
  <div>
    <div
      v-if="!isEdit"
      class="flex items-center text-[12px] flex-wrap"
      :class="isShowAtFirstLine ? '!items-start' : ''"
    >
      <slot name="text"></slot>
      <Button
        class="text-[#979BA5] ml-[5px]"
        :disabled="disabled"
        text
        @click="handleClickEdit"
      >
        <slot name="edit-icon">
          <EditLine :style="{ fontSize: computedIconSize }"> </EditLine>
        </slot>
      </Button>
    </div>
    <div
      v-else
      class="flex items-center text-[12px] z-[1000]"
    >
      <slot
        :focus="handleFocus"
        name="edit"
      ></slot>
      <SvgIcon
        v-if="loading"
        height="16px"
        icon="bkms-icon-loading"
        width="16px"
      />
      <div
        v-else
        class="flex items-center"
        :class="isShowAtFirstLine ? '!items-start mt-[6px]' : ''"
      >
        <Button
          text
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Divider
          class="h-[10px] !mx-[8px]"
          color="#DCDEE5"
          direction="vertical"
          type="solid"
        />
        <Button
          text
          theme="primary"
          @click="handleCancel"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import { Button, Divider, TagInput } from 'bkui-vue';
  import { EditLine } from 'bkui-vue/lib/icon';

  import type { InputType } from 'bkui-vue/lib/input/input';

  const props = defineProps({
    disabled: {
      type: Boolean,
      default: false,
    },
    isShowAtFirstLine: {
      type: Boolean,
      default: false,
    },
    iconSize: {
      type: [String, Number],
      default: 12,
    },
    loading: {
      type: Boolean,
    },
  });

  const emit = defineEmits(['edit', 'confirm', 'cancel']);

  const isEdit = ref(false);

  // 计算图标大小，自动添加 px 单位
  const computedIconSize = computed(() => {
    const size = props.iconSize;
    const numValue = Number(size);
    return isNaN(numValue) ? size : `${numValue}px`;
  });

  const handleClickEdit = () => {
    if (props.disabled) return;
    emit('edit');
    isEdit.value = true;
  };

  const handleConfirm = () => {
    emit('confirm');
  };

  const handleCancel = () => {
    emit('cancel');
    isEdit.value = false;
  };

  // 监听 loading 变化，从 true 变为 false 时关闭编辑态
  watch(
    () => props.loading,
    (newVal, oldVal) => {
      if (oldVal && !newVal && isEdit.value) {
        isEdit.value = false;
      }
    },
  );

  /**
   * @param componentRef 组件ref
   * @description 用于处理Input, TagInput自动聚焦逻辑
   */
  const handleFocus = async (componentRef: InputType | typeof TagInput, type: 'input' | 'tagInput' = 'input') => {
    await nextTick();
    if (!componentRef) return;
    if (type === 'input') {
      componentRef.focus();
    } else if (type === 'tagInput') {
      componentRef.focusInputTrigger();
    }
  };
</script>
