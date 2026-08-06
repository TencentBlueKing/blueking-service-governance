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
  <div :class="['user-selector-wrapper', { 'cursor-not-allowed': attrs.disabled }]">
    <BkUserSelector
      v-model="internalValue"
      :api-base-url="bkUserApiUrl"
      :current-user-id="currentUserId"
      :tenant-id="tenantId"
      v-bind="$attrs"
    />
    <Close
      v-if="showClearBtn"
      class="user-selector-clear-btn"
      @click="handleClear"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, useAttrs } from 'vue';

  import BkUserSelector from '@blueking/bk-user-selector';
  import { Close } from 'bkui-vue/lib/icon';

  import '@blueking/bk-user-selector/vue3/vue3.css';

  defineOptions({ inheritAttrs: false });

  interface Props {
    /** 是否显示清空按钮，仅在多选模式下生效 */
    clearable?: boolean;
    /** 快捷选中"我"时需要填写 */
    currentUserId?: string;
    modelValue?: string[];
    // 租户 ID
    tenantId?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: () => [],
    currentUserId: '',
    tenantId: 'default',
    clearable: false,
  });

  /** API 基础地址：开发走 Vite 代理，生产拼完整 URL */
  const bkUserApiUrl = (() => {
    if (import.meta.env.DEV) return '/api-bk-user-selector';
    const template = import.meta.env.BK_API_URL_TMPL || '';
    if (!template) return '';
    return template.includes('{api_name}') ? `${template.replace('{api_name}', 'bk-user-web')}/prod` : template;
  })();

  const emit = defineEmits<{
    'update:modelValue': [value: string[]];
  }>();

  const attrs = useAttrs();

  /** 判断是否为多选模式 */
  const isMultiple = () => (attrs as Record<string, unknown>).multiple !== false;

  /** 是否显示清空按钮：clearable 开启且处于多选模式且有已选值 */
  const showClearBtn = computed(
    () => props.clearable && !attrs.disabled && isMultiple() && (props.modelValue ?? []).length > 0,
  );

  /** 清空已选用户 */
  const handleClear = () => {
    emit('update:modelValue', []);
  };

  /**
   * 处理单选/多选的数据格式转换
   * 统一对外暴露数组格式，内部根据模式转换
   */
  const internalValue = computed({
    get: () => {
      const value = props.modelValue ?? [];
      // 单选模式：返回第一个元素/多选模式：返回数组
      return !isMultiple() && value.length > 0 ? value[0] : value;
    },
    set: (value: string | string[]) => {
      // 统一转换为数组格式后 emit
      const arrayValue = Array.isArray(value) ? value : value ? [value] : [];
      emit('update:modelValue', arrayValue);
    },
  });
</script>
<style lang="postcss" scoped>
  .bk-user-selector {
    height: unset !important;
  }

  .user-selector-wrapper {
    position: relative;
  }

  :deep(.tags-container.focused) {
    position: relative;
    z-index: 999;
    max-height: 250px;
    overflow-y: auto;
  }

  .user-selector-clear-btn {
    position: absolute;
    top: 50%;
    right: 6px;
    z-index: 9999;
    transform: translateY(-50%);
    color: #c4c6cc;
    opacity: 0;
    cursor: pointer;
    transition: opacity 0.2s ease;
  }

  .user-selector-wrapper:hover .user-selector-clear-btn {
    opacity: 1;
  }

  .user-selector-clear-btn:hover {
    color: #979ba5;
  }
</style>
