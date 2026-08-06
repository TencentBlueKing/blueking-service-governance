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
    :disabled="disabledAdd"
    text
    @click="handleAdd"
  >
    <div :class="['flex items-center', disabledAdd ? 'text-[#C4C6CC]' : 'text-[#3A84FF]']">
      <span class="bkms-icon bkms-icon-plus-circle-shape text-[14px]"></span>
      <span class="text-[12px] ml-[6px]">{{ $t('添加') }}</span>
    </div>
  </Button>
  <div
    v-for="(item, index) in value"
    :key="`${index}-${item.key}`"
    class="flex items-center mt-[10px]"
  >
    <Input
      v-model.trim="item.key"
      class="w-[136px]"
      placeholder="key"
    />
    <div
      :class="[
        'size-[32px] flex justify-center items-center text-[#979BA5] text-[12px]',
        'border-1 border-[#c4c6cc] mx-[8px] radius-[2px]',
      ]"
    >
      =
    </div>
    <Input
      v-model.trim="item.value"
      class="w-[136px]"
      placeholder="value"
    />
    <Button
      text
      @click="handleDel(index)"
    >
      <Del
        class="ml-[8px]"
        fill="#979BA5"
        height="14px"
        width="14px"
      ></Del>
    </Button>
  </div>
</template>
<script setup lang="ts">
  import type { PropType } from 'vue';
  import { computed, ref, watch } from 'vue';

  import { Button, Input } from 'bkui-vue';
  import { Del } from 'bkui-vue/lib/icon';

  const prop = defineProps({
    modelValue: {
      type: Array as PropType<
        {
          key: string;
          value: string;
        }[]
      >,
      default: () => [],
    },
    max: {
      type: Number,
      default: 10,
    },
  });
  const emits = defineEmits(['update:modelValue']);

  const value = ref(prop.modelValue);
  const disabledAdd = computed(() => value.value.length >= prop.max);
  function handleAdd() {
    if (value.value.length >= prop.max) {
      return;
    }
    value.value.push({ key: '', value: '' });
  }
  function handleDel(index: number) {
    value.value.splice(index, 1);
  }

  watch(
    value,
    val => {
      emits('update:modelValue', val);
    },
    { deep: true },
  );
</script>
