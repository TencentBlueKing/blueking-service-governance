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
  <div class="text-[12px] flex w-full h-[30px]">
    <div
      class="text-[#979BA5] text-right shrink-0"
      :style="{
        minWidth: labelStyle,
      }"
    >
      {{ label }}：
    </div>
    <div class="text-[#313238] text-wrap break-all">
      <slot>{{ value ?? '--' }}</slot>
    </div>
  </div>
</template>
<script setup lang="ts">
  import { computed } from 'vue';

  const props = defineProps({
    label: {
      type: String,
      default: '',
    },
    labelWidth: {
      type: [String, Number],
      default: 100,
    },
    value: {
      type: String,
      default: '',
    },
  });

  const labelStyle = computed(() =>
    typeof props.labelWidth === 'string' ? props.labelWidth : `${props.labelWidth}px`,
  );
</script>
