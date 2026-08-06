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
import type { DownloadAppInstanceLogsRequest } from '~/@types/v1/instance-log';

export const InstanceLogService = {
  /**
   * 下载应用运行实例（Pod）日志
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances/{instanceID}/logs/download
   * @tag instance-log
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param instanceID path string required 实例 ID
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param previous query boolean 是否获取重启前日志
   * @response 200 string binary log stream
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  downloadAppInstanceLogs: async <Request extends DownloadAppInstanceLogsRequest = DownloadAppInstanceLogsRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/{instanceID}/logs/download')(params, config),
};
