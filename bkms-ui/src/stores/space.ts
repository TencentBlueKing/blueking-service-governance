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

import { ref, shallowRef } from 'vue';

import { random } from 'bkui-vue/lib/shared';
import { defineStore } from 'pinia';
import { useI18n } from 'vue-i18n';
import { WorkspaceService } from '~/api/modules/v1/workspace';

import type {
  CreateWorkspaceRequest,
  DeleteWorkspaceRequest,
  ListWorkspacesRequest,
  UpdateWorkspaceInfoRequest,
  WorkspaceInfoOutputObj,
} from '~/@types/v1/workspace';

export const useSpaceStore = defineStore('space', () => {
  const list = ref<WorkspaceInfoOutputObj[]>([]);
  const currentSpace = ref<string>('');
  const routeViewKey = shallowRef(random(10));
  // 当前空间详情
  const workspaceDetail = ref<null | WorkspaceInfoOutputObj>(null);
  const isBoundExistedBKCIProject = ref<boolean>(false);
  const isLoading = ref<boolean>(false);
  const { t } = useI18n();

  // 空间状态
  enum spaceState {
    Disabled = 'Disabled',
    Ready = 'Ready',
  }

  // 空间分类
  const statusTab = ref('');

  const repositoryTypeMap: Record<string, string> = {
    system: t('系统内置'),
  };

  function getRepositoryTypeName(type?: string) {
    if (type && repositoryTypeMap?.[type]) {
      return repositoryTypeMap[type];
    }
    if (type) {
      return type;
    }
    return '--';
  }

  // 获取空间列表
  async function handleGetWorkspaceList(params?: Partial<ListWorkspacesRequest>) {
    isLoading.value = true;
    list.value = await WorkspaceService.listWorkspaces(params).catch(() => []);
    validateSpace();
    isLoading.value = false;
    return list.value;
  }
  // 创建空间
  async function handleCreateWorkspace(params: CreateWorkspaceRequest) {
    return await WorkspaceService.createWorkspace(params)
      .then(() => true)
      .catch(() => false);
  }
  // 更新空间
  async function handleUpdateWorkspace(params: UpdateWorkspaceInfoRequest) {
    return await WorkspaceService.updateWorkspaceInfo(params)
      .then(() => true)
      .catch(() => false);
  }
  // 删除空间
  async function handleDeleteWorkspace(params: DeleteWorkspaceRequest) {
    return await WorkspaceService.deleteWorkspace(params)
      .then(() => true)
      .catch(() => false);
  }

  // 更新当前space缓存
  function updateCurrentSpace(space: string) {
    currentSpace.value = space;
    validateSpace();
    updateSpaceSource();
  }

  // 更新当前space source
  function updateSpaceSource() {
    const spaceData = list.value.find(space => space.id === currentSpace.value);
    if (spaceData) {
      workspaceDetail.value = spaceData;
      isBoundExistedBKCIProject.value = spaceData.bkSystems?.isBoundExistedBKCIProject ?? false;
    } else {
      workspaceDetail.value = null;
      isBoundExistedBKCIProject.value = false;
    }
  }

  // 校验space正确
  function validateSpace() {
    if (!currentSpace.value || !list.value?.length) return;
    const exist = list.value.find(item => item.id === currentSpace.value);
    if (!exist) {
      console.warn(`${currentSpace.value} is not in workspace list`);
      currentSpace.value = '';
    }
  }

  function handleChangeStatusTab(type: string) {
    statusTab.value = type;
  }

  function refreshRouteViewKey() {
    routeViewKey.value = random(10);
  }

  return {
    list,
    currentSpace,
    workspaceDetail,
    isBoundExistedBKCIProject,
    isLoading,
    statusTab,
    spaceState,
    routeViewKey,
    getRepositoryTypeName,
    validateSpace,
    updateCurrentSpace,
    handleGetWorkspaceList,
    handleCreateWorkspace,
    handleUpdateWorkspace,
    handleDeleteWorkspace,
    handleChangeStatusTab,
    refreshRouteViewKey,
  };
});
