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
/**
 * BKCI 代码仓库分支/Tag 选择。
 * 无搜索的完整列表只请求一次并缓存；搜索结果只用于当前展示，关闭下拉后还原缓存，不再请求。
 */
import { type MaybeRefOrGetter, computed, ref, toValue, watch } from 'vue';

import { useDebounce } from '@vueuse/core';
import { BkintegrationsBkciService } from '~/api/modules/v1';

import type {
  BkCIRepositoryRefOutput,
  ListBkCIRepositoryBranchesRequest,
  ListBkCIRepositoryTagsRequest,
} from '~/@types/v1/bkintegrations-bkci';
import type { Config } from '~/api/interceptors';

/** branches / tags 查询参数结构一致，用交叉类型共用组装逻辑 */
type ListBkCIRepositoryRefRequest = ListBkCIRepositoryBranchesRequest & ListBkCIRepositoryTagsRequest;

/** 仓库标识类型：当前业务固定按仓库名称（别名）查询 */
const REPOSITORY_TYPE_NAME = 'NAME';

/** 下拉首次加载每页条数（接口不返回 total，尽量一次多拉；更多结果请搜索） */
const DEFAULT_PAGE_SIZE = 100;

/** 按下拉字段分组（分支 / Tag） */
export type RepoRefGroup = {
  children: RepoRefOption[];
  id: RepoRefGroupId;
};

/** 分组标识：分支 / Tag */
export type RepoRefGroupId = 'branch' | 'tag';

/** 下拉分组内单个选项 */
export type RepoRefOption = {
  id: string;
  text: string;
};

type UseRepoRefSelectOptions = {
  /** 代码仓库标识，通常为 buildConfig.repoBuildConfig.repoAlias */
  repositoryID: MaybeRefOrGetter<string>;
  /** 工作空间 ID */
  workspaceID: MaybeRefOrGetter<string>;
};

/** 分组与 BKCI 列表接口的映射 */
const GROUP_FETCHERS = {
  branch: BkintegrationsBkciService.listBkCIRepositoryBranches.bind(BkintegrationsBkciService),
  tag: BkintegrationsBkciService.listBkCIRepositoryTags.bind(BkintegrationsBkciService),
} as const satisfies Record<
  RepoRefGroupId,
  (params?: ListBkCIRepositoryRefRequest, config?: Config) => Promise<BkCIRepositoryRefOutput[]>
>;

/**
 * 分支/Tag 下拉数据逻辑，由 RepoRefSelect 组件封装。
 */
