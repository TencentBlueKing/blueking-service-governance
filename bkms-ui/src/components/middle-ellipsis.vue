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
  <span
    ref="containerRef"
    class="middle-ellipsis"
  >
    {{ displayText }}
  </span>
</template>

<script lang="ts" setup>
  import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

  interface Props {
    text?: string;
  }

  const props = defineProps<Props>();

  const containerRef = ref<HTMLElement>();
  const displayText = ref(props.text || '');
  let resizeObserver: ResizeObserver | undefined;

  function createCandidate(text: string, keepLength: number) {
    const headLength = Math.floor(keepLength * 0.4);
    const tailLength = keepLength - headLength;
    return `${text.slice(0, headLength)}…${text.slice(-tailLength)}`;
  }

  function fitsInTwoLines(text: string, width: number) {
    const measureNode = document.createElement('span');
    measureNode.className = 'middle-ellipsis middle-ellipsis-measure';
    measureNode.style.width = `${width}px`;
    measureNode.textContent = text;
    document.body.appendChild(measureNode);
    const fits = measureNode.getBoundingClientRect().height <= 40;
    measureNode.remove();
    return fits;
  }

  function getDisplayText(text: string) {
    const container = containerRef.value;
    if (!container || !text || typeof document === 'undefined') return text;

    const width = container.getBoundingClientRect().width;
    if (!width || fitsInTwoLines(text, width)) return text;

    let low = 1;
    let high = text.length - 1;
    let best = createCandidate(text, 1);

    while (low <= high) {
      const keepLength = Math.floor((low + high) / 2);
      const candidate = createCandidate(text, keepLength);
      if (fitsInTwoLines(candidate, width)) {
        best = candidate;
        low = keepLength + 1;
      } else {
        high = keepLength - 1;
      }
    }

    return best;
  }

  async function updateDisplayText() {
    await nextTick();
    displayText.value = getDisplayText(props.text || '');
  }

  watch(
    () => props.text,
    () => {
      displayText.value = props.text || '';
      updateDisplayText();
    },
  );

  onMounted(() => {
    updateDisplayText();
    if (typeof ResizeObserver === 'undefined') return;
    resizeObserver = new ResizeObserver(updateDisplayText);
    if (containerRef.value) resizeObserver.observe(containerRef.value);
  });

  onBeforeUnmount(() => resizeObserver?.disconnect());
</script>

<style>
  .middle-ellipsis {
    display: -webkit-box;
    width: 100%;
    max-width: 100%;
    max-height: 40px;
    overflow: hidden;
    -webkit-line-clamp: 2;
    line-height: 20px;
    word-break: break-all;
    white-space: normal;
    -webkit-box-orient: vertical;
  }

  .middle-ellipsis-measure {
    position: absolute;
    top: -9999px;
    left: -9999px;
    display: block;
    visibility: hidden;
    max-height: none;
    -webkit-line-clamp: unset;
  }
</style>
