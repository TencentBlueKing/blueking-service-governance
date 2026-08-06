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
    class="flex items-center px-[4px] h-[32px] bg-[#565E7A] rounded-[4px] shadow-[0_4px_12px_0_rgba(41,45,51,0.08)] z-10 select-none"
  >
    <!-- 1:1 -->
    <span
      v-bk-tooltips="$t('恢复100%')"
      class="flex items-center justify-center w-[24px] h-[24px] cursor-pointer rounded-[2px] hover:bg-[#41475C]"
      @click="zoomToOriginal"
    >
      <i class="bkms-icon bkms-icon-reset-1_1 text-[#F5F7FA] text-[16px]" />
    </span>

    <Divider
      class="h-[50%]"
      color="#717C8F"
      direction="vertical"
      type="solid"
    />

    <!-- 缩小 -->
    <span
      v-bk-tooltips="$t('缩小')"
      class="flex items-center justify-center w-[24px] h-[24px] cursor-pointer rounded-[2px] hover:bg-[#41475C]"
      @click="zoomOut"
    >
      <i class="bkms-icon bkms-icon-jianhao text-[#F5F7FA] text-[16px]" />
    </span>

    <!-- 缩放比例 -->
    <span class="min-w-[48px] text-center text-[13px] text-white tabular-nums">{{ zoomPercent }}%</span>

    <!-- 放大 -->
    <span
      v-bk-tooltips="$t('放大')"
      class="flex items-center justify-center w-[24px] h-[24px] cursor-pointer rounded-[2px] hover:bg-[#41475C]"
      @click="zoomIn"
    >
      <i class="bkms-icon bkms-icon-jiahao text-[#F5F7FA] text-[16px]" />
    </span>

    <Divider
      class="h-[50%]"
      color="#717C8F"
      direction="vertical"
      type="solid"
    />

    <!-- 缩略图 -->
    <span
      v-bk-tooltips="minimapDisabled ? $t('节点数量过多，已禁用缩略图') : $t('缩略图')"
      :class="[
        'flex items-center justify-center w-[24px] h-[24px] cursor-pointer rounded-[2px] hover:bg-[#41475C]',
        showMinimap ? '!bg-[#3A84FF]' : '',
        minimapDisabled ? 'opacity-50 cursor-not-allowed hover:bg-transparent' : '',
      ]"
      @click="!minimapDisabled && changeShowMinimap()"
    >
      <i class="bkms-icon bkms-icon-map text-[#F5F7FA] text-[16px]" />
    </span>

    <div
      class="custom-minimap !absolute top-[40px] left-0 rounded border-none overflow-hidden bg-[#e9eef5] shadow-[0_2px_10px_0_rgba(0,0,0,0.12)]"
      :style="{ display: showMinimap ? 'block' : 'none' }"
    ></div>
  </div>
</template>

<script lang="ts" setup>
  import { nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { Divider } from 'bkui-vue';
  import { isEmpty } from 'lodash-es';

  import type { Graph } from '@antv/g6';

  const props = defineProps<{
    graph: Graph | null;
    /** 节点数量超过阈值时禁用 minimap */
    minimapDisabled?: boolean;
  }>();

  // 是否展示缩略图, 默认不展示
  const showMinimap = ref(false);

  // 当 minimap 被禁用时，强制关闭 minimap
  watch(
    () => props.minimapDisabled,
    disabled => {
      if (disabled) {
        showMinimap.value = false;
      }
    },
  );

  const zoomPercent = ref(100);

  const syncZoomPercent = () => {
    nextTick(() => {
      if (isEmpty(props.graph)) return;
      const zoom = props.graph.getZoom() ?? 1;
      zoomPercent.value = Math.round(zoom * 100);
    });
  };

  // 监听 graph 实例变化，及时注册/注销事件
  // （子组件 onMounted 先于父组件执行，此时 graph 仍为 null，需要通过 watch 捕获赋值时机）
  let prevGraph: Graph | null = null;
  watch(
    () => props.graph,
    graph => {
      if (prevGraph) {
        prevGraph.off('aftertransform', syncZoomPercent);
        prevGraph.off('afterrender', syncZoomPercent);
      }
      if (graph) {
        graph.on('aftertransform', syncZoomPercent);
        graph.on('afterrender', syncZoomPercent);
        syncZoomPercent();
      }
      prevGraph = graph;
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    prevGraph?.off('aftertransform', syncZoomPercent);
    prevGraph?.off('afterrender', syncZoomPercent);
  });

  // 切换缩略图显示
  function changeShowMinimap() {
    showMinimap.value = !showMinimap.value;
  }

  function fitView() {
    props.graph?.fitView(undefined, { duration: 300 });
  }

  // 获取图中心点，用于缩放时的缩放中心
  function getZoomOrigin(): [number, number] | undefined {
    const graph = props.graph;
    if (!graph) return undefined;

    const nodes = graph.getNodeData();
    if (!nodes?.length) {
      return graph.getViewportByCanvas(graph.getViewportCenter()) as [number, number];
    }

    const visibleNodes = nodes.filter(node => graph.getElementVisibility(node.id!) !== 'hidden');
    const targetNodes = visibleNodes.length ? visibleNodes : nodes;

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;

    for (const node of targetNodes) {
      const { min, max } = graph.getElementRenderBounds(node.id!);
      minX = Math.min(minX, min[0]);
      maxX = Math.max(maxX, max[0]);
      minY = Math.min(minY, min[1]);
      maxY = Math.max(maxY, max[1]);
    }

    if (!Number.isFinite(minX) || !Number.isFinite(minY) || !Number.isFinite(maxX) || !Number.isFinite(maxY)) {
      return graph.getViewportByCanvas(graph.getViewportCenter()) as [number, number];
    }

    const centerCanvas: [number, number] = [(minX + maxX) / 2, (minY + maxY) / 2];
    return graph.getViewportByCanvas(centerCanvas) as [number, number];
  }

  function zoomIn() {
    const origin = getZoomOrigin();
    props.graph?.zoomBy(1.2, { duration: 200 }, origin);
  }

  function zoomOut() {
    const origin = getZoomOrigin();
    props.graph?.zoomBy(0.8, { duration: 200 }, origin);
  }

  async function zoomToOriginal() {
    await props.graph?.zoomTo(1, { duration: 200 });
    await props.graph?.fitCenter({ duration: 200 });
  }

  defineExpose({ fitView, zoomIn, zoomOut, zoomToOriginal });
</script>
