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
  <div class="group border rounded-[2px] text-[#4D4F56] hover:border-[#1562FC]">
    <div
      :class="[
        'bg-[#FAFBFD] px-[16px] py-[8px] border-b leading-[22px] flex justify-between items-center relative',
        headerClass,
      ]"
    >
      <div class="flex-1 whitespace-nowrap overflow-hidden text-ellipsis">
        <span class="font-bold">{{ data?.type || '--' }}</span>
        <span
          v-if="data?.name"
          class="text-[#979aa3]"
        >
          ({{ data.name }})</span
        >
      </div>
      <div class="group-hover:w-auto w-0 overflow-hidden transition flex items-center">
        <Button
          v-if="visibleButtonList.includes('edit')"
          v-bk-tooltips="{ content: $t('编辑') }"
          class="hover:text-[#3a84ff]"
          text
          @click="handleEdit"
        >
          <EditLine />
        </Button>
        <Button
          v-if="visibleButtonList.includes('clone')"
          v-bk-tooltips="{ content: cloneDesc || $t('克隆') }"
          class="ml-[16px]"
          :disabled="disabledClone"
          text
          @click="handleClone"
        >
          <i :class="['bkms-icon bkms-icon-clone', { 'text-[#979BA5] hover:text-[#3a84ff]': !disabledClone }]"></i>
        </Button>
        <Button
          v-if="visibleButtonList.includes('delete')"
          v-bk-tooltips="{ content: delDesc || $t('删除') }"
          :class="['ml-[16px]', { 'hover:text-[#ea3636]': !disabledDelete }]"
          :disabled="disabledDelete"
          text
          @click="handleDel"
        >
          <Del />
        </Button>
      </div>
    </div>
    <div :class="contentClass">
      <div class="mx-[22px] my-[12px] h-[94px] overflow-hidden box-border text-[12px]">
        <div class="h-[24px] flex items-center">
          <span class="mr-[5px]">{{ $t('版本号') }}: </span>
          <bk-tag
            v-if="data?.version"
            class="text-[10px] h-[16px]"
            >{{ data.version }}</bk-tag
          >
        </div>
        <div
          v-for="(item, index) in propertyList"
          :key="`${item.key}${index}`"
          class="leading-[24px] flex items-center"
        >
          <span class="mr-[4px]">{{ item.key }}:</span>
          <span class="ellipsis">{{ item.value }}</span>
        </div>
        <div
          v-if="showEllipsis"
          class="leading-[24px] font-bold"
        >
          <span>...</span>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
  import type { PropType } from 'vue';
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { Del, EditLine } from 'bkui-vue/lib/icon';
  import { Component as ComponentDefinition } from '~/@types/app';

  const props = defineProps({
    disabledDelete: {
      type: Boolean,
      default: false,
    },
    delDesc: {
      type: String,
      default: '',
    },
    data: {
      type: Object as PropType<ComponentDefinition>,
      default: () => ({}),
    },
    headerClass: {
      type: String,
      default: '',
    },
    contentClass: {
      type: String,
      default: '',
    },
    disabledClone: {
      type: Boolean,
      default: false,
    },
    cloneDesc: {
      type: String,
      default: '',
    },
    visibleButtonList: {
      type: Array as PropType<Array<'clone' | 'delete' | 'edit'>>,
      default: () => ['clone', 'delete', 'edit'],
    },
  });

  const emit = defineEmits(['edit', 'del', 'clone']);

  const maxShowProperties = 3; // 最多显示的属性数量
  const showEllipsis = computed(() => Object.keys(props.data?.properties || {}).length > maxShowProperties);
  const propertyList = computed(() => {
    const list = Object.keys(props.data?.properties || {}).map(key => ({ key, value: props.data?.properties[key] }));
    return list.slice(0, maxShowProperties);
  });

  /**
   * 克隆
   */
  function handleClone() {
    emit('clone', props.data);
  }

  /**
   * 删除
   */
  function handleDel() {
    emit('del', props.data);
  }

  /**
   * 编辑
   */
  function handleEdit() {
    emit('edit');
  }
</script>