export function useRepoRefSelect(options: UseRepoRefSelectOptions) {
  const workspaceID = computed(() => toValue(options.workspaceID));
  const repositoryID = computed(() => toValue(options.repositoryID));

  /** 当前展示的选项（完整列表或搜索结果） */
  const groups = ref<RepoRefGroup[]>([]);
  /** 无搜索时的完整列表缓存，关闭下拉或清空搜索时用来还原 */
  const fullGroups = ref<RepoRefGroup[]>([]);
  /** 下拉全局 loading（含分组刷新） */
  const optionsLoading = ref(false);
  /** 远程搜索关键字；关闭下拉时会清空，不能据此再打接口 */
  const searchKeyword = ref('');
  /** 下拉是否展开：Select 选中后会 remote-method('')，需靠此标志忽略关闭时的空搜索 */
  const dropdownOpen = ref(false);
  /** 是否已成功拉过完整列表（空仓库也算已加载，避免反复请求） */
  const loaded = ref(false);

  /** 请求世代：仓库切换 / reset / refresh 后丢弃过期响应 */
  let requestId = 0;
  /** 完整列表预拉与展开下拉共用，避免重复打 branches/tags */
  let inflight: null | Promise<void> = null;

  /** 工作空间和仓库都就绪才允许请求 */
  function canQuery() {
    return Boolean(workspaceID.value && repositoryID.value);
  }

  /**
   * 组装 branches / tags 共用查询参数。
   * 空 search 不要写成 search: undefined，否则会变成 query 里的 search=undefined。
   */
  function buildParams(search = ''): ListBkCIRepositoryRefRequest | null {
    if (!canQuery()) return null;
    return {
      workspaceID: workspaceID.value,
      repositoryID: repositoryID.value,
      repositoryType: REPOSITORY_TYPE_NAME,
      page: 1,
      pageSize: DEFAULT_PAGE_SIZE,
      ...(search ? { search } : {}),
    };
  }

  /**
   * 按分组请求对应接口。
   * 入参需拷贝：请求层会 delete path 参数，并行调用若共用同一对象会互相污染。
   */
  async function fetchGroup(groupId: RepoRefGroupId, search = ''): Promise<RepoRefOption[]> {
    const params = buildParams(search);
    if (!params) return [];

    const list = await GROUP_FETCHERS[groupId]({ ...params }, { validateCode: false }).catch(
      () => [] as BkCIRepositoryRefOutput[],
    );

    return (list || []).map(item => ({ id: item.name || '', text: item.name || '' })).filter(opt => opt.id);
  }

  /** 按固定顺序组装分组：分支在前、Tag 在后；空列表不展示该组 */
  function toGroups(branches: RepoRefOption[], tags: RepoRefOption[]): RepoRefGroup[] {
    const next: RepoRefGroup[] = [];
    if (branches.length) next.push({ id: 'branch', children: branches });
    if (tags.length) next.push({ id: 'tag', children: tags });
    return next;
  }

  /** 取当前展示中某一组的 children，供分组刷新时保留另一组 */
  function childrenOf(groupId: RepoRefGroupId) {
    return groups.value.find(group => group.id === groupId)?.children ?? [];
  }

  /** 用缓存的完整列表覆盖当前展示，不发请求 */
  function showFullList() {
    groups.value = fullGroups.value;
  }

  /**
   * 并行拉取分支与 Tag。
   * search 为空时写入 fullGroups；非空时只更新当前展示，不覆盖缓存。
   */
  async function fetchAll(search = '') {
    if (!canQuery()) {
      groups.value = [];
      return;
    }

    const id = ++requestId;
    optionsLoading.value = true;
    try {
      const [branches, tags] = await Promise.all([fetchGroup('branch', search), fetchGroup('tag', search)]);
      if (id !== requestId) return;

      const next = toGroups(branches, tags);
      groups.value = next;
      loaded.value = true;
      if (!search) {
        fullGroups.value = next;
      }
    } finally {
      if (id === requestId) {
        optionsLoading.value = false;
      }
    }
  }

  /** 完整列表未加载时请求一次；多次调用复用同一 Promise */
  function ensureOptionsLoaded() {
    if (loaded.value) return Promise.resolve();
    if (!inflight) {
      inflight = fetchAll('').finally(() => {
        inflight = null;
      });
    }
    return inflight;
  }

  /** 远程搜索：只改关键字，由防抖 watch 决定是否请求 */
  function handleSearch(keyword: string) {
    searchKeyword.value = keyword?.trim() || '';
  }

  /**
   * 刷新指定分组：只请求该组接口，另一组保持现状。
   * 无搜索关键字时同步更新完整列表缓存。
   */
  async function refresh(groupId: RepoRefGroupId) {
    if (optionsLoading.value || !canQuery()) return;

    const id = ++requestId;
    optionsLoading.value = true;
    try {
      const children = await fetchGroup(groupId, searchKeyword.value);
      if (id !== requestId) return;

      const branches = groupId === 'branch' ? children : childrenOf('branch');
      const tags = groupId === 'tag' ? children : childrenOf('tag');
      groups.value = toGroups(branches, tags);
      loaded.value = true;
      if (!searchKeyword.value) {
        fullGroups.value = groups.value;
      }
    } finally {
      if (id === requestId) {
        optionsLoading.value = false;
      }
    }
  }

  /** 清空本地状态并作废进行中的请求（关闭弹层、切换仓库时） */
  function reset() {
    requestId += 1;
    inflight = null;
    searchKeyword.value = '';
    loaded.value = false;
    groups.value = [];
    fullGroups.value = [];
    optionsLoading.value = false;
  }

  /** 展开下拉：若完整列表尚未返回则等待同一请求，然后展示缓存 */
  async function onDropdownOpen() {
    dropdownOpen.value = true;
    await ensureOptionsLoaded();
    showFullList();
  }

  /**
   * 收起下拉：清空搜索关键字并还原完整列表。
   * 不发请求。选中后 Select 清空搜索框触发的 remote-method('') 会被 dropdownOpen=false 挡住。
   */
  function onDropdownClose() {
    dropdownOpen.value = false;
    searchKeyword.value = '';
    showFullList();
  }

  // 仓库变化时仅清空本地状态；列表在 prepare / 展开下拉时懒加载
  watch([workspaceID, repositoryID], () => {
    reset();
  });

  // 仅在下拉展开时搜索；清空关键字还原缓存，不发请求
  watch(useDebounce(searchKeyword, 300), async keyword => {
    if (!dropdownOpen.value || !loaded.value) return;
    if (!keyword) {
      showFullList();
      return;
    }
    await fetchAll(keyword);
  });

  return {
    groups,
    optionsLoading,
    handleSearch,
    refresh,
    reset,
    ensureOptionsLoaded,
    onDropdownOpen,
    onDropdownClose,
  };
}
