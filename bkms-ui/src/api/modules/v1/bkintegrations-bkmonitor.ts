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
import type { GetApmServiceNameRequest, GetApmServiceNameOutput, GetInstanceTimeSeriesRequest, MetricTimeSeries, GetEnvApmRequest, GetEnvApmOutput, CreateEnvApmRequest, ApmOutput, BindApmToEnvRequest, EmptyOutput, ListApmsRequest, ListApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';

export const BkintegrationsBkmonitorService = {
  /**
   * 获取应用环境的 APM 服务名称
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/bkmonitor/apm-service-name
   * @tag bkintegrations-bkmonitor
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 GetApmServiceNameResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getApmServiceName: async <Request extends GetApmServiceNameRequest = GetApmServiceNameRequest, ResponseData = GetApmServiceNameOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/bkmonitor/apm-service-name')(params, config),
  /**
   * 查询实例监控指标时序数据
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/bkmonitor/instance-time-series
   * @tag bkintegrations-bkmonitor
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param instances query string[] required 实例名称列表
   * @param metricKey query string required 指标标识
   * @param startTime query number required 开始时间（Unix 时间戳）
   * @param endTime query number required 结束时间（Unix 时间戳）
   * @param interval query number 汇聚周期（秒），默认 60
   * @response 200 InstanceTimeSeriesResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getInstanceTimeSeries: async <Request extends GetInstanceTimeSeriesRequest = GetInstanceTimeSeriesRequest, ResponseData = Record<string, MetricTimeSeries>>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/bkmonitor/instance-time-series')(params, config),
  /**
   * 查询环境绑定的 APM
   *
   * @method GET
   * @path /envs/{envID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @response 200 GetEnvApmResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvApm: async <Request extends GetEnvApmRequest = GetEnvApmRequest, ResponseData = GetEnvApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/bkmonitor/apms')(params, config),
  /**
   * 为环境创建 APM 并绑定
   *
   * @method POST
   * @path /envs/{envID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @response 200 CreateEnvApmResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  createEnvApm: async <Request extends CreateEnvApmRequest = CreateEnvApmRequest, ResponseData = ApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/envs/{envID}/bkmonitor/apms')(params, config),
  /**
   * 将环境绑定到指定 APM
   *
   * @method PUT
   * @path /envs/{envID}/bkmonitor/apms/{apmID}
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @param apmID path string required APM ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  bindApmToEnv: async <Request extends BindApmToEnvRequest = BindApmToEnvRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/envs/{envID}/bkmonitor/apms/{apmID}')(params, config),
  /**
   * 获取工作空间下的 APM 列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListApmsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listApms: async <Request extends ListApmsRequest = ListApmsRequest, ResponseData = ListApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/apms')(params, config),
};
