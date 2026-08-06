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
  <Teleport to="body">
    <div
      v-if="visible"
      ref="menuRef"
      class="fixed min-w-[120px] bg-white rounded-[4px] shadow-[0_4px_12px_0_rgba(0,0,0,0.14)] py-[4px] z-[9999]"
      :style="menuStyle"
      @contextmenu.prevent
    >
      <div
        v-for="item in menuItems"
        :key="item.id"
        v-bk-tooltips="{
          content: item.tip,
          disabled: !item.disabled,
          placement: 'left',
        }"
        class="flex items-center gap-[6px] px-[16px] py-[6px] text-[12px] text-[#63656E] cursor-pointer whitespace-nowrap hover:bg-[#F0F1F5] hover:text-[#3A84FF]"
        :class="{
          'text-[#C4C6CC]! cursor-not-allowed! hover:bg-transparent! hover:text-[#C4C6CC]!': item.disabled,
        }"
        @click="handleItemClick(item)"
      >
        <span>{{ item.label }}</span>
      </div>
      <div
        v-if="!menuItems.length"
        class="px-[16px] py-[8px] text-[12px] text-[#C4C6CC] text-center"
      >
        {{ $t('暂无操作') }}
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

  import type { ContextMenuItem } from './types';

  const props = defineProps<{
    menuItems: ContextMenuItem[];
    nodeId: string;
    visible: boolean;
    x: number;
    y: number;
  }>();

  const emit = defineEmits<{
    close: [];
    'menu-click': [action: string, nodeId: string];
  }>();

  const menuRef = ref<HTMLElement | null>(null);

  const menuStyle = computed(() => ({
    left: `${props.x}px`,
    top: `${props.y}px`,
  }));

  function handleClickOutside(e: MouseEvent) {
    if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
      emit('close');
    }
  }

  function handleItemClick(item: ContextMenuItem) {
    if (item.disabled) return;
    emit('menu-click', item.id, props.nodeId);
    emit('close');
  }

  watch(
    () => props.visible,
    val => {
      if (val) {
        setTimeout(() => document.addEventListener('click', handleClickOutside), 0);
      } else {
        document.removeEventListener('click', handleClickOutside);
      }
    },
  );

  onMounted(() => {
    if (props.visible) {
      document.addEventListener('click', handleClickOutside);
    }
  });

  onBeforeUnmount(() => {
    document.removeEventListener('click', handleClickOutside);
  });
</script>
