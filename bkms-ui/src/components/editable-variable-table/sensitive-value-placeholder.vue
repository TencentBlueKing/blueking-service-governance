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
  <span class="flex items-center gap-[4px]">
    <!-- 加密占位 -->
    <span
      v-for="dot in count"
      :key="dot"
      class="inline-block w-[6px] h-[6px] rounded-full bg-[#63656e]"
    ></span>
  </span>
</template>

<script setup lang="ts">
  interface IProps {
    count?: number;
  }

  withDefaults(defineProps<IProps>(), {
    count: 6,
  });
</script>
