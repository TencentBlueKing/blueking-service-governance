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
    :class="[
      'p-[12px] border border-[#DCDEE5] rounded-[2px] text-[#4D4F56] hover:bg-[#F0F5FF] hover:border-[#3A84FF] group',
      { 'bg-[#FAFBFD] border-[#DCDEE5]': disabled },
      active ? '!border-[#3A84FF]' : 'border-[#C4C6CC]',
    ]"
  >
    <FlexRow class="mb-[4px]">
      <template #left>
        <span class="font-bold text-[#000000]">{{ curCom?.displayName || curCom?.name || '--' }}</span>
      </template>
      <template #right>
        <span
          v-bk-tooltips="{
            content: disabledText,
            disabled: !disabled,
          }"
        >
          <Button
            :class="{ 'bg-[#fff]': !disabled }"
            :disabled="active || disabled"
            size="small"
            @click="handleClick"
          >
            {{ active ? $t('已选') : $t('选择') }}
          </Button>
        </span>
      </template>
    </FlexRow>
    <div class="flex">
      <span class="text-[#979BA5] mr-[4px] shrink-0">{{ $t('说明') }} : </span>
      <div class="text-[#313238]">{{ curCom?.description || '--' }}</div>
    </div>
  </div>
</template>
<script setup lang="ts">
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { ComponentDefOutputObj } from '~/@types/v1/component-defs';

  interface Emits {
    (e: 'selected', value: ComponentDefOutputObj): void;
  }
  type IProps = {
    active?: boolean;
    data: ComponentDefOutputObj;
    disabled?: boolean;
    disabledText?: string;
  };
  const props = defineProps<IProps>();

  const emits = defineEmits<Emits>();

  const curCom = computed(() => props.data || {});

  function handleClick() {
    if (props.disabled) return;
    emits('selected', props.data || {});
  }
</script>
