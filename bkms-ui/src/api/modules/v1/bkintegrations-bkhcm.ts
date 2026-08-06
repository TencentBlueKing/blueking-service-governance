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
import type { ListBkHCMSubnetsRequest, SubnetOutput, ListBkHCMVPCsRequest, VPCOutput, CreateBkHCMLoadBalancerApplicationRequest, CreateBkHCMLoadBalancerData, ListBkHCMRegionsRequest, RegionOutput, ListBkHCMZonesRequest, ZoneOutput } from '~/@types/v1/bkintegrations-bkhcm';

export const BkintegrationsBkhcmService = {
  /**
   * 查询子网列表
   *
   * @method POST
   * @path /bkhcm/bizs/{bkBizID}/subnets
   * @tag bkintegrations-bkhcm
   * @param bkBizID path number required 业务 ID
   * @param body body BkHCMListInput required 查询参数
   * @response 200 ListBkHCMSubnetsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkHCMSubnets: async <Request extends ListBkHCMSubnetsRequest = ListBkHCMSubnetsRequest, ResponseData = SubnetOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/bkhcm/bizs/{bkBizID}/subnets')(params, config),
  /**
   * 查询 VPC 列表
   *
   * @method POST
   * @path /bkhcm/bizs/{bkBizID}/vpcs
   * @tag bkintegrations-bkhcm
   * @param bkBizID path number required 业务 ID
   * @param body body BkHCMListInput required 查询参数
   * @response 200 ListBkHCMVPCsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkHCMVPCs: async <Request extends ListBkHCMVPCsRequest = ListBkHCMVPCsRequest, ResponseData = VPCOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/bkhcm/bizs/{bkBizID}/vpcs')(params, config),
  /**
   * 创建负载均衡申请
   *
   * @method POST
   * @path /bkhcm/load-balancers
   * @tag bkintegrations-bkhcm
   * @param body body BkHCMCreateLoadBalancerInput required 创建负载均衡参数
   * @response 200 CreateBkHCMLoadBalancerOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createBkHCMLoadBalancerApplication: async <Request extends CreateBkHCMLoadBalancerApplicationRequest = CreateBkHCMLoadBalancerApplicationRequest, ResponseData = CreateBkHCMLoadBalancerData>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/bkhcm/load-balancers')(params, config),
  /**
   * 查询云地域列表
   *
   * @method POST
   * @path /bkhcm/regions
   * @tag bkintegrations-bkhcm
   * @param body body BkHCMListInput required 查询参数
   * @response 200 ListBkHCMRegionsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkHCMRegions: async <Request extends ListBkHCMRegionsRequest = ListBkHCMRegionsRequest, ResponseData = RegionOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/bkhcm/regions')(params, config),
  /**
   * 查询可用区列表
   *
   * @method POST
   * @path /bkhcm/regions/{region}/zones
   * @tag bkintegrations-bkhcm
   * @param region path string required 地域 ID
   * @param body body BkHCMListInput required 查询参数
   * @response 200 ListBkHCMZonesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkHCMZones: async <Request extends ListBkHCMZonesRequest = ListBkHCMZonesRequest, ResponseData = ZoneOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/bkhcm/regions/{region}/zones')(params, config),
};
