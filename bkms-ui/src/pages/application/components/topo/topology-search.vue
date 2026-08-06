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
    ref="searchWrapperRef"
    class="topo-search relative"
  >
    <!-- 搜索输入框 -->
    <div
      :class="[
        'topo-search__input group flex items-center h-[32px] border rounded-[2px] transition-colors',
        isFocused ? 'border-[#3A84FF] bg-white' : 'border-[#C4C6CC] bg-white hover:border-[#979BA5]',
      ]"
    >
      <input
        ref="inputRef"
        v-model="inputValue"
        class="flex-1 h-full px-[10px] text-[12px] text-[#63656E] outline-none bg-transparent placeholder-[#C4C6CC]"
        :placeholder="t('搜索资源名称')"
        @focus="handleFocus"
        @keydown.enter="handleEnter"
      />
      <!-- 清除按钮 -->
      <span
        v-if="inputValue"
        :class="[
          'flex items-center justify-center w-[20px] h-[20px] cursor-pointer text-[#C4C6CC] hover:text-[#979BA5] shrink-0 transition-opacity',
          isFocused ? 'opacity-100' : 'opacity-0 group-hover:opacity-100',
        ]"
        @click.stop="handleClear"
      >
        <i class="bkms-icon bkms-icon-close-circle-shape text-[14px]" />
      </span>
      <template v-if="!showDropdown">
        <!-- 有命中结果时显示计数和翻页 -->
        <div
          v-if="inputValue && matchedNodes.length > 0"
          class="flex items-center gap-[2px] text-[12px] text-[#979BA5] shrink-0"
        >
          <span
            class="flex items-center justify-center w-[20px] h-[20px] cursor-pointer hover:text-[#3A84FF] rounded-[2px] hover:bg-[#F0F1F5]"
            :title="t('上一个')"
            @click.stop="navigatePrev"
          >
            <i class="bkms-icon bkms-icon-angle-left text-[14px]" />
          </span>
          <span class="tabular-nums">{{ currentIndex + 1 }} / {{ matchedNodes.length }}</span>
          <span
            class="flex items-center justify-center w-[20px] h-[20px] cursor-pointer hover:text-[#3A84FF] rounded-[2px] hover:bg-[#F0F1F5]"
            :title="t('下一个')"
            @click.stop="navigateNext"
          >
            <i class="bkms-icon bkms-icon-angle-right text-[14px]" />
          </span>
        </div>
        <!-- 无结果提示 -->
        <span
          v-else-if="inputValue && matchedNodes.length === 0"
          class="text-[12px] text-[#C4C6CC] shrink-0 pr-[4px]"
        >
          {{ t('无结果') }}
        </span>
      </template>
      <!-- 搜索图标（始终显示，可点击触发搜索） -->
      <span
        class="flex items-center justify-center w-[32px] h-full cursor-pointer text-[#979BA5] hover:text-[#3A84FF]"
        @click="handleSearchClick"
      >
        <i class="bkms-icon bkms-icon-sousuo text-[14px]" />
      </span>
    </div>

    <!-- 下拉搜索结果面板 -->
    <Teleport to="body">
      <div
        v-show="showDropdown"
        ref="dropdownRef"
        class="topo-search-dropdown fixed z-[9999] bg-white rounded-[4px] shadow-[0_3px_8px_0_rgba(0,0,0,0.16)] border border-[#DCDEE5] max-h-[320px] overflow-y-auto py-[4px]"
        :style="dropdownStyle"
      >
        <!-- 有结果 -->
        <template v-if="matchedNodes.length > 0">
          <div
            v-for="(node, index) in matchedNodes"
            :key="node.id"
            :class="[
              'topo-search-item flex items-center gap-[8px] h-[32px] px-[12px] cursor-pointer text-[12px] text-[#63656E] hover:bg-[#F5F7FA]',
            ]"
            @click="handleSelect(index)"
          >
            <component
              :is="KIND_ICON_ASIDE_MAP[node.kind]"
              v-if="KIND_ICON_ASIDE_MAP[node.kind]"
              class="w-[16px] h-[16px] shrink-0"
            />
            <span
              v-else
              class="bkms-icon bkms-icon-space-basic text-[16px]"
            ></span>
            <span class="truncate">{{ node.name }}</span>
          </div>
        </template>
        <!-- 无结果 -->
        <div
          v-else-if="inputValue"
          class="flex items-center justify-center gap-[8px] h-[32px] px-[12px] text-[12px] text-[#C4C6CC]"
        >
          <i class="bkms-icon bkms-icon-sousuo text-[14px]" />
          <span>{{ t('无结果') }}</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

  import { onClickOutside } from '@vueuse/core';
  import { useI18n } from 'vue-i18n';
  import { TopologyNodeOutputObj } from '~/@types/v1/topology';

  import { KIND_ICON_ASIDE_MAP } from './constants';

  /** 搜索场景中使用的节点类型（id/name/kind 保证存在） */
  export type SearchNode = Omit<TopologyNodeOutputObj, 'id' | 'kind' | 'name'> & {
    id: string;
    kind: string;
    name: string;
  };

  const props = withDefaults(
    defineProps<{
      nodes?: SearchNode[];
    }>(),
    {
      nodes: () => [],
    },
  );

  const emit = defineEmits<{
    /** 当前定位的节点 ID（用于拓扑图定位） */
    locate: [nodeId: string];
    /** 选中/高亮的节点 ID 列表（所有匹配项） */
    'update:selectedIds': [ids: string[]];
  }>();

  const { t } = useI18n();

  const inputRef = ref<HTMLInputElement>();
  const searchWrapperRef = ref<HTMLDivElement>();
  const dropdownRef = ref<HTMLDivElement>();
  const inputValue = ref('');
  const isFocused = ref(false);
  const showDropdown = ref(false);
  const currentIndex = ref(0);
  /** 程序回填搜索框时为 true，避免触发 watch 重新打开下拉 */
  let isSettingValue = false;

  // 下拉面板位置
  const dropdownStyle = ref<Record<string, string>>({});

  function updateDropdownPosition() {
    if (!searchWrapperRef.value) return;
    const rect = searchWrapperRef.value.getBoundingClientRect();
    dropdownStyle.value = {
      top: `${rect.bottom + 4}px`,
      left: `${rect.left}px`,
      width: `${rect.width}px`,
    };
  }

  // 搜索匹配（输入为空时返回全部节点）
  const matchedNodes = computed(() => {
    if (!inputValue.value) return props.nodes;
    const keyword = inputValue.value.toLowerCase();
    return props.nodes.filter(node => node.name.toLowerCase().includes(keyword));
  });

  watch(inputValue, () => {
    if (isSettingValue) return;
    if (isFocused.value) {
      showDropdown.value = true;
      updateDropdownPosition();
    }
  });

  function handleFocus() {
    isFocused.value = true;
    if (matchedNodes.value.length > 0) {
      showDropdown.value = true;
      updateDropdownPosition();
    }
  }

  // 点击外部关闭下拉
  onClickOutside(
    searchWrapperRef,
    () => {
      showDropdown.value = false;
      isFocused.value = false;
    },
    { ignore: [dropdownRef] },
  );

  // 执行搜索（Enter 和点击搜索图标共用）—— 按 inputValue 实际值搜索
  function executeSearch() {
    if (!inputValue.value) {
      emit('update:selectedIds', []);
      showDropdown.value = false;
      inputRef.value?.blur();
      isFocused.value = false;
      return;
    }
    const ids = matchedNodes.value.map(n => n.id);
    emit('update:selectedIds', ids);
    // 定位到第一个匹配节点
    if (matchedNodes.value.length > 0) {
      currentIndex.value = 0;
      const node = matchedNodes.value[0];
      emit('locate', node.id);
    }
    showDropdown.value = false;
    inputRef.value?.blur();
    isFocused.value = false;
  }

  // 清除搜索
  function handleClear() {
    inputValue.value = '';
    currentIndex.value = 0;
    showDropdown.value = false;
    emit('update:selectedIds', []);
  }

  // Enter 确认当前选中
  function handleEnter() {
    executeSearch();
  }

  // 点击搜索图标
  function handleSearchClick() {
    executeSearch();
  }

  // 选中某个搜索结果（点击下拉列表项）
  function handleSelect(index: number) {
    const node = matchedNodes.value[index];
    if (node) {
      // 回填节点名称到搜索框
      isSettingValue = true;
      inputValue.value = node.name;
      nextTick(() => {
        isSettingValue = false;
        // 回填后 matchedNodes 已重新计算，用新的匹配结果来确定 selectedIds
        const newMatchedIds = matchedNodes.value.map(n => n.id);
        emit('update:selectedIds', newMatchedIds);
        // 找到被点击节点在新列表中的正确索引
        const newIndex = matchedNodes.value.findIndex(n => n.id === node.id);
        currentIndex.value = newIndex >= 0 ? newIndex : 0;
      });
      emit('locate', node.id);
    }
    showDropdown.value = false;
    inputRef.value?.blur();
    isFocused.value = false;
  }

  // 上一个/下一个命中项
  function navigateNext() {
    if (matchedNodes.value.length === 0) return;
    currentIndex.value = (currentIndex.value + 1) % matchedNodes.value.length;
    const node = matchedNodes.value[currentIndex.value];
    if (node) emit('locate', node.id);
  }

  function navigatePrev() {
    if (matchedNodes.value.length === 0) return;
    currentIndex.value = (currentIndex.value - 1 + matchedNodes.value.length) % matchedNodes.value.length;
    const node = matchedNodes.value[currentIndex.value];
    if (node) emit('locate', node.id);
  }

  // 监听窗口 resize 更新下拉位置
  onMounted(() => {
    window.addEventListener('resize', updateDropdownPosition);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('resize', updateDropdownPosition);
  });

  defineExpose({
    focus: () => inputRef.value?.focus(),
  });
</script>
