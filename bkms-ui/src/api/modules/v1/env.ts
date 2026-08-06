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
import type { ListAppEnvsRequest, EnvOutput, ListFeatureEnvsRequest, FeatureEnvOutput, CreateFeatureEnvRequest, GetEnvRequest, EnvDetailOutput, DeleteEnvRequest, EmptyOutput, UpdateEnvBasicInfoRequest, UpdateEnvClusterRequest, ListEnvsRequest, CreateEnvRequest, EnvIDOutput, ListEnvTrafficLanesRequest, TrafficLaneOutput } from '~/@types/v1/env';

export const EnvService = {
  /**
   * 获取应用可用环境列表，包含工作空间下的标准环境以及应用专用的特性环境。
   *
   * @method GET
   * @path /apps/{appID}/envs
   * @tag env
   * @param appID path string required 应用 ID
   * @response 200 ListEnvsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppEnvs: async <Request extends ListAppEnvsRequest = ListAppEnvsRequest, ResponseData = EnvOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs')(params, config),
  /**
   * 获取应用的所有特性环境
   *
   * 返回管理页展示所需的特性环境、来源环境、部署位置及创建信息。
   *
   * @method GET
   * @path /apps/{appID}/feat-envs
   * @tag env
   * @param appID path string required 应用 ID
   * @param with_deploy_status query boolean 是否返回当前应用在每个特性环境下的部署状态
   * @response 200 ListFeatureEnvsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  listFeatureEnvs: async <Request extends ListFeatureEnvsRequest = ListFeatureEnvsRequest, ResponseData = FeatureEnvOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/feat-envs')(params, config),
  /**
   * 创建应用特性环境
   *
   * @method POST
   * @path /apps/{appID}/feat-envs
   * @tag env
   * @param appID path string required 应用 ID
   * @param body body CreateFeatureEnvInput required 创建特性环境请求
   * @response 200 CreateFeatureEnvOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createFeatureEnv: async <Request extends CreateFeatureEnvRequest = CreateFeatureEnvRequest, ResponseData = EnvOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/feat-envs')(params, config),
  /**
   * 获取单个环境详情
   *
   * @method GET
   * @path /envs/{envID}
   * @tag env
   * @param envID path string required 环境 ID
   * @response 200 GetEnvOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getEnv: async <Request extends GetEnvRequest = GetEnvRequest, ResponseData = EnvDetailOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}')(params, config),
  /**
   * 删除环境
   *
   * @method DELETE
   * @path /envs/{envID}
   * @tag env
   * @param envID path string required 环境 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteEnv: async <Request extends DeleteEnvRequest = DeleteEnvRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/envs/{envID}')(params, config),
  /**
   * 更新部署环境基本信息
   *
   * @method PUT
   * @path /envs/{envID}/basic-info
   * @tag env
   * @param envID path string required 环境 ID
   * @param body body UpdateEnvBasicInfoInput required 更新环境基本信息请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateEnvBasicInfo: async <Request extends UpdateEnvBasicInfoRequest = UpdateEnvBasicInfoRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/envs/{envID}/basic-info')(params, config),
  /**
   * 更新部署环境集群配置
   *
   * @method PUT
   * @path /envs/{envID}/cluster
   * @tag env
   * @param envID path string required 环境 ID
   * @param body body UpdateEnvClusterInput required 更新环境集群配置请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateEnvCluster: async <Request extends UpdateEnvClusterRequest = UpdateEnvClusterRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/envs/{envID}/cluster')(params, config),
  /**
   * 获取空间下的环境列表
   *
   * [bkms-cli 使用] 避免破坏性修改
   *
   * @method GET
   * @path /workspaces/{workspaceID}/envs
   * @tag env
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListEnvsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEnvs: async <Request extends ListEnvsRequest = ListEnvsRequest, ResponseData = EnvOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/envs')(params, config),
  /**
   * 创建部署环境
   *
   * @method POST
   * @path /workspaces/{workspaceID}/envs
   * @tag env
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateEnvInput required 创建环境请求
   * @response 200 CreateEnvOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createEnv: async <Request extends CreateEnvRequest = CreateEnvRequest, ResponseData = EnvIDOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/envs')(params, config),
  /**
   * 获取指定环境下的泳道列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/envs/{envName}/traffic-lanes
   * @tag env
   * @param workspaceID path string required 工作空间 ID
   * @param envName path string required 环境名称
   * @response 200 ListEnvTrafficLanesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEnvTrafficLanes: async <Request extends ListEnvTrafficLanesRequest = ListEnvTrafficLanesRequest, ResponseData = TrafficLaneOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/envs/{envName}/traffic-lanes')(params, config),
};
