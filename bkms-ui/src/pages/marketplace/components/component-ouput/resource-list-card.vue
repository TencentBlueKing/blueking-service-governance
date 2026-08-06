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
  <CollapseCard>
    <template #header-left>
      <slot name="header-left" />
    </template>
    <template #header-right>
      <slot name="header-right" />
    </template>
    <!-- 列表区域 -->
    <div
      ref="scrollContainerRef"
      class="flex flex-col gap-y-[12px]"
    >
      <div
        v-for="(item, index) in items"
        :key="item.id"
        class="bg-[#F0F1F5] p-[16px]"
      >
        <!-- 顶部行：slot header-left + 删除按钮 -->
        <div class="flex items-center mb-[12px]">
          <div class="flex-1 min-w-0">
            <slot
              :index="index"
              :item="item"
              name="item-header-left"
            />
          </div>
          <Button
            v-bk-tooltips="{
              content: t('组件输出至少要包含一个资源'),
              disabled: !disableRemove,
            }"
            class="px-[8px] h-[32px] text-[#4D4F56] hover:text-[#3A84FF] shrink-0 ml-[14px]"
            :disabled="disableRemove"
            text
            @click="handleRemove(index)"
          >
            <Del
              height="14px"
              width="14px"
            />
          </Button>
        </div>
        <!-- 编辑器区域 -->
        <MsEditorPlus
          :ref="el => setEditorRef(el, index)"
          :model-value="item.content"
          :title="editorTitle"
          :validator="[requiredValidator]"
          @update:model-value="val => handleContentUpdate(index, val)"
        />
      </div>
    </div>
    <!-- 添加按钮 -->
    <Button
      v-if="addButtonText"
      class="mt-[12px]"
      text
      theme="primary"
      @click="handleAdd"
    >
      <i class="bkms-icon bkms-icon-plus-circle-shape text-[12px] mr-[4px]"></i>
      <span class="text-[12px]">{{ addButtonText }}</span>
    </Button>
  </CollapseCard>
</template>

<script lang="ts" setup>
  import { nextTick, ref } from 'vue';

  import { Button } from 'bkui-vue';
  import { Del } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import MsEditorPlus from '~/components/monaco-editor/ms-editor-plus.vue';

  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  export interface IResourceListItem {
    content: string;
    id: number;
  }

  interface IProps {
    addButtonText?: string;
    /** 是否禁用删除按钮 */
    disableRemove?: boolean;
    editorTitle?: string;
  }

  withDefaults(defineProps<IProps>(), {
    disableRemove: false,
    addButtonText: '',
    editorTitle: '',
  });

  const { t } = useI18n();

  const items = defineModel<IResourceListItem[]>('items', { required: true });

  const emit = defineEmits<{
    (e: 'add'): void;
  }>();

  const scrollContainerRef = ref<HTMLElement>();
  const editorRefs = ref<(InstanceType<typeof MsEditorPlus> | null)[]>([]);

  function handleAdd() {
    emit('add');
    nextTick(() => {
      scrollContainerRef.value?.scrollTo({
        top: scrollContainerRef.value.scrollHeight,
        behavior: 'smooth',
      });
    });
  }

  function handleContentUpdate(index: number, val: string) {
    items.value[index] = { ...items.value[index], content: val };
  }

  function handleRemove(index: number) {
    items.value.splice(index, 1);
  }

  /** 校验所有编辑器是否有错误（含自定义 validator 错误） */
  function isValid() {
    // 用 map 遍历所有编辑器执行校验（map 不会短路，确保每个编辑器都被校验）
    const results = editorRefs.value?.map(editor => editor?.validate()) ?? [];
    if (!results.length) return true;
    return results.every(Boolean);
  }

  /** 必填校验：内容为空则报错 */
  function requiredValidator(value: string): IMonacoEditorErrorMarkerItem[] {
    if (!value || !value.trim()) {
      return [
        {
          severity: 4 as number,
          message: t('不能为空'),
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: 1,
          endColumn: 1,
        },
      ];
    }
    return [];
  }

  function setEditorRef(el: unknown, index: number) {
    if (el) {
      editorRefs.value[index] = el as InstanceType<typeof MsEditorPlus>;
    }
  }

  defineExpose({ scrollContainerRef, isValid });
</script>
