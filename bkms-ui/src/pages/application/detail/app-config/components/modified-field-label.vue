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
    class="flex items-center justify-end gap-[6px] text-[#979BA5]"
    :style="{ width: `${labelWidth}px`, lineHeight: `${lineHeight}px` }"
  >
    <i
      v-if="modified"
      class="w-[3px] h-[18px] bg-[#ff9c01]"
    ></i>
    {{ label }}：
  </div>
</template>

<script lang="ts" setup>
  interface Props {
    /** 字段名文本 */
    label: string;
    /** label 宽度，默认 205px */
    labelWidth?: number;
    /** 行高，默认 30px */
    lineHeight?: number;
    /** 是否显示修改标识 */
    modified?: boolean;
  }

  withDefaults(defineProps<Props>(), {
    labelWidth: 205,
    lineHeight: 30,
    modified: false,
  });
</script>
