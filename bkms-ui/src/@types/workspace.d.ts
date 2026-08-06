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

import { type UserStatisticsOutputObj, type UserWorkspaceStatisticsOutputObj } from './v1/workspace';

import type {
  CreateWorkspaceRequest as CreateWorkspaceRequestV1,
  DeleteWorkspaceRequest,
  ListWorkspacesRequest as ListWorkspacesRequestV1,
  UpdateWorkspaceInfoRequest as UpdateWorkspaceInfoRequestV1,
  WorkspaceDetailOutputObj as WorkspaceDetailOutputObjV1,
  WorkspaceInfoOutputObj as WorkspaceInfoOutputObjV1,
  WorkspaceWithAppsOutputObj as WorkspaceWithAppsOutputObjV1,
} from './v1/workspace';
import type { WorkspaceComponentOutputObj as WorkspaceComponentOutputObjV1 } from './v1/workspace-components';

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type CreateWorkspaceRequest = CreateWorkspaceRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type DeleteWorkspaceRequest = DeleteWorkspaceRequest;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ListWorkspacesRequest = ListWorkspacesRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type UpdateWorkspaceInfoRequest = UpdateWorkspaceInfoRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type UserStatistics = UserStatisticsOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type UserWorkspaceStatistics = UserWorkspaceStatisticsOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type WorkspaceComponentOutputObj = WorkspaceComponentOutputObjV1;

/** 工作空间 (占位) */
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type WorkspaceDetailOutputObj = WorkspaceDetailOutputObjV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type WorkspaceInfoOutputObj = WorkspaceInfoOutputObjV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type WorkspaceWithAppsOutputObj = WorkspaceWithAppsOutputObjV1;
