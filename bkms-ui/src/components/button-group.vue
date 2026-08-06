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
  <div class="bk-button-group">
    <Button
      v-for="item in list"
      :key="item.value"
      class="text-[12px]"
      :class="modelValue === item.value ? 'is-selected' : ''"
      @click="modelValue = item.value"
    >
      {{ item.label }}
      <slot
        :data="item"
        name="suffix"
      ></slot>
    </Button>
  </div>
</template>

<script lang="ts" setup>
  import { Button } from 'bkui-vue';

  interface IProps {
    list: {
      label: string;
      value: number | string;
    }[];
  }
  const modelValue = defineModel<number | string>('modelValue');
  defineProps<IProps>();
</script>

<style lang="postcss" scoped>
  .bk-button-group {
    .is-selected {
      background-color: #3a84ff !important;
      color: #ffffff !important;
    }
  }
</style>
