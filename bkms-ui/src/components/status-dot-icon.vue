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
  <svg
    v-if="colors"
    aria-hidden="true"
    class="mr-[4px] shrink-0"
    :style="iconSize"
    viewBox="0 0 12 12"
  >
    <circle
      cx="6"
      cy="6"
      :fill="colors.outer"
      r="6"
    />
    <circle
      cx="6"
      cy="6"
      :fill="colors.inner"
      r="3.23"
    />
  </svg>
  <ColorIcon
    v-else
    class="mr-[4px] shrink-0"
    :icon="icon"
    :size="size"
  />
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import ColorIcon from '~/components/color-icon.vue';

  interface Props {
    icon: string;
    size?: number;
  }

  const props = withDefaults(defineProps<Props>(), {
    size: 12,
  });

  const colorMap: Record<string, { inner: string; outer: string }> = {
    normal: { inner: '#3FC06D', outer: '#E5F6EA' },
    abnormal: { inner: '#EA3636', outer: '#FFE6E6' },
  };

  const colors = computed(() => colorMap[props.icon]);
  const iconSize = computed(() => ({
    height: `${props.size}px`,
    width: `${props.size}px`,
  }));
</script>
