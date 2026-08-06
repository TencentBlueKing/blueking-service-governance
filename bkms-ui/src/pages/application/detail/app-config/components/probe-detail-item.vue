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
    class="text-[12px] flex items-center"
    :style="{ lineHeight: resolvedLineHeight }"
  >
    <span class="text-[#979BA5]">{{ label }}：</span>
    <span class="text-[#313238]">
      <slot>{{ value ?? '--' }}</slot>
    </span>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  const props = withDefaults(
    defineProps<{
      label: string;
      lineHeight?: number | string;
      value?: number | string;
    }>(),
    {
      lineHeight: 32,
    },
  );

  const resolvedLineHeight = computed(() =>
    typeof props.lineHeight === 'number' ? `${props.lineHeight}px` : props.lineHeight,
  );
</script>

<style lang="postcss" scoped>
  .probe-detail-label {
    color: var(--probe-detail-label-color, #979ba5);
  }

  .probe-detail-value {
    color: var(--probe-detail-value-color, #313238);
  }
</style>
