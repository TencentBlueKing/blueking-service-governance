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
    class="flex items-center border-[1px] border-[#C4C6CC] border-solid h-[32px] leading-[32px] text-[#4D4F56] bg-[#fff] min-w-[160px]"
  >
    <div
      class="text-center bg-[#FAFBFD] text-center h-full border-r-[1px] border-solid border-[#C4C6CC]"
      :style="`min-width: ${keyWidth ? keyWidth : 0}px`"
    >
      {{ keyName }}
    </div>
    <div class="flex items-center min-w-[100px]">
      <slot></slot>
    </div>
  </div>
</template>

<script lang="ts" setup>
  interface IProps {
    keyName: string;
    keyWidth: number;
  }

  defineProps<IProps>();
</script>
