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
  <ResetIcon
    v-if="showReset"
    class="absolute top-[-20px] right-0 z-[9]"
    @click="$emit('reset')"
  />
  <p
    v-if="tip"
    class="mt-[4px] text-[12px] text-[#979BA5]"
  >
    {{ tip }}
  </p>
</template>

<script setup lang="ts">
  import ResetIcon from './reset-icon.vue';

  defineProps<{
    /** 是否显示重置图标 */
    showReset: boolean;
    /** 默认值提示文字（空字符串时不显示） */
    tip: string;
  }>();

  defineEmits<{
    reset: [];
  }>();
</script>
