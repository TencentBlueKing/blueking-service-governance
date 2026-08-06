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
  <div class="w-full bkms-content rounded-[2px] overflow-visible">
    <div
      :class="[
        'flex items-center justify-between bg-[#F5F7FA] h-[32px] px-[16px] bkms-content-title',
        { 'cursor-pointer': collapsible },
      ]"
      @click.stop="collapsible && handleToggle()"
    >
      <div class="flex items-center text-[14px] font-bold text-[#4D4F56] leading-[32px]">
        <RightShape
          v-if="collapsible"
          :class="['transition duration-300 mr-[6px] mt-[-1px]', innerCollapsed ? '' : 'rotate-90']"
          fill="#979BA5"
          :height="14"
          :width="14"
        />
        <slot name="title">
          <span class="text-[#313238]">{{ title }}</span>
        </slot>
        <Button
          v-if="showEditIcon"
          v-bk-tooltips="editTooltip"
          class="ml-[10px]"
          :class="{ '!hover:text-[#3A84FF]': !editDisabled }"
          :disabled="editDisabled"
          text
          @click.stop="handleEdit"
        >
          <EditLine />
          <span class="text-[12px] font-400 mt-[1px]">{{ $t('编辑') }}</span>
        </Button>
      </div>
      <div>
        <slot name="action"></slot>
      </div>
    </div>
    <div
      v-show="!innerCollapsed || !collapsible"
      class="contents"
    >
      <slot></slot>
    </div>
  </div>
</template>
<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button } from 'bkui-vue';
  import { EditLine, RightShape } from 'bkui-vue/lib/icon';

  interface Props {
    collapsed?: boolean;
    collapsible?: boolean;
    editDisabled?: boolean;
    editDisabledTips?: string;
    showEditIcon?: boolean;
    title?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    collapsible: false,
    collapsed: false,
  });
  const emits = defineEmits(['edit', 'update:collapsed', 'collapse-change']);

  const innerCollapsed = ref(props.collapsed);

  watch(
    () => props.collapsed,
    val => {
      innerCollapsed.value = val;
    },
  );

  watch(innerCollapsed, val => {
    emits('update:collapsed', val);
    emits('collapse-change', val);
  });

  const editTooltip = computed(() => {
    if (props.editDisabled && props.editDisabledTips) {
      return props.editDisabledTips;
    }
    return { disabled: true };
  });

  const handleEdit = () => {
    emits('edit');
  };

  const handleToggle = () => {
    innerCollapsed.value = !innerCollapsed.value;
  };
</script>
