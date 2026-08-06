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
  <div class="flex items-center">
    <svg
      v-if="pending"
      class="size-[16px] mr-[4px]"
    >
      <use :xlink:href="`#bkms-icon-loading`"></use>
    </svg>
    <svg
      v-else
      class="size-[16px] mr-[4px]"
    >
      <use :xlink:href="`#bkms-icon-${statusClass}`"></use>
    </svg>
    <slot>
      <span
        v-bk-tooltips="{ content: message, disabled: !message }"
        :class="['flex-1 ellipsis', message ? 'border-b border-dashed border-[#979ba5] !flex-none' : '']"
      >
        {{ statusText }}
      </span>
    </slot>
  </div>
</template>
<script setup lang="ts">
  import type { PropType } from 'vue';
  import { computed, toRefs } from 'vue';

  const props = defineProps({
    pending: {
      type: Boolean,
      default: false,
    },
    status: {
      type: String,
      default: '',
    },
    statusTextMap: {
      type: Object,
      default: () => ({}),
    },
    // 每种状态对应的颜色, 默认黄色
    statusColorMap: {
      type: Object,
      default: () => ({
        running: 'green',
        completed: 'green',
        failed: 'red',
        FAILURE: 'red',
        terminating: 'blue',
        true: 'green',
        false: 'red',
        unknown: 'gray',
      }),
    },
    type: {
      type: String as PropType<'persistence' | 'result'>,
      default: 'persistence', // persistence 或 result
    },
    hideText: {
      type: Boolean,
      default: false,
    },
    message: {
      type: String,
      default: '',
    },
  });

  const { statusColorMap, statusTextMap, status, type, hideText } = toRefs(props);
  const svgEnums: { [key in string]: string } = {
    green: 'normal',
    red: 'abnormal',
    blue: 'status-unknown',
    gray: 'status-unknown',
    orange: 'warning-2',
  };

  const resultEnums: { [key in string]: string } = {
    green: 'success',
    red: 'failed',
    blue: 'default',
    gray: 'default',
    orange: 'waiting',
  };
  const color = computed(() => statusColorMap.value[status.value.toLowerCase()] || statusColorMap.value[status.value]);
  const statusClass = computed(() =>
    type.value === 'persistence'
      ? svgEnums[color.value] || svgEnums.orange
      : resultEnums[color.value] || resultEnums.orange,
  );
  const statusText = computed(() => {
    if (hideText.value) return '';
    return statusTextMap.value[status.value] || status.value || '--';
  });
</script>
