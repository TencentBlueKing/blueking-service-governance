/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { computed, nextTick, ref } from 'vue';

import { useSpaceStore } from '~/stores/space';

import type RepoRefSelect from '~/components/repo-ref-select/repo-ref-select.vue';

/**
 * 应用详情页内 RepoRefSelect 的通用 wiring：
 * workspaceId / repoAlias / ref / prepare / reset / 嵌套 Popover 点击收起。
 */
export function useAppRepoRefSelect(getRepoAlias: () => string) {
  const spaceStore = useSpaceStore();

  /** 当前工作空间 ID */
  const workspaceId = computed(() => spaceStore.currentSpace || '');
  /** 代码仓库别名，作为 repositoryID（repositoryType 固定 NAME） */
  const repoAlias = computed(getRepoAlias);
  const branchSelectRef = ref<InstanceType<typeof RepoRefSelect>>();

  /**
   * 回填默认分支并预拉 branches/tags。
   * 若组件在 v-if 内，需先 await nextTick() 再调用。
   */
  function prepareBranch(preferred = '') {
    return branchSelectRef.value?.prepare(preferred) ?? preferred;
  }

  /** 关闭弹层时收起下拉并清空缓存 */
  function resetBranchSelect() {
    branchSelectRef.value?.hidePopover?.();
    branchSelectRef.value?.reset?.();
  }

  /** 嵌套 Popover：点击内容区外部时收起分支下拉 */
  function dismissBranchSelectOnOutsideMouseDown(e: MouseEvent) {
    branchSelectRef.value?.dismissIfOutside?.(e.target as Node | null);
  }

  /** v-if 挂载后再 prepare 的便捷封装 */
  async function prepareBranchAfterMount(preferred = '') {
    await nextTick();
    return prepareBranch(preferred);
  }

  return {
    workspaceId,
    repoAlias,
    branchSelectRef,
    prepareBranch,
    prepareBranchAfterMount,
    resetBranchSelect,
    dismissBranchSelectOnOutsideMouseDown,
  };
}
