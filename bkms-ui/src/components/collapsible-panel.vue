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
      <div
        class="collapse-handle"
        :class="position === 'left' ? 'is-left' : 'is-right'"
        @click="toggle"
      >
        <component
          :is="collapsed ? expandIcon : collapseIcon"
          width="26"
          height="26"
        />
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

  function toggle() {
    const newCollapsed = !props.collapsed;
    emits('update:collapsed', newCollapsed);
    emits('toggle', newCollapsed);
  }
</script>

<style lang="postcss" scoped>
  /* 对齐 bkui ResizeLayout 的 .bk-resize-collapse：16×64，白箭头 26，悬停主色。 */
  .collapse-handle {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 64px;
    color: #fff;
    cursor: pointer;
    background: #E4E9F2;
    border: 0;
  }

  .collapse-handle.is-left {
    border-radius: 0 4px 4px 0;
  }

  .collapse-handle.is-right {
    border-radius: 4px 0 0 4px;
  }

  .collapse-handle:hover {
    background: #3B6EF5;
  }

  .collapse-handle :deep(svg) {
    position: absolute;
    width: 26px;
    height: 26px;
    fill: currentcolor;
  }
</style>
