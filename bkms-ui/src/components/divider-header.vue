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
    <slot name="title">
      <span
        class="text-[#313238] mr-[6px]"
        :style="{ fontSize: parseCss(props.titleSize) }"
        >{{ title || '--' }}</span
      >
    </slot>
    <Divider
      v-if="showDivider"
      class="h-[14px] text-[12px]"
      color="#DCDEE5"
      direction="vertical"
      type="solid"
    />
    <span class="flex items-center gap-[20px] text-[14px] text-[#979BA5] flex-1 overflow-hidden">
      <slot></slot>
    </span>
  </div>
</template>
<script lang="ts" setup>
  import { Divider } from 'bkui-vue';
  import { parseCss } from '~/common/util';
  interface IProps {
    showDivider?: boolean;
    title?: string;
    titleSize?: number | string;
  }
  const props = withDefaults(defineProps<IProps>(), {
    showDivider: true,
    titleSize: 20,
  });
</script>
