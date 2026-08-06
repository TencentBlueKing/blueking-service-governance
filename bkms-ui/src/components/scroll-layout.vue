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
  <div class="w-full h-full flex flex-col">
    <!-- Content 区域：未溢出时保持自然高度，溢出时收缩并启用滚动 -->
    <div
      ref="contentRef"
      :class="['flex-[0_1_auto] min-h-0 overflow-y-auto flex flex-col items-center', contentClass]"
    >
      <slot :has-scroll="hasScroll" />
    </div>
    <!-- Footer 区域：不收缩，始终在底部，滚动条不会进入此区域 -->
    <div :class="['flex-shrink-0', footerClass]">
      <slot
        :has-scroll="hasScroll"
        name="footer"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { onBeforeUnmount, onMounted, ref } from 'vue';

  defineProps<{
    /** Content 区域附加样式类 */
    contentClass?: string;
    /** Footer 区域附加样式类 */
    footerClass?: string;
  }>();

  const contentRef = ref<HTMLElement | null>(null);
  const hasScroll = ref(false);
  let observer: null | ResizeObserver = null;

  onMounted(() => {
    if (contentRef.value) {
      observer = new ResizeObserver(() => {
        if (!contentRef.value) return;
        hasScroll.value = contentRef.value.scrollHeight > contentRef.value.clientHeight;
      });
      observer.observe(contentRef.value);
    }
  });

  onBeforeUnmount(() => {
    observer?.disconnect();
    observer = null;
  });

  defineExpose({
    hasScroll,
  });
</script>
