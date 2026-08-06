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
import type { ListBscpCfgEnvBindingsRequest, EnvBindingOutput, GetBscpCfgEnvBindingRequest, CreateBscpCfgEnvBindingRequest, DeleteBscpCfgEnvBindingRequest, PatchBscpCfgEnvBindingRequest, GetBscpCfgMetadataRequest, MetadataOutput, InitBscpCfgMetadataRequest, DeleteBscpCfgMetadataRequest, PatchBscpCfgMetadataRequest } from '~/@types/v1/bscpcfg';

export const BscpcfgService = {
  /**
   * 获取所有环境的绑定列表
   *
   * @method GET
   * @path /apps/{appID}/bscpcfg/envs
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @response 200 EnvBindingListResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBscpCfgEnvBindings: async <Request extends ListBscpCfgEnvBindingsRequest = ListBscpCfgEnvBindingsRequest, ResponseData = EnvBindingOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/bscpcfg/envs')(params, config),
  /**
   * 获取指定环境的绑定详情
   *
   * @method GET
   * @path /apps/{appID}/bscpcfg/envs/{envName}/binding
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EnvBindingResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getBscpCfgEnvBinding: async <Request extends GetBscpCfgEnvBindingRequest = GetBscpCfgEnvBindingRequest, ResponseData = EnvBindingOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/bscpcfg/envs/{envName}/binding')(params, config),
  /**
   * 创建环境绑定
   *
   * @method POST
   * @path /apps/{appID}/bscpcfg/envs/{envName}/binding
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EnvBindingResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createBscpCfgEnvBinding: async <Request extends CreateBscpCfgEnvBindingRequest = CreateBscpCfgEnvBindingRequest, ResponseData = EnvBindingOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/bscpcfg/envs/{envName}/binding')(params, config),
  /**
   * 删除环境绑定
   *
   * @method DELETE
   * @path /apps/{appID}/bscpcfg/envs/{envName}/binding
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteBscpCfgEnvBinding: async <Request extends DeleteBscpCfgEnvBindingRequest = DeleteBscpCfgEnvBindingRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/bscpcfg/envs/{envName}/binding')(params, config),
  /**
   * 更新环境绑定（更新绑定的服务列表）
   *
   * @method PATCH
   * @path /apps/{appID}/bscpcfg/envs/{envName}/binding
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body PatchEnvBindingInput required 更新配置请求体
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  patchBscpCfgEnvBinding: async <Request extends PatchBscpCfgEnvBindingRequest = PatchBscpCfgEnvBindingRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/bscpcfg/envs/{envName}/binding')(params, config),
  /**
   * 获取配置管理元信息
   *
   * @method GET
   * @path /apps/{appID}/bscpcfg/metadata
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @response 200 MetadataResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getBscpCfgMetadata: async <Request extends GetBscpCfgMetadataRequest = GetBscpCfgMetadataRequest, ResponseData = MetadataOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/bscpcfg/metadata')(params, config),
  /**
   * 初始化配置管理
   *
   * @method POST
   * @path /apps/{appID}/bscpcfg/metadata
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @response 200 MetadataResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  initBscpCfgMetadata: async <Request extends InitBscpCfgMetadataRequest = InitBscpCfgMetadataRequest, ResponseData = MetadataOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/bscpcfg/metadata')(params, config),
  /**
   * 删除配置管理元信息（级联删除所有环境绑定）
   *
   * @method DELETE
   * @path /apps/{appID}/bscpcfg/metadata
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteBscpCfgMetadata: async <Request extends DeleteBscpCfgMetadataRequest = DeleteBscpCfgMetadataRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/bscpcfg/metadata')(params, config),
  /**
   * 修改配置管理元信息（mountPath）
   *
   * @method PATCH
   * @path /apps/{appID}/bscpcfg/metadata
   * @tag bscpcfg
   * @param appID path string required 应用 ID
   * @param body body PatchMetadataInput required 更新配置请求体
   * @response 200 MetadataResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  patchBscpCfgMetadata: async <Request extends PatchBscpCfgMetadataRequest = PatchBscpCfgMetadataRequest, ResponseData = MetadataOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/bscpcfg/metadata')(params, config),
};
