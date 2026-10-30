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
  <Dropdown
    ref="sortDropdownRef"
    placement="bottom-end"
    :popover-options="{ extCls: 'bkms-sort-popover' }"
    trigger="click"
    @hide="isMenuOpen = false"
    @show="isMenuOpen = true"
  >
    <section
      :class="[
        'bkms-sort-trigger inline-flex items-center justify-between h-[32px] min-w-[160px] pl-[12px] pr-[8px]',
        'bg-[#fff] border border-solid border-[#C1CDE5] rounded-[6px] text-[12px] text-[#4A5973] whitespace-nowrap cursor-pointer',
        'hover:border-[#8F9FBD]',
        { 'is-open': isMenuOpen },
      ]"
    >
      <span class="inline-flex items-center min-w-0">
        <span class="mr-[8px]">{{ currentOption?.label }}</span>
        <span
          class="bkms-sort-tag bkms-sort-tag--trigger"
          @click.stop="toggleFieldOrder(field)"
        >
          <svg
            :class="['bkms-sort-arrow', { 'is-desc': order === 'desc' }]"
            viewBox="0 0 1024 1024"
          >
            <path
              d="M144 512l70.4 70.4 214.4 214.4 70.4-72L336 561.6h544v-99.2H336l163.2-163.2-70.4-72-214.4 214.4z"
            />
          </svg>
          <span>{{ currentDirectionName }}</span>
        </span>
      </span>
      <svg
        :class="['bkms-sort-caret text-[#8F9FBD] transition-transform duration-200', { 'rotate-180': !isMenuOpen }]"
        viewBox="0 0 12 12"
        aria-hidden="true"
      >
        <path
          d="M2.25 7.75 6 4.25 9.75 7.75"
          fill="none"
          stroke="currentColor"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="1.5"
        />
      </svg>
    </section>
    <template #content>
      <Dropdown.DropdownMenu
        class="bkms-sort-menu"
        ext-cls="bkms-sort-menu"
      >
        <Dropdown.DropdownItem
          v-for="item in options"
          :key="item.value"
          :class="['bkms-sort-item', { 'is-active': field === item.value }]"
          @click="chooseSort(item.value, getItemOrder(item.value), true)"
        >
          <span class="flex-1 leading-[20px]">{{ item.label }}</span>
          <span
            :class="['bkms-sort-tag', { 'is-active': field === item.value }]"
            @click.stop="toggleFieldOrder(item.value)"
          >
            <svg
              :class="['bkms-sort-arrow is-menu', { 'is-desc': getItemOrder(item.value) === 'desc' }]"
              viewBox="0 0 1024 1024"
            >
              <path
                d="M144 512l70.4 70.4 214.4 214.4 70.4-72L336 561.6h544v-99.2H336l163.2-163.2-70.4-72-214.4 214.4z"
              />
            </svg>
            <span class="leading-[16px]">{{ getItemDirectionName(item) }}</span>
          </span>
        </Dropdown.DropdownItem>
      </Dropdown.DropdownMenu>
    </template>
  </Dropdown>
</template>

<script lang="ts" setup generic="T extends string">
  import { computed, nextTick, ref } from 'vue';

  import { Dropdown } from 'bkui-vue';

  export interface SortOption<T extends string = string> {
    ascName: string;
    descName: string;
    label: string;
    value: T;
  }

  const field = defineModel<T>({ required: true });
  const order = defineModel<'asc' | 'desc'>('order', { required: true });

  const props = defineProps<{
    options: SortOption<T>[];
  }>();

  const isMenuOpen = ref(false);
  const sortDropdownRef = ref<{ popoverRef?: { hide?: () => void } }>();

  const currentOption = computed(
    () => props.options.find(item => item.value === field.value) ?? props.options[0],
  );
  const currentDirectionName = computed(() => {
    if (!currentOption.value) return '';
    return order.value === 'asc' ? currentOption.value.ascName : currentOption.value.descName;
  });

  /** 未选中的字段在菜单里按升序展示，选中字段跟随当前方向 */
  function getItemOrder(value: T) {
    return field.value === value ? order.value : 'asc';
  }

  function getItemDirectionName(item: SortOption<T>) {
    return getItemOrder(item.value) === 'asc' ? item.ascName : item.descName;
  }

  function chooseSort(value: T, nextOrder: 'asc' | 'desc', shouldClose = false) {
    field.value = value;
    order.value = nextOrder;
    // 字段变化会重渲染触发器。等这次渲染结束后再关，避免关完又被打开。
    if (shouldClose) {
      nextTick(() => {
        sortDropdownRef.value?.popoverRef?.hide?.();
      });
    }
  }

  function toggleFieldOrder(value: T) {
    const nextOrder = field.value === value && order.value === 'asc' ? 'desc' : 'asc';
    chooseSort(value, nextOrder);
  }
</script>

<style lang="postcss" scoped>
  .bkms-sort-trigger.is-open {
    border-color: #3b6ef5;
  }

  .bkms-sort-tag {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    height: 20px;
    padding: 2px 8px;
    font-size: 10px;
    color: #4a5973;
    cursor: pointer;
    background: #f7f9fd;
    border: 1px solid #e4e9f2;
    border-radius: 6px;
  }

  .bkms-sort-tag:hover {
    background: #e4e9f2;
  }

  .bkms-sort-tag--trigger {
    padding: 0 6px;
    margin-right: 4px;
    color: #4a5973;
    background: #e9edf5;
    border-color: transparent;
  }

  .bkms-sort-tag--trigger:hover {
    background: #e4e9f2;
  }

  .bkms-sort-tag.is-active {
    color: #3b6ef5;
    background: #e5edff;
    border-color: transparent;
  }

  .bkms-sort-tag.is-active:hover {
    background: #99b9ff;
  }

  .bkms-sort-caret {
    flex-shrink: 0;
    width: 12px;
    height: 12px;
  }

  .bkms-sort-arrow {
    width: 12px;
    height: 12px;
    margin-right: 2px;
    fill: #8f9fbd;
    transform: rotate(90deg);
  }

  .bkms-sort-arrow.is-menu {
    width: 10px;
    height: 10px;
  }

  .bkms-sort-arrow.is-desc {
    transform: rotate(-90deg);
  }

  .bkms-sort-tag.is-active .bkms-sort-arrow {
    fill: #3b6ef5;
  }
</style>

<style lang="postcss">
  .bkms-sort-popover.bk-dropdown-popover .bk-dropdown-menu.bkms-sort-menu {
    min-width: 230px;
    padding: 4px 0;
    background: #fff;
    border-color: #e4e9f2;
    border-radius: 6px;
    box-shadow: 0 2px 4px #1919290d;
  }

  .bkms-sort-popover .bkms-sort-item.bk-dropdown-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: auto;
    min-height: 32px;
    padding: 6px 8px;
    font-size: 12px;
    line-height: 20px;
    color: #8f9fbd;
    background: transparent;
  }

  .bkms-sort-popover .bkms-sort-item.bk-dropdown-item:hover {
    color: #8f9fbd;
    background: transparent;
  }

  .bkms-sort-popover .bkms-sort-item.bk-dropdown-item.is-active,
  .bkms-sort-popover .bkms-sort-item.bk-dropdown-item.is-active:hover {
    color: #3b6ef5;
    background: #f0f4ff;
  }

  .bkms-sort-popover .bkms-sort-item.bk-dropdown-item.is-active:has(.bkms-sort-tag:hover) {
    background: #e5edff;
  }
</style>
