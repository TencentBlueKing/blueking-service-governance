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
  <div class="flex flex-col gap-[16px]">
    <template v-if="label">
      <div class="flex items-center gap-[5px]">
        <span class="text-[#313238] font-700 label">{{ label }}</span>
        <InfoLine
          v-if="showIcon"
          fill="#979BA5"
          :height="font"
          :width="font"
        />
      </div>
    </template>
    <template v-else-if="slots.label">
      <slot name="label" />
    </template>
    <slot />
  </div>
</template>

<script setup lang="ts">
  import { computed, useSlots } from 'vue';

  import { InfoLine } from 'bkui-vue/lib/icon';

  const props = defineProps({
    label: {
      type: String,
      default: '',
    },
    showIcon: {
      type: Boolean,
      default: false,
    },
    font: {
      type: String,
      default: '14',
    },
  });

  const fontSize = computed(() => `${props.font}px`);

  const slots = useSlots();
</script>

<style lang="postcss" scoped>
  .label {
    font-size: v-bind(fontSize);
  }
</style>
