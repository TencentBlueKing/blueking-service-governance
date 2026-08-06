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
  <div class="password-input-wrapper">
    <!-- 已有密码（编辑态入口）：6个圆点占位，点击进入编辑 -->
    <div
      v-if="!isEditing && isEditMode && props.hasCredential"
      class="flex items-center h-[32px] px-[10px] border border-solid border-[#c4c6cc] rounded-[2px] cursor-text text-[14px] text-[#63656e] bg-white leading-[30px] hover:border-[#979ba5]"
      @click="handlePlaceholderClick"
    >
      <span class="text-[#63656e]">••••••</span>
    </div>
    <!-- 编辑态/创建态：真实输入框 -->
    <Input
      v-else
      ref="inputRef"
      v-model.trim="localValue"
      :placeholder="$t('请输入密码')"
      type="password"
    >
      <template #suffix>
        <div class="flex items-center pr-[8px]">
          <i
            v-if="isEditMode"
            class="bkms-icon bkms-icon-redo cursor-pointer text-[#c4c6cc] text-[16px] hover:text-[#313238]"
            @click="handleReset"
          />
        </div>
      </template>
    </Input>
  </div>
</template>

<script lang="ts" setup>
  import { nextTick, ref, watch } from 'vue';

  import { Input } from 'bkui-vue';

  /** 密码值，与父组件双向绑定 */
  const modelValue = defineModel<string>({ default: '' });
  /** 密码是否已被修改，与父组件双向绑定 */
  const modified = defineModel<boolean>('modified', { default: false });

  /** 是否为编辑模式（编辑场景下接口不返回密码，需由父组件显式告知） */
  const props = withDefaults(defineProps<{ hasCredential?: boolean; isEditMode?: boolean }>(), {
    isEditMode: false,
    hasCredential: true,
  });

  /** 是否处于编辑态（显示真实输入框） */
  const isEditing = ref(false);
  /** 输入框内的本地值 */
  const localValue = ref('');
  /** 输入框组件实例引用 */
  const inputRef = ref<InstanceType<typeof Input>>();
  /** 标记是否为内部更新，用于避免 watch 循环 */
  const isInternalUpdate = ref(false);
  /** 标记父组件同步/重置等程序性赋值，避免误判为用户修改密码 */
  const isProgrammaticUpdate = ref(false);

  /** 点击占位符区域，切换为编辑态并聚焦输入框 */
  const handlePlaceholderClick = () => {
    isEditing.value = true;
    localValue.value = '';
    modified.value = true;
    nextTick(() => {
      inputRef.value?.focus();
    });
  };

  /** 点击重置按钮：清空密码并退回圆点占位状态（仅编辑态可用） */
  const handleReset = () => {
    isProgrammaticUpdate.value = true;
    isEditing.value = false;
    localValue.value = '';
    modelValue.value = '';
    modified.value = false;
    nextTick(() => {
      isProgrammaticUpdate.value = false;
    });
  };

  /** 监听本地值变化，同步到父组件并标记已修改 */
  watch(localValue, val => {
    isInternalUpdate.value = true;
    modelValue.value = val;
    if (!isProgrammaticUpdate.value) {
      modified.value = true;
    }
    nextTick(() => {
      isInternalUpdate.value = false;
    });
  });

  /** 监听父组件值变化，外部设置值时重置为占位符状态 */
  watch(modelValue, val => {
    // 跳过内部更新触发的变化，避免循环
    if (isInternalUpdate.value) return;

    if (val !== localValue.value) {
      isProgrammaticUpdate.value = true;
      localValue.value = val;
      isEditing.value = false;
      modified.value = false;
      nextTick(() => {
        isProgrammaticUpdate.value = false;
      });
    }
  });
</script>
