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
import type { ListBSCPBizsRequest, BSCPBizOutput, ListBSCPServicesRequest, BSCPServiceOutput, ListBSCPConfigsRequest, BSCPConfigOutput, GetBSCPConfigRequest, BSCPConfigDetailOutput } from '~/@types/v1/bkintegrations-bscp';

export const BkintegrationsBscpService = {
  /**
   * 获取用户的 BSCP 业务列表
   *
   * @method GET
   * @path /bscp/bizs
   * @tag bkintegrations-bscp
   * @response 200 ListBSCPBizsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBSCPBizs: async <Request extends ListBSCPBizsRequest = ListBSCPBizsRequest, ResponseData = BSCPBizOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bscp/bizs')(params, config),
  /**
   * 获取 BSCP 业务下的服务列表
   *
   * @method GET
   * @path /bscp/bizs/{bizID}/services
   * @tag bkintegrations-bscp
   * @param bizID path string required BSCP 业务 ID
   * @response 200 ListBSCPServicesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBSCPServices: async <Request extends ListBSCPServicesRequest = ListBSCPServicesRequest, ResponseData = BSCPServiceOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bscp/bizs/{bizID}/services')(params, config),
  /**
   * 获取 BSCP 服务下的配置列表
   *
   * @method GET
   * @path /bscp/bizs/{bizID}/services/{serviceID}/configs
   * @tag bkintegrations-bscp
   * @param bizID path string required BSCP 业务 ID
   * @param serviceID path string required BSCP 服务 ID
   * @response 200 ListBSCPConfigsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listBSCPConfigs: async <Request extends ListBSCPConfigsRequest = ListBSCPConfigsRequest, ResponseData = BSCPConfigOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bscp/bizs/{bizID}/services/{serviceID}/configs')(params, config),
  /**
   * 获取 BSCP 配置项内容
   *
   * @method GET
   * @path /bscp/bizs/{bizID}/services/{serviceID}/configs/{configID}
   * @tag bkintegrations-bscp
   * @param bizID path string required BSCP 业务 ID
   * @param serviceID path string required BSCP 服务 ID
   * @param configID path string required BSCP 配置项 ID
   * @response 200 GetBSCPConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getBSCPConfig: async <Request extends GetBSCPConfigRequest = GetBSCPConfigRequest, ResponseData = BSCPConfigDetailOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bscp/bizs/{bizID}/services/{serviceID}/configs/{configID}')(params, config),
};
