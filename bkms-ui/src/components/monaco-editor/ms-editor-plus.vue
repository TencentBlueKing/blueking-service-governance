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
    ref="fullscreenTargetRef"
    class="w-full"
    :style="{ height: `${containerHeight}px` }"
  >
    <ResizeLayout
      ref="resizeLayoutRef"
      :auto-minimize="true"
      class="h-full w-full border-none"
      :collapsible="true"
      :disabled="!editorErr.length"
      :initial-divide="rememberedHeight"
      :is-collapsed="isCollapsed"
      :max="resizeMaxHeight"
      :min="48"
      placement="bottom"
      @collapse-change="handleCollapseChange"
      @fullscreen="handleFullscreenChange"
    >
      <template
        v-if="!editorErr.length"
        #collapse-trigger
      />
      <template #aside>
        <div class="relative h-full">
          <EditorStatus :message="editorErr" />
          <Error
            class="absolute top-[8px] right-[8px] cursor-pointer text-[#979BA5] hover:text-[#C4C6CC]"
            :height="18"
            :width="18"
            @click="isCollapsed = true"
          />
        </div>
      </template>
      <template #main>
        <MsEditor
          ref="editorRef"
          :fullscreen-target-ref="fullscreenTargetRef"
          :model-value="modelValue"
          :readonly="readonly"
          :title="title"
          :validator="validator"
          @error="handleEditorError"
          @fullscreen="handleFullscreenChange"
          @update:model-value="val => $emit('update:modelValue', val)"
        />
      </template>
    </ResizeLayout>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onMounted, ref, watch } from 'vue';

  import { ResizeLayout } from 'bkui-vue';
  import { Error } from 'bkui-vue/lib/icon';

  import EditorStatus from './editor-status.vue';
  import MsEditor from './ms-editor.vue';

  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  interface IProps {
    containerHeight?: number;
    modelValue: string;
    readonly?: boolean;
    title?: string;
    validator?: ((value: string) => IMonacoEditorErrorMarkerItem[])[];
  }

  const props = withDefaults(defineProps<IProps>(), {
    title: '',
    readonly: false,
    validator: () => [],
    containerHeight: 300,
  });

  defineEmits<{
    (e: 'update:modelValue', value: string): void;
  }>();

  /** resizeLayout组件设置maxHeight时与实际设置的偏差值 */
  const RESIZE_LAYOUT_OFFSET_HEIGHT = {
    default: 3,
    fullscreen: 6,
  };
  /** 编辑器工具栏高度 */
  const EDITOR_TOOLBAR_HEIGHT = 40;
  /** 编辑器最大高度百分比 */
  const MAX_PERCENT = 0.9;
  const MIN_HEIGHT = 48;
  const fullscreenTargetRef = ref<HTMLElement | null>(null);
  const editorRef = ref<InstanceType<typeof MsEditor> | null>(null);

  /** 当前编辑器错误信息列表 */
  const editorErr = ref<string[]>([]);
  /** 折叠面板是否折叠 */
  const isCollapsed = ref(true);
  /** ResizeLayout 最大高度 */
  const resizeMaxHeight = ref(0);
  const resizeLayoutRef = ref();
  /** 折叠前记忆的 aside 高度 */
  const rememberedHeight = ref(MIN_HEIGHT);

  const defaultMaxHeight = computed(
    () => (props.containerHeight - EDITOR_TOOLBAR_HEIGHT) * MAX_PERCENT - RESIZE_LAYOUT_OFFSET_HEIGHT.default,
  );

  // 在 DOM 更新前捕获折叠前的高度（flush: 'pre' 时 DOM 尚未 patched，实际高度仍是旧值）
  watch(
    isCollapsed,
    (newVal, oldVal) => {
      if (oldVal === false && newVal === true) {
        // 正在折叠，DOM 尚未更新，读取当前 aside 实际高度并记忆
        const asideEl = resizeLayoutRef.value?.asideRef;
        const asideHeight = asideEl?.style.height?.match(/\d+/)?.[0];
        if (asideHeight) {
          rememberedHeight.value = Number(asideHeight);
        }
      } else if (oldVal === true && newVal === false) {
        // 正在展开，DOM 更新后恢复记忆的高度（兜底：不超过当前 maxHeight）
        nextTick(() => {
          const asideEl = resizeLayoutRef.value?.asideRef;
          if (asideEl && rememberedHeight.value > MIN_HEIGHT) {
            asideEl.style.height = `${Math.min(rememberedHeight.value, resizeMaxHeight.value)}px`;
          }
        });
      }
    },
    { flush: 'pre' },
  );

  function handleCollapseChange(val: boolean) {
    isCollapsed.value = val;
  }

  function handleEditorError(err: IMonacoEditorErrorMarkerItem[]) {
    editorErr.value = err.map(item => item.message);
    if (err.length) {
      isCollapsed.value = false;
    } else {
      isCollapsed.value = true;
    }
  }

  async function handleFullscreenChange(isFullscreen: boolean) {
    if (isFullscreen) {
      const fullscreenMaxHeight =
        (window.innerHeight - EDITOR_TOOLBAR_HEIGHT) * MAX_PERCENT - RESIZE_LAYOUT_OFFSET_HEIGHT.fullscreen;
      resizeMaxHeight.value = fullscreenMaxHeight;
    } else {
      resizeMaxHeight.value = defaultMaxHeight.value;
      if (!resizeLayoutRef.value.isCollapsed) {
        const asideHeight = resizeLayoutRef.value.asideRef.style.height?.match(/\d+/)?.[0];
        if (Number(asideHeight) > defaultMaxHeight.value) {
          resizeLayoutRef.value.asideRef.style.height = `${defaultMaxHeight.value}px`;
        }
      }
    }
  }

  /** 校验编辑器是否有错误，返回 true 表示有错误 */
  function validate() {
    const hasError = editorRef.value?.validate() ?? false;
    if (hasError) {
      isCollapsed.value = false;
    }
    // 校验成功返回true，失败返回false
    return !hasError;
  }

  onMounted(() => {
    resizeMaxHeight.value = defaultMaxHeight.value;
  });

  defineExpose({ validate });
