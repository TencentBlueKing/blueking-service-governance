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
    <template v-if="!isShow">
      <div class="flex-1 font-mono tracking-wider">{{ placeholder }}</div>
      <Eye
        class="ml-[4px] cursor-pointer hover:text-[#3A84FF]"
        @click="handleToggle"
      />
    </template>
    <template v-else>
      <span class="text-[#313238]">{{ value || emptyValuePlaceholder }}</span>
      <Unvisible
        class="ml-[4px] cursor-pointer hover:text-[#3A84FF]"
        @click="handleToggle"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Eye, Unvisible } from 'bkui-vue/lib/icon';

  interface IProps {
    emptyValuePlaceholder?: string;
    placeholder?: string;
    showValue?: boolean;
    value: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    showValue: false,
    placeholder: '********',
    emptyValuePlaceholder: '--',
  });
  const emit = defineEmits(['toggle']);

  const isShow = ref(false);

  const handleToggle = () => {
    const target = !isShow.value;
    isShow.value = target;
    emit('toggle', target);
  };

  watch(
    () => props.showValue,
    newVal => {
      isShow.value = newVal;
    },
  );
</script>
