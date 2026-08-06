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
    :class="[type === 'card' ? 'px-[20px] py-[24px] bg-[#fff] shadow-[0_2px_4px_0_#1919290d]' : '', 'flex flex-col']"
  >
    <div
      :class="[
        'flex items-center justify-between cursor-pointer text-[#313238] leading-[22px]',
        type === 'normal' ? 'px-[16px] py-[4px] text-[12px]' : 'text-[14px]',
        headerClass,
      ]"
      :style="normalStyle"
      @click="handleClick"
    >
      <div class="flex items-center">
        <!-- 展开/收起图标：slot > prop > 默认 RightShape，旋转 CSS 在组件内部 -->
        <span :class="['inline-flex transition duration-300', active ? rotateClass : '']">
          <template v-if="$slots.icon">
            <slot name="icon"></slot>
          </template>
          <i
            v-else-if="icon"
            :class="icon"
          ></i>
          <RightShape
            v-else
            fill="#313238"
            :height="12"
            :width="12"
          />
        </span>
        <slot name="title">
          <span class="ml-10px select-none font-bold text-[14px]">{{ name }}</span>
        </slot>
      </div>
      <div
        class="flex items-center"
        @click.stop
      >
        <slot name="header-right"></slot>
      </div>
    </div>
    <div
      v-show="active"
      :class="['mt-[10px] pt-[10px] overflow-hidden', contentClass]"
    >
      <slot></slot>
    </div>
  </div>
</template>
<script lang="ts" setup>
  import type { PropType } from 'vue';
  import { computed, ref, watch } from 'vue';

  import { RightShape } from 'bkui-vue/lib/icon';

  interface IEmits {
    (e: 'update:modelValue', value: boolean): void;
    (e: 'change', value: boolean): void;
    (e: 'click'): void;
  }

  const props = defineProps({
    name: {
      type: String,
      default: '标题',
    },
    type: {
      type: String as PropType<'card' | 'normal'>,
      default: 'card',
    },
    contentClass: {
      type: String,
      default: '',
    },
    headerClass: {
      type: String,
      default: '',
    },
    /** 自定义图标 CSS class（如 bkms-icon bkms-icon-angle-right），优先级低于 #icon 插槽 */
    icon: {
      type: String,
      default: '',
    },
    /** 展开时图标旋转的 Tailwind class，默认 rotate-90 */
    rotateClass: {
      type: String,
      default: 'rotate-90',
    },
    normalBgColor: {
      type: String,
      default: '#F0F1F5',
    },
    normalColor: {
      type: String,
      default: '#4D4F56',
    },
    stopPropagation: {
      type: Boolean,
      default: true,
    },
    modelValue: {
      type: Boolean,
      default: true,
    },
  });

  const emits = defineEmits<IEmits>();

  const internalActive = ref(true);

  // modelValue 外部传值时，同步到内部控制状态
  watch(
    () => props.modelValue,
    val => {
      if (val !== undefined) {
        internalActive.value = val;
      }
    },
    { immediate: true },
  );

  const active = computed({
    get: () => internalActive.value,
    set: val => {
      if (props.modelValue !== undefined) {
        emits('update:modelValue', val);
      }
      internalActive.value = val;
    },
  });

  const normalStyle = computed(() => {
    if (props.type !== 'normal') return {};
    return {
      backgroundColor: props.normalBgColor,
      color: props.normalColor,
    };
  });

  function handleClick(e: MouseEvent) {
    if (props.stopPropagation) e.stopPropagation();
    active.value = !active.value;
    emits('click');
  }

  /**
   * 切换折叠状态
   * @param status 是否折叠
   */
  function handleCollapse(status?: boolean) {
    if (status !== undefined) {
      active.value = status;
      return;
    }
    active.value = !active.value;
  }

  watch(active, val => {
    emits('change', val);
  });

  defineExpose({
    handleCollapse,
  });
</script>
