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
  <div class="flex items-center gap-[36px] w-full">
    <!-- 环形进度条 -->
    <div
      class="relative flex-shrink-0"
      :style="{
        width: `${size}px`,
        height: `${size}px`,
      }"
    >
      <svg
        class="w-full h-full -rotate-90"
        :viewBox="`0 0 ${center * 2} ${center * 2}`"
      >
        <!-- 背景圆环 -->
        <circle
          :cx="center"
          :cy="center"
          fill="none"
          :r="circleRadius"
          stroke="#F0F1F5"
          :stroke-width="STROKE_WIDTH"
        />
        <!-- 致命问题圆环（红色） -->
        <circle
          :cx="center"
          :cy="center"
          fill="none"
          :r="circleRadius"
          stroke="#EA3636"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="riskDashOffset"
          :stroke-width="STROKE_WIDTH"
        />
        <!-- 预警问题圆环（橙色） -->
        <circle
          :cx="center"
          :cy="center"
          fill="none"
          :r="circleRadius"
          stroke="#F59500"
          :stroke-dasharray="warnDashArray"
          :stroke-dashoffset="0"
          :stroke-width="STROKE_WIDTH"
          :transform="`rotate(${riskPercentage * 360} ${center} ${center})`"
        />
      </svg>
      <!-- 中心文字 -->
      <div class="absolute inset-0 flex flex-col items-center justify-center">
        <div
          class="progress-value font-bold leading-none"
          :style="{
            color: count?.RISK != null ? '#EA3636' : '#979BA5',
            fontSize: `${valueSize}px`,
          }"
        >
          {{ count?.RISK ?? '--' }}
        </div>
        <div class="text-[10px] text-[#979BA5] mt-[4px]">
          {{ $t('致命问题') }}
        </div>
      </div>
    </div>

    <!-- 右侧图例 -->
    <div
      v-if="showLegend"
      class="flex flex-col gap-[12px]"
    >
      <div class="flex items-center gap-[8px]">
        <span class="inline-block w-[8px] h-[8px] rounded-full bg-[#EA3636]"></span>
        <span class="text-[12px] text-[#63656E]">{{ $t('致命问题') }}：{{ count?.RISK ?? '--' }}</span>
      </div>
      <div class="flex items-center gap-[8px]">
        <span class="inline-block w-[8px] h-[8px] rounded-full bg-[#F59500]"></span>
        <span class="text-[12px] text-[#63656E]">{{ $t('预警问题') }}：{{ count?.WARN ?? '--' }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  interface IProps {
    radius?: number; // 圆环半径（默认40px）
    showLegend?: boolean;
    valueSize?: number;
    count: {
      RISK?: number;
      WARN?: number;
    };
  }

  const props = withDefaults(defineProps<IProps>(), {
    radius: 40,
    valueSize: 28,
    showLegend: true,
  });

  // 固定配置
  const STROKE_WIDTH = 10;

  // 根据半径计算宽高（直径）
  const size = computed(() => props.radius * 2);

  // 动态计算圆心坐标
  const center = computed(() => props.radius);

  // 圆环实际半径（减去描边宽度的一半，确保描边不超出容器）
  const circleRadius = computed(() => props.radius - STROKE_WIDTH / 2);

  // 圆环周长
  const circumference = computed(() => 2 * Math.PI * circleRadius.value);

  // 致命问题占比
  const riskPercentage = computed(() => {
    const risk = props.count?.RISK ?? 0;
    const warn = props.count?.WARN ?? 0;
    const total = risk + warn;
    return total > 0 ? risk / total : 0;
  });

  // 预警问题占比
  const warnPercentage = computed(() => {
    const risk = props.count?.RISK ?? 0;
    const warn = props.count?.WARN ?? 0;
    const total = risk + warn;
    return total > 0 ? warn / total : 0;
  });

  // 致命问题圆环配置（从0度开始）
  const riskDashOffset = computed(() => {
    return circumference.value * (1 - riskPercentage.value);
  });

  // 预警问题圆环配置（从致命问题结束位置开始）
  const warnDashArray = computed(() => {
    // dasharray: 显示长度 空白长度
    const warnLength = circumference.value * warnPercentage.value;
    return `${warnLength} ${circumference.value}`;
  });
</script>

<style scoped>
  .progress-value {
    font-family: Arial-BoldMT, Arial, sans-serif;
  }
</style>
