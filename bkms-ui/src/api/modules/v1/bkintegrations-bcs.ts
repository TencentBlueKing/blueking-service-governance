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

/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListBCSAuthorizedProjectsRequest, BCSProjectOutput, GetBCSProjectRequest, ListClustersByProjectRequest, ClusterOutput, ListNamespacesByClusterRequest, NamespaceOutput } from '~/@types/v1/bkintegrations-bcs';

export const BkintegrationsBcsService = {
  /**
   * 获取有权限的 BCS 项目列表
   *
   * @method GET
   * @path /bcs/projects/authorized
   * @tag bkintegrations-bcs
   * @response 200 ListBCSAuthorizedProjectsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBCSAuthorizedProjects: async <Request extends ListBCSAuthorizedProjectsRequest = ListBCSAuthorizedProjectsRequest, ResponseData = BCSProjectOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bcs/projects/authorized')(params, config),
  /**
   * 根据项目 ID 获取 BCS 项目详情
   *
   * @method GET
   * @path /bcs/projects/{projectID}
   * @tag bkintegrations-bcs
   * @param projectID path string required BCS 项目 ID
   * @response 200 GetBCSProjectOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getBCSProject: async <Request extends GetBCSProjectRequest = GetBCSProjectRequest, ResponseData = BCSProjectOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bcs/projects/{projectID}')(params, config),
  /**
   * 获取 BCS 项目下的集群列表
   *
   * @method GET
   * @path /bcs/projects/{projectID}/clusters
   * @tag bkintegrations-bcs
   * @param projectID path string required BCS 项目 ID
   * @response 200 ListClustersByProjectOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listClustersByProject: async <Request extends ListClustersByProjectRequest = ListClustersByProjectRequest, ResponseData = ClusterOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bcs/projects/{projectID}/clusters')(params, config),
  /**
   * 获取 BCS 集群下的命名空间列表
   *
   * @method GET
   * @path /bcs/projects/{projectID}/clusters/{clusterID}/namespaces
   * @tag bkintegrations-bcs
   * @param projectID path string required BCS 项目 ID
   * @param clusterID path string required 集群 ID
   * @response 200 ListNamespacesByClusterOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listNamespacesByCluster: async <Request extends ListNamespacesByClusterRequest = ListNamespacesByClusterRequest, ResponseData = NamespaceOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bcs/projects/{projectID}/clusters/{clusterID}/namespaces')(params, config),
};
