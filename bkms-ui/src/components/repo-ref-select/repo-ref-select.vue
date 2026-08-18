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
  <!-- 分支/Tag 分组下拉：打开表单时预拉；分组刷新只请求对应接口 -->
  <Select
    ref="selectRef"
    class="repo-ref-select"
    :clearable="false"
    :disabled="disabled || !repositoryId"
    filterable
    :loading="optionsLoading"
    :model-value="modelValue"
    :no-data-text="$t('暂无选项')"
    :no-match-text="$t('暂无选项')"
    :popover-options="{ extCls: popoverExtCls }"
    :remote-method="handleRemoteSearch"
    :search-placeholder="$t('输入关键字搜索')"
    v-bind="$attrs"
    @toggle="handleToggle"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <Select.Group
      v-for="group in groups"
      :key="group.id"
      class="option-group"
      :label="getGroupLabel(group.id)"
    >
      <!-- 自定义组头：左侧分组名，右侧刷新按钮 -->
      <template #label>
        <div class="flex items-center justify-between pr-[10px]">
          <span>{{ getGroupLabel(group.id) }} ({{ group.children.length }})</span>
          <Button
            text
            theme="primary"
            @click.stop="refresh(group.id)"
          >
            {{ $t('刷新') }}
          </Button>
        </div>
      </template>
      <Select.Option
        v-for="option in group.children"
        :id="option.id"
        :key="`${group.id}:${option.id}`"
        :name="option.text"
      />
    </Select.Group>
  </Select>
</template>

<script lang="ts" setup>
  import { ref, toRef, useId } from 'vue';

  import { Button, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import { type RepoRefGroupId, useRepoRefSelect } from './use-repo-ref-select';

  defineOptions({ inheritAttrs: false });

  const props = withDefaults(
    defineProps<{
      /** 是否禁用 */
      disabled?: boolean;
      /** 选中值（短分支名 / Tag 名） */
      modelValue?: string;
      /** 代码仓库标识（repoBuildConfig.repoAlias） */
      repositoryId: string;
      /** 工作空间 ID */
      workspaceId: string;
    }>(),
    {
      disabled: false,
      modelValue: '',
    },
  );

  const emit = defineEmits<{
    /** 初始值回填完成 */
    prepared: [value: string];
    'update:modelValue': [value: string];
  }>();

  const { t } = useI18n();
  const selectRef = ref<InstanceType<typeof Select>>();
  /** 每个实例独立 extCls，避免 document.querySelector 命中其它下拉 */
  const popoverExtCls = `repo-ref-select-popover-${useId().replace(/:/g, '')}`;

  const { groups, optionsLoading, handleSearch, refresh, reset, onDropdownOpen, onDropdownClose, ensureOptionsLoaded } =
    useRepoRefSelect({
      workspaceID: toRef(props, 'workspaceId'),
      repositoryID: toRef(props, 'repositoryId'),
    });

  /** 判断节点是否落在 Select 触发器或本实例的下拉面板上 */
  function containsTarget(target: Node | null) {
    if (!target) return false;

    const selectEl = (selectRef.value as undefined | { $el?: HTMLElement })?.$el;
    if (selectEl?.contains(target)) return true;

    return Boolean(document.querySelector(`.${popoverExtCls}`)?.contains(target));
  }

  /** 嵌套 Popover 场景：点击外部时收起下拉 */
  function dismissIfOutside(target: Node | null) {
    if (!containsTarget(target)) {
      hidePopover();
    }
  }

  /** 分组标题：branch → 代码分支，tag → Tag */
  function getGroupLabel(groupId: RepoRefGroupId) {
    return groupId === 'branch' ? t('代码分支') : 'Tag';
  }

  /** 远程搜索：关闭 Select 内置 searchLoading，交由 hook 防抖处理 */
  function handleRemoteSearch(keyword: string) {
    if (selectRef.value) {
      selectRef.value.searchLoading = false;
    }
    handleSearch(keyword);
  }

  /** 下拉展开/收起：收起时不把「清空搜索框」当成一次远程搜索 */
  function handleToggle(isOpen: boolean) {
    if (isOpen) {
      onDropdownOpen();
    } else {
      onDropdownClose();
    }
  }

  /** 关闭下拉面板（嵌套 Popover 场景下父级可主动调用） */
  function hidePopover() {
    selectRef.value?.hidePopover?.();
  }

  /**
   * 打开表单时回填默认分支并预拉完整列表（不 await，避免挡住推荐 Tag）。
   */
  function prepare(preferred = '') {
    emit('update:modelValue', preferred);
    emit('prepared', preferred);
    ensureOptionsLoaded();
    return preferred;
  }

  defineExpose({
    prepare,
    hidePopover,
    reset,
    refresh,
    containsTarget,
    dismissIfOutside,
  });
</script>

<!-- 下拉面板 teleport 到 body，用 extCls 限定样式作用域 -->
<style lang="postcss">
  [class*='repo-ref-select-popover-'] {
    .option-group {
      position: relative;
    }
    .bk-option-group-label {
      border-bottom: 1px solid #dcdee5;
      margin: 0 16px;
      padding: 0 !important;
    }
  }
</style>
