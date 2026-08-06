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
    ref="containerRef"
    class="flex overflow-hidden items-center w-full min-w-0"
    :style="{ gap: `${gap}px` }"
  >
    <template
      v-for="(item, index) in visibleTags"
      :key="index"
    >
      <span class="shrink-0 min-w-max inline-flex">
        <slot
          :index="index"
          :item="item"
        >
          <Tag class="shrink-0">
            {{ item }}
          </Tag>
        </slot>
      </span>
    </template>
    <Tag
      v-if="hiddenCount > 0"
      v-bk-tooltips="{ content: hiddenTags.join('，'), placement: 'top', theme: 'light' }"
      class="shrink-0 cursor-pointer"
    >
      +{{ hiddenCount }}
    </Tag>
  </div>

  <!--
    用于测量真实 tag 宽度的隐藏容器：避免字符宽度估算带来的边界偏差
    注意：这里复用同一个 slot，确保 closable/图标等影响宽度的内容也被考虑进去
  -->
  <div
    ref="measureRef"
    class="absolute invisible pointer-events-none flex items-center top-0 left-0"
    :style="{ gap: `${gap}px` }"
  >
    <template
      v-for="(item, index) in props.tags"
      :key="`measure-${index}`"
    >
      <span
        :ref="el => setMeasureTagRef(el, index)"
        class="shrink-0 min-w-max inline-flex"
      >
        <slot
          :index="index"
          :item="item"
        >
          <Tag class="shrink-0">
            {{ item }}
          </Tag>
        </slot>
      </span>
    </template>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';

  import { Tag } from 'bkui-vue';

  interface Props {
    gap?: number;
    moreTagWidth?: number;
    tagExtraWidth?: number;
    tags: string[];
  }

  const props = withDefaults(defineProps<Props>(), {
    gap: 4,
    moreTagWidth: 48,
    tagExtraWidth: 0,
  });

  /** 容器元素引用 */
  const containerRef = ref<HTMLElement | null>(null);
  /** 当前可见的 Tag 数量 */
  const visibleCount = ref(props.tags.length);

  /** 隐藏测量容器引用 */
  const measureRef = ref<HTMLElement | null>(null);
  /** 隐藏测量容器下每个 tag 的实际宽度元素引用 */
  const measureTagRefs = ref<HTMLElement[]>([]);
  /** 实际 tag 宽度缓存（与 props.tags 下标一一对应） */
  const realTagWidths = ref<number[]>([]);

  /** Tag 左右 padding 合计 */
  const TAG_PADDING = 16;
  /** 单字符预估宽度 */
  const CHAR_WIDTH = 7;

  /**
   * 根据容器宽度计算可显示的 Tag 数量
   * 先尝试不放 "+n" 标签的情况，若放不下则为 "+n" 标签预留空间后重新计算
   */
  let overflowRecalcSeq = 0;

  async function calcVisibleCount() {
    const seq = ++overflowRecalcSeq;
    if (!containerRef.value) return;

    const width = containerRef.value.clientWidth;
    if (width <= 0) return;

    const tags = props.tags;
    if (tags.length === 0) {
      visibleCount.value = 0;
      return;
    }

    // 优先使用真实测量宽度（避免字符估算偏差）
    const measured = realTagWidths.value;
    const tagWidths = measured.length === tags.length ? measured : null;

    const calcCount = (maxWidth: number) => {
      let usedWidth = 0;
      let count = 0;
      for (let i = 0; i < tags.length; i++) {
        const tagWidth = (tagWidths ? tagWidths[i] : measureTagWidth(tags[i])) + (i > 0 ? props.gap : 0);
        if (usedWidth + tagWidth <= maxWidth) {
          usedWidth += tagWidth;
          count++;
        } else {
          break;
        }
      }
      return count;
    };

    // 尝试不放 +n Tag 的情况
    const countWithoutMore = calcCount(width);
    // 若放得下所有 tag，就不需要预留 +n 空间
    const candidateCount = countWithoutMore === tags.length ? countWithoutMore : calcCount(width - props.moreTagWidth);

    // 保证至少显示 1 个 Tag
    visibleCount.value = Math.max(1, candidateCount);

    // 为避免“估算略偏小导致最后一个 tag 被裁一半”，基于真实 DOM overflow 兜底校正
    await nextTick();
    if (seq !== overflowRecalcSeq) return;

    // 等一帧，确保 resize/样式切换造成的布局已完全落地
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
    if (seq !== overflowRecalcSeq) return;

    const isOverflow = () => {
      if (!containerRef.value) return false;
      const containerEl = containerRef.value;
      const containerRect = containerEl.getBoundingClientRect();
      const lastEl = containerEl.lastElementChild as HTMLElement | null;
      if (!lastEl) return false;
      // 使用 bbox 判断是否在布局层面超出容器右边界（比 scrollWidth 更贴近“被裁一半”场景）
      return lastEl.getBoundingClientRect().right > containerRect.right + 0.5;
    };

    // 一般只会差 0~1 个 tag；用有限重试避免极端情况下多次触发布局开销
    const MAX_DOM_CORRECTION_STEPS = 20;
    for (let i = 0; i < MAX_DOM_CORRECTION_STEPS; i++) {
      if (visibleCount.value <= 1 || !isOverflow()) break;
      visibleCount.value -= 1;
      await nextTick();
      if (seq !== overflowRecalcSeq) return;
    }
  }

  function measureRealTagWidths() {
    if (!measureRef.value) return;
    const refs = measureTagRefs.value;
    if (!refs.length) return;
    realTagWidths.value = refs.map(el => Math.ceil(el.getBoundingClientRect().width));
  }

  /**
   * 预估单个 Tag 的渲染宽度
   * @param tag - Tag 文本内容
   */
  function measureTagWidth(tag: string): number {
    return TAG_PADDING + tag.length * CHAR_WIDTH + props.tagExtraWidth;
  }

  function setMeasureTagRef(el: unknown, index: number) {
    if (el instanceof HTMLElement) {
      measureTagRefs.value[index] = el;
    }
  }

  /** 当前可见的 Tag 列表 */
  const visibleTags = computed(() => props.tags.slice(0, visibleCount.value));
  /** 被隐藏的 Tag 列表 */
  const hiddenTags = computed(() => props.tags.slice(visibleCount.value));
  /** 被隐藏的 Tag 数量 */
  const hiddenCount = computed(() => hiddenTags.value.length);

  /** 容器尺寸变化监听器 */
  let resizeObserver: null | ResizeObserver = null;

  onMounted(() => {
    nextTick(async () => {
      // 确保隐藏测量容器已经完成布局再测宽
      await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
      measureRealTagWidths();
      calcVisibleCount();
    });
    if (containerRef.value) {
      resizeObserver = new ResizeObserver(() => {
        requestAnimationFrame(() => void calcVisibleCount());
      });
      resizeObserver.observe(containerRef.value);
    }
  });

  onUnmounted(() => {
    resizeObserver?.disconnect();
    resizeObserver = null;
  });

  /** tags 内容变化后，需要重新测量真实宽度 */
  watch(
    () => props.tags,
    async () => {
      measureTagRefs.value = [];
      await nextTick();
      await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
      measureRealTagWidths();
      calcVisibleCount();
    },
  );

  /** 只影响“排版计算”的参数变化，直接重算可见数量即可 */
  watch(
    () => [props.gap, props.moreTagWidth, props.tagExtraWidth],
    async () => {
      await nextTick();
      await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
      calcVisibleCount();
    },
  );
</script>