</script>

<style lang="postcss" scoped>
  :deep(.bk-resize-layout) {
    .bk-resize-layout-aside {
      border-top: 6px;
      margin-top: -1px;
      position: relative;

      .bk-resize-trigger {
        background-color: #2e2e2e !important;

        &::before {
          width: 4px;
          height: 5px;
          content: '';
          background-color: #b34747;
          position: absolute;
        }

        &::after {
          position: absolute;
          width: 2px;
          height: 2px;
          color: #c4c6cc;
          background: #c4c6cc;
          content: '';
          box-shadow:
            0 4px 0 0 #c4c6cc,
            0 8px 0 0 #c4c6cc,
            0 -4px 0 0 #c4c6cc,
            0 -8px 0 0 #c4c6cc;
          top: 4px;
          left: 50%;
          transform: translate3d(-50%, 0, 0) rotate(90deg);
          z-index: 1;
        }
      }

      .bk-resize-collapse {
        display: none !important;
      }
    }
  }

  :deep(.bk-resize-layout-collapsed) {
    .bk-resize-layout-aside {
      margin-top: -1px;

      .bk-resize-trigger {
        height: 0 !important;

        &::before {
          display: none;
        }

        &::after {
          display: none;
        }
      }

      .bk-resize-collapse {
        background-color: #4d4f56;
        display: inline-flex !important;
        bottom: calc(100% - 1px);

        &:hover {
          background-color: #3a84ff;
        }
      }
    }
  }

  :deep(.monaco-editor) {
    outline-color: #1e1e1e;
  }
</style>
