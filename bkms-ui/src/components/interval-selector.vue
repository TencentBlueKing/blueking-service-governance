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
    ref="dropdownRef"
    placement="bottom-start"
    trigger="click"
  >
    <!-- 触发区：自动刷新图标 + 文案 + 下拉箭头 -->
    <span class="flex items-center h-[32px] cursor-pointer select-none">
      <span
        v-bk-tooltips="t('自动刷新设置')"
        class="flex items-center px-[4px] py-[2px] rounded-[2px] hover:bg-[#eaf3ff] hover:text-[#3a84ff]"
      >
        <i class="bkms-icon bkms-icon-auto-refresh-line"></i>
        <span :class="['text-[12px] pl-[4px]', textActive ? 'text-[#3a84ff]' : '']">{{ triggerText }}</span>
      </span>
      <Divider
        class="h-[12px] min-h-[16px] mx-[12px]"
        color="#DCDEE5"
        direction="vertical"
        type="solid"
      />
      <!-- 立即刷新 -->
      <i
        v-bk-tooltips="t('刷新')"
        :class="[
          'bkms-icon bkms-icon-refresh cursor-pointer text-[#63656E] px-[4px] py-[2px] hover:bg-[#eaf3ff] hover:text-[#3a84ff] rounded-[2px]',
        ]"
        @click.stop="handleRefresh"
        @mousedown.stop
      ></i>
    </span>
    <template #content>
      <Dropdown.DropdownMenu>
        <Dropdown.DropdownItem
          v-for="option in options"
          :key="option.id"
          :class="['!h-[32px] !leading-[32px]', option.id === value ? 'text-[#3a84ff] !bg-[#eaf3ff]' : '']"
          @click="handleSelect(option)"
        >
          {{ option.name }}
        </Dropdown.DropdownItem>
      </Dropdown.DropdownMenu>
    </template>
  </Dropdown>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Divider, Dropdown } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  interface IEmits {
    (e: 'update:value', id: number): void;
    (e: 'select', id: number): void;
    (e: 'refresh'): void;
  }

  interface IIntervalOption {
    /** 选项值（毫秒），-1 表示关闭自动刷新 */
    id: number;
    /** 选项展示名称 */
    name: string;
  }

  const props = withDefaults(
    defineProps<{
      /** 可选刷新间隔列表 */
      list?: IIntervalOption[];
    }>(),
    {
      list: () => [],
    },
  );

  const value = defineModel<number>('value', { default: -1 });
  const emit = defineEmits<IEmits>();

  const { t } = useI18n();

  /** 默认刷新间隔列表 */
  const defaultList = computed<IIntervalOption[]>(() => [
    { id: -1, name: `${t('关闭')}（off）` },
    { id: 60 * 1000, name: '1m' },
    { id: 5 * 60 * 1000, name: '5m' },
    { id: 15 * 60 * 1000, name: '15m' },
    { id: 30 * 60 * 1000, name: '30m' },
    { id: 60 * 60 * 1000, name: '1h' },
    { id: 2 * 60 * 60 * 1000, name: '2h' },
    { id: 24 * 60 * 60 * 1000, name: '1d' },
  ]);

  const options = computed(() => (props.list.length ? props.list : defaultList.value));

  const OFF_LABEL = computed(() => `${t('关闭')}（off）`);

  /** 当前选中项 */
  const selectedOption = computed(() => options.value.find(item => item.id === value.value) ?? options.value[0]);

  /** 触发区域展示文案，关闭时展示 off */
  const triggerText = computed(() => {
    const name = selectedOption.value?.name ?? OFF_LABEL.value;
    return name === OFF_LABEL.value ? 'off' : name;
  });

  /** 是否处于自动刷新状态（非关闭） */
  const textActive = computed(() => value.value !== -1);

  const dropdownRef = ref<InstanceType<typeof Dropdown>>();

  /** 点击立即刷新 */
  function handleRefresh(e: MouseEvent) {
    e.stopPropagation();
    emit('refresh');
  }

  /** 选中刷新间隔 */
  function handleSelect(option: IIntervalOption) {
    value.value = option.id;
    emit('select', option.id);
    dropdownRef.value?.popoverRef?.hide();
  }
</script>
