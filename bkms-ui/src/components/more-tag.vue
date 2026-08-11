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
    class="inline-flex items-center gap-[4px] w-full overflow-hidden"
  >
    <!-- 隐藏的测量容器，用于获取每个 item 的真实宽度 -->
    <div
      ref="measureRef"
      class="flex items-center gap-[4px] absolute invisible pointer-events-none whitespace-nowrap"
    >
      <span
        v-for="(item, index) in list"
        :key="index"
        class="inline-flex"
      >
        <slot
          :index="index"
          :item="item"
        >
          <Tag>{{ item }}</Tag>
        </slot>
      </span>
    </div>

    <!-- 实际可见的 items -->
    <template
      v-for="(item, index) in visibleItems"
      :key="index"
    >
      <slot
        :index="index"
        :item="item"
      >
        <Tag>{{ item }}</Tag>
      </slot>
    </template>

    <!-- overflowMode: popover -->
    <Popover
      v-if="hasMore && overflowMode === 'popover'"
      :component-event-delay="100"
      :theme="popoverTheme"
    >
      <Tag class="shrink-0"> +{{ remainingCount }} </Tag>
      <template #content>
        <slot
          :items="remainingItems"
          name="more-content"
          :visible-count="visibleCount"
        >
          <div class="flex flex-wrap gap-[4px]">
            <template
              v-for="(item, index) in remainingItems"
              :key="index"
            >
              <slot
                :index="index + visibleCount"
                :item="item"
                name="more"
              >
                <slot
                  :index="index + visibleCount"
                  :item="item"
                >
                  <Tag class="mr-[4px]">
                    {{ item }}
                  </Tag>
                </slot>
              </slot>
            </template>
          </div>
        </slot>
      </template>
    </Popover>

    <!-- overflowMode: tooltip -->
    <Tag
      v-if="hasMore && overflowMode === 'tooltip'"
      v-bk-tooltips="{ content: tooltipContent, placement: 'top', theme: 'light' }"
      class="shrink-0 cursor-pointer"
    >
      +{{ remainingCount }}
    </Tag>
  </div>
</template>

<script lang="ts" setup generic="T = unknown">
  import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';

  import { Popover, Tag } from 'bkui-vue';
  import { debounce } from 'lodash-es';

  interface Props {
    /** 数据列表（支持任意类型） */
    data?: T[];
    /** item 之间的间距，默认 4px（与 gap-[4px] 对应） */
    gap?: number;
    /** 可选的最大显示数量上限，不传则完全由容器宽度决定 */
    maxShow?: number;
    /** "+n" 标签的预估宽度，默认 40px */
    moreTagWidth?: number;
    /** 剩余项展示模式：popover（弹出面板）或 tooltip（文字提示） */
    overflowMode?: 'popover' | 'tooltip';
    popoverTheme?: string;
    /** 字符串标签列表（兼容原 OverflowTags 用法） */
    tags?: T[];
  }

  const props = withDefaults(defineProps<Props>(), {
    data: undefined,
    tags: undefined,
    maxShow: Infinity,
    moreTagWidth: 40,
    popoverTheme: 'light',
    gap: 4,
    overflowMode: 'popover',
  });

  defineSlots<{
    'more-content'?(props: { items: T[]; visibleCount: number }): void;
    default?(props: { index: number; item: T }): void;
    more?(props: { index: number; item: T }): void;
  }>();

  /** 统一数据源：优先使用 data，其次 tags */
  const list = computed(() => (props.data ?? props.tags ?? []) as T[]);

  const containerRef = ref<HTMLElement | null>(null);
  const measureRef = ref<HTMLElement | null>(null);
  const visibleCount = ref(list.value.length || 0);

  /**
   * 根据容器宽度和测量容器中每个 item 的实际宽度，计算可展示数量
   */
  function calcVisibleCount() {
    if (!containerRef.value || !measureRef.value) return;

    const containerWidth = containerRef.value.clientWidth;
    if (containerWidth <= 0) return;

    const items = Array.from(measureRef.value.children) as HTMLElement[];
    if (items.length === 0) {
      visibleCount.value = 0;
      return;
    }

    const total = items.length;

    // 尝试不放 "+n" 标签的情况
    let usedWidth = 0;
    let count = 0;
    for (let i = 0; i < total; i++) {
      const itemWidth = items[i].offsetWidth + (i > 0 ? props.gap : 0);
      if (usedWidth + itemWidth <= containerWidth) {
        usedWidth += itemWidth;
        count++;
      } else {
        break;
      }
    }

    // 全部放得下
    if (count >= total) {
      visibleCount.value = Math.min(total, props.maxShow);
      // 如果 maxShow 限制导致有剩余，需要为 "+n" 标签腾空间
      if (props.maxShow < total) {
        const available = containerWidth - props.moreTagWidth - props.gap;
        usedWidth = 0;
        count = 0;
        for (let i = 0; i < props.maxShow; i++) {
          const itemWidth = items[i].offsetWidth + (i > 0 ? props.gap : 0);
          if (usedWidth + itemWidth <= available) {
            usedWidth += itemWidth;
            count++;
          } else {
            break;
          }
        }
        visibleCount.value = Math.max(1, count);
      }
      return;
    }

    // 放不下全部，为 "+n" 标签预留空间重新计算
    const availableWidth = containerWidth - props.moreTagWidth - props.gap;
    usedWidth = 0;
    count = 0;
    const limit = Math.min(total, props.maxShow);
    for (let i = 0; i < limit; i++) {
      const itemWidth = items[i].offsetWidth + (i > 0 ? props.gap : 0);
      if (usedWidth + itemWidth <= availableWidth) {
        usedWidth += itemWidth;
        count++;
      } else {
        break;
      }
    }

    visibleCount.value = Math.max(1, count);
  }

  const visibleItems = computed(() => list.value.slice(0, visibleCount.value) || []);
  const remainingItems = computed(() => list.value.slice(visibleCount.value) || []);
  const hasMore = computed(() => (list.value.length || 0) > visibleCount.value);
  const remainingCount = computed(() => (list.value.length || 0) - visibleCount.value);
  /** tooltip 模式下的提示文本 */
  const tooltipContent = computed(() => remainingItems.value.join('，'));

  let resizeObserver: null | ResizeObserver = null;
  const debouncedCalcVisibleCount = debounce(calcVisibleCount, 100);

  onMounted(() => {
    nextTick(calcVisibleCount);
    resizeObserver = new ResizeObserver(debouncedCalcVisibleCount);
    if (containerRef.value) {
      resizeObserver.observe(containerRef.value);
    }
    // 观察 measureRef：当 slot 内容依赖异步数据（如名称映射）时，
    // item 宽度会随后变化，需要重新计算可见数量
    if (measureRef.value) {
      resizeObserver.observe(measureRef.value);
    }
  });

  onUnmounted(() => {
    resizeObserver?.disconnect();
    resizeObserver = null;
    debouncedCalcVisibleCount.cancel();
  });

  watch(
    () => list.value,
    () => {
      nextTick(calcVisibleCount);
    },
  );
</script>
