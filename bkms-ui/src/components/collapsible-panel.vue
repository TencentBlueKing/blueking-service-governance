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
  <div class="relative bg-white shadow-[2px_2px_4px_0_#1919290d]">
    <!-- 可折叠面板 -->
    <div
      class="transition-all duration-300 flex-shrink-0 h-full"
      :class="panelClass"
      :style="{ width: collapsed ? '0' : `${width}px`, overflow: 'hidden' }"
    >
      <div
        class="h-full"
        :style="{ width: `${width}px` }"
      >
        <slot></slot>
      </div>
    </div>
    <!-- 折叠按钮 -->
    <div
      class="absolute top-1/2 transform -translate-y-1/2 z-10 transition-all duration-300"
      :style="{ [position]: collapsed ? '0px' : `${width}px` }"
    >
      <div class="flex">
        <div
          class="flex items-center justify-center w-[16px] h-[64px] bg-[#DCDEE5] border-0 cursor-pointer"
          :style="collapsed ? buttonStyle.expand : buttonStyle.collapse"
          @click="toggle"
        >
          <component
            :is="collapsed ? expandIcon : collapseIcon"
            class="text-[16px] text-[#63656E]"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { AngleLeft, AngleRight } from 'bkui-vue/lib/icon';

  interface IEmits {
    (e: 'update:collapsed', collapsed: boolean): void;
    (e: 'toggle', collapsed: boolean): void;
  }

  interface IProps {
    // 是否折叠
    collapsed?: boolean;
    // 面板额外样式类
    panelClass?: string;
    // 折叠位置: left | right
    position?: 'left' | 'right';
    // 面板宽度
    width?: number;
  }

  const props = withDefaults(defineProps<IProps>(), {
    width: 260,
    collapsed: false,
    position: 'left',
    panelClass: '',
  });

  const emits = defineEmits<IEmits>();

  // 根据位置确定图标
  const expandIcon = computed(() => {
    return props.position === 'left' ? AngleRight : AngleLeft;
  });

  const collapseIcon = computed(() => {
    return props.position === 'left' ? AngleLeft : AngleRight;
  });

  // 根据位置确定按钮样式
  const buttonStyle = computed(() => {
    if (props.position === 'left') {
      return {
        expand: 'border-radius: 0 4px 4px 0;',
        collapse: 'border-radius: 0 4px 4px 0;',
      };
    } else {
      return {
        expand: 'border-radius: 4px 0 0 4px;',
        collapse: 'border-radius: 4px 0 0 4px;',
      };
    }
  });

  function toggle() {
    const newCollapsed = !props.collapsed;
    emits('update:collapsed', newCollapsed);
    emits('toggle', newCollapsed);
  }
</script>
