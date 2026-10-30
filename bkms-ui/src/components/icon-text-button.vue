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
  <Button
    :class="['!px-[8px] !py-[6px] select-none', { '!bg-[#E5EDFF] rounded-[6px]': !!active }]"
    :disabled="disabled"
    text
    theme="primary"
    @click="handleClick"
  >
    <i
      v-if="icon"
      :class="['text-[#3B6EF5] text-[14px] mr-[4px]', icon, { 'text-[#E4E9F2] cursor-not-allowed': disabled }]"
    ></i>
    <span class="text-[12px]">{{ text }}</span>
  </Button>
</template>
<script lang="ts" setup>
  import { Button } from 'bkui-vue';

  interface IEmit {
    (e: 'click'): void;
  }
  interface IProps {
    active?: boolean;
    disabled?: boolean;
    icon?: string;
    text: string;
  }
  defineProps<IProps>();
  const emits = defineEmits<IEmit>();

  function handleClick() {
    emits('click');
  }
</script>
