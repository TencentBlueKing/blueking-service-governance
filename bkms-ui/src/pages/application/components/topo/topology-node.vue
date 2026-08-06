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
    ref="nodeRef"
    class="antialiased"
    :style="
      nodeCount > MINIMAP_NODE_MAX || !isIntersecting
        ? null
        : {
            zoom: scaleFactor,
            transform: `scale(${1 / scaleFactor})`,
            transformOrigin: 'top left',
          }
    "
  >
    <template v-if="isIntersecting">
      <div
        :class="[
          'relative flex items-center px-[8px] py-[4px] rounded-[10px] border border-[#fff] shadow-[0_4px_8px_0_rgba(41,45,51,0.08)]',
          data.data?.kind !== 'App' ? 'topo-node cursor-pointer' : '',
        ]"
        :style="nodeStyle"
      >
        <div
          class="flex items-center justify-center size-[32px] rounded-[6px] shrink-0 mr-[8px]"
          :style="{
            background: statusConfig.iconBgColor,
            color: statusConfig.color,
          }"
        >
          <component
            :is="KIND_ICON_MAP[data.data?.kind || '']"
            v-if="KIND_ICON_MAP[data.data?.kind || '']"
            :class="['size-[24px] svg-icon', data.data?.kind === 'PolarisConfig' ? 'text-[#fff]' : '']"
          />
          <span
            v-else
            class="text-[16px] font-600 text-[#fff]"
          >
            {{ data.data?.kind?.charAt(0) || '?' }}
          </span>
        </div>
        <div class="flex-1 min-w-0 leading-none">
          <div
            :class="[
              'text-[12px] font-500 text-[#000000] truncate leading-[20px]',
              statusConfig.badgeIconColor ? 'mr-[20px]' : '',
            ]"
            :title="data.data?.name"
          >
            {{ data.data?.name }}
          </div>
          <div class="text-[12px] text-[#8E9BB3] leading-[20px]">
            {{ KIND_SHORT_MAP[data.data?.kind || ''] ?? data.data?.kind }}
          </div>
        </div>
        <Popover
          v-if="statusConfig.badgeIconColor"
          ext-cls="!py-[4px] !px-[8px]"
          max-width="500"
          placement="top"
        >
          <i
            class="absolute right-[8px] top-[50%] translate-y-[-60%] text-[16px] bkms-icon bkms-icon-info-circle-shape"
            :style="{ color: statusConfig.badgeIconColor }"
          />
          <template #content>
            {{
              data.data?.reason
                ? `${data.data?.status ?? 'unknown'}：${data.data.reason}`
                : `${data.data?.status ?? 'unknown'}`
            }}
          </template>
        </Popover>
        <template v-if="data.data?.hasChildren">
          <svg
            v-if="data.data?.collapsed"
            class="absolute top-[50%] left-full pointer-events-none z-[-1]"
            :style="{ width: `${EDGE_LENGTH / 2}px`, height: '2px' }"
          >
            <line
              stroke="#ABB5CC"
              stroke-width="2"
              x1="0"
              :x2="EDGE_LENGTH / 2"
              y1="1"
              y2="1"
            />
          </svg>
          <span
            class="collapse-btn absolute top-[50%] translate-y-[-50%] select-none bg-[#F5F7FA] rounded-[50%] flex items-center justify-center w-[18px] h-[18px]"
            :style="{ left: `calc(100% + ${EDGE_LENGTH / 2 - 9}px)` }"
            @click.stop.prevent="handleCollapse"
            @pointerdown.stop
            @pointerenter.stop
          >
            <i
              class="text-[#ABB5CC] text-[18px] hover:text-[#3A84FF] custom-collapse"
              :class="data.data?.collapsed ? 'bkms-icon bkms-icon-plus' : 'bkms-icon bkms-icon-minus'"
            ></i>
          </span>
        </template>
      </div>
    </template>
    <div
      v-else
      class="bg-[#fff] rounded-[10px] shadow-[0_4px_8px_0_rgba(41,45,51,0.08)]"
      :style="{
        width: `${NODE_WIDTH}px`,
        height: `${NODE_HEIGHT}px`,
      }"
    ></div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, useTemplateRef } from 'vue';

  import { useIntersectionObserver } from '@vueuse/core';
  import { Popover } from 'bkui-vue';

  import {
    EDGE_LENGTH,
    KIND_ICON_MAP,
    KIND_SHORT_MAP,
    MINIMAP_NODE_MAX,
    NODE_HEIGHT,
    NODE_SCALE_FACTOR,
    NODE_WIDTH,
    normalizeStatus,
    STATUS_CONFIG,
  } from './constants';

  import type { TopoNodeData } from './types';

  // 节点属性定义
  const props = withDefaults(
    defineProps<{
      data: {
        data: TopoNodeData;
        /** 节点 ID（G6 内置节点 ID） */
        id: string;
        /** 节点状态 (G6 内置的状态) */
        // https://g6.antv.antgroup.com/manual/element/state
        states?: string[];
      };
      // 节点总数
      nodeCount: number;
      /** 超采样缩放因子，默认取 constants 中定义的值 */
      scaleFactor?: number;
    }>(),
    {
      scaleFactor: NODE_SCALE_FACTOR,
    },
  );

  const emit = defineEmits<{ (e: 'toggleCollapse', nodeId: string): void }>();

  // 状态优先级配置：[状态名, 对应样式]
  const stateOutlineMap = [
    ['focused', '4px solid #FF9C01'],
    ['highlighted', '4px solid #FFE8C3'],
    ['selected', '4px solid #FF9C01'],
  ] as const;

  const statusConfig = computed(
    () => STATUS_CONFIG[props.data.data?.nodeStatus ?? normalizeStatus(props.data.data?.status ?? 'unknown')],
  );

  const nodeStyle = computed(() => ({
    background: statusConfig.value.bgColor,
    width: `${NODE_WIDTH}px`,
    height: `${NODE_HEIGHT}px`,
    outline:
      props.data.data?.kind === 'App'
        ? ''
        : (stateOutlineMap.find(([state]) => props.data.states?.includes(state))?.[1] ?? ''),
  }));

  function handleCollapse() {
    emit('toggleCollapse', props.data.id);
  }

  const nodeRef = useTemplateRef<HTMLElement>('nodeRef');
  // 节点是否在视口内
  const isIntersecting = ref(false);

  useIntersectionObserver(
    nodeRef,
    ([entry], _observerElement) => {
      // entry 对应的就是原生的 IntersectionObserverEntry
      isIntersecting.value = entry.isIntersecting;
    },
    {
      // 指定容器
      root: document.getElementById('topology-graph')?.getElementsByTagName('div')[0],
      // 💡 核心配置：设置缓冲区
      // 表示将视口的判定边界向外扩大多少像素
      rootMargin: '200px',
    },
  );
</script>

<style scoped>
  .svg-icon :deep(svg) {
    width: 100%;
    height: 100%;
  }

  /* hover 时改变背景，但排除折叠按钮 hover 的情况 */
  .topo-node:hover:not(:has(.collapse-btn:hover)) {
    background: linear-gradient(90deg, #f5f8ff 0%, #fcfdff 100%) !important;
  }
</style>
