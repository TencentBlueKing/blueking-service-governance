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
import type { GetAppEnvVarsRequest, AppDefinedEnvVarOutputObj, CreateAppEnvVarsRequest, ListDetailedAppEnvVarsRequest, AppEnvVarDetailedOutputObj, UpdateAppEnvVarsRequest, DeleteAppEnvVarsRequest, EmptyOutput, ListAppBgEnvVarsRequest, BgEnvVarOutputObj, ListAppEnvVarsRequest, EnvVarOutputObj, ListEnvAvailableEnvVarsRequest, ListEnvBgEnvVarsRequest, ListDetailedEnvScopedEnvVarsRequest, ScopedEnvVarDetailedOutputObj, CreateScopedEnvVarRequest, ScopedEnvVarOutputObj, ListPublicScopedEnvVarsRequest, UpdateScopedEnvVarRequest, DeleteScopedEnvVarRequest } from '~/@types/v1/envvars';

export const EnvvarsService = {
  /**
   * 获取应用直接定义的环境变量列表
   *
   * 只返回 AppModel.workload.envVars 中直接定义的变量，不包含任何继承或合并后的变量。
   *
   * @method GET
   * @path /apps/{appID}/env-vars
   * @tag envvars
   * @param appID path string required 应用 ID
   * @response 200 ListAppDefinedEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppEnvVars: async <Request extends GetAppEnvVarsRequest = GetAppEnvVarsRequest, ResponseData = AppDefinedEnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/env-vars')(params, config),
  /**
   * 创建应用直接定义的环境变量
   *
   * key 在同一应用内必须唯一，重复时服务端会拒绝写入。
   *
   * @method POST
   * @path /apps/{appID}/env-vars
   * @tag envvars
   * @param appID path string required 应用 ID
   * @param body body CreateAppDefinedEnvVarInput required 创建应用环境变量请求
   * @response 200 CreateAppDefinedEnvVarOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppEnvVars: async <Request extends CreateAppEnvVarsRequest = CreateAppEnvVarsRequest, ResponseData = AppDefinedEnvVarOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/env-vars')(params, config),
  /**
   * 获取指定应用的环境变量详情
   *
   * 获取指定应用的环境变量详情，包含可能的 Key 冲突信息。
   *
   * @method GET
   * @path /apps/{appID}/env-vars/detailed-list
   * @tag envvars
   * @param appID path string required 应用 ID
   * @response 200 ListDetailedAppEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listDetailedAppEnvVars: async <Request extends ListDetailedAppEnvVarsRequest = ListDetailedAppEnvVarsRequest, ResponseData = AppEnvVarDetailedOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/env-vars/detailed-list')(params, config),
  /**
   * 更新应用直接定义的环境变量
   *
   * key 表示当前变量 key，updatedKey 表示更新后的变量 key，因此该接口支持“重命名 key”。
   *
   * @method PUT
   * @path /apps/{appID}/env-vars/{key}
   * @tag envvars
   * @param appID path string required 应用 ID
   * @param key path string required 旧环境变量 Key
   * @param body body UpdateAppDefinedEnvVarInput required 更新应用环境变量请求
   * @response 200 UpdateAppDefinedEnvVarOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppEnvVars: async <Request extends UpdateAppEnvVarsRequest = UpdateAppEnvVarsRequest, ResponseData = AppDefinedEnvVarOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/env-vars/{key}')(params, config),
  /**
   * 删除应用直接定义的环境变量
   *
   * 按应用内 key 删除，不再通过整份 AppModelSpec 回写。
   *
   * @method DELETE
   * @path /apps/{appID}/env-vars/{key}
   * @tag envvars
   * @param appID path string required 应用 ID
   * @param key path string required 环境变量 Key
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppEnvVars: async <Request extends DeleteAppEnvVarsRequest = DeleteAppEnvVarsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/env-vars/{key}')(params, config),
  /**
   * 查询应用在某个环境下的背景环境变量列表
   *
   * 背景环境变量指除了作用域为当前应用的其他环境变量，结果列表已按优先级对同 Key 变量去重。
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/bg-env-vars
   * @tag envvars
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 ListAppBgEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppBgEnvVars: async <Request extends ListAppBgEnvVarsRequest = ListAppBgEnvVarsRequest, ResponseData = BgEnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/bg-env-vars')(params, config),
  /**
   * 获取应用部署到某个环境后可用的环境变量
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/env-variables
   * @tag envvars
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 ListAppEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppEnvVars: async <Request extends ListAppEnvVarsRequest = ListAppEnvVarsRequest, ResponseData = EnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/env-variables')(params, config),
  /**
   * 查询指定环境下所有可用的环境变量列表
   *
   * 包含所有内置、公共及当前环境下直接配置的全部变量，结果列表已按优先级去重。这些变量将在创建应用等应用缺席的场景中被使用。
   *
   * @method GET
   * @path /envs/{envID}/available-env-vars
   * @tag envvars
   * @param envID path string required 环境 ID
   * @response 200 ListEnvAvailableEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEnvAvailableEnvVars: async <Request extends ListEnvAvailableEnvVarsRequest = ListEnvAvailableEnvVarsRequest, ResponseData = EnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/available-env-vars')(params, config),
  /**
   * 查询指定环境的背景环境变量列表
   *
   * 背景环境变量指除了作用域为当前环境的其他环境变量，结果列表已按优先级对同 Key 变量去重。
   *
   * @method GET
   * @path /envs/{envID}/bg-env-vars
   * @tag envvars
   * @param envID path string required 环境 ID
   * @response 200 ListEnvBgEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEnvBgEnvVars: async <Request extends ListEnvBgEnvVarsRequest = ListEnvBgEnvVarsRequest, ResponseData = BgEnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/bg-env-vars')(params, config),
  /**
   * 获取指定环境下作用域为当前环境的环境变量详情
   *
   * @method GET
   * @path /scoped-env-vars/detailed-list/{envID}
   * @tag envvars
   * @param envID path string required 环境 ID
   * @response 200 ListDetailedEnvScopedEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listDetailedEnvScopedEnvVars: async <Request extends ListDetailedEnvScopedEnvVarsRequest = ListDetailedEnvScopedEnvVarsRequest, ResponseData = ScopedEnvVarDetailedOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/scoped-env-vars/detailed-list/{envID}')(params, config),
  /**
   * 创建作用域级别的环境变量（ScopedEnvVar）
   *
   * @method POST
   * @path /workspaces/{workspaceID}/scoped-env-vars
   * @tag envvars
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateScopedEnvVarInput required 创建作用域级别环境变量请求
   * @response 200 CreateScopedEnvVarOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createScopedEnvVar: async <Request extends CreateScopedEnvVarRequest = CreateScopedEnvVarRequest, ResponseData = ScopedEnvVarOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/scoped-env-vars')(params, config),
  /**
   * 获取指定空间下的公共环境变量列表
   *
   * 公开环境变量，指作用域为 workspace（工作空间）和 envType（环境类型）的作用域级别环境变量，不包含作用域为 env（单环境）。
   *
   * @method GET
   * @path /workspaces/{workspaceID}/scoped-env-vars/public-vars
   * @tag envvars
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListPublicScopedEnvVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPublicScopedEnvVars: async <Request extends ListPublicScopedEnvVarsRequest = ListPublicScopedEnvVarsRequest, ResponseData = ScopedEnvVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/scoped-env-vars/public-vars')(params, config),
  /**
   * 更新作用域级别的环境变量（ScopedEnvVar）
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID}
   * @tag envvars
   * @param workspaceID path string required 工作空间 ID
   * @param scopedEnvVarID path string required Scoped EnvVar ID
   * @param body body UpdateScopedEnvVarInput required 更新作用域级别环境变量请求
   * @response 200 UpdateScopedEnvVarOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateScopedEnvVar: async <Request extends UpdateScopedEnvVarRequest = UpdateScopedEnvVarRequest, ResponseData = ScopedEnvVarOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID}')(params, config),
  /**
   * 删除作用域级别的环境变量（ScopedEnvVar）
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID}
   * @tag envvars
   * @param workspaceID path string required 工作空间 ID
   * @param scopedEnvVarID path string required Scoped EnvVar ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteScopedEnvVar: async <Request extends DeleteScopedEnvVarRequest = DeleteScopedEnvVarRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID}')(params, config),
};
