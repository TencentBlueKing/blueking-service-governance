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
  <!-- 无代码仓库标识时降级为手动输入 -->
  <Input
    v-if="!repositoryId"
    ref="inputRef"
    clearable
    :disabled="disabled"
    :model-value="modelValue"
    v-bind="$attrs"
    @update:model-value="handleInputChange"
  />
  <!-- 分支/Tag 分组下拉：打开表单时预拉；分组刷新只请求对应接口 -->
  <Select
    v-else
    ref="selectRef"
    class="repo-ref-select"
    :clearable="false"
    :disabled="disabled"
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
    @update:model-value="handleSelectUpdate"
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
  import { onBeforeUnmount, ref, toRef, useId } from 'vue';

  import { Button, Input, Select } from 'bkui-vue';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';

  import { type RepoRefGroupId, useRepoRefSelect } from './use-repo-ref-select';

  defineOptions({ inheritAttrs: false });

  const props = withDefaults(
    defineProps<{
      /** 是否禁用 */
      disabled?: boolean;
      /** 选中值（短分支名 / Tag 名） */
      modelValue?: string;
      /** 代码仓库标识（repoBuildConfig.repoAlias）；空字符串表示流水线手动输入模式 */
      repositoryId?: string;
      /** 工作空间 ID */
      workspaceId: string;
    }>(),
    {
      disabled: false,
      modelValue: '',
      repositoryId: '',
    },
  );

  const emit = defineEmits<{
    /** 分支值已确认：Select 选中时 / Input change 防抖后 / prepare 回填时 */
    branchCommit: [value: string];
    /** 初始值回填完成 */
    prepared: [value: string];
    'update:modelValue': [value: string];
  }>();

  const { t } = useI18n();
  const inputRef = ref<InstanceType<typeof Input>>();
  const selectRef = ref<InstanceType<typeof Select>>();
  /** 每个实例独立 extCls，避免 document.querySelector 命中其它下拉 */
  const popoverExtCls = `repo-ref-select-popover-${useId().replace(/:/g, '')}`;

  const { groups, optionsLoading, handleSearch, refresh, reset, onDropdownOpen, onDropdownClose, ensureOptionsLoaded } =
    useRepoRefSelect({
      workspaceID: toRef(props, 'workspaceId'),
      repositoryID: toRef(props, 'repositoryId'),
    });

  /** 判断节点是否落在输入框、Select 触发器或本实例的下拉面板上 */
  function containsTarget(target: Node | null) {
    if (!target) return false;

    // 无仓库时仅 Input 模式
    if (!props.repositoryId) {
      const inputEl = (inputRef.value as undefined | { $el?: HTMLElement })?.$el;
      return inputEl?.contains(target) ?? false;
    }

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

  /** 非空分支值确认后通知父级拉取推荐 Tag */
  function emitBranchCommit(value: string) {
    const branch = value.trim();
    if (!branch) return;
    emit('branchCommit', branch);
  }

  /** 分组标题：branch → 代码分支，tag → Tag */
  function getGroupLabel(groupId: RepoRefGroupId) {
    return groupId === 'branch' ? t('代码分支') : 'Tag';
  }

  /** Input 输入变化：立即 trim 同步绑定值，防抖后确认分支 */
  function handleInputChange(value: string) {
    const branch = value.trim();
    emit('update:modelValue', branch);
    debouncedInputConfirm(branch);
  }

  /** 防抖确认：停止输入后通知父级拉取推荐 Tag */
  const debouncedInputConfirm = debounce((value: string) => {
    emitBranchCommit(value);
  }, 500);

  /** 重置下拉状态，并取消未触发的 Input 防抖确认 */
  function handleReset() {
    debouncedInputConfirm.cancel();
    reset();
  }

  onBeforeUnmount(() => {
    debouncedInputConfirm.cancel();
  });

  /** 远程搜索：关闭 Select 内置 searchLoading，交由 hook 防抖处理 */
  function handleRemoteSearch(keyword: string) {
    if (selectRef.value) {
      selectRef.value.searchLoading = false;
    }
    handleSearch(keyword);
  }

  /** Select 选中后立即确认分支值 */
  function handleSelectUpdate(value: string) {
    emit('update:modelValue', value);
    emitBranchCommit(value);
  }

  /** 下拉展开/收起：收起时不把「清空搜索框」当成一次远程搜索 */
  function handleToggle(isOpen: boolean) {
    return isOpen ? onDropdownOpen() : onDropdownClose();
  }

  /** 关闭下拉面板（嵌套 Popover 场景下父级可主动调用；Input 模式无操作） */
  function hidePopover() {
    if (!props.repositoryId) return;
    selectRef.value?.hidePopover?.();
  }

  /**
   * 打开表单时回填默认分支并预拉完整列表（不 await，避免挡住推荐 Tag）。
   */
  function prepare(preferred = '') {
    emit('update:modelValue', preferred);
    emit('prepared', preferred);
    emitBranchCommit(preferred);
    // 有仓库标识时才预拉 branches/tags
    if (props.repositoryId) {
      ensureOptionsLoaded();
    }
    return preferred;
  }

  defineExpose({
    prepare,
    hidePopover,
    reset: handleReset,
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
