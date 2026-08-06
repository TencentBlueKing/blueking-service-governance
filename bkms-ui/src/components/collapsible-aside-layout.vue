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
  <ResizeLayout
    class="collapsible-aside-layout"
    collapsible
    :initial-divide="initialDivide"
    :is-collapsed="isCollapsed"
    :max="asideMaxWidth"
    :min="min"
    :placement="placement"
    :style="containerStyle"
    @after-resize="handleAfterResize"
    @collapse-change="handleCollapseChange"
  >
    <template #collapse-trigger>
      <slot name="collapse-trigger" />
    </template>
    <template #main>
      <slot name="main" />
    </template>
    <template #aside>
      <slot name="aside" />
    </template>
  </ResizeLayout>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { ResizeLayout } from 'bkui-vue';

  interface LayoutConfig {
    /** CSS 高度值（如 "100%"、"500px"），默认 "100%" */
    height?: string;
    /** 视口顶部偏移量（px），基于 window.innerHeight 计算高度，优先级最高 */
    viewportOffset?: number;
  }

  /** 可变宽度的侧边栏的最大宽度偏移量 */
  const ASIDE_MAX_OFFSET_WITDH = 3;

  interface Props {
    initialDivide?: number | string;
    layoutConfig?: LayoutConfig;
    max?: number;
    min?: number;
    placement?: 'left' | 'right';
  }

  const props = withDefaults(defineProps<Props>(), {
    placement: 'right',
    max: 800,
    min: 500,
    initialDivide: '50%',
    layoutConfig: () => ({ height: '100%' }),
  });

  const isCollapsed = defineModel<boolean>('isCollapsed', { default: false });

  const emit = defineEmits<{
    (e: 'after-resize'): void;
  }>();

  /** 侧边栏的去除偏移量的最大宽度 */
  const asideMaxWidth = computed(() => props.max - ASIDE_MAX_OFFSET_WITDH);

  function handleAfterResize() {
    emit('after-resize');
  }

  const containerStyle = computed(() => {
    const style: Record<string, string> = {};
    if (props.layoutConfig?.viewportOffset !== undefined) {
      // +1px 避免出现滚动条
      style.height = `calc(100vh - ${props.layoutConfig.viewportOffset + 1}px)`;
      return style;
    }
    style.height = props.layoutConfig?.height ?? '100%';
    return style;
  });

  function handleCollapseChange(val: boolean) {
    isCollapsed.value = val;
  }
</script>

<style lang="postcss" scoped>
  .collapsible-aside-layout {
    border: none;

    & > :deep(.bk-resize-layout-aside) {
      min-height: 0;
    }
    & > :deep(.bk-resize-layout-main) {
      min-height: 0;
      overflow-y: auto;
      scrollbar-gutter: stable;
    }
  }
</style>
